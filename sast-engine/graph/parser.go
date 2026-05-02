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
		switch {
		case isCFile:
			currentContext = parseCFunctionDefinition(node, sourceCode, graph, file)
		case isCppFile:
			currentContext = parseCppFunctionDefinition(node, sourceCode, graph, file, currentContext)
		case isPythonSourceFile:
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

	// Java-specific node types. The C/C++ grammars emit several of these
	// same node types (block, if/while/do/for, binary_expression) with
	// different AST shapes, so the Java handlers must be gated by language —
	// otherwise they pollute C/C++ graphs with Java-tagged nodes.
	case "block":
		if isJavaSourceFile {
			parseBlockStatement(node, sourceCode, graph, file, isJavaSourceFile)
		}

	case "yield_statement":
		if isJavaSourceFile {
			parseYieldStatement(node, sourceCode, graph, file, isJavaSourceFile)
		}

	case "if_statement":
		if isJavaSourceFile {
			parseIfStatement(node, sourceCode, graph, file, isJavaSourceFile)
		}
		if isGoSourceFile {
			parseGoIfStatement(node, sourceCode, graph, file)
		}

	case "while_statement":
		if isJavaSourceFile {
			parseWhileStatement(node, sourceCode, graph, file, isJavaSourceFile)
		}

	case "do_statement":
		if isJavaSourceFile {
			parseDoStatement(node, sourceCode, graph, file, isJavaSourceFile)
		}

	case "for_statement":
		if isJavaSourceFile {
			parseForStatement(node, sourceCode, graph, file, isJavaSourceFile)
		}
		if isGoSourceFile {
			parseGoForStatement(node, sourceCode, graph, file)
		}

	case "binary_expression":
		if isJavaSourceFile {
			currentContext = parseJavaBinaryExpression(node, sourceCode, graph, file, isJavaSourceFile)
		}

	case "method_declaration":
		if isJavaSourceFile {
			currentContext = parseJavaMethodDeclaration(node, sourceCode, graph, file)
		} else if isGoSourceFile {
			currentContext = parseGoMethodDeclaration(node, sourceCode, graph, file)
		}

	case "method_invocation":
		parseJavaMethodInvocation(node, sourceCode, graph, currentContext, file)

	case "class_declaration":
		if isJavaSourceFile {
			parseJavaClassDeclaration(node, sourceCode, graph, file)
		}

	case "block_comment":
		if isJavaSourceFile {
			parseJavaBlockComment(node, sourceCode, graph, file)
		}

	case "local_variable_declaration":
		parseJavaVariableDeclaration(node, sourceCode, graph, file)

	case "field_declaration":
		// tree-sitter overloads field_declaration:
		//   - Java: class fields (handled by parseJavaVariableDeclaration)
		//   - C:    struct fields (handled by parseCStructSpecifier via clike;
		//           the bare nodes here are siblings of an already-recorded
		//           struct, so we skip them to avoid duplicate nodes)
		//   - C++:  data members AND inline method declarations inside a
		//           class body (handled by parseCppFieldDeclaration)
		if isCppFile {
			parseCppFieldDeclaration(node, sourceCode, graph, file, currentContext)
		} else if !isCFile {
			parseJavaVariableDeclaration(node, sourceCode, graph, file)
		}

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
		switch {
		case isCFile:
			parseCCallExpression(node, sourceCode, graph, file, currentContext)
		case isCppFile:
			parseCppCallExpression(node, sourceCode, graph, file, currentContext)
		case isGoSourceFile:
			parseGoCallExpression(node, sourceCode, graph, file, currentContext)
		}

	// C and C++ shared node types — each language has its own parse
	// function that sets the right Language tag and handles language-
	// specific concerns (e.g. C++ struct inheritance via base_class_clause).
	case "struct_specifier":
		if isCFile {
			parseCStructSpecifier(node, sourceCode, graph, file)
		} else if isCppFile {
			parseCppStructSpecifier(node, sourceCode, graph, file)
		}

	case "enum_specifier":
		if isCFile {
			parseCEnumSpecifier(node, sourceCode, graph, file)
		} else if isCppFile {
			parseCppEnumSpecifier(node, sourceCode, graph, file)
		}

	case "type_definition":
		if isCFile {
			parseCTypeDefinition(node, sourceCode, graph, file)
		} else if isCppFile {
			parseCppTypeDefinition(node, sourceCode, graph, file)
		}

	case "declaration":
		if isCFile || isCppFile {
			parseCLikeDeclaration(node, sourceCode, graph, file, currentContext, isCppFile)
		}

	case "preproc_include":
		if isCFile || isCppFile {
			parseCLikeInclude(node, sourceCode, graph, file, isCppFile)
		}

	// C++-only node types. The dispatcher returns the new node from
	// class_specifier and namespace_definition so the recursion picks up
	// the surrounding scope as currentContext for member resolution.
	case "class_specifier":
		if isCppFile {
			currentContext = parseCppClassSpecifier(node, sourceCode, graph, file, currentContext)
		}

	case "namespace_definition":
		if isCppFile {
			currentContext = parseCppNamespaceDefinition(node, sourceCode, graph, file, currentContext)
		}

	case "template_declaration":
		if isCppFile {
			parseCppTemplateDeclaration(node, sourceCode, graph, file)
		}

	case "throw_statement":
		if isCppFile {
			parseCppThrowStatement(node, sourceCode, graph, file, currentContext)
		}

	case "try_statement":
		if isCppFile {
			parseCppTryStatement(node, sourceCode, graph, file, currentContext)
		}

	case "access_specifier":
		if isCppFile {
			recordAccessSpecifier(node, sourceCode, currentContext)
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
