package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeCallGraph_EmptyCodeGraph(t *testing.T) {
	tmpDir := t.TempDir()

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: []*graph.Edge{},
	}

	callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, callGraph)
	assert.NotNil(t, registry)
	assert.NotNil(t, patternRegistry)
}

func TestInitializeCallGraph_WithSimpleProject(t *testing.T) {
	// Create a simple test project
	tmpDir := t.TempDir()

	// Create a Python file
	testFile := filepath.Join(tmpDir, "test.py")
	sourceCode := []byte(`
def get_input():
    return input()

def process():
    data = get_input()
    eval(data)
`)
	err := os.WriteFile(testFile, sourceCode, 0644)
	require.NoError(t, err)

	// Create a minimal code graph
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:         "node1",
				Type:       "function_definition",
				Name:       "get_input",
				File:       testFile,
				LineNumber: 2,
			},
			"node2": {
				ID:         "node2",
				Type:       "function_definition",
				Name:       "process",
				File:       testFile,
				LineNumber: 5,
			},
		},
		Edges: []*graph.Edge{},
	}

	callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, callGraph)
	assert.NotNil(t, registry)
	assert.NotNil(t, patternRegistry)

	// Verify module registry was built
	assert.NotEmpty(t, registry.Modules)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)

	// Verify patterns were loaded
	assert.NotEmpty(t, patternRegistry.Patterns)
}

func TestAnalyzePatterns_NoMatches(t *testing.T) {
	// Create call graph with safe functions
	callGraph := NewCallGraph()
	callGraph.AddCallSite("myapp.safe_function", CallSite{
		Target:    "print",
		TargetFQN: "builtins.print",
	})

	patternRegistry := NewPatternRegistry()
	patternRegistry.LoadDefaultPatterns()

	matches := AnalyzePatterns(callGraph, patternRegistry)

	assert.Empty(t, matches)
}

func TestAnalyzePatterns_WithMatch(t *testing.T) {
	// Create call graph that matches code injection pattern
	callGraph := NewCallGraph()

	// Source: get_input() calls input()
	callGraph.AddCallSite("myapp.get_input", CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	// Sink: process() calls eval()
	callGraph.AddCallSite("myapp.process", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	// Create path from source to sink
	callGraph.AddEdge("myapp.get_input", "myapp.process")

	patternRegistry := NewPatternRegistry()
	patternRegistry.LoadDefaultPatterns()

	matches := AnalyzePatterns(callGraph, patternRegistry)

	require.Len(t, matches, 1)
	assert.Equal(t, "CODE-INJECTION-001", matches[0].PatternID)
	assert.Equal(t, "Code injection via eval with user input", matches[0].PatternName)
	assert.Equal(t, SeverityCritical, matches[0].Severity)
	assert.Equal(t, "CWE-94", matches[0].CWE)
}

func TestAnalyzePatterns_WithSanitizer(t *testing.T) {
	// Create call graph with sanitizer in the path
	callGraph := NewCallGraph()

	// Source: get_input() calls input()
	callGraph.AddCallSite("myapp.get_input", CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	// Sanitizer: sanitize_input() calls sanitize()
	callGraph.AddCallSite("myapp.sanitize_input", CallSite{
		Target:    "sanitize",
		TargetFQN: "myapp.utils.sanitize",
	})

	// Sink: process() calls eval()
	callGraph.AddCallSite("myapp.process", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	// Create path with sanitizer: source -> sanitizer -> sink
	callGraph.AddEdge("myapp.get_input", "myapp.sanitize_input")
	callGraph.AddEdge("myapp.sanitize_input", "myapp.process")

	patternRegistry := NewPatternRegistry()
	patternRegistry.LoadDefaultPatterns()

	matches := AnalyzePatterns(callGraph, patternRegistry)

	// Should not match because sanitizer is present
	assert.Empty(t, matches)
}

func TestPatternMatch_Structure(t *testing.T) {
	match := PatternMatch{
		PatternID:   "TEST-001",
		PatternName: "Test Pattern",
		Description: "Test description",
		Severity:    SeverityHigh,
		CWE:         "CWE-123",
		OWASP:       "A01:2021-Test",
	}

	assert.Equal(t, "TEST-001", match.PatternID)
	assert.Equal(t, "Test Pattern", match.PatternName)
	assert.Equal(t, "Test description", match.Description)
	assert.Equal(t, SeverityHigh, match.Severity)
	assert.Equal(t, "CWE-123", match.CWE)
	assert.Equal(t, "A01:2021-Test", match.OWASP)
}

func TestInitializeCallGraph_Integration(t *testing.T) {
	// End-to-end integration test
	tmpDir := t.TempDir()

	// Create a Python package structure
	utilsDir := filepath.Join(tmpDir, "utils")
	err := os.MkdirAll(utilsDir, 0755)
	require.NoError(t, err)

	// Create utils/helpers.py
	helpersFile := filepath.Join(utilsDir, "helpers.py")
	helpersCode := []byte(`
def sanitize(data):
    return data.strip()
`)
	err = os.WriteFile(helpersFile, helpersCode, 0644)
	require.NoError(t, err)

	// Create main.py
	mainFile := filepath.Join(tmpDir, "main.py")
	mainCode := []byte(`
from utils.helpers import sanitize

def get_input():
    return input()

def process():
    data = get_input()
    clean_data = sanitize(data)
    eval(clean_data)
`)
	err = os.WriteFile(mainFile, mainCode, 0644)
	require.NoError(t, err)

	// Create code graph
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:         "node1",
				Type:       "function_definition",
				Name:       "sanitize",
				File:       helpersFile,
				LineNumber: 2,
			},
			"node2": {
				ID:         "node2",
				Type:       "function_definition",
				Name:       "get_input",
				File:       mainFile,
				LineNumber: 4,
			},
			"node3": {
				ID:         "node3",
				Type:       "function_definition",
				Name:       "process",
				File:       mainFile,
				LineNumber: 7,
			},
		},
		Edges: []*graph.Edge{},
	}

	// Initialize call graph
	callGraph, registry, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, callGraph)
	assert.NotNil(t, registry)
	assert.NotNil(t, patternRegistry)

	// Verify modules were registered
	assert.Contains(t, registry.Modules, "utils.helpers")
	assert.Contains(t, registry.Modules, "main")

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)
}
