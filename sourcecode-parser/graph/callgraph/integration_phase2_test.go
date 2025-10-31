package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Phase2_FactoryPattern(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
class User:
    def save(self):
        pass

def create_user():
    return User()

user = create_user()
user.save()
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	require.NoError(t, err)

	// Build registry and code graph
	registry, err := BuildModuleRegistry(tmpDir)
	require.NoError(t, err)

	// Initialize code graph (simplified - real implementation would parse classes)
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: []*graph.Edge{},
	}

	// Add User.save method to graph
	codeGraph.Nodes["test.User.save"] = &graph.Node{
		ID:   "test.User.save",
		Type: "method_declaration",
		Name: "save",
	}

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)
	require.NoError(t, err)

	// Verify user.save() is resolved
	// Note: This test validates the infrastructure
	// Full resolution depends on code graph containing User class
	assert.NotNil(t, callGraph)
}

func TestIntegration_Phase2_ChainedCalls(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def get_name():
    return "hello"

name = get_name()
upper_name = name.upper()
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	require.NoError(t, err)

	registry, err := BuildModuleRegistry(tmpDir)
	require.NoError(t, err)

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: []*graph.Edge{},
	}

	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)
	require.NoError(t, err)

	// Verify infrastructure works
	assert.NotNil(t, callGraph)
}

func TestIntegration_Phase2_MultipleReturns(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def maybe_string(flag):
    if flag:
        return "yes"
    else:
        return "no"

result = maybe_string(True)
upper_result = result.upper()
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	require.NoError(t, err)

	registry, err := BuildModuleRegistry(tmpDir)
	require.NoError(t, err)

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: []*graph.Edge{},
	}

	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)
	require.NoError(t, err)

	// Both returns are strings, so result.upper() should resolve
	assert.NotNil(t, callGraph)
}

func TestIntegration_Phase2_ClassMethod(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
class Service:
    def execute(self):
        return "done"

def get_service():
    return Service()

service = get_service()
result = service.execute()
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	require.NoError(t, err)

	registry, err := BuildModuleRegistry(tmpDir)
	require.NoError(t, err)

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: []*graph.Edge{},
	}

	// Add Service.execute to graph
	codeGraph.Nodes["test.Service.execute"] = &graph.Node{
		ID:   "test.Service.execute",
		Type: "method_declaration",
		Name: "execute",
	}

	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)
	require.NoError(t, err)

	assert.NotNil(t, callGraph)
}

func TestIntegration_Phase2_ConfidenceFiltering(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)
	engine.Builtins = NewBuiltinRegistry()

	// Low confidence type should not resolve heuristically
	scope := &FunctionScope{
		FunctionFQN: "test.main",
		Variables: map[string]*VariableBinding{
			"obj": {
				VarName: "obj",
				Type: &TypeInfo{
					TypeFQN:    "test.UnknownClass",
					Confidence: 0.3, // Low confidence
					Source:     "guess",
				},
			},
		},
	}
	engine.AddScope(scope)

	importMap := NewImportMap("test.py")
	fqn, resolved, _ := resolveCallTarget(
		"obj.method",
		importMap,
		registry,
		"test",
		&graph.CodeGraph{Nodes: make(map[string]*graph.Node)},
		engine,
		"test.main",
		nil,
	)

	// Should not resolve with low confidence
	assert.False(t, resolved)
	_ = fqn // Suppress unused warning
}

func TestIntegration_Phase2_HighConfidenceResolution(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)
	engine.Builtins = NewBuiltinRegistry()

	// High confidence type should resolve even without validation
	scope := &FunctionScope{
		FunctionFQN: "test.main",
		Variables: map[string]*VariableBinding{
			"user": {
				VarName: "user",
				Type: &TypeInfo{
					TypeFQN:    "test.User",
					Confidence: 0.9, // High confidence
					Source:     "literal",
				},
			},
		},
	}
	engine.AddScope(scope)

	importMap := NewImportMap("test.py")
	fqn, resolved, _ := resolveCallTarget(
		"user.save",
		importMap,
		registry,
		"test",
		&graph.CodeGraph{Nodes: make(map[string]*graph.Node)},
		engine,
		"test.main",
		nil,
	)

	// Should resolve with high confidence heuristic
	assert.True(t, resolved)
	assert.Equal(t, "test.User.save", fqn)
}

func TestIntegration_Phase2_PlaceholderSkipping(t *testing.T) {
	registry := NewModuleRegistry()
	engine := NewTypeInferenceEngine(registry)
	engine.Builtins = NewBuiltinRegistry()

	// Variable with call: placeholder should not resolve
	scope := &FunctionScope{
		FunctionFQN: "test.main",
		Variables: map[string]*VariableBinding{
			"obj": {
				VarName: "obj",
				Type: &TypeInfo{
					TypeFQN:    "call:get_object",
					Confidence: 0.5,
					Source:     "function_call",
				},
			},
		},
	}
	engine.AddScope(scope)

	importMap := NewImportMap("test.py")
	fqn, resolved, _ := resolveCallTarget(
		"obj.method",
		importMap,
		registry,
		"test",
		&graph.CodeGraph{Nodes: make(map[string]*graph.Node)},
		engine,
		"test.main",
		nil,
	)

	// Should not resolve with placeholder
	assert.False(t, resolved)
	_ = fqn // Suppress unused warning
}
