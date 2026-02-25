package resolution

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// errNotFound is returned by mockStdlibLoader for unknown packages/functions.
var errNotFound = errors.New("not found")

// mockStdlibLoader implements core.GoStdlibLoader for testing without network access.
type mockGoTypesStdlibLoader struct {
	packages  map[string]bool
	functions map[string]*core.GoStdlibFunction // key: "importPath.funcName"
}

func (m *mockGoTypesStdlibLoader) ValidateStdlibImport(importPath string) bool {
	return m.packages[importPath]
}

func (m *mockGoTypesStdlibLoader) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	key := importPath + "." + funcName
	fn, ok := m.functions[key]
	if !ok {
		return nil, errNotFound
	}
	return fn, nil
}

func (m *mockGoTypesStdlibLoader) GetType(_, _ string) (*core.GoStdlibType, error) {
	return nil, errNotFound
}

func (m *mockGoTypesStdlibLoader) PackageCount() int {
	return len(m.packages)
}

// ===== Engine Creation Tests =====

func TestGoTypeInferenceEngine_NewEngine(t *testing.T) {
	registry := &core.GoModuleRegistry{
		ModulePath: "github.com/example/myapp",
	}

	engine := NewGoTypeInferenceEngine(registry)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.Scopes)
	assert.NotNil(t, engine.ReturnTypes)
	assert.Equal(t, registry, engine.Registry)
	assert.Equal(t, 0, len(engine.Scopes))
	assert.Equal(t, 0, len(engine.ReturnTypes))
}

func TestGoTypeInferenceEngine_NewEngine_NilRegistry(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.Scopes)
	assert.NotNil(t, engine.ReturnTypes)
	assert.Nil(t, engine.Registry)
}

// ===== Scope Operations Tests =====

func TestGoTypeInferenceEngine_AddGetScope(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	scope := NewGoFunctionScope("myapp.HandleRequest")
	engine.AddScope(scope)

	retrieved := engine.GetScope("myapp.HandleRequest")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "myapp.HandleRequest", retrieved.FunctionFQN)
}

func TestGoTypeInferenceEngine_GetScope_NotFound(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	retrieved := engine.GetScope("nonexistent")
	assert.Nil(t, retrieved)
}

func TestGoTypeInferenceEngine_HasScope(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	scope := NewGoFunctionScope("myapp.Handler")
	engine.AddScope(scope)

	assert.True(t, engine.HasScope("myapp.Handler"))
	assert.False(t, engine.HasScope("myapp.NonExistent"))
}

func TestGoTypeInferenceEngine_AddScope_Nil(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	engine.AddScope(nil)

	assert.Equal(t, 0, len(engine.GetAllScopes()))
}

func TestGoTypeInferenceEngine_GetAllScopes(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	scope1 := NewGoFunctionScope("myapp.Handler1")
	scope2 := NewGoFunctionScope("myapp.Handler2")
	engine.AddScope(scope1)
	engine.AddScope(scope2)

	allScopes := engine.GetAllScopes()
	assert.Len(t, allScopes, 2)
	assert.Contains(t, allScopes, "myapp.Handler1")
	assert.Contains(t, allScopes, "myapp.Handler2")

	// Verify it's a copy - modifications don't affect original
	allScopes["myapp.Handler3"] = NewGoFunctionScope("myapp.Handler3")
	assert.Len(t, engine.GetAllScopes(), 2) // Still 2
}

// ===== Return Type Operations Tests =====

func TestGoTypeInferenceEngine_AddGetReturnType(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	typeInfo := &core.TypeInfo{
		TypeFQN:    "myapp.User",
		Confidence: 0.95,
		Source:     "declaration",
	}

	engine.AddReturnType("myapp.GetUser", typeInfo)

	retrieved, ok := engine.GetReturnType("myapp.GetUser")
	assert.True(t, ok)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "myapp.User", retrieved.TypeFQN)
	assert.Equal(t, float32(0.95), retrieved.Confidence)
	assert.Equal(t, "declaration", retrieved.Source)
}

func TestGoTypeInferenceEngine_GetReturnType_NotFound(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	retrieved, ok := engine.GetReturnType("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestGoTypeInferenceEngine_HasReturnType(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	typeInfo := &core.TypeInfo{TypeFQN: "myapp.User"}
	engine.AddReturnType("myapp.GetUser", typeInfo)

	assert.True(t, engine.HasReturnType("myapp.GetUser"))
	assert.False(t, engine.HasReturnType("myapp.NonExistent"))
}

func TestGoTypeInferenceEngine_AddReturnType_Nil(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	engine.AddReturnType("myapp.Func", nil)

	_, ok := engine.GetReturnType("myapp.Func")
	assert.False(t, ok)
}

func TestGoTypeInferenceEngine_GetAllReturnTypes(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	type1 := &core.TypeInfo{TypeFQN: "myapp.User"}
	type2 := &core.TypeInfo{TypeFQN: "myapp.Config"}
	engine.AddReturnType("myapp.GetUser", type1)
	engine.AddReturnType("myapp.GetConfig", type2)

	allTypes := engine.GetAllReturnTypes()
	assert.Len(t, allTypes, 2)
	assert.Contains(t, allTypes, "myapp.GetUser")
	assert.Contains(t, allTypes, "myapp.GetConfig")

	// Verify it's a copy
	allTypes["myapp.NewFunc"] = &core.TypeInfo{TypeFQN: "myapp.New"}
	assert.Len(t, engine.GetAllReturnTypes(), 2)
}

// ===== Function Scope Tests =====

func TestGoFunctionScope_NewScope(t *testing.T) {
	scope := NewGoFunctionScope("myapp.HandleRequest")

	assert.NotNil(t, scope)
	assert.Equal(t, "myapp.HandleRequest", scope.FunctionFQN)
	assert.NotNil(t, scope.Variables)
	assert.Equal(t, 0, len(scope.Variables))
}

func TestGoFunctionScope_AddVariable(t *testing.T) {
	scope := NewGoFunctionScope("myapp.HandleRequest")

	binding := &GoVariableBinding{
		VarName: "user",
		Type:    &core.TypeInfo{TypeFQN: "myapp.User"},
	}

	scope.AddVariable(binding)

	assert.True(t, scope.HasVariable("user"))
	retrieved := scope.GetVariable("user")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "user", retrieved.VarName)
}

func TestGoFunctionScope_AddVariable_Nil(t *testing.T) {
	scope := NewGoFunctionScope("myapp.Handler")

	scope.AddVariable(nil)

	assert.Equal(t, 0, len(scope.Variables))
}

func TestGoFunctionScope_GetVariable_NotFound(t *testing.T) {
	scope := NewGoFunctionScope("myapp.Handler")

	retrieved := scope.GetVariable("nonexistent")
	assert.Nil(t, retrieved)
}

func TestGoFunctionScope_MultipleBindings(t *testing.T) {
	scope := NewGoFunctionScope("myapp.HandleRequest")

	// First assignment: user := GetUser(1)
	scope.AddVariable(&GoVariableBinding{
		VarName:      "user",
		Type:         &core.TypeInfo{TypeFQN: "myapp.User"},
		AssignedFrom: "myapp.GetUser",
		Location:     Location{File: "handler.go", Line: 10},
	})

	// Reassignment: user = NewUser()
	scope.AddVariable(&GoVariableBinding{
		VarName:      "user",
		Type:         &core.TypeInfo{TypeFQN: "myapp.User"},
		AssignedFrom: "myapp.NewUser",
		Location:     Location{File: "handler.go", Line: 20},
	})

	// GetVariable should return latest (line 20)
	latest := scope.GetVariable("user")
	assert.NotNil(t, latest)
	assert.Equal(t, uint32(20), latest.Location.Line)
	assert.Equal(t, "myapp.NewUser", latest.AssignedFrom)

	// GetAllBindings should return both
	all := scope.GetAllBindings("user")
	assert.Len(t, all, 2)
	assert.Equal(t, uint32(10), all[0].Location.Line)
	assert.Equal(t, uint32(20), all[1].Location.Line)
}

func TestGoFunctionScope_HasVariable(t *testing.T) {
	scope := NewGoFunctionScope("myapp.Handler")

	assert.False(t, scope.HasVariable("user"))

	scope.AddVariable(&GoVariableBinding{VarName: "user"})
	assert.True(t, scope.HasVariable("user"))
}

func TestGoFunctionScope_GetAllBindings_Empty(t *testing.T) {
	scope := NewGoFunctionScope("myapp.Handler")

	bindings := scope.GetAllBindings("nonexistent")
	assert.Nil(t, bindings)
}

// ===== Thread Safety Tests =====

func TestGoTypeInferenceEngine_ThreadSafety(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes to scopes
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fqn := fmt.Sprintf("myapp.Func%d", i)
			scope := NewGoFunctionScope(fqn)
			engine.AddScope(scope)
		}(i)
	}

	// Concurrent writes to return types
	for i := range numGoroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fqn := fmt.Sprintf("myapp.Func%d", i)
			typeInfo := &core.TypeInfo{TypeFQN: "myapp.Type"}
			engine.AddReturnType(fqn, typeInfo)
		}(i)
	}

	wg.Wait()

	// Verify all added correctly
	scopes := engine.GetAllScopes()
	assert.Len(t, scopes, numGoroutines)

	types := engine.GetAllReturnTypes()
	assert.Len(t, types, numGoroutines)
}

func TestGoTypeInferenceEngine_ConcurrentReads(t *testing.T) {
	engine := NewGoTypeInferenceEngine(nil)

	// Setup data
	for i := range 10 {
		fqn := fmt.Sprintf("myapp.Func%d", i)
		scope := NewGoFunctionScope(fqn)
		engine.AddScope(scope)
		engine.AddReturnType(fqn, &core.TypeInfo{TypeFQN: "myapp.Type"})
	}

	var wg sync.WaitGroup
	numReaders := 100

	// Concurrent reads
	for i := range numReaders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fqn := fmt.Sprintf("myapp.Func%d", i%10)

			scope := engine.GetScope(fqn)
			assert.NotNil(t, scope)

			typeInfo, ok := engine.GetReturnType(fqn)
			assert.True(t, ok)
			assert.NotNil(t, typeInfo)
		}(i)
	}

	wg.Wait()
}

// =============================================================================
// GetReturnType — stdlib fallback
// =============================================================================

func TestGetReturnType_StdlibFallback_SimpleFunction(t *testing.T) {
	// fmt.Sprintf returns string → expect builtin.string with confidence 1.0
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"fmt": true},
			functions: map[string]*core.GoStdlibFunction{
				"fmt.Sprintf": {
					Name:    "Sprintf",
					Returns: []*core.GoReturnValue{{Type: "string"}},
				},
			},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("fmt.Sprintf")

	assert.True(t, ok)
	assert.Equal(t, "builtin.string", info.TypeFQN)
	assert.Equal(t, float32(1.0), info.Confidence)
	assert.Equal(t, "stdlib", info.Source)
}

func TestGetReturnType_StdlibFallback_PointerReturn(t *testing.T) {
	// net/http.Get returns (*Response, error) → expect net/http.Response
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"net/http": true},
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
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("net/http.Get")

	assert.True(t, ok)
	assert.Equal(t, "net/http.Response", info.TypeFQN)
	assert.Equal(t, float32(1.0), info.Confidence)
	assert.Equal(t, "stdlib", info.Source)
}

func TestGetReturnType_StdlibFallback_CrossPackageType(t *testing.T) {
	// A function that returns io.Reader (cross-package qualified) → returned as-is.
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"bufio": true},
			functions: map[string]*core.GoStdlibFunction{
				"bufio.NewReader": {
					Name:    "NewReader",
					Returns: []*core.GoReturnValue{{Type: "io.Reader"}},
				},
			},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("bufio.NewReader")

	assert.True(t, ok)
	assert.Equal(t, "io.Reader", info.TypeFQN)
}

func TestGetReturnType_StdlibFallback_ErrorOnlyReturn(t *testing.T) {
	// os.Remove returns only error → no usable type, expect (nil, false)
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"os": true},
			functions: map[string]*core.GoStdlibFunction{
				"os.Remove": {
					Name:    "Remove",
					Returns: []*core.GoReturnValue{{Type: "error"}},
				},
			},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("os.Remove")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestGetReturnType_StdlibFallback_NoReturns(t *testing.T) {
	// A void stdlib function → (nil, false)
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"fmt": true},
			functions: map[string]*core.GoStdlibFunction{
				"fmt.Println": {
					Name:    "Println",
					Returns: []*core.GoReturnValue{},
				},
			},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("fmt.Println")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestGetReturnType_StdlibFallback_FunctionNotFound(t *testing.T) {
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages:  map[string]bool{"fmt": true},
			functions: map[string]*core.GoStdlibFunction{},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("fmt.NonExistent")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestGetReturnType_StdlibFallback_NotStdlibPackage(t *testing.T) {
	// Package is known to the loader but not in the stdlib set.
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages:  map[string]bool{"fmt": true}, // only "fmt" is stdlib
			functions: map[string]*core.GoStdlibFunction{},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	// Third-party package — not in stdlib
	info, ok := engine.GetReturnType("github.com/myapp/utils.GetUser")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestGetReturnType_StdlibFallback_NilStdlibLoader(t *testing.T) {
	// Registry present but StdlibLoader nil — must not panic, return (nil, false).
	reg := &core.GoModuleRegistry{
		ModulePath:   "myapp",
		StdlibLoader: nil,
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("fmt.Sprintf")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestGetReturnType_StdlibFallback_NoDotInFQN(t *testing.T) {
	// Malformed FQN with no dot — must not panic.
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"fmt": true},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("nodot")

	assert.False(t, ok)
	assert.Nil(t, info)
}

func TestGetReturnType_LocalTakesPriorityOverStdlib(t *testing.T) {
	// A locally-registered type for a stdlib FQN must win over the stdlib lookup.
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"fmt": true},
			functions: map[string]*core.GoStdlibFunction{
				"fmt.Sprintf": {
					Name:    "Sprintf",
					Returns: []*core.GoReturnValue{{Type: "string"}},
				},
			},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	// Register a local override (e.g., from a parsed user wrapper).
	local := &core.TypeInfo{TypeFQN: "myapp.CustomString", Confidence: 0.8, Source: "declaration"}
	engine.AddReturnType("fmt.Sprintf", local)

	info, ok := engine.GetReturnType("fmt.Sprintf")

	assert.True(t, ok)
	assert.Equal(t, "myapp.CustomString", info.TypeFQN, "local registration must win over stdlib")
	assert.Equal(t, "declaration", info.Source)
}

// =============================================================================
// stdlibNormalizeType
// =============================================================================

func TestStdlibNormalizeType_BuiltinString(t *testing.T) {
	assert.Equal(t, "builtin.string", stdlibNormalizeType("string", "fmt"))
}

func TestStdlibNormalizeType_BuiltinError(t *testing.T) {
	assert.Equal(t, "builtin.error", stdlibNormalizeType("error", "os"))
}

func TestStdlibNormalizeType_BuiltinInt(t *testing.T) {
	assert.Equal(t, "builtin.int", stdlibNormalizeType("int", "math"))
}

func TestStdlibNormalizeType_BuiltinBool(t *testing.T) {
	assert.Equal(t, "builtin.bool", stdlibNormalizeType("bool", "strings"))
}

func TestStdlibNormalizeType_BuiltinByte(t *testing.T) {
	assert.Equal(t, "builtin.byte", stdlibNormalizeType("byte", "os"))
}

func TestStdlibNormalizeType_BuiltinRune(t *testing.T) {
	assert.Equal(t, "builtin.rune", stdlibNormalizeType("rune", "unicode"))
}

func TestStdlibNormalizeType_PointerBuiltin(t *testing.T) {
	assert.Equal(t, "builtin.int", stdlibNormalizeType("*int", "sync/atomic"))
}

func TestStdlibNormalizeType_PointerPackageType(t *testing.T) {
	assert.Equal(t, "net/http.Request", stdlibNormalizeType("*Request", "net/http"))
}

func TestStdlibNormalizeType_SliceBuiltin(t *testing.T) {
	assert.Equal(t, "builtin.byte", stdlibNormalizeType("[]byte", "os"))
}

func TestStdlibNormalizeType_UnqualifiedType(t *testing.T) {
	assert.Equal(t, "os.File", stdlibNormalizeType("File", "os"))
}

func TestStdlibNormalizeType_CrossPackageType(t *testing.T) {
	assert.Equal(t, "io.Reader", stdlibNormalizeType("io.Reader", "net/http"))
}

func TestStdlibNormalizeType_EmptyAfterStrip(t *testing.T) {
	// "[]" stripped to "" — edge case.
	assert.Equal(t, "", stdlibNormalizeType("[]", "fmt"))
}

func TestGetReturnType_StdlibFallback_EmptyTypeFQNSkipped(t *testing.T) {
	// First return is "[]" (normalizes to ""), second is a real type.
	// The loop must skip the empty one and use the second.
	reg := &core.GoModuleRegistry{
		StdlibLoader: &mockGoTypesStdlibLoader{
			packages: map[string]bool{"os": true},
			functions: map[string]*core.GoStdlibFunction{
				"os.Weird": {
					Name: "Weird",
					Returns: []*core.GoReturnValue{
						{Type: "[]"}, // normalizes to ""
						{Type: "File"},
					},
				},
			},
		},
	}
	engine := NewGoTypeInferenceEngine(reg)

	info, ok := engine.GetReturnType("os.Weird")

	assert.True(t, ok)
	assert.Equal(t, "os.File", info.TypeFQN)
}

func TestStdlibNormalizeType_AllNumericBuiltins(t *testing.T) {
	for _, typ := range []string{
		"int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
	} {
		assert.Equal(t, "builtin."+typ, stdlibNormalizeType(typ, "math"))
	}
}
