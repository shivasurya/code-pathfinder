package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
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

// ========== findGoNodeByByteRange tests ==========

func TestFindGoNodeByByteRange(t *testing.T) {
	sourceCode := []byte(`package main

func hello() {
	fmt.Println("hello")
}

func add(a, b int) int {
	return a + b
}
`)

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	root := tree.RootNode()

	tests := []struct {
		name       string
		startByte  uint32
		endByte    uint32
		expectNil  bool
		expectName string
	}{
		{
			name:       "Find hello function",
			startByte:  findGoNodeStartByte(root, "function_declaration", "hello", sourceCode),
			endByte:    findGoNodeEndByte(root, "function_declaration", "hello", sourceCode),
			expectNil:  false,
			expectName: "hello",
		},
		{
			name:       "Find add function",
			startByte:  findGoNodeStartByte(root, "function_declaration", "add", sourceCode),
			endByte:    findGoNodeEndByte(root, "function_declaration", "add", sourceCode),
			expectNil:  false,
			expectName: "add",
		},
		{
			name:      "No match at wrong bytes",
			startByte: 0,
			endByte:   5,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findGoNodeByByteRange(root, tt.startByte, tt.endByte)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, "function_declaration", result.Type())
				nameNode := result.ChildByFieldName("name")
				require.NotNil(t, nameNode)
				assert.Equal(t, tt.expectName, nameNode.Content(sourceCode))
			}
		})
	}
}

func TestFindGoNodeByByteRange_Method(t *testing.T) {
	sourceCode := []byte(`package main

type Server struct{}

func (s *Server) Start() {
	fmt.Println("starting")
}
`)

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	require.NoError(t, err)
	defer tree.Close()

	root := tree.RootNode()

	// Find the method_declaration node
	methodNode := findFirstGoNodeOfType(root, "method_declaration")
	require.NotNil(t, methodNode)

	result := findGoNodeByByteRange(root, methodNode.StartByte(), methodNode.EndByte())
	require.NotNil(t, result)
	assert.Equal(t, "method_declaration", result.Type())
}

func TestFindGoNodeByByteRange_NilRoot(t *testing.T) {
	result := findGoNodeByByteRange(nil, 0, 100)
	assert.Nil(t, result)
}

// ========== Go test helpers ==========

func findFirstGoNodeOfType(root *sitter.Node, nodeType string) *sitter.Node {
	if root == nil {
		return nil
	}
	if root.Type() == nodeType {
		return root
	}
	for i := 0; i < int(root.ChildCount()); i++ {
		if result := findFirstGoNodeOfType(root.Child(i), nodeType); result != nil {
			return result
		}
	}
	return nil
}

func findGoNodeByName(root *sitter.Node, nodeType, name string, src []byte) *sitter.Node {
	if root == nil {
		return nil
	}
	if root.Type() == nodeType {
		nameNode := root.ChildByFieldName("name")
		if nameNode != nil && nameNode.Content(src) == name {
			return root
		}
	}
	for i := 0; i < int(root.ChildCount()); i++ {
		if result := findGoNodeByName(root.Child(i), nodeType, name, src); result != nil {
			return result
		}
	}
	return nil
}

func findGoNodeStartByte(root *sitter.Node, nodeType, name string, src []byte) uint32 {
	node := findGoNodeByName(root, nodeType, name, src)
	if node != nil {
		return node.StartByte()
	}
	return 0
}

func findGoNodeEndByte(root *sitter.Node, nodeType, name string, src []byte) uint32 {
	node := findGoNodeByName(root, nodeType, name, src)
	if node != nil {
		return node.EndByte()
	}
	return 0
}

// ========== splitGoTypeFQN tests ==========

func TestSplitGoTypeFQN(t *testing.T) {
	tests := []struct {
		name       string
		typeFQN    string
		wantImport string
		wantType   string
		wantOK     bool
	}{
		{"Standard stdlib type", "database/sql.DB", "database/sql", "DB", true},
		{"Net/http type", "net/http.Request", "net/http", "Request", true},
		{"Simple package", "fmt.Stringer", "fmt", "Stringer", true},
		{"os package", "os.File", "os", "File", true},
		{"Third-party deep path", "github.com/lib/pq.Connector", "github.com/lib/pq", "Connector", true},
		{"No dot — bare type", "error", "", "", false},
		{"Empty string", "", "", "", false},
		{"Trailing dot", "pkg.", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importPath, typeName, ok := splitGoTypeFQN(tt.typeFQN)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantImport, importPath)
				assert.Equal(t, tt.wantType, typeName)
			}
		})
	}
}
