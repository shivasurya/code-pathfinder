package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

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
	node1 := &model.Node{ID: "node1", Type: "class"}
	node2 := &model.Node{ID: "node2", Type: "method"}
	node3 := &model.Node{ID: "node3", Type: "class"}
	node4 := &model.Node{ID: "node4", Type: "interface"}
	node5 := &model.Node{ID: "node5", Type: "method"}

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
		graph.AddNode(&model.Node{ID: fmt.Sprintf("node%d", i), Type: "class"})
	}

	nodes := graph.FindNodesByType("class")
	if len(nodes) != 5 {
		t.Errorf("FindNodesByType('class') returned %d nodes, want 5", len(nodes))
	}
}

func TestFindNodesByTypeCaseSensitivity(t *testing.T) {
	graph := NewCodeGraph()
	graph.AddNode(&model.Node{ID: "node1", Type: "Class"})
	graph.AddNode(&model.Node{ID: "node2", Type: "class"})

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

	graph := Initialize(tempDir)

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

	graph := Initialize(tempDir)

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
	graph := Initialize(nonExistentDir)

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

	graph := Initialize(tempDir)

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
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("File%d.java", i)
		content := fmt.Sprintf("public class File%d { }", i)
		path := filepath.Join(tempDir, fileName)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	graph := Initialize(tempDir)

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
			buildQLTreeFromAST(root, []byte(tt.sourceCode), graph, "test.java")

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
		name           string
		sourceCode     string
		expectedName   string
		expectedIDPart string
	}{
		{
			name:           "Simple method",
			sourceCode:     "public void simpleMethod() {}",
			expectedName:   "simpleMethod",
			expectedIDPart: "e4bf121a07daa7b5fc0821f04fe31f22689361aaa7604264034bf231640c0b94",
		},
		{
			name:           "Method with parameters",
			sourceCode:     "private int complexMethod(String a, int b) {}",
			expectedName:   "complexMethod",
			expectedIDPart: "8fa7666614f2db09a92d83f0ec126328a0c0fc93ac0919ffce2be2ce65e5fed5",
		},
		{
			name:           "Generic method",
			sourceCode:     "public <T> List<T> genericMethod(T item) {}",
			expectedName:   "genericMethod",
			expectedIDPart: "4072dc9bf8d115f9c73a0ff3ff2205ef2866845921ac3dd218530ffe85966d96",
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

			if !strings.Contains(id, tt.expectedIDPart) {
				t.Errorf("Expected method ID to contain %s, but got %s", tt.expectedIDPart, id)
			}
		})
	}
}
