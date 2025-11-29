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
