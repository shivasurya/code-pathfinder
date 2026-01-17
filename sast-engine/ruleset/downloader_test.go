package ruleset

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDownloader(t *testing.T) {
	tempDir := t.TempDir()
	config := &DownloadConfig{
		BaseURL:       "https://example.com",
		CacheDir:      tempDir,
		CacheTTL:      24 * time.Hour,
		HTTPTimeout:   30 * time.Second,
		RetryAttempts: 3,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	if downloader.config.BaseURL != config.BaseURL {
		t.Errorf("expected baseURL %s, got %s", config.BaseURL, downloader.config.BaseURL)
	}

	if downloader.cache == nil {
		t.Errorf("cache should not be nil")
	}

	if downloader.manifestLoader == nil {
		t.Errorf("manifestLoader should not be nil")
	}
}

func TestDownloadSuccess(t *testing.T) {
	tempDir := t.TempDir()

	// Create test zip file
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath, map[string]string{
		"rule1.py": "# rule 1",
		"rule2.py": "# rule 2",
	}); err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	// Calculate checksum
	checksum, err := calculateFileChecksum(zipPath)
	if err != nil {
		t.Fatalf("failed to calculate checksum: %v", err)
	}

	// Read zip file
	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	// Create test server
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/docker/manifest.json":
			manifest := &Manifest{
				Category: "docker",
				Bundles: map[string]*Bundle{
					"security": {
						Name:        "Security",
						FileCount:   2,
						ZipSize:     int64(len(zipData)),
						Checksum:    checksum,
						DownloadURL: serverURL + "/docker/security.zip",
					},
				},
			}
			json.NewEncoder(w).Encode(manifest)
		case "/docker/security.zip":
			w.Header().Set("Content-Type", "application/zip")
			w.Write(zipData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	config := &DownloadConfig{
		BaseURL:       server.URL,
		CacheDir:      filepath.Join(tempDir, "cache"),
		CacheTTL:      1 * time.Hour,
		HTTPTimeout:   10 * time.Second,
		RetryAttempts: 3,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	// Test download
	extractedPath, err := downloader.Download("docker/security")
	if err != nil {
		t.Fatalf("failed to download: %v", err)
	}

	// Verify extracted files
	rule1 := filepath.Join(extractedPath, "rule1.py")
	if _, err := os.Stat(rule1); os.IsNotExist(err) {
		t.Errorf("rule1.py should exist")
	}

	rule2 := filepath.Join(extractedPath, "rule2.py")
	if _, err := os.Stat(rule2); os.IsNotExist(err) {
		t.Errorf("rule2.py should exist")
	}

	// Test cache hit
	cachedPath, err := downloader.Download("docker/security")
	if err != nil {
		t.Fatalf("failed to download from cache: %v", err)
	}

	if cachedPath != extractedPath {
		t.Errorf("expected cached path %s, got %s", extractedPath, cachedPath)
	}
}

func TestDownloadInvalidSpec(t *testing.T) {
	tempDir := t.TempDir()
	config := &DownloadConfig{
		BaseURL:       "https://example.com",
		CacheDir:      tempDir,
		CacheTTL:      1 * time.Hour,
		HTTPTimeout:   10 * time.Second,
		RetryAttempts: 3,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	_, err = downloader.Download("invalid-spec")
	if err == nil {
		t.Errorf("expected error for invalid spec, got nil")
	}
}

func TestDownloadChecksumMismatch(t *testing.T) {
	tempDir := t.TempDir()

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath, map[string]string{
		"rule1.py": "# rule 1",
	}); err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	zipData, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}

	// Create server with wrong checksum
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/docker/manifest.json":
			manifest := &Manifest{
				Category: "docker",
				Bundles: map[string]*Bundle{
					"security": {
						Name:        "Security",
						FileCount:   1,
						ZipSize:     int64(len(zipData)),
						Checksum:    "wrongchecksum123", // Wrong checksum
						DownloadURL: serverURL + "/docker/security.zip",
					},
				},
			}
			json.NewEncoder(w).Encode(manifest)
		case "/docker/security.zip":
			w.Write(zipData)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	config := &DownloadConfig{
		BaseURL:       server.URL,
		CacheDir:      filepath.Join(tempDir, "cache"),
		CacheTTL:      1 * time.Hour,
		HTTPTimeout:   10 * time.Second,
		RetryAttempts: 1,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	_, err = downloader.Download("docker/security")
	if err == nil {
		t.Errorf("expected error for checksum mismatch, got nil")
	}
}

func TestRefreshCache(t *testing.T) {
	tempDir := t.TempDir()

	config := &DownloadConfig{
		BaseURL:       "https://example.com",
		CacheDir:      tempDir,
		CacheTTL:      1 * time.Hour,
		HTTPTimeout:   10 * time.Second,
		RetryAttempts: 3,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	// Create cache entry
	spec := &RulesetSpec{Category: "docker", Bundle: "security"}
	extractedPath := filepath.Join(tempDir, "docker", "security")
	os.MkdirAll(extractedPath, 0755)
	downloader.cache.Set(spec, extractedPath, "abc123", 1*time.Hour)

	// Refresh cache
	err = downloader.RefreshCache("docker/security")
	if err != nil {
		t.Errorf("failed to refresh cache: %v", err)
	}

	// Verify cache is invalidated
	_, err = downloader.cache.Get(spec, "abc123")
	if err == nil {
		t.Errorf("cache should be invalidated")
	}
}

func TestExtractFile(t *testing.T) {
	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "dest")

	// Create test zip
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipPath, map[string]string{
		"test.txt": "test content",
		"subdir/nested.txt": "nested content",
	}); err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	// Open zip
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	// Extract files
	for _, f := range r.File {
		if err := extractFile(f, destDir); err != nil {
			t.Fatalf("failed to extract file: %v", err)
		}
	}

	// Verify extracted files
	if _, err := os.Stat(filepath.Join(destDir, "test.txt")); os.IsNotExist(err) {
		t.Errorf("test.txt should exist")
	}

	if _, err := os.Stat(filepath.Join(destDir, "subdir", "nested.txt")); os.IsNotExist(err) {
		t.Errorf("subdir/nested.txt should exist")
	}
}

func TestExtractFileZipSlip(t *testing.T) {
	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "dest")

	// Create malicious zip with path traversal
	zipPath := filepath.Join(tempDir, "malicious.zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}

	zw := zip.NewWriter(zf)

	// Try to create file outside dest dir
	fw, err := zw.Create("../../../etc/passwd")
	if err != nil {
		t.Fatalf("failed to add malicious file: %v", err)
	}
	fw.Write([]byte("malicious content"))

	zw.Close()
	zf.Close()

	// Try to extract
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	// Should fail with zip slip prevention
	for _, f := range r.File {
		err := extractFile(f, destDir)
		if err == nil {
			t.Errorf("expected error for zip slip attempt, got nil")
		}
	}
}

// Helper functions

func createTestZip(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func calculateFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
