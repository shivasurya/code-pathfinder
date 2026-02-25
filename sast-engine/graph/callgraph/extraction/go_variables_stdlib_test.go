package extraction

import (
	"errors"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMockNotImplemented is returned by mock methods that are not exercised.
var errMockNotImplemented = errors.New("not implemented by mock")

// mockStdlibLoader implements core.GoStdlibLoader for testing without network access.
type mockStdlibLoader struct {
	stdlibPkgs map[string]bool
	functions  map[string]*core.GoStdlibFunction // key: "importPath.funcName"
}

func (m *mockStdlibLoader) ValidateStdlibImport(importPath string) bool {
	return m.stdlibPkgs[importPath]
}

func (m *mockStdlibLoader) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	key := importPath + "." + funcName
	fn, ok := m.functions[key]
	if !ok {
		return nil, errMockNotImplemented
	}
	return fn, nil
}

func (m *mockStdlibLoader) GetType(_, _ string) (*core.GoStdlibType, error) {
	return nil, errMockNotImplemented
}

func (m *mockStdlibLoader) PackageCount() int {
	return len(m.stdlibPkgs)
}

// -----------------------------------------------------------------------------
// normalizeStdlibReturnType
// -----------------------------------------------------------------------------

func TestNormalizeStdlibReturnType_BuiltinString(t *testing.T) {
	assert.Equal(t, "builtin.string", normalizeStdlibReturnType("string", "fmt"))
}

func TestNormalizeStdlibReturnType_BuiltinError(t *testing.T) {
	assert.Equal(t, "builtin.error", normalizeStdlibReturnType("error", "os"))
}

func TestNormalizeStdlibReturnType_BuiltinInt(t *testing.T) {
	assert.Equal(t, "builtin.int", normalizeStdlibReturnType("int", "math"))
}

func TestNormalizeStdlibReturnType_BuiltinBool(t *testing.T) {
	assert.Equal(t, "builtin.bool", normalizeStdlibReturnType("bool", "strings"))
}

func TestNormalizeStdlibReturnType_PointerType(t *testing.T) {
	assert.Equal(t, "net/http.Request", normalizeStdlibReturnType("*Request", "net/http"))
}

func TestNormalizeStdlibReturnType_UnqualifiedType(t *testing.T) {
	assert.Equal(t, "os.File", normalizeStdlibReturnType("File", "os"))
}

func TestNormalizeStdlibReturnType_SliceStripsToBuiltin(t *testing.T) {
	assert.Equal(t, "builtin.byte", normalizeStdlibReturnType("[]byte", "os"))
}

func TestNormalizeStdlibReturnType_CrossPackageType(t *testing.T) {
	// Types qualified with a different package name are returned as-is.
	assert.Equal(t, "io.Reader", normalizeStdlibReturnType("io.Reader", "net/http"))
}

func TestNormalizeStdlibReturnType_EmptyAfterStrip(t *testing.T) {
	// "[]" (slice of nothing) — edge-case guard.
	assert.Equal(t, "", normalizeStdlibReturnType("[]", "fmt"))
}

func TestNormalizeStdlibReturnType_PointerBuiltin(t *testing.T) {
	assert.Equal(t, "builtin.int", normalizeStdlibReturnType("*int", "sync/atomic"))
}

// -----------------------------------------------------------------------------
// inferTypeFromStdlibFunction
// -----------------------------------------------------------------------------

func TestInferTypeFromStdlibFunction_NilLoader(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	// StdlibLoader is nil by default.
	result := inferTypeFromStdlibFunction("fmt", "Sprintf", reg)
	assert.Nil(t, result)
}

func TestInferTypeFromStdlibFunction_NotStdlib(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
	}
	result := inferTypeFromStdlibFunction("github.com/myapp/utils", "GetUser", reg)
	assert.Nil(t, result)
}

func TestInferTypeFromStdlibFunction_FunctionNotFound(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
		functions:  map[string]*core.GoStdlibFunction{},
	}
	result := inferTypeFromStdlibFunction("fmt", "NonExistent", reg)
	assert.Nil(t, result)
}

func TestInferTypeFromStdlibFunction_NoReturns(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
		functions: map[string]*core.GoStdlibFunction{
			"fmt.Println": {
				Name:    "Println",
				Returns: []*core.GoReturnValue{}, // variadic print, returns (n int, err error) — simplified to empty
			},
		},
	}
	result := inferTypeFromStdlibFunction("fmt", "Println", reg)
	assert.Nil(t, result)
}

func TestInferTypeFromStdlibFunction_BuiltinReturn(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
		functions: map[string]*core.GoStdlibFunction{
			"fmt.Sprintf": {
				Name:    "Sprintf",
				Returns: []*core.GoReturnValue{{Type: "string"}},
			},
		},
	}
	result := inferTypeFromStdlibFunction("fmt", "Sprintf", reg)
	require.NotNil(t, result)
	assert.Equal(t, "builtin.string", result.TypeFQN)
	assert.InDelta(t, 0.9, float64(result.Confidence), 0.001)
	assert.Equal(t, "stdlib_registry", result.Source)
}

func TestInferTypeFromStdlibFunction_PointerReturn(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions: map[string]*core.GoStdlibFunction{
			"net/http.NewRequest": {
				Name: "NewRequest",
				Returns: []*core.GoReturnValue{
					{Type: "*Request"},
					{Type: "error"},
				},
			},
		},
	}
	result := inferTypeFromStdlibFunction("net/http", "NewRequest", reg)
	require.NotNil(t, result)
	assert.Equal(t, "net/http.Request", result.TypeFQN)
}

func TestInferTypeFromStdlibFunction_ErrorOnlyReturn(t *testing.T) {
	// Functions that only return error (e.g., os.Remove) should yield nil
	// because there is no useful concrete type to record.
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"os": true},
		functions: map[string]*core.GoStdlibFunction{
			"os.Remove": {
				Name:    "Remove",
				Returns: []*core.GoReturnValue{{Type: "error"}},
			},
		},
	}
	result := inferTypeFromStdlibFunction("os", "Remove", reg)
	assert.Nil(t, result)
}

func TestInferTypeFromStdlibFunction_SkipsErrorPicksFirst(t *testing.T) {
	// When the first return is error and the second is a real type,
	// the first non-error return should be chosen.
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = &mockStdlibLoader{
		stdlibPkgs: map[string]bool{"os": true},
		functions: map[string]*core.GoStdlibFunction{
			// hypothetical function returning (error, *File) order
			"os.OpenOrCreate": {
				Name: "OpenOrCreate",
				Returns: []*core.GoReturnValue{
					{Type: "error"},
					{Type: "*File"},
				},
			},
		},
	}
	result := inferTypeFromStdlibFunction("os", "OpenOrCreate", reg)
	require.NotNil(t, result)
	assert.Equal(t, "os.File", result.TypeFQN)
}

// -----------------------------------------------------------------------------
// Integration: ExtractGoVariableAssignments with stdlib loader
// -----------------------------------------------------------------------------

func TestExtractGoVariables_StdlibFunctionReturn(t *testing.T) {
	code := `package main

import "net/http"

func Handler() {
	resp, _ := http.Get("https://example.com")
	_ = resp
}`

	reg := &core.GoModuleRegistry{
		ModulePath:  "test",
		DirToImport: map[string]string{"/test": "test"},
		StdlibLoader: &mockStdlibLoader{
			stdlibPkgs: map[string]bool{"net/http": true},
			functions: map[string]*core.GoStdlibFunction{
				"net/http.Get": {
					Name: "Get",
					Returns: []*core.GoReturnValue{
						{Type: "*Response"},
						{Type: "error"},
					},
				},
			},
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)
	importMap := &core.GoImportMap{
		Imports: map[string]string{"http": "net/http"},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap)
	require.NoError(t, err)

	scope := typeEngine.GetScope("test.Handler")
	require.NotNil(t, scope, "scope for test.Handler should exist")

	bindings, ok := scope.Variables["resp"]
	require.True(t, ok, "resp should have a binding")
	require.NotEmpty(t, bindings)
	assert.Equal(t, "net/http.Response", bindings[0].Type.TypeFQN)
	assert.Equal(t, "stdlib_registry", bindings[0].Type.Source)
}

func TestExtractGoVariables_StdlibNoLoader(t *testing.T) {
	// Without a StdlibLoader, stdlib calls should leave variables untyped.
	code := `package main

import "os"

func ReadFile() {
	f, _ := os.Open("/etc/hosts")
	_ = f
}`

	reg := &core.GoModuleRegistry{
		ModulePath:  "test",
		DirToImport: map[string]string{"/test": "test"},
		// StdlibLoader intentionally nil
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)
	importMap := &core.GoImportMap{
		Imports: map[string]string{"os": "os"},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap)
	require.NoError(t, err)

	scope := typeEngine.GetScope("test.ReadFile")
	// Scope may be nil or f may have no binding — both are acceptable.
	if scope != nil {
		if bindings, ok := scope.Variables["f"]; ok {
			assert.Empty(t, bindings, "f should not have a type binding without StdlibLoader")
		}
	}
}

func TestExtractGoVariables_StdlibBuiltinReturn(t *testing.T) {
	// fmt.Sprintf returns string — verify builtin.string is recorded.
	code := `package main

import "fmt"

func Greet(name string) {
	msg := fmt.Sprintf("Hello, %s", name)
	_ = msg
}`

	reg := &core.GoModuleRegistry{
		ModulePath:  "test",
		DirToImport: map[string]string{"/test": "test"},
		StdlibLoader: &mockStdlibLoader{
			stdlibPkgs: map[string]bool{"fmt": true},
			functions: map[string]*core.GoStdlibFunction{
				"fmt.Sprintf": {
					Name:    "Sprintf",
					Returns: []*core.GoReturnValue{{Type: "string"}},
				},
			},
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(reg)
	importMap := &core.GoImportMap{
		Imports: map[string]string{"fmt": "fmt"},
	}

	err := ExtractGoVariableAssignments("/test/main.go", []byte(code), typeEngine, reg, importMap)
	require.NoError(t, err)

	scope := typeEngine.GetScope("test.Greet")
	require.NotNil(t, scope)
	bindings, ok := scope.Variables["msg"]
	require.True(t, ok)
	require.NotEmpty(t, bindings)
	assert.Equal(t, "builtin.string", bindings[0].Type.TypeFQN)
}
