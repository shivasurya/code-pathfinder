package graph

import (
	"context"
	"fmt"
	"github.com/shivasurya/code-pathfinder/sast-engine/model"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/python"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestNewCodeGraph(t *testing.T) {
	graph := NewCodeGraph()
	if graph == nil {
		t.Error("NewCodeGraph() returned nil")
	}
	if graph != nil && graph.Nodes == nil {
		t.Error("NewCodeGraph() returned graph with nil Nodes")
	}
	if graph != nil && graph.Edges == nil {
		t.Error("NewCodeGraph() returned graph with nil Edges")
	}
	if graph != nil && len(graph.Nodes) != 0 {
		t.Errorf("NewCodeGraph() returned graph with non-empty Nodes, got %d nodes", len(graph.Nodes))
	}
	if graph != nil && len(graph.Edges) != 0 {
		t.Errorf("NewCodeGraph() returned graph with non-empty Edges, got %d edges", len(graph.Edges))
	}
}

func TestAddNode(t *testing.T) {
	graph := NewCodeGraph()
	node := &Node{ID: "test_node"}
	graph.AddNode(node)

	if len(graph.Nodes) != 1 {
		t.Errorf("AddNode() failed to add node, expected 1 node, got %d", len(graph.Nodes))
	}
	if graph.Nodes["test_node"] != node {
		t.Error("AddNode() failed to add node with correct ID")
	}
}

func TestAddEdge(t *testing.T) {
	graph := NewCodeGraph()
	node1 := &Node{ID: "node1"}
	node2 := &Node{ID: "node2"}
	graph.AddNode(node1)
	graph.AddNode(node2)

	graph.AddEdge(node1, node2)

	if len(graph.Edges) != 1 {
		t.Errorf("AddEdge() failed to add edge, expected 1 edge, got %d", len(graph.Edges))
	}
	if graph.Edges[0].From != node1 || graph.Edges[0].To != node2 {
		t.Error("AddEdge() failed to add edge with correct From and To nodes")
	}
	if len(node1.OutgoingEdges) != 1 {
		t.Errorf("AddEdge() failed to add outgoing edge to From node, expected 1 edge, got %d", len(node1.OutgoingEdges))
	}
	if node1.OutgoingEdges[0].To != node2 {
		t.Error("AddEdge() failed to add correct outgoing edge to From node")
	}
}

func TestAddMultipleNodesAndEdges(t *testing.T) {
	graph := NewCodeGraph()
	node1 := &Node{ID: "node1"}
	node2 := &Node{ID: "node2"}
	node3 := &Node{ID: "node3"}

	graph.AddNode(node1)
	graph.AddNode(node2)
	graph.AddNode(node3)

	graph.AddEdge(node1, node2)
	graph.AddEdge(node2, node3)
	graph.AddEdge(node1, node3)

	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(graph.Edges))
	}
	if len(node1.OutgoingEdges) != 2 {
		t.Errorf("Expected 2 outgoing edges for node1, got %d", len(node1.OutgoingEdges))
	}
	if len(node2.OutgoingEdges) != 1 {
		t.Errorf("Expected 1 outgoing edge for node2, got %d", len(node2.OutgoingEdges))
	}
	if len(node3.OutgoingEdges) != 0 {
		t.Errorf("Expected 0 outgoing edges for node3, got %d", len(node3.OutgoingEdges))
	}
}

func TestIsJavaSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Valid Java file", "Example.java", true},
		{"Lowercase extension", "example.java", true},
		{"Non-Java file", "example.txt", false},
		{"No extension", "javafile", false},
		{"Empty string", "", false},
		{"Java file with path", "/path/to/Example.java", true},
		{"Java file with Windows path", "C:\\path\\to\\Example.java", true},
		{"File with multiple dots", "my.test.file.java", true},
		{"Hidden Java file", ".hidden.java", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJavaSourceFile(tt.filename); got != tt.want {
				t.Errorf("isJavaSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
func TestParseJavadocTags(t *testing.T) {
	tests := []struct {
		name           string
		commentContent string
		want           *model.Javadoc
	}{
		{
			name: "Multi-line comment with various tags",
			commentContent: `/**
 * This is a multi-line comment
 * @author John Doe
 * @param input The input string
 * @throws IllegalArgumentException if input is null
 * @see SomeOtherClass
 * @version 1.0
 * @since 2021-01-01
 */`,
			want: &model.Javadoc{
				NumberOfCommentLines: 9,
				CommentedCodeElements: `/**
 * This is a multi-line comment
 * @author John Doe
 * @param input The input string
 * @throws IllegalArgumentException if input is null
 * @see SomeOtherClass
 * @version 1.0
 * @since 2021-01-01
 */`,
				Author:  "John Doe",
				Version: "1.0",
				Tags: []*model.JavadocTag{
					model.NewJavadocTag("author", "John Doe", "author"),
					model.NewJavadocTag("param", "input The input string", "param"),
					model.NewJavadocTag("throws", "IllegalArgumentException if input is null", "throws"),
					model.NewJavadocTag("see", "SomeOtherClass", "see"),
					model.NewJavadocTag("version", "1.0", "version"),
					model.NewJavadocTag("since", "2021-01-01", "since"),
				},
			},
		},
		{
			name: "Comment with unknown tag",
			commentContent: `/**
 * @customTag This is a custom tag
 */`,
			want: &model.Javadoc{
				NumberOfCommentLines: 3,
				CommentedCodeElements: `/**
 * @customTag This is a custom tag
 */`,
				Tags: []*model.JavadocTag{
					model.NewJavadocTag("customTag", "This is a custom tag", "unknown"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJavadocTags(tt.commentContent)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJavadocTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
		isJava  bool
	}{
		{"file1.java", "Java content", true},
		{"file2.txt", "Text content", false},
		{"file3.java", "Another Java file", true},
		{"subdir/file4.java", "Nested Java file", true},
		{"file5", "No extension file", false},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(path, []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Run getFiles
	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	// Check results
	expectedJavaFiles := 3
	if len(files) != expectedJavaFiles {
		t.Errorf("Expected %d Java files, but got %d", expectedJavaFiles, len(files))
	}

	for _, file := range files {
		if filepath.Ext(file) != ".java" {
			t.Errorf("Non-Java file found: %s", file)
		}
	}

	// Check if nested file is included
	nestedFile := filepath.Join(tempDir, "subdir", "file4.java")
	found := slices.Contains(files, nestedFile)
	if !found {
		t.Errorf("Nested Java file not found in results")
	}
}

func TestGetFilesEmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_empty")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty directory, but got %d", len(files))
	}
}

func TestGetFilesNonExistentDirectory(t *testing.T) {
	nonExistentDir := "/path/to/non/existent/directory"
	_, err := getFiles(nonExistentDir)
	if err == nil {
		t.Error("Expected an error for non-existent directory, but got nil")
	}
}

func TestGetFilesWithSymlinks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_symlinks")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a Java file
	javaFile := filepath.Join(tempDir, "original.java")
	err = os.WriteFile(javaFile, []byte("Java content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a symlink to the Java file
	symlinkFile := filepath.Join(tempDir, "symlink.java")
	err = os.Symlink(javaFile, symlinkFile)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 Java files (original + symlink), but got %d", len(files))
	}

	foundOriginal := false
	foundSymlink := false
	for _, file := range files {
		if file == javaFile {
			foundOriginal = true
		}
		if file == symlinkFile {
			foundSymlink = true
		}
	}

	if !foundOriginal {
		t.Error("Original Java file not found in results")
	}
	if !foundSymlink {
		t.Error("Symlinked Java file not found in results")
	}
}

func TestFindNodesByType(t *testing.T) {
	graph := NewCodeGraph()
	node1 := &Node{ID: "node1", Type: "class"}
	node2 := &Node{ID: "node2", Type: "method"}
	node3 := &Node{ID: "node3", Type: "class"}
	node4 := &Node{ID: "node4", Type: "interface"}
	node5 := &Node{ID: "node5", Type: "method"}

	graph.AddNode(node1)
	graph.AddNode(node2)
	graph.AddNode(node3)
	graph.AddNode(node4)
	graph.AddNode(node5)

	tests := []struct {
		name     string
		nodeType string
		want     int
	}{
		{"Find class nodes", "class", 2},
		{"Find method nodes", "method", 2},
		{"Find interface nodes", "interface", 1},
		{"Find non-existent node type", "enum", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := graph.FindNodesByType(tt.nodeType)
			if len(nodes) != tt.want {
				t.Errorf("FindNodesByType(%q) returned %d nodes, want %d", tt.nodeType, len(nodes), tt.want)
			}
			for _, node := range nodes {
				if node.Type != tt.nodeType {
					t.Errorf("FindNodesByType(%q) returned node with type %q, want %q", tt.nodeType, node.Type, tt.nodeType)
				}
			}
		})
	}
}

func TestFindNodesByTypeEmptyGraph(t *testing.T) {
	graph := NewCodeGraph()
	nodes := graph.FindNodesByType("class")
	if len(nodes) != 0 {
		t.Errorf("FindNodesByType on empty graph returned %d nodes, want 0", len(nodes))
	}
}

func TestFindNodesByTypeAllSameType(t *testing.T) {
	graph := NewCodeGraph()
	for i := range 5 {
		graph.AddNode(&Node{ID: fmt.Sprintf("node%d", i), Type: "class"})
	}

	nodes := graph.FindNodesByType("class")
	if len(nodes) != 5 {
		t.Errorf("FindNodesByType('class') returned %d nodes, want 5", len(nodes))
	}
}

func TestFindNodesByTypeCaseSensitivity(t *testing.T) {
	graph := NewCodeGraph()
	graph.AddNode(&Node{ID: "node1", Type: "Class"})
	graph.AddNode(&Node{ID: "node2", Type: "class"})

	upperNodes := graph.FindNodesByType("Class")
	lowerNodes := graph.FindNodesByType("class")

	if len(upperNodes) != 1 || len(lowerNodes) != 1 {
		t.Errorf("FindNodesByType is not case-sensitive: 'Class' returned %d, 'class' returned %d", len(upperNodes), len(lowerNodes))
	}
}

func TestExtractVisibilityModifier(t *testing.T) {
	tests := []struct {
		name      string
		modifiers string
		want      string
	}{
		{"Public modifier", "public static final", "public"},
		{"Private modifier", "private volatile", "private"},
		{"Protected modifier", "protected transient", "protected"},
		{"No visibility modifier", "static final", ""},
		{"Multiple modifiers", "static public final", "public"},
		{"Empty string", "", ""},
		{"Only non-visibility modifiers", "static final transient", ""},
		{"Mixed case modifiers", "Static Public Final", ""},
		{"Visibility modifier in the middle", "static public final", "public"},
		{"Multiple visibility modifiers", "public private protected", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVisibilityModifier(tt.modifiers)
			if got != tt.want {
				t.Errorf("extractVisibilityModifier(%q) = %v, want %v", tt.modifiers, got, tt.want)
			}
		})
	}
}

func TestExtractVisibilityModifierWithLeadingTrailingSpaces(t *testing.T) {
	tests := []struct {
		name      string
		modifiers string
		want      string
	}{
		{"Leading spaces", "  public static", "public"},
		{"Trailing spaces", "private final  ", "private"},
		{"Leading and trailing spaces", "  protected   ", "protected"},
		{"Multiple spaces between modifiers", "static   public   final", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVisibilityModifier(tt.modifiers)
			if got != tt.want {
				t.Errorf("extractVisibilityModifier(%q) = %v, want %v", tt.modifiers, got, tt.want)
			}
		})
	}
}

func TestExtractVisibilityModifierWithInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		modifiers string
		want      string
	}{
		{"Numbers only", "123 456", ""},
		{"Special characters", "@#$%^&*", ""},
		{"Similar words", "publicly privateer protect", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVisibilityModifier(tt.modifiers)
			if got != tt.want {
				t.Errorf("extractVisibilityModifier(%q) = %v, want %v", tt.modifiers, got, tt.want)
			}
		})
	}
}

func TestInitialize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_initialize")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"File1.java", "public class File1 { }"},
		{"File2.java", "public class File2 { }"},
		{"subdir/File3.java", "public class File3 { }"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(path, []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	graph := Initialize(tempDir, nil)

	if graph == nil {
		t.Fatal("Initialize returned nil graph")
	}

	expectedNodeCount := 3 // One for each file
	if len(graph.Nodes) != expectedNodeCount {
		t.Errorf("Expected %d nodes, but got %d", expectedNodeCount, len(graph.Nodes))
	}

	nodeTypes := map[string]int{"class": 0, "interface": 0, "enum": 0}
	for _, node := range graph.Nodes {
		nodeTypes[node.Type]++
	}

	if nodeTypes["class_declaration"] != 3 {
		t.Errorf("Unexpected node type distribution: %v", nodeTypes)
	}
}

func TestInitializeEmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_initialize_empty")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	graph := Initialize(tempDir, nil)

	if graph == nil {
		t.Fatal("Initialize returned nil graph for empty directory")
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for empty directory, but got %d", len(graph.Nodes))
	}

	if len(graph.Edges) != 0 {
		t.Errorf("Expected 0 edges for empty directory, but got %d", len(graph.Edges))
	}
}

func TestInitializeNonExistentDirectory(t *testing.T) {
	nonExistentDir := "/path/to/non/existent/directory"
	graph := Initialize(nonExistentDir, nil)

	if graph == nil {
		t.Fatal("Initialize returned nil graph for non-existent directory")
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for non-existent directory, but got %d", len(graph.Nodes))
	}

	if len(graph.Edges) != 0 {
		t.Errorf("Expected 0 edges for non-existent directory, but got %d", len(graph.Edges))
	}
}

func TestInitializeWithNonJavaFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_initialize_non_java")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"File1.java", "public class File1 { }"},
		{"File2.txt", "This is a text file"},
		{"File3.cpp", "int main() { return 0; }"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(path, []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	graph := Initialize(tempDir, nil)

	if graph == nil {
		t.Fatal("Initialize returned nil graph")
	}

	expectedNodeCount := 1 // Only one Java file
	if len(graph.Nodes) != expectedNodeCount {
		t.Errorf("Expected %d node, but got %d", expectedNodeCount, len(graph.Nodes))
	}

	for _, node := range graph.Nodes {
		if node.Type != "class_declaration" {
			t.Errorf("Expected node type to be 'class', but got '%s'", node.Type)
		}
	}
}

func TestInitializeWithLargeNumberOfFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_initialize_large")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a large number of test files
	numFiles := 100
	for i := range numFiles {
		fileName := fmt.Sprintf("File%d.java", i)
		content := fmt.Sprintf("public class File%d { }", i)
		path := filepath.Join(tempDir, fileName)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	graph := Initialize(tempDir, nil)

	if graph == nil {
		t.Fatal("Initialize returned nil graph")
	}

	if len(graph.Nodes) != numFiles {
		t.Errorf("Expected %d nodes, but got %d", numFiles, len(graph.Nodes))
	}

	for _, node := range graph.Nodes {
		if node.Type != "class_declaration" {
			t.Errorf("Expected node type to be 'class_declaration', but got '%s'", node.Type)
		}
	}
}

func TestReadFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_read_file")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		content  string
		wantErr  bool
		expected string
	}{
		{"Valid file", "Hello, World!", false, "Hello, World!"},
		{"Empty file", "", false, ""},
		{"File with special characters", "!@#$%^&*()", false, "!@#$%^&*()"},
		{"File with multiple lines", "Line 1\nLine 2\nLine 3", false, "Line 1\nLine 2\nLine 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "testfile.txt")
			if !tt.wantErr {
				err := os.WriteFile(filePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			got, err := readFile(filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("readFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.expected {
				t.Errorf("readFile() = %v, want %v", string(got), tt.expected)
			}
		})
	}
}

func TestBuildGraphFromAST(t *testing.T) {
	tests := []struct {
		name            string
		sourceCode      string
		expectedNodes   int
		expectedEdges   int
		expectedTypes   []string
		unexpectedTypes []string
	}{
		{
			name: "Simple class with method",
			sourceCode: `
				public class SimpleClass {
					public void simpleMethod() {
						int x = 5;
					}
				}
			`,
			expectedNodes:   4,
			expectedEdges:   0,
			expectedTypes:   []string{"class_declaration", "method_declaration", "variable_declaration", "BlockStmt"},
			unexpectedTypes: []string{"method_invocation"},
		},
		{
			name: "Class with method invocation",
			sourceCode: `
				public class InvocationClass {
					public void caller() {
						callee();
					}
					private void callee() {
					  fmt.Println("Hello, World!");
					}
				}
			`,
			expectedNodes:   7,
			expectedEdges:   2,
			expectedTypes:   []string{"class_declaration", "method_declaration", "method_invocation", "BlockStmt"},
			unexpectedTypes: []string{"variable_declaration"},
		},
		{
			name: "Class with binary expression",
			sourceCode: `
				public class BinaryExprClass {
					public int add() {
						return 5 + 3;
					}
				}
			`,
			expectedNodes:   6,
			expectedEdges:   0,
			expectedTypes:   []string{"class_declaration", "method_declaration", "binary_expression", "ReturnStmt"},
			unexpectedTypes: []string{"variable_declaration"},
		},
		{
			name: "Class with multiple binary expressions",
			sourceCode: `
				public class MultiBinaryExprClass {
					public boolean complex() {
						int a = 5 - 1;
						int b = 20 / 2;
						boolean c = 20 == 2;
                        int d = 1 * 2;
						int e = 10 % 3;
						int f = 10 >> 3;
						int g = 10 << 3;
						int h = 1 & 1;
                        int i = 1 | 1;
                        int j = 1 ^ 1;
                        int l = 1 >>> 1;
                        outerlabel:
					    while (a > 0) {
							a--;
                            if (a == 0) {
								break outerlabel;
							} else {
								continue outerlabel;
							}
						}
                        for (int i = 0; i < 10; i++) {
							System.out.println(i);
                             break;
						}
						switch (day) {
									case "MONDAY" -> 1;
									case "TUESDAY" -> 2;
									case "WEDNESDAY" -> 3;
									case "THURSDAY" -> 4;
									case "FRIDAY" -> 5;
									case "SATURDAY" -> 6;
									case "SUNDAY" -> 7;
									default -> {
										System.out.println("Invalid day: " + day);
										yield 9;  // Using 'yield' to return a value from this case
									}
						};
						do {
							System.out.println("Hello, World!");
						} while (a > 0);
		                if (a < 0) {
							System.out.println("Negative number");
						} else {
							System.out.println("Positive number");	
						}
						return (5 > 3) && (10 <= 20) || (15 != 12) || (20 > 15);
					}
				}
			`,
			expectedNodes:   83,
			expectedEdges:   5,
			expectedTypes:   []string{"class_declaration", "method_declaration", "binary_expression", "comp_expression", "and_expression", "or_expression", "IfStmt", "ForStmt", "WhileStmt", "DoStmt", "BreakStmt", "ContinueStmt", "YieldStmt", "ReturnStmt", "BlockStmt"},
			unexpectedTypes: []string{""},
		},
		{
			name: "Class with Javadoc and annotations",
			sourceCode: `
				/**
				 * @author John Doe
				 * @version 1.0
				 */
				@Deprecated
				public class AnnotatedClass {
					@Override
					public String toString() {
						return "AnnotatedClass";
					}
				}
			`,
			expectedNodes:   5,
			expectedEdges:   0,
			expectedTypes:   []string{"class_declaration", "method_declaration", "block_comment", "ReturnStmt"},
			unexpectedTypes: []string{"variable_declaration", "binary_expression"},
		},
		// add testcase for object creation expression
		{
			name: "Class with object creation expression",
			sourceCode: `
				public class ObjectCreationClass {
					public static void main(String[] args) {
						ObjectCreationClass obj = new ObjectCreationClass();
						Socket socket = new Socket("www.google.com", 80);
					}
				}
			`,
			expectedNodes:   7,
			expectedEdges:   0,
			expectedTypes:   []string{"class_declaration", "method_declaration", "ClassInstanceExpr"},
			unexpectedTypes: []string{"binary_expression"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(java.GetLanguage())
			tree, err := parser.ParseCtx(context.TODO(), nil, []byte(tt.sourceCode))
			if err != nil {
				t.Fatalf("Failed to parse source code: %v", err)
			}
			root := tree.RootNode()

			graph := NewCodeGraph()
			buildGraphFromAST(root, []byte(tt.sourceCode), graph, nil, "test.java")

			if len(graph.Nodes) != tt.expectedNodes {
				t.Errorf("Expected %d nodes, but got %d", tt.expectedNodes, len(graph.Nodes))
			}

			if len(graph.Edges) != tt.expectedEdges {
				t.Errorf("Expected %d edges, but got %d", tt.expectedEdges, len(graph.Edges))
			}

			nodeTypes := make(map[string]bool)
			for _, node := range graph.Nodes {
				nodeTypes[node.Type] = true
			}

			for _, expectedType := range tt.expectedTypes {
				if !nodeTypes[expectedType] {
					t.Errorf("Expected node type %s not found", expectedType)
				}
			}

			for _, unexpectedType := range tt.unexpectedTypes {
				if nodeTypes[unexpectedType] {
					t.Errorf("Unexpected node type %s found", unexpectedType)
				}
			}
		})
	}
}

func TestExtractMethodName(t *testing.T) {
	tests := []struct {
		name         string
		sourceCode   string
		expectedName string
	}{
		{
			name:         "Simple method",
			sourceCode:   "public void simpleMethod() {}",
			expectedName: "simpleMethod",
		},
		{
			name:         "Method with parameters",
			sourceCode:   "private int complexMethod(String a, int b) {}",
			expectedName: "complexMethod",
		},
		{
			name:         "Generic method",
			sourceCode:   "public <T> List<T> genericMethod(T item) {}",
			expectedName: "genericMethod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(java.GetLanguage())
			tree, err := parser.ParseCtx(context.TODO(), nil, []byte(tt.sourceCode))
			if err != nil {
				t.Fatalf("Failed to parse source code: %v", err)
			}
			root := tree.RootNode()

			methodNode := root.NamedChild(0)
			name, id := extractMethodName(methodNode, []byte(tt.sourceCode), "test.java")

			if name != tt.expectedName {
				t.Errorf("Expected method name %s, but got %s", tt.expectedName, name)
			}

			// Verify ID is non-empty and contains the method name (with prefix)
			if id == "" {
				t.Error("Expected non-empty method ID")
			}

			// Method declarations should have IDs prefixed with method:
			if methodNode.Type() == "method_declaration" {
				// The ID is a hash, but we can verify it was generated (non-empty)
				if len(id) != 64 {
					t.Errorf("Expected method ID to be SHA256 hash (64 chars), got length %d", len(id))
				}
			}
		})
	}
}

// Python-specific tests

func TestIsPythonSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Valid Python file", "example.py", true},
		{"Python file with path", "/path/to/script.py", true},
		{"Python file with Windows path", "C:\\path\\to\\script.py", true},
		{"File with multiple dots", "my.test.script.py", true},
		{"Hidden Python file", ".hidden.py", true},
		{"Non-Python file", "example.txt", false},
		{"Java file", "Example.java", false},
		{"No extension", "pythonfile", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPythonSourceFile(tt.filename); got != tt.want {
				t.Errorf("isPythonSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestGetFilesMixedLanguages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_mixed")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []struct {
		name       string
		content    string
		shouldFind bool
	}{
		{"file1.py", "print('Hello')", true},
		{"file2.txt", "Text content", false},
		{"file3.py", "def func(): pass", true},
		{"subdir/file4.py", "class Test: pass", true},
		{"file5", "No extension file", false},
		{"test.java", "public class Test {}", true},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(path, []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Run getFiles
	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	// Check that both .py and .java files are found
	expectedFiles := 4 // 3 Python + 1 Java
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, but got %d", expectedFiles, len(files))
	}

	// Verify file extensions
	pythonCount := 0
	javaCount := 0
	for _, file := range files {
		ext := filepath.Ext(file)
		switch ext {
		case ".py":
			pythonCount++
		case ".java":
			javaCount++
		default:
			t.Errorf("Unexpected file extension: %s", ext)
		}
	}

	if pythonCount != 3 {
		t.Errorf("Expected 3 Python files, got %d", pythonCount)
	}
	if javaCount != 1 {
		t.Errorf("Expected 1 Java file, got %d", javaCount)
	}
}

func TestBuildGraphFromASTPythonFunctionDefinition(t *testing.T) {
	tests := []struct {
		name              string
		sourceCode        string
		expectedNodeCount int
		expectedName      string
		expectedParams    []string
	}{
		{
			name: "Simple function without parameters",
			sourceCode: `def simple_func():
    pass`,
			expectedNodeCount: 1,
			expectedName:      "simple_func",
			expectedParams:    []string{},
		},
		{
			name: "Function with parameters",
			sourceCode: `def add(x, y):
    return x + y`,
			expectedNodeCount: 2, // function + return
			expectedName:      "add",
			expectedParams:    []string{"x", "y"},
		},
		{
			name: "Method with self parameter",
			sourceCode: `def method(self, arg1, arg2):
    self.value = arg1`,
			expectedNodeCount: 2, // function + assignment
			expectedName:      "method",
			expectedParams:    []string{"self", "arg1", "arg2"},
		},
		{
			name: "Function with default parameters",
			sourceCode: `def func_with_defaults(x, y=10, z=20):
    return x + y + z`,
			expectedNodeCount: 2, // function + return
			expectedName:      "func_with_defaults",
			expectedParams:    []string{"x", "y=10", "z=20"}, // Parser captures default values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(python.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.sourceCode))
			if err != nil {
				t.Fatalf("Failed to parse Python source code: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			graph := NewCodeGraph()
			buildGraphFromAST(root, []byte(tt.sourceCode), graph, nil, "test.py")

			if len(graph.Nodes) < tt.expectedNodeCount {
				t.Errorf("Expected at least %d nodes, but got %d", tt.expectedNodeCount, len(graph.Nodes))
			}

			// Find the function_definition node
			var funcNode *Node
			for _, node := range graph.Nodes {
				if node.Type == "function_definition" {
					funcNode = node
					break
				}
			}

			if funcNode == nil {
				t.Fatal("No function_definition node found")
			}

			if funcNode.Name != tt.expectedName {
				t.Errorf("Expected function name %s, got %s", tt.expectedName, funcNode.Name)
			}

			if !funcNode.isPythonSourceFile {
				t.Error("Expected isPythonSourceFile to be true")
			}

			if len(tt.expectedParams) > 0 {
				if len(funcNode.MethodArgumentsValue) != len(tt.expectedParams) {
					t.Errorf("Expected %d parameters, got %d", len(tt.expectedParams), len(funcNode.MethodArgumentsValue))
				}
				for i, param := range tt.expectedParams {
					if i < len(funcNode.MethodArgumentsValue) && funcNode.MethodArgumentsValue[i] != param {
						t.Errorf("Expected parameter %d to be %s, got %s", i, param, funcNode.MethodArgumentsValue[i])
					}
				}
			}
		})
	}
}

func TestBuildGraphFromASTPythonClassDefinition(t *testing.T) {
	tests := []struct {
		name              string
		sourceCode        string
		expectedNodeCount int
		expectedClassName string
		expectedBases     []string
	}{
		{
			name: "Simple class without base",
			sourceCode: `class SimpleClass:
    pass`,
			expectedNodeCount: 1,
			expectedClassName: "SimpleClass",
			expectedBases:     []string{},
		},
		{
			name: "Class with single base",
			sourceCode: `class Derived(Base):
    pass`,
			expectedNodeCount: 1,
			expectedClassName: "Derived",
			expectedBases:     []string{"Base"},
		},
		{
			name: "Class with multiple bases",
			sourceCode: `class MultiDerived(Base1, Base2, Base3):
    pass`,
			expectedNodeCount: 1,
			expectedClassName: "MultiDerived",
			expectedBases:     []string{"Base1", "Base2", "Base3"},
		},
		{
			name: "Class with method",
			sourceCode: `class MyClass:
    def my_method(self):
        return 42`,
			expectedNodeCount: 3, // class + method + return
			expectedClassName: "MyClass",
			expectedBases:     []string{},
		},
		{
			name: "Class with __init__ method",
			sourceCode: `class Person:
    def __init__(self, name):
        self.name = name`,
			expectedNodeCount: 3, // class + __init__ + assignment
			expectedClassName: "Person",
			expectedBases:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(python.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.sourceCode))
			if err != nil {
				t.Fatalf("Failed to parse Python source code: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			graph := NewCodeGraph()
			buildGraphFromAST(root, []byte(tt.sourceCode), graph, nil, "test.py")

			if len(graph.Nodes) < tt.expectedNodeCount {
				t.Errorf("Expected at least %d nodes, but got %d", tt.expectedNodeCount, len(graph.Nodes))
			}

			// Find the class_definition node
			var classNode *Node
			for _, node := range graph.Nodes {
				if node.Type == "class_definition" {
					classNode = node
					break
				}
			}

			if classNode == nil {
				t.Fatal("No class_definition node found")
			}

			if classNode.Name != tt.expectedClassName {
				t.Errorf("Expected class name %s, got %s", tt.expectedClassName, classNode.Name)
			}

			if !classNode.isPythonSourceFile {
				t.Error("Expected isPythonSourceFile to be true")
			}

			if len(tt.expectedBases) > 0 {
				if len(classNode.Interface) != len(tt.expectedBases) {
					t.Errorf("Expected %d base classes, got %d", len(tt.expectedBases), len(classNode.Interface))
				}
				for i, base := range tt.expectedBases {
					if i < len(classNode.Interface) && classNode.Interface[i] != base {
						t.Errorf("Expected base class %d to be %s, got %s", i, base, classNode.Interface[i])
					}
				}
			} else if len(classNode.Interface) != 0 {
				t.Errorf("Expected no base classes, got %d", len(classNode.Interface))
			}
		})
	}
}

func TestBuildGraphFromASTPythonStatements(t *testing.T) {
	tests := []struct {
		name          string
		sourceCode    string
		expectedTypes []string
		minNodeCount  int
	}{
		{
			name: "Function with return statement",
			sourceCode: `def get_value():
    return 42`,
			expectedTypes: []string{"function_definition", "ReturnStmt"},
			minNodeCount:  2,
		},
		{
			name: "Function with assert statement",
			sourceCode: `def validate(x):
    assert x > 0, "must be positive"`,
			expectedTypes: []string{"function_definition", "AssertStmt"},
			minNodeCount:  2,
		},
		{
			name: "Function with break and continue",
			sourceCode: `def loop():
    for i in range(10):
        if i == 5:
            break
        if i == 3:
            continue`,
			expectedTypes: []string{"function_definition", "BreakStmt", "ContinueStmt"},
			minNodeCount:  3,
		},
		{
			name: "Generator function with yield",
			sourceCode: `def gen():
    yield 1
    yield 2`,
			expectedTypes: []string{"function_definition", "YieldStmt"},
			minNodeCount:  2,
		},
		{
			name: "Function with variable assignment",
			sourceCode: `def compute():
    result = 10 + 20
    return result`,
			expectedTypes: []string{"function_definition", "variable_assignment", "ReturnStmt"},
			minNodeCount:  3,
		},
		{
			name: "Function with function calls",
			sourceCode: `def caller():
    print("hello")
    other_func()`,
			expectedTypes: []string{"function_definition", "call"},
			minNodeCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(python.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.sourceCode))
			if err != nil {
				t.Fatalf("Failed to parse Python source code: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			graph := NewCodeGraph()
			buildGraphFromAST(root, []byte(tt.sourceCode), graph, nil, "test.py")

			if len(graph.Nodes) < tt.minNodeCount {
				t.Errorf("Expected at least %d nodes, but got %d", tt.minNodeCount, len(graph.Nodes))
			}

			// Verify expected node types exist
			nodeTypes := make(map[string]bool)
			pythonSpecificTypes := map[string]bool{
				"function_definition": true,
				"class_definition":    true,
				"call":                true,
				"variable_assignment": true,
				"ReturnStmt":          true,
				"AssertStmt":          true,
				"BreakStmt":           true,
				"ContinueStmt":        true,
				"YieldStmt":           true,
			}

			for _, node := range graph.Nodes {
				nodeTypes[node.Type] = true

				// Python-specific nodes should be marked as Python
				if pythonSpecificTypes[node.Type] && !node.isPythonSourceFile {
					t.Errorf("Node %s (type: %s) should have isPythonSourceFile=true", node.ID, node.Type)
				}
			}

			for _, expectedType := range tt.expectedTypes {
				if !nodeTypes[expectedType] {
					t.Errorf("Expected node type %s not found. Found types: %v", expectedType, getKeys(nodeTypes))
				}
			}
		})
	}
}

// Helper function to get map keys.
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Go-specific tests

func TestIsGoSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Valid Go file", "main.go", true},
		{"Go file with path", "/path/to/main.go", true},
		{"Go file with Windows path", "C:\\path\\to\\main.go", true},
		{"File with multiple dots", "my.test.file.go", true},
		{"Hidden Go file", ".hidden.go", true},
		{"Go test file", "main_test.go", true},
		{"Non-Go file", "main.py", false},
		{"Java file", "Example.java", false},
		{"No extension", "gofile", false},
		{"Empty string", "", false},
		{"Go-like extension", "file.goo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGoSourceFile(tt.filename); got != tt.want {
				t.Errorf("isGoSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestGetFilesIncludesGo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_go")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files of various types
	testFiles := []struct {
		name       string
		content    string
		shouldFind bool
	}{
		{"main.go", "package main", true},
		{"handler.go", "package handlers", true},
		{"script.py", "print('hello')", true},
		{"Test.java", "public class Test {}", true},
		{"readme.txt", "text file", false},
		{"subdir/util.go", "package subdir", true},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	// Count files by extension
	goCount := 0
	pyCount := 0
	javaCount := 0
	for _, file := range files {
		switch filepath.Ext(file) {
		case ".go":
			goCount++
		case ".py":
			pyCount++
		case ".java":
			javaCount++
		}
	}

	if goCount != 3 {
		t.Errorf("Expected 3 Go files, got %d", goCount)
	}
	if pyCount != 1 {
		t.Errorf("Expected 1 Python file, got %d", pyCount)
	}
	if javaCount != 1 {
		t.Errorf("Expected 1 Java file, got %d", javaCount)
	}
	if len(files) != 5 {
		t.Errorf("Expected 5 total files, got %d", len(files))
	}
}

func TestGetFilesSkipsVendor(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_vendor")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a Go file at the root and inside vendor/
	rootFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(rootFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	vendorDir := filepath.Join(tempDir, "vendor", "github.com", "dep")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}
	vendorFile := filepath.Join(vendorDir, "dep.go")
	if err := os.WriteFile(vendorFile, []byte("package dep"), 0644); err != nil {
		t.Fatalf("Failed to create vendor file: %v", err)
	}

	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file (vendor/ should be skipped), got %d: %v", len(files), files)
	}
}

func TestGetFilesSkipsTestdata(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_testdata")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rootFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(rootFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	testdataDir := filepath.Join(tempDir, "testdata")
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata dir: %v", err)
	}
	testdataFile := filepath.Join(testdataDir, "fixture.go")
	if err := os.WriteFile(testdataFile, []byte("package testdata"), 0644); err != nil {
		t.Fatalf("Failed to create testdata file: %v", err)
	}

	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file (testdata/ should be skipped), got %d: %v", len(files), files)
	}
}

func TestGetFilesSkipsUnderscoreDirs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_get_files_underscore")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rootFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(rootFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	underscoreDir := filepath.Join(tempDir, "_build")
	if err := os.MkdirAll(underscoreDir, 0755); err != nil {
		t.Fatalf("Failed to create underscore dir: %v", err)
	}
	underscoreFile := filepath.Join(underscoreDir, "output.go")
	if err := os.WriteFile(underscoreFile, []byte("package build"), 0644); err != nil {
		t.Fatalf("Failed to create underscore dir file: %v", err)
	}

	files, err := getFiles(tempDir)
	if err != nil {
		t.Fatalf("getFiles returned an error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file (_build/ should be skipped), got %d: %v", len(files), files)
	}
}

func TestInitializeWithGoFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_go_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	goCode := `package main

import "fmt"

const MaxRetries = 3

var version = "1.0"

func main() {
	fmt.Println("hello")
}

func add(x, y int) int {
	return x + y
}
`
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("Failed to write Go file: %v", err)
	}

	graph := Initialize(tmpDir, nil)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	// PR-01 only verifies that parsing doesn't panic.
	// No Go-specific nodes are extracted yet (that's PR-03+).
	// The graph should exist but may be empty since stubs don't create nodes.
}

func TestInitializeWithGoFileDoesNotBreakPython(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_mixed_go_py")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Python file
	pyCode := `def hello():
    return "world"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "script.py"), []byte(pyCode), 0644); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	// Create a Go file
	goCode := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goCode), 0644); err != nil {
		t.Fatalf("Failed to write Go file: %v", err)
	}

	graph := Initialize(tmpDir, nil)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	// Verify Python function was still extracted correctly
	var foundPythonFunc bool
	for _, node := range graph.Nodes {
		if node.Type == "function_definition" && node.Name == "hello" && node.isPythonSourceFile {
			foundPythonFunc = true
			break
		}
	}

	if !foundPythonFunc {
		t.Error("Python function 'hello' should still be extracted in mixed Go+Python project")
	}
}

func TestBuildGraphFromASTGoFileNoPanic(t *testing.T) {
	goCode := `package main

import "fmt"

const MaxSize = 100

var version = "1.0"

type Server struct {
	Host string
	Port int
}

func NewServer(host string, port int) *Server {
	return &Server{Host: host, Port: port}
}

func (s *Server) Start() error {
	fmt.Println("starting")
	return nil
}

func main() {
	s := NewServer("localhost", 8080)
	s.Start()
	defer fmt.Println("done")
	go func() {
		fmt.Println("goroutine")
	}()
}
`
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(goCode))
	if err != nil {
		t.Fatalf("Failed to parse Go source code: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	graph := NewCodeGraph()

	// This should not panic — all Go node types hit stubs or are recursively traversed.
	buildGraphFromAST(root, []byte(goCode), graph, nil, "main.go")

	// PR-01: No nodes expected (stubs don't create nodes).
	// The important thing is it doesn't crash.
}
