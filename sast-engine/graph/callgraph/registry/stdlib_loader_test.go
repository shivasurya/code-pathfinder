package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdlibRegistryLoader_LoadManifest(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create manifest
	manifest := &core.Manifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		PythonVersion: core.PythonVersionInfo{
			Major: 3,
			Minor: 14,
			Patch: 0,
			Full:  "3.14.0",
		},
		Modules: []*core.ModuleEntry{
			{
				Name:      "os",
				File:      "os_stdlib.json",
				SizeBytes: 1000,
				Checksum:  "sha256:abc123",
			},
		},
		Statistics: &core.RegistryStats{
			TotalModules:   1,
			TotalFunctions: 10,
		},
	}

	// Write manifest
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, data, 0644))

	// Load manifest
	loader := &StdlibRegistryLoader{RegistryPath: tmpDir}
	loadedManifest, err := loader.loadManifestFromFile(manifestPath)

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", loadedManifest.SchemaVersion)
	assert.Equal(t, 3, loadedManifest.PythonVersion.Major)
	assert.Equal(t, 1, len(loadedManifest.Modules))
	assert.Equal(t, "os", loadedManifest.Modules[0].Name)
}

func TestStdlibRegistryLoader_LoadModule(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create module
	module := &core.StdlibModule{
		Module:        "os",
		PythonVersion: "3.14.0",
		Functions: map[string]*core.StdlibFunction{
			"getcwd": {
				ReturnType: "builtins.str",
				Confidence: 1.0,
				Params:     []*core.FunctionParam{},
				Source:     "annotation",
			},
		},
		Constants: map[string]*core.StdlibConstant{
			"sep": {
				Type:       "builtins.str",
				Value:      "\"/\"",
				Confidence: 1.0,
			},
		},
		Attributes: map[string]*core.StdlibAttribute{
			"environ": {
				Type:        "os._Environ",
				BehavesLike: "builtins.dict",
				Confidence:  0.9,
			},
		},
	}

	// Write module
	modulePath := filepath.Join(tmpDir, "os_stdlib.json")
	data, err := json.Marshal(module)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(modulePath, data, 0644))

	// Load module
	loader := &StdlibRegistryLoader{RegistryPath: tmpDir}
	loadedModule, err := loader.loadModuleFromFile(modulePath)

	require.NoError(t, err)
	assert.Equal(t, "os", loadedModule.Module)
	assert.Equal(t, 1, len(loadedModule.Functions))
	assert.Equal(t, "builtins.str", loadedModule.Functions["getcwd"].ReturnType)
	assert.Equal(t, 1, len(loadedModule.Constants))
	assert.Equal(t, "builtins.str", loadedModule.Constants["sep"].Type)
	assert.Equal(t, 1, len(loadedModule.Attributes))
	assert.Equal(t, "builtins.dict", loadedModule.Attributes["environ"].BehavesLike)
}

func TestStdlibRegistryLoader_LoadRegistry(t *testing.T) {
	// Use actual generated registries if they exist
	registryPath := "../../../registries/python3.14/stdlib/v1"
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: registries not generated yet")
	}

	loader := &StdlibRegistryLoader{RegistryPath: registryPath}
	registry, err := loader.LoadRegistry()

	require.NoError(t, err)
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Manifest)
	assert.Greater(t, len(registry.Modules), 0)

	// Verify manifest
	assert.Equal(t, "1.0.0", registry.Manifest.SchemaVersion)
	assert.Equal(t, 3, registry.Manifest.PythonVersion.Major)
	assert.Equal(t, 14, registry.Manifest.PythonVersion.Minor)

	// Verify modules loaded
	assert.Greater(t, registry.ModuleCount(), 100) // Should have 180+ modules

	// Verify os module exists
	assert.True(t, registry.HasModule("os"))
	osModule := registry.GetModule("os")
	assert.NotNil(t, osModule)
	assert.Greater(t, len(osModule.Functions), 0)

	// Verify specific function
	getcwd := registry.GetFunction("os", "getcwd")
	assert.NotNil(t, getcwd)

	// Verify constants
	assert.Greater(t, len(osModule.Constants), 0)

	// Verify attributes
	environ := registry.GetAttribute("os", "environ")
	assert.NotNil(t, environ)
}

func TestStdlibRegistryLoader_VerifyChecksum(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")

	content := []byte(`{"test": "data"}`)
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	loader := &StdlibRegistryLoader{RegistryPath: tmpDir}

	// Calculate the actual checksum
	// sha256 of `{"test": "data"}` =40b61fe1b15af0a4d5402735b26343e8cf8a045f4d81710e6108a21d91eaf366
	validChecksum := "sha256:40b61fe1b15af0a4d5402735b26343e8cf8a045f4d81710e6108a21d91eaf366"
	invalidChecksum := "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	// Test valid checksum
	assert.True(t, loader.verifyChecksum(filePath, validChecksum))

	// Test invalid checksum
	assert.False(t, loader.verifyChecksum(filePath, invalidChecksum))
}

func TestStdlibRegistryLoader_MissingManifest(t *testing.T) {
	tmpDir := t.TempDir()

	loader := &StdlibRegistryLoader{RegistryPath: tmpDir}
	_, err := loader.LoadRegistry()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load manifest")
}

func TestStdlibRegistryLoader_CorruptedManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid JSON
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte("not valid json"), 0644))

	loader := &StdlibRegistryLoader{RegistryPath: tmpDir}
	_, err := loader.LoadRegistry()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load manifest")
}

func TestStdlibRegistry_GetMethods(t *testing.T) {
	registry := core.NewStdlibRegistry()

	// Add test module
	registry.Modules["test"] = &core.StdlibModule{
		Module: "test",
		Functions: map[string]*core.StdlibFunction{
			"func1": {ReturnType: "str"},
		},
		Classes: map[string]*core.StdlibClass{
			"Class1": {Type: "class"},
		},
		Constants: map[string]*core.StdlibConstant{
			"CONST1": {Type: "int"},
		},
		Attributes: map[string]*core.StdlibAttribute{
			"attr1": {Type: "dict"},
		},
	}

	// Test GetFunction
	func1 := registry.GetFunction("test", "func1")
	assert.NotNil(t, func1)
	assert.Equal(t, "str", func1.ReturnType)

	// Test GetClass
	class1 := registry.GetClass("test", "Class1")
	assert.NotNil(t, class1)

	// Test GetConstant
	const1 := registry.GetConstant("test", "CONST1")
	assert.NotNil(t, const1)

	// Test GetAttribute
	attr1 := registry.GetAttribute("test", "attr1")
	assert.NotNil(t, attr1)

	// Test non-existent items
	assert.Nil(t, registry.GetFunction("test", "nonexistent"))
	assert.Nil(t, registry.GetFunction("nonexistent", "func1"))
}
