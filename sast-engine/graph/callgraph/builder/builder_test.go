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

func TestBuildCallGraph(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def greet(name):
    return f"Hello, {name}"

def main():
    message = greet("World")
    print(message)
`), 0644)
	require.NoError(t, err)

	// Parse project
	codeGraph := graph.Initialize(tmpDir)

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)

	// Verify edges exist
	assert.NotNil(t, callGraph.Edges)

	// Verify reverse edges exist
	assert.NotNil(t, callGraph.ReverseEdges)
}

func TestIndexFunctions(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(mainPy, []byte(`
def func1():
    pass

def func2():
    pass

class MyClass:
    def method1(self):
        pass
`), 0644)
	require.NoError(t, err)

	// Parse project
	codeGraph := graph.Initialize(tmpDir)

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	// Create call graph and index functions
	callGraph := core.NewCallGraph()
	IndexFunctions(codeGraph, callGraph, moduleRegistry)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)

	// Count functions/methods
	functionCount := 0
	for _, node := range callGraph.Functions {
		if node.Type == "function_definition" || node.Type == "method_declaration" {
			functionCount++
		}
	}
	assert.GreaterOrEqual(t, functionCount, 3, "Should have at least 3 functions/methods")
}

func TestGetFunctionsInFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	err := os.WriteFile(testFile, []byte(`
def func1():
    pass

def func2():
    pass
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir)

	// Get functions in file
	functions := GetFunctionsInFile(codeGraph, testFile)

	// Verify functions were found
	assert.NotEmpty(t, functions)
	assert.GreaterOrEqual(t, len(functions), 2, "Should find at least 2 functions")
}

func TestFindContainingFunction(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
def outer_function():
    x = 1
    y = 2
    return x + y
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir)

	// Get functions
	functions := GetFunctionsInFile(codeGraph, testFile)
	require.NotEmpty(t, functions)

	// Test finding containing function for a location inside the function
	location := core.Location{
		File:   testFile,
		Line:   3,
		Column: 5, // Inside function body
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath)

	// Should find the outer_function
	assert.NotEmpty(t, containingFQN)
	assert.Contains(t, containingFQN, "outer_function")
}

func TestFindContainingFunction_ModuleLevel(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
MODULE_VAR = 42

def my_function():
    pass
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir)

	functions := GetFunctionsInFile(codeGraph, testFile)

	// Test module-level code (column == 1)
	location := core.Location{
		File:   testFile,
		Line:   2,
		Column: 1, // Module level
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath)

	// Should return empty for module-level code
	assert.Empty(t, containingFQN)
}

func TestValidateFQN(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()

	// Add a test module
	moduleRegistry.Modules["mymodule"] = "/path/to/mymodule.py"
	moduleRegistry.FileToModule["/path/to/mymodule.py"] = "mymodule"

	tests := []struct {
		name     string
		fqn      string
		expected bool
	}{
		{"Valid module FQN", "mymodule.func", true},
		{"Invalid module FQN", "unknownmodule.func", false},
		{"Empty FQN", "", false},
		{"Valid module name without dot", "mymodule", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFQN(tt.fqn, moduleRegistry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectPythonVersion(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	// Test with .python-version file
	pythonVersionFile := filepath.Join(tmpDir, ".python-version")
	err := os.WriteFile(pythonVersionFile, []byte("3.11.0\n"), 0644)
	require.NoError(t, err)

	version := DetectPythonVersion(tmpDir)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "3.11")
}

func TestDetectPythonVersion_NoPythonVersionFile(t *testing.T) {
	// Create an empty temporary directory
	tmpDir := t.TempDir()

	// Should fall back to checking pyproject.toml or default
	version := DetectPythonVersion(tmpDir)
	// Should return a default version or detect from system
	assert.NotEmpty(t, version)
}

func TestBuildCallGraph_WithEdges(t *testing.T) {
	// Create a project with function calls
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def helper():
    return 42

def caller():
    result = helper()
    return result
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir)

	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify edges were created
	assert.NotEmpty(t, callGraph.Edges)

	// Check that caller has edges to helper
	foundEdge := false
	for callerFQN, callees := range callGraph.Edges {
		if len(callees) > 0 {
			foundEdge = true
			t.Logf("Function %s calls: %v", callerFQN, callees)
		}
	}

	assert.True(t, foundEdge, "Expected at least one call edge")
}
