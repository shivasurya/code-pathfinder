package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// GoStdlibRegistryRemote loads Go stdlib packages from a remote CDN with lazy loading
// and in-memory caching. It implements the core.GoStdlibLoader interface.
//
// Packages are downloaded on-demand when first requested (lazy loading) and cached
// in memory to avoid redundant HTTP requests. All cache access is protected by an
// RWMutex using the double-check locking pattern for safe concurrent use.
type GoStdlibRegistryRemote struct {
	baseURL      string                             // CDN base URL (e.g., "https://assets.codepathfinder.dev/registries")
	goVersion    string                             // Go version tag (e.g., "1.21", "1.26")
	manifest     *core.GoManifest                   // Loaded manifest (nil until LoadManifest is called)
	packageCache map[string]*core.GoStdlibPackage   // Import path → downloaded package data
	cacheMutex   sync.RWMutex                       // Guards packageCache and manifest
	httpClient   *http.Client                       // Reusable HTTP client with timeout
}

// NewGoStdlibRegistryRemote creates an initialized GoStdlibRegistryRemote.
//
// Parameters:
//   - baseURL: CDN base URL (trailing slash is stripped automatically)
//   - goVersion: Go version tag (e.g., "1.21", "1.26.0")
func NewGoStdlibRegistryRemote(baseURL, goVersion string) *GoStdlibRegistryRemote {
	return &GoStdlibRegistryRemote{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		goVersion:    goVersion,
		packageCache: make(map[string]*core.GoStdlibPackage),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoadManifest downloads and parses manifest.json from the CDN.
// It must be called before ValidateStdlibImport, PackageCount, or GetPackage.
//
// Parameters:
//   - logger: structured logger for progress and diagnostic messages
func (r *GoStdlibRegistryRemote) LoadManifest(logger *output.Logger) error {
	manifestURL := fmt.Sprintf("%s/go%s/stdlib/v1/manifest.json", r.baseURL, r.goVersion)
	logger.Debug("Downloading Go stdlib manifest: %s", manifestURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, manifestURL, nil)
	if err != nil {
		return fmt.Errorf("creating manifest request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("manifest download returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading manifest body: %w", err)
	}

	var manifest core.GoManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing manifest JSON: %w", err)
	}

	r.cacheMutex.Lock()
	r.manifest = &manifest
	r.cacheMutex.Unlock()

	logger.Statistic("Loaded Go stdlib manifest: %d packages for Go %s",
		len(manifest.Packages), r.goVersion)
	return nil
}

// GetPackage retrieves a stdlib package by import path, downloading it from the CDN
// if it is not already cached. Returns an error if the package cannot be fetched
// or if checksum verification fails.
func (r *GoStdlibRegistryRemote) GetPackage(importPath string) (*core.GoStdlibPackage, error) {
	// Fast path: check cache with a read lock.
	r.cacheMutex.RLock()
	if pkg, ok := r.packageCache[importPath]; ok {
		r.cacheMutex.RUnlock()
		return pkg, nil
	}
	r.cacheMutex.RUnlock()

	// Slow path: acquire write lock, then delegate to the locked helper.
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	return r.fetchPackageLocked(importPath)
}

// fetchPackageLocked downloads and caches a package.
// It must be called with cacheMutex write-locked.
// It first re-checks the cache in case another goroutine already downloaded it.
func (r *GoStdlibRegistryRemote) fetchPackageLocked(importPath string) (*core.GoStdlibPackage, error) {
	// Double-check: another goroutine may have populated the cache between the
	// read-lock miss and the write-lock acquisition.
	if pkg, ok := r.packageCache[importPath]; ok {
		return pkg, nil
	}

	filename := goPackageToFilename(importPath)
	pkgURL := fmt.Sprintf("%s/go%s/stdlib/v1/%s", r.baseURL, r.goVersion, filename)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pkgURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", importPath, err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading package %s: %w", importPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("package %s not found in Go stdlib registry", importPath)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package %s download returned HTTP %d", importPath, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading package %s body: %w", importPath, err)
	}

	if err := r.verifyChecksum(importPath, data); err != nil {
		return nil, err
	}

	var pkg core.GoStdlibPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package %s JSON: %w", importPath, err)
	}

	r.packageCache[importPath] = &pkg
	return &pkg, nil
}

// ValidateStdlibImport implements core.GoStdlibLoader.
// It reports whether the given import path appears in the loaded manifest.
// Returns false if the manifest has not been loaded yet.
func (r *GoStdlibRegistryRemote) ValidateStdlibImport(importPath string) bool {
	r.cacheMutex.RLock()
	manifest := r.manifest
	r.cacheMutex.RUnlock()

	if manifest == nil {
		return false
	}
	return manifest.HasPackage(importPath)
}

// GetFunction implements core.GoStdlibLoader.
// It returns the metadata for a named function in the given stdlib package,
// downloading the package from the CDN if needed.
func (r *GoStdlibRegistryRemote) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	pkg, err := r.GetPackage(importPath)
	if err != nil {
		return nil, err
	}

	fn, ok := pkg.Functions[funcName]
	if !ok {
		return nil, fmt.Errorf("function %s not found in stdlib package %s", funcName, importPath)
	}
	return fn, nil
}

// GetType implements core.GoStdlibLoader.
// It returns the metadata for a named type in the given stdlib package,
// downloading the package from the CDN if needed.
func (r *GoStdlibRegistryRemote) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	pkg, err := r.GetPackage(importPath)
	if err != nil {
		return nil, err
	}

	typ, ok := pkg.Types[typeName]
	if !ok {
		return nil, fmt.Errorf("type %s not found in stdlib package %s", typeName, importPath)
	}
	return typ, nil
}

// PackageCount implements core.GoStdlibLoader.
// It returns the total number of stdlib packages declared in the manifest.
// Returns 0 if the manifest has not been loaded.
func (r *GoStdlibRegistryRemote) PackageCount() int {
	r.cacheMutex.RLock()
	manifest := r.manifest
	r.cacheMutex.RUnlock()

	if manifest == nil {
		return 0
	}
	if manifest.Statistics == nil {
		return len(manifest.Packages)
	}
	return manifest.Statistics.TotalPackages
}

// CacheSize returns the number of packages currently held in the in-memory cache.
func (r *GoStdlibRegistryRemote) CacheSize() int {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	return len(r.packageCache)
}

// ClearCache evicts all packages from the in-memory cache.
// The manifest is retained; only downloaded package data is released.
func (r *GoStdlibRegistryRemote) ClearCache() {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	r.packageCache = make(map[string]*core.GoStdlibPackage)
}

// IsManifestLoaded reports whether the manifest has been successfully loaded.
func (r *GoStdlibRegistryRemote) IsManifestLoaded() bool {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	return r.manifest != nil
}

// GoVersion returns the Go version tag this registry was initialized for.
func (r *GoStdlibRegistryRemote) GoVersion() string {
	return r.goVersion
}

// verifyChecksum validates the SHA256 checksum of downloaded package data against
// the expected checksum recorded in the manifest.
func (r *GoStdlibRegistryRemote) verifyChecksum(importPath string, data []byte) error {
	if r.manifest == nil {
		// When manifest is not loaded, skip verification rather than blocking downloads.
		return nil
	}

	entry := r.manifest.GetPackageEntry(importPath)
	if entry == nil {
		// Package not in manifest — verification not possible; allow the download.
		return nil
	}

	if entry.Checksum == "" {
		return nil
	}

	sum := sha256.Sum256(data)
	actual := "sha256:" + hex.EncodeToString(sum[:])
	if actual != entry.Checksum {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
			importPath, entry.Checksum, actual)
	}
	return nil
}

// goPackageToFilename converts a stdlib import path to a registry filename.
// e.g., "net/http" → "net_http_stdlib.json".
func goPackageToFilename(importPath string) string {
	return strings.ReplaceAll(importPath, "/", "_") + "_stdlib.json"
}
