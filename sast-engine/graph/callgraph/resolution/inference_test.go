package resolution

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
)

// TestTypeInfo_Creation tests TypeInfo struct creation and field access.
func TestTypeInfo_Creation(t *testing.T) {
	tests := []struct {
		name       string
		typeFQN    string
		confidence float32
		source     string
	}{
		{
			name:       "builtin string type",
			typeFQN:    "builtins.str",
			confidence: 1.0,
			source:     "literal",
		},
		{
			name:       "user-defined class type",
			typeFQN:    "myapp.models.User",
			confidence: 0.9,
			source:     "annotation",
		},
		{
			name:       "unknown type",
			typeFQN:    "",
			confidence: 0.0,
			source:     "unknown",
		},
		{
			name:       "heuristic inference",
			typeFQN:    "builtins.list",
			confidence: 0.5,
			source:     "heuristic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo := &core.TypeInfo{
				TypeFQN:    tt.typeFQN,
				Confidence: tt.confidence,
				Source:     tt.source,
			}

			assert.Equal(t, tt.typeFQN, typeInfo.TypeFQN)
			assert.Equal(t, tt.confidence, typeInfo.Confidence)
			assert.Equal(t, tt.source, typeInfo.Source)
		})
	}
}

// TestVariableBinding_Creation tests VariableBinding struct creation.
func TestVariableBinding_Creation(t *testing.T) {
	tests := []struct {
		name         string
		varName      string
		typeFQN      string
		confidence   float32
		source       string
		assignedFrom string
		location     Location
	}{
		{
			name:         "simple variable",
			varName:      "user",
			typeFQN:      "myapp.models.User",
			confidence:   1.0,
			source:       "assignment",
			assignedFrom: "myapp.controllers.get_user",
			location: Location{
				File:   "/path/to/file.py",
				Line:   10,
				Column: 5,
			},
		},
		{
			name:         "builtin variable",
			varName:      "data",
			typeFQN:      "builtins.str",
			confidence:   1.0,
			source:       "literal",
			assignedFrom: "",
			location: Location{
				File:   "/path/to/file.py",
				Line:   20,
				Column: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo := &core.TypeInfo{
				TypeFQN:    tt.typeFQN,
				Confidence: tt.confidence,
				Source:     tt.source,
			}

			binding := &VariableBinding{
				VarName:      tt.varName,
				Type:         typeInfo,
				AssignedFrom: tt.assignedFrom,
				Location:     tt.location,
			}

			assert.Equal(t, tt.varName, binding.VarName)
			assert.Equal(t, tt.typeFQN, binding.Type.TypeFQN)
			assert.Equal(t, tt.confidence, binding.Type.Confidence)
			assert.Equal(t, tt.source, binding.Type.Source)
			assert.Equal(t, tt.assignedFrom, binding.AssignedFrom)
			assert.Equal(t, tt.location.File, binding.Location.File)
			assert.Equal(t, tt.location.Line, binding.Location.Line)
			assert.Equal(t, tt.location.Column, binding.Location.Column)
		})
	}
}

// TestFunctionScope_Creation tests FunctionScope struct creation.
func TestFunctionScope_Creation(t *testing.T) {
	tests := []struct {
		name        string
		functionFQN string
	}{
		{
			name:        "module-level function",
			functionFQN: "myapp.controllers.get_user",
		},
		{
			name:        "class method",
			functionFQN: "myapp.models.User.save",
		},
		{
			name:        "nested function",
			functionFQN: "myapp.utils.process_data.helper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := NewFunctionScope(tt.functionFQN)

			assert.NotNil(t, scope)
			assert.Equal(t, tt.functionFQN, scope.FunctionFQN)
			assert.NotNil(t, scope.Variables)
			assert.Equal(t, 0, len(scope.Variables))
			assert.Nil(t, scope.ReturnType)
		})
	}
}

// TestFunctionScope_AddVariable tests adding variables to a scope.
func TestFunctionScope_AddVariable(t *testing.T) {
	scope := NewFunctionScope("myapp.controllers.get_user")

	// Add first variable
	binding1 := &VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "assignment",
		},
		Location: Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}
	scope.Variables["user"] = []*VariableBinding{binding1}

	// Add second variable
	binding2 := &VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 0.9,
			Source:     "heuristic",
		},
		Location: Location{File: "/path/to/file.py", Line: 15, Column: 5},
	}
	scope.Variables["result"] = []*VariableBinding{binding2}

	// Verify both variables exist
	assert.Equal(t, 2, len(scope.Variables))
	assert.Equal(t, "myapp.models.User", scope.Variables["user"][0].Type.TypeFQN)
	assert.Equal(t, "builtins.dict", scope.Variables["result"][0].Type.TypeFQN)

	// Update existing variable (append new binding)
	binding3 := &VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "annotation",
		},
		Location: Location{File: "/path/to/file.py", Line: 20, Column: 5},
	}
	scope.Variables["user"] = []*VariableBinding{binding3}

	// Verify update
	assert.Equal(t, 2, len(scope.Variables))
	assert.Equal(t, "annotation", scope.Variables["user"][0].Type.Source)
}

// TestFunctionScope_ReturnType tests setting return type on a scope.
func TestFunctionScope_ReturnType(t *testing.T) {
	scope := NewFunctionScope("myapp.controllers.get_user")

	// Initially nil
	assert.Nil(t, scope.ReturnType)

	// Set return type
	returnType := &core.TypeInfo{
		TypeFQN:    "myapp.models.User",
		Confidence: 1.0,
		Source:     "annotation",
	}
	scope.ReturnType = returnType

	// Verify
	assert.NotNil(t, scope.ReturnType)
	assert.Equal(t, "myapp.models.User", scope.ReturnType.TypeFQN)
	assert.Equal(t, float32(1.0), scope.ReturnType.Confidence)
	assert.Equal(t, "annotation", scope.ReturnType.Source)
}

// TestTypeInferenceEngine_Creation tests TypeInferenceEngine initialization.
func TestTypeInferenceEngine_Creation(t *testing.T) {
	registry := core.NewModuleRegistry()

	engine := NewTypeInferenceEngine(registry)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.Scopes)
	assert.NotNil(t, engine.ReturnTypes)
	assert.Equal(t, 0, len(engine.Scopes))
	assert.Equal(t, 0, len(engine.ReturnTypes))
	assert.Nil(t, engine.Builtins) // Not initialized by default
	assert.Equal(t, registry, engine.Registry)
}

// TestTypeInferenceEngine_AddAndGetScope tests scope management.
func TestTypeInferenceEngine_AddAndGetScope(t *testing.T) {
	registry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Initially no scopes
	assert.Nil(t, engine.GetScope("myapp.controllers.get_user"))

	// Add first scope
	scope1 := NewFunctionScope("myapp.controllers.get_user")
	scope1.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "assignment",
		},
		Location: Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}}
	engine.AddScope(scope1)

	// Verify first scope
	retrieved1 := engine.GetScope("myapp.controllers.get_user")
	assert.NotNil(t, retrieved1)
	assert.Equal(t, "myapp.controllers.get_user", retrieved1.FunctionFQN)
	assert.Equal(t, 1, len(retrieved1.Variables))

	// Add second scope
	scope2 := NewFunctionScope("myapp.models.User.save")
	engine.AddScope(scope2)

	// Verify both scopes exist
	assert.Equal(t, 2, len(engine.Scopes))
	assert.NotNil(t, engine.GetScope("myapp.controllers.get_user"))
	assert.NotNil(t, engine.GetScope("myapp.models.User.save"))

	// Non-existent scope returns nil
	assert.Nil(t, engine.GetScope("nonexistent.function"))
}

// TestTypeInferenceEngine_AddNilScope tests that adding nil scope is handled gracefully.
func TestTypeInferenceEngine_AddNilScope(t *testing.T) {
	registry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add nil scope should not panic
	engine.AddScope(nil)

	// Verify no scopes added
	assert.Equal(t, 0, len(engine.Scopes))
}

// TestTypeInferenceEngine_UpdateScope tests updating an existing scope.
func TestTypeInferenceEngine_UpdateScope(t *testing.T) {
	registry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add initial scope
	scope1 := NewFunctionScope("myapp.controllers.get_user")
	scope1.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 0.8,
			Source:     "heuristic",
		},
		Location: Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}}
	engine.AddScope(scope1)

	// Update with new scope
	scope2 := NewFunctionScope("myapp.controllers.get_user")
	scope2.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "annotation",
		},
		Location: Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}}
	scope2.Variables["result"] = []*VariableBinding{&VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "literal",
		},
		Location: Location{File: "/path/to/file.py", Line: 15, Column: 5},
	}}
	engine.AddScope(scope2)

	// Verify updated scope
	retrieved := engine.GetScope("myapp.controllers.get_user")
	assert.NotNil(t, retrieved)
	assert.Equal(t, 2, len(retrieved.Variables))
	assert.Equal(t, float32(1.0), retrieved.Variables["user"][0].Type.Confidence)
	assert.Equal(t, "annotation", retrieved.Variables["user"][0].Type.Source)
	assert.NotNil(t, retrieved.Variables["result"])

	// Still only one scope in total
	assert.Equal(t, 1, len(engine.Scopes))
}

// TestTypeInfo_ConfidenceValidation tests confidence score edge cases.
func TestTypeInfo_ConfidenceValidation(t *testing.T) {
	tests := []struct {
		name       string
		confidence float32
	}{
		{name: "zero confidence", confidence: 0.0},
		{name: "low confidence", confidence: 0.1},
		{name: "medium confidence", confidence: 0.5},
		{name: "high confidence", confidence: 0.9},
		{name: "full confidence", confidence: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo := &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: tt.confidence,
				Source:     "test",
			}

			assert.Equal(t, tt.confidence, typeInfo.Confidence)
			assert.GreaterOrEqual(t, typeInfo.Confidence, float32(0.0))
			assert.LessOrEqual(t, typeInfo.Confidence, float32(1.0))
		})
	}
}

// TestTypeInferenceEngine_ReturnTypeTracking tests tracking return types.
func TestTypeInferenceEngine_ReturnTypeTracking(t *testing.T) {
	registry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add return type for a function
	returnType1 := &core.TypeInfo{
		TypeFQN:    "myapp.models.User",
		Confidence: 1.0,
		Source:     "annotation",
	}
	engine.ReturnTypes["myapp.controllers.get_user"] = returnType1

	// Add return type for another function
	returnType2 := &core.TypeInfo{
		TypeFQN:    "builtins.dict",
		Confidence: 0.9,
		Source:     "heuristic",
	}
	engine.ReturnTypes["myapp.utils.process_data"] = returnType2

	// Verify both return types
	assert.Equal(t, 2, len(engine.ReturnTypes))
	assert.Equal(t, "myapp.models.User", engine.ReturnTypes["myapp.controllers.get_user"].TypeFQN)
	assert.Equal(t, "builtins.dict", engine.ReturnTypes["myapp.utils.process_data"].TypeFQN)

	// Non-existent function returns nil
	assert.Nil(t, engine.ReturnTypes["nonexistent.function"])
}

// TestTypeInferenceEngine_WithBuiltinRegistry tests using the builtin registry.
func TestTypeInferenceEngine_WithBuiltinRegistry(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Initially nil
	assert.Nil(t, engine.Builtins)

	// Set builtin registry
	engine.Builtins = registry.NewBuiltinRegistry()
	assert.NotNil(t, engine.Builtins)

	// Verify we can access builtin types
	strType := engine.Builtins.GetType("builtins.str")
	assert.NotNil(t, strType)
	assert.Equal(t, "builtins.str", strType.FQN)

	// Verify we can get builtin methods
	upperMethod := engine.Builtins.GetMethod("builtins.str", "upper")
	assert.NotNil(t, upperMethod)
	assert.Equal(t, "builtins.str", upperMethod.ReturnType.TypeFQN)

	// Verify literal type inference
	typeInfo := engine.Builtins.InferLiteralType(`"hello"`)
	assert.NotNil(t, typeInfo)
	assert.Equal(t, "builtins.str", typeInfo.TypeFQN)
	assert.Equal(t, float32(1.0), typeInfo.Confidence)
}

// TestTypeInferenceEngine_ResolveVariableType tests variable type resolution from function returns.
func TestTypeInferenceEngine_ResolveVariableType(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Add a return type for a function
	engine.ReturnTypes["myapp.models.get_user"] = &core.TypeInfo{
		TypeFQN:    "myapp.models.User",
		Confidence: 1.0,
		Source:     "return_annotation",
	}

	// Test resolving variable type from function return
	resolvedType := engine.ResolveVariableType("myapp.models.get_user", 1.0)
	assert.NotNil(t, resolvedType)
	assert.Equal(t, "myapp.models.User", resolvedType.TypeFQN)
	assert.Equal(t, "function_call_propagation", resolvedType.Source)
	// Confidence should be reduced: 1.0 * 1.0 * 0.95 = 0.95
	assert.Equal(t, float32(0.95), resolvedType.Confidence)

	// Test with lower base confidence
	resolvedType2 := engine.ResolveVariableType("myapp.models.get_user", 0.8)
	assert.NotNil(t, resolvedType2)
	assert.Equal(t, "myapp.models.User", resolvedType2.TypeFQN)
	// Confidence: 1.0 * 0.8 * 0.95 = 0.76
	assert.Equal(t, float32(0.76), resolvedType2.Confidence)

	// Test with function that has no return type
	resolvedType3 := engine.ResolveVariableType("nonexistent.function", 1.0)
	assert.Nil(t, resolvedType3)
}

// TestTypeInferenceEngine_UpdateVariableBindingsWithFunctionReturns tests updating call: placeholders.
func TestTypeInferenceEngine_UpdateVariableBindingsWithFunctionReturns(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return types for functions
	// Note: Simple names in call: will be qualified with scope's module path
	// Scope is myapp.controllers.login, so create_user becomes myapp.controllers.create_user
	engine.ReturnTypes["myapp.controllers.create_user"] = &core.TypeInfo{
		TypeFQN:    "myapp.models.User",
		Confidence: 1.0,
		Source:     "return_literal",
	}
	engine.ReturnTypes["myapp.controllers.get_config"] = &core.TypeInfo{
		TypeFQN:    "builtins.dict",
		Confidence: 0.9,
		Source:     "return_literal",
	}

	// Create a scope with call: placeholders
	scope := NewFunctionScope("myapp.controllers.login")
	scope.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "call:create_user",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}
	scope.Variables["config"] = []*VariableBinding{&VariableBinding{
		VarName: "config",
		Type: &core.TypeInfo{
			TypeFQN:    "call:get_config",
			Confidence: 0.9,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 15, Column: 5},
	}}
	scope.Variables["name"] = []*VariableBinding{&VariableBinding{
		VarName: "name",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
		Location: Location{File: "/test/file.py", Line: 20, Column: 5},
	}}

	engine.AddScope(scope)

	// Update variable bindings
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify user was resolved
	userBinding := engine.GetScope("myapp.controllers.login").Variables["user"][0]
	assert.Equal(t, "myapp.models.User", userBinding.Type.TypeFQN)
	assert.Equal(t, "function_call_propagation", userBinding.Type.Source)
	assert.Equal(t, "myapp.controllers.create_user", userBinding.AssignedFrom)
	// Confidence: 1.0 * 0.8 * 0.95 = 0.76
	assert.Equal(t, float32(0.76), userBinding.Type.Confidence)

	// Verify config was resolved
	configBinding := engine.GetScope("myapp.controllers.login").Variables["config"][0]
	assert.Equal(t, "builtins.dict", configBinding.Type.TypeFQN)
	assert.Equal(t, "function_call_propagation", configBinding.Type.Source)
	assert.Equal(t, "myapp.controllers.get_config", configBinding.AssignedFrom)

	// Verify name was NOT changed (not a call: placeholder)
	nameBinding := engine.GetScope("myapp.controllers.login").Variables["name"][0]
	assert.Equal(t, "builtins.str", nameBinding.Type.TypeFQN)
	assert.Equal(t, "literal", nameBinding.Type.Source)
}

// TestTypeInferenceEngine_UpdateVariableBindings_QualifiedName tests qualified function calls.
func TestTypeInferenceEngine_UpdateVariableBindings_QualifiedName(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return type for qualified function
	engine.ReturnTypes["logging.getLogger"] = &core.TypeInfo{
		TypeFQN:    "logging.Logger",
		Confidence: 1.0,
		Source:     "stdlib",
	}

	// Create scope with qualified call
	scope := NewFunctionScope("myapp.utils.helper")
	scope.Variables["logger"] = []*VariableBinding{&VariableBinding{
		VarName: "logger",
		Type: &core.TypeInfo{
			TypeFQN:    "call:logging.getLogger",
			Confidence: 1.0,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify logger was resolved using the qualified name
	loggerBinding := engine.GetScope("myapp.utils.helper").Variables["logger"][0]
	assert.Equal(t, "logging.Logger", loggerBinding.Type.TypeFQN)
	assert.Equal(t, "logging.getLogger", loggerBinding.AssignedFrom)
}

// TestTypeInferenceEngine_UpdateVariableBindings_ModuleLevelScope tests module-level function.
func TestTypeInferenceEngine_UpdateVariableBindings_ModuleLevelScope(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return type
	engine.ReturnTypes["myapp.helper"] = &core.TypeInfo{
		TypeFQN:    "builtins.str",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	// Create module-level scope (no dots in FunctionFQN)
	scope := NewFunctionScope("myapp")
	scope.Variables["result"] = []*VariableBinding{&VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:helper",
			Confidence: 1.0,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 3, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify result was resolved with module path prepended
	resultBinding := engine.GetScope("myapp").Variables["result"][0]
	assert.Equal(t, "builtins.str", resultBinding.Type.TypeFQN)
	assert.Equal(t, "myapp.helper", resultBinding.AssignedFrom)
}

// TestTypeInferenceEngine_UpdateVariableBindings_UnresolvedCall tests unresolved function calls.
func TestTypeInferenceEngine_UpdateVariableBindings_UnresolvedCall(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Create scope with call that has no return type
	scope := NewFunctionScope("myapp.controllers.view")
	scope.Variables["unknown"] = []*VariableBinding{&VariableBinding{
		VarName: "unknown",
		Type: &core.TypeInfo{
			TypeFQN:    "call:unknown_func",
			Confidence: 0.5,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify unknown remains as call: placeholder (not resolved)
	unknownBinding := engine.GetScope("myapp.controllers.view").Variables["unknown"][0]
	assert.Equal(t, "call:unknown_func", unknownBinding.Type.TypeFQN)
	assert.Equal(t, "assignment", unknownBinding.Type.Source)
}

// TestTypeInferenceEngine_UpdateVariableBindings_NilType tests handling of nil types.
func TestTypeInferenceEngine_UpdateVariableBindings_NilType(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Create scope with nil type (edge case)
	scope := NewFunctionScope("myapp.test")
	scope.Variables["nullvar"] = []*VariableBinding{&VariableBinding{
		VarName:  "nullvar",
		Type:     nil,
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	engine.AddScope(scope)

	// Should not panic
	assert.NotPanics(t, func() {
		engine.UpdateVariableBindingsWithFunctionReturns()
	})

	// Verify variable still has nil type
	nullvarBinding := engine.GetScope("myapp.test").Variables["nullvar"][0]
	assert.Nil(t, nullvarBinding.Type)
}

// TestTypeInferenceEngine_UpdateVariableBindings_InstanceMethod tests instance method call resolution.
// This tests the fix for receiver.method() patterns where receiver is a variable with a concrete type.
func TestTypeInferenceEngine_UpdateVariableBindings_InstanceMethod(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return type for UserManager.create_user method
	engine.ReturnTypes["myapp.models.UserManager.create_user"] = &core.TypeInfo{
		TypeFQN:    "myapp.models.User",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	// Create scope with:
	// 1. manager variable with concrete type (UserManager)
	// 2. user variable with placeholder "call:manager.create_user"
	scope := NewFunctionScope("myapp.controllers.signup")

	// manager = UserManager() - already resolved
	scope.Variables["manager"] = []*VariableBinding{&VariableBinding{
		VarName: "manager",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.UserManager",
			Confidence: 0.8,
			Source:     "class_instantiation",
		},
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	// user = manager.create_user() - placeholder
	scope.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "call:manager.create_user",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify user was resolved
	userBinding := engine.GetScope("myapp.controllers.signup").Variables["user"][0]
	assert.Equal(t, "myapp.models.User", userBinding.Type.TypeFQN)
	assert.Equal(t, "function_call_propagation", userBinding.Type.Source)
	assert.Equal(t, "myapp.models.UserManager.create_user", userBinding.AssignedFrom)

	// Verify manager was NOT changed (not a placeholder)
	managerBinding := engine.GetScope("myapp.controllers.signup").Variables["manager"][0]
	assert.Equal(t, "myapp.models.UserManager", managerBinding.Type.TypeFQN)
	assert.Equal(t, "class_instantiation", managerBinding.Type.Source)
}

// TestTypeInferenceEngine_UpdateVariableBindings_ChainedInstanceMethods tests chained instance methods.
// Tests: user = User(); profile = user.get_profile(); name = profile.get_name().
func TestTypeInferenceEngine_UpdateVariableBindings_ChainedInstanceMethods(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return types
	engine.ReturnTypes["myapp.models.User.get_profile"] = &core.TypeInfo{
		TypeFQN:    "myapp.models.Profile",
		Confidence: 1.0,
		Source:     "return_literal",
	}
	engine.ReturnTypes["myapp.models.Profile.get_name"] = &core.TypeInfo{
		TypeFQN:    "builtins.str",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	// Create scope with chain
	scope := NewFunctionScope("myapp.test.test_chain")

	// user = User() - concrete type
	scope.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 0.9,
			Source:     "class_instantiation",
		},
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	// profile = user.get_profile() - placeholder
	scope.Variables["profile"] = []*VariableBinding{&VariableBinding{
		VarName: "profile",
		Type: &core.TypeInfo{
			TypeFQN:    "call:user.get_profile",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}

	// name = profile.get_name() - placeholder (will be resolved in second pass)
	scope.Variables["name"] = []*VariableBinding{&VariableBinding{
		VarName: "name",
		Type: &core.TypeInfo{
			TypeFQN:    "call:profile.get_name",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 15, Column: 5},
	}}

	engine.AddScope(scope)

	// Run update - may need multiple passes depending on iteration order
	// Go map iteration is randomized, so profile may be resolved before or after name is processed
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify profile was resolved
	profileBinding := engine.GetScope("myapp.test.test_chain").Variables["profile"][0]
	assert.Equal(t, "myapp.models.Profile", profileBinding.Type.TypeFQN)
	assert.Equal(t, "myapp.models.User.get_profile", profileBinding.AssignedFrom)

	// name may or may not be resolved in first pass (depends on iteration order)
	// Run second pass to ensure all transitive dependencies are resolved
	engine.UpdateVariableBindingsWithFunctionReturns()

	// After second pass, name should definitely be resolved
	nameBinding := engine.GetScope("myapp.test.test_chain").Variables["name"][0]
	assert.Equal(t, "builtins.str", nameBinding.Type.TypeFQN)
	assert.Equal(t, "myapp.models.Profile.get_name", nameBinding.AssignedFrom)
}

// TestTypeInferenceEngine_UpdateVariableBindings_InstanceMethodVsModuleFunction tests distinguishing
// between instance.method() and module.function() patterns.
func TestTypeInferenceEngine_UpdateVariableBindings_InstanceMethodVsModuleFunction(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return types for both patterns
	engine.ReturnTypes["myapp.models.User.save"] = &core.TypeInfo{
		TypeFQN:    "builtins.bool",
		Confidence: 1.0,
		Source:     "return_literal",
	}
	engine.ReturnTypes["logging.getLogger"] = &core.TypeInfo{
		TypeFQN:    "logging.Logger",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	scope := NewFunctionScope("myapp.test")

	// user variable with concrete type
	scope.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 0.9,
			Source:     "class_instantiation",
		},
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	// result = user.save() - instance method (user is a variable)
	scope.Variables["result"] = []*VariableBinding{&VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:user.save",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}

	// logger = logging.getLogger() - module function (logging is NOT a variable)
	scope.Variables["logger"] = []*VariableBinding{&VariableBinding{
		VarName: "logger",
		Type: &core.TypeInfo{
			TypeFQN:    "call:logging.getLogger",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 15, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify result resolved via instance method path
	resultBinding := engine.GetScope("myapp.test").Variables["result"][0]
	assert.Equal(t, "builtins.bool", resultBinding.Type.TypeFQN)
	assert.Equal(t, "myapp.models.User.save", resultBinding.AssignedFrom)

	// Verify logger resolved via module function path
	loggerBinding := engine.GetScope("myapp.test").Variables["logger"][0]
	assert.Equal(t, "logging.Logger", loggerBinding.Type.TypeFQN)
	assert.Equal(t, "logging.getLogger", loggerBinding.AssignedFrom)
}

// TestTypeInferenceEngine_UpdateVariableBindings_UnresolvedReceiverType tests when receiver type is unresolved.
// When receiver has placeholder type, the call should be skipped (not resolved).
func TestTypeInferenceEngine_UpdateVariableBindings_UnresolvedReceiverType(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return type for method
	engine.ReturnTypes["myapp.models.User.get_profile"] = &core.TypeInfo{
		TypeFQN:    "myapp.models.Profile",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	scope := NewFunctionScope("myapp.test")

	// user variable with UNRESOLVED type (still a placeholder)
	scope.Variables["user"] = []*VariableBinding{&VariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "call:get_user", // Still a placeholder!
			Confidence: 0.5,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	// profile = user.get_profile() - placeholder depending on unresolved receiver
	scope.Variables["profile"] = []*VariableBinding{&VariableBinding{
		VarName: "profile",
		Type: &core.TypeInfo{
			TypeFQN:    "call:user.get_profile",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify profile was NOT resolved (receiver type is unresolved)
	profileBinding := engine.GetScope("myapp.test").Variables["profile"][0]
	assert.Equal(t, "call:user.get_profile", profileBinding.Type.TypeFQN)
	assert.Equal(t, "assignment", profileBinding.Type.Source)
}

// TestTypeInferenceEngine_UpdateVariableBindings_NestedMethod tests deeply nested method calls.
// Tests: a.b.c.method() pattern (multiple dots).
func TestTypeInferenceEngine_UpdateVariableBindings_NestedMethod(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up return type - note: we use SplitN(funcName, ".", 2)
	// So "a.b.c.method" becomes receiver="a", method="b.c.method"
	// This will look up a's type and append ".b.c.method"
	engine.ReturnTypes["myapp.models.Container.b.c.method"] = &core.TypeInfo{
		TypeFQN:    "builtins.str",
		Confidence: 1.0,
		Source:     "return_literal",
	}

	scope := NewFunctionScope("myapp.test")

	// a variable with concrete type
	scope.Variables["a"] = []*VariableBinding{&VariableBinding{
		VarName: "a",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.Container",
			Confidence: 0.9,
			Source:     "class_instantiation",
		},
		Location: Location{File: "/test/file.py", Line: 5, Column: 5},
	}}

	// result = a.b.c.method() - nested attribute access
	scope.Variables["result"] = []*VariableBinding{&VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:a.b.c.method",
			Confidence: 0.8,
			Source:     "assignment",
		},
		Location: Location{File: "/test/file.py", Line: 10, Column: 5},
	}}

	engine.AddScope(scope)
	engine.UpdateVariableBindingsWithFunctionReturns()

	// Verify result was resolved
	resultBinding := engine.GetScope("myapp.test").Variables["result"][0]
	assert.Equal(t, "builtins.str", resultBinding.Type.TypeFQN)
	assert.Equal(t, "myapp.models.Container.b.c.method", resultBinding.AssignedFrom)
}

// TestGetModuleVariableType tests retrieving type information for module-level variables.
func TestGetModuleVariableType(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up a module-level scope with typed variables
	scope := NewFunctionScope("main")
	scope.Variables["x"] = []*VariableBinding{&VariableBinding{
		VarName: "x",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "literal",
		},
		Location: Location{File: "/test/main.py", Line: 3, Column: 1},
	}}
	scope.Variables["name"] = []*VariableBinding{&VariableBinding{
		VarName: "name",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
		Location: Location{File: "/test/main.py", Line: 4, Column: 1},
	}}
	scope.Variables["calc"] = []*VariableBinding{&VariableBinding{
		VarName: "calc",
		Type: &core.TypeInfo{
			TypeFQN:    "helpers.Calculator",
			Confidence: 0.95,
			Source:     "class_instantiation",
		},
		Location: Location{File: "/test/main.py", Line: 10, Column: 1},
	}}
	engine.AddScope(scope)

	tests := []struct {
		name           string
		modulePath     string
		varName        string
		expectNil      bool
		expectedType   string
		expectedConf   float64
		expectedSource string
	}{
		{
			name:           "literal int variable",
			modulePath:     "main",
			varName:        "x",
			expectNil:      false,
			expectedType:   "builtins.int",
			expectedConf:   1.0,
			expectedSource: "literal",
		},
		{
			name:           "literal str variable",
			modulePath:     "main",
			varName:        "name",
			expectNil:      false,
			expectedType:   "builtins.str",
			expectedConf:   1.0,
			expectedSource: "literal",
		},
		{
			name:           "class instantiation variable",
			modulePath:     "main",
			varName:        "calc",
			expectNil:      false,
			expectedType:   "helpers.Calculator",
			expectedConf:   0.95, // float32 → float64 conversion: use InDelta
			expectedSource: "class_instantiation",
		},
		{
			name:       "non-existent variable",
			modulePath: "main",
			varName:    "nonexistent",
			expectNil:  true,
		},
		{
			name:       "non-existent module",
			modulePath: "nonexistent",
			varName:    "x",
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.GetModuleVariableType(tt.modulePath, tt.varName, 0)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedType, result.TypeFQN)
				assert.InDelta(t, tt.expectedConf, result.Confidence, 0.001)
				assert.Equal(t, tt.expectedSource, result.Source)
			}
		})
	}
}

// TestGetModuleVariableType_NilAndPlaceholder tests edge cases for GetModuleVariableType.
func TestGetModuleVariableType_NilAndPlaceholder(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	scope := NewFunctionScope("mymod")

	// Variable with nil type
	scope.Variables["nilvar"] = []*VariableBinding{&VariableBinding{
		VarName:  "nilvar",
		Type:     nil,
		Location: Location{File: "/test/mymod.py", Line: 1, Column: 1},
	}}

	// Variable with unresolved call: placeholder
	scope.Variables["unresolved"] = []*VariableBinding{&VariableBinding{
		VarName: "unresolved",
		Type: &core.TypeInfo{
			TypeFQN:    "call:some_func",
			Confidence: 0.5,
			Source:     "function_call_placeholder",
		},
		Location: Location{File: "/test/mymod.py", Line: 2, Column: 1},
	}}

	// Nil binding
	scope.Variables["nilbinding"] = []*VariableBinding{nil}

	engine.AddScope(scope)

	// nil type should return nil
	assert.Nil(t, engine.GetModuleVariableType("mymod", "nilvar", 0))

	// call: placeholder should return nil (unresolved)
	assert.Nil(t, engine.GetModuleVariableType("mymod", "unresolved", 0))

	// nil binding should return nil
	assert.Nil(t, engine.GetModuleVariableType("mymod", "nilbinding", 0))
}

// TestGetModuleVariableType_ConfidenceConversion tests float32 to float64 confidence conversion.
func TestGetModuleVariableType_ConfidenceConversion(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	scope := NewFunctionScope("testmod")
	scope.Variables["pi"] = []*VariableBinding{&VariableBinding{
		VarName: "pi",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.float",
			Confidence: 0.95,
			Source:     "literal",
		},
		Location: Location{File: "/test/testmod.py", Line: 1, Column: 1},
	}}
	engine.AddScope(scope)

	result := engine.GetModuleVariableType("testmod", "pi", 0)
	assert.NotNil(t, result)
	assert.Equal(t, "builtins.float", result.TypeFQN)
	// float32(0.95) converted to float64 should be close to 0.95
	assert.InDelta(t, 0.95, result.Confidence, 0.001)
}

// TestGetModuleVariableType_ImplementsInterface tests that TypeInferenceEngine satisfies ModuleVariableProvider.
func TestGetModuleVariableType_ImplementsInterface(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Verify TypeInferenceEngine satisfies ModuleVariableProvider interface
	var provider core.ModuleVariableProvider = engine
	assert.NotNil(t, provider)

	// Should return nil for empty engine
	result := provider.GetModuleVariableType("any", "any", 0)
	assert.Nil(t, result)
}

// TestAddImportMap_GetImportMap tests ImportMap storage and retrieval (P0 fix).
func TestAddImportMap_GetImportMap(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(moduleRegistry)

	// Create test ImportMaps
	importMap1 := core.NewImportMap("/test/file1.py")
	importMap1.AddImport("Controller", "controller.Controller")
	importMap1.AddImport("PaymentService", "payment.PaymentService")

	importMap2 := core.NewImportMap("/test/file2.py")
	importMap2.AddImport("DataAdapter", "adapters.DataAdapter")

	// Test adding ImportMaps
	engine.AddImportMap("/test/file1.py", importMap1)
	engine.AddImportMap("/test/file2.py", importMap2)

	// Test retrieving ImportMaps
	retrieved1 := engine.GetImportMap("/test/file1.py")
	assert.NotNil(t, retrieved1)
	assert.Equal(t, importMap1, retrieved1)

	retrieved2 := engine.GetImportMap("/test/file2.py")
	assert.NotNil(t, retrieved2)
	assert.Equal(t, importMap2, retrieved2)

	// Test retrieving non-existent ImportMap
	retrieved3 := engine.GetImportMap("/test/nonexistent.py")
	assert.Nil(t, retrieved3)

	// Test adding nil ImportMap (should be ignored)
	engine.AddImportMap("/test/file3.py", nil)
	retrieved4 := engine.GetImportMap("/test/file3.py")
	assert.Nil(t, retrieved4)

	// Test adding ImportMap with empty path (should be ignored)
	engine.AddImportMap("", importMap1)
	retrieved5 := engine.GetImportMap("")
	assert.Nil(t, retrieved5)
}

// TestAddImportMap_ThreadSafety tests concurrent access to ImportMaps.
func TestAddImportMap_ThreadSafety(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(moduleRegistry)

	// Add ImportMaps concurrently
	done := make(chan bool, 10)
	for i := range 10 {
		go func(idx int) {
			filePath := "/test/file" + string(rune(idx+'0')) + ".py"
			importMap := core.NewImportMap(filePath)
			importMap.AddImport("TestClass", "module.TestClass")
			engine.AddImportMap(filePath, importMap)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Verify all ImportMaps were added
	for i := range 10 {
		filePath := "/test/file" + string(rune(i+'0')) + ".py"
		importMap := engine.GetImportMap(filePath)
		assert.NotNil(t, importMap, "ImportMap for %s should exist", filePath)
	}
}

// TestGetModuleVariableType_Reassignment tests that two bindings for the same variable
// return the correct type based on line number.
func TestGetModuleVariableType_Reassignment(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)
	scope := NewFunctionScope("main")
	// val = "" at line 15
	scope.Variables["val"] = append(scope.Variables["val"], &VariableBinding{
		VarName:  "val",
		Type:     &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 1.0, Source: "literal"},
		Location: Location{File: "/test/main.py", Line: 15, Column: 1},
	})
	// val = 5 at line 16
	scope.Variables["val"] = append(scope.Variables["val"], &VariableBinding{
		VarName:  "val",
		Type:     &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "literal"},
		Location: Location{File: "/test/main.py", Line: 16, Column: 1},
	})
	engine.AddScope(scope)

	// Line 15 should return str
	result15 := engine.GetModuleVariableType("main", "val", 15)
	assert.NotNil(t, result15)
	assert.Equal(t, "builtins.str", result15.TypeFQN)

	// Line 16 should return int
	result16 := engine.GetModuleVariableType("main", "val", 16)
	assert.NotNil(t, result16)
	assert.Equal(t, "builtins.int", result16.TypeFQN)

	// Line 0 should return last binding (int)
	result0 := engine.GetModuleVariableType("main", "val", 0)
	assert.NotNil(t, result0)
	assert.Equal(t, "builtins.int", result0.TypeFQN)
}

// TestGetModuleVariableType_VarPlaceholder verifies that var: prefix is filtered out.
func TestGetModuleVariableType_VarPlaceholder(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)
	scope := NewFunctionScope("mymod")
	scope.Variables["leaked"] = []*VariableBinding{{
		VarName:  "leaked",
		Type:     &core.TypeInfo{TypeFQN: "var:result", Confidence: 0.2, Source: "return_variable"},
		Location: Location{File: "/test/mymod.py", Line: 5, Column: 1},
	}}
	engine.AddScope(scope)
	assert.Nil(t, engine.GetModuleVariableType("mymod", "leaked", 5))
	assert.Nil(t, engine.GetModuleVariableType("mymod", "leaked", 0))
}

// TestResolveReturnVariableReferences verifies that var: placeholders in return types
// are resolved to concrete types from variable bindings.
func TestResolveReturnVariableReferences(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up function with var:result return type
	engine.ReturnTypes["helpers.Calculator.add"] = &core.TypeInfo{
		TypeFQN: "var:result", Confidence: 0.2, Source: "return_variable",
	}

	// Set up scope with result variable having concrete type
	scope := NewFunctionScope("helpers.Calculator.add")
	scope.Variables["result"] = []*VariableBinding{{
		VarName:  "result",
		Type:     &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 1.0, Source: "literal"},
		Location: Location{File: "/test/helpers.py", Line: 5, Column: 1},
	}}
	engine.AddScope(scope)

	// Resolve
	engine.ResolveReturnVariableReferences()

	// Return type should now be builtins.int
	rt, ok := engine.GetReturnType("helpers.Calculator.add")
	assert.True(t, ok)
	assert.Equal(t, "builtins.int", rt.TypeFQN)
	assert.Equal(t, "return_variable_resolved", rt.Source)
	assert.InDelta(t, 0.2, rt.Confidence, 0.01) // 0.2 * 1.0
}

// TestResolveReturnVariableReferences_Unresolved verifies that var: stays unresolved
// when no matching variable binding exists.
func TestResolveReturnVariableReferences_Unresolved(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	// Set up function with var:unknown return type
	engine.ReturnTypes["mymod.foo"] = &core.TypeInfo{
		TypeFQN: "var:unknown", Confidence: 0.2, Source: "return_variable",
	}

	// Scope exists but has no "unknown" variable
	scope := NewFunctionScope("mymod.foo")
	engine.AddScope(scope)

	engine.ResolveReturnVariableReferences()

	// Should remain unresolved
	rt, ok := engine.GetReturnType("mymod.foo")
	assert.True(t, ok)
	assert.Equal(t, "var:unknown", rt.TypeFQN)
}
