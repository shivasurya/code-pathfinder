package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/stretchr/testify/assert"
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
	var upperCallSite *CallSite
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
// TODO: Enable after code graph supports nested function definitions
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
