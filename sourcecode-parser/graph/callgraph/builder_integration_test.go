package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/stretchr/testify/assert"
)

func TestFindFunctionAtLine(t *testing.T) {
	sourceCode := []byte(`
def foo():
    x = 1
    return x

def bar():
    y = 2
    return y
`)

	tree, err := ParsePythonFile(sourceCode)
	assert.NoError(t, err)
	assert.NotNil(t, tree)
	defer tree.Close()

	// Test finding function at line 2 (foo)
	funcNode := findFunctionAtLine(tree.RootNode(), 2)
	assert.NotNil(t, funcNode)
	assert.Equal(t, "function_definition", funcNode.Type())

	// Test finding function at line 6 (bar)
	funcNode = findFunctionAtLine(tree.RootNode(), 6)
	assert.NotNil(t, funcNode)
	assert.Equal(t, "function_definition", funcNode.Type())

	// Test line with no function
	funcNode = findFunctionAtLine(tree.RootNode(), 3)
	assert.Nil(t, funcNode)

	// Test nil root
	funcNode = findFunctionAtLine(nil, 2)
	assert.Nil(t, funcNode)
}

func TestGenerateTaintSummaries_EmptyCallGraph(t *testing.T) {
	callGraph := NewCallGraph()
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}
	registry := &ModuleRegistry{
		Modules:      make(map[string]string),
		FileToModule: make(map[string]string),
	}

	// Should not crash with empty call graph
	generateTaintSummaries(callGraph, codeGraph, registry)

	assert.Equal(t, 0, len(callGraph.Summaries))
}

func TestGenerateTaintSummaries_Integration(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := `
def vulnerable():
    x = request.GET['input']
    eval(x)
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	assert.NoError(t, err)

	// Create mock registry
	registry := &ModuleRegistry{
		Modules: map[string]string{
			"test": testFile,
		},
		FileToModule: map[string]string{
			testFile: "test",
		},
	}

	// Create mock code graph with function node
	funcNode := &graph.Node{
		ID:         "test.vulnerable",
		Type:       "function_definition",
		Name:       "vulnerable",
		File:       testFile,
		LineNumber: 2,
	}

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			funcNode.ID: funcNode,
		},
		Edges: make([]*graph.Edge, 0),
	}

	// Create call graph and index the function
	callGraph := NewCallGraph()
	callGraph.Functions["test.vulnerable"] = funcNode

	// Generate taint summaries
	generateTaintSummaries(callGraph, codeGraph, registry)

	// Verify summary was created
	assert.Equal(t, 1, len(callGraph.Summaries))
	summary, exists := callGraph.Summaries["test.vulnerable"]
	assert.True(t, exists)
	assert.NotNil(t, summary)
	assert.Equal(t, "test.vulnerable", summary.FunctionFQN)
}

func TestGenerateTaintSummaries_FileReadError(t *testing.T) {
	// Create mock function with non-existent file
	funcNode := &graph.Node{
		ID:         "test.func",
		Type:       "function_definition",
		Name:       "func",
		File:       "/nonexistent/file.py",
		LineNumber: 1,
	}

	callGraph := NewCallGraph()
	callGraph.Functions["test.func"] = funcNode

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			funcNode.ID: funcNode,
		},
		Edges: make([]*graph.Edge, 0),
	}

	registry := &ModuleRegistry{
		Modules:      make(map[string]string),
		FileToModule: make(map[string]string),
	}

	// Should handle error gracefully and not crash
	generateTaintSummaries(callGraph, codeGraph, registry)

	// No summary should be created for failed file
	assert.Equal(t, 0, len(callGraph.Summaries))
}

func TestGenerateTaintSummaries_ParseError(t *testing.T) {
	// Create a temporary test file with invalid Python
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.py")

	invalidCode := `
def broken(
    # Missing closing paren - syntax error
`

	err := os.WriteFile(testFile, []byte(invalidCode), 0644)
	assert.NoError(t, err)

	funcNode := &graph.Node{
		ID:         "test.broken",
		Type:       "function_definition",
		Name:       "broken",
		File:       testFile,
		LineNumber: 2,
	}

	callGraph := NewCallGraph()
	callGraph.Functions["test.broken"] = funcNode

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			funcNode.ID: funcNode,
		},
		Edges: make([]*graph.Edge, 0),
	}

	registry := &ModuleRegistry{
		Modules:      make(map[string]string),
		FileToModule: make(map[string]string),
	}

	// Should handle parse error gracefully
	generateTaintSummaries(callGraph, codeGraph, registry)

	// Even with parse errors, tree-sitter may succeed but we might not find the function
	// Either way, it should not crash
	assert.NotPanics(t, func() {
		generateTaintSummaries(callGraph, codeGraph, registry)
	})
}
