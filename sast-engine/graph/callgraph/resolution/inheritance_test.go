package resolution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	cgregistry "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// --- helpers ---

func setupInheritanceServer() *httptest.Server {
	// Simulate CDN with django module containing View class with get method
	djangoModule := &core.StdlibModule{
		Functions: map[string]*core.StdlibFunction{},
		Classes: map[string]*core.StdlibClass{
			"views.View": {
				Type: "class",
				Methods: map[string]*core.StdlibFunction{
					"get": {
						ReturnType: "django.http.HttpResponse",
						Confidence: 0.95,
						Params: []*core.FunctionParam{
							{Name: "self", Type: "django.views.View", Required: true},
							{Name: "request", Type: "django.http.HttpRequest", Required: true},
							{Name: "*args", Type: "", Required: false},
							{Name: "**kwargs", Type: "", Required: false},
						},
					},
					"post": {
						ReturnType: "django.http.HttpResponse",
						Confidence: 0.95,
						Params: []*core.FunctionParam{
							{Name: "self", Type: "django.views.View", Required: true},
							{Name: "request", Type: "django.http.HttpRequest", Required: true},
						},
					},
					"dispatch": {
						ReturnType: "django.http.HttpResponse",
						Confidence: 0.95,
						Params: []*core.FunctionParam{
							{Name: "self", Type: "django.views.View", Required: true},
							{Name: "request", Type: "django.http.HttpRequest", Required: true},
						},
					},
				},
				Attributes: map[string]*core.StdlibAttribute{
					"request": {
						Type:       "django.http.HttpRequest",
						Confidence: 0.95,
						Kind:       "attribute",
					},
					"kwargs": {
						Type:       "builtins.dict",
						Confidence: 0.90,
						Kind:       "attribute",
					},
				},
			},
			"http.HttpRequest": {
				Type: "class",
				Methods: map[string]*core.StdlibFunction{
					"get_host": {
						ReturnType: "builtins.str",
						Confidence: 0.95,
					},
				},
				Attributes: map[string]*core.StdlibAttribute{
					"GET": {
						Type:       "django.http.QueryDict",
						Confidence: 0.95,
						Kind:       "attribute",
					},
					"POST": {
						Type:       "django.http.QueryDict",
						Confidence: 0.95,
						Kind:       "attribute",
					},
					"method": {
						Type:       "builtins.str",
						Confidence: 0.95,
						Kind:       "attribute",
					},
				},
			},
			"http.QueryDict": {
				Type: "class",
				Methods: map[string]*core.StdlibFunction{
					"get": {
						ReturnType: "builtins.str",
						Confidence: 0.95,
						Params: []*core.FunctionParam{
							{Name: "self", Type: "", Required: true},
							{Name: "key", Type: "builtins.str", Required: true},
						},
					},
				},
			},
		},
		Constants:  map[string]*core.StdlibConstant{},
		Attributes: map[string]*core.StdlibAttribute{},
	}

	requestsModule := &core.StdlibModule{
		Functions: map[string]*core.StdlibFunction{
			"api.get": {ReturnType: "requests.api.Response", Confidence: 0.95},
		},
		Classes: map[string]*core.StdlibClass{
			"sessions.Session": {
				Type: "class",
				Methods: map[string]*core.StdlibFunction{
					"get": {ReturnType: "requests.models.Response", Confidence: 0.95},
				},
			},
		},
		Constants:  map[string]*core.StdlibConstant{},
		Attributes: map[string]*core.StdlibAttribute{},
	}

	// Pre-serialize modules and compute checksums
	djangoJSON, _ := json.Marshal(djangoModule)
	requestsJSON, _ := json.Marshal(requestsModule)

	djangoChecksum := "sha256:" + func() string {
		h := sha256.Sum256(djangoJSON)
		return hex.EncodeToString(h[:])
	}()
	requestsChecksum := "sha256:" + func() string {
		h := sha256.Sum256(requestsJSON)
		return hex.EncodeToString(h[:])
	}()

	manifest := core.Manifest{
		Modules: []*core.ModuleEntry{
			{Name: "django", File: "django.json", Checksum: djangoChecksum},
			{Name: "requests", File: "requests.json", Checksum: requestsChecksum},
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/thirdparty/v1/manifest.json":
			json.NewEncoder(w).Encode(manifest) //nolint:errcheck
		case "/thirdparty/v1/django.json":
			w.Write(djangoJSON) //nolint:errcheck
		case "/thirdparty/v1/requests.json":
			w.Write(requestsJSON) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
}

func newInheritanceTestLogger() *output.Logger {
	return output.NewLogger(output.VerbosityDefault)
}

func newThirdPartyLoader(t *testing.T, serverURL string) *cgregistry.ThirdPartyRegistryRemote {
	t.Helper()
	logger := newInheritanceTestLogger()
	loader := cgregistry.NewThirdPartyRegistryRemote(serverURL)
	if err := loader.LoadManifest(logger); err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}
	return loader
}

// --- splitModuleAndClass tests ---

func TestSplitModuleAndClass(t *testing.T) {
	tests := []struct {
		fqn        string
		wantModule string
		wantClass  string
	}{
		{"django.views.View", "django", "views.View"},
		{"requests.Session", "requests", "Session"},
		{"builtins.str", "builtins", "str"},
		{"django", "django", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fqn, func(t *testing.T) {
			mod, cls := splitModuleAndClass(tt.fqn)
			if mod != tt.wantModule || cls != tt.wantClass {
				t.Errorf("splitModuleAndClass(%q) = (%q, %q), want (%q, %q)",
					tt.fqn, mod, cls, tt.wantModule, tt.wantClass)
			}
		})
	}
}

// --- ResolveParentClassFQN tests ---

func TestResolveParentClassFQN_ViaImport(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	te.AddImportMap("/app/views.py", &core.ImportMap{
		Imports: map[string]string{
			"View": "django.views.View",
		},
	})

	result := ResolveParentClassFQN(
		"myapp.views.TestView",
		"View",
		"/app/views.py",
		te,
		nil,
	)

	if result != "django.views.View" {
		t.Errorf("Expected 'django.views.View', got %q", result)
	}
}

func TestResolveParentClassFQN_DottedSuperclass(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	te.AddImportMap("/app/views.py", &core.ImportMap{
		Imports: map[string]string{
			"views": "django.views",
		},
	})

	result := ResolveParentClassFQN(
		"myapp.views.TestView",
		"views.View",
		"/app/views.py",
		te,
		nil,
	)

	if result != "django.views.View" {
		t.Errorf("Expected 'django.views.View', got %q", result)
	}
}

func TestResolveParentClassFQN_EmptySuperclass(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	result := ResolveParentClassFQN("myapp.TestClass", "", "/app/test.py", te, nil)
	if result != "" {
		t.Errorf("Expected empty string for empty superclass, got %q", result)
	}
}

func TestResolveParentClassFQN_SameModule(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	te.AddImportMap("/app/views.py", &core.ImportMap{
		Imports: map[string]string{},
	})

	result := ResolveParentClassFQN(
		"myapp.views.ChildView",
		"BaseView",
		"/app/views.py",
		te,
		nil,
	)

	if result != "myapp.views.BaseView" {
		t.Errorf("Expected 'myapp.views.BaseView', got %q", result)
	}
}

func TestResolveParentClassFQN_NoImportMap(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	// No import map for this file

	result := ResolveParentClassFQN(
		"myapp.views.TestView",
		"View",
		"/app/views.py",
		te,
		nil,
	)

	// Should fall through to same-module resolution
	if result != "myapp.views.View" {
		t.Errorf("Expected 'myapp.views.View', got %q", result)
	}
}

// --- PropagateParentParamTypes tests ---

func TestPropagateParentParamTypes_DjangoView(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	// Create type engine with child method scope
	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.TestView.get")
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.get",
		"django.views.View",
		"get",
		te,
		loader,
		logger,
	)

	// Check that "request" parameter was propagated
	requestBinding := childScope.GetVariable("request")
	if requestBinding == nil {
		t.Fatal("Expected 'request' variable binding to be propagated")
	}
	if requestBinding.Type == nil {
		t.Fatal("Expected 'request' to have type info")
	}
	if requestBinding.Type.TypeFQN != "django.http.HttpRequest" {
		t.Errorf("Expected type 'django.http.HttpRequest', got %q", requestBinding.Type.TypeFQN)
	}
	if requestBinding.Type.Source != "parent_method_signature" {
		t.Errorf("Expected source 'parent_method_signature', got %q", requestBinding.Type.Source)
	}
	if requestBinding.Type.Confidence != 0.90 {
		t.Errorf("Expected confidence 0.90, got %f", requestBinding.Type.Confidence)
	}
}

func TestPropagateParentParamTypes_SkipsSelf(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.TestView.get")
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.get",
		"django.views.View",
		"get",
		te,
		loader,
		logger,
	)

	// "self" should NOT be propagated
	selfBinding := childScope.GetVariable("self")
	if selfBinding != nil {
		t.Errorf("'self' should not be propagated, got %v", selfBinding)
	}
}

func TestPropagateParentParamTypes_SkipsExistingType(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.TestView.get")
	// Pre-add a typed request binding (e.g., from annotation)
	childScope.AddVariable(&VariableBinding{
		VarName: "request",
		Type: &core.TypeInfo{
			TypeFQN:    "custom.Request",
			Confidence: 1.0,
			Source:     "annotation",
		},
	})
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.get",
		"django.views.View",
		"get",
		te,
		loader,
		logger,
	)

	// Existing type should NOT be overwritten
	requestBinding := childScope.GetVariable("request")
	if requestBinding.Type.TypeFQN != "custom.Request" {
		t.Errorf("Expected existing type 'custom.Request' to be preserved, got %q", requestBinding.Type.TypeFQN)
	}
}

func TestPropagateParentParamTypes_NilRemote(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	logger := newInheritanceTestLogger()

	// Should not panic
	PropagateParentParamTypes(
		"myapp.TestView.get",
		"django.views.View",
		"get",
		te,
		nil,
		logger,
	)
}

func TestPropagateParentParamTypes_MethodNotFound(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.TestView.custom_method")
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.custom_method",
		"django.views.View",
		"custom_method", // doesn't exist in parent
		te,
		loader,
		logger,
	)

	// No variables should be added
	if len(childScope.Variables) != 0 {
		t.Errorf("Expected no variables, got %d", len(childScope.Variables))
	}
}

func TestPropagateParentParamTypes_PostMethod(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.TestView.post")
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.post",
		"django.views.View",
		"post",
		te,
		loader,
		logger,
	)

	requestBinding := childScope.GetVariable("request")
	if requestBinding == nil || requestBinding.Type == nil {
		t.Fatal("Expected 'request' to be propagated from post method")
	}
	if requestBinding.Type.TypeFQN != "django.http.HttpRequest" {
		t.Errorf("Expected 'django.http.HttpRequest', got %q", requestBinding.Type.TypeFQN)
	}
}

// --- ResolveInheritedSelfAttribute tests ---

func TestResolveInheritedSelfAttribute_Request(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	typeInfo := ResolveInheritedSelfAttribute(
		"django.views.View",
		"request",
		loader,
		logger,
	)

	if typeInfo == nil {
		t.Fatal("Expected type info for 'request' attribute")
	}
	if typeInfo.TypeFQN != "django.http.HttpRequest" {
		t.Errorf("Expected 'django.http.HttpRequest', got %q", typeInfo.TypeFQN)
	}
	if typeInfo.Source != "parent_class_attribute" {
		t.Errorf("Expected source 'parent_class_attribute', got %q", typeInfo.Source)
	}
}

func TestResolveInheritedSelfAttribute_Kwargs(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	typeInfo := ResolveInheritedSelfAttribute(
		"django.views.View",
		"kwargs",
		loader,
		logger,
	)

	if typeInfo == nil {
		t.Fatal("Expected type info for 'kwargs'")
	}
	if typeInfo.TypeFQN != "builtins.dict" {
		t.Errorf("Expected 'builtins.dict', got %q", typeInfo.TypeFQN)
	}
}

func TestResolveInheritedSelfAttribute_NotFound(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	typeInfo := ResolveInheritedSelfAttribute(
		"django.views.View",
		"nonexistent_attr",
		loader,
		logger,
	)

	if typeInfo != nil {
		t.Errorf("Expected nil for nonexistent attribute, got %v", typeInfo)
	}
}

func TestResolveInheritedSelfAttribute_NilRemote(t *testing.T) {
	logger := newInheritanceTestLogger()
	typeInfo := ResolveInheritedSelfAttribute("django.views.View", "request", nil, logger)
	if typeInfo != nil {
		t.Errorf("Expected nil with nil remote, got %v", typeInfo)
	}
}

func TestResolveInheritedSelfAttribute_UnknownModule(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	typeInfo := ResolveInheritedSelfAttribute(
		"unknown_module.SomeClass",
		"attr",
		loader,
		logger,
	)

	if typeInfo != nil {
		t.Errorf("Expected nil for unknown module, got %v", typeInfo)
	}
}

// --- Integration: end-to-end Django View resolution chain ---

func TestInheritance_DjangoViewChain(t *testing.T) {
	// Simulates: class TestView(View): def get(self, request): ...
	// Verifies that request gets type django.http.HttpRequest from parent View.get
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	te.ThirdPartyRemote = loader

	// Add import map for the file
	te.AddImportMap("/app/views.py", &core.ImportMap{
		Imports: map[string]string{
			"View": "django.views.View",
		},
	})

	// Step 1: Resolve parent class FQN
	parentFQN := ResolveParentClassFQN(
		"myapp.views.TestView",
		"View",
		"/app/views.py",
		te,
		nil,
	)
	if parentFQN != "django.views.View" {
		t.Fatalf("Step 1 failed: expected 'django.views.View', got %q", parentFQN)
	}

	// Step 2: Create child method scope and propagate params
	childScope := NewFunctionScope("myapp.views.TestView.get")
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.get",
		parentFQN,
		"get",
		te,
		loader,
		logger,
	)

	// Step 3: Verify request has type
	requestBinding := childScope.GetVariable("request")
	if requestBinding == nil || requestBinding.Type == nil {
		t.Fatal("Step 3 failed: request not typed")
	}
	if requestBinding.Type.TypeFQN != "django.http.HttpRequest" {
		t.Errorf("Step 3: expected 'django.http.HttpRequest', got %q", requestBinding.Type.TypeFQN)
	}

	// Step 4: Resolve self.request attribute via parent
	selfReqType := ResolveInheritedSelfAttribute(
		parentFQN,
		"request",
		loader,
		logger,
	)
	if selfReqType == nil || selfReqType.TypeFQN != "django.http.HttpRequest" {
		t.Fatalf("Step 4 failed: self.request not resolved to HttpRequest")
	}

	// Step 5: Verify request.GET attribute is available
	getAttrType := ResolveInheritedSelfAttribute(
		"django.http.HttpRequest",
		"GET",
		loader,
		logger,
	)
	if getAttrType == nil || getAttrType.TypeFQN != "django.http.QueryDict" {
		t.Fatalf("Step 5 failed: request.GET not resolved to QueryDict")
	}

	t.Logf("Full chain resolved: View → request: HttpRequest → GET: QueryDict")
}

// --- Multiple inheritance ---

func TestResolveParentClassFQN_MultipleParents(t *testing.T) {
	te := NewTypeInferenceEngine(nil)
	te.AddImportMap("/app/views.py", &core.ImportMap{
		Imports: map[string]string{
			"LoginRequiredMixin": "django.contrib.auth.mixins.LoginRequiredMixin",
			"View":              "django.views.View",
		},
	})

	// First parent: LoginRequiredMixin
	result1 := ResolveParentClassFQN(
		"myapp.views.ProtectedView",
		"LoginRequiredMixin",
		"/app/views.py",
		te,
		nil,
	)
	if result1 != "django.contrib.auth.mixins.LoginRequiredMixin" {
		t.Errorf("Expected mixin FQN, got %q", result1)
	}

	// Second parent: View
	result2 := ResolveParentClassFQN(
		"myapp.views.ProtectedView",
		"View",
		"/app/views.py",
		te,
		nil,
	)
	if result2 != "django.views.View" {
		t.Errorf("Expected View FQN, got %q", result2)
	}
}

func TestPropagateParentParamTypes_SkipsArgsKwargs(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.TestView.get")
	te.AddScope(childScope)

	PropagateParentParamTypes(
		"myapp.views.TestView.get",
		"django.views.View",
		"get",
		te,
		loader,
		logger,
	)

	// *args and **kwargs should be skipped
	argsBinding := childScope.GetVariable("*args")
	if argsBinding != nil {
		t.Errorf("*args should not be propagated")
	}
	kwargsBinding := childScope.GetVariable("**kwargs")
	if kwargsBinding != nil {
		t.Errorf("**kwargs should not be propagated")
	}
}

// Verify graceful fallback when parent class isn't in typeshed.
func TestPropagateParentParamTypes_UserLandParent(t *testing.T) {
	server := setupInheritanceServer()
	defer server.Close()

	logger := newInheritanceTestLogger()
	loader := newThirdPartyLoader(t, server.URL)

	te := NewTypeInferenceEngine(nil)
	childScope := NewFunctionScope("myapp.views.ChildView.process")
	te.AddScope(childScope)

	// Parent is userland (not in CDN) — should gracefully do nothing
	PropagateParentParamTypes(
		"myapp.views.ChildView.process",
		"myapp.views.BaseView", // not in third-party registry
		"process",
		te,
		loader,
		logger,
	)

	if len(childScope.Variables) != 0 {
		t.Errorf("Expected no propagation for userland parent, got %d vars", len(childScope.Variables))
	}
}

// Suppress unused import warning.
var _ = fmt.Sprintf
