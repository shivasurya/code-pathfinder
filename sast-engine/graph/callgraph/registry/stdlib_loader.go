package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// StdlibRegistryLoader loads stdlib registries from local filesystem.
type StdlibRegistryLoader struct {
	RegistryPath string // Path to registries directory (e.g., "registries/python3.14/stdlib/v1")
}

// LoadRegistry loads manifest and all modules from local directory.
func (l *StdlibRegistryLoader) LoadRegistry() (*core.StdlibRegistry, error) {
	// 1. Load manifest.json
	manifestPath := filepath.Join(l.RegistryPath, "manifest.json")
	manifest, err := l.loadManifestFromFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// 2. Create registry
	registry := core.NewStdlibRegistry()
	registry.Manifest = manifest

	// 3. Load all module JSON files
	successCount := 0
	failCount := 0

	for _, moduleEntry := range manifest.Modules {
		modulePath := filepath.Join(l.RegistryPath, moduleEntry.File)

		module, err := l.loadModuleFromFile(modulePath)
		if err != nil {
			log.Printf("Warning: failed to load module %s: %v", moduleEntry.Name, err)
			failCount++
			continue
		}

		// Verify checksum
		if !l.verifyChecksum(modulePath, moduleEntry.Checksum) {
			log.Printf("Warning: checksum mismatch for module %s", moduleEntry.Name)
			failCount++
			continue
		}

		registry.Modules[module.Module] = module
		successCount++
	}

	if failCount > 0 {
		log.Printf("Loaded %d/%d stdlib modules (%d failed)", successCount, len(manifest.Modules), failCount)
	}

	return registry, nil
}

// loadManifestFromFile loads and parses manifest.json.
func (l *StdlibRegistryLoader) loadManifestFromFile(path string) (*core.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest core.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	return &manifest, nil
}

// loadModuleFromFile loads and parses a module registry JSON file.
func (l *StdlibRegistryLoader) loadModuleFromFile(path string) (*core.StdlibModule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read module file: %w", err)
	}

	var module core.StdlibModule
	if err := json.Unmarshal(data, &module); err != nil {
		return nil, fmt.Errorf("failed to parse module JSON: %w", err)
	}

	return &module, nil
}

// verifyChecksum verifies the SHA256 checksum of a file.
func (l *StdlibRegistryLoader) verifyChecksum(path string, expectedChecksum string) bool {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	// Calculate SHA256
	hash := sha256.Sum256(data)
	actualChecksum := "sha256:" + hex.EncodeToString(hash[:])

	return actualChecksum == expectedChecksum
}
