package java

import (
	"github.com/smacker/go-tree-sitter/java"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
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
