package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCallGraphFromPath(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	// Create a simple Python file
	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def greet(name):
    return f"Hello, {name}"

def main():
    message = greet("World")
    print(message)

if __name__ == "__main__":
    main()
`), 0644)
	require.NoError(t, err)

	// Parse the project to get code graph
	codeGraph := graph.Initialize(tmpDir)
	assert.NotNil(t, codeGraph)

	// Build call graph from path
	callGraph, moduleRegistry, err := BuildCallGraphFromPath(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)
	assert.NotNil(t, callGraph)
	assert.NotNil(t, moduleRegistry)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)

	// Verify module registry was built
	assert.NotEmpty(t, moduleRegistry.Modules)
}

func TestBuildCallGraphFromPath_EmptyProject(t *testing.T) {
	// Create an empty temporary directory
	tmpDir := t.TempDir()

	// Parse the empty project
	codeGraph := graph.Initialize(tmpDir)

	// Build call graph should succeed but be empty
	callGraph, moduleRegistry, err := BuildCallGraphFromPath(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)
	assert.NotNil(t, callGraph)
	assert.NotNil(t, moduleRegistry)
	assert.Empty(t, callGraph.Functions)
}

func TestBuildCallGraphFromPath_WithImports(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	// Create utils.py
	utilsPy := filepath.Join(tmpDir, "utils.py")
	err := os.WriteFile(utilsPy, []byte(`
def helper():
    return 42
`), 0644)
	require.NoError(t, err)

	// Create main.py that imports utils
	mainPy := filepath.Join(tmpDir, "main.py")
	err = os.WriteFile(mainPy, []byte(`
from utils import helper

def main():
    result = helper()
    return result
`), 0644)
	require.NoError(t, err)

	// Parse the project
	codeGraph := graph.Initialize(tmpDir)

	// Build call graph
	callGraph, moduleRegistry, err := BuildCallGraphFromPath(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)
	assert.NotNil(t, callGraph)
	assert.NotNil(t, moduleRegistry)

	// Verify both modules are registered
	assert.GreaterOrEqual(t, len(moduleRegistry.Modules), 2)

	// Verify functions from both files are indexed
	assert.NotEmpty(t, callGraph.Functions)
}
