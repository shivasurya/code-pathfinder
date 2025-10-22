package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeWithEmptyDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test_empty_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	graph := Initialize(tmpDir)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for empty directory, got %d", len(graph.Nodes))
	}
}

func TestInitializeWithJavaFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_java_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Java file
	javaCode := `
public class HelloWorld {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}
`
	javaFile := filepath.Join(tmpDir, "HelloWorld.java")
	if err := os.WriteFile(javaFile, []byte(javaCode), 0644); err != nil {
		t.Fatalf("Failed to write Java file: %v", err)
	}

	graph := Initialize(tmpDir)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}
	if len(graph.Nodes) == 0 {
		t.Error("Expected nodes to be created from Java file")
	}

	// Check for class node
	hasClassNode := false
	for _, node := range graph.Nodes {
		if node.Type == "class_declaration" && node.Name == "HelloWorld" {
			hasClassNode = true
			break
		}
	}
	if !hasClassNode {
		t.Error("Expected to find HelloWorld class node")
	}
}

func TestInitializeWithPythonFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_python_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Python file
	pythonCode := `
def greet(name):
    return f"Hello, {name}!"

class Greeter:
    def __init__(self, greeting):
        self.greeting = greeting
`
	pythonFile := filepath.Join(tmpDir, "greet.py")
	if err := os.WriteFile(pythonFile, []byte(pythonCode), 0644); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	graph := Initialize(tmpDir)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}
	if len(graph.Nodes) == 0 {
		t.Error("Expected nodes to be created from Python file")
	}

	// Check for function and class nodes
	hasFunctionNode := false
	hasClassNode := false
	for _, node := range graph.Nodes {
		if node.Type == "function_definition" && node.Name == "greet" {
			hasFunctionNode = true
		}
		if node.Type == "class_definition" && node.Name == "Greeter" {
			hasClassNode = true
		}
	}
	if !hasFunctionNode {
		t.Error("Expected to find greet function node")
	}
	if !hasClassNode {
		t.Error("Expected to find Greeter class node")
	}
}

func TestInitializeWithMixedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_mixed_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Java file
	javaCode := `public class Test { }`
	javaFile := filepath.Join(tmpDir, "Test.java")
	if err := os.WriteFile(javaFile, []byte(javaCode), 0644); err != nil {
		t.Fatalf("Failed to write Java file: %v", err)
	}

	// Create a Python file
	pythonCode := `def test(): pass`
	pythonFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(pythonFile, []byte(pythonCode), 0644); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	// Create a non-source file (should be ignored)
	txtFile := filepath.Join(tmpDir, "readme.txt")
	if err := os.WriteFile(txtFile, []byte("This is a readme"), 0644); err != nil {
		t.Fatalf("Failed to write txt file: %v", err)
	}

	graph := Initialize(tmpDir)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}
	if len(graph.Nodes) == 0 {
		t.Error("Expected nodes to be created from source files")
	}

	// Check that both Java and Python nodes exist
	hasJavaNode := false
	hasPythonNode := false
	for _, node := range graph.Nodes {
		if node.isJavaSourceFile {
			hasJavaNode = true
		}
		if node.isPythonSourceFile {
			hasPythonNode = true
		}
	}
	if !hasJavaNode {
		t.Error("Expected to find Java nodes")
	}
	if !hasPythonNode {
		t.Error("Expected to find Python nodes")
	}
}

func TestInitializeWithNonExistentDirectory(t *testing.T) {
	graph := Initialize("/path/that/does/not/exist")

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph even for non-existent directory")
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for non-existent directory, got %d", len(graph.Nodes))
	}
}

func TestInitializeWithNestedDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_nested_dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested structure
	subDir := filepath.Join(tmpDir, "src", "main")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create Java file in subdirectory
	javaCode := `public class Nested { }`
	javaFile := filepath.Join(subDir, "Nested.java")
	if err := os.WriteFile(javaFile, []byte(javaCode), 0644); err != nil {
		t.Fatalf("Failed to write Java file: %v", err)
	}

	graph := Initialize(tmpDir)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}
	if len(graph.Nodes) == 0 {
		t.Error("Expected nodes to be created from nested file")
	}

	// Check that nested file was processed
	hasNestedClass := false
	for _, node := range graph.Nodes {
		if node.Type == "class_declaration" && node.Name == "Nested" {
			hasNestedClass = true
			break
		}
	}
	if !hasNestedClass {
		t.Error("Expected to find Nested class from subdirectory")
	}
}
