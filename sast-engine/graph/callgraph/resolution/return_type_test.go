package resolution

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractReturnTypes_Literals(t *testing.T) {
	sourceCode := []byte(`
def get_string():
    return "hello"

def get_number():
    return 42

def get_list():
    return [1, 2, 3]

def get_dict():
    return {"key": "value"}

def get_none():
    return None
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Len(t, returns, 5)

	expectedTypes := map[string]string{
		"test.get_string": "builtins.str",
		"test.get_number": "builtins.int",
		"test.get_list":   "builtins.list",
		"test.get_dict":   "builtins.dict",
		"test.get_none":   "builtins.NoneType",
	}

	for _, ret := range returns {
		expectedType, ok := expectedTypes[ret.FunctionFQN]
		require.True(t, ok, "Unexpected function: %s", ret.FunctionFQN)
		assert.Equal(t, expectedType, ret.ReturnType.TypeFQN)
		assert.Equal(t, float32(1.0), ret.ReturnType.Confidence)
		assert.Equal(t, "return_literal", ret.ReturnType.Source)
	}
}

func TestExtractReturnTypes_EmptyFunction(t *testing.T) {
	sourceCode := []byte(`
def empty_func():
    pass
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Empty(t, returns, "Functions with no explicit return should not generate return types")
}

func TestExtractReturnTypes_MultipleReturns(t *testing.T) {
	sourceCode := []byte(`
def maybe_string(flag):
    if flag:
        return "yes"
    else:
        return "no"
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Len(t, returns, 2, "Should capture both return statements")

	for _, ret := range returns {
		assert.Equal(t, "test.maybe_string", ret.FunctionFQN)
		assert.Equal(t, "builtins.str", ret.ReturnType.TypeFQN)
	}
}

func TestExtractReturnTypes_NestedFunctions(t *testing.T) {
	sourceCode := []byte(`
def outer():
    def inner():
        return 123
    return "outer"
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Len(t, returns, 2)

	types := make(map[string]string)
	for _, ret := range returns {
		types[ret.FunctionFQN] = ret.ReturnType.TypeFQN
	}

	// Nested function gets FQN test.outer.inner
	assert.Equal(t, "builtins.int", types["test.outer.inner"])
	assert.Equal(t, "builtins.str", types["test.outer"])
}

func TestExtractReturnTypes_FunctionCall(t *testing.T) {
	sourceCode := []byte(`
def create_user():
    return User()

def get_value():
    return str(42)
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Len(t, returns, 2)

	types := make(map[string]*core.TypeInfo)
	for _, ret := range returns {
		types[ret.FunctionFQN] = ret.ReturnType
	}

	// User() is now detected as class instantiation (Task 7)
	assert.Equal(t, "test.User", types["test.create_user"].TypeFQN)
	assert.Greater(t, types["test.create_user"].Confidence, float32(0.5))
	assert.Less(t, types["test.create_user"].Confidence, float32(1.0))

	// str() is a builtin constructor
	assert.Equal(t, "builtins.str", types["test.get_value"].TypeFQN)
	assert.Equal(t, float32(0.9), types["test.get_value"].Confidence)
}

func TestExtractReturnTypes_ReturnVariable(t *testing.T) {
	sourceCode := []byte(`
def process():
    result = "done"
    return result
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	require.Len(t, returns, 1)

	assert.Equal(t, "var:result", returns[0].ReturnType.TypeFQN)
	assert.Equal(t, "return_variable", returns[0].ReturnType.Source)
}

func TestExtractReturnTypes_AllLiteralTypes(t *testing.T) {
	sourceCode := []byte(`
def get_float():
    return 3.14

def get_bool_true():
    return True

def get_bool_false():
    return False

def get_set():
    return {1, 2, 3}

def get_tuple():
    return (1, 2, 3)
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)

	assert.Len(t, returns, 5)

	expectedTypes := map[string]string{
		"test.get_float":      "builtins.float",
		"test.get_bool_true":  "builtins.bool",
		"test.get_bool_false": "builtins.bool",
		"test.get_set":        "builtins.set",
		"test.get_tuple":      "builtins.tuple",
	}

	for _, ret := range returns {
		expectedType, ok := expectedTypes[ret.FunctionFQN]
		require.True(t, ok, "Unexpected function: %s", ret.FunctionFQN)
		assert.Equal(t, expectedType, ret.ReturnType.TypeFQN)
		assert.Equal(t, float32(1.0), ret.ReturnType.Confidence)
	}
}

func TestMergeReturnTypes_SingleReturn(t *testing.T) {
	statements := []*ReturnStatement{
		{
			FunctionFQN: "test.func1",
			ReturnType: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "return_literal",
			},
		},
	}

	merged := MergeReturnTypes(statements)
	assert.Len(t, merged, 1)
	assert.Equal(t, "builtins.str", merged["test.func1"].TypeFQN)
}

func TestMergeReturnTypes_MultipleReturnsHighestConfidence(t *testing.T) {
	statements := []*ReturnStatement{
		{
			FunctionFQN: "test.func1",
			ReturnType: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "return_literal",
			},
		},
		{
			FunctionFQN: "test.func1",
			ReturnType: &core.TypeInfo{
				TypeFQN:    "call:unknown",
				Confidence: 0.3,
				Source:     "return_function_call",
			},
		},
	}

	merged := MergeReturnTypes(statements)
	assert.Len(t, merged, 1)
	// Should take the higher confidence return
	assert.Equal(t, "builtins.str", merged["test.func1"].TypeFQN)
	assert.Equal(t, float32(1.0), merged["test.func1"].Confidence)
}

func TestMergeReturnTypes_DifferentFunctions(t *testing.T) {
	statements := []*ReturnStatement{
		{
			FunctionFQN: "test.func1",
			ReturnType: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "return_literal",
			},
		},
		{
			FunctionFQN: "test.func2",
			ReturnType: &core.TypeInfo{
				TypeFQN:    "builtins.int",
				Confidence: 1.0,
				Source:     "return_literal",
			},
		},
	}

	merged := MergeReturnTypes(statements)
	assert.Len(t, merged, 2)
	assert.Equal(t, "builtins.str", merged["test.func1"].TypeFQN)
	assert.Equal(t, "builtins.int", merged["test.func2"].TypeFQN)
}

func TestTypeInferenceEngine_AddReturnTypes(t *testing.T) {
	modRegistry := core.NewModuleRegistry()
	engine := NewTypeInferenceEngine(modRegistry)

	returnTypes := map[string]*core.TypeInfo{
		"test.func1": {
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "return_literal",
		},
		"test.func2": {
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "return_literal",
		},
	}

	engine.AddReturnTypesToEngine(returnTypes)

	assert.Len(t, engine.ReturnTypes, 2)
	assert.Equal(t, "builtins.str", engine.ReturnTypes["test.func1"].TypeFQN)
	assert.Equal(t, "builtins.int", engine.ReturnTypes["test.func2"].TypeFQN)
}

func TestExtractReturnTypes_Location(t *testing.T) {
	sourceCode := []byte(`
def func():
    return "test"
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	require.Len(t, returns, 1)

	assert.Equal(t, "test.py", returns[0].Location.File)
	assert.Equal(t, uint32(3), returns[0].Location.Line) // Line 3 (1-indexed)
	assert.Greater(t, returns[0].Location.Column, uint32(0))
}
