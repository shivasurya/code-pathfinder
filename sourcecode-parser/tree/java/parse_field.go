package java

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func parseFieldModifers(modifiers string) []string {
	// modifier string can be like "@Override\n public strictfp"
	// trim modifier and split by new line and then by space
	modifiers = strings.TrimSpace(modifiers)
	modifiers = strings.ReplaceAll(modifiers, "\n", " ")

	modifiersArray := strings.Split(modifiers, " ")

	for i := 0; i < len(modifiersArray); i++ {
		modifiersArray[i] = strings.TrimSpace(modifiersArray[i])
	}

	return modifiersArray
}

func extractFieldVisibilityModifier(accessModifiers []string) string {
	visibilityTypes := []string{"public", "private", "protected"}
	for _, currentModifier := range accessModifiers {
		for _, visibilityType := range visibilityTypes {
			if currentModifier == visibilityType {
				return currentModifier
			}
		}
	}
	return ""
}

func hasFieldModifier(modifiers []string, targetModifier string) bool {
	for _, modifier := range modifiers {
		if modifier == targetModifier {
			return true
		}
	}
	return false
}

func ParseField(node *sitter.Node, sourceCode []byte, file string) *model.FieldDeclaration {
	var fieldDeclaration *model.FieldDeclaration
	variableName := []string{}
	variableType := ""
	variableModifier := []string{}
	variableValue := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "variable_declarator":
			variable := child.Content(sourceCode)
			for j := 0; j < int(child.ChildCount()); j++ {
				if child.Child(j).Type() == "identifier" {
					variable = child.Child(j).Content(sourceCode)
				}
				// if child type contains =, iterate through and get remaining content
				if child.Child(j).Type() == "=" {
					for k := j + 1; k < int(child.ChildCount()); k++ {
						variableValue += child.Child(k).Content(sourceCode)
					}
				}
			}
			variableName = append(variableName, variable)
			// remove spaces from variable value
			variableValue = strings.ReplaceAll(variableValue, " ", "")
			// remove new line from variable value
			variableValue = strings.ReplaceAll(variableValue, "\n", "")
		case "modifiers":
			variableModifier = parseModifers(child.Content(sourceCode))
		}
		// if child type contains type, get the type of variable
		if strings.Contains(child.Type(), "type") {
			variableType = child.Content(sourceCode)
		}
	}
	// Create a new node for the variable
	fieldDeclaration = &model.FieldDeclaration{
		Type:              variableType,
		FieldNames:        variableName,
		Visibility:        extractFieldVisibilityModifier(variableModifier),
		IsStatic:          hasFieldModifier(variableModifier, "static"),
		IsFinal:           hasFieldModifier(variableModifier, "final"),
		IsVolatile:        hasFieldModifier(variableModifier, "volatile"),
		IsTransient:       hasFieldModifier(variableModifier, "transient"),
		SourceDeclaration: file,
	}
	return fieldDeclaration
}
