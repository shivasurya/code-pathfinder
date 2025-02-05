package java

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseVariableOrField(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	variableName := ""
	variableType := ""
	variableModifier := ""
	variableValue := ""
	var scope string
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
	if node.Type() == "local_variable_declaration" {
		scope = "local"
		//nolint:all
		// hasAccessValue = hasAccess(node.NextSibling(), variableName, sourceCode)
	} else {
		scope = "field"
	}
	// Create a new node for the variable
	variableNode := &model.Node{
		ID:               util.GenerateSha256(variableName, []string{}, file),
		Type:             "variable_declaration",
		Name:             variableName,
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1,
		Modifier:         ExtractVisibilityModifier(variableModifier),
		DataType:         variableType,
		Scope:            scope,
		VariableValue:    variableValue,
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
	}
	return variableNode
}
