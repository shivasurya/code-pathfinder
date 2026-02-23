package resolution

import (
	"fmt"
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

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
