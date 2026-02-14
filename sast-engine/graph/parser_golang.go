package graph

import (
	golangpkg "github.com/shivasurya/code-pathfinder/sast-engine/graph/golang"
	sitter "github.com/smacker/go-tree-sitter"
)

// setGoSourceLocation sets the SourceLocation on a Node from a tree-sitter node.
// Every Go node must call this to enable call graph construction,
// which uses StartByte/EndByte ranges to determine function containment.
func setGoSourceLocation(node *Node, tsNode *sitter.Node, file string) {
	node.SourceLocation = &SourceLocation{
		File:      file,
		StartByte: tsNode.StartByte(),
		EndByte:   tsNode.EndByte(),
	}
}

// parseGoFunctionDeclaration parses a Go function_declaration into a CodeGraph node.
// Returns the node so buildGraphFromAST can set it as currentContext.
func parseGoFunctionDeclaration(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) *Node {
	info := golangpkg.ParseFunctionDeclaration(tsNode, sourceCode)

	nodeType := "function_declaration"
	if info.IsInit {
		nodeType = "init_function"
	}

	methodID := GenerateMethodID("function:"+info.Name, info.Params.Names, file, info.LineNumber)
	node := &Node{
		ID:                   methodID,
		Type:                 nodeType,
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.Params.Types,
		MethodArgumentsValue: info.Params.Names,
		Modifier:             info.Visibility,
		File:                 file,
		isGoSourceFile:       true,
	}
	setGoSourceLocation(node, tsNode, file)
	graph.AddNode(node)
	return node
}

// parseGoMethodDeclaration parses a Go method_declaration into a CodeGraph node.
// Returns the node so buildGraphFromAST can set it as currentContext.
func parseGoMethodDeclaration(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) *Node {
	info := golangpkg.ParseMethodDeclaration(tsNode, sourceCode)

	methodID := GenerateMethodID("method:"+info.ReceiverType+"."+info.Name, info.Params.Names, file, info.LineNumber)
	node := &Node{
		ID:                   methodID,
		Type:                 "method",
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.Params.Types,
		MethodArgumentsValue: info.Params.Names,
		Modifier:             info.Visibility,
		Interface:            []string{info.ReceiverType},
		File:                 file,
		isGoSourceFile:       true,
	}
	setGoSourceLocation(node, tsNode, file)
	graph.AddNode(node)
	return node
}

// parseGoTypeDeclaration parses a Go type_declaration into CodeGraph nodes.
// Handles grouped declarations (type ( A int; B string )) which produce multiple nodes.
// Does not return a node — types don't set currentContext.
func parseGoTypeDeclaration(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	types := golangpkg.ParseTypeDeclaration(tsNode, sourceCode)

	for _, info := range types {
		var nodeType string
		var iface []string

		switch info.Kind {
		case "struct":
			nodeType = "struct_definition"
			iface = info.Fields
		case "interface":
			nodeType = "interface"
			iface = info.Methods
		default:
			nodeType = "type_alias"
		}

		typeID := GenerateMethodID("class:"+info.Name, []string{}, file, info.LineNumber)
		node := &Node{
			ID:             typeID,
			Type:           nodeType,
			Name:           info.Name,
			LineNumber:     info.LineNumber,
			Modifier:       info.Visibility,
			Interface:      iface,
			File:           file,
			isGoSourceFile: true,
			SourceLocation: &SourceLocation{
				File:      file,
				StartByte: info.StartByte,
				EndByte:   info.EndByte,
			},
		}
		graph.AddNode(node)
	}
}

// parseGoVarDeclaration parses a Go var_declaration into CodeGraph nodes.
// Handles grouped declarations and multi-name vars. Does not return a node.
func parseGoVarDeclaration(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	vars := golangpkg.ParseVarDeclaration(tsNode, sourceCode)

	for _, info := range vars {
		varID := GenerateMethodID(info.Name, []string{}, file, info.LineNumber)
		node := &Node{
			ID:             varID,
			Type:           "module_variable",
			Name:           info.Name,
			LineNumber:     info.LineNumber,
			VariableValue:  info.Value,
			DataType:       info.TypeName,
			Modifier:       info.Visibility,
			File:           file,
			isGoSourceFile: true,
			SourceLocation: &SourceLocation{
				File:      file,
				StartByte: info.StartByte,
				EndByte:   info.EndByte,
			},
		}
		graph.AddNode(node)
	}
}

// parseGoShortVarDeclaration parses a Go short_var_declaration into CodeGraph nodes.
// Handles multi-variable assignments (x, y := foo()). Does not return a node.
func parseGoShortVarDeclaration(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	vars := golangpkg.ParseShortVarDeclaration(tsNode, sourceCode)

	for _, info := range vars {
		nodeType := "variable_assignment"
		if info.IsMulti {
			nodeType = "multi_var_assignment"
		}

		varID := GenerateMethodID(info.Name, []string{}, file, info.LineNumber)
		node := &Node{
			ID:             varID,
			Type:           nodeType,
			Name:           info.Name,
			LineNumber:     info.LineNumber,
			VariableValue:  info.Value,
			Modifier:       info.Visibility,
			File:           file,
			isGoSourceFile: true,
			SourceLocation: &SourceLocation{
				File:      file,
				StartByte: info.StartByte,
				EndByte:   info.EndByte,
			},
		}
		graph.AddNode(node)
	}
}

// parseGoConstDeclaration parses a Go const_declaration into CodeGraph nodes.
// Handles grouped const declarations with iota. Does not return a node.
func parseGoConstDeclaration(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	consts := golangpkg.ParseConstDeclaration(tsNode, sourceCode)

	for _, info := range consts {
		constID := GenerateMethodID(info.Name, []string{}, file, info.LineNumber)
		node := &Node{
			ID:             constID,
			Type:           "constant",
			Name:           info.Name,
			LineNumber:     info.LineNumber,
			VariableValue:  info.Value,
			Modifier:       info.Visibility,
			File:           file,
			isGoSourceFile: true,
			SourceLocation: &SourceLocation{
				File:      file,
				StartByte: info.StartByte,
				EndByte:   info.EndByte,
			},
		}
		graph.AddNode(node)
	}
}

// parseGoAssignment parses a Go assignment_statement into CodeGraph nodes.
// Handles multi-variable assignments (x, y = 1, 2). Does not return a node.
func parseGoAssignment(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	vars := golangpkg.ParseAssignment(tsNode, sourceCode)

	for _, info := range vars {
		nodeType := "variable_assignment"
		if info.IsMulti {
			nodeType = "multi_var_assignment"
		}

		varID := GenerateMethodID(info.Name, []string{}, file, info.LineNumber)
		node := &Node{
			ID:             varID,
			Type:           nodeType,
			Name:           info.Name,
			LineNumber:     info.LineNumber,
			VariableValue:  info.Value,
			Modifier:       info.Visibility,
			File:           file,
			isGoSourceFile: true,
			SourceLocation: &SourceLocation{
				File:      file,
				StartByte: info.StartByte,
				EndByte:   info.EndByte,
			},
		}
		graph.AddNode(node)
	}
}

// parseGoCallExpression parses a Go call_expression into a CodeGraph node.
// CRITICAL: Creates an edge from currentContext to callNode to establish parent-child relationship.
// Does not return a node — calls don't set currentContext.
func parseGoCallExpression(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) {
	info := golangpkg.ParseCallExpression(tsNode, sourceCode)
	if info == nil {
		return
	}

	// Determine node type based on whether it's a selector expression
	nodeType := "call"
	if info.IsSelector {
		nodeType = "method_expression"
	}

	// Generate unique ID for this call
	callID := GenerateMethodID(info.FunctionName, info.Arguments, file, info.LineNumber)

	node := &Node{
		ID:                   callID,
		Type:                 nodeType,
		Name:                 info.FunctionName,
		LineNumber:           info.LineNumber,
		MethodArgumentsValue: info.Arguments,
		IsExternal:           true,
		File:                 file,
		isGoSourceFile:       true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: info.StartByte,
			EndByte:   info.EndByte,
		},
	}

	// Store object name in Interface field for method calls
	if info.IsSelector && info.ObjectName != "" {
		node.Interface = []string{info.ObjectName}
	}

	graph.AddNode(node)

	// CRITICAL: Create edge from parent function/method to this call
	// This enables call graph construction in PR-08
	if currentContext != nil {
		graph.AddEdge(currentContext, node)
	}
}
