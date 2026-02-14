package golang

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// parseGoSnippet parses a Go source snippet and returns the root node.
func parseGoSnippet(t *testing.T, code string) (*sitter.Tree, *sitter.Node) {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse Go code: %v", err)
	}
	return tree, tree.RootNode()
}

// findNode recursively finds the first node of a given type.
func findNode(node *sitter.Node, nodeType string) *sitter.Node {
	if node.Type() == nodeType {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if found := findNode(node.Child(i), nodeType); found != nil {
			return found
		}
	}
	return nil
}

func TestExtractParameters(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedNames []string
		expectedTypes []string
	}{
		{
			name:          "Simple params",
			code:          `package p; func Foo(x int, y string) {}`,
			expectedNames: []string{"x", "y"},
			expectedTypes: []string{"x: int", "y: string"},
		},
		{
			name:          "Grouped params sharing type",
			code:          `package p; func Foo(a, b int) {}`,
			expectedNames: []string{"a", "b"},
			expectedTypes: []string{"a: int", "b: int"},
		},
		{
			name:          "Empty param list",
			code:          `package p; func Foo() {}`,
			expectedNames: nil,
			expectedTypes: nil,
		},
		{
			name:          "Variadic param",
			code:          `package p; func Foo(args ...string) {}`,
			expectedNames: []string{"args"},
			expectedTypes: []string{"args: ...string"},
		},
		{
			name:          "Mixed params with pointer type",
			code:          `package p; func Foo(w http.ResponseWriter, r *http.Request) {}`,
			expectedNames: []string{"w", "r"},
			expectedTypes: []string{"w: http.ResponseWriter", "r: *http.Request"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			funcDecl := findNode(root, "function_declaration")
			if funcDecl == nil {
				t.Fatal("function_declaration not found")
			}
			paramList := funcDecl.ChildByFieldName("parameters")

			result := ExtractParameters(paramList, []byte(tt.code))

			if len(result.Names) != len(tt.expectedNames) {
				t.Fatalf("Names: expected %d, got %d: %v", len(tt.expectedNames), len(result.Names), result.Names)
			}
			for i, name := range tt.expectedNames {
				if result.Names[i] != name {
					t.Errorf("Names[%d]: expected %q, got %q", i, name, result.Names[i])
				}
			}

			if len(result.Types) != len(tt.expectedTypes) {
				t.Fatalf("Types: expected %d, got %d: %v", len(tt.expectedTypes), len(result.Types), result.Types)
			}
			for i, typ := range tt.expectedTypes {
				if result.Types[i] != typ {
					t.Errorf("Types[%d]: expected %q, got %q", i, typ, result.Types[i])
				}
			}
		})
	}
}

func TestExtractParametersNil(t *testing.T) {
	result := ExtractParameters(nil, nil)
	if len(result.Names) != 0 || len(result.Types) != 0 {
		t.Errorf("Expected empty GoParams for nil input, got Names=%v Types=%v", result.Names, result.Types)
	}
}

func TestExtractParametersUnnamedParams(t *testing.T) {
	// func(int, string) — params without names
	code := `package p; type F = func(int, string)`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	funcType := findNode(root, "function_type")
	if funcType == nil {
		t.Fatal("function_type not found")
	}
	paramList := funcType.ChildByFieldName("parameters")

	result := ExtractParameters(paramList, []byte(code))
	// Unnamed params: names are empty strings, types are the type names
	if len(result.Types) != 2 {
		t.Fatalf("Expected 2 types, got %d: %v", len(result.Types), result.Types)
	}
}

func TestExtractReturnType(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "Single return type",
			code:     `package p; func Foo() int { return 0 }`,
			expected: "int",
		},
		{
			name:     "Multiple return types",
			code:     `package p; func Foo() (string, error) { return "", nil }`,
			expected: "(string, error)",
		},
		{
			name:     "Named returns",
			code:     `package p; func Foo() (n int, err error) { return }`,
			expected: "(n int, err error)",
		},
		{
			name:     "No return type",
			code:     `package p; func Foo() {}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			funcDecl := findNode(root, "function_declaration")
			if funcDecl == nil {
				t.Fatal("function_declaration not found")
			}
			resultNode := funcDecl.ChildByFieldName("result")

			got := ExtractReturnType(resultNode, []byte(tt.code))
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestExtractReturnTypeNil(t *testing.T) {
	got := ExtractReturnType(nil, nil)
	if got != "" {
		t.Errorf("Expected empty string for nil input, got %q", got)
	}
}

func TestExtractReceiverType(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "Pointer receiver",
			code:     `package p; type S struct{}; func (s *S) M() {}`,
			expected: "S",
		},
		{
			name:     "Value receiver",
			code:     `package p; type S struct{}; func (s S) M() {}`,
			expected: "S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			methodDecl := findNode(root, "method_declaration")
			if methodDecl == nil {
				t.Fatal("method_declaration not found")
			}
			receiverNode := methodDecl.ChildByFieldName("receiver")

			got := ExtractReceiverType(receiverNode, []byte(tt.code))
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestExtractReceiverTypeGeneric(t *testing.T) {
	// Generic receiver: *Stack[T] has generic_type, not type_identifier.
	// ExtractReceiverType currently does not handle generics, returns "".
	code := `package p
type Stack[T any] struct{ items []T }
func (s *Stack[T]) Pop() T { var zero T; return zero }`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	methodDecl := findNode(root, "method_declaration")
	if methodDecl == nil {
		t.Fatal("method_declaration not found")
	}
	receiverNode := methodDecl.ChildByFieldName("receiver")

	got := ExtractReceiverType(receiverNode, []byte(code))
	// Generic receivers are not handled in Phase 1, expect ""
	if got != "" {
		t.Errorf("Expected empty string for generic receiver, got %q", got)
	}
}

func TestExtractReceiverTypeNil(t *testing.T) {
	got := ExtractReceiverType(nil, nil)
	if got != "" {
		t.Errorf("Expected empty string for nil input, got %q", got)
	}
}

func TestExtractStructFields(t *testing.T) {
	code := `package p
type Server struct {
	Host string
	Port int
	Logger
}`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	structType := findNode(root, "struct_type")
	if structType == nil {
		t.Fatal("struct_type not found")
	}

	fields := ExtractStructFields(structType, []byte(code))

	// Should have 2 named fields + 1 embedded type
	if len(fields) < 2 {
		t.Fatalf("Expected at least 2 fields, got %d: %v", len(fields), fields)
	}

	// Check named fields
	if fields[0] != "Host: string" {
		t.Errorf("Field[0]: expected %q, got %q", "Host: string", fields[0])
	}
	if fields[1] != "Port: int" {
		t.Errorf("Field[1]: expected %q, got %q", "Port: int", fields[1])
	}
}

func TestExtractStructFieldsNil(t *testing.T) {
	fields := ExtractStructFields(nil, nil)
	if len(fields) != 0 {
		t.Errorf("Expected empty fields for nil input, got %v", fields)
	}
}

func TestExtractStructFieldsEmpty(t *testing.T) {
	code := `package p; type Empty struct{}`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	structType := findNode(root, "struct_type")
	if structType == nil {
		t.Fatal("struct_type not found")
	}

	fields := ExtractStructFields(structType, []byte(code))
	if len(fields) != 0 {
		t.Errorf("Expected 0 fields for empty struct, got %d: %v", len(fields), fields)
	}
}

func TestExtractStructFieldsEmbedded(t *testing.T) {
	code := `package p
type Admin struct {
	User
	Name string
}`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	structType := findNode(root, "struct_type")
	if structType == nil {
		t.Fatal("struct_type not found")
	}

	fields := ExtractStructFields(structType, []byte(code))
	if len(fields) < 1 {
		t.Fatalf("Expected at least 1 field, got %d: %v", len(fields), fields)
	}
}

func TestExtractInterfaceMethods(t *testing.T) {
	code := `package p
type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
	Close() error
}`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	interfaceType := findNode(root, "interface_type")
	if interfaceType == nil {
		t.Fatal("interface_type not found")
	}

	methods := ExtractInterfaceMethods(interfaceType, []byte(code))

	if len(methods) < 2 {
		t.Fatalf("Expected at least 2 methods, got %d: %v", len(methods), methods)
	}
}

func TestExtractInterfaceMethodsNil(t *testing.T) {
	methods := ExtractInterfaceMethods(nil, nil)
	if len(methods) != 0 {
		t.Errorf("Expected empty methods for nil input, got %v", methods)
	}
}

func TestExtractInterfaceMethodsWithEmbedded(t *testing.T) {
	code := `package p
type ReadWriteCloser interface {
	io.Reader
	io.Writer
	Close() error
}`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	interfaceType := findNode(root, "interface_type")
	if interfaceType == nil {
		t.Fatal("interface_type not found")
	}

	methods := ExtractInterfaceMethods(interfaceType, []byte(code))
	if len(methods) < 3 {
		t.Fatalf("Expected at least 3 entries (2 embedded + 1 method), got %d: %v", len(methods), methods)
	}
}

func TestDetermineVisibility(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Exported uppercase", "HandleRequest", "public"},
		{"Unexported lowercase", "handleRequest", "private"},
		{"Single uppercase", "A", "public"},
		{"Single lowercase", "a", "private"},
		{"Empty string", "", "private"},
		{"Underscore prefix", "_internal", "private"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineVisibility(tt.input)
			if got != tt.expected {
				t.Errorf("DetermineVisibility(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsInitFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"init function", "init", true},
		{"main function", "main", false},
		{"other function", "handleRequest", false},
		{"empty string", "", false},
		{"Init capitalized", "Init", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInitFunction(tt.input)
			if got != tt.expected {
				t.Errorf("IsInitFunction(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsGoKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Go keywords
		{"keyword func", "func", true},
		{"keyword return", "return", true},
		{"keyword defer", "defer", true},
		{"keyword go", "go", true},
		// Predeclared identifiers
		{"true", "true", true},
		{"false", "false", true},
		{"nil", "nil", true},
		{"iota", "iota", true},
		// Predeclared types
		{"int", "int", true},
		{"string", "string", true},
		{"error", "error", true},
		{"any (Go 1.18)", "any", true},
		{"comparable", "comparable", true},
		// Builtin functions
		{"append", "append", true},
		{"len", "len", true},
		{"make", "make", true},
		{"panic", "panic", true},
		{"recover", "recover", true},
		{"clear", "clear", true},
		{"max", "max", true},
		{"min", "min", true},
		// Non-keywords
		{"user variable", "myVar", false},
		{"user type", "Server", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGoKeyword(tt.input)
			if got != tt.expected {
				t.Errorf("IsGoKeyword(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
