package golang

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/stretchr/testify/assert"
)

func TestParseCallExpression(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		expectedFunc     string
		expectedObj      string
		expectedArgs     []string
		expectedSelector bool
	}{
		{
			name:             "Simple function call",
			code:             "package p\nfunc f() { foo(x, y) }",
			expectedFunc:     "foo",
			expectedObj:      "",
			expectedArgs:     []string{"x", "y"},
			expectedSelector: false,
		},
		{
			name:             "Method call",
			code:             "package p\nfunc f() { obj.Method(a, b) }",
			expectedFunc:     "Method",
			expectedObj:      "obj",
			expectedArgs:     []string{"a", "b"},
			expectedSelector: true,
		},
		{
			name:             "Package function call",
			code:             "package p\nfunc f() { fmt.Println(\"hello\") }",
			expectedFunc:     "Println",
			expectedObj:      "fmt",
			expectedArgs:     []string{"\"hello\""},
			expectedSelector: true,
		},
		{
			name:             "Call with no arguments",
			code:             "package p\nfunc f() { bar() }",
			expectedFunc:     "bar",
			expectedObj:      "",
			expectedArgs:     []string{},
			expectedSelector: false,
		},
		{
			name:             "Chained method call",
			code:             "package p\nfunc f() { http.Get(url) }",
			expectedFunc:     "Get",
			expectedObj:      "http",
			expectedArgs:     []string{"url"},
			expectedSelector: true,
		},
		{
			name:             "Call with complex arguments",
			code:             "package p\nfunc f() { process(a+b, foo(), \"str\") }",
			expectedFunc:     "process",
			expectedObj:      "",
			expectedArgs:     []string{"a+b", "foo()", "\"str\""},
			expectedSelector: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the call_expression node
			callNode := findCallExpression(tree.RootNode())
			assert.NotNil(t, callNode, "call_expression node not found")

			// Parse the call expression
			info := ParseCallExpression(callNode, []byte(tt.code))
			assert.NotNil(t, info)

			// Verify fields
			assert.Equal(t, tt.expectedFunc, info.FunctionName, "FunctionName mismatch")
			assert.Equal(t, tt.expectedObj, info.ObjectName, "ObjectName mismatch")
			assert.Equal(t, tt.expectedSelector, info.IsSelector, "IsSelector mismatch")
			assert.Equal(t, tt.expectedArgs, info.Arguments, "Arguments mismatch")
			assert.Greater(t, info.LineNumber, uint32(0), "LineNumber should be set")
			assert.Greater(t, info.EndByte, info.StartByte, "EndByte should be > StartByte")
		})
	}
}

func TestParseSelectorExpression(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedObj  string
		expectedField string
	}{
		{
			name:         "Simple selector",
			code:         "package p\nfunc f() { obj.Method() }",
			expectedObj:  "obj",
			expectedField: "Method",
		},
		{
			name:         "Package selector",
			code:         "package p\nfunc f() { fmt.Println() }",
			expectedObj:  "fmt",
			expectedField: "Println",
		},
		{
			name:         "Field access",
			code:         "package p\nfunc f() { user.Name }",
			expectedObj:  "user",
			expectedField: "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			assert.NoError(t, err)
			defer tree.Close()

			// Find the selector_expression node
			selNode := findSelectorExpression(tree.RootNode())
			assert.NotNil(t, selNode, "selector_expression node not found")

			// Parse the selector expression
			obj, field := ParseSelectorExpression(selNode, []byte(tt.code))

			// Verify fields
			assert.Equal(t, tt.expectedObj, obj, "Object mismatch")
			assert.Equal(t, tt.expectedField, field, "Field mismatch")
		})
	}
}

func TestParseCallExpressionNil(t *testing.T) {
	// Test nil node
	info := ParseCallExpression(nil, []byte(""))
	assert.Nil(t, info)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p\nfunc f() {}"))
	defer tree.Close()

	// Pass a non-call_expression node
	info = ParseCallExpression(tree.RootNode(), []byte("package p"))
	assert.Nil(t, info)
}

func TestParseSelectorExpressionNil(t *testing.T) {
	// Test nil node
	obj, field := ParseSelectorExpression(nil, []byte(""))
	assert.Equal(t, "", obj)
	assert.Equal(t, "", field)

	// Test wrong node type
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	tree, _ := parser.ParseCtx(context.Background(), nil, []byte("package p"))
	defer tree.Close()

	// Pass a non-selector_expression node
	obj, field = ParseSelectorExpression(tree.RootNode(), []byte("package p"))
	assert.Equal(t, "", obj)
	assert.Equal(t, "", field)
}

// Helper: recursively find the first call_expression node.
func findCallExpression(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "call_expression" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findCallExpression(child); result != nil {
			return result
		}
	}
	return nil
}

// Helper: recursively find the first selector_expression node.
func findSelectorExpression(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "selector_expression" {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if result := findSelectorExpression(child); result != nil {
			return result
		}
	}
	return nil
}
