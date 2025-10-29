package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCallTarget_SimpleImportedFunction(t *testing.T) {
	// Test resolving a simple imported function name
	// from myapp.utils import sanitize
	// sanitize() → myapp.utils.sanitize

	registry := NewModuleRegistry()
	registry.AddModule("myapp.utils", "/project/myapp/utils.py")
	registry.AddModule("myapp.views", "/project/myapp/views.py")

	importMap := NewImportMap("/project/myapp/views.py")
	importMap.AddImport("sanitize", "myapp.utils.sanitize")

	codeGraph := &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}
	fqn, resolved := resolveCallTarget("sanitize", importMap, registry, "myapp.views", codeGraph)

	assert.True(t, resolved)
	assert.Equal(t, "myapp.utils.sanitize", fqn)
}

func TestResolveCallTarget_QualifiedImport(t *testing.T) {
	// Test resolving a qualified call through imported module
	// import myapp.utils as utils
	// utils.sanitize() → myapp.utils.sanitize

	registry := NewModuleRegistry()
	registry.AddModule("myapp.utils", "/project/myapp/utils.py")
	registry.AddModule("myapp.views", "/project/myapp/views.py")

	importMap := NewImportMap("/project/myapp/views.py")
	importMap.AddImport("utils", "myapp.utils")

	codeGraph := &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}
	fqn, resolved := resolveCallTarget("utils.sanitize", importMap, registry, "myapp.views", codeGraph)

	assert.True(t, resolved)
	assert.Equal(t, "myapp.utils.sanitize", fqn)
}

func TestResolveCallTarget_SameModuleFunction(t *testing.T) {
	// Test resolving a function in the same module
	// No imports needed - just local function call

	registry := NewModuleRegistry()
	registry.AddModule("myapp.views", "/project/myapp/views.py")

	importMap := NewImportMap("/project/myapp/views.py")

	codeGraph := &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}
	fqn, resolved := resolveCallTarget("helper", importMap, registry, "myapp.views", codeGraph)

	assert.True(t, resolved)
	assert.Equal(t, "myapp.views.helper", fqn)
}

func TestResolveCallTarget_UnresolvedMethodCall(t *testing.T) {
	// Test that method calls on objects are marked as unresolved
	// obj.method() → can't resolve without type inference

	registry := NewModuleRegistry()
	registry.AddModule("myapp.views", "/project/myapp/views.py")

	importMap := NewImportMap("/project/myapp/views.py")

	codeGraph := &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}
	fqn, resolved := resolveCallTarget("obj.method", importMap, registry, "myapp.views", codeGraph)

	assert.False(t, resolved)
	assert.Equal(t, "obj.method", fqn)
}

func TestResolveCallTarget_NonExistentFunction(t *testing.T) {
	// Test resolving a function that doesn't exist in registry

	registry := NewModuleRegistry()
	registry.AddModule("myapp.views", "/project/myapp/views.py")

	importMap := NewImportMap("/project/myapp/views.py")
	importMap.AddImport("missing", "nonexistent.module.function")

	codeGraph := &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}
	fqn, resolved := resolveCallTarget("missing", importMap, registry, "myapp.views", codeGraph)

	assert.False(t, resolved)
	assert.Equal(t, "nonexistent.module.function", fqn)
}

func TestValidateFQN_ModuleExists(t *testing.T) {
	registry := NewModuleRegistry()
	registry.AddModule("myapp.utils", "/project/myapp/utils.py")

	valid := validateFQN("myapp.utils", registry)
	assert.True(t, valid)
}

func TestValidateFQN_FunctionInModule(t *testing.T) {
	registry := NewModuleRegistry()
	registry.AddModule("myapp.utils", "/project/myapp/utils.py")

	// Even though "myapp.utils.sanitize" isn't explicitly registered,
	// it's valid because parent module "myapp.utils" exists
	valid := validateFQN("myapp.utils.sanitize", registry)
	assert.True(t, valid)
}

func TestValidateFQN_NonExistent(t *testing.T) {
	registry := NewModuleRegistry()
	registry.AddModule("myapp.utils", "/project/myapp/utils.py")

	valid := validateFQN("nonexistent.module", registry)
	assert.False(t, valid)
}

func TestIndexFunctions(t *testing.T) {
	// Test indexing function definitions from code graph

	registry := NewModuleRegistry()
	registry.AddModule("myapp.views", "/project/myapp/views.py")
	registry.AddModule("myapp.utils", "/project/myapp/utils.py")

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:         "node1",
				Type:       "function_definition",
				Name:       "get_user",
				File:       "/project/myapp/views.py",
				LineNumber: 10,
			},
			"node2": {
				ID:         "node2",
				Type:       "function_definition",
				Name:       "sanitize",
				File:       "/project/myapp/utils.py",
				LineNumber: 5,
			},
			"node3": {
				ID:   "node3",
				Type: "class_declaration",
				Name: "MyClass",
				File: "/project/myapp/views.py",
			},
		},
	}

	callGraph := NewCallGraph()
	indexFunctions(codeGraph, callGraph, registry)

	// Should have indexed both functions
	assert.Len(t, callGraph.Functions, 2)
	assert.NotNil(t, callGraph.Functions["myapp.views.get_user"])
	assert.NotNil(t, callGraph.Functions["myapp.utils.sanitize"])
	// Should not index class declaration
	assert.Nil(t, callGraph.Functions["myapp.views.MyClass"])
}

func TestGetFunctionsInFile(t *testing.T) {
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:         "node1",
				Type:       "function_definition",
				Name:       "func1",
				File:       "/project/file1.py",
				LineNumber: 10,
			},
			"node2": {
				ID:         "node2",
				Type:       "function_definition",
				Name:       "func2",
				File:       "/project/file1.py",
				LineNumber: 20,
			},
			"node3": {
				ID:         "node3",
				Type:       "function_definition",
				Name:       "func3",
				File:       "/project/file2.py",
				LineNumber: 5,
			},
		},
	}

	functions := getFunctionsInFile(codeGraph, "/project/file1.py")

	assert.Len(t, functions, 2)
	names := []string{functions[0].Name, functions[1].Name}
	assert.Contains(t, names, "func1")
	assert.Contains(t, names, "func2")
}

func TestFindContainingFunction(t *testing.T) {
	functions := []*graph.Node{
		{
			Name:       "func1",
			LineNumber: 10,
		},
		{
			Name:       "func2",
			LineNumber: 30,
		},
	}

	tests := []struct {
		name           string
		callLine       int
		expectedFQN    string
		expectedEmpty  bool
	}{
		{
			name:          "Call before any function",
			callLine:      5,
			expectedEmpty: true,
		},
		{
			name:        "Call in first function",
			callLine:    15,
			expectedFQN: "myapp.func1",
		},
		{
			name:        "Call in second function",
			callLine:    35,
			expectedFQN: "myapp.func2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location := Location{Line: tt.callLine}
			fqn := findContainingFunction(location, functions, "myapp")

			if tt.expectedEmpty {
				assert.Empty(t, fqn)
			} else {
				assert.Equal(t, tt.expectedFQN, fqn)
			}
		})
	}
}

func TestBuildCallGraph_SimpleCase(t *testing.T) {
	// Test building a simple call graph with one file and one function call

	// Create a temporary test fixture
	tmpDir := t.TempDir()
	viewsFile := filepath.Join(tmpDir, "views.py")

	sourceCode := []byte(`
def get_user():
    sanitize(data)
`)

	err := os.WriteFile(viewsFile, sourceCode, 0644)
	require.NoError(t, err)

	// Build module registry
	registry := NewModuleRegistry()
	registry.AddModule("views", viewsFile)

	// Create a minimal code graph with function definition
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:         "node1",
				Type:       "function_definition",
				Name:       "get_user",
				File:       viewsFile,
				LineNumber: 2,
			},
			"node2": {
				ID:         "node2",
				Type:       "function_definition",
				Name:       "sanitize",
				File:       viewsFile,
				LineNumber: 10, // Hypothetical
			},
		},
	}

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)

	require.NoError(t, err)
	require.NotNil(t, callGraph)

	// Verify call sites were extracted
	assert.NotEmpty(t, callGraph.CallSites)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)
}

func TestBuildCallGraph_WithImports(t *testing.T) {
	// Test building call graph with imports between modules

	// Create temporary test fixtures
	tmpDir := t.TempDir()
	utilsDir := filepath.Join(tmpDir, "utils")
	err := os.MkdirAll(utilsDir, 0755)
	require.NoError(t, err)

	utilsFile := filepath.Join(utilsDir, "helpers.py")
	viewsFile := filepath.Join(tmpDir, "views.py")

	utilsCode := []byte(`
def sanitize(data):
    return data.strip()
`)

	viewsCode := []byte(`
from utils.helpers import sanitize

def get_user():
    sanitize(data)
`)

	err = os.WriteFile(utilsFile, utilsCode, 0644)
	require.NoError(t, err)
	err = os.WriteFile(viewsFile, viewsCode, 0644)
	require.NoError(t, err)

	// Build module registry
	registry := NewModuleRegistry()
	registry.AddModule("utils.helpers", utilsFile)
	registry.AddModule("views", viewsFile)

	// Create code graph with both functions
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:         "node1",
				Type:       "function_definition",
				Name:       "get_user",
				File:       viewsFile,
				LineNumber: 4,
			},
			"node2": {
				ID:         "node2",
				Type:       "function_definition",
				Name:       "sanitize",
				File:       utilsFile,
				LineNumber: 2,
			},
		},
	}

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, registry, tmpDir)

	require.NoError(t, err)
	require.NotNil(t, callGraph)

	// Verify call sites
	viewsCallSites := callGraph.CallSites["views.get_user"]
	assert.NotEmpty(t, viewsCallSites, "Expected call sites for views.get_user")

	// Verify at least one call was found
	if len(viewsCallSites) > 0 {
		// Check that the call target was resolved
		found := false
		for _, cs := range viewsCallSites {
			if cs.Target == "sanitize" {
				found = true
				// Should be resolved to utils.helpers.sanitize
				assert.True(t, cs.Resolved, "Call should be resolved")
				assert.Equal(t, "utils.helpers.sanitize", cs.TargetFQN)
			}
		}
		assert.True(t, found, "Expected to find call to sanitize")
	}

	// Verify edges
	callees := callGraph.GetCallees("views.get_user")
	assert.Contains(t, callees, "utils.helpers.sanitize", "Expected edge from get_user to sanitize")

	// Verify reverse edges
	callers := callGraph.GetCallers("utils.helpers.sanitize")
	assert.Contains(t, callers, "views.get_user", "Expected reverse edge from sanitize to get_user")
}

func TestBuildCallGraph_WithTestFixture(t *testing.T) {
	// Integration test with actual test fixtures

	// Use the callsites_test fixture we created in PR #5
	fixturePath := filepath.Join("..", "..", "..", "test-src", "python", "callsites_test")
	absFixturePath, err := filepath.Abs(fixturePath)
	require.NoError(t, err)

	// Check if fixture exists
	if _, err := os.Stat(absFixturePath); os.IsNotExist(err) {
		t.Skipf("Fixture directory not found: %s", absFixturePath)
	}

	// Build module registry
	registry, err := BuildModuleRegistry(absFixturePath)
	require.NoError(t, err)

	// For this test, create a minimal code graph
	// In real usage, this would come from the main graph building
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Scan for Python files and create function nodes
	err = filepath.Walk(absFixturePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".py" {
			return nil
		}

		modulePath, ok := registry.FileToModule[path]
		if !ok {
			return nil
		}

		// Add some dummy function nodes
		// In real scenario these would be parsed from AST
		nodeID := "node_" + modulePath + "_process_data"
		codeGraph.Nodes[nodeID] = &graph.Node{
			ID:         nodeID,
			Type:       "function_definition",
			Name:       "process_data",
			File:       path,
			LineNumber: 3,
		}

		return nil
	})
	require.NoError(t, err)

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, registry, absFixturePath)

	require.NoError(t, err)
	require.NotNil(t, callGraph)

	// Just verify it runs without error
	// Detailed validation would require more sophisticated fixtures
	assert.NotNil(t, callGraph.Edges)
	assert.NotNil(t, callGraph.CallSites)
}
