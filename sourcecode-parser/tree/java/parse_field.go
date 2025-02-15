package java

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseField(node *sitter.Node, sourceCode []byte, file string) *model.FieldDeclaration {
	var fieldDeclaration *model.FieldDeclaration
	variableName := ""
	variableType := ""
	variableModifier := ""
	variableValue := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "variable_declarator":
			variableName = child.Content(sourceCode)
			for j := 0; j < int(child.ChildCount()); j++ {
				if child.Child(j).Type() == "identifier" {
					variableName = child.Child(j).Content(sourceCode)
				}
				// if child type contains =, iterate through and get remaining content
				if child.Child(j).Type() == "=" {
					for k := j + 1; k < int(child.ChildCount()); k++ {
						variableValue += child.Child(k).Content(sourceCode)
					}
				}

			}
			// remove spaces from variable value
			variableValue = strings.ReplaceAll(variableValue, " ", "")
			// remove new line from variable value
			variableValue = strings.ReplaceAll(variableValue, "\n", "")
		case "modifiers":
			variableModifier = child.Content(sourceCode)
		}
		// if child type contains type, get the type of variable
		if strings.Contains(child.Type(), "type") {
			variableType = child.Content(sourceCode)
		}
	}
	// Create a new node for the variable
	fieldDeclaration = &model.FieldDeclaration{
		Type:       variableType,
		FieldNames: []string{variableName},
		Visibility: variableModifier,
	}
	return fieldDeclaration
}
