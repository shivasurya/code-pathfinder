package graph

import (
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/model"
	sitter "github.com/smacker/go-tree-sitter"
)

// parseJavaBinaryExpression parses Java binary expressions.
func parseJavaBinaryExpression(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) *Node {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	operator := node.ChildByFieldName("operator")
	operatorType := operator.Type()
	expressionNode := model.BinaryExpr{}
	expressionNode.LeftOperand = &model.Expr{Node: *leftNode, NodeString: leftNode.Content(sourceCode)}
	expressionNode.RightOperand = &model.Expr{Node: *rightNode, NodeString: rightNode.Content(sourceCode)}
	expressionNode.Op = operatorType
	
	var exprType string
	switch operatorType {
	case "+":
		exprType = "add_expression"
	case "-":
		exprType = "sub_expression"
	case "*":
		exprType = "mul_expression"
	case "/":
		exprType = "div_expression"
	case ">", "<", ">=", "<=":
		exprType = "comp_expression"
	case "%":
		exprType = "rem_expression"
	case ">>":
		exprType = "right_shift_expression"
	case "<<":
		exprType = "left_shift_expression"
	case "!=":
		exprType = "ne_expression"
	case "==":
		exprType = "eq_expression"
	case "&":
		exprType = "bitwise_and_expression"
	case "&&":
		exprType = "and_expression"
	case "||":
		exprType = "or_expression"
	case "|":
		exprType = "bitwise_or_expression"
	case ">>>":
		exprType = "bitwise_right_shift_expression"
	case "^":
		exprType = "bitwise_xor_expression"
	default:
		exprType = "binary_expression"
	}

	exprNode := &Node{
		ID:               GenerateSha256(exprType + node.Content(sourceCode)),
		Type:             exprType,
		Name:             node.Content(sourceCode),
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		BinaryExpr:       &expressionNode,
	}
	graph.AddNode(exprNode)

	invokedNode := &Node{
		ID:               GenerateSha256("binary_expression" + node.Content(sourceCode)),
		Type:             "binary_expression",
		Name:             node.Content(sourceCode),
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		BinaryExpr:       &expressionNode,
	}
	graph.AddNode(invokedNode)
	return invokedNode
}

// parseJavaMethodDeclaration parses Java method declarations.
func parseJavaMethodDeclaration(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) *Node {
	var javadoc *model.Javadoc
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
		commentContent := node.PrevSibling().Content(sourceCode)
		if strings.HasPrefix(commentContent, "/*") {
			javadoc = parseJavadocTags(commentContent)
		}
	}
	methodName, methodID := extractMethodName(node, sourceCode, file)
	modifiers := ""
	returnType := ""
	throws := []string{}
	methodArgumentType := []string{}
	methodArgumentValue := []string{}
	annotationMarkers := []string{}

	for i := 0; i < int(node.ChildCount()); i++ {
		childNode := node.Child(i)
		childType := childNode.Type()

		switch childType {
		case "throws":
			for j := 0; j < int(childNode.NamedChildCount()); j++ {
				namedChild := childNode.NamedChild(j)
				if namedChild.Type() == "type_identifier" {
					throws = append(throws, namedChild.Content(sourceCode))
				}
			}
		case "modifiers":
			modifiers = childNode.Content(sourceCode)
			for j := 0; j < int(childNode.ChildCount()); j++ {
				if childNode.Child(j).Type() == "marker_annotation" {
					annotationMarkers = append(annotationMarkers, childNode.Child(j).Content(sourceCode))
				}
			}
		case "void_type", "type_identifier":
			returnType = childNode.Content(sourceCode)
		case "formal_parameters":
			for j := 0; j < int(childNode.NamedChildCount()); j++ {
				param := childNode.NamedChild(j)
				if param.Type() == "formal_parameter" {
					paramType := param.Child(0).Content(sourceCode)
					paramValue := param.Child(1).Content(sourceCode)
					methodArgumentType = append(methodArgumentType, paramType)
					methodArgumentValue = append(methodArgumentValue, paramValue)
				}
			}
		}
	}

	invokedNode := &Node{
		ID:                   methodID,
		Type:                 "method_declaration",
		Name:                 methodName,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:           node.StartPoint().Row + 1,
		Modifier:             extractVisibilityModifier(modifiers),
		ReturnType:           returnType,
		MethodArgumentsType:  methodArgumentType,
		MethodArgumentsValue: methodArgumentValue,
		File:                 file,
		isJavaSourceFile:     true,
		ThrowsExceptions:     throws,
		Annotation:           annotationMarkers,
		JavaDoc:              javadoc,
	}
	graph.AddNode(invokedNode)
	return invokedNode
}

// parseJavaMethodInvocation parses Java method invocations.
func parseJavaMethodInvocation(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *Node, file string) {
	methodName, methodID := extractMethodName(node, sourceCode, file)
	arguments := []string{}
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

	invokedNode := &Node{
		ID:                   methodID,
		Type:                 "method_invocation",
		Name:                 methodName,
		IsExternal:           true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:           node.StartPoint().Row + 1,
		MethodArgumentsValue: arguments,
		File:                 file,
		isJavaSourceFile:     true,
	}
	graph.AddNode(invokedNode)

	if currentContext != nil {
		graph.AddEdge(currentContext, invokedNode)
	}
}

// parseJavaClassDeclaration parses Java class declarations.
func parseJavaClassDeclaration(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	var javadoc *model.Javadoc
	if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
		commentContent := node.PrevSibling().Content(sourceCode)
		if strings.HasPrefix(commentContent, "/*") {
			javadoc = parseJavadocTags(commentContent)
		}
	}
	className := node.ChildByFieldName("name").Content(sourceCode)
	packageName := ""
	accessModifier := ""
	superClass := ""
	annotationMarkers := []string{}
	implementedInterface := []string{}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "modifiers":
			accessModifier = child.Content(sourceCode)
			for j := 0; j < int(child.ChildCount()); j++ {
				if child.Child(j).Type() == "marker_annotation" {
					annotationMarkers = append(annotationMarkers, child.Child(j).Content(sourceCode))
				}
			}
		case "superclass":
			for j := 0; j < int(child.ChildCount()); j++ {
				if child.Child(j).Type() == "type_identifier" {
					superClass = child.Child(j).Content(sourceCode)
				}
			}
		case "super_interfaces":
			for j := 0; j < int(child.ChildCount()); j++ {
				typeList := child.Child(j)
				for k := 0; k < int(typeList.ChildCount()); k++ {
					implementedInterface = append(implementedInterface, typeList.Child(k).Content(sourceCode))
				}
			}
		}
	}

	classNode := &Node{
		ID:               GenerateMethodID("class:"+className, []string{}, file),
		Type:             "class_declaration",
		Name:             className,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		PackageName:      packageName,
		Modifier:         extractVisibilityModifier(accessModifier),
		SuperClass:       superClass,
		Interface:        implementedInterface,
		File:             file,
		isJavaSourceFile: true,
		JavaDoc:          javadoc,
		Annotation:       annotationMarkers,
	}
	graph.AddNode(classNode)
}

// parseJavaBlockComment parses Java block comments.
func parseJavaBlockComment(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	if strings.HasPrefix(node.Content(sourceCode), "/*") {
		commentContent := node.Content(sourceCode)
		javadocTags := parseJavadocTags(commentContent)

		commentNode := &Node{
			ID:               GenerateMethodID(node.Content(sourceCode), []string{}, file),
			Type:             "block_comment",
			SourceLocation:   &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
			LineNumber:       node.StartPoint().Row + 1,
			File:             file,
			isJavaSourceFile: true,
			JavaDoc:          javadocTags,
		}
		graph.AddNode(commentNode)
	}
}

// parseJavaVariableDeclaration parses Java variable declarations (local and field).
func parseJavaVariableDeclaration(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	variableName := ""
	variableType := ""
	variableModifier := ""
	variableValue := ""
	hasAccessValue := false
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
				if child.Child(j).Type() == "=" {
					for k := j + 1; k < int(child.ChildCount()); k++ {
						variableValue += child.Child(k).Content(sourceCode)
					}
				}

			}
			variableValue = strings.ReplaceAll(variableValue, " ", "")
			variableValue = strings.ReplaceAll(variableValue, "\n", "")
		case "modifiers":
			variableModifier = child.Content(sourceCode)
		}
		if strings.Contains(child.Type(), "type") {
			variableType = child.Content(sourceCode)
		}
	}
	if node.Type() == "local_variable_declaration" {
		scope = "local"
	} else {
		scope = "field"
	}
	variableNode := &Node{
		ID:               GenerateMethodID(variableName, []string{}, file),
		Type:             "variable_declaration",
		Name:             variableName,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		Modifier:         extractVisibilityModifier(variableModifier),
		DataType:         variableType,
		Scope:            scope,
		VariableValue:    variableValue,
		hasAccess:        hasAccessValue,
		File:             file,
		isJavaSourceFile: true,
	}
	graph.AddNode(variableNode)
}

// parseJavaObjectCreation parses Java object creation expressions.
func parseJavaObjectCreation(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	className := ""
	classInstanceExpression := model.ClassInstanceExpr{
		ClassName: "",
		Args:      []*model.Expr{},
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_identifier" || child.Type() == "scoped_type_identifier" {
			className = child.Content(sourceCode)
			classInstanceExpression.ClassName = className
		}
		if child.Type() == "argument_list" {
			classInstanceExpression.Args = []*model.Expr{}
			for j := 0; j < int(child.ChildCount()); j++ {
				argType := child.Child(j).Type()
				argumentStopWords := map[string]bool{
					"(": true, ")": true, "{": true, "}": true,
					"[": true, "]": true, ",": true,
				}
				if !argumentStopWords[argType] {
					argument := &model.Expr{}
					argument.Type = child.Child(j).Type()
					argument.NodeString = child.Child(j).Content(sourceCode)
					classInstanceExpression.Args = append(classInstanceExpression.Args, argument)
				}
			}
		}
	}

	objectNode := &Node{
		ID:                GenerateMethodID(className, []string{strconv.Itoa(int(node.StartPoint().Row + 1))}, file),
		Type:              "ClassInstanceExpr",
		Name:              className,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:        node.StartPoint().Row + 1,
		File:              file,
		isJavaSourceFile:  true,
		ClassInstanceExpr: &classInstanceExpression,
	}
	graph.AddNode(objectNode)
}
