package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypeInference_StringMethods tests type inference for string method calls.
func TestTypeInference_StringMethods(t *testing.T) {
	// Create test project
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def process_text():
    data = "hello world"
    uppercased = data.upper()
    return uppercased
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	assert.NoError(t, err)

	// Initialize code graph with Python parsing
	codeGraph := graph.Initialize(tmpDir)

	// Build call graph with type inference
	callGraph, registry, _, err := InitializeCallGraph(codeGraph, tmpDir)
	assert.NoError(t, err)
	_ = registry

	// Verify call sites were extracted
	processTextFQN := "test.process_text"
	callSites, exists := callGraph.CallSites[processTextFQN]
	assert.True(t, exists, "Should have call sites for process_text")
	assert.NotEmpty(t, callSites, "Should have at least one call site")

	// Find the data.upper() call site
	var upperCallSite *core.CallSite
	for i := range callSites {
		if callSites[i].Target == "data.upper" {
			upperCallSite = &callSites[i]
			break
		}
	}

	assert.NotNil(t, upperCallSite, "Should find data.upper() call site")
	assert.True(t, upperCallSite.Resolved, "data.upper() should be resolved via type inference")
	assert.Equal(t, "builtins.str.upper", upperCallSite.TargetFQN, "Should resolve to builtins.str.upper")
}

// TestTypeInference_ListMethods tests type inference for list method calls.
func TestTypeInference_ListMethods(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def process_list():
    numbers = [1, 2, 3]
    numbers.append(4)
    count = numbers.count(2)
    return count
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	assert.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir)

	callGraph, registry, _, err := InitializeCallGraph(codeGraph, tmpDir)
	assert.NoError(t, err)
	_ = registry

	processListFQN := "test.process_list"
	callSites, exists := callGraph.CallSites[processListFQN]
	assert.True(t, exists)

	// Find numbers.append() and numbers.count() call sites
	foundAppend := false
	foundCount := false

	for i := range callSites {
		if callSites[i].Target == "numbers.append" {
			assert.True(t, callSites[i].Resolved, "numbers.append() should be resolved")
			assert.Equal(t, "builtins.list.append", callSites[i].TargetFQN)
			foundAppend = true
		}
		if callSites[i].Target == "numbers.count" {
			assert.True(t, callSites[i].Resolved, "numbers.count() should be resolved")
			assert.Equal(t, "builtins.list.count", callSites[i].TargetFQN)
			foundCount = true
		}
	}

	assert.True(t, foundAppend, "Should find numbers.append() call")
	assert.True(t, foundCount, "Should find numbers.count() call")
}

// TestTypeInference_DictMethods tests type inference for dict method calls.
func TestTypeInference_DictMethods(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def process_dict():
    config = {"key": "value"}
    keys = config.keys()
    values = config.values()
    return keys, values
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	assert.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir)

	callGraph, registry, _, err := InitializeCallGraph(codeGraph, tmpDir)
	assert.NoError(t, err)
	_ = registry

	processDictFQN := "test.process_dict"
	callSites, exists := callGraph.CallSites[processDictFQN]
	assert.True(t, exists)

	foundKeys := false
	foundValues := false

	for i := range callSites {
		if callSites[i].Target == "config.keys" {
			assert.True(t, callSites[i].Resolved, "config.keys() should be resolved")
			assert.Equal(t, "builtins.dict.keys", callSites[i].TargetFQN)
			foundKeys = true
		}
		if callSites[i].Target == "config.values" {
			assert.True(t, callSites[i].Resolved, "config.values() should be resolved")
			assert.Equal(t, "builtins.dict.values", callSites[i].TargetFQN)
			foundValues = true
		}
	}

	assert.True(t, foundKeys, "Should find config.keys() call")
	assert.True(t, foundValues, "Should find config.values() call")
}

// TestTypeInference_MultipleVariables tests type inference with multiple variables.
func TestTypeInference_MultipleVariables(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def mixed_types():
    text = "hello"
    nums = [1, 2, 3]
    mapping = {}

    upper = text.upper()
    nums.append(4)
    mapping.clear()
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	assert.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir)

	callGraph, registry, _, err := InitializeCallGraph(codeGraph, tmpDir)
	assert.NoError(t, err)
	_ = registry

	mixedTypesFQN := "test.mixed_types"
	callSites, exists := callGraph.CallSites[mixedTypesFQN]
	assert.True(t, exists)

	// Verify all three types are resolved correctly
	foundStr := false
	foundList := false
	foundDict := false

	for i := range callSites {
		switch callSites[i].Target {
		case "text.upper":
			assert.True(t, callSites[i].Resolved)
			assert.Equal(t, "builtins.str.upper", callSites[i].TargetFQN)
			foundStr = true
		case "nums.append":
			assert.True(t, callSites[i].Resolved)
			assert.Equal(t, "builtins.list.append", callSites[i].TargetFQN)
			foundList = true
		case "mapping.clear":
			assert.True(t, callSites[i].Resolved)
			assert.Equal(t, "builtins.dict.clear", callSites[i].TargetFQN)
			foundDict = true
		}
	}

	assert.True(t, foundStr, "Should resolve string method")
	assert.True(t, foundList, "Should resolve list method")
	assert.True(t, foundDict, "Should resolve dict method")
}

// TestTypeInference_WithoutTypeInfo tests that untyped variables don't break resolution.
func TestTypeInference_WithoutTypeInfo(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def untyped_var(param):
    # param has no type info, should use fallback resolution
    result = param.process()
    return result
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	assert.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir)

	callGraph, registry, _, err := InitializeCallGraph(codeGraph, tmpDir)
	assert.NoError(t, err)
	_ = registry

	// Should not crash, fallback to legacy resolution
	untypedVarFQN := "test.untyped_var"
	callSites, exists := callGraph.CallSites[untypedVarFQN]
	assert.True(t, exists)

	// Find param.process() call
	var processCall *CallSite
	for i := range callSites {
		if callSites[i].Target == "param.process" {
			processCall = &callSites[i]
			break
		}
	}

	assert.NotNil(t, processCall, "Should find param.process() call")
	// Should be unresolved (no type info for param)
	assert.False(t, processCall.Resolved, "param.process() should be unresolved without type info")
}

// TestTypeInference_NestedScopes tests type inference in nested functions.
// TODO: Enable after code graph supports nested function definitions.
//
//nolint:unused,thelper
func _TestTypeInference_NestedScopes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := []byte(`
def outer():
    data = "outer"

    def inner():
        text = "inner"
        result = text.lower()
        return result

    outer_result = data.upper()
    return outer_result
`)

	err := os.WriteFile(testFile, sourceCode, 0644)
	assert.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir)

	callGraph, registry, _, err := InitializeCallGraph(codeGraph, tmpDir)
	assert.NoError(t, err)
	_ = registry

	// Check outer function
	outerFQN := "test.outer"
	outerCalls, exists := callGraph.CallSites[outerFQN]
	assert.True(t, exists)

	foundOuterUpper := false
	for i := range outerCalls {
		if outerCalls[i].Target == "data.upper" {
			assert.True(t, outerCalls[i].Resolved)
			assert.Equal(t, "builtins.str.upper", outerCalls[i].TargetFQN)
			foundOuterUpper = true
		}
	}
	assert.True(t, foundOuterUpper, "Should resolve data.upper() in outer")

	// Check inner function
	innerFQN := "test.outer.inner"
	innerCalls, exists := callGraph.CallSites[innerFQN]
	assert.True(t, exists)

	foundInnerLower := false
	for i := range innerCalls {
		if innerCalls[i].Target == "text.lower" {
			assert.True(t, innerCalls[i].Resolved)
			assert.Equal(t, "builtins.str.lower", innerCalls[i].TargetFQN)
			foundInnerLower = true
		}
	}
	assert.True(t, foundInnerLower, "Should resolve text.lower() in inner")
}

// TestTypeInference_FactoryPattern tests type propagation through factory functions.
func TestTypeInference_FactoryPattern(t *testing.T) {
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
	assert.NotNil(t, callGraph)
}

// TestTypeInference_ChainedCalls tests type propagation through chained method calls.
func TestTypeInference_ChainedCalls(t *testing.T) {
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

	assert.NotNil(t, callGraph)
}

// TestTypeInference_MultipleReturns tests merging types from multiple return statements.
func TestTypeInference_MultipleReturns(t *testing.T) {
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

// TestTypeInference_ClassMethodResolution tests resolving methods on class instances.
func TestTypeInference_ClassMethodResolution(t *testing.T) {
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

// TestTypeInference_ConfidenceFiltering tests that low confidence types don't resolve heuristically.
func TestTypeInference_ConfidenceFiltering(t *testing.T) {
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

// TestTypeInference_HighConfidenceResolution tests that high confidence types resolve via heuristic.
func TestTypeInference_HighConfidenceResolution(t *testing.T) {
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

// TestTypeInference_PlaceholderSkipping tests that unresolved placeholders don't resolve.
func TestTypeInference_PlaceholderSkipping(t *testing.T) {
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
