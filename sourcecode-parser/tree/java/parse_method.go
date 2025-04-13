package java

import (
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func extractMethodName(node *sitter.Node, sourceCode []byte, filepath string) (string, string) {
	var methodID string

	// if the child node is method_declaration, extract method name, modifiers, parameters, and return type
	var methodName string
	var modifiers, parameters []string

	if node.Type() == "method_declaration" {
		// Iterate over all children of the method_declaration node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "modifiers", "marker_annotation", "annotation":
				// This child is a modifier or annotation, add its content to modifiers
				modifiers = append(modifiers, child.Content(sourceCode)) //nolint:all
			case "identifier":
				// This child is the method name
				methodName = child.Content(sourceCode)
			case "formal_parameters":
				// This child represents formal parameters; iterate through its children
				for j := 0; j < int(child.NamedChildCount()); j++ {
					param := child.NamedChild(j)
					parameters = append(parameters, param.Content(sourceCode))
				}
			}
		}
	}

	// check if type is method_invocation
	// if the child node is method_invocation, extract method name
	if node.Type() == "method_invocation" {
		for j := 0; j < int(node.ChildCount()); j++ {
			child := node.Child(j)
			if child.Type() == "identifier" {
				if methodName == "" {
					methodName = child.Content(sourceCode)
				} else {
					methodName = methodName + "." + child.Content(sourceCode)
				}
			}

			argumentsNode := node.ChildByFieldName("argument_list")
			// add data type of arguments list
			if argumentsNode != nil {
				for k := 0; k < int(argumentsNode.ChildCount()); k++ {
					argument := argumentsNode.Child(k)
					parameters = append(parameters, argument.Child(0).Content(sourceCode))
				}
			}

		}
	}
	content := node.Content(sourceCode)
	lineNumber := int(node.StartPoint().Row) + 1
	columnNumber := int(node.StartPoint().Column) + 1
	// convert to string and merge
	content += " " + strconv.Itoa(lineNumber) + ":" + strconv.Itoa(columnNumber)
	methodID = util.GenerateMethodID(methodName, parameters, filepath+"/"+content+"/"+strconv.Itoa(lineNumber)+":"+strconv.Itoa(columnNumber))
	return methodName, methodID
}

func parseModifers(modifiers string) []string {
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

func extractVisibilityModifier(accessModifiers []string) string {
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

func hasModifier(modifiers []string, targetModifier string) bool {
	for _, modifier := range modifiers {
		if modifier == targetModifier {
			return true
		}
	}
	return false
}

func ParseMethodDeclaration(node *sitter.Node, sourceCode []byte, file string, parentNode *model.TreeNode) *model.Method {
	methodName, methodID := extractMethodName(node, sourceCode, file)
	modifiers := []string{}
	returnType := ""
	throws := []string{}
	methodArgumentType := []string{}
	methodArgumentValue := []string{}
	annotationMarkers := []string{}
	classId := ""

	for i := 0; i < int(node.ChildCount()); i++ {
		childNode := node.Child(i)
		childType := childNode.Type()

		switch childType {
		case "throws":
			// namedChild
			for j := 0; j < int(childNode.NamedChildCount()); j++ {
				namedChild := childNode.NamedChild(j)
				if namedChild.Type() == "type_identifier" {
					throws = append(throws, namedChild.Content(sourceCode))
				}
			}
		case "modifiers":
			modifiers = parseModifers(childNode.Content(sourceCode))
			for j := 0; j < int(childNode.ChildCount()); j++ {
				if childNode.Child(j).Type() == "marker_annotation" {
					annotationMarkers = append(annotationMarkers, childNode.Child(j).Content(sourceCode))
				}
			}
		case "void_type", "type_identifier":
			// get return type of method
			returnType = childNode.Content(sourceCode)
		case "formal_parameters":
			// get method arguments
			for j := 0; j < int(childNode.NamedChildCount()); j++ {
				param := childNode.NamedChild(j)
				if param.Type() == "formal_parameter" {
					// get type of argument and add to method arguments
					paramType := param.Child(0).Content(sourceCode)
					paramValue := param.Child(1).Content(sourceCode)
					methodArgumentType = append(methodArgumentType, paramType)
					methodArgumentValue = append(methodArgumentValue, paramValue)
				}
			}
		}
	}

	if parentNode != nil && parentNode.Node.ClassDecl != nil && parentNode.Node.ClassDecl.ClassId != "" {
		classId = parentNode.Node.ClassDecl.ClassId
	}

	methodNode := &model.Method{
		Name:              methodName,
		QualifiedName:     methodName,
		ReturnType:        returnType,
		ParameterNames:    methodArgumentType,
		Parameters:        methodArgumentValue,
		Visibility:        extractVisibilityModifier(modifiers),
		IsAbstract:        hasModifier(modifiers, "abstract"),
		IsStatic:          hasModifier(modifiers, "static"),
		IsFinal:           hasModifier(modifiers, "final"),
		IsConstructor:     false,
		IsStrictfp:        hasModifier(modifiers, "strictfp"),
		SourceDeclaration: file,
		ID:                methodID,
		ClassId:           classId,
	}

	return methodNode
}

func ParseMethodInvoker(node *sitter.Node, sourceCode []byte, file string) *model.MethodCall {
	var methodCall *model.MethodCall
	methodName, _ := extractMethodName(node, sourceCode, file)
	arguments := []string{}
	// get argument list from arguments node iterate for child node
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "argument_list" {
			argumentsNode := node.Child(i)
			for j := 0; j < int(argumentsNode.ChildCount()); j++ {
				argument := argumentsNode.Child(j)
				switch argument.Type() {
				case "identifier":
					arguments = append(arguments, argument.Content(sourceCode))
				case "string_literal":
					stringliteral := argument.Content(sourceCode)
					stringliteral = strings.TrimPrefix(stringliteral, "\"")
					stringliteral = strings.TrimSuffix(stringliteral, "\"")
					arguments = append(arguments, stringliteral)
				default:
					arguments = append(arguments, argument.Content(sourceCode))
				}
			}
		}
	}
	methodCall = &model.MethodCall{
		MethodName:      methodName,
		Arguments:       arguments,
		QualifiedMethod: methodName,
		TypeArguments:   []string{},
	}
	return methodCall
}
