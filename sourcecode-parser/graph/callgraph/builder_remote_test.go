package callgraph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCallGraph_RemoteStdlibLoading(t *testing.T) {
	// Create test manifest
	manifest := Manifest{
		SchemaVersion: "1.0.0",
		Modules: []*ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:fb04c597a080bf9cba624b9e3d809bcd8339379368c2eeb3c8c04ae56f5d5ee1"},
		},
	}

	// Create test module
	module := StdlibModule{
		Module:        "os",
		PythonVersion: "3.14",
		Functions: map[string]*StdlibFunction{
			"getcwd": {ReturnType: "str"},
		},
	}

	// Create mock CDN server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			moduleJSON, _ := json.Marshal(module)
			w.Write(moduleJSON)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create temporary project directory
	tmpDir := t.TempDir()

	// Write .python-version file
	versionFile := filepath.Join(tmpDir, ".python-version")
	err := os.WriteFile(versionFile, []byte("3.14.0\n"), 0644)
	require.NoError(t, err)

	// Create a simple code graph with a Python file
	codeGraph := graph.NewCodeGraph()
	registry := NewModuleRegistry()

	// Note: We can't fully test BuildCallGraph here because it needs a real code graph
	// Instead, we test the individual components that BuildCallGraph uses

	// Test 1: Version detection
	version := detectPythonVersion(tmpDir)
	assert.Equal(t, "3.14", version)

	// Test 2: Remote loader initialization
	remoteLoader := NewStdlibRegistryRemote(server.URL, version)
	err = remoteLoader.LoadManifest()
	require.NoError(t, err)
	assert.Equal(t, 1, remoteLoader.ModuleCount())

	// Test 3: Module lazy loading
	osModule, err := remoteLoader.GetModule("os")
	require.NoError(t, err)
	assert.NotNil(t, osModule)
	assert.Equal(t, "os", osModule.Module)

	// Test 4: Verify cache works
	assert.Equal(t, 1, remoteLoader.CacheSize())

	// Minimal call graph build to verify no compilation errors
	_, err = BuildCallGraph(codeGraph, registry, tmpDir)
	// We expect this to succeed even with empty graph
	assert.NoError(t, err)
}

func TestBuildCallGraph_RemoteStdlibFallback(t *testing.T) {
	// Create a server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	codeGraph := graph.NewCodeGraph()
	registry := NewModuleRegistry()

	// BuildCallGraph should succeed even if CDN is unavailable
	// It should log a warning and continue without stdlib resolution
	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, callGraph)
}

func TestValidateStdlibFQN_WithRemoteLoader(t *testing.T) {
	// Create test module
	module := StdlibModule{
		Module:        "os",
		PythonVersion: "3.14",
		Functions: map[string]*StdlibFunction{
			"getcwd": {ReturnType: "str"},
		},
		Classes: map[string]*StdlibClass{
			"DirEntry": {Type: "class"},
		},
	}

	// Calculate checksum
	moduleJSON, _ := json.Marshal(module)

	// Create manifest with correct checksum
	manifest := Manifest{
		SchemaVersion: "1.0.0",
		Modules: []*ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:4cfe6f2495a04780243e6c0c32720082a774cb2f99a4e5c68db2b8623ec11919"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remoteLoader := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remoteLoader.LoadManifest()
	require.NoError(t, err)

	// Test function resolution
	assert.True(t, validateStdlibFQN("os.getcwd", remoteLoader))

	// Test class resolution
	assert.True(t, validateStdlibFQN("os.DirEntry", remoteLoader))

	// Test non-existent function
	assert.False(t, validateStdlibFQN("os.nonexistent", remoteLoader))

	// Test non-existent module
	assert.False(t, validateStdlibFQN("fake.module", remoteLoader))

	// Test nil loader
	assert.False(t, validateStdlibFQN("os.getcwd", nil))

	// Test invalid FQN (too short)
	assert.False(t, validateStdlibFQN("os", remoteLoader))
}

func TestValidateStdlibFQN_ModuleAlias(t *testing.T) {
	// Create posixpath module (alias for os.path on POSIX systems)
	module := StdlibModule{
		Module:        "posixpath",
		PythonVersion: "3.14",
		Functions: map[string]*StdlibFunction{
			"join": {ReturnType: "str"},
		},
	}

	moduleJSON, _ := json.Marshal(module)

	manifest := Manifest{
		SchemaVersion: "1.0.0",
		Modules: []*ModuleEntry{
			{Name: "posixpath", File: "posixpath.json", Checksum: "sha256:b8fe94908624c2d0e9157477e50b916617202ccffbad4ec35f05b4ff0d16840c"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/posixpath.json" {
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remoteLoader := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remoteLoader.LoadManifest()
	require.NoError(t, err)

	// Test that os.path.join is resolved to posixpath.join via alias
	assert.True(t, validateStdlibFQN("os.path.join", remoteLoader))
}

func TestDetectPythonVersion_Integration(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		expected string
	}{
		{
			name: "from .python-version file",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, ".python-version"), []byte("3.11.5"), 0644)
			},
			expected: "3.11",
		},
		{
			name: "from pyproject.toml requires-python",
			setup: func(dir string) {
				content := `[project]
requires-python = ">=3.10"
`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expected: "3.10",
		},
		{
			name: "from pyproject.toml poetry",
			setup: func(dir string) {
				content := `[tool.poetry.dependencies]
python = "^3.9"
`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expected: "3.9",
		},
		{
			name:     "default version",
			setup:    func(dir string) {},
			expected: "3.14",
		},
		{
			name: "priority: .python-version over pyproject.toml",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, ".python-version"), []byte("3.12"), 0644)
				content := `[project]
requires-python = ">=3.8"
`
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)
			},
			expected: "3.12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)
			version := detectPythonVersion(tmpDir)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestRemoteLoader_CachingInBuildCallGraph(t *testing.T) {
	downloadCount := 0

	// Create test module
	module := StdlibModule{
		Module:        "os",
		PythonVersion: "3.14",
		Functions: map[string]*StdlibFunction{
			"getcwd": {ReturnType: "str"},
		},
	}
	moduleJSON, _ := json.Marshal(module)

	manifest := Manifest{
		SchemaVersion: "1.0.0",
		Modules: []*ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:fb04c597a080bf9cba624b9e3d809bcd8339379368c2eeb3c8c04ae56f5d5ee1"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			downloadCount++
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remoteLoader := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remoteLoader.LoadManifest()
	require.NoError(t, err)

	// Call validateStdlibFQN multiple times
	validateStdlibFQN("os.getcwd", remoteLoader)
	validateStdlibFQN("os.getcwd", remoteLoader)
	validateStdlibFQN("os.getcwd", remoteLoader)

	// Module should only be downloaded once
	assert.Equal(t, 1, downloadCount, "Module should be cached after first download")
	assert.Equal(t, 1, remoteLoader.CacheSize())
}
