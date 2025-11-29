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

// StdlibRegistryRemote loads Python stdlib registries from a remote CDN.
// It uses lazy loading (downloads modules on-demand) and in-memory caching.
type StdlibRegistryRemote struct {
	BaseURL       string                        // CDN base URL (e.g., "https://assets.codepathfinder.dev/registries")
	PythonVersion string                        // Python version (e.g., "3.14")
	Manifest      *core.Manifest                // Loaded manifest
	ModuleCache   map[string]*core.StdlibModule // In-memory cache of loaded modules
	CacheMutex    sync.RWMutex                  // Mutex for thread-safe cache access
	HTTPClient    *http.Client                  // HTTP client for downloads
}

// NewStdlibRegistryRemote creates a new remote registry loader.
//
// Parameters:
//   - baseURL: CDN base URL (e.g., "https://assets.codepathfinder.dev/registries")
//   - pythonVersion: Python version (e.g., "3.14")
//
// Returns:
//   - Initialized StdlibRegistryRemote
func NewStdlibRegistryRemote(baseURL, pythonVersion string) *StdlibRegistryRemote {
	return &StdlibRegistryRemote{
		BaseURL:       strings.TrimSuffix(baseURL, "/"),
		PythonVersion: pythonVersion,
		ModuleCache:   make(map[string]*core.StdlibModule),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoadManifest downloads and parses the manifest.json file from the CDN.
//
// Parameters:
//   - logger: structured logger for diagnostics
//
// Returns:
//   - error if download or parsing fails
func (r *StdlibRegistryRemote) LoadManifest(logger *output.Logger) error {
	manifestURL := fmt.Sprintf("%s/python%s/stdlib/v1/manifest.json",
		r.BaseURL, r.PythonVersion)

	logger.Debug("Downloading manifest from: %s", manifestURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, manifestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create manifest request: %w", err)
	}

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("manifest download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest body: %w", err)
	}

	var manifest core.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	r.Manifest = &manifest
	logger.Statistic("Loaded manifest: %d modules", len(manifest.Modules))

	return nil
}

// GetModule retrieves a module by name, downloading it if not cached.
// This implements lazy loading - modules are only downloaded when needed.
//
// Parameters:
//   - moduleName: name of the module (e.g., "os", "sys")
//   - logger: structured logger for diagnostics
//
// Returns:
//   - StdlibModule if found, nil otherwise
//   - error if download or parsing fails
func (r *StdlibRegistryRemote) GetModule(moduleName string, logger *output.Logger) (*core.StdlibModule, error) {
	// Check cache first (read lock)
	r.CacheMutex.RLock()
	if module, ok := r.ModuleCache[moduleName]; ok {
		r.CacheMutex.RUnlock()
		return module, nil
	}
	r.CacheMutex.RUnlock()

	// Find module entry in manifest
	if r.Manifest == nil {
		return nil, fmt.Errorf("manifest not loaded")
	}

	var moduleEntry *core.ModuleEntry
	for _, entry := range r.Manifest.Modules {
		if entry.Name == moduleName {
			moduleEntry = entry
			break
		}
	}

	if moduleEntry == nil {
		return nil, nil //nolint:nilnil // nil module is valid when not found
	}

	// Download module (write lock)
	r.CacheMutex.Lock()
	defer r.CacheMutex.Unlock()

	// Double-check cache (another goroutine might have loaded it)
	if module, ok := r.ModuleCache[moduleName]; ok {
		return module, nil
	}

	// Download module file
	moduleURL := fmt.Sprintf("%s/python%s/stdlib/v1/%s",
		r.BaseURL, r.PythonVersion, moduleEntry.File)

	logger.Debug("Downloading module: %s from %s", moduleName, moduleURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, moduleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create module request: %w", err)
	}

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download module %s: %w", moduleName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("module download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read module body: %w", err)
	}

	// Verify checksum
	if !r.verifyChecksum(data, moduleEntry.Checksum) {
		return nil, fmt.Errorf("checksum mismatch for module %s", moduleName)
	}

	// Parse module JSON
	var module core.StdlibModule
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, fmt.Errorf("failed to parse module JSON: %w", err)
	}

	// Cache the module
	r.ModuleCache[moduleName] = &module

	return &module, nil
}

// verifyChecksum validates the SHA256 checksum of downloaded data.
//
// Parameters:
//   - data: downloaded file content
//   - expectedChecksum: expected checksum in format "sha256:hex"
//
// Returns:
//   - true if checksum matches, false otherwise
func (r *StdlibRegistryRemote) verifyChecksum(data []byte, expectedChecksum string) bool {
	hash := sha256.Sum256(data)
	actualChecksum := "sha256:" + hex.EncodeToString(hash[:])
	return actualChecksum == expectedChecksum
}

// HasModule checks if a module exists in the manifest (without downloading it).
//
// Parameters:
//   - moduleName: name of the module
//
// Returns:
//   - true if module exists in manifest, false otherwise
func (r *StdlibRegistryRemote) HasModule(moduleName string) bool {
	if r.Manifest == nil {
		return false
	}

	for _, entry := range r.Manifest.Modules {
		if entry.Name == moduleName {
			return true
		}
	}

	return false
}

// GetFunction retrieves a function from a module, downloading the module if needed.
//
// Parameters:
//   - moduleName: name of the module (e.g., "os")
//   - functionName: name of the function (e.g., "getcwd")
//   - logger: structured logger for diagnostics
//
// Returns:
//   - StdlibFunction if found, nil otherwise
func (r *StdlibRegistryRemote) GetFunction(moduleName, functionName string, logger *output.Logger) *core.StdlibFunction {
	module, err := r.GetModule(moduleName, logger)
	if err != nil {
		logger.Warning("Failed to get module %s: %v", moduleName, err)
		return nil
	}
	if module == nil {
		return nil
	}

	return module.Functions[functionName]
}

// GetClass retrieves a class from a module, downloading the module if needed.
//
// Parameters:
//   - moduleName: name of the module
//   - className: name of the class
//   - logger: structured logger for diagnostics
//
// Returns:
//   - StdlibClass if found, nil otherwise
func (r *StdlibRegistryRemote) GetClass(moduleName, className string, logger *output.Logger) *core.StdlibClass {
	module, err := r.GetModule(moduleName, logger)
	if err != nil {
		logger.Warning("Failed to get module %s: %v", moduleName, err)
		return nil
	}
	if module == nil {
		return nil
	}

	return module.Classes[className]
}

// ModuleCount returns the number of modules in the manifest.
//
// Returns:
//   - Number of modules, or 0 if manifest not loaded
func (r *StdlibRegistryRemote) ModuleCount() int {
	if r.Manifest == nil {
		return 0
	}
	return len(r.Manifest.Modules)
}

// CacheSize returns the number of modules currently cached in memory.
//
// Returns:
//   - Number of cached modules
func (r *StdlibRegistryRemote) CacheSize() int {
	r.CacheMutex.RLock()
	defer r.CacheMutex.RUnlock()
	return len(r.ModuleCache)
}

// ClearCache clears the in-memory module cache.
// Useful for testing or memory management.
func (r *StdlibRegistryRemote) ClearCache() {
	r.CacheMutex.Lock()
	defer r.CacheMutex.Unlock()
	r.ModuleCache = make(map[string]*core.StdlibModule)
}
