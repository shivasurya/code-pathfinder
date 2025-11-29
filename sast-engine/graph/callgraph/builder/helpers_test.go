package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileBytes(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!\nTest content")

	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Test reading the file
	content, err := ReadFileBytes(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestReadFileBytes_NonExistent(t *testing.T) {
	content, err := ReadFileBytes("/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Nil(t, content)
}

func TestFindFunctionAtLine(t *testing.T) {
	sourceCode := []byte(`
def function_at_line_2():
    pass

def function_at_line_5():
    return 42

class MyClass:
    def method_at_line_9(self):
        pass
`)

	tree, err := extraction.ParsePythonFile(sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	tests := []struct {
		name       string
		lineNumber uint32
		expected   bool
	}{
		{"Find function at line 2", 2, true},
		{"Find function at line 5", 5, true},
		{"Find method at line 9", 9, true},
		{"No function at line 1", 1, false},
		{"No function at line 3", 3, false},
		{"No function at line 10", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFunctionAtLine(tree.RootNode(), tt.lineNumber)
			if tt.expected {
				assert.NotNil(t, result, "Expected to find function at line %d", tt.lineNumber)
				assert.Equal(t, "function_definition", result.Type())
			} else {
				assert.Nil(t, result, "Expected no function at line %d", tt.lineNumber)
			}
		})
	}
}

func TestFindFunctionAtLine_NilRoot(t *testing.T) {
	result := FindFunctionAtLine(nil, 1)
	assert.Nil(t, result)
}

func TestFindFunctionAtLine_NestedFunctions(t *testing.T) {
	sourceCode := []byte(`
def outer():
    def inner():
        pass
    return inner
`)

	tree, err := extraction.ParsePythonFile(sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	// Should find outer function at line 2
	result := FindFunctionAtLine(tree.RootNode(), 2)
	assert.NotNil(t, result)
	assert.Equal(t, "function_definition", result.Type())

	// Should find inner function at line 3
	result = FindFunctionAtLine(tree.RootNode(), 3)
	assert.NotNil(t, result)
	assert.Equal(t, "function_definition", result.Type())
}
