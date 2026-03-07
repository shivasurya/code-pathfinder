package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger is defined in stdlib_remote_test.go (same package)

// ── Test helpers ─────────────────────────────────────────────────────────────

func sha256Checksum(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

// buildTestModule creates a realistic module JSON for testing.
func buildTestModule() *core.StdlibModule {
	return &core.StdlibModule{
		Module:        "requests",
		PythonVersion: "any",
		Functions: map[string]*core.StdlibFunction{
			"get": {ReturnType: "requests.Response", Confidence: 0.95,
				Params: []*core.FunctionParam{{Name: "url", Type: "builtins.str", Required: true}},
				Source: "typeshed"},
		},
		Classes: map[string]*core.StdlibClass{
			"Response": {
				Type: "class",
				Methods: map[string]*core.StdlibFunction{
					"json": {ReturnType: "builtins.dict", Confidence: 0.95, Source: "typeshed"},
				},
				Attributes: map[string]*core.StdlibAttribute{
					"status_code": {Type: "builtins.int", Confidence: 0.95, Kind: "attribute", Source: "typeshed"},
					"content":     {Type: "builtins.bytes", Confidence: 0.95, Kind: "property", Source: "typeshed"},
				},
				Bases: []string{"requests.models.BaseResponse"},
				MRO:   []string{"requests.Response", "requests.models.BaseResponse", "builtins.object"},
				InheritedMethods: map[string]*core.InheritedMember{
					"close": {ReturnType: "builtins.NoneType", Confidence: 0.95, Source: "typeshed",
						InheritedFrom: "requests.models.BaseResponse"},
				},
				InheritedAttributes: map[string]*core.InheritedMember{
					"encoding": {Type: "builtins.str", Confidence: 0.95, Source: "typeshed",
						Kind: "attribute", InheritedFrom: "requests.models.BaseResponse"},
				},
			},
			"Session": {
				Type: "class",
				Methods: map[string]*core.StdlibFunction{
					"get":  {ReturnType: "requests.Response", Confidence: 0.95, Source: "typeshed"},
					"post": {ReturnType: "requests.Response", Confidence: 0.95, Source: "typeshed"},
				},
				MRO: []string{"requests.Session", "builtins.object"},
			},
		},
		Constants:  map[string]*core.StdlibConstant{},
		Attributes: map[string]*core.StdlibAttribute{},
	}
}

// startTestServer creates an httptest server serving manifest + module JSON.
func startTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	module := buildTestModule()
	moduleJSON, err := json.Marshal(module)
	require.NoError(t, err)

	checksum := sha256Checksum(moduleJSON)

	manifest := core.Manifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		Modules: []*core.ModuleEntry{
			{Name: "requests", File: "requests_thirdparty.json",
				SizeBytes: int64(len(moduleJSON)), Checksum: checksum},
		},
		Statistics: &core.RegistryStats{TotalModules: 1, TotalFunctions: 1, TotalClasses: 2},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/thirdparty/v1/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest) //nolint:errcheck
		case "/thirdparty/v1/requests_thirdparty.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write(moduleJSON) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
}

// ── Constructor tests ────────────────────────────────────────────────────────

func TestNewThirdPartyRegistryRemote(t *testing.T) {
	r := NewThirdPartyRegistryRemote("https://cdn.example.com/registries")
	assert.Equal(t, "https://cdn.example.com/registries", r.BaseURL)
	assert.NotNil(t, r.ModuleCache)
	assert.Nil(t, r.Manifest)
	assert.NotNil(t, r.HTTPClient)
}

func TestNewThirdPartyRegistryRemote_TrimsTrailingSlash(t *testing.T) {
	r := NewThirdPartyRegistryRemote("https://cdn.example.com/")
	assert.Equal(t, "https://cdn.example.com", r.BaseURL)
}

// ── LoadManifest tests ──────────────────────────────────────────────────────

func TestLoadManifest(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	err := r.LoadManifest(newTestLogger())

	assert.NoError(t, err)
	assert.NotNil(t, r.Manifest)
	assert.Len(t, r.Manifest.Modules, 1)
	assert.Equal(t, "requests", r.Manifest.Modules[0].Name)
}

func TestLoadManifest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	err := r.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status: 500")
}

func TestThirdPartyLoadManifest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json")) //nolint:errcheck
	}))
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	err := r.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse manifest JSON")
}

// ── GetModule tests ─────────────────────────────────────────────────────────

func TestGetModule_LazyLoad(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	module, err := r.GetModule("requests", newTestLogger())
	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, "requests", module.Module)
	assert.Contains(t, module.Functions, "get")
	assert.Contains(t, module.Classes, "Response")
}

func TestGetModule_CacheHit(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	// First call downloads
	mod1, err := r.GetModule("requests", newTestLogger())
	require.NoError(t, err)

	// Second call returns cached
	mod2, err := r.GetModule("requests", newTestLogger())
	require.NoError(t, err)

	assert.Same(t, mod1, mod2)
	assert.Equal(t, 1, r.CacheSize())
}

func TestGetModule_NotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	module, err := r.GetModule("nonexistent", newTestLogger())
	assert.NoError(t, err)
	assert.Nil(t, module)
}

func TestGetModule_ManifestNotLoaded(t *testing.T) {
	r := NewThirdPartyRegistryRemote("https://cdn.example.com")

	_, err := r.GetModule("requests", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not loaded")
}

func TestGetModule_ChecksumMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/thirdparty/v1/manifest.json":
			manifest := core.Manifest{
				Modules: []*core.ModuleEntry{
					{Name: "bad", File: "bad_thirdparty.json", Checksum: "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
				},
			}
			json.NewEncoder(w).Encode(manifest) //nolint:errcheck
		case "/thirdparty/v1/bad_thirdparty.json":
			w.Write([]byte(`{"module":"bad"}`)) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	_, err := r.GetModule("bad", newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestGetModule_Concurrent(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			mod, err := r.GetModule("requests", newTestLogger())
			assert.NoError(t, err)
			assert.NotNil(t, mod)
		})
	}
	wg.Wait()

	assert.Equal(t, 1, r.CacheSize())
}

// ── HasModule tests ─────────────────────────────────────────────────────────

func TestHasModule(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.True(t, r.HasModule("requests"))
	assert.False(t, r.HasModule("nonexistent"))
	assert.Equal(t, 0, r.CacheSize()) // HasModule should not trigger download
}

func TestHasModule_NoManifest(t *testing.T) {
	r := NewThirdPartyRegistryRemote("https://cdn.example.com")
	assert.False(t, r.HasModule("requests"))
}

// ── GetFunction tests ───────────────────────────────────────────────────────

func TestGetFunction(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	fn := r.GetFunction("requests", "get", newTestLogger())
	assert.NotNil(t, fn)
	assert.Equal(t, "requests.Response", fn.ReturnType)
}

func TestThirdPartyGetFunction_NotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	fn := r.GetFunction("requests", "nonexistent", newTestLogger())
	assert.Nil(t, fn)
}

func TestGetFunction_ModuleNotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	fn := r.GetFunction("nonexistent", "get", newTestLogger())
	assert.Nil(t, fn)
}

func TestGetFunction_ModuleError(t *testing.T) {
	r := NewThirdPartyRegistryRemote("https://cdn.example.com")
	// Manifest not loaded → GetModule returns error → GetFunction returns nil
	fn := r.GetFunction("requests", "get", newTestLogger())
	assert.Nil(t, fn)
}

// ── GetClass tests ──────────────────────────────────────────────────────────

func TestGetClass(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	cls := r.GetClass("requests", "Response", newTestLogger())
	assert.NotNil(t, cls)
	assert.Equal(t, "class", cls.Type)
	assert.Contains(t, cls.Methods, "json")
	assert.Contains(t, cls.Attributes, "status_code")
	assert.Equal(t, []string{"requests.models.BaseResponse"}, cls.Bases)
	assert.Len(t, cls.MRO, 3)
}

func TestGetClass_NotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	cls := r.GetClass("requests", "NonExistent", newTestLogger())
	assert.Nil(t, cls)
}

func TestGetClass_ModuleNotInManifest(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	cls := r.GetClass("nonexistent", "Foo", newTestLogger())
	assert.Nil(t, cls)
}

func TestGetClass_ModuleError(t *testing.T) {
	r := NewThirdPartyRegistryRemote("https://cdn.example.com")
	cls := r.GetClass("requests", "Response", newTestLogger())
	assert.Nil(t, cls)
}

// ── GetClassAttribute tests ─────────────────────────────────────────────────

func TestGetClassAttribute_OwnAttribute(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	attr := r.GetClassAttribute("requests", "Response", "status_code", newTestLogger())
	assert.NotNil(t, attr)
	assert.Equal(t, "builtins.int", attr.Type)
	assert.Equal(t, "attribute", attr.Kind)
}

func TestGetClassAttribute_Property(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	attr := r.GetClassAttribute("requests", "Response", "content", newTestLogger())
	assert.NotNil(t, attr)
	assert.Equal(t, "builtins.bytes", attr.Type)
	assert.Equal(t, "property", attr.Kind)
}

func TestGetClassAttribute_Inherited(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	attr := r.GetClassAttribute("requests", "Response", "encoding", newTestLogger())
	assert.NotNil(t, attr)
	assert.Equal(t, "builtins.str", attr.Type)
}

func TestGetClassAttribute_NotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	attr := r.GetClassAttribute("requests", "Response", "nonexistent", newTestLogger())
	assert.Nil(t, attr)
}

func TestGetClassAttribute_ClassNotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	attr := r.GetClassAttribute("requests", "NonExistent", "foo", newTestLogger())
	assert.Nil(t, attr)
}

// ── GetClassMethod tests ────────────────────────────────────────────────────

func TestGetClassMethod_OwnMethod(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	method := r.GetClassMethod("requests", "Response", "json", newTestLogger())
	assert.NotNil(t, method)
	assert.Equal(t, "builtins.dict", method.ReturnType)
}

func TestGetClassMethod_Inherited(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	method := r.GetClassMethod("requests", "Response", "close", newTestLogger())
	assert.NotNil(t, method)
	assert.Equal(t, "builtins.NoneType", method.ReturnType)
}

func TestGetClassMethod_NotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	method := r.GetClassMethod("requests", "Response", "nonexistent", newTestLogger())
	assert.Nil(t, method)
}

func TestGetClassMethod_ClassNotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	method := r.GetClassMethod("requests", "NonExistent", "json", newTestLogger())
	assert.Nil(t, method)
}

// ── IsSubclass tests ────────────────────────────────────────────────────────

func TestIsSubclass_DirectParent(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.True(t, r.IsSubclass("requests", "Response", "requests.models.BaseResponse", newTestLogger()))
}

func TestIsSubclass_Self(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.True(t, r.IsSubclass("requests", "Response", "requests.Response", newTestLogger()))
}

func TestIsSubclass_Object(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.True(t, r.IsSubclass("requests", "Response", "builtins.object", newTestLogger()))
}

func TestIsSubclass_NotSubclass(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.False(t, r.IsSubclass("requests", "Response", "django.views.View", newTestLogger()))
}

func TestIsSubclass_ClassNotFound(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.False(t, r.IsSubclass("requests", "NonExistent", "builtins.object", newTestLogger()))
}

func TestIsSubclass_NoMRO(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	// Session has MRO but no BaseResponse in it
	assert.False(t, r.IsSubclass("requests", "Session", "requests.models.BaseResponse", newTestLogger()))
}

// ── ModuleCount / CacheSize tests ───────────────────────────────────────────

func TestModuleCount(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	assert.Equal(t, 0, r.ModuleCount())

	require.NoError(t, r.LoadManifest(newTestLogger()))
	assert.Equal(t, 1, r.ModuleCount())
}

func TestThirdPartyCacheSize(t *testing.T) {
	server := startTestServer(t)
	defer server.Close()

	r := NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, r.LoadManifest(newTestLogger()))

	assert.Equal(t, 0, r.CacheSize())

	_, err := r.GetModule("requests", newTestLogger())
	require.NoError(t, err)

	assert.Equal(t, 1, r.CacheSize())
}

// ── verifyThirdPartyChecksum tests ──────────────────────────────────────────

func TestVerifyThirdPartyChecksum_Valid(t *testing.T) {
	data := []byte(`{"module":"test"}`)
	checksum := sha256Checksum(data)
	assert.True(t, verifyThirdPartyChecksum(data, checksum))
}

func TestVerifyThirdPartyChecksum_Invalid(t *testing.T) {
	data := []byte(`{"module":"test"}`)
	assert.False(t, verifyThirdPartyChecksum(data, "sha256:0000000000000000000000000000000000000000000000000000000000000000"))
}

func TestVerifyThirdPartyChecksum_Deterministic(t *testing.T) {
	// Marshal the same struct twice — Go sorts map keys, so output must be identical.
	// Guards against non-deterministic serialization breaking checksum verification.
	mod := buildTestModule()

	data1, err := json.Marshal(mod)
	require.NoError(t, err)
	data2, err := json.Marshal(mod)
	require.NoError(t, err)

	assert.Equal(t, data1, data2, "json.Marshal must produce identical bytes for identical input")

	checksum := sha256Checksum(data1)
	assert.True(t, verifyThirdPartyChecksum(data2, checksum),
		"checksum of first marshal must verify against second marshal")
}
