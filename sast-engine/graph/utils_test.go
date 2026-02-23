package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func TestParseJavadocTagsComprehensive(t *testing.T) {
	tests := []struct {
		name                 string
		comment              string
		expectedAuthor       string
		expectedVersion      string
		expectedTagCount     int
		expectedCommentLines int
	}{
		{
			name: "Complete Javadoc",
			comment: `/**
 * This is a test class
 * @author John Doe
 * @version 1.0.0
 * @param name the parameter
 * @return the result
 * @throws IOException if error
 * @see OtherClass
 * @since 1.0
 */`,
			expectedAuthor:       "John Doe",
			expectedVersion:      "1.0.0",
			expectedTagCount:     7,
			expectedCommentLines: 10,
		},
		{
			name: "Minimal Javadoc",
			comment: `/**
 * Simple comment
 */`,
			expectedAuthor:       "",
			expectedVersion:      "",
			expectedTagCount:     0,
			expectedCommentLines: 3,
		},
		{
			name: "Multiple params",
			comment: `/**
 * @param x first param
 * @param y second param
 * @param z third param
 */`,
			expectedAuthor:       "",
			expectedVersion:      "",
			expectedTagCount:     3,
			expectedCommentLines: 5,
		},
		{
			name: "Unknown tags",
			comment: `/**
 * @deprecated use new method
 * @custom custom tag
 */`,
			expectedTagCount:     2,
			expectedCommentLines: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJavadocTags(tt.comment)

			if result == nil {
				t.Fatal("parseJavadocTags returned nil")
			}

			if result.Author != tt.expectedAuthor {
				t.Errorf("Expected author '%s', got '%s'", tt.expectedAuthor, result.Author)
			}

			if result.Version != tt.expectedVersion {
				t.Errorf("Expected version '%s', got '%s'", tt.expectedVersion, result.Version)
			}

			if len(result.Tags) != tt.expectedTagCount {
				t.Errorf("Expected %d tags, got %d", tt.expectedTagCount, len(result.Tags))
			}

			if result.NumberOfCommentLines != tt.expectedCommentLines {
				t.Errorf("Expected %d comment lines, got %d", tt.expectedCommentLines, result.NumberOfCommentLines)
			}

			if result.CommentedCodeElements != tt.comment {
				t.Error("CommentedCodeElements should match original comment")
			}
		})
	}
}

func TestExtractMethodNameComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedMethod string
		shouldHaveID   bool
	}{
		{
			name: "Simple method declaration",
			code: `public void testMethod() {
    System.out.println("test");
}`,
			expectedMethod: "testMethod",
			shouldHaveID:   true,
		},
		{
			name: "Method with parameters",
			code: `public String calculate(int x, String y) {
    return "result";
}`,
			expectedMethod: "calculate",
			shouldHaveID:   true,
		},
		{
			name: "Method with annotations",
			code: `@Override
public void toString() {
    return "test";
}`,
			expectedMethod: "toString",
			shouldHaveID:   true,
		},
		{
			name:           "Method invocation",
			code:           `System.out.println("hello")`,
			expectedMethod: "println",
			shouldHaveID:   true,
		},
		{
			name:           "Chained method invocation",
			code:           `object.getX().getY()`,
			expectedMethod: "getY",
			shouldHaveID:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(java.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()

			// Find the relevant node
			var targetNode *sitter.Node
			var findNode func(*sitter.Node)
			findNode = func(node *sitter.Node) {
				if node.Type() == "method_declaration" || node.Type() == "method_invocation" {
					targetNode = node
					return
				}
				for i := 0; i < int(node.ChildCount()); i++ {
					if targetNode == nil {
						findNode(node.Child(i))
					}
				}
			}
			findNode(root)

			if targetNode == nil {
				t.Skip("Could not find method node in parsed tree")
			}

			methodName, methodID := extractMethodName(targetNode, []byte(tt.code), "Test.java")

			if methodName != tt.expectedMethod {
				t.Errorf("Expected method name '%s', got '%s'", tt.expectedMethod, methodName)
			}

			if tt.shouldHaveID && methodID == "" {
				t.Error("Expected non-empty method ID")
			}

			if methodID != "" && len(methodID) != 64 {
				t.Errorf("Expected SHA256 hash length 64, got %d", len(methodID))
			}
		})
	}
}

func TestGetFilesComprehensive(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "test_getfiles")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := []struct {
		path        string
		shouldMatch bool
	}{
		{"Test1.java", true},
		{"Test2.java", true},
		{"script.py", true},
		{"README.md", false},
		{"config.json", false},
		{"app.js", false},
		{"subdir/Test3.java", true},
		{"subdir/script2.py", true},
		{"subdir/other.txt", false},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Test getFiles
	files, err := getFiles(tmpDir)
	if err != nil {
		t.Fatalf("getFiles failed: %v", err)
	}

	// Count expected files
	expectedCount := 0
	for _, tf := range testFiles {
		if tf.shouldMatch {
			expectedCount++
		}
	}

	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}

	// Verify only Java and Python files
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext != ".java" && ext != ".py" {
			t.Errorf("Unexpected file extension: %s", ext)
		}
	}
}

func TestGetFilesErrors(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		wantError bool
	}{
		{
			name:      "Non-existent directory",
			directory: "/path/that/does/not/exist/xyz123",
			wantError: true,
		},
		{
			name:      "Empty path",
			directory: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := getFiles(tt.directory)

			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}

			if err == nil && len(files) > 0 {
				t.Errorf("Expected empty files list for invalid directory, got %d files", len(files))
			}
		})
	}
}

func TestReadFileComprehensive(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "test_readfile_*.java")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := []byte("public class Test {\n    public static void main(String[] args) {}\n}")
	if _, err := tmpFile.Write(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	t.Run("Read existing file", func(t *testing.T) {
		content, err := readFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("readFile failed: %v", err)
		}

		if string(content) != string(testContent) {
			t.Error("Content mismatch")
		}
	})

	t.Run("Read non-existent file", func(t *testing.T) {
		_, err := readFile("/non/existent/file.java")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("Read empty file", func(t *testing.T) {
		emptyFile, err := os.CreateTemp("", "test_empty_*.java")
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}
		defer os.Remove(emptyFile.Name())
		emptyFile.Close()

		content, err := readFile(emptyFile.Name())
		if err != nil {
			t.Fatalf("readFile failed: %v", err)
		}

		if len(content) != 0 {
			t.Errorf("Expected empty content, got %d bytes", len(content))
		}
	})

	t.Run("Read large file", func(t *testing.T) {
		largeFile, err := os.CreateTemp("", "test_large_*.java")
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}
		defer os.Remove(largeFile.Name())

		// Write 1MB of data
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		largeFile.Write(largeContent)
		largeFile.Close()

		content, err := readFile(largeFile.Name())
		if err != nil {
			t.Fatalf("readFile failed: %v", err)
		}

		if len(content) != len(largeContent) {
			t.Errorf("Expected %d bytes, got %d", len(largeContent), len(content))
		}
	})
}

func TestHasAccessComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		variableName string
		expected     bool
	}{
		{
			name: "Variable exists in simple code",
			code: `public class Test {
    void method() {
        int x = 10;
        System.out.println(x);
    }
}`,
			variableName: "x",
			expected:     true,
		},
		{
			name: "Variable does not exist",
			code: `public class Test {
    void method() {
        int x = 10;
    }
}`,
			variableName: "y",
			expected:     false,
		},
		{
			name: "Variable in nested scope",
			code: `public class Test {
    void method() {
        if (true) {
            int nested = 5;
            System.out.println(nested);
        }
    }
}`,
			variableName: "nested",
			expected:     true,
		},
		{
			name: "Class name as variable",
			code: `public class Test {
    void method() {
        Test obj = new Test();
    }
}`,
			variableName: "Test",
			expected:     true,
		},
		{
			name:         "Null node",
			code:         "public class Test {}",
			variableName: "anything",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(java.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			result := hasAccess(root, tt.variableName, []byte(tt.code))

			if result != tt.expected {
				t.Errorf("Expected %v for variable '%s', got %v", tt.expected, tt.variableName, result)
			}
		})
	}

	// Test nil node explicitly
	t.Run("Nil node returns false", func(t *testing.T) {
		result := hasAccess(nil, "test", []byte("anything"))
		if result {
			t.Error("hasAccess with nil node should return false")
		}
	})
}

func TestAppendUniqueComprehensive(t *testing.T) {
	t.Run("Append to empty slice", func(t *testing.T) {
		var slice []*Node
		node := &Node{ID: "test1", Name: "Node1"}

		result := appendUnique(slice, node)

		if len(result) != 1 {
			t.Errorf("Expected length 1, got %d", len(result))
		}
		if result[0] != node {
			t.Error("Node not appended correctly")
		}
	})

	t.Run("Append unique nodes", func(t *testing.T) {
		node1 := &Node{ID: "test1", Name: "Node1"}
		node2 := &Node{ID: "test2", Name: "Node2"}
		node3 := &Node{ID: "test3", Name: "Node3"}

		slice := []*Node{node1}
		slice = appendUnique(slice, node2)
		slice = appendUnique(slice, node3)

		if len(slice) != 3 {
			t.Errorf("Expected length 3, got %d", len(slice))
		}
	})

	t.Run("Append duplicate node", func(t *testing.T) {
		node := &Node{ID: "test1", Name: "Node1"}
		slice := []*Node{node}

		result := appendUnique(slice, node)

		if len(result) != 1 {
			t.Errorf("Expected length 1 after duplicate, got %d", len(result))
		}
		if result[0] != node {
			t.Error("Node reference changed")
		}
	})

	t.Run("Multiple duplicates", func(t *testing.T) {
		node1 := &Node{ID: "test1"}
		node2 := &Node{ID: "test2"}

		slice := []*Node{node1, node2}
		slice = appendUnique(slice, node1)
		slice = appendUnique(slice, node2)
		slice = appendUnique(slice, node1)

		if len(slice) != 2 {
			t.Errorf("Expected length 2, got %d", len(slice))
		}
	})

	t.Run("Nil node", func(t *testing.T) {
		slice := []*Node{&Node{ID: "test1"}}
		result := appendUnique(slice, nil)

		if len(result) != 2 {
			t.Errorf("Expected length 2, got %d", len(result))
		}
	})
}

func TestFormatTypeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"String", "hello world", "hello world"},
		{"Empty string", "", ""},
		{"Int", 42, "42"},
		{"Int64", int64(9223372036854775807), "9223372036854775807"},
		{"Negative int", -100, "-100"},
		{"Float32", float32(3.14), "3.14"},
		{"Float64", 2.71828, "2.72"},
		{"Zero float", 0.0, "0.00"},
		{"Bool true", true, "true"},
		{"Bool false", false, "false"},
		{"Nil", nil, "<nil>"},
		{"Empty slice", []any{}, "[]"},
		{"Int slice", []any{1, 2, 3}, "[1,2,3]"},
		{"Mixed slice", []any{1, "two", 3.0}, "[1,\"two\",3]"},
		{"Nested slice", []any{[]any{1, 2}, []any{3, 4}}, "[[1,2],[3,4]]"},
		{"Struct", struct{ Name string }{"test"}, "{test}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestVerboseLoggingComprehensive(t *testing.T) {
	// Save original state
	originalVerbose := verboseFlag
	defer func() { verboseFlag = originalVerbose }()

	t.Run("EnableVerboseLogging sets flag", func(t *testing.T) {
		verboseFlag = false
		EnableVerboseLogging()
		if !verboseFlag {
			t.Error("verboseFlag should be true after EnableVerboseLogging")
		}
	})

	t.Run("Log when verbose enabled", func(t *testing.T) {
		verboseFlag = true
		// Should not panic
		Log("test message")
		Log("test with args: %s %d", "hello", 42)
	})

	t.Run("Log when verbose disabled", func(t *testing.T) {
		verboseFlag = false
		// Should not panic
		Log("this should not print")
	})

	t.Run("Fmt when verbose enabled", func(t *testing.T) {
		verboseFlag = true
		// Should not panic
		Fmt("test: %s\n", "hello")
		Fmt("numbers: %d %d\n", 1, 2)
	})

	t.Run("Fmt when verbose disabled", func(t *testing.T) {
		verboseFlag = false
		// Should not panic
		Fmt("this should not print\n")
	})
}

func TestIsGitHubActionsComprehensive(t *testing.T) {
	// Save original environment
	original := os.Getenv("GITHUB_ACTIONS")
	defer os.Setenv("GITHUB_ACTIONS", original)

	tests := []struct {
		name     string
		envValue string
		unset    bool
		expected bool
	}{
		{"Environment is 'true'", "true", false, true},
		{"Environment is 'false'", "false", false, false},
		{"Environment is '1'", "1", false, false},
		{"Environment is empty string", "", false, false},
		{"Environment is unset", "", true, false},
		{"Environment is 'True' (capitalized)", "True", false, false},
		{"Environment is 'TRUE'", "TRUE", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				os.Unsetenv("GITHUB_ACTIONS")
			} else {
				os.Setenv("GITHUB_ACTIONS", tt.envValue)
			}

			result := IsGitHubActions()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v (env=%q)", tt.expected, result, tt.envValue)
			}
		})
	}
}

func BenchmarkGenerateMethodID(b *testing.B) {
	params := []string{"int", "String", "Object"}
	for i := 0; i < b.N; i++ {
		GenerateMethodID("testMethod", params, "Test.java")
	}
}

func BenchmarkGenerateSha256(b *testing.B) {
	input := "test input for sha256 hashing"
	for i := 0; i < b.N; i++ {
		GenerateSha256(input)
	}
}

func BenchmarkParseJavadocTags(b *testing.B) {
	comment := `/**
 * Test method
 * @param x first parameter
 * @param y second parameter
 * @return result
 */`
	for i := 0; i < b.N; i++ {
		parseJavadocTags(comment)
	}
}

func BenchmarkHasAccess(b *testing.B) {
	code := `public class Test {
    void method() {
        int x = 10;
        System.out.println(x);
    }
}`
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	defer parser.Close()

	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(code))
	defer tree.Close()
	root := tree.RootNode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasAccess(root, "x", []byte(code))
	}
}
