package clike

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	clang "github.com/smacker/go-tree-sitter/c"
	cpplang "github.com/smacker/go-tree-sitter/cpp"
)

// parseCSnippet parses the given C source and returns (tree, root). Caller
// must defer tree.Close().
func parseCSnippet(t *testing.T, code string) (*sitter.Tree, *sitter.Node) {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(clang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("parse C: %v", err)
	}
	return tree, tree.RootNode()
}

// parseCppSnippet parses the given C++ source and returns (tree, root).
// Caller must defer tree.Close().
func parseCppSnippet(t *testing.T, code string) (*sitter.Tree, *sitter.Node) {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(cpplang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("parse C++: %v", err)
	}
	return tree, tree.RootNode()
}

// snippet parses code with the C or C++ grammar selected by language.
func snippet(t *testing.T, language, code string) (*sitter.Tree, *sitter.Node) {
	t.Helper()
	if language == "cpp" {
		return parseCppSnippet(t, code)
	}
	return parseCSnippet(t, code)
}

// equalStringSlices reports whether two string slices are element-wise equal.
// nil and empty are treated as equal so tests can compare against literal
// []string{} without distinguishing the two representations.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// findNode performs a pre-order search and returns the first descendant of
// node whose Type() matches nodeType. Returns nil when no match exists.
func findNode(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}
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
