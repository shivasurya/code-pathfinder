package ruleset

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Downloader handles remote ruleset downloads.
type Downloader struct {
	config         *DownloadConfig
	cache          *Cache
	manifestLoader *ManifestLoader
	httpClient     *http.Client
}

// NewDownloader creates a new downloader.
func NewDownloader(config *DownloadConfig) (*Downloader, error) {
	cache, err := NewCache(config.CacheDir)
	if err != nil {
		return nil, err
	}

	return &Downloader{
		config:         config,
		cache:          cache,
		manifestLoader: NewManifestLoader(config.BaseURL, config.CacheDir),
		httpClient:     &http.Client{Timeout: config.HTTPTimeout},
	}, nil
}

// Download downloads and caches a ruleset, returns path to extracted rules.
func (d *Downloader) Download(spec string) (string, error) {
	// Parse spec
	rulesetSpec, err := ParseSpec(spec)
	if err != nil {
		return "", err
	}

	if err := rulesetSpec.Validate(); err != nil {
		return "", err
	}

	// Load manifest to get bundle metadata
	manifest, err := d.manifestLoader.LoadCategoryManifest(rulesetSpec.Category)
	if err != nil {
		return "", fmt.Errorf("failed to load manifest: %w", err)
	}

	bundle, err := manifest.GetBundle(rulesetSpec.Bundle)
	if err != nil {
		return "", err
	}

	// Check cache
	cachedPath, err := d.cache.Get(rulesetSpec, bundle.Checksum)
	if err == nil {
		fmt.Printf("✓ Using cached ruleset (checksum: %s...)\n", bundle.Checksum[:8])
		return cachedPath, nil
	}

	// Cache miss or expired - download
	fmt.Printf("📦 Downloading ruleset: %s\n", spec)
	return d.downloadAndCache(rulesetSpec, bundle)
}

// downloadAndCache downloads zip, verifies checksum, extracts, and caches.
func (d *Downloader) downloadAndCache(spec *RulesetSpec, bundle *Bundle) (string, error) {
	// Download zip file
	zipPath, err := d.downloadZip(bundle.DownloadURL, bundle.ZipSize)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(zipPath) // Clean up zip after extraction

	// Verify checksum
	fmt.Printf("🔒 Verifying checksum...\n")
	if err := VerifyChecksum(zipPath, bundle.Checksum); err != nil {
		return "", fmt.Errorf("checksum verification failed: %w", err)
	}
	fmt.Printf("✓ Checksum verified\n")

	// Extract to cache directory
	extractPath := filepath.Join(d.config.CacheDir, spec.Category, spec.Bundle)
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return "", err
	}

	fmt.Printf("📂 Extracting rules...\n")
	fileCount, err := d.extractZip(zipPath, extractPath)
	if err != nil {
		return "", fmt.Errorf("extraction failed: %w", err)
	}
	fmt.Printf("✓ Extracted %d rules\n", fileCount)

	// Store in cache
	if err := d.cache.Set(spec, extractPath, bundle.Checksum, d.config.CacheTTL); err != nil {
		return "", fmt.Errorf("cache save failed: %w", err)
	}

	return extractPath, nil
}

// downloadZip downloads a file with retry logic.
func (d *Downloader) downloadZip(url string, expectedSize int64) (string, error) {
	tempFile, err := os.CreateTemp("", "ruleset-*.zip")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	var lastErr error
	for attempt := 0; attempt < d.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			fmt.Printf("⚠️  Retry %d/%d...\n", attempt, d.config.RetryAttempts)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := d.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		// Download with progress (simplified)
		written, err := io.Copy(tempFile, resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if expectedSize > 0 && written != expectedSize {
			lastErr = fmt.Errorf("size mismatch: expected %d, got %d", expectedSize, written)
			continue
		}

		// Success
		fmt.Printf("✓ Downloaded %s (%.1f KB)\n", filepath.Base(url), float64(written)/1024)
		return tempFile.Name(), nil
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", d.config.RetryAttempts, lastErr)
}

// extractZip extracts a zip file to destination.
func (d *Downloader) extractZip(zipPath, destDir string) (int, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	count := 0
	for _, f := range r.File {
		if err := extractFile(f, destDir); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// extractFile extracts a single file from zip.
func extractFile(f *zip.File, destDir string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(destDir, f.Name)

	// Security: prevent zip slip by checking if path is within destDir
	cleanDest := filepath.Clean(destDir)
	cleanPath := filepath.Clean(path)
	relPath, err := filepath.Rel(cleanDest, cleanPath)
	if err != nil || len(relPath) > 0 && (relPath[0:1] == "." || filepath.IsAbs(relPath)) {
		return fmt.Errorf("illegal file path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// RefreshCache invalidates and re-downloads a ruleset.
func (d *Downloader) RefreshCache(spec string) error {
	rulesetSpec, err := ParseSpec(spec)
	if err != nil {
		return err
	}

	return d.cache.Invalidate(rulesetSpec)
}
