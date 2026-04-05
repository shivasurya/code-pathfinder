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

// GoThirdPartyRegistryRemote loads Go third-party library packages from a remote
// CDN with lazy loading and in-memory caching. It implements the core.GoThirdPartyLoader
// interface.
//
// Architecture mirrors GoStdlibRegistryRemote: manifest-first loading with lazy
// per-package downloads, checksum verification (using "sha256:" prefix format),
// and RWMutex-protected cache with double-check locking for safe concurrent use.
//
// CDN URL structure:
//
//	{baseURL}/go-thirdparty/v1/manifest.json
//	{baseURL}/go-thirdparty/v1/{encoded}.json
//
// Module path encoding: slashes replaced with underscores
//
//	"gorm.io/gorm"           → "gorm.io_gorm.json"
//	"github.com/gin-gonic/gin" → "github.com_gin-gonic_gin.json"
type GoThirdPartyRegistryRemote struct {
	baseURL      string                           // CDN base URL (trailing slash stripped on construction)
	manifest     *core.GoManifest                 // Loaded manifest (nil until LoadManifest is called)
	packageCache map[string]*core.GoStdlibPackage // Import path → downloaded package data
	cacheMutex   sync.RWMutex                     // Guards packageCache and manifest
	httpClient   *http.Client                     // Reusable HTTP client with timeout
	logger       *output.Logger                   // Structured logger for progress and diagnostics
}

// NewGoThirdPartyRegistryRemote creates an initialized GoThirdPartyRegistryRemote.
//
// Parameters:
//   - baseURL: CDN base URL (trailing slash is stripped automatically)
//   - logger: structured logger for progress and diagnostic messages
func NewGoThirdPartyRegistryRemote(baseURL string, logger *output.Logger) *GoThirdPartyRegistryRemote {
	return &GoThirdPartyRegistryRemote{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		packageCache: make(map[string]*core.GoStdlibPackage),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// LoadManifest downloads and parses manifest.json from the CDN.
// It must be called before ValidateImport, PackageCount, or GetPackage.
func (r *GoThirdPartyRegistryRemote) LoadManifest() error {
	manifestURL := fmt.Sprintf("%s/go-thirdparty/v1/manifest.json", r.baseURL)
	r.logger.Debug("Downloading Go third-party manifest: %s", manifestURL)

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

	r.logger.Statistic("Loaded Go third-party manifest: %d packages", len(manifest.Packages))
	return nil
}

// GetPackage retrieves a third-party package by import path, downloading it from
// the CDN if not already cached. The manifest must be loaded before calling this
// method. Checksum verification is performed when the manifest entry includes a
// non-empty checksum.
func (r *GoThirdPartyRegistryRemote) GetPackage(importPath string) (*core.GoStdlibPackage, error) {
	// Fast path: check cache with a read lock.
	r.cacheMutex.RLock()
	if pkg, ok := r.packageCache[importPath]; ok {
		r.cacheMutex.RUnlock()
		return pkg, nil
	}
	r.cacheMutex.RUnlock()

	// Slow path: acquire write lock to serialize downloads and prevent duplicate
	// HTTP requests from concurrent callers.
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	return r.fetchPackageLocked(importPath)
}

// fetchPackageLocked downloads and caches a package.
// It must be called with cacheMutex write-locked.
func (r *GoThirdPartyRegistryRemote) fetchPackageLocked(importPath string) (*core.GoStdlibPackage, error) {
	// Double-check: another goroutine may have populated the cache between the
	// read-lock miss and this write-lock acquisition.
	if pkg, ok := r.packageCache[importPath]; ok {
		return pkg, nil
	}

	// Verify manifest is loaded and package is declared in it.
	if r.manifest == nil {
		return nil, fmt.Errorf("manifest not loaded")
	}
	entry := r.manifest.GetPackageEntry(importPath)
	if entry == nil {
		return nil, fmt.Errorf("package %q not in manifest", importPath)
	}

	// Download the package JSON.
	encoded := encodeModulePath(importPath)
	pkgURL := fmt.Sprintf("%s/go-thirdparty/v1/%s.json", r.baseURL, encoded)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pkgURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating package request for %s: %w", importPath, err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading package %s: %w", importPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package download returned HTTP %d for %s", resp.StatusCode, importPath)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading package body for %s: %w", importPath, err)
	}

	// Checksum verification: uses "sha256:" + hex prefix format (same as stdlib).
	// An empty Checksum in the manifest entry skips verification (development convenience).
	if entry.Checksum != "" {
		hash := sha256.Sum256(data)
		actual := "sha256:" + hex.EncodeToString(hash[:])
		if actual != entry.Checksum {
			return nil, fmt.Errorf("checksum mismatch for %s: expected %s, got %s",
				importPath, entry.Checksum, actual)
		}
	}

	var pkg core.GoStdlibPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package JSON for %s: %w", importPath, err)
	}

	r.packageCache[importPath] = &pkg
	return &pkg, nil
}

// ValidateImport implements core.GoThirdPartyLoader.
// It reports whether the given import path is declared in the loaded manifest.
// Returns false if the manifest has not been loaded yet.
func (r *GoThirdPartyRegistryRemote) ValidateImport(importPath string) bool {
	r.cacheMutex.RLock()
	manifest := r.manifest
	r.cacheMutex.RUnlock()

	if manifest == nil {
		return false
	}
	return manifest.HasPackage(importPath)
}

// GetFunction implements core.GoThirdPartyLoader.
// It returns the metadata for a named package-level function in the given
// third-party package, downloading the package from the CDN if needed.
func (r *GoThirdPartyRegistryRemote) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	pkg, err := r.GetPackage(importPath)
	if err != nil {
		return nil, err
	}
	if fn, ok := pkg.Functions[funcName]; ok {
		return fn, nil
	}
	return nil, fmt.Errorf("function %s not found in %s", funcName, importPath)
}

// GetType implements core.GoThirdPartyLoader.
// It returns the metadata for a named type in the given third-party package,
// downloading the package from the CDN if needed.
func (r *GoThirdPartyRegistryRemote) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	pkg, err := r.GetPackage(importPath)
	if err != nil {
		return nil, err
	}
	if t, ok := pkg.Types[typeName]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("type %s not found in %s", typeName, importPath)
}

// PackageCount implements core.GoThirdPartyLoader.
// It returns the total number of third-party packages declared in the manifest.
// Returns 0 if the manifest has not been loaded.
func (r *GoThirdPartyRegistryRemote) PackageCount() int {
	r.cacheMutex.RLock()
	manifest := r.manifest
	r.cacheMutex.RUnlock()

	if manifest == nil {
		return 0
	}
	return len(manifest.Packages)
}

// IsManifestLoaded reports whether the manifest has been successfully loaded.
func (r *GoThirdPartyRegistryRemote) IsManifestLoaded() bool {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	return r.manifest != nil
}

// CacheSize returns the number of packages currently held in the in-memory cache.
func (r *GoThirdPartyRegistryRemote) CacheSize() int {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	return len(r.packageCache)
}

// ClearCache evicts all packages from the in-memory cache.
// The manifest is retained; only downloaded package data is released.
func (r *GoThirdPartyRegistryRemote) ClearCache() {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	r.packageCache = make(map[string]*core.GoStdlibPackage)
}

// encodeModulePath encodes a Go module import path for use as a CDN filename.
// Slashes are replaced with underscores.
//
// Examples:
//
//	"gorm.io/gorm"               → "gorm.io_gorm"
//	"github.com/gin-gonic/gin"   → "github.com_gin-gonic_gin"
//	"github.com/jackc/pgx/v5"    → "github.com_jackc_pgx_v5"
func encodeModulePath(modulePath string) string {
	return strings.ReplaceAll(modulePath, "/", "_")
}
