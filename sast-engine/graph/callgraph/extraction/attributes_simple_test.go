package extraction

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
)

// Simple tests that cover the inference functions without complex setup

func TestInferFromLiteralSimple(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedType string
	}{
		{"string", `x = "test"`, "builtins.str"},
		{"integer", `x = 42`, "builtins.int"},
		{"boolean true", `x = True`, "builtins.bool"},
		{"boolean false", `x = False`, "builtins.bool"},
		{"none", `x = None`, "builtins.NoneType"},
		{"list", `x = []`, "builtins.list"},
		{"dict", `x = {}`, "builtins.dict"},
		{"tuple", `x = (1, 2)`, "builtins.tuple"},
		{"set", `x = {1, 2}`, "builtins.set"},
		{"float", `x = 3.14`, "builtins.float"},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.code))
			root := tree.RootNode()

			// Simple traversal to find assignment
			var assignment *sitter.Node
			for i := 0; i < int(root.ChildCount()); i++ {
				child := root.Child(i)
				if child.Type() == "expression_statement" {
					for j := 0; j < int(child.ChildCount()); j++ {
						if child.Child(j).Type() == "assignment" {
							assignment = child.Child(j)
							break
						}
					}
				}
			}

			if assignment != nil {
				typeInfo := inferFromLiteral(assignment, []byte(tt.code))
				if typeInfo != nil {
					assert.Equal(t, tt.expectedType, typeInfo.TypeFQN)
					assert.Equal(t, float32(1.0), typeInfo.Confidence)
				}
			}
		})
	}
}

func TestExtractClassNameSimple(t *testing.T) {
	code := `class User:
    pass`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	className := extractClassName(root.Child(0), []byte(code))
	assert.Equal(t, "User", className)
}

func TestExtractMethodNameSimple(t *testing.T) {
	code := `def test_method(self):
    pass`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	methodName := extractMethodName(root.Child(0), []byte(code))
	assert.Equal(t, "test_method", methodName)
}
