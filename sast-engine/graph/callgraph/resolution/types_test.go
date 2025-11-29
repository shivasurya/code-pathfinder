package resolution

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestNewFunctionScope(t *testing.T) {
	scope := NewFunctionScope("myapp.utils.helper")

	assert.NotNil(t, scope)
	assert.Equal(t, "myapp.utils.helper", scope.FunctionFQN)
	assert.NotNil(t, scope.Variables)
	assert.Equal(t, 0, len(scope.Variables))
	assert.Nil(t, scope.ReturnType)
}

// Duplicate test removed - same test exists in inference_test.go

func TestFunctionScope_AddVariable_Nil(t *testing.T) {
	scope := NewFunctionScope("test.func")

	// Add nil binding
	scope.AddVariable(nil)
	assert.Equal(t, 0, len(scope.Variables))

	// Add binding with empty name
	scope.AddVariable(&VariableBinding{VarName: ""})
	assert.Equal(t, 0, len(scope.Variables))
}

func TestFunctionScope_GetVariable(t *testing.T) {
	scope := NewFunctionScope("test.func")

	// Get non-existent variable
	result := scope.GetVariable("x")
	assert.Nil(t, result)

	// Add and get variable
	binding := &VariableBinding{
		VarName: "x",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
		},
	}
	scope.AddVariable(binding)

	result = scope.GetVariable("x")
	assert.NotNil(t, result)
	assert.Equal(t, binding, result)
	assert.Equal(t, "x", result.VarName)
	assert.Equal(t, "builtins.int", result.Type.TypeFQN)
}

func TestFunctionScope_HasVariable(t *testing.T) {
	scope := NewFunctionScope("test.func")

	// Check non-existent variable
	assert.False(t, scope.HasVariable("x"))

	// Add variable
	binding := &VariableBinding{
		VarName: "x",
		Type:    &core.TypeInfo{TypeFQN: "builtins.int"},
	}
	scope.AddVariable(binding)

	// Check existing variable
	assert.True(t, scope.HasVariable("x"))
	assert.False(t, scope.HasVariable("y"))
}

func TestVariableBinding(t *testing.T) {
	binding := &VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.models.User",
			Confidence: 0.8,
			Source:     "return_type",
		},
		AssignedFrom: "myapp.services.get_user",
		Location: Location{
			File:      "myapp/views.py",
			Line:      42,
			Column:    8,
			StartByte: 1024,
			EndByte:   1050,
		},
	}

	assert.Equal(t, "result", binding.VarName)
	assert.NotNil(t, binding.Type)
	assert.Equal(t, "myapp.models.User", binding.Type.TypeFQN)
	assert.Equal(t, float32(0.8), binding.Type.Confidence)
	assert.Equal(t, "return_type", binding.Type.Source)
	assert.Equal(t, "myapp.services.get_user", binding.AssignedFrom)
	assert.Equal(t, "myapp/views.py", binding.Location.File)
	assert.Equal(t, uint32(42), binding.Location.Line)
}

func TestLocation(t *testing.T) {
	loc := Location{
		File:      "test.py",
		Line:      100,
		Column:    20,
		StartByte: 5000,
		EndByte:   5100,
	}

	assert.Equal(t, "test.py", loc.File)
	assert.Equal(t, uint32(100), loc.Line)
	assert.Equal(t, uint32(20), loc.Column)
	assert.Equal(t, uint32(5000), loc.StartByte)
	assert.Equal(t, uint32(5100), loc.EndByte)
}

func TestFunctionScope_UpdateVariable(t *testing.T) {
	scope := NewFunctionScope("test.func")

	// Add initial binding
	binding1 := &VariableBinding{
		VarName: "x",
		Type:    &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 0.5},
	}
	scope.AddVariable(binding1)

	// Update with new binding
	binding2 := &VariableBinding{
		VarName: "x",
		Type:    &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 1.0},
	}
	scope.AddVariable(binding2)

	// Should have only one variable with updated type
	assert.Equal(t, 1, len(scope.Variables))
	result := scope.GetVariable("x")
	assert.Equal(t, "builtins.str", result.Type.TypeFQN)
	assert.Equal(t, float32(1.0), result.Type.Confidence)
}
