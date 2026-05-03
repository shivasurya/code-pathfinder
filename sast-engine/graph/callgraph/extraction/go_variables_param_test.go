package extraction

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
)

// mockStdlibLoaderWithTypes extends mockStdlibLoader to support GetType.
type mockStdlibLoaderWithTypes struct {
	stdlibPkgs map[string]bool
	functions  map[string]*core.GoStdlibFunction // key: "importPath.funcName"
	types      map[string]*core.GoStdlibType     // key: "importPath.TypeName"
}

func (m *mockStdlibLoaderWithTypes) ValidateStdlibImport(importPath string) bool {
	return m.stdlibPkgs[importPath]
}

func (m *mockStdlibLoaderWithTypes) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	key := importPath + "." + funcName
	fn, ok := m.functions[key]
	if !ok {
		return nil, errMockNotImplemented
	}
	return fn, nil
}

func (m *mockStdlibLoaderWithTypes) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	key := importPath + "." + typeName
	t, ok := m.types[key]
	if !ok {
		return nil, errMockNotImplemented
	}
	return t, nil
}

func (m *mockStdlibLoaderWithTypes) GetPackage(_ string) (*core.GoStdlibPackage, error) {
	return nil, errMockNotImplemented
}

func (m *mockStdlibLoaderWithTypes) PackageCount() int {
	return len(m.stdlibPkgs)
}

// mockThirdPartyLoaderWithTypes is a core.GoThirdPartyLoader that serves type data.
type mockThirdPartyLoaderWithTypes struct {
	packages map[string]bool
	types    map[string]*core.GoStdlibType // key: "importPath.TypeName"
}

func (m *mockThirdPartyLoaderWithTypes) ValidateImport(importPath string) bool {
	return m.packages[importPath]
}

func (m *mockThirdPartyLoaderWithTypes) GetFunction(_, _ string) (*core.GoStdlibFunction, error) {
	return nil, errMockNotImplemented
}

func (m *mockThirdPartyLoaderWithTypes) GetType(importPath, typeName string) (*core.GoStdlibType, error) {
	key := importPath + "." + typeName
	t, ok := m.types[key]
	if !ok {
		return nil, errMockNotImplemented
	}
	return t, nil
}

func (m *mockThirdPartyLoaderWithTypes) PackageCount() int {
	return len(m.packages)
}

// ---------------------------------------------------------------------------
// extractionSplitGoTypeFQN tests
// ---------------------------------------------------------------------------

func TestExtractionSplitGoTypeFQN_Valid(t *testing.T) {
	imp, name, ok := extractionSplitGoTypeFQN("net/http.Request")
	assert.True(t, ok)
	assert.Equal(t, "net/http", imp)
	assert.Equal(t, "Request", name)
}

func TestExtractionSplitGoTypeFQN_NoPackage(t *testing.T) {
	_, _, ok := extractionSplitGoTypeFQN("string")
	assert.False(t, ok)
}

func TestExtractionSplitGoTypeFQN_Empty(t *testing.T) {
	_, _, ok := extractionSplitGoTypeFQN("")
	assert.False(t, ok)
}

func TestExtractionSplitGoTypeFQN_TrailingDot(t *testing.T) {
	_, _, ok := extractionSplitGoTypeFQN("net/http.")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// extractionResolveGoTypeFQN tests
// ---------------------------------------------------------------------------

func TestExtractionResolveGoTypeFQN_KnownAlias(t *testing.T) {
	importMap := &core.GoImportMap{
		Imports: map[string]string{"http": "net/http"},
	}
	result := extractionResolveGoTypeFQN("http.Request", importMap)
	assert.Equal(t, "net/http.Request", result)
}

func TestExtractionResolveGoTypeFQN_UnknownAlias(t *testing.T) {
	importMap := &core.GoImportMap{
		Imports: map[string]string{},
	}
	result := extractionResolveGoTypeFQN("gin.Context", importMap)
	assert.Equal(t, "gin.Context", result)
}

func TestExtractionResolveGoTypeFQN_NilImportMap(t *testing.T) {
	result := extractionResolveGoTypeFQN("http.Request", nil)
	assert.Equal(t, "http.Request", result)
}

func TestExtractionResolveGoTypeFQN_Unqualified(t *testing.T) {
	importMap := &core.GoImportMap{Imports: map[string]string{}}
	result := extractionResolveGoTypeFQN("MyStruct", importMap)
	assert.Equal(t, "MyStruct", result)
}

// ---------------------------------------------------------------------------
// inferTypeFromParamMethodCall — unit tests via ExtractGoVariableAssignments
// ---------------------------------------------------------------------------

// TestParamAwareRHSInference_StdlibParam tests that `input := r.FormValue("id")`
// resolves to builtin.string when r is a *http.Request parameter.
func TestParamAwareRHSInference_StdlibParam(t *testing.T) {
	code := `package main

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	input := r.FormValue("id")
	_ = input
}
`

	// Build a minimal registry pointing /test → "main".
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}

	// Attach a stdlib loader that knows about net/http.Request.FormValue.
	loader := &mockStdlibLoaderWithTypes{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions:  map[string]*core.GoStdlibFunction{},
		types: map[string]*core.GoStdlibType{
			"net/http.Request": {
				Name: "Request",
				Methods: map[string]*core.GoStdlibFunction{
					"FormValue": {
						Name:    "FormValue",
						Returns: []*core.GoReturnValue{{Type: "string"}},
					},
				},
			},
		},
	}
	reg.StdlibLoader = loader

	importMap := &core.GoImportMap{
		Imports: map[string]string{"http": "net/http"},
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	// Build a minimal call graph with the handler function so param lookup works.
	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.handler": {
				ID:                  "handler",
				Name:                "handler",
				Type:                "function_declaration",
				File:                "/test/main.go",
				MethodArgumentsValue: []string{"w", "r"},
				MethodArgumentsType:  []string{"w: http.ResponseWriter", "r: *http.Request"},
			},
		},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	assert.NoError(t, err)

	scope := typeEngine.GetScope("main.handler")
	assert.NotNil(t, scope, "scope should exist for handler")

	bindings, ok := scope.Variables["input"]
	assert.True(t, ok, "input binding should be created")
	if assert.Len(t, bindings, 1) {
		assert.Equal(t, "builtin.string", bindings[0].Type.TypeFQN)
		assert.Equal(t, "method_return_type", bindings[0].Type.Source)
		assert.InDelta(t, 0.85, bindings[0].Type.Confidence, 0.001)
	}
}

// TestParamAwareRHSInference_NilCallGraph ensures no panic and graceful nil
// return when callGraph is nil.
func TestParamAwareRHSInference_NilCallGraph(t *testing.T) {
	code := `package main

import "net/http"

func handler(r *http.Request) {
	input := r.FormValue("id")
	_ = input
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}
	importMap := &core.GoImportMap{Imports: map[string]string{"http": "net/http"}}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	// nil callGraph — must not panic; input binding should simply be absent.
	assert.NotPanics(t, func() {
		_ = ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, nil)
	})
}

// TestParamAwareRHSInference_ThirdPartyParam tests that `q := c.Query("search")`
// resolves via ThirdPartyLoader when c is a third-party type parameter.
func TestParamAwareRHSInference_ThirdPartyParam(t *testing.T) {
	code := `package main

import "github.com/gin-gonic/gin"

func handle(c *gin.Context) {
	q := c.Query("search")
	_ = q
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}

	tpLoader := &mockThirdPartyLoaderWithTypes{
		packages: map[string]bool{"github.com/gin-gonic/gin": true},
		types: map[string]*core.GoStdlibType{
			"github.com/gin-gonic/gin.Context": {
				Name: "Context",
				Methods: map[string]*core.GoStdlibFunction{
					"Query": {
						Name:    "Query",
						Returns: []*core.GoReturnValue{{Type: "string"}},
					},
				},
			},
		},
	}
	reg.ThirdPartyLoader = tpLoader

	importMap := &core.GoImportMap{
		Imports: map[string]string{"gin": "github.com/gin-gonic/gin"},
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.handle": {
				ID:                  "handle",
				Name:                "handle",
				Type:                "function_declaration",
				File:                "/test/main.go",
				MethodArgumentsValue: []string{"c"},
				MethodArgumentsType:  []string{"c: *gin.Context"},
			},
		},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	assert.NoError(t, err)

	scope := typeEngine.GetScope("main.handle")
	assert.NotNil(t, scope)

	bindings, ok := scope.Variables["q"]
	assert.True(t, ok, "q binding should be created via ThirdPartyLoader")
	if assert.Len(t, bindings, 1) {
		assert.Equal(t, "builtin.string", bindings[0].Type.TypeFQN)
		assert.Equal(t, "method_return_type", bindings[0].Type.Source)
	}
}

// TestParamAwareRHSInference_UnknownParam tests that no binding is created when
// the parameter name is not in the function's MethodArgumentsValue list.
func TestParamAwareRHSInference_UnknownParam(t *testing.T) {
	code := `package main

import "net/http"

func handler(r *http.Request) {
	input := x.FormValue("id")
	_ = input
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}
	importMap := &core.GoImportMap{Imports: map[string]string{"http": "net/http"}}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.handler": {
				ID:                  "handler",
				Name:                "handler",
				Type:                "function_declaration",
				File:                "/test/main.go",
				MethodArgumentsValue: []string{"r"},
				MethodArgumentsType:  []string{"r: *http.Request"},
			},
		},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	assert.NoError(t, err)

	scope := typeEngine.GetScope("main.handler")
	if scope != nil {
		_, hasBinding := scope.Variables["input"]
		assert.False(t, hasBinding, "no binding for x (not a parameter named r)")
	}
}

// TestParamAwareRHSInference_PackageAlias ensures package-qualified calls like
// http.NewRequest() are NOT intercepted by param-aware inference (they're already
// handled by inferTypeFromFunctionCall).
func TestParamAwareRHSInference_PackageAlias(t *testing.T) {
	code := `package main

import "net/http"

func makeReq() {
	req, _ := http.NewRequest("GET", "/", nil)
	_ = req
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}

	loader := &mockStdlibLoaderWithTypes{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions: map[string]*core.GoStdlibFunction{
			"net/http.NewRequest": {
				Name:    "NewRequest",
				Returns: []*core.GoReturnValue{{Type: "*Request"}, {Type: "error"}},
			},
		},
		types: map[string]*core.GoStdlibType{},
	}
	reg.StdlibLoader = loader

	importMap := &core.GoImportMap{Imports: map[string]string{"http": "net/http"}}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.makeReq": {
				ID:   "makeReq",
				Name: "makeReq",
				Type: "function_declaration",
				File: "/test/main.go",
			},
		},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	assert.NoError(t, err)

	// req should be resolved via inferTypeFromFunctionCall (stdlib path), not param-aware
	scope := typeEngine.GetScope("main.makeReq")
	assert.NotNil(t, scope)
	bindings, ok := scope.Variables["req"]
	assert.True(t, ok, "req should have a binding from stdlib lookup")
	if assert.Len(t, bindings, 1) {
		assert.Equal(t, "net/http.Request", bindings[0].Type.TypeFQN)
	}
}

// TestParamAwareRHSInference_MethodNotFound ensures no binding is created when the
// type is known but does not have the called method.
func TestParamAwareRHSInference_MethodNotFound(t *testing.T) {
	code := `package main

import "net/http"

func handler(r *http.Request) {
	v := r.UnknownMethod()
	_ = v
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}

	loader := &mockStdlibLoaderWithTypes{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions:  map[string]*core.GoStdlibFunction{},
		types: map[string]*core.GoStdlibType{
			"net/http.Request": {
				Name:    "Request",
				Methods: map[string]*core.GoStdlibFunction{},
			},
		},
	}
	reg.StdlibLoader = loader

	importMap := &core.GoImportMap{Imports: map[string]string{"http": "net/http"}}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.handler": {
				ID:                   "handler",
				Name:                 "handler",
				Type:                 "function_declaration",
				File:                 "/test/main.go",
				MethodArgumentsValue: []string{"r"},
				MethodArgumentsType:  []string{"r: *http.Request"},
			},
		},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	assert.NoError(t, err)

	scope := typeEngine.GetScope("main.handler")
	if scope != nil {
		_, hasBinding := scope.Variables["v"]
		assert.False(t, hasBinding, "no binding when method is not in type's method set")
	}
}

// TestParamAwareRHSInference_ErrorOnlyReturns ensures no binding is created when
// the method only returns error (no non-error return value to infer from).
func TestParamAwareRHSInference_ErrorOnlyReturns(t *testing.T) {
	code := `package main

import "net/http"

func handler(r *http.Request) {
	err := r.ParseForm()
	_ = err
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}

	loader := &mockStdlibLoaderWithTypes{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions:  map[string]*core.GoStdlibFunction{},
		types: map[string]*core.GoStdlibType{
			"net/http.Request": {
				Name: "Request",
				Methods: map[string]*core.GoStdlibFunction{
					"ParseForm": {
						Name:    "ParseForm",
						Returns: []*core.GoReturnValue{{Type: "error"}},
					},
				},
			},
		},
	}
	reg.StdlibLoader = loader

	importMap := &core.GoImportMap{Imports: map[string]string{"http": "net/http"}}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.handler": {
				ID:                   "handler",
				Name:                 "handler",
				Type:                 "function_declaration",
				File:                 "/test/main.go",
				MethodArgumentsValue: []string{"r"},
				MethodArgumentsType:  []string{"r: *http.Request"},
			},
		},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	assert.NoError(t, err)

	scope := typeEngine.GetScope("main.handler")
	if scope != nil {
		_, hasBinding := scope.Variables["err"]
		assert.False(t, hasBinding, "no binding when method only returns error")
	}
}

// TestParamAwareRHSInference_UnqualifiedParamType verifies graceful handling when
// the parameter type is unqualified (no package prefix, e.g. "MyStruct").
func TestParamAwareRHSInference_UnqualifiedParamType(t *testing.T) {
	code := `package main

func process(s MyStruct) {
	v := s.Compute()
	_ = v
}
`
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/app"
	reg.DirToImport = map[string]string{"/test": "main"}

	importMap := &core.GoImportMap{Imports: map[string]string{}}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"main.process": {
				ID:                   "process",
				Name:                 "process",
				Type:                 "function_declaration",
				File:                 "/test/main.go",
				MethodArgumentsValue: []string{"s"},
				MethodArgumentsType:  []string{"s: MyStruct"},
			},
		},
	}

	// Must not panic; type has no dot, so extractionSplitGoTypeFQN returns false.
	assert.NotPanics(t, func() {
		_ = ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap, callGraph)
	})
}
