package registry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *output.Logger {
	return output.NewLoggerWithWriter(output.VerbosityDefault, &bytes.Buffer{})
}

func TestNewStdlibRegistryRemote(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com/registries", "3.14")

	assert.Equal(t, "https://example.com/registries", remote.BaseURL)
	assert.Equal(t, "3.14", remote.PythonVersion)
	assert.NotNil(t, remote.ModuleCache)
	assert.NotNil(t, remote.HTTPClient)
	assert.Equal(t, 30*time.Second, remote.HTTPClient.Timeout)
}

func TestNewStdlibRegistryRemote_TrimsSuffix(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com/registries/", "3.14")
	assert.Equal(t, "https://example.com/registries", remote.BaseURL)
}

func TestStdlibRegistryRemote_LoadManifest_Success(t *testing.T) {
	// Create test manifest
	manifest := core.Manifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "1.0.0",
		Modules: []*core.ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:abc123"},
			{Name: "sys", File: "sys.json", Checksum: "sha256:def456"},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/python3.14/stdlib/v1/manifest.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write(manifestJSON)
	}))
	defer server.Close()

	// Test
	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())

	require.NoError(t, err)
	assert.NotNil(t, remote.Manifest)
	assert.Equal(t, "1.0.0", remote.Manifest.SchemaVersion)
	assert.Len(t, remote.Manifest.Modules, 2)
}

func TestStdlibRegistryRemote_LoadManifest_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest download failed with status: 404")
}

func TestStdlibRegistryRemote_LoadManifest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest JSON")
}

func TestStdlibRegistryRemote_GetModule_Success(t *testing.T) {
	// Create test module
	module := core.StdlibModule{
		Module:        "os",
		PythonVersion: "3.14",
		Functions: map[string]*core.StdlibFunction{
			"getcwd": {ReturnType: "str"},
		},
	}
	moduleJSON, _ := json.Marshal(module)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:fb04c597a080bf9cba624b9e3d809bcd8339379368c2eeb3c8c04ae56f5d5ee1"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	// Test
	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	loadedModule, err := remote.GetModule("os", newTestLogger())
	require.NoError(t, err)
	assert.NotNil(t, loadedModule)
	assert.Equal(t, "os", loadedModule.Module)
	assert.Equal(t, "3.14", loadedModule.PythonVersion)
	assert.Len(t, loadedModule.Functions, 1)
}

func TestStdlibRegistryRemote_GetModule_Caching(t *testing.T) {
	downloadCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:809e7ae20b2cc78116920277412fc74e7669752fc3f807a7eeef91b36188d34f"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			downloadCount++
			module := core.StdlibModule{Module: "os", PythonVersion: "3.14"}
			moduleJSON, _ := json.Marshal(module)
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	// First call - should download
	module1, err := remote.GetModule("os", newTestLogger())
	require.NoError(t, err)
	assert.NotNil(t, module1)
	assert.Equal(t, 1, downloadCount)

	// Second call - should use cache
	module2, err := remote.GetModule("os", newTestLogger())
	require.NoError(t, err)
	assert.NotNil(t, module2)
	assert.Equal(t, 1, downloadCount, "Should not download again")

	// Verify cache size
	assert.Equal(t, 1, remote.CacheSize())
}

func TestStdlibRegistryRemote_GetModule_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := core.Manifest{
			SchemaVersion: "1.0.0",
			Modules: []*core.ModuleEntry{
				{Name: "os", File: "os.json", Checksum: "sha256:abc"},
			},
		}
		manifestJSON, _ := json.Marshal(manifest)
		w.Write(manifestJSON)
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	module, err := remote.GetModule("nonexistent", newTestLogger())
	assert.NoError(t, err)
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_GetModule_ManifestNotLoaded(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com", "3.14")
	module, err := remote.GetModule("os", newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not loaded")
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_GetModule_ChecksumMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:wrongchecksum"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			module := core.StdlibModule{Module: "os"}
			moduleJSON, _ := json.Marshal(module)
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	module, err := remote.GetModule("os", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_HasModule(t *testing.T) {
	manifest := core.Manifest{
		SchemaVersion: "1.0.0",
		Modules: []*core.ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:abc"},
		},
	}

	remote := NewStdlibRegistryRemote("https://example.com", "3.14")
	remote.Manifest = &manifest

	assert.True(t, remote.HasModule("os"))
	assert.False(t, remote.HasModule("nonexistent"))
}

func TestStdlibRegistryRemote_HasModule_ManifestNotLoaded(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com", "3.14")
	assert.False(t, remote.HasModule("os"))
}

func TestStdlibRegistryRemote_GetFunction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:b00ae23881127c94ad43008c8c45ca1feea852cc149acce4f81648677befeb00"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			module := core.StdlibModule{
				Module: "os",
				Functions: map[string]*core.StdlibFunction{
					"getcwd": {ReturnType: "str"},
				},
			}
			moduleJSON, _ := json.Marshal(module)
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	fn := remote.GetFunction("os", "getcwd", newTestLogger())
	require.NotNil(t, fn, "GetFunction should return non-nil function")
	assert.Equal(t, "str", fn.ReturnType)
}

func TestStdlibRegistryRemote_GetClass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "pathlib", File: "pathlib.json", Checksum: "sha256:40fdc2a17eb383a81c197d8b2453e2a99605cfb1fa5c91e25f3f905ac803c7b8"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/pathlib.json" {
			module := core.StdlibModule{
				Module: "pathlib",
				Classes: map[string]*core.StdlibClass{
					"Path": {Type: "class"},
				},
			}
			moduleJSON, _ := json.Marshal(module)
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	cls := remote.GetClass("pathlib", "Path", newTestLogger())
	require.NotNil(t, cls, "GetClass should return non-nil class")
	assert.Equal(t, "class", cls.Type)
}

func TestStdlibRegistryRemote_ModuleCount(t *testing.T) {
	manifest := core.Manifest{
		SchemaVersion: "1.0.0",
		Modules: []*core.ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:abc"},
			{Name: "sys", File: "sys.json", Checksum: "sha256:def"},
		},
	}

	remote := NewStdlibRegistryRemote("https://example.com", "3.14")
	remote.Manifest = &manifest

	assert.Equal(t, 2, remote.ModuleCount())
}

func TestStdlibRegistryRemote_ModuleCount_NoManifest(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com", "3.14")
	assert.Equal(t, 0, remote.ModuleCount())
}

func TestStdlibRegistryRemote_ClearCache(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com", "3.14")
	remote.ModuleCache["os"] = &core.StdlibModule{Module: "os"}
	remote.ModuleCache["sys"] = &core.StdlibModule{Module: "sys"}

	assert.Equal(t, 2, remote.CacheSize())

	remote.ClearCache()

	assert.Equal(t, 0, remote.CacheSize())
}

func TestStdlibRegistryRemote_VerifyChecksum(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com", "3.14")

	data := []byte(`{"test": "data"}`)
	// Calculated using: echo -n '{"test": "data"}' | shasum -a 256
	validChecksum := "sha256:40b61fe1b15af0a4d5402735b26343e8cf8a045f4d81710e6108a21d91eaf366"
	invalidChecksum := "sha256:wronghash"

	assert.True(t, remote.verifyChecksum(data, validChecksum))
	assert.False(t, remote.verifyChecksum(data, invalidChecksum))
}

func TestStdlibRegistryRemote_GetFunction_ModuleNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := core.Manifest{
			SchemaVersion: "1.0.0",
			Modules: []*core.ModuleEntry{
				{Name: "sys", File: "sys.json", Checksum: "sha256:abc"},
			},
		}
		manifestJSON, _ := json.Marshal(manifest)
		w.Write(manifestJSON)
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	fn := remote.GetFunction("os", "getcwd", newTestLogger())
	assert.Nil(t, fn)
}

func TestStdlibRegistryRemote_GetFunction_FunctionNotFound(t *testing.T) {
	module := core.StdlibModule{
		Module: "os",
		Functions: map[string]*core.StdlibFunction{
			"getcwd": {ReturnType: "str"},
		},
	}
	moduleJSON, _ := json.Marshal(module)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:b00ae23881127c94ad43008c8c45ca1feea852cc149acce4f81648677befeb00"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	// Request a function that doesn't exist
	fn := remote.GetFunction("os", "nonexistent", newTestLogger())
	assert.Nil(t, fn)
}

func TestStdlibRegistryRemote_GetClass_ModuleNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := core.Manifest{
			SchemaVersion: "1.0.0",
			Modules: []*core.ModuleEntry{
				{Name: "sys", File: "sys.json", Checksum: "sha256:abc"},
			},
		}
		manifestJSON, _ := json.Marshal(manifest)
		w.Write(manifestJSON)
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	cls := remote.GetClass("os", "Path", newTestLogger())
	assert.Nil(t, cls)
}

func TestStdlibRegistryRemote_GetClass_ClassNotFound(t *testing.T) {
	module := core.StdlibModule{
		Module: "pathlib",
		Classes: map[string]*core.StdlibClass{
			"Path": {Type: "class"},
		},
	}
	moduleJSON, _ := json.Marshal(module)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "pathlib", File: "pathlib.json", Checksum: "sha256:40fdc2a17eb383a81c197d8b2453e2a99605cfb1fa5c91e25f3f905ac803c7b8"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/pathlib.json" {
			w.Write(moduleJSON)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	// Request a class that doesn't exist
	cls := remote.GetClass("pathlib", "NonExistent", newTestLogger())
	assert.Nil(t, cls)
}

func TestStdlibRegistryRemote_GetFunction_ModuleLoadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:wrongchecksum"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.Write([]byte(`{"module": "os"}`))
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	// This will trigger an error in GetModule due to checksum mismatch
	fn := remote.GetFunction("os", "getcwd", newTestLogger())
	assert.Nil(t, fn)
}

func TestStdlibRegistryRemote_GetClass_ModuleLoadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "pathlib", File: "pathlib.json", Checksum: "sha256:wrongchecksum"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/pathlib.json" {
			w.Write([]byte(`{"module": "pathlib"}`))
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	// This will trigger an error in GetModule due to checksum mismatch
	cls := remote.GetClass("pathlib", "Path", newTestLogger())
	assert.Nil(t, cls)
}

func TestStdlibRegistryRemote_LoadManifest_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		// Close connection immediately to cause read error
	}))
	server.Close() // Close server to cause connection error

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download manifest")
}

func TestStdlibRegistryRemote_GetModule_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:9e1ff4275ee1300de350456bdb3d63d7a66e565f65181e8f94f329a782503d26"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.Write([]byte("invalid json"))
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	module, err := remote.GetModule("os", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse module JSON")
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_GetClassMethod(t *testing.T) {
	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"sqlite3": {
				Classes: map[string]*core.StdlibClass{
					"Connection": {
						Methods: map[string]*core.StdlibFunction{
							"cursor": {ReturnType: "sqlite3.Cursor", Confidence: 0.9},
						},
					},
				},
			},
		},
	}
	logger := newTestLogger()

	method := remote.GetClassMethod("sqlite3", "Connection", "cursor", logger)
	assert.NotNil(t, method)
	assert.Equal(t, "sqlite3.Cursor", method.ReturnType)

	// Non-existent method
	assert.Nil(t, remote.GetClassMethod("sqlite3", "Connection", "nonexistent", logger))

	// Non-existent class
	assert.Nil(t, remote.GetClassMethod("sqlite3", "NonExistent", "cursor", logger))

	// Non-existent module
	assert.Nil(t, remote.GetClassMethod("nonexistent", "Connection", "cursor", logger))
}

func TestStdlibRegistryRemote_GetClassMethod_Inherited(t *testing.T) {
	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"sqlite3": {
				Classes: map[string]*core.StdlibClass{
					"Connection": {
						Methods: map[string]*core.StdlibFunction{
							"cursor": {ReturnType: "sqlite3.Cursor", Confidence: 0.9},
						},
						InheritedMethods: map[string]*core.InheritedMember{
							"close": {ReturnType: "None", Confidence: 0.8, Source: "object"},
						},
					},
				},
			},
		},
	}
	logger := newTestLogger()

	// Inherited method should be found
	method := remote.GetClassMethod("sqlite3", "Connection", "close", logger)
	assert.NotNil(t, method)
	assert.Equal(t, "None", method.ReturnType)
	assert.Equal(t, float32(0.8), method.Confidence)

	// Own method takes priority over inherited
	own := remote.GetClassMethod("sqlite3", "Connection", "cursor", logger)
	assert.NotNil(t, own)
	assert.Equal(t, "sqlite3.Cursor", own.ReturnType)
}

func TestStdlibRegistryRemote_VerifyChecksum_EdgeCases(t *testing.T) {
	remote := NewStdlibRegistryRemote("https://example.com", "3.14")

	tests := []struct {
		name     string
		data     []byte
		checksum string
		expected bool
	}{
		{
			name:     "invalid format no sha256 prefix",
			data:     []byte(`{"test": "data"}`),
			checksum: "40b61fe1b15af0a4d5402735b26343e8cf8a045f4d81710e6108a21d91eaf366",
			expected: false,
		},
		{
			name:     "uppercase hex is rejected (case sensitive)",
			data:     []byte(`{"test": "data"}`),
			checksum: "sha256:40B61FE1B15AF0A4D5402735B26343E8CF8A045F4D81710E6108A21D91EAF366",
			expected: false,
		},
		{
			name:     "empty data with correct hash",
			data:     []byte{},
			checksum: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := remote.verifyChecksum(tt.data, tt.checksum)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStdlibRegistryRemote_GetClassMRO(t *testing.T) {
	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"collections": {
				Classes: map[string]*core.StdlibClass{
					"OrderedDict": {
						MRO: []string{"collections.OrderedDict", "builtins.dict", "builtins.object"},
					},
					"EmptyMRO": {
						MRO: []string{},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		module     string
		class      string
		expected   []string
		expectNil  bool
	}{
		{
			name:     "module in cache with MRO",
			module:   "collections",
			class:    "OrderedDict",
			expected: []string{"collections.OrderedDict", "builtins.dict", "builtins.object"},
		},
		{
			name:      "module in cache class not found",
			module:    "collections",
			class:     "NonExistent",
			expectNil: true,
		},
		{
			name:      "module not in cache",
			module:    "nonexistent",
			class:     "SomeClass",
			expectNil: true,
		},
		{
			name:     "class has empty MRO",
			module:   "collections",
			class:    "EmptyMRO",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := remote.GetClassMRO(tt.module, tt.class)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStdlibRegistryRemote_IsSubclassSimple(t *testing.T) {
	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"collections": {
				Classes: map[string]*core.StdlibClass{
					"OrderedDict": {
						MRO: []string{"collections.OrderedDict", "builtins.dict", "builtins.object"},
					},
				},
			},
		},
	}

	tests := []struct {
		name      string
		module    string
		class     string
		parentFQN string
		expected  bool
	}{
		{
			name:      "class is subclass",
			module:    "collections",
			class:     "OrderedDict",
			parentFQN: "builtins.dict",
			expected:  true,
		},
		{
			name:      "class is not subclass",
			module:    "collections",
			class:     "OrderedDict",
			parentFQN: "builtins.list",
			expected:  false,
		},
		{
			name:      "module not in cache",
			module:    "nonexistent",
			class:     "SomeClass",
			parentFQN: "builtins.object",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := remote.IsSubclassSimple(tt.module, tt.class, tt.parentFQN)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStdlibRegistryRemote_GetModule_HTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:abc"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	module, err := remote.GetModule("os", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module download failed with status: 500")
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_GetModule_ConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:abc"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		}
	}))

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	// Close server to simulate connection error
	server.Close()

	module, err := remote.GetModule("os", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download module os")
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_GetModule_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/python3.14/stdlib/v1/manifest.json" {
			manifest := core.Manifest{
				SchemaVersion: "1.0.0",
				Modules: []*core.ModuleEntry{
					{Name: "os", File: "os.json", Checksum: "sha256:abc"},
				},
			}
			manifestJSON, _ := json.Marshal(manifest)
			w.Write(manifestJSON)
		} else if r.URL.Path == "/python3.14/stdlib/v1/os.json" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	remote := NewStdlibRegistryRemote(server.URL, "3.14")
	err := remote.LoadManifest(newTestLogger())
	require.NoError(t, err)

	module, err := remote.GetModule("os", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module download failed with status: 404")
	assert.Nil(t, module)
}

func TestStdlibRegistryRemote_FindClassMethodAlias(t *testing.T) {
	// tarfile.open is a module-level alias for TarFile.open (classmethod)
	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"tarfile": {
				Module:    "tarfile",
				Functions: map[string]*core.StdlibFunction{},
				Classes: map[string]*core.StdlibClass{
					"TarFile": {
						Type: "class",
						Methods: map[string]*core.StdlibFunction{
							"open":       {ReturnType: "tarfile.Self", Confidence: 0.95},
							"extractall": {ReturnType: "builtins.NoneType", Confidence: 0.95},
						},
					},
					"TarInfo": {
						Type: "class",
						Methods: map[string]*core.StdlibFunction{
							"isdir": {ReturnType: "builtins.bool", Confidence: 0.9},
						},
					},
				},
			},
		},
	}
	logger := newTestLogger()

	// Found: tarfile.open → TarFile.open
	method, className := remote.FindClassMethodAlias("tarfile", "open", logger)
	require.NotNil(t, method, "should find TarFile.open as alias for tarfile.open")
	assert.Equal(t, "TarFile", className)
	assert.Equal(t, "tarfile.Self", method.ReturnType)
	assert.InDelta(t, 0.95, method.Confidence, 0.001)

	// Found: tarfile.extractall → TarFile.extractall
	method2, className2 := remote.FindClassMethodAlias("tarfile", "extractall", logger)
	require.NotNil(t, method2)
	assert.Equal(t, "TarFile", className2)
	assert.Equal(t, "builtins.NoneType", method2.ReturnType)

	// Found: tarfile.isdir → TarInfo.isdir
	method3, className3 := remote.FindClassMethodAlias("tarfile", "isdir", logger)
	require.NotNil(t, method3)
	assert.Equal(t, "TarInfo", className3)

	// Not found: no class has "nonexistent"
	method4, className4 := remote.FindClassMethodAlias("tarfile", "nonexistent", logger)
	assert.Nil(t, method4)
	assert.Equal(t, "", className4)

	// Not found: module not in cache
	method5, className5 := remote.FindClassMethodAlias("unknown", "open", logger)
	assert.Nil(t, method5)
	assert.Equal(t, "", className5)
}

func TestStdlibRegistryRemote_FindClassMethodAlias_PrefersModuleNameClass(t *testing.T) {
	// When multiple classes have the same method name, prefer the class
	// whose name matches the module (case-insensitive).
	// Use many non-matching classes to increase chance the preferred one
	// is not the first iterated (Go map order is random).
	classes := map[string]*core.StdlibClass{
		"MyMod": {
			Type: "class",
			Methods: map[string]*core.StdlibFunction{
				"connect": {ReturnType: "mymod.MyMod", Confidence: 0.95},
			},
		},
	}
	// Add many non-matching classes with the same method to force iteration
	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"} {
		classes[name] = &core.StdlibClass{
			Type: "class",
			Methods: map[string]*core.StdlibFunction{
				"connect": {ReturnType: "mymod." + name, Confidence: 0.8},
			},
		}
	}

	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"mymod": {
				Module:    "mymod",
				Functions: map[string]*core.StdlibFunction{},
				Classes:   classes,
			},
		},
	}

	// Run multiple times to exercise different map iteration orders
	for i := 0; i < 20; i++ {
		method, className := remote.FindClassMethodAlias("mymod", "connect", newTestLogger())
		require.NotNil(t, method, "iteration %d: should find connect", i)
		assert.Equal(t, "MyMod", className, "iteration %d: should always prefer class matching module name", i)
		assert.Equal(t, "mymod.MyMod", method.ReturnType, "iteration %d", i)
	}
}

func TestStdlibRegistryRemote_FindClassMethodAlias_Inherited(t *testing.T) {
	// FindClassMethodAlias uses GetClassMethod which checks inherited methods too.
	remote := &StdlibRegistryRemote{
		ModuleCache: map[string]*core.StdlibModule{
			"mymod": {
				Module:    "mymod",
				Functions: map[string]*core.StdlibFunction{},
				Classes: map[string]*core.StdlibClass{
					"Base": {
						Type:    "class",
						Methods: map[string]*core.StdlibFunction{},
						InheritedMethods: map[string]*core.InheritedMember{
							"read": {ReturnType: "builtins.bytes", Confidence: 0.85, Source: "io.IOBase"},
						},
					},
				},
			},
		},
	}

	method, className := remote.FindClassMethodAlias("mymod", "read", newTestLogger())
	require.NotNil(t, method, "should find inherited method via GetClassMethod")
	assert.Equal(t, "Base", className)
	assert.Equal(t, "builtins.bytes", method.ReturnType)
}
