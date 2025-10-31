package callgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableType(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add return type for a function
	engine.ReturnTypes["test.create_user"] = &TypeInfo{
		TypeFQN:    "test.User",
		Confidence: 0.9,
		Source:     "class_instantiation",
	}

	// Resolve variable type
	varType := engine.ResolveVariableType("test.create_user", 0.8)

	require.NotNil(t, varType)
	assert.Equal(t, "test.User", varType.TypeFQN)
	assert.Less(t, varType.Confidence, float32(0.9)) // Should be reduced
	assert.Equal(t, "function_call_propagation", varType.Source)
}

func TestResolveVariableType_NoReturnType(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// No return type registered
	varType := engine.ResolveVariableType("test.unknown_func", 1.0)

	assert.Nil(t, varType)
}

func TestUpdateVariableBindingsWithFunctionReturns(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Register return type
	engine.ReturnTypes["test.create_user"] = &TypeInfo{
		TypeFQN:    "test.User",
		Confidence: 0.9,
		Source:     "class_instantiation",
	}

	// Create scope with placeholder
	scope := &FunctionScope{
		FunctionFQN: "test.main",
		Variables: map[string]*VariableBinding{
			"user": {
				VarName: "user",
				Type: &TypeInfo{
					TypeFQN:    "call:create_user",
					Confidence: 0.5,
					Source:     "function_call",
				},
			},
		},
	}
	engine.AddScope(scope)

	// Update bindings
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify resolution
	userBinding := engine.Scopes["test.main"].Variables["user"]
	assert.Equal(t, "test.User", userBinding.Type.TypeFQN)
	assert.Equal(t, "test.create_user", userBinding.AssignedFrom)
	assert.Equal(t, "function_call_propagation", userBinding.Type.Source)
}

func TestUpdateVariableBindingsWithFunctionReturns_NoMatch(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// No return type registered for unknown_func
	scope := &FunctionScope{
		FunctionFQN: "test.main",
		Variables: map[string]*VariableBinding{
			"obj": {
				VarName: "obj",
				Type: &TypeInfo{
					TypeFQN:    "call:unknown_func",
					Confidence: 0.5,
					Source:     "function_call",
				},
			},
		},
	}
	engine.AddScope(scope)

	// Update bindings
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Should remain unresolved
	objBinding := engine.Scopes["test.main"].Variables["obj"]
	assert.Equal(t, "call:unknown_func", objBinding.Type.TypeFQN)
}

func TestUpdateVariableBindingsWithFunctionReturns_Literals(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Scope with literal (not a call placeholder)
	scope := &FunctionScope{
		FunctionFQN: "test.main",
		Variables: map[string]*VariableBinding{
			"name": {
				VarName: "name",
				Type: &TypeInfo{
					TypeFQN:    "builtins.str",
					Confidence: 1.0,
					Source:     "literal",
				},
			},
		},
	}
	engine.AddScope(scope)

	// Update bindings
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Literal should remain unchanged
	nameBinding := engine.Scopes["test.main"].Variables["name"]
	assert.Equal(t, "builtins.str", nameBinding.Type.TypeFQN)
	assert.Equal(t, "literal", nameBinding.Type.Source)
}

func TestUpdateVariableBindingsWithFunctionReturns_ModuleLevel(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Register return type
	engine.ReturnTypes["mymodule.get_config"] = &TypeInfo{
		TypeFQN:    "builtins.dict",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	// Create module-level scope (no dots in FQN)
	scope := &FunctionScope{
		FunctionFQN: "mymodule",
		Variables: map[string]*VariableBinding{
			"config": {
				VarName: "config",
				Type: &TypeInfo{
					TypeFQN:    "call:get_config",
					Confidence: 0.5,
					Source:     "function_call",
				},
			},
		},
	}
	engine.AddScope(scope)

	// Update bindings
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify resolution
	configBinding := engine.Scopes["mymodule"].Variables["config"]
	assert.Equal(t, "builtins.dict", configBinding.Type.TypeFQN)
	assert.Equal(t, "mymodule.get_config", configBinding.AssignedFrom)
}
