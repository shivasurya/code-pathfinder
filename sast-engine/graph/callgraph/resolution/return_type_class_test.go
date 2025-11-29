package resolution

import (
	"context"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"PascalCase class", "User", true},
		{"Multi-word class", "UserProfile", true},
		{"camelCase", "userName", false},
		{"snake_case", "user_name", false},
		{"UPPER_CASE constant", "MAX_SIZE", false},
		{"lowercase", "user", false},
		{"Single char upper", "U", true},
		{"Single char lower", "u", false},
		{"Empty string", "", false},
		{"Acronym class", "HTTPServer", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveClassInstantiation_Simple(t *testing.T) {
	sourceCode := []byte(`User()`)

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	callNode := tree.RootNode().Child(0).Child(0) // expression_statement -> call

	registry := core.NewModuleRegistry()
	typeInfo := ResolveClassInstantiation(callNode, sourceCode, "test", nil, registry)

	require.NotNil(t, typeInfo)
	assert.Equal(t, "test.User", typeInfo.TypeFQN)
	assert.Greater(t, typeInfo.Confidence, float32(0.5))
	assert.Contains(t, typeInfo.Source, "class_instantiation")
}

func TestResolveClassInstantiation_WithModule(t *testing.T) {
	sourceCode := []byte(`models.User()`)

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	callNode := tree.RootNode().Child(0).Child(0)

	importMap := core.NewImportMap("test.py")
	importMap.AddImport("models", "myapp.models")

	typeInfo := ResolveClassInstantiation(callNode, sourceCode, "test", importMap, nil)

	require.NotNil(t, typeInfo)
	assert.Equal(t, "myapp.models.User", typeInfo.TypeFQN)
	assert.Greater(t, typeInfo.Confidence, float32(0.8))
}

func TestResolveClassInstantiation_NotAClass(t *testing.T) {
	sourceCode := []byte(`calculate_total()`)

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	callNode := tree.RootNode().Child(0).Child(0)

	typeInfo := ResolveClassInstantiation(callNode, sourceCode, "test", nil, nil)

	// Should return nil for non-PascalCase function names
	assert.Nil(t, typeInfo)
}

func TestExtractReturnTypes_ClassInstantiation(t *testing.T) {
	sourceCode := []byte(`
def create_user():
    return User()

def get_profile():
    return UserProfile()

def build_server():
    return HTTPServer()
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Len(t, returns, 3)

	expectedClasses := map[string]string{
		"test.create_user":  "test.User",
		"test.get_profile":  "test.UserProfile",
		"test.build_server": "test.HTTPServer",
	}

	for _, ret := range returns {
		expectedClass, ok := expectedClasses[ret.FunctionFQN]
		require.True(t, ok, "Unexpected function: %s", ret.FunctionFQN)
		assert.Equal(t, expectedClass, ret.ReturnType.TypeFQN)
		assert.Greater(t, ret.ReturnType.Confidence, float32(0.5))
	}
}

func TestExtractReturnTypes_MixedReturns(t *testing.T) {
	sourceCode := []byte(`
def maybe_user(flag):
    if flag:
        return User()
    else:
        return None
`)

	builtinRegistry := registry.NewBuiltinRegistry()
	returns, err := ExtractReturnTypes("test.py", sourceCode, "test", builtinRegistry)
	require.NoError(t, err)
	assert.Len(t, returns, 2)

	// Check that both returns were captured
	types := make(map[string]int)
	for _, ret := range returns {
		if ret.ReturnType.TypeFQN == "test.User" {
			types["User"]++
		} else if ret.ReturnType.TypeFQN == "builtins.NoneType" {
			types["None"]++
		}
	}
	assert.Equal(t, 1, types["User"], "Should capture User() return")
	assert.Equal(t, 1, types["None"], "Should capture None return")

	// Merge will pick NoneType (confidence 1.0) over User (confidence 0.6 without imports/registry)
	// This is expected behavior - without context, we're more confident about literals
	merged := MergeReturnTypes(returns)
	assert.NotNil(t, merged["test.maybe_user"])
	assert.Contains(t, []string{"test.User", "builtins.NoneType"}, merged["test.maybe_user"].TypeFQN)
}
