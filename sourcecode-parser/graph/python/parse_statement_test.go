package python

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

func TestParseReturnStatement(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "return with value",
			code:     "return 42",
			expected: "42",
		},
		{
			name:     "return without value",
			code:     "return",
			expected: "",
		},
		{
			name:     "return with expression",
			code:     "return x + y",
			expected: "x + y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(python.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			returnNode := root.Child(0)

			result := ParseReturnStatement(returnNode, []byte(tt.code))

			if result == nil {
				t.Fatal("ParseReturnStatement returned nil")
			}

			switch {
			case tt.expected == "" && result.Result != nil:
				t.Errorf("Expected nil result, got %v", result.Result)
			case tt.expected != "" && result.Result == nil:
				t.Errorf("Expected result %s, got nil", tt.expected)
			case tt.expected != "" && result.Result != nil && result.Result.NodeString != tt.expected:
				t.Errorf("Expected result %s, got %s", tt.expected, result.Result.NodeString)
			}
		})
	}
}

func TestParseBreakStatement(t *testing.T) {
	code := "break"
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	breakNode := root.Child(0)

	result := ParseBreakStatement(breakNode, []byte(code))

	if result == nil {
		t.Fatal("ParseBreakStatement returned nil")
	}

	// Python break statements don't have labels
	if result.Label != "" {
		t.Errorf("Expected empty label, got %s", result.Label)
	}
}

func TestParseContinueStatement(t *testing.T) {
	code := "continue"
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	continueNode := root.Child(0)

	result := ParseContinueStatement(continueNode, []byte(code))

	if result == nil {
		t.Fatal("ParseContinueStatement returned nil")
	}

	// Python continue statements don't have labels
	if result.Label != "" {
		t.Errorf("Expected empty label, got %s", result.Label)
	}
}

func TestParseAssertStatement(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectedExpr    string
		expectedMessage string
	}{
		{
			name:            "assert without message",
			code:            "assert x > 0",
			expectedExpr:    "x > 0",
			expectedMessage: "",
		},
		{
			name:            "assert with message",
			code:            "assert x > 0, \"x must be positive\"",
			expectedExpr:    "x > 0",
			expectedMessage: "\"x must be positive\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(python.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			assertNode := root.Child(0)

			result := ParseAssertStatement(assertNode, []byte(tt.code))

			if result == nil {
				t.Fatal("ParseAssertStatement returned nil")
			}

			if result.Expr == nil {
				t.Fatal("Expected non-nil Expr")
			}

			if result.Expr.NodeString != tt.expectedExpr {
				t.Errorf("Expected expr %s, got %s", tt.expectedExpr, result.Expr.NodeString)
			}

			if tt.expectedMessage == "" && result.Message != nil {
				t.Errorf("Expected nil message, got %v", result.Message)
			} else if tt.expectedMessage != "" && result.Message == nil {
				t.Errorf("Expected message %s, got nil", tt.expectedMessage)
			}
		})
	}
}

func TestParseYieldStatement(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "yield with value",
			code:     "yield 42",
			expected: "42",
		},
		{
			name:     "yield with expression",
			code:     "yield x + y",
			expected: "x + y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(python.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}
			defer tree.Close()

			root := tree.RootNode()
			// Navigate to the actual yield node: module -> expression_statement -> yield
			exprStmt := root.Child(0)
			yieldNode := exprStmt.Child(0)

			result := ParseYieldStatement(yieldNode, []byte(tt.code))

			if result == nil {
				t.Fatal("ParseYieldStatement returned nil")
			}

			if result.Value == nil {
				t.Fatal("Expected non-nil Value")
			}

			if result.Value.NodeString != tt.expected {
				t.Errorf("Expected value %s, got %s", tt.expected, result.Value.NodeString)
			}
		})
	}
}

func TestParseBlockStatement(t *testing.T) {
	code := `if True:
    x = 1
    y = 2`

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	ifNode := root.Child(0)
	blockNode := ifNode.ChildByFieldName("consequence")

	result := ParseBlockStatement(blockNode, []byte(code))

	if result == nil {
		t.Fatal("ParseBlockStatement returned nil")
	}

	if len(result.Stmts) == 0 {
		t.Error("Expected non-empty statements list")
	}
}

func BenchmarkParseReturnStatement(b *testing.B) {
	code := "return x + y"
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(python.GetLanguage())

	tree, _ := parser.ParseCtx(context.Background(), nil, []byte(code))
	defer tree.Close()
	root := tree.RootNode()
	returnNode := root.Child(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseReturnStatement(returnNode, []byte(code))
	}
}
