package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRequestsModule creates a test module mimicking the requests library.
func buildRequestsModule() *core.StdlibModule {
	return &core.StdlibModule{
		Module: "requests",
		Functions: map[string]*core.StdlibFunction{
			"get": {
				ReturnType: "requests.models.Response",
				Confidence: 0.95,
				Source:     "typeshed",
			},
			"post": {
				ReturnType: "requests.models.Response",
				Confidence: 0.95,
				Source:     "typeshed",
			},
		},
		Classes: map[string]*core.StdlibClass{
			"Session": {
				Type: "requests.Session",
				Methods: map[string]*core.StdlibFunction{
					"get": {
						ReturnType: "requests.models.Response",
						Confidence: 0.95,
						Source:     "typeshed",
					},
				},
				Attributes: map[string]*core.StdlibAttribute{},
			},
			"models.Response": {
				Type: "requests.models.Response",
				Methods: map[string]*core.StdlibFunction{
					"json": {
						ReturnType: "builtins.object",
						Confidence: 0.90,
						Source:     "typeshed",
					},
					"raise_for_status": {
						ReturnType: "builtins.None",
						Confidence: 0.95,
						Source:     "typeshed",
					},
				},
				Attributes: map[string]*core.StdlibAttribute{
					"status_code": {
						Type:       "builtins.int",
						Confidence: 0.95,
						Kind:       "attribute",
					},
					"text": {
						Type:       "builtins.str",
						Confidence: 0.95,
						Kind:       "property",
					},
					"headers": {
						Type:       "requests.structures.CaseInsensitiveDict",
						Confidence: 0.90,
						Kind:       "attribute",
					},
				},
			},
		},
		Constants:  map[string]*core.StdlibConstant{},
		Attributes: map[string]*core.StdlibAttribute{},
	}
}

// buildRequestsModuleRealistic creates a module matching real CDN structure.
// Functions have sub-module prefixes (api.get), return types reference sub-module paths.
func buildRequestsModuleRealistic() *core.StdlibModule {
	return &core.StdlibModule{
		Module: "requests",
		Functions: map[string]*core.StdlibFunction{
			"api.get": {
				ReturnType: "requests.api.Response",
				Confidence: 0.95,
				Source:     "typeshed",
			},
			"api.post": {
				ReturnType: "requests.api.Response",
				Confidence: 0.95,
				Source:     "typeshed",
			},
		},
		Classes: map[string]*core.StdlibClass{
			"Session": {
				Type: "requests.Session",
				Methods: map[string]*core.StdlibFunction{
					"get": {
						ReturnType: "requests.sessions.Response",
						Confidence: 0.95,
						Source:     "typeshed",
					},
				},
				Attributes: map[string]*core.StdlibAttribute{},
			},
			"Response": {
				Type: "requests.Response",
				Methods: map[string]*core.StdlibFunction{
					"json": {
						ReturnType: "typing.Any",
						Confidence: 0.90,
						Source:     "typeshed",
					},
					"raise_for_status": {
						ReturnType: "builtins.None",
						Confidence: 0.95,
						Source:     "typeshed",
					},
				},
				Attributes: map[string]*core.StdlibAttribute{
					"status_code": {
						Type:       "builtins.int",
						Confidence: 0.95,
						Kind:       "attribute",
					},
					"text": {
						Type:       "builtins.str",
						Confidence: 0.95,
						Kind:       "property",
					},
				},
			},
		},
		Constants:  map[string]*core.StdlibConstant{},
		Attributes: map[string]*core.StdlibAttribute{},
	}
}

// setupRealisticThirdPartyServer creates a mock server matching real CDN structure.
func setupRealisticThirdPartyServer(t *testing.T) (*httptest.Server, *registry.ThirdPartyRegistryRemote) {
	t.Helper()

	module := buildRequestsModuleRealistic()
	moduleJSON, err := json.Marshal(module)
	require.NoError(t, err)

	checksum := "sha256:" + func() string {
		h := sha256.Sum256(moduleJSON)
		return hex.EncodeToString(h[:])
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/thirdparty/v1/manifest.json":
			manifest := core.Manifest{
				Modules: []*core.ModuleEntry{
					{Name: "requests", File: "requests.json", Checksum: checksum},
				},
			}
			json.NewEncoder(w).Encode(manifest) //nolint:errcheck
		case "/thirdparty/v1/requests.json":
			w.Write(moduleJSON) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))

	loader := registry.NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, loader.LoadManifest(output.NewLogger(output.VerbosityDefault)))

	return server, loader
}

// setupThirdPartyServer creates a mock HTTP server and loader with the requests module.
func setupThirdPartyServer(t *testing.T) (*httptest.Server, *registry.ThirdPartyRegistryRemote) {
	t.Helper()

	module := buildRequestsModule()
	moduleJSON, err := json.Marshal(module)
	require.NoError(t, err)

	checksum := "sha256:" + func() string {
		h := sha256.Sum256(moduleJSON)
		return hex.EncodeToString(h[:])
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/thirdparty/v1/manifest.json":
			manifest := core.Manifest{
				Modules: []*core.ModuleEntry{
					{Name: "requests", File: "requests.json", Checksum: checksum},
				},
			}
			json.NewEncoder(w).Encode(manifest) //nolint:errcheck
		case "/thirdparty/v1/requests.json":
			w.Write(moduleJSON) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))

	loader := registry.NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, loader.LoadManifest(output.NewLogger(output.VerbosityDefault)))

	return server, loader
}

// --- splitModuleAndName tests ---

func TestSplitModuleAndName_Simple(t *testing.T) {
	mod, name := splitModuleAndName("requests.Response")
	assert.Equal(t, "requests", mod)
	assert.Equal(t, "Response", name)
}

func TestSplitModuleAndName_Nested(t *testing.T) {
	mod, name := splitModuleAndName("django.http.HttpRequest")
	assert.Equal(t, "django", mod)
	assert.Equal(t, "http.HttpRequest", name)
}

func TestSplitModuleAndName_NoDot(t *testing.T) {
	mod, name := splitModuleAndName("requests")
	assert.Equal(t, "requests", mod)
	assert.Equal(t, "", name)
}

// --- validateThirdPartyFQN tests ---

func TestValidateThirdPartyFQN_Function(t *testing.T) {
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	assert.True(t, validateThirdPartyFQN("requests.get", loader, logger))
}

func TestValidateThirdPartyFQN_Class(t *testing.T) {
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	assert.True(t, validateThirdPartyFQN("requests.Session", loader, logger))
}

func TestValidateThirdPartyFQN_NotFound(t *testing.T) {
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	assert.False(t, validateThirdPartyFQN("requests.nonexistent", loader, logger))
}

func TestValidateThirdPartyFQN_UnknownModule(t *testing.T) {
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	assert.False(t, validateThirdPartyFQN("unknown_lib.func", loader, logger))
}

func TestValidateThirdPartyFQN_NoDot(t *testing.T) {
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	assert.False(t, validateThirdPartyFQN("requests", loader, logger))
}

// --- resolveThirdPartyVariableBindings tests ---

func TestResolveThirdPartyVariableBindings_FunctionReturnType(t *testing.T) {
	// response = requests.get(url) → type requests.models.Response
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "response",
		Type: &core.TypeInfo{
			TypeFQN:    "call:requests.get",
			Confidence: 0.9,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	binding := scope.GetVariable("response")
	require.NotNil(t, binding)
	require.NotNil(t, binding.Type)
	assert.Equal(t, "requests.models.Response", binding.Type.TypeFQN)
	assert.Equal(t, "typeshed", binding.Type.Source)
	assert.Equal(t, "requests.get", binding.AssignedFrom)
}

func TestResolveThirdPartyVariableBindings_Constructor(t *testing.T) {
	// session = requests.Session() → type requests.Session
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "session",
		Type: &core.TypeInfo{
			TypeFQN:    "call:requests.Session",
			Confidence: 0.9,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	binding := scope.GetVariable("session")
	require.NotNil(t, binding)
	require.NotNil(t, binding.Type)
	assert.Equal(t, "requests.Session", binding.Type.TypeFQN)
	assert.Equal(t, "typeshed", binding.Type.Source)
}

func TestResolveThirdPartyVariableBindings_InstanceMethod(t *testing.T) {
	// session = requests.Session() (already resolved)
	// result = session.get(url) → type requests.models.Response
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	// session already resolved (Phase A would have done this)
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "session",
		Type: &core.TypeInfo{
			TypeFQN:    "requests.Session",
			Confidence: 0.85,
			Source:     "typeshed",
		},
	})
	// result still has call: placeholder
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:session.get",
			Confidence: 0.85,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	binding := scope.GetVariable("result")
	require.NotNil(t, binding)
	require.NotNil(t, binding.Type)
	assert.Equal(t, "requests.models.Response", binding.Type.TypeFQN)
	assert.Equal(t, "typeshed", binding.Type.Source)
}

func TestResolveThirdPartyVariableBindings_TwoPhaseResolution(t *testing.T) {
	// Phase A: session = requests.Session() → requests.Session
	// Phase B: result = session.get(url) → requests.models.Response
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "session",
		Type: &core.TypeInfo{
			TypeFQN:    "call:requests.Session",
			Confidence: 0.9,
		},
	})
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:session.get",
			Confidence: 0.85,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	// Phase A should resolve session
	sessionBinding := scope.GetVariable("session")
	require.NotNil(t, sessionBinding)
	assert.Equal(t, "requests.Session", sessionBinding.Type.TypeFQN)

	// Phase B should resolve result using session's resolved type
	resultBinding := scope.GetVariable("result")
	require.NotNil(t, resultBinding)
	assert.Equal(t, "requests.models.Response", resultBinding.Type.TypeFQN)
}

func TestResolveThirdPartyVariableBindings_NilRemote(t *testing.T) {
	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	// ThirdPartyRemote is nil — should return immediately without panic
	resolveThirdPartyVariableBindings(engine, output.NewLogger(output.VerbosityDefault))
}

func TestResolveThirdPartyVariableBindings_SkipsNonThirdParty(t *testing.T) {
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	// This is a userland call — should NOT be modified
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "x",
		Type: &core.TypeInfo{
			TypeFQN:    "call:myapp.create_user",
			Confidence: 0.9,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	binding := scope.GetVariable("x")
	require.NotNil(t, binding)
	assert.Equal(t, "call:myapp.create_user", binding.Type.TypeFQN, "non-third-party calls should be unchanged")
}

// --- resolveCallTarget third-party integration tests ---

func TestResolveCallTarget_ThirdPartyMethodOnTypedVariable(t *testing.T) {
	// response.json() where response has type requests.models.Response
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	modRegistry := core.NewModuleRegistry()
	engine := resolution.NewTypeInferenceEngine(modRegistry)
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main.process")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "response",
		Type: &core.TypeInfo{
			TypeFQN:    "requests.models.Response",
			Confidence: 0.9,
			Source:     "typeshed",
		},
	})
	engine.AddScope(scope)

	importMap := core.NewImportMap("/test/main.py")
	callGraph := core.NewCallGraph()

	fqn, resolved, typeInfo := resolveCallTarget(
		"response.json",
		importMap, modRegistry, "main", nil,
		engine, "main.process", callGraph, logger,
	)

	assert.True(t, resolved)
	assert.Equal(t, "requests.models.Response.json", fqn)
	require.NotNil(t, typeInfo)
	assert.Equal(t, "requests.models.Response", typeInfo.TypeFQN)
	assert.Equal(t, "typeshed", typeInfo.Source)
}

func TestResolveCallTarget_ThirdPartyImportedFunction(t *testing.T) {
	// requests.get(url) where requests is imported
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	modRegistry := core.NewModuleRegistry()
	engine := resolution.NewTypeInferenceEngine(modRegistry)
	engine.ThirdPartyRemote = loader

	importMap := core.NewImportMap("/test/main.py")
	importMap.AddImport("requests", "requests")
	callGraph := core.NewCallGraph()

	fqn, resolved, _ := resolveCallTarget(
		"requests.get",
		importMap, modRegistry, "main", nil,
		engine, "main.process", callGraph, logger,
	)

	assert.True(t, resolved)
	assert.Equal(t, "requests.get", fqn)
}

func TestResolveCallTarget_ThirdPartyFallback_NoRegression(t *testing.T) {
	// unknown_lib.func() with no third-party match → should remain unresolved
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	modRegistry := core.NewModuleRegistry()
	engine := resolution.NewTypeInferenceEngine(modRegistry)
	engine.ThirdPartyRemote = loader

	importMap := core.NewImportMap("/test/main.py")
	callGraph := core.NewCallGraph()

	_, resolved, _ := resolveCallTarget(
		"unknown_lib.unknown_func",
		importMap, modRegistry, "main", nil,
		engine, "main.process", callGraph, logger,
	)

	assert.False(t, resolved, "unknown third-party call should remain unresolved")
}

func TestResolveCallTarget_ThirdPartyMethodNotFound(t *testing.T) {
	// response.nonexistent() where response has type requests.models.Response
	// Method doesn't exist in registry → falls through to confidence heuristic
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	modRegistry := core.NewModuleRegistry()
	engine := resolution.NewTypeInferenceEngine(modRegistry)
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main.process")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "response",
		Type: &core.TypeInfo{
			TypeFQN:    "requests.models.Response",
			Confidence: 0.9,
			Source:     "typeshed",
		},
	})
	engine.AddScope(scope)

	importMap := core.NewImportMap("/test/main.py")
	callGraph := core.NewCallGraph()

	fqn, resolved, typeInfo := resolveCallTarget(
		"response.nonexistent",
		importMap, modRegistry, "main", nil,
		engine, "main.process", callGraph, logger,
	)

	// Should be resolved via confidence heuristic (>= 0.7)
	assert.True(t, resolved)
	assert.Equal(t, "requests.models.Response.nonexistent", fqn)
	require.NotNil(t, typeInfo)
	// Type info comes from confidence heuristic, not typeshed
	assert.Equal(t, "requests.models.Response", typeInfo.TypeFQN)
}

func TestResolveCallTarget_ThirdPartyLastResort(t *testing.T) {
	// requests.post where requests isn't in imports but exists in third-party
	server, loader := setupThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	modRegistry := core.NewModuleRegistry()
	engine := resolution.NewTypeInferenceEngine(modRegistry)
	engine.ThirdPartyRemote = loader

	importMap := core.NewImportMap("/test/main.py")
	callGraph := core.NewCallGraph()

	fqn, resolved, _ := resolveCallTarget(
		"requests.post",
		importMap, modRegistry, "main", nil,
		engine, "main.process", callGraph, logger,
	)

	assert.True(t, resolved)
	assert.Equal(t, "requests.post", fqn)
}

// --- Re-export / realistic CDN structure tests ---

func TestResolveThirdPartyBindings_ReExportedFunction(t *testing.T) {
	// CDN stores api.get, user writes requests.get → should still resolve
	server, loader := setupRealisticThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "response",
		Type: &core.TypeInfo{
			TypeFQN:    "call:requests.get",
			Confidence: 0.9,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	binding := scope.GetVariable("response")
	require.NotNil(t, binding)
	require.NotNil(t, binding.Type)
	assert.Equal(t, "requests.api.Response", binding.Type.TypeFQN)
	assert.Equal(t, "typeshed", binding.Type.Source)
}

func TestResolveCallTarget_ReExportedMethodOnSubModuleType(t *testing.T) {
	// response has type requests.api.Response, CDN stores class as Response
	// response.json() should resolve via stripped class name
	server, loader := setupRealisticThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	modRegistry := core.NewModuleRegistry()
	engine := resolution.NewTypeInferenceEngine(modRegistry)
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main.process")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "response",
		Type: &core.TypeInfo{
			TypeFQN:    "requests.api.Response",
			Confidence: 0.85,
			Source:     "typeshed",
		},
	})
	engine.AddScope(scope)

	importMap := core.NewImportMap("/test/main.py")
	callGraph := core.NewCallGraph()

	fqn, resolved, typeInfo := resolveCallTarget(
		"response.json",
		importMap, modRegistry, "main", nil,
		engine, "main.process", callGraph, logger,
	)

	assert.True(t, resolved)
	assert.Equal(t, "requests.api.Response.json", fqn)
	require.NotNil(t, typeInfo)
	assert.Equal(t, "typeshed", typeInfo.Source)
}

func TestResolveThirdPartyBindings_TwoPhaseWithReExports(t *testing.T) {
	// Full chain: session = requests.Session() → session.get(url)
	// Session stored directly, Session.get returns requests.sessions.Response
	// response.json() → should resolve via stripped class name
	server, loader := setupRealisticThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	scope := resolution.NewFunctionScope("main")
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "session",
		Type: &core.TypeInfo{
			TypeFQN:    "call:requests.Session",
			Confidence: 0.9,
		},
	})
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:session.get",
			Confidence: 0.85,
		},
	})
	engine.AddScope(scope)

	resolveThirdPartyVariableBindings(engine, logger)

	// Phase A: session resolved as constructor
	sessionBinding := scope.GetVariable("session")
	require.NotNil(t, sessionBinding)
	assert.Equal(t, "requests.Session", sessionBinding.Type.TypeFQN)

	// Phase B: result resolved via Session.get return type
	resultBinding := scope.GetVariable("result")
	require.NotNil(t, resultBinding)
	assert.Equal(t, "requests.sessions.Response", resultBinding.Type.TypeFQN)
}

func TestValidateThirdPartyFQN_ReExportedFunction(t *testing.T) {
	// requests.get should validate even though stored as api.get
	server, loader := setupRealisticThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	assert.True(t, validateThirdPartyFQN("requests.get", loader, logger))
	assert.True(t, validateThirdPartyFQN("requests.post", loader, logger))
	assert.True(t, validateThirdPartyFQN("requests.Session", loader, logger))
	assert.False(t, validateThirdPartyFQN("requests.nonexistent", loader, logger))
}

// setupDjangoThirdPartyServer creates a server with django-like class structure.
// Classes are stored with sub-module paths (e.g., db.models.fields.related.ForeignKey).
func setupDjangoThirdPartyServer(t *testing.T) (*httptest.Server, *registry.ThirdPartyRegistryRemote) {
	t.Helper()

	module := &core.StdlibModule{
		Module: "django",
		Functions: map[string]*core.StdlibFunction{
			"urls.path": {ReturnType: "django.urls.URLPattern", Confidence: 0.95, Source: "typeshed"},
			"urls.re_path": {ReturnType: "django.urls.URLPattern", Confidence: 0.95, Source: "typeshed"},
			"conf.settings": {ReturnType: "django.conf.LazySettings", Confidence: 0.95, Source: "typeshed"},
		},
		Classes: map[string]*core.StdlibClass{
			"db.models.fields.related.ForeignKey": {Type: "django.db.models.ForeignKey"},
			"db.models.fields.CharField":          {Type: "django.db.models.CharField"},
			"db.models.fields.IntegerField":       {Type: "django.db.models.IntegerField"},
			"db.models.fields.DateTimeField":      {Type: "django.db.models.DateTimeField"},
			"db.models.fields.TextField":          {Type: "django.db.models.TextField"},
			"db.models.fields.BooleanField":       {Type: "django.db.models.BooleanField"},
			"db.models.fields.json.JSONField":     {Type: "django.db.models.JSONField"},
			"db.models.query.QuerySet":            {Type: "django.db.models.QuerySet"},
			"db.models.Q":                         {Type: "django.db.models.Q"},
		},
		Constants:  map[string]*core.StdlibConstant{},
		Attributes: map[string]*core.StdlibAttribute{},
	}
	moduleJSON, err := json.Marshal(module)
	require.NoError(t, err)

	checksum := "sha256:" + func() string {
		h := sha256.Sum256(moduleJSON)
		return hex.EncodeToString(h[:])
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/thirdparty/v1/manifest.json":
			manifest := core.Manifest{
				Modules: []*core.ModuleEntry{
					{Name: "django", File: "django.json", Checksum: checksum},
				},
			}
			json.NewEncoder(w).Encode(manifest) //nolint:errcheck
		case "/thirdparty/v1/django.json":
			w.Write(moduleJSON) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))

	loader := registry.NewThirdPartyRegistryRemote(server.URL)
	require.NoError(t, loader.LoadManifest(output.NewLogger(output.VerbosityDefault)))

	return server, loader
}

func TestValidateThirdPartyFQN_LastComponentSuffixMatch(t *testing.T) {
	// django.db.models.ForeignKey → stored as db.models.fields.related.ForeignKey.
	// Last component "ForeignKey" suffix match should find it.
	server, loader := setupDjangoThirdPartyServer(t)
	defer server.Close()
	logger := output.NewLogger(output.VerbosityDefault)

	// These FQNs don't exist as exact keys but match via last-component suffix.
	assert.True(t, validateThirdPartyFQN("django.db.models.ForeignKey", loader, logger))
	assert.True(t, validateThirdPartyFQN("django.db.models.CharField", loader, logger))
	assert.True(t, validateThirdPartyFQN("django.db.models.IntegerField", loader, logger))
	assert.True(t, validateThirdPartyFQN("django.db.models.JSONField", loader, logger))

	// Direct key match still works.
	assert.True(t, validateThirdPartyFQN("django.db.models.Q", loader, logger))

	// Function suffix match.
	assert.True(t, validateThirdPartyFQN("django.path", loader, logger))

	// Non-existent class.
	assert.False(t, validateThirdPartyFQN("django.db.models.NonExistent", loader, logger))
}

func TestResolveWithPrefixStripping(t *testing.T) {
	registry := core.NewModuleRegistry()
	registry.Modules["core.utils.params"] = "/project/core/utils/params.py"

	cg := core.NewCallGraph()
	cg.Functions["core.views.index"] = &graph.Node{Name: "index"}

	tests := []struct {
		name       string
		fqn        string
		expectFQN  string
		expectOK   bool
		useCG      bool
	}{
		{"strips to registry match", "label_studio.core.utils.params.get_env", "core.utils.params.get_env", true, false},
		{"strips to callGraph match", "label_studio.core.views.index", "core.views.index", true, true},
		{"no match", "totally.unknown.module.func", "totally.unknown.module.func", false, false},
		{"single dot no strip", "module.func", "module.func", false, false},
		{"no dots", "func", "func", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var graph *core.CallGraph
			if tt.useCG {
				graph = cg
			}
			resultFQN, ok := resolveWithPrefixStripping(tt.fqn, registry, graph)
			assert.Equal(t, tt.expectOK, ok)
			assert.Equal(t, tt.expectFQN, resultFQN)
		})
	}
}

func TestCategorizeResolutionFailure(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		targetFQN string
		expected  string
	}{
		{"orm objects.filter", "Task.objects.filter", "Task.objects.filter", "orm_pattern"},
		{"orm .objects", "Task.objects", "Task.objects", "orm_pattern"},
		{"orm .all", "qs.all", "qs.all", "orm_pattern"},
		{"orm .count", "items.count", "items.count", "orm_pattern"},
		{"super call", "super().save", "super().save", "super_call"},
		{"super dot", "super.save", "super.save", "super_call"},
		{"variable self", "self.save", "self.save", "variable_method"},
		{"variable request", "request.GET", "request.GET", "variable_method"},
		{"variable cls", "cls.save", "cls.save", "variable_method"},
		{"variable data", "data.process", "data.process", "variable_method"},
		{"attribute chain", "foo.bar", "foo.bar", "attribute_chain"},
		{"not in imports", "some_func", "some_func", "not_in_imports"},
		{"unknown dotted upper", "Module.Something", "Module.Something", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass nil typeEngine to test non-registry paths.
			result := categorizeResolutionFailure(tt.target, tt.targetFQN, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategorizeResolutionFailure_WithStdlibRegistry(t *testing.T) {
	// Create a StdlibRegistryRemote with a manifest containing "os".
	stdlibLoader := registry.NewStdlibRegistryRemote("https://fake.cdn", "3.14")
	stdlibLoader.Manifest = &core.Manifest{
		Modules: []*core.ModuleEntry{
			{Name: "os", File: "os.json", Checksum: "sha256:abc"},
			{Name: "sys", File: "sys.json", Checksum: "sha256:def"},
		},
	}

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.StdlibRemote = stdlibLoader

	// os.getcwd is in stdlib → should return "stdlib_unresolved".
	result := categorizeResolutionFailure("os.getcwd", "os.getcwd", engine)
	assert.Equal(t, "stdlib_unresolved", result)

	// sys.argv is in stdlib → should return "stdlib_unresolved".
	result = categorizeResolutionFailure("sys.argv", "sys.argv", engine)
	assert.Equal(t, "stdlib_unresolved", result)

	// unknown.func is not in any registry → falls through to heuristics.
	result = categorizeResolutionFailure("unknown.func", "unknown.func", engine)
	assert.Equal(t, "attribute_chain", result)
}

func TestGetOptimalWorkerCount(t *testing.T) {
	// Basic test: should return a value between 2 and 16.
	count := getOptimalWorkerCount()
	assert.GreaterOrEqual(t, count, 2)
	assert.LessOrEqual(t, count, 16)
}

func TestGetOptimalWorkerCount_EnvOverride(t *testing.T) {
	t.Setenv("PATHFINDER_MAX_WORKERS", "8")
	count := getOptimalWorkerCount()
	assert.Equal(t, 8, count)
}

func TestGetOptimalWorkerCount_EnvOverrideCapped(t *testing.T) {
	t.Setenv("PATHFINDER_MAX_WORKERS", "100")
	count := getOptimalWorkerCount()
	assert.Equal(t, 32, count)
}

func TestGetOptimalWorkerCount_EnvInvalid(t *testing.T) {
	t.Setenv("PATHFINDER_MAX_WORKERS", "notanumber")
	count := getOptimalWorkerCount()
	// Should fall through to CPU-based calculation.
	assert.GreaterOrEqual(t, count, 2)
	assert.LessOrEqual(t, count, 16)
}

func TestCategorizeResolutionFailure_WithRegistry(t *testing.T) {
	server, loader := setupDjangoThirdPartyServer(t)
	defer server.Close()

	engine := resolution.NewTypeInferenceEngine(core.NewModuleRegistry())
	engine.ThirdPartyRemote = loader

	// Django FQN should be detected as external_framework via HasModule.
	result := categorizeResolutionFailure("models.ForeignKey", "django.db.models.ForeignKey", engine)
	assert.Equal(t, "external_framework", result)

	// Unknown module should fall through to other heuristics.
	result = categorizeResolutionFailure("foo.bar", "unknown_pkg.foo.bar", engine)
	assert.Equal(t, "attribute_chain", result)
}
