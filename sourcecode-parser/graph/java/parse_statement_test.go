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
