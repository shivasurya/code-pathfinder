package graph

import sitter "github.com/smacker/go-tree-sitter"

// buildGraphFromAST builds a code graph from an Abstract Syntax Tree.
func buildGraphFromAST(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *Node, file string) {
	isJavaSourceFile := isJavaSourceFile(file)
	isPythonSourceFile := isPythonSourceFile(file)
	
	switch node.Type() {
	// Python-specific node types
	case "function_definition":
		if isPythonSourceFile {
			currentContext = parsePythonFunctionDefinition(node, sourceCode, graph, file)
		}

	case "class_definition":
		if isPythonSourceFile {
			parsePythonClassDefinition(node, sourceCode, graph, file)
		}

	case "call":
		if isPythonSourceFile {
			parsePythonCall(node, sourceCode, graph, currentContext, file)
		}

	case "return_statement":
		parseReturnStatement(node, sourceCode, graph, file, isJavaSourceFile, isPythonSourceFile)

	case "break_statement":
		parseBreakStatement(node, sourceCode, graph, file, isJavaSourceFile, isPythonSourceFile)

	case "continue_statement":
		parseContinueStatement(node, sourceCode, graph, file, isJavaSourceFile, isPythonSourceFile)

	case "assert_statement":
		parseAssertStatement(node, sourceCode, graph, file, isJavaSourceFile, isPythonSourceFile)

	case "expression_statement":
		if isPythonSourceFile {
			parsePythonYieldExpression(node, sourceCode, graph, file)
		}

	case "assignment":
		if isPythonSourceFile {
			parsePythonAssignment(node, sourceCode, graph, file)
		}

	// Java-specific node types
	case "block":
		parseBlockStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "yield_statement":
		parseYieldStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "if_statement":
		parseIfStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "while_statement":
		parseWhileStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "do_statement":
		parseDoStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "for_statement":
		parseForStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "binary_expression":
		currentContext = parseJavaBinaryExpression(node, sourceCode, graph, file, isJavaSourceFile)

	case "method_declaration":
		currentContext = parseJavaMethodDeclaration(node, sourceCode, graph, file)

	case "method_invocation":
		parseJavaMethodInvocation(node, sourceCode, graph, currentContext, file)

	case "class_declaration":
		parseJavaClassDeclaration(node, sourceCode, graph, file)

	case "block_comment":
		parseJavaBlockComment(node, sourceCode, graph, file)

	case "local_variable_declaration", "field_declaration":
		parseJavaVariableDeclaration(node, sourceCode, graph, file)

	case "object_creation_expression":
		parseJavaObjectCreation(node, sourceCode, graph, file)
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, graph, currentContext, file)
	}

	// Post-processing: Link method invocations to declarations
	for _, node := range graph.Nodes {
		if node.Type == "method_declaration" {
			for _, invokedNode := range graph.Nodes {
				if invokedNode.Type == "method_invocation" {
					if invokedNode.Name == node.Name {
						if len(invokedNode.MethodArgumentsValue) == len(node.MethodArgumentsType) {
							node.hasAccess = true
						}
					}
				}
			}
		}
	}
}
