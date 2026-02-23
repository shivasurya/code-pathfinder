package diagnostic

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractAllFunctions_SimpleFile tests extraction from a single file with top-level functions.
func TestExtractAllFunctions_SimpleFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := `
def function_one():
    pass

def function_two(arg1, arg2):
    return arg1 + arg2
`
	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	require.NoError(t, err)

	// Extract functions
	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)

	// Verify count
	assert.Equal(t, 2, len(functions), "Should find 2 functions")

	// Verify first function
	f1 := functions[0]
	assert.Equal(t, "function_one", f1.FunctionName)
	assert.Equal(t, "test.function_one", f1.FQN)
	assert.Equal(t, 2, f1.StartLine)
	assert.Equal(t, 3, f1.EndLine)
	assert.Equal(t, 2, f1.LOC)
	assert.False(t, f1.IsMethod)
	assert.False(t, f1.IsAsync)
	assert.Empty(t, f1.ClassName)

	// Verify second function
	f2 := functions[1]
	assert.Equal(t, "function_two", f2.FunctionName)
	assert.Contains(t, f2.SourceCode, "def function_two")
	assert.Contains(t, f2.SourceCode, "return arg1 + arg2")
}

// TestExtractAllFunctions_ClassMethods tests extraction of class methods.
func TestExtractAllFunctions_ClassMethods(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "models.py")

	sourceCode := `
class User:
    def __init__(self, name):
        self.name = name

    def save(self):
        pass

    @classmethod
    def load(cls, id):
        pass
`
	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	require.NoError(t, err)

	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)

	// Should find 3 methods
	assert.Equal(t, 3, len(functions))

	// Verify all are methods
	for _, f := range functions {
		assert.True(t, f.IsMethod, "All should be methods")
		assert.Equal(t, "User", f.ClassName)
		assert.Equal(t, "models.User."+f.FunctionName, f.FQN)
	}

	// Check function names
	names := []string{functions[0].FunctionName, functions[1].FunctionName, functions[2].FunctionName}
	assert.Contains(t, names, "__init__")
	assert.Contains(t, names, "save")
	assert.Contains(t, names, "load")
}

// TestExtractAllFunctions_AsyncFunctions tests async function detection.
func TestExtractAllFunctions_AsyncFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "async_test.py")

	sourceCode := `
async def fetch_data():
    pass

def sync_function():
    pass
`
	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	require.NoError(t, err)

	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, 2, len(functions))

	// Find async function
	var asyncFunc *FunctionMetadata
	for _, f := range functions {
		if f.FunctionName == "fetch_data" {
			asyncFunc = f
			break
		}
	}

	require.NotNil(t, asyncFunc, "Should find fetch_data")
	assert.True(t, asyncFunc.IsAsync, "fetch_data should be async")

	// Verify sync function is not async
	var syncFunc *FunctionMetadata
	for _, f := range functions {
		if f.FunctionName == "sync_function" {
			syncFunc = f
			break
		}
	}

	require.NotNil(t, syncFunc)
	assert.False(t, syncFunc.IsAsync)
}

// TestExtractAllFunctions_NestedFunctions tests nested function handling.
func TestExtractAllFunctions_NestedFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "nested.py")

	sourceCode := `
def outer():
    def inner():
        pass
    return inner
`
	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	require.NoError(t, err)

	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)

	// Should find both outer and inner
	assert.Equal(t, 2, len(functions))

	names := []string{functions[0].FunctionName, functions[1].FunctionName}
	assert.Contains(t, names, "outer")
	assert.Contains(t, names, "inner")
}

// TestExtractAllFunctions_MultipleFiles tests extraction across multiple files.
func TestExtractAllFunctions_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files
	file1 := filepath.Join(tmpDir, "views.py")
	file2 := filepath.Join(tmpDir, "models.py")

	err := os.WriteFile(file1, []byte("def view_func():\n    pass"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(file2, []byte("def model_func():\n    pass"), 0644)
	require.NoError(t, err)

	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, 2, len(functions))

	// Should have different FQNs
	fqns := []string{functions[0].FQN, functions[1].FQN}
	assert.Contains(t, fqns, "views.view_func")
	assert.Contains(t, fqns, "models.model_func")
}

// TestExtractAllFunctions_SkipsDirectories tests that common directories are skipped.
func TestExtractAllFunctions_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create __pycache__ directory with Python file
	pycacheDir := filepath.Join(tmpDir, "__pycache__")
	err := os.Mkdir(pycacheDir, 0755)
	require.NoError(t, err)

	pycacheFile := filepath.Join(pycacheDir, "test.py")
	err = os.WriteFile(pycacheFile, []byte("def should_skip():\n    pass"), 0644)
	require.NoError(t, err)

	// Create normal file
	normalFile := filepath.Join(tmpDir, "normal.py")
	err = os.WriteFile(normalFile, []byte("def should_find():\n    pass"), 0644)
	require.NoError(t, err)

	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)

	// Should only find function from normal file
	assert.Equal(t, 1, len(functions))
	assert.Equal(t, "should_find", functions[0].FunctionName)
}

// TestBuildModuleName tests module name construction.
func TestBuildModuleName(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		projectRoot string
		expected    string
	}{
		{
			name:        "simple file",
			filePath:    "/project/myapp/views.py",
			projectRoot: "/project",
			expected:    "myapp.views",
		},
		{
			name:        "nested directory",
			filePath:    "/project/myapp/api/v1/endpoints.py",
			projectRoot: "/project",
			expected:    "myapp.api.v1.endpoints",
		},
		{
			name:        "__init__ file",
			filePath:    "/project/myapp/__init__.py",
			projectRoot: "/project",
			expected:    "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildModuleName(tt.filePath, tt.projectRoot)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractAllFunctions_ErrorHandling tests error handling cases.
func TestExtractAllFunctions_ErrorHandling(t *testing.T) {
	// Test with non-existent directory
	_, err := ExtractAllFunctions("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to walk directory")
}

// TestExtractAllFunctions_EmptyDirectory tests empty directory.
func TestExtractAllFunctions_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 0, len(functions))
}

// TestExtractAllFunctions_InvalidPython tests handling of invalid Python syntax.
func TestExtractAllFunctions_InvalidPython(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.py")

	// Write invalid Python (but tree-sitter will still parse it partially)
	sourceCode := `
def incomplete_function(
    # Missing closing parenthesis and body
`
	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	require.NoError(t, err)

	// Should not crash, just handle gracefully
	functions, err := ExtractAllFunctions(tmpDir)
	require.NoError(t, err)
	// May or may not find the incomplete function, but shouldn't crash
	_ = functions
}

// TestShouldSkipDir tests directory skipping logic.
func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/project/__pycache__/file.py", true},
		{"/project/.git/file.py", true},
		{"/project/.venv/file.py", true},
		{"/project/venv/file.py", true},
		{"/project/node_modules/file.py", true},
		{"/project/.tox/file.py", true},
		{"/project/.pytest_cache/file.py", true},
		{"/project/build/file.py", true},
		{"/project/dist/file.py", true},
		{"/project/.eggs/file.py", true},
		{"/project/myapp/views.py", false},
		{"/project/src/main.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldSkipDir(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// BenchmarkExtractAllFunctions benchmarks extraction performance.
func BenchmarkExtractAllFunctions(b *testing.B) {
	// Create temporary directory with 100 functions
	tmpDir := b.TempDir()

	for i := range 100 {
		fileName := filepath.Join(tmpDir, fmt.Sprintf("file%d.py", i))
		sourceCode := `
def function_one():
    pass

def function_two():
    pass

class MyClass:
    def method_one(self):
        pass
`
		err := os.WriteFile(fileName, []byte(sourceCode), 0644)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractAllFunctions(tmpDir)
	}
}
