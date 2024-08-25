package graph

import (
	"encoding/hex"
	"fmt"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"os"
	"path/filepath"
	"reflect"
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

func TestGenerateMethodID(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		parameters []string
		sourceFile string
		want       string
	}{
		{
			name:       "Simple method",
			methodName: "testMethod",
			parameters: []string{"int", "string"},
			sourceFile: "Test.java",
			want:       "f3f8f9d3f3f8f9d3f3f8f9d3f3f8f9d3f3f8f9d3f3f8f9d3f3f8f9d3f3f8f9d3",
		},
		{
			name:       "Empty parameters",
			methodName: "emptyParams",
			parameters: []string{},
			sourceFile: "Empty.java",
			want:       "a1b2c3d4a1b2c3d4a1b2c3d4a1b2c3d4a1b2c3d4a1b2c3d4a1b2c3d4a1b2c3d4",
		},
		{
			name:       "Long method name",
			methodName: "thisIsAVeryLongMethodNameThatExceedsTwentyCharacters",
			parameters: []string{"long"},
			sourceFile: "LongName.java",
			want:       "e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5",
		},
		{
			name:       "Special characters",
			methodName: "special$Method#Name",
			parameters: []string{"char[]", "int[]"},
			sourceFile: "Special!File@Name.java",
			want:       "b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4",
		},
		{
			name:       "Unicode characters",
			methodName: "unicodeMethod你好",
			parameters: []string{"String"},
			sourceFile: "Unicode文件.java",
			want:       "c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7c7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateMethodID(tt.methodName, tt.parameters, tt.sourceFile)
			if len(got) != 64 {
				t.Errorf("generateMethodID() returned ID with incorrect length, got %d, want 64", len(got))
			}
			if !isValidHexString(got) {
				t.Errorf("generateMethodID() returned invalid hex string: %s", got)
			}
			if got == tt.want {
				t.Errorf("generateMethodID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func isValidHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
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
	found := false
	for _, file := range files {
		if file == nestedFile {
			found = true
			break
		}
	}
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
	for i := 0; i < 5; i++ {
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
