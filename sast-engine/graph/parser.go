package graph

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	sitter "github.com/smacker/go-tree-sitter"
)

// buildGraphFromAST builds a code graph from an Abstract Syntax Tree.
func buildGraphFromAST(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *Node, file string) {
	isJavaSourceFile := isJavaSourceFile(file)
	isPythonSourceFile := isPythonSourceFile(file)
	isGoSourceFile := isGoSourceFile(file)
	isCFile := clike.IsCSourceFile(file)
	isCppFile := clike.IsCppSourceFile(file)

	switch node.Type() {
	// Python and C share the function_definition node type — dispatch by
	// language. C/C++ branches come first because the dispatcher is the
	// only place these node types are handled.
	case "function_definition":
		if isCFile {
			currentContext = parseCFunctionDefinition(node, sourceCode, graph, file)
		} else if isPythonSourceFile {
			currentContext = parsePythonFunctionDefinition(node, sourceCode, graph, file, currentContext)
		}

	case "class_definition":
		if isPythonSourceFile {
			currentContext = parsePythonClassDefinition(node, sourceCode, graph, file)
		}

	case "call":
		if isPythonSourceFile {
			parsePythonCall(node, sourceCode, graph, currentContext, file)
		}

	case "return_statement":
		parseReturnStatement(node, sourceCode, graph, file, isJavaSourceFile, isPythonSourceFile)
		if isGoSourceFile {
			parseGoReturnStatement(node, sourceCode, graph, file)
		}

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
			parsePythonAssignment(node, sourceCode, graph, file, currentContext)
		}

	// Java-specific node types
	case "block":
		parseBlockStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "yield_statement":
		parseYieldStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "if_statement":
		parseIfStatement(node, sourceCode, graph, file, isJavaSourceFile)
		if isGoSourceFile {
			parseGoIfStatement(node, sourceCode, graph, file)
		}

	case "while_statement":
		parseWhileStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "do_statement":
		parseDoStatement(node, sourceCode, graph, file, isJavaSourceFile)

	case "for_statement":
		parseForStatement(node, sourceCode, graph, file, isJavaSourceFile)
		if isGoSourceFile {
			parseGoForStatement(node, sourceCode, graph, file)
		}

	case "binary_expression":
		currentContext = parseJavaBinaryExpression(node, sourceCode, graph, file, isJavaSourceFile)

	case "method_declaration":
		if isJavaSourceFile {
			currentContext = parseJavaMethodDeclaration(node, sourceCode, graph, file)
		} else if isGoSourceFile {
			currentContext = parseGoMethodDeclaration(node, sourceCode, graph, file)
		}

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

	// Go-specific node types (stubs for PR-02+)
	case "function_declaration":
		if isGoSourceFile {
			currentContext = parseGoFunctionDeclaration(node, sourceCode, graph, file)
		}

	case "type_declaration":
		if isGoSourceFile {
			parseGoTypeDeclaration(node, sourceCode, graph, file)
		}

	case "call_expression":
		if isCFile {
			parseCCallExpression(node, sourceCode, graph, file, currentContext)
		} else if isGoSourceFile {
			parseGoCallExpression(node, sourceCode, graph, file, currentContext)
		}

	// C/C++ specific node types. struct_specifier appears in C only at the
	// top level (C++ uses class_specifier for the equivalent construct);
	// the remaining four are shared between C and C++.
	case "struct_specifier":
		if isCFile {
			parseCStructSpecifier(node, sourceCode, graph, file)
		}

	case "enum_specifier":
		if isCFile || isCppFile {
			parseCEnumSpecifier(node, sourceCode, graph, file)
		}

	case "type_definition":
		if isCFile || isCppFile {
			parseCTypeDefinition(node, sourceCode, graph, file)
		}

	case "declaration":
		if isCFile || isCppFile {
			parseCLikeDeclaration(node, sourceCode, graph, file, currentContext, isCppFile)
		}

	case "preproc_include":
		if isCFile || isCppFile {
			parseCLikeInclude(node, sourceCode, graph, file, isCppFile)
		}

	case "short_var_declaration":
		if isGoSourceFile {
			parseGoShortVarDeclaration(node, sourceCode, graph, file)
		}

	case "var_declaration":
		if isGoSourceFile {
			parseGoVarDeclaration(node, sourceCode, graph, file)
		}

	case "const_declaration":
		if isGoSourceFile {
			parseGoConstDeclaration(node, sourceCode, graph, file)
		}

	case "func_literal":
		if isGoSourceFile {
			currentContext = parseGoFuncLiteral(node, sourceCode, graph, file, currentContext)
		}

	case "defer_statement":
		if isGoSourceFile {
			parseGoDeferStatement(node, sourceCode, graph, file, currentContext)
		}

	case "go_statement":
		if isGoSourceFile {
			parseGoGoStatement(node, sourceCode, graph, file, currentContext)
		}

	case "assignment_statement":
		if isGoSourceFile {
			parseGoAssignment(node, sourceCode, graph, file)
		}
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, graph, currentContext, file)
	}
}
