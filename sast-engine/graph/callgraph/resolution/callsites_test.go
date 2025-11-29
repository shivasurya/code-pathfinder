package resolution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCallSites_SimpleFunctionCalls(t *testing.T) {
	sourceCode := []byte(`
def process():
    foo()
    bar()
    baz()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 3)

	// Check targets (callees)
	assert.Equal(t, "foo", callSites[0].Target)
	assert.Empty(t, callSites[0].Arguments)

	assert.Equal(t, "bar", callSites[1].Target)
	assert.Equal(t, "baz", callSites[2].Target)
}

func TestExtractCallSites_MethodCalls(t *testing.T) {
	sourceCode := []byte(`
def process():
    obj.method()
    self.helper()
    db.query()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 3)

	assert.Equal(t, "obj.method", callSites[0].Target)
	assert.Equal(t, "self.helper", callSites[1].Target)
	assert.Equal(t, "db.query", callSites[2].Target)
}

func TestExtractCallSites_WithArguments(t *testing.T) {
	sourceCode := []byte(`
def process():
    foo(x)
    bar(a, b)
    baz(data, size=10)
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 3)

	// foo(x) - single positional argument
	assert.Equal(t, "foo", callSites[0].Target)
	require.Len(t, callSites[0].Arguments, 1)
	assert.Equal(t, "x", callSites[0].Arguments[0].Value)

	// bar(a, b) - two positional arguments
	assert.Equal(t, "bar", callSites[1].Target)
	require.Len(t, callSites[1].Arguments, 2)
	assert.Equal(t, "a", callSites[1].Arguments[0].Value)
	assert.Equal(t, "b", callSites[1].Arguments[1].Value)

	// baz(data, size=10) - positional and keyword argument
	assert.Equal(t, "baz", callSites[2].Target)
	require.Len(t, callSites[2].Arguments, 2)
	assert.Equal(t, "data", callSites[2].Arguments[0].Value)
	assert.Equal(t, "size=10", callSites[2].Arguments[1].Value)
}

func TestExtractCallSites_NestedCalls(t *testing.T) {
	sourceCode := []byte(`
def outer():
    result = foo(bar(x))
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 2)

	// Both calls should be detected
	callees := []string{callSites[0].Target, callSites[1].Target}
	assert.Contains(t, callees, "foo")
	assert.Contains(t, callees, "bar")
}

func TestExtractCallSites_MultipleFunctions(t *testing.T) {
	sourceCode := []byte(`
def func1():
    foo()

def func2():
    bar()
    baz()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 3)

	// Check callers

	// Check callees
	assert.Equal(t, "foo", callSites[0].Target)
	assert.Equal(t, "bar", callSites[1].Target)
	assert.Equal(t, "baz", callSites[2].Target)
}

func TestExtractCallSites_ClassMethods(t *testing.T) {
	sourceCode := []byte(`
class MyClass:
    def method1(self):
        self.helper()

    def method2(self):
        self.method1()
        other.method()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 3)

	// Check that method names are extracted as callers
	assert.Equal(t, "self.helper", callSites[0].Target)

	assert.Equal(t, "self.method1", callSites[1].Target)

	assert.Equal(t, "other.method", callSites[2].Target)
}

func TestExtractCallSites_ChainedCalls(t *testing.T) {
	sourceCode := []byte(`
def process():
    result = obj.method1().method2()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	// Should detect both the initial call and the chained call
	assert.GreaterOrEqual(t, len(callSites), 1)
}

func TestExtractCallSites_NoFunctionContext(t *testing.T) {
	// Calls at module level (no function context)
	sourceCode := []byte(`
foo()
bar()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 2)

	// Caller should be empty string (module level)

	assert.Equal(t, "foo", callSites[0].Target)
	assert.Equal(t, "bar", callSites[1].Target)
}

func TestExtractCallSites_SourceLocation(t *testing.T) {
	sourceCode := []byte(`
def process():
    foo()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 1)

	// Check location is populated
	assert.NotNil(t, callSites[0].Location)
	assert.Equal(t, "/test/file.py", callSites[0].Location.File)
	assert.Greater(t, callSites[0].Location.Line, 0)
	assert.Greater(t, callSites[0].Location.Column, 0)
}

func TestExtractCallSites_EmptyFile(t *testing.T) {
	sourceCode := []byte(`
# Just comments
# No function calls
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	assert.Empty(t, callSites)
}

func TestExtractCallSites_ComplexArguments(t *testing.T) {
	sourceCode := []byte(`
def process():
    foo(x + y)
    bar([1, 2, 3])
    baz({"key": "value"})
    qux(lambda x: x * 2)
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 4)

	// Each call should have arguments
	assert.NotEmpty(t, callSites[0].Arguments)
	assert.NotEmpty(t, callSites[1].Arguments)
	assert.NotEmpty(t, callSites[2].Arguments)
	assert.NotEmpty(t, callSites[3].Arguments)
}

func TestExtractCallSites_NestedMethodCalls(t *testing.T) {
	sourceCode := []byte(`
def process():
    obj.attr.method()
    self.db.query()
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 2)

	assert.Equal(t, "obj.attr.method", callSites[0].Target)
	assert.Equal(t, "self.db.query", callSites[1].Target)
}

func TestExtractCallSites_WithTestFixture(t *testing.T) {
	// Create a test fixture
	fixturePath := filepath.Join("..", "..", "..", "test-fixtures", "python", "callsites_test", "simple_calls.py")

	// Check if fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Fixture file not found: %s", fixturePath)
	}

	sourceCode, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	absFixturePath, err := filepath.Abs(fixturePath)
	require.NoError(t, err)

	importMap := core.NewImportMap(absFixturePath)
	callSites, err := ExtractCallSites(absFixturePath, sourceCode, importMap)

	require.NoError(t, err)
	assert.NotEmpty(t, callSites)

	// Verify at least one call site was extracted
	assert.Greater(t, len(callSites), 0)

	// Verify structure of first call site
	if len(callSites) > 0 {
		assert.NotEmpty(t, callSites[0].Target)
		assert.NotNil(t, callSites[0].Location)
		assert.Equal(t, absFixturePath, callSites[0].Location.File)
	}
}

func TestExtractArguments_EmptyArgumentList(t *testing.T) {
	sourceCode := []byte(`foo()`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 1)
	assert.Empty(t, callSites[0].Arguments)
}

func TestExtractArguments_OnlyKeywordArguments(t *testing.T) {
	sourceCode := []byte(`
def process():
    foo(name="test", value=42, enabled=True)
`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 1)
	require.Len(t, callSites[0].Arguments, 3)

	assert.Equal(t, "name=\"test\"", callSites[0].Arguments[0].Value)

	assert.Equal(t, "value=42", callSites[0].Arguments[1].Value)

	assert.Equal(t, "enabled=True", callSites[0].Arguments[2].Value)
}

func TestExtractCalleeName_Identifier(t *testing.T) {
	sourceCode := []byte(`foo()`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 1)
	assert.Equal(t, "foo", callSites[0].Target)
}

func TestExtractCalleeName_Attribute(t *testing.T) {
	sourceCode := []byte(`obj.method()`)

	importMap := core.NewImportMap("/test/file.py")
	callSites, err := ExtractCallSites("/test/file.py", sourceCode, importMap)

	require.NoError(t, err)
	require.Len(t, callSites, 1)
	assert.Equal(t, "obj.method", callSites[0].Target)
}
