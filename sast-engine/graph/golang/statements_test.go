package golang

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/stretchr/testify/assert"
)

func TestParseReturnStatement(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedValues []string
	}{
		{
			name:           "Single return value",
			code:           "package p\nfunc f() int { return 42 }",
			expectedValues: []string{"42"},
		},
		{
			name:           "Multiple return values",
			code:           "package p\nfunc f() (int, error) { return 0, nil }",
			expectedValues: []string{"0", "nil"},
		},
		{
			name:           "Return expression",
			code:           "package p\nfunc f() int { return x + 1 }",
			expectedValues: []string{"x + 1"},
		},
		{
			name:           "Return no values",
			code:           "package p\nfunc f() { return }",
			expectedValues: []string{},
		},
		{
			name:           "Return function call",
			code:           "package p\nfunc f() error { return foo() }",
			expectedValues: []string{"foo()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the return_statement node
			retNode := findReturnStatement(tree.RootNode())
			assert.NotNil(t, retNode, "return_statement node not found")

			// Parse the return_statement
			info := ParseReturnStatement(retNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedValues, info.Values, "Values mismatch")
			assert.Greater(t, info.LineNumber, uint32(0), "LineNumber should be set")
			assert.Greater(t, info.EndByte, info.StartByte, "EndByte should be > StartByte")
		})
	}
}

func TestParseForStatement(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedRange bool
		expectedCond  string
		expectedInit  string
		expectedUpd   string
		expectedLeft  string
		expectedRight string
	}{
		{
			name:          "C-style for loop",
			code:          "package p\nfunc f() { for i := 0; i < 10; i++ {} }",
			expectedRange: false,
			expectedCond:  "i < 10",
			expectedInit:  "i := 0",
			expectedUpd:   "i++",
		},
		{
			name:          "Range for loop with index and value",
			code:          "package p\nfunc f() { for i, v := range items {} }",
			expectedRange: true,
			expectedLeft:  "i, v",
			expectedRight: "items",
		},
		{
			name:          "Range for loop with blank identifier",
			code:          "package p\nfunc f() { for _, v := range items {} }",
			expectedRange: true,
			expectedLeft:  "_, v",
			expectedRight: "items",
		},
		{
			name:          "Infinite loop",
			code:          "package p\nfunc f() { for {} }",
			expectedRange: false,
			expectedCond:  "",
			expectedInit:  "",
			expectedUpd:   "",
		},
		{
			name:          "While-style loop",
			code:          "package p\nfunc f() { for i < 10 {} }",
			expectedRange: false,
			expectedCond:  "",
			expectedInit:  "i < 10",
			expectedUpd:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the for_statement node
			forNode := findForStatement(tree.RootNode())
			assert.NotNil(t, forNode, "for_statement node not found")

			// Parse the for_statement
			info := ParseForStatement(forNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedRange, info.IsRange, "IsRange mismatch")
			if tt.expectedRange {
				assert.Equal(t, tt.expectedLeft, info.Left, "Left mismatch")
				assert.Equal(t, tt.expectedRight, info.Right, "Right mismatch")
			} else {
				assert.Equal(t, tt.expectedCond, info.Condition, "Condition mismatch")
				assert.Equal(t, tt.expectedInit, info.Init, "Init mismatch")
				assert.Equal(t, tt.expectedUpd, info.Update, "Update mismatch")
			}
			assert.Greater(t, info.LineNumber, uint32(0), "LineNumber should be set")
			assert.Greater(t, info.EndByte, info.StartByte, "EndByte should be > StartByte")
		})
	}
}

func TestParseIfStatement(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedCond  string
	}{
		{
			name:         "Simple if statement",
			code:         "package p\nfunc f() { if err != nil { return err } }",
			expectedCond: "err != nil",
		},
		{
			name:         "If with complex condition",
			code:         "package p\nfunc f() { if x > 0 && y < 10 { } }",
			expectedCond: "x > 0 && y < 10",
		},
		{
			name:         "If with function call",
			code:         "package p\nfunc f() { if isValid() { } }",
			expectedCond: "isValid()",
		},
		{
			name:         "If with assignment",
			code:         "package p\nfunc f() { if err := foo(); err != nil { } }",
			expectedCond: "err != nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the if_statement node
			ifNode := findIfStatement(tree.RootNode())
			assert.NotNil(t, ifNode, "if_statement node not found")

			// Parse the if_statement
			info := ParseIfStatement(ifNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedCond, info.Condition, "Condition mismatch")
			assert.Greater(t, info.LineNumber, uint32(0), "LineNumber should be set")
			assert.Greater(t, info.EndByte, info.StartByte, "EndByte should be > StartByte")
		})
	}
}

func TestParseReturnStatementNil(t *testing.T) {
	// Test nil node
	info := ParseReturnStatement(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-return_statement node
	info = ParseReturnStatement(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

func TestParseForStatementNil(t *testing.T) {
	// Test nil node
	info := ParseForStatement(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-for_statement node
	info = ParseForStatement(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

func TestParseIfStatementNil(t *testing.T) {
	// Test nil node
	info := ParseIfStatement(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-if_statement node
	info = ParseIfStatement(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

// Helper: recursively find the first return_statement node.
func findReturnStatement(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "return_statement" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findReturnStatement(child); result != nil {
			return result
		}
	}
	return nil
}

// Helper: recursively find the first for_statement node.
func findForStatement(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "for_statement" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findForStatement(child); result != nil {
			return result
		}
	}
	return nil
}

// Helper: recursively find the first if_statement node.
func findIfStatement(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "if_statement" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findIfStatement(child); result != nil {
			return result
		}
	}
	return nil
}
