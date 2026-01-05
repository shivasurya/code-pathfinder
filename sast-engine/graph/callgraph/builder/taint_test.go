package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTaintSummaries(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	// Create a Python file with potential taint flow
	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def get_user_input():
    return input("Enter data: ")

def sanitize(data):
    return data.strip()

def process(data):
    clean_data = sanitize(data)
    return clean_data

def unsafe_process(data):
    # Direct use without sanitization
    exec(data)

def main():
    user_data = get_user_input()
    result = process(user_data)
    unsafe_process(user_data)
`), 0644)
	require.NoError(t, err)

	// Parse the project
	codeGraph := graph.Initialize(tmpDir)

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Generate taint summaries
	GenerateTaintSummaries(callGraph, codeGraph, moduleRegistry)

	// Verify summaries were generated
	assert.NotNil(t, callGraph.Summaries)

	// Check that some functions have taint summaries
	foundSummary := false
	for funcFQN := range callGraph.Functions {
		if summary, exists := callGraph.Summaries[funcFQN]; exists {
			foundSummary = true
			assert.NotNil(t, summary)
			// Summaries should have detections
			if len(summary.Detections) > 0 {
				t.Logf("Function %s has taint summary: %d detections",
					funcFQN, len(summary.Detections))
			}
		}
	}

	assert.True(t, foundSummary, "Expected at least one taint summary to be generated")
}

func TestGenerateTaintSummaries_EmptyCallGraph(t *testing.T) {
	callGraph := core.NewCallGraph()
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}
	moduleRegistry := core.NewModuleRegistry()

	// Should not panic with empty inputs
	GenerateTaintSummaries(callGraph, codeGraph, moduleRegistry)

	// Summaries should be initialized but empty
	assert.NotNil(t, callGraph.Summaries)
	assert.Empty(t, callGraph.Summaries)
}

func TestGenerateTaintSummaries_NoTaintFlow(t *testing.T) {
	// Create a temporary project with safe code
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def add(a, b):
    return a + b

def multiply(a, b):
    return a * b

def calculate():
    x = add(2, 3)
    y = multiply(x, 4)
    return y
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir)

	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Generate taint summaries
	GenerateTaintSummaries(callGraph, codeGraph, moduleRegistry)

	// Summaries should exist but most won't have sources/sinks
	assert.NotNil(t, callGraph.Summaries)
}
