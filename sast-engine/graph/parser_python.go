package graph

import (
	"fmt"

	pythonlang "github.com/shivasurya/code-pathfinder/sast-engine/graph/python"
	sitter "github.com/smacker/go-tree-sitter"
)

// parsePythonFunctionDefinition parses Python function definitions.
func parsePythonFunctionDefinition(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) *Node {
	// Extract function name and parameters
	functionName := ""
	parameters := []string{}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		functionName = nameNode.Content(sourceCode)
	}

	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode != nil {
		for i := 0; i < int(parametersNode.NamedChildCount()); i++ {
			param := parametersNode.NamedChild(i)
			if param.Type() == "identifier" || param.Type() == "typed_parameter" || param.Type() == "default_parameter" {
				parameters = append(parameters, param.Content(sourceCode))
			}
		}
	}

	methodID := GenerateMethodID("function:"+functionName, parameters, file)
	functionNode := &Node{
		ID:                   methodID,
		Type:                 "function_definition",
		Name:                 functionName,
		SourceLocation:       &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:           node.StartPoint().Row + 1,
		MethodArgumentsValue: parameters,
		File:                 file,
		isPythonSourceFile:   true,
	}
	graph.AddNode(functionNode)
	return functionNode
}

// parsePythonClassDefinition parses Python class definitions.
func parsePythonClassDefinition(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	// Extract class name and bases
	className := ""
	superClasses := []string{}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		className = nameNode.Content(sourceCode)
	}

	superclassNode := node.ChildByFieldName("superclasses")
	if superclassNode != nil {
		for i := 0; i < int(superclassNode.NamedChildCount()); i++ {
			superClass := superclassNode.NamedChild(i)
			if superClass.Type() == "identifier" || superClass.Type() == "attribute" {
				superClasses = append(superClasses, superClass.Content(sourceCode))
			}
		}
	}

	classNode := &Node{
		ID:                 GenerateMethodID("class:"+className, []string{}, file),
		Type:               "class_definition",
		Name:               className,
		SourceLocation:     &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:         node.StartPoint().Row + 1,
		Interface:          superClasses,
		File:               file,
		isPythonSourceFile: true,
	}
	graph.AddNode(classNode)
}

// parsePythonCall parses Python function calls.
func parsePythonCall(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *Node, file string) {
	// Python function calls
	callName := ""
	arguments := []string{}

	functionNode := node.ChildByFieldName("function")
	if functionNode != nil {
		callName = functionNode.Content(sourceCode)
	}

	argumentsNode := node.ChildByFieldName("arguments")
	if argumentsNode != nil {
		for i := 0; i < int(argumentsNode.NamedChildCount()); i++ {
			arg := argumentsNode.NamedChild(i)
			arguments = append(arguments, arg.Content(sourceCode))
		}
	}

	callID := GenerateMethodID(callName, arguments, file)
	callNode := &Node{
		ID:                   callID,
		Type:                 "call",
		Name:                 callName,
		IsExternal:           true,
		SourceLocation:       &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:           node.StartPoint().Row + 1,
		MethodArgumentsValue: arguments,
		File:                 file,
		isPythonSourceFile:   true,
	}
	graph.AddNode(callNode)
	if currentContext != nil {
		graph.AddEdge(currentContext, callNode)
	}
}

// parsePythonReturnStatement parses Python return statements.
func parsePythonReturnStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	returnNode := pythonlang.ParseReturnStatement(node, sourceCode)
	uniqueReturnID := fmt.Sprintf("return_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	returnStmtNode := &Node{
		ID:                 GenerateSha256(uniqueReturnID),
		Type:               "ReturnStmt",
		LineNumber:         node.StartPoint().Row + 1,
		Name:               "ReturnStmt",
		IsExternal:         true,
		SourceLocation:     &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		ReturnStmt:         returnNode,
	}
	graph.AddNode(returnStmtNode)
}

// parsePythonBreakStatement parses Python break statements.
func parsePythonBreakStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	breakNode := pythonlang.ParseBreakStatement(node, sourceCode)
	uniquebreakstmtID := fmt.Sprintf("breakstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	breakStmtNode := &Node{
		ID:                 GenerateSha256(uniquebreakstmtID),
		Type:               "BreakStmt",
		LineNumber:         node.StartPoint().Row + 1,
		Name:               "BreakStmt",
		IsExternal:         true,
		SourceLocation:     &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		BreakStmt:          breakNode,
	}
	graph.AddNode(breakStmtNode)
}

// parsePythonContinueStatement parses Python continue statements.
func parsePythonContinueStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	continueNode := pythonlang.ParseContinueStatement(node, sourceCode)
	uniquecontinueID := fmt.Sprintf("continuestmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	continueStmtNode := &Node{
		ID:                 GenerateSha256(uniquecontinueID),
		Type:               "ContinueStmt",
		LineNumber:         node.StartPoint().Row + 1,
		Name:               "ContinueStmt",
		IsExternal:         true,
		SourceLocation:     &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		ContinueStmt:       continueNode,
	}
	graph.AddNode(continueStmtNode)
}

// parsePythonAssertStatement parses Python assert statements.
func parsePythonAssertStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	assertNode := pythonlang.ParseAssertStatement(node, sourceCode)
	uniqueAssertID := fmt.Sprintf("assert_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	assertStmtNode := &Node{
		ID:                 GenerateSha256(uniqueAssertID),
		Type:               "AssertStmt",
		LineNumber:         node.StartPoint().Row + 1,
		Name:               "AssertStmt",
		IsExternal:         true,
		SourceLocation:     &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		AssertStmt:         assertNode,
	}
	graph.AddNode(assertStmtNode)
}

// parsePythonYieldExpression parses Python yield expressions.
func parsePythonYieldExpression(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	// Handle yield expressions in Python
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "yield" {
			yieldNode := pythonlang.ParseYieldStatement(child, sourceCode)
			uniqueyieldID := fmt.Sprintf("yield_%d_%d_%s", child.StartPoint().Row+1, child.StartPoint().Column+1, file)
			yieldStmtNode := &Node{
				ID:                 GenerateSha256(uniqueyieldID),
				Type:               "YieldStmt",
				LineNumber:         child.StartPoint().Row + 1,
				Name:               "YieldStmt",
				IsExternal:         true,
				SourceLocation:     &SourceLocation{
					File:      file,
					StartByte: child.StartByte(),
					EndByte:   child.EndByte(),
				},
				File:               file,
				isPythonSourceFile: true,
				YieldStmt:          yieldNode,
			}
			graph.AddNode(yieldStmtNode)
			break
		}
	}
}

// parsePythonAssignment parses Python variable assignments.
func parsePythonAssignment(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	// Python variable assignments
	variableName := ""
	variableValue := ""

	leftNode := node.ChildByFieldName("left")
	if leftNode != nil {
		variableName = leftNode.Content(sourceCode)
	}

	rightNode := node.ChildByFieldName("right")
	if rightNode != nil {
		variableValue = rightNode.Content(sourceCode)
	}

	variableNode := &Node{
		ID:                 GenerateMethodID(variableName, []string{}, file),
		Type:               "variable_assignment",
		Name:               variableName,
		SourceLocation:     &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:         node.StartPoint().Row + 1,
		VariableValue:      variableValue,
		Scope:              "local",
		File:               file,
		isPythonSourceFile: true,
	}
	graph.AddNode(variableNode)
}
