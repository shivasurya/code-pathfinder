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
