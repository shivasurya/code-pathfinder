package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// ThirdPartyRegistryRemote loads third-party type registries from a remote CDN.
// It mirrors StdlibRegistryRemote but targets the thirdparty/v1/ path and adds
// inheritance-aware lookups (GetClassAttribute, GetClassMethod, IsSubclass).
type ThirdPartyRegistryRemote struct {
	BaseURL     string                        // CDN base URL
	Manifest    *core.Manifest                // Loaded manifest
	ModuleCache map[string]*core.StdlibModule // In-memory cache
	CacheMutex  sync.RWMutex                  // Thread-safe access
	HTTPClient  *http.Client                  // HTTP client
}

// NewThirdPartyRegistryRemote creates a new third-party registry loader.
func NewThirdPartyRegistryRemote(baseURL string) *ThirdPartyRegistryRemote {
	return &ThirdPartyRegistryRemote{
		BaseURL:     strings.TrimSuffix(baseURL, "/"),
		ModuleCache: make(map[string]*core.StdlibModule),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoadManifest downloads and parses the manifest.json from the CDN.
func (r *ThirdPartyRegistryRemote) LoadManifest(logger *output.Logger) error {
	manifestURL := fmt.Sprintf("%s/thirdparty/v1/manifest.json", r.BaseURL)

	logger.Debug("Downloading third-party manifest from: %s", manifestURL)

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
	logger.Statistic("Loaded third-party manifest: %d modules", len(manifest.Modules))

	return nil
}

// GetModule retrieves a module by name, downloading it lazily if not cached.
func (r *ThirdPartyRegistryRemote) GetModule(moduleName string, logger *output.Logger) (*core.StdlibModule, error) {
	// Check cache first (read lock)
	r.CacheMutex.RLock()
	if module, ok := r.ModuleCache[moduleName]; ok {
		r.CacheMutex.RUnlock()
		return module, nil
	}
	r.CacheMutex.RUnlock()

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

	// Write lock for download
	r.CacheMutex.Lock()
	defer r.CacheMutex.Unlock()

	// Double-check cache (another goroutine may have loaded it)
	if module, ok := r.ModuleCache[moduleName]; ok {
		return module, nil
	}

	moduleURL := fmt.Sprintf("%s/thirdparty/v1/%s", r.BaseURL, moduleEntry.File)

	logger.Debug("Downloading third-party module: %s from %s", moduleName, moduleURL)

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

	if !verifyThirdPartyChecksum(data, moduleEntry.Checksum) {
		return nil, fmt.Errorf("checksum mismatch for module %s", moduleName)
	}

	var module core.StdlibModule
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, fmt.Errorf("failed to parse module JSON: %w", err)
	}

	r.ModuleCache[moduleName] = &module

	return &module, nil
}

// verifyThirdPartyChecksum validates the SHA256 checksum of downloaded data.
func verifyThirdPartyChecksum(data []byte, expectedChecksum string) bool {
	hash := sha256.Sum256(data)
	actualChecksum := "sha256:" + hex.EncodeToString(hash[:])
	return actualChecksum == expectedChecksum
}

// HasModule checks if a module exists in the manifest without downloading it.
func (r *ThirdPartyRegistryRemote) HasModule(moduleName string) bool {
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
func (r *ThirdPartyRegistryRemote) GetFunction(moduleName, functionName string, logger *output.Logger) *core.StdlibFunction {
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
// Supports dotted class names (e.g., "http.HttpRequest" in the django module).
func (r *ThirdPartyRegistryRemote) GetClass(moduleName, className string, logger *output.Logger) *core.StdlibClass {
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

// GetClassAttribute retrieves a class attribute, checking own attributes first,
// then inherited attributes.
func (r *ThirdPartyRegistryRemote) GetClassAttribute(
	moduleName, className, attrName string, logger *output.Logger,
) *core.StdlibAttribute {
	cls := r.GetClass(moduleName, className, logger)
	if cls == nil {
		return nil
	}

	// Check own attributes first
	if attr, ok := cls.Attributes[attrName]; ok {
		return attr
	}

	// Check inherited attributes
	if cls.InheritedAttributes != nil {
		if inherited, ok := cls.InheritedAttributes[attrName]; ok {
			return &core.StdlibAttribute{
				Type:       inherited.Type,
				Confidence: inherited.Confidence,
				Source:     inherited.Source,
				Kind:       inherited.Kind,
			}
		}
	}

	return nil
}

// GetClassMethod retrieves a class method, checking own methods first,
// then inherited methods.
func (r *ThirdPartyRegistryRemote) GetClassMethod(
	moduleName, className, methodName string, logger *output.Logger,
) *core.StdlibFunction {
	cls := r.GetClass(moduleName, className, logger)
	if cls == nil {
		return nil
	}

	// Check own methods first
	if method, ok := cls.Methods[methodName]; ok {
		return method
	}

	// Check inherited methods
	if cls.InheritedMethods != nil {
		if inherited, ok := cls.InheritedMethods[methodName]; ok {
			return &core.StdlibFunction{
				ReturnType: inherited.ReturnType,
				Confidence: inherited.Confidence,
				Params:     inherited.Params,
				Source:     inherited.Source,
			}
		}
	}

	return nil
}

// IsSubclass checks if a class is a subclass of parentFQN using pre-computed MRO.
// No runtime MRO computation — just walks the stored MRO list.
func (r *ThirdPartyRegistryRemote) IsSubclass(
	moduleName, className, parentFQN string, logger *output.Logger,
) bool {
	cls := r.GetClass(moduleName, className, logger)
	if cls == nil {
		return false
	}

	return slices.Contains(cls.MRO, parentFQN)
}

// GetClassMRO returns the MRO list for a class, or nil if not found.
// This is a logger-free convenience method for use by packages that cannot import output.
func (r *ThirdPartyRegistryRemote) GetClassMRO(moduleName, className string) []string {
	r.CacheMutex.RLock()
	module, ok := r.ModuleCache[moduleName]
	r.CacheMutex.RUnlock()
	if !ok || module == nil {
		return nil
	}
	cls := module.Classes[className]
	if cls == nil {
		return nil
	}
	return cls.MRO
}

// IsSubclassSimple checks if a class is a subclass of parentFQN without requiring a logger.
// This is a convenience method for packages that cannot import output.
func (r *ThirdPartyRegistryRemote) IsSubclassSimple(moduleName, className, parentFQN string) bool {
	mro := r.GetClassMRO(moduleName, className)
	return slices.Contains(mro, parentFQN)
}

// ModuleCount returns the number of modules in the manifest.
func (r *ThirdPartyRegistryRemote) ModuleCount() int {
	if r.Manifest == nil {
		return 0
	}

	return len(r.Manifest.Modules)
}

// CacheSize returns the number of modules currently cached in memory.
func (r *ThirdPartyRegistryRemote) CacheSize() int {
	r.CacheMutex.RLock()
	defer r.CacheMutex.RUnlock()

	return len(r.ModuleCache)
}
