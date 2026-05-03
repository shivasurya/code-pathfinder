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

	graph := Initialize(tmpDir, nil)

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

	graph := Initialize(tmpDir, nil)

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

	graph := Initialize(tmpDir, nil)

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

	graph := Initialize(tmpDir, nil)

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
	graph := Initialize("/path/that/does/not/exist", nil)

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

	graph := Initialize(tmpDir, nil)

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

func TestInitializeWithProgressCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_progress_callbacks")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple Java files
	for i := 1; i <= 3; i++ {
		javaCode := `public class TestClass` + string(rune('0'+i)) + ` { }`
		javaFile := filepath.Join(tmpDir, "Test"+string(rune('0'+i))+".java")
		if err := os.WriteFile(javaFile, []byte(javaCode), 0644); err != nil {
			t.Fatalf("Failed to write Java file: %v", err)
		}
	}

	// Track callback invocations
	var startCalled bool
	var startTotal int
	var progressCalls int

	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {
			startCalled = true
			startTotal = totalFiles
		},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	if !startCalled {
		t.Error("OnStart callback was not called")
	}

	if startTotal != 3 {
		t.Errorf("OnStart received total=%d, expected 3", startTotal)
	}

	if progressCalls != 3 {
		t.Errorf("OnProgress called %d times, expected 3", progressCalls)
	}
}

func TestInitializeWithNilCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_nil_callbacks")
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

	// Should not panic with nil callbacks
	graph := Initialize(tmpDir, nil)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}
}

func TestInitializeWithPartialCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_partial_callbacks")
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

	// Test with only OnStart
	callbacks1 := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
	}
	graph1 := Initialize(tmpDir, callbacks1)
	if graph1 == nil {
		t.Fatal("Initialize should handle callbacks with only OnStart")
	}

	// Test with only OnProgress
	callbacks2 := &ProgressCallbacks{
		OnProgress: func() {},
	}
	graph2 := Initialize(tmpDir, callbacks2)
	if graph2 == nil {
		t.Fatal("Initialize should handle callbacks with only OnProgress")
	}
}

func TestInitializeWithDockerfileReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can read all files)")
	}

	tmpDir, err := os.MkdirTemp("", "test_dockerfile_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Dockerfile with no read permissions to trigger read error
	unreadableDockerfile := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(unreadableDockerfile, []byte("FROM ubuntu"), 0000); err != nil {
		t.Fatalf("Failed to write Dockerfile: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	// Fix permissions for cleanup
	os.Chmod(unreadableDockerfile, 0644)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph even with read errors")
	}

	// Progress should still be called even when file read fails
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for unreadable Dockerfile, got %d", progressCalls)
	}
}

func TestInitializeWithDockerComposeParseError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_compose_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an invalid docker-compose.yml that will cause parse error
	invalidCompose := filepath.Join(tmpDir, "docker-compose.yml")
	// Invalid YAML syntax
	if err := os.WriteFile(invalidCompose, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write invalid compose file: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph even with parse errors")
	}

	// Progress should still be called even when parsing fails
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for failed compose file, got %d", progressCalls)
	}
}

func TestInitializeWithValidDockerfilesAndCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_valid_dockerfile")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid Dockerfile
	validDockerfile := filepath.Join(tmpDir, "Dockerfile")
	dockerContent := `FROM ubuntu:20.04
RUN apt-get update
CMD ["/bin/bash"]`
	if err := os.WriteFile(validDockerfile, []byte(dockerContent), 0644); err != nil {
		t.Fatalf("Failed to write valid Dockerfile: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	// Progress should be called for successful parse
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for successful Dockerfile, got %d", progressCalls)
	}
}

func TestInitializeWithValidDockerComposeAndCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_valid_compose")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid docker-compose.yml
	validCompose := filepath.Join(tmpDir, "docker-compose.yml")
	composeContent := `version: '3'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"`
	if err := os.WriteFile(validCompose, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write valid compose file: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	// Progress should be called for successful parse
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for successful compose file, got %d", progressCalls)
	}
}

func TestInitializeWithMalformedJavaAndCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_malformed_java")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a malformed Java file that tree-sitter might have issues with
	// Tree-sitter is generally permissive, so we create valid Java
	javaFile := filepath.Join(tmpDir, "Test.java")
	if err := os.WriteFile(javaFile, []byte("public class Test { }"), 0644); err != nil {
		t.Fatalf("Failed to write Java file: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	// Progress should be called for Java file processing
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for Java file, got %d", progressCalls)
	}
}

func TestInitializeWithMalformedPythonAndCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_malformed_python")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Python file
	pyFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(pyFile, []byte("def test(): pass"), 0644); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph")
	}

	// Progress should be called for Python file processing
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for Python file, got %d", progressCalls)
	}
}

func TestInitializeWithJavaFileReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can read all files)")
	}

	tmpDir, err := os.MkdirTemp("", "test_java_read_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Java file with no read permissions
	unreadableJavaFile := filepath.Join(tmpDir, "Test.java")
	if err := os.WriteFile(unreadableJavaFile, []byte("public class Test {}"), 0000); err != nil {
		t.Fatalf("Failed to write Java file: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	// Fix permissions for cleanup
	os.Chmod(unreadableJavaFile, 0644)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph even with read errors")
	}

	// Progress should be called even when file read fails
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for unreadable Java file, got %d", progressCalls)
	}
}

func TestInitializeWithPythonFileReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can read all files)")
	}

	tmpDir, err := os.MkdirTemp("", "test_python_read_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Python file with no read permissions
	unreadablePyFile := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(unreadablePyFile, []byte("def test(): pass"), 0000); err != nil {
		t.Fatalf("Failed to write Python file: %v", err)
	}

	// Track callback invocations
	var progressCalls int
	callbacks := &ProgressCallbacks{
		OnStart: func(totalFiles int) {},
		OnProgress: func() {
			progressCalls++
		},
	}

	graph := Initialize(tmpDir, callbacks)

	// Fix permissions for cleanup
	os.Chmod(unreadablePyFile, 0644)

	if graph == nil {
		t.Fatal("Initialize should return a non-nil graph even with read errors")
	}

	// Progress should be called even when file read fails
	if progressCalls != 1 {
		t.Errorf("OnProgress should be called once for unreadable Python file, got %d", progressCalls)
	}
}

// TestInitialize_NoDuplicateOutgoingEdges guards against a regression where
// the result-collection step in Initialize re-attached every per-file edge to
// its source node (calling codeGraph.AddEdge on edges already inserted by the
// worker). That doubled OutgoingEdges and surfaced as 2× detections on every
// rule that walked the call graph (PR-07/08 C/C++ builders, etc.).
//
// The test parses a tiny C source containing two distinct calls inside one
// function and asserts that the function node ends up with exactly two
// OutgoingEdges, not four.
func TestInitialize_NoDuplicateOutgoingEdges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_no_dup_edges")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	src := `void f(const char *s) {
    strcpy(0, s);
    system(s);
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.c"), []byte(src), 0644); err != nil {
		t.Fatalf("write c source: %v", err)
	}

	g := Initialize(tmpDir, nil)
	if g == nil {
		t.Fatal("Initialize returned nil")
	}

	var fn *Node
	for _, n := range g.Nodes {
		if n != nil && n.Language == "c" && n.Type == "function_definition" && n.Name == "f" {
			fn = n
			break
		}
	}
	if fn == nil {
		t.Fatal("expected function_definition for f")
	}

	if got := len(fn.OutgoingEdges); got != 2 {
		t.Fatalf("expected 2 outgoing edges (one per distinct call), got %d", got)
	}

	seen := map[string]int{}
	for _, e := range fn.OutgoingEdges {
		if e == nil || e.To == nil {
			t.Fatalf("nil edge or destination on f")
		}
		key := e.To.Name + "@" + e.To.ID
		seen[key]++
	}
	for k, c := range seen {
		if c != 1 {
			t.Errorf("edge %s seen %d times, expected exactly 1", k, c)
		}
	}
}

// TestInitialize_PreservesDistinctSameLineCalls guards the dedup fix's
// non-regression contract: when several calls live on the same line — either
// distinct targets (`printf("%s", strdup(s));`) or the same target nested
// (`strcpy(a, strcpy(b, c))`) — every call site must remain visible. The fix
// removes a duplicate edge that was attached twice to the same OutgoingEdges
// slice; it must not collapse genuinely distinct sites that happen to share a
// line number.
func TestInitialize_PreservesDistinctSameLineCalls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_same_line_calls")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	src := `void f(const char *src) {
    char a[16], b[16], c[16];
    printf("%s", strdup(src));
    strcpy(c, strcpy(a, b));
    memcpy(strcat(a, strdup(src)), b, 4);
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.c"), []byte(src), 0644); err != nil {
		t.Fatalf("write c source: %v", err)
	}

	g := Initialize(tmpDir, nil)
	if g == nil {
		t.Fatal("Initialize returned nil")
	}

	var fn *Node
	for _, n := range g.Nodes {
		if n != nil && n.Language == "c" && n.Type == "function_definition" && n.Name == "f" {
			fn = n
			break
		}
	}
	if fn == nil {
		t.Fatal("expected function_definition for f")
	}

	// Line 3: printf(...) + strdup(...) — 2 calls
	// Line 4: strcpy(c, ...) outer + strcpy(a, b) inner — 2 calls of the same target
	// Line 5: memcpy(...) + strcat(...) + strdup(...) — 3 calls
	// Total: 7 syntactic call sites.
	const expectedTotal = 7
	if got := len(fn.OutgoingEdges); got != expectedTotal {
		t.Fatalf("expected %d outgoing edges across all same-line calls, got %d", expectedTotal, got)
	}

	byTarget := map[string]int{}
	byLine := map[uint32]int{}
	for _, e := range fn.OutgoingEdges {
		if e == nil || e.To == nil {
			t.Fatalf("nil edge or destination on f")
		}
		byTarget[e.To.Name]++
		byLine[e.To.LineNumber]++
	}

	// Both nested strcpys on line 4 must survive (same target, same line).
	if got := byTarget["strcpy"]; got != 2 {
		t.Errorf("expected 2 strcpy edges on line 4 (outer + nested), got %d", got)
	}
	// Distinct targets on line 3 must both survive.
	if got := byTarget["printf"]; got != 1 {
		t.Errorf("expected 1 printf edge on line 3, got %d", got)
	}
	// strdup appears once on line 3 and once on line 5 — both must survive.
	if got := byTarget["strdup"]; got != 2 {
		t.Errorf("expected 2 strdup edges (line 3 + line 5), got %d", got)
	}
	// Three distinct calls on the multi-call line must all survive.
	for _, n := range []string{"memcpy", "strcat"} {
		if got := byTarget[n]; got != 1 {
			t.Errorf("expected 1 %s edge on the 3-call line, got %d", n, got)
		}
	}
}
