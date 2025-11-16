package callgraph

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
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
			typeInfo := &TypeInfo{
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
		location     resolution.Location
	}{
		{
			name:       "simple variable",
			varName:    "user",
			typeFQN:    "myapp.models.User",
			confidence: 1.0,
			source:     "assignment",
			assignedFrom: "myapp.controllers.get_user",
			location: resolution.Location{
				File:   "/path/to/file.py",
				Line:   10,
				Column: 5,
			},
		},
		{
			name:       "builtin variable",
			varName:    "data",
			typeFQN:    "builtins.str",
			confidence: 1.0,
			source:     "literal",
			assignedFrom: "",
			location: resolution.Location{
				File:   "/path/to/file.py",
				Line:   20,
				Column: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo := &TypeInfo{
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
		Type: &TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "assignment",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}
	scope.Variables["user"] = binding1

	// Add second variable
	binding2 := &VariableBinding{
		VarName: "result",
		Type: &TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 0.9,
			Source:     "heuristic",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 15, Column: 5},
	}
	scope.Variables["result"] = binding2

	// Verify both variables exist
	assert.Equal(t, 2, len(scope.Variables))
	assert.Equal(t, "myapp.models.User", scope.Variables["user"].Type.TypeFQN)
	assert.Equal(t, "builtins.dict", scope.Variables["result"].Type.TypeFQN)

	// Update existing variable
	binding3 := &VariableBinding{
		VarName: "user",
		Type: &TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "annotation",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 20, Column: 5},
	}
	scope.Variables["user"] = binding3

	// Verify update
	assert.Equal(t, 2, len(scope.Variables))
	assert.Equal(t, "annotation", scope.Variables["user"].Type.Source)
}

// TestFunctionScope_ReturnType tests setting return type on a scope.
func TestFunctionScope_ReturnType(t *testing.T) {
	scope := NewFunctionScope("myapp.controllers.get_user")

	// Initially nil
	assert.Nil(t, scope.ReturnType)

	// Set return type
	returnType := &TypeInfo{
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
	registry := NewModuleRegistry()

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
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Initially no scopes
	assert.Nil(t, engine.GetScope("myapp.controllers.get_user"))

	// Add first scope
	scope1 := NewFunctionScope("myapp.controllers.get_user")
	scope1.Variables["user"] = &VariableBinding{
		VarName: "user",
		Type: &TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "assignment",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}
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
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add nil scope should not panic
	engine.AddScope(nil)

	// Verify no scopes added
	assert.Equal(t, 0, len(engine.Scopes))
}

// TestTypeInferenceEngine_UpdateScope tests updating an existing scope.
func TestTypeInferenceEngine_UpdateScope(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add initial scope
	scope1 := NewFunctionScope("myapp.controllers.get_user")
	scope1.Variables["user"] = &VariableBinding{
		VarName: "user",
		Type: &TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 0.8,
			Source:     "heuristic",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}
	engine.AddScope(scope1)

	// Update with new scope
	scope2 := NewFunctionScope("myapp.controllers.get_user")
	scope2.Variables["user"] = &VariableBinding{
		VarName: "user",
		Type: &TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 1.0,
			Source:     "annotation",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 10, Column: 5},
	}
	scope2.Variables["result"] = &VariableBinding{
		VarName: "result",
		Type: &TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "literal",
		},
		Location: resolution.Location{File: "/path/to/file.py", Line: 15, Column: 5},
	}
	engine.AddScope(scope2)

	// Verify updated scope
	retrieved := engine.GetScope("myapp.controllers.get_user")
	assert.NotNil(t, retrieved)
	assert.Equal(t, 2, len(retrieved.Variables))
	assert.Equal(t, float32(1.0), retrieved.Variables["user"].Type.Confidence)
	assert.Equal(t, "annotation", retrieved.Variables["user"].Type.Source)
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
			typeInfo := &TypeInfo{
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
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Add return type for a function
	returnType1 := &TypeInfo{
		TypeFQN:    "myapp.models.User",
		Confidence: 1.0,
		Source:     "annotation",
	}
	engine.ReturnTypes["myapp.controllers.get_user"] = returnType1

	// Add return type for another function
	returnType2 := &TypeInfo{
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
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)

	// Initially nil
	assert.Nil(t, engine.Builtins)

	// Set builtin registry
	engine.Builtins = NewBuiltinRegistry()
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
