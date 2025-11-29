package java

import (
	"github.com/smacker/go-tree-sitter/java"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/model"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
)

func TestParseBreakStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *model.BreakStmt
	}{
		{
			name:     "Simple break statement without label",
			input:    "break;",
			expected: &model.BreakStmt{Label: ""},
		},
		{
			name:     "Break statement with label",
			input:    "break myLabel;",
			expected: &model.BreakStmt{Label: "myLabel"},
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			node := tree.RootNode().Child(0)
			result := ParseBreakStatement(node, []byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseContinueStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *model.ContinueStmt
	}{
		{
			name:     "Simple continue statement without label",
			input:    "continue;",
			expected: &model.ContinueStmt{Label: ""},
		},
		{
			name:     "Continue statement with label",
			input:    "continue outerLoop;",
			expected: &model.ContinueStmt{Label: "outerLoop"},
		},
		{
			name:     "Continue statement with complex label",
			input:    "continue COMPLEX_LABEL_123;",
			expected: &model.ContinueStmt{Label: "COMPLEX_LABEL_123"},
		},
		{
			name:     "Continue statement with underscore label",
			input:    "continue outer_loop_label;",
			expected: &model.ContinueStmt{Label: "outer_loop_label"},
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			node := tree.RootNode().Child(0)
			result := ParseContinueStatement(node, []byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseYieldStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *model.YieldStmt
	}{
		{
			name:  "Simple yield statement with literal",
			input: "yield 42;",
			expected: &model.YieldStmt{
				Value: &model.Expr{NodeString: "42"},
			},
		},
		{
			name:  "Yield statement with variable",
			input: "yield result;",
			expected: &model.YieldStmt{
				Value: &model.Expr{NodeString: "result"},
			},
		},
		{
			name:  "Yield statement with expression",
			input: "yield a + b;",
			expected: &model.YieldStmt{
				Value: &model.Expr{NodeString: "a + b"},
			},
		},
		{
			name:  "Yield statement with method call",
			input: "yield getValue();",
			expected: &model.YieldStmt{
				Value: &model.Expr{NodeString: "getValue()"},
			},
		},
		{
			name:  "Yield statement with string literal",
			input: "yield \"hello\";",
			expected: &model.YieldStmt{
				Value: &model.Expr{NodeString: "\"hello\""},
			},
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			node := tree.RootNode().Child(0)
			result := ParseYieldStatement(node, []byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAssertStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *model.AssertStmt
	}{
		{
			name:  "Simple assert statement without message",
			input: "assert x > 0;",
			expected: &model.AssertStmt{
				Expr:    &model.Expr{NodeString: "x > 0"},
				Message: nil,
			},
		},
		{
			name:  "Assert statement with message",
			input: "assert condition : \"Value must be positive\";",
			expected: &model.AssertStmt{
				Expr:    &model.Expr{NodeString: "condition"},
				Message: &model.Expr{NodeString: "\"Value must be positive\""},
			},
		},
		{
			name:  "Assert statement with boolean literal",
			input: "assert true;",
			expected: &model.AssertStmt{
				Expr:    &model.Expr{NodeString: "true"},
				Message: nil,
			},
		},
		{
			name:  "Assert statement with complex expression",
			input: "assert x != null && x.isValid();",
			expected: &model.AssertStmt{
				Expr:    &model.Expr{NodeString: "x != null && x.isValid()"},
				Message: nil,
			},
		},
		{
			name:  "Assert statement with method call and message",
			input: "assert obj.validate() : \"Validation failed\";",
			expected: &model.AssertStmt{
				Expr:    &model.Expr{NodeString: "obj.validate()"},
				Message: &model.Expr{NodeString: "\"Validation failed\""},
			},
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			node := tree.RootNode().Child(0)
			result := ParseAssertStatement(node, []byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseBlockStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *model.BlockStmt
	}{
		{
			name:  "Empty block statement",
			input: "{}",
			expected: &model.BlockStmt{
				Stmts: []model.Stmt{
					{NodeString: "{"},
					{NodeString: "}"},
				},
			},
		},
		{
			name:  "Single statement block",
			input: "{return true;}",
			expected: &model.BlockStmt{
				Stmts: []model.Stmt{
					{NodeString: "{"},
					{NodeString: "return true;"},
					{NodeString: "}"},
				},
			},
		},
		{
			name:  "Multiple statement block",
			input: "{int x = 1; x++; return x;}",
			expected: &model.BlockStmt{
				Stmts: []model.Stmt{
					{NodeString: "{"},
					{NodeString: "int x = 1;"},
					{NodeString: "x++;"},
					{NodeString: "return x;"},
					{NodeString: "}"},
				},
			},
		},
		{
			name:  "Nested block statements",
			input: "{{int x = 1;}{int y = 2;}}",
			expected: &model.BlockStmt{
				Stmts: []model.Stmt{
					{NodeString: "{"},
					{NodeString: "{int x = 1;}"},
					{NodeString: "{int y = 2;}"},
					{NodeString: "}"},
				},
			},
		},
		{
			name:  "Block with complex statements",
			input: "{System.out.println(\"Hello\"); if(x > 0) { return true; } throw new Exception();}",
			expected: &model.BlockStmt{
				Stmts: []model.Stmt{
					{NodeString: "{"},
					{NodeString: "System.out.println(\"Hello\");"},
					{NodeString: "if(x > 0) { return true; }"},
					{NodeString: "throw new Exception();"},
					{NodeString: "}"},
				},
			},
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			node := tree.RootNode().Child(0)
			result := ParseBlockStatement(node, []byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}
