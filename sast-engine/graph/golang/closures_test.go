package golang

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/stretchr/testify/assert"
)

func TestParseFuncLiteral(t *testing.T) {
	tests := []struct {
		name               string
		code               string
		expectedParamNames []string
		expectedParamTypes []string
		expectedReturnType string
	}{
		{
			name:               "Simple closure with params and return",
			code:               "package p\nfunc f() { _ = func(x int) int { return x + 1 } }",
			expectedParamNames: []string{"x"},
			expectedParamTypes: []string{"x: int"},
			expectedReturnType: "int",
		},
		{
			name:               "Closure with no params or return",
			code:               "package p\nfunc f() { _ = func() { println(\"hi\") } }",
			expectedParamNames: []string{},
			expectedParamTypes: []string{},
			expectedReturnType: "",
		},
		{
			name:               "Closure with multiple params",
			code:               "package p\nfunc f() { _ = func(a, b int, c string) { } }",
			expectedParamNames: []string{"a", "b", "c"},
			expectedParamTypes: []string{"a: int", "b: int", "c: string"},
			expectedReturnType: "",
		},
		{
			name:               "Closure with multiple return types",
			code:               "package p\nfunc f() { _ = func(s string) (int, error) { return 0, nil } }",
			expectedParamNames: []string{"s"},
			expectedParamTypes: []string{"s: string"},
			expectedReturnType: "(int, error)",
		},
		{
			name:               "IIFE (immediately invoked)",
			code:               "package p\nfunc f() { func() { println(\"hi\") }() }",
			expectedParamNames: []string{},
			expectedParamTypes: []string{},
			expectedReturnType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the func_literal node
			funcLitNode := findFuncLiteral(tree.RootNode())
			assert.NotNil(t, funcLitNode, "func_literal node not found")

			// Parse the func_literal
			info := ParseFuncLiteral(funcLitNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedParamNames, info.Params.Names, "Param names mismatch")
			assert.Equal(t, tt.expectedParamTypes, info.Params.Types, "Param types mismatch")
			assert.Equal(t, tt.expectedReturnType, info.ReturnType, "Return type mismatch")
			assert.Greater(t, info.LineNumber, uint32(0), "LineNumber should be set")
			assert.Greater(t, info.EndByte, info.StartByte, "EndByte should be > StartByte")
		})
	}
}

func TestParseDeferStatement(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		expectedFunc     string
		expectedObj      string
		expectedSelector bool
		expectedArgs     []string
	}{
		{
			name:             "Defer method call",
			code:             "package p\nfunc f() { defer f.Close() }",
			expectedFunc:     "Close",
			expectedObj:      "f",
			expectedSelector: true,
			expectedArgs:     []string{},
		},
		{
			name:             "Defer simple function",
			code:             "package p\nfunc f() { defer cleanup() }",
			expectedFunc:     "cleanup",
			expectedObj:      "",
			expectedSelector: false,
			expectedArgs:     []string{},
		},
		{
			name:             "Defer with arguments",
			code:             "package p\nfunc f() { defer recover(x, y) }",
			expectedFunc:     "recover",
			expectedObj:      "",
			expectedSelector: false,
			expectedArgs:     []string{"x", "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the defer_statement node
			deferNode := findDeferStatement(tree.RootNode())
			assert.NotNil(t, deferNode, "defer_statement node not found")

			// Parse the defer_statement
			info := ParseDeferStatement(deferNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedFunc, info.FunctionName, "FunctionName mismatch")
			assert.Equal(t, tt.expectedObj, info.ObjectName, "ObjectName mismatch")
			assert.Equal(t, tt.expectedSelector, info.IsSelector, "IsSelector mismatch")
			assert.Equal(t, tt.expectedArgs, info.Arguments, "Arguments mismatch")
		})
	}
}

func TestParseGoStatement(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		expectedFunc     string
		expectedObj      string
		expectedSelector bool
		expectedArgs     []string
	}{
		{
			name:             "Go simple function",
			code:             "package p\nfunc f() { go handler(conn) }",
			expectedFunc:     "handler",
			expectedObj:      "",
			expectedSelector: false,
			expectedArgs:     []string{"conn"},
		},
		{
			name:             "Go method call",
			code:             "package p\nfunc f() { go srv.Start() }",
			expectedFunc:     "Start",
			expectedObj:      "srv",
			expectedSelector: true,
			expectedArgs:     []string{},
		},
		{
			name:             "Go with closure",
			code:             "package p\nfunc f() { go func() { println(\"hi\") }() }",
			expectedFunc:     "",
			expectedObj:      "",
			expectedSelector: false,
			expectedArgs:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the go_statement node
			goNode := findGoStatement(tree.RootNode())
			assert.NotNil(t, goNode, "go_statement node not found")

			// Parse the go_statement
			info := ParseGoStatement(goNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedFunc, info.FunctionName, "FunctionName mismatch")
			assert.Equal(t, tt.expectedObj, info.ObjectName, "ObjectName mismatch")
			assert.Equal(t, tt.expectedSelector, info.IsSelector, "IsSelector mismatch")
			assert.Equal(t, tt.expectedArgs, info.Arguments, "Arguments mismatch")
		})
	}
}

func TestParseFuncLiteralNil(t *testing.T) {
	// Test nil node
	info := ParseFuncLiteral(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-func_literal node
	info = ParseFuncLiteral(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

func TestParseDeferStatementNil(t *testing.T) {
	// Test nil node
	info := ParseDeferStatement(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-defer_statement node
	info = ParseDeferStatement(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

func TestParseGoStatementNil(t *testing.T) {
	// Test nil node
	info := ParseGoStatement(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-go_statement node
	info = ParseGoStatement(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

// Helper: recursively find the first func_literal node
func findFuncLiteral(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "func_literal" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findFuncLiteral(child); result != nil {
			return result
		}
	}
	return nil
}

// Helper: recursively find the first defer_statement node
func findDeferStatement(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "defer_statement" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findDeferStatement(child); result != nil {
			return result
		}
	}
	return nil
}

// Helper: recursively find the first go_statement node
func findGoStatement(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "go_statement" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findGoStatement(child); result != nil {
			return result
		}
	}
	return nil
}
