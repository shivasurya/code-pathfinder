package graph

import (
	"strings"
	"sync"

	golangpkg "github.com/shivasurya/code-pathfinder/sast-engine/graph/golang"
	"github.com/shivasurya/code-pathfinder/sast-engine/model"
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
		Language:             "go",
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
		Language:             "go",
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
			Language:       "go",
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
			Language:       "go",
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
			Language:       "go",
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
			Language:       "go",
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
			Language:       "go",
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
		Language:             "go",
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

// anonCounters maps parent context names to their anonymous function counter.
// Used to generate unique $anon_N names scoped to each parent function.
var anonCounters = make(map[string]int)
var anonCountersMutex sync.Mutex

// generateAnonName generates a unique anonymous function name scoped to the current context.
// Returns "$anon_1", "$anon_2", etc. incrementing for each parent context.
// Thread-safe for concurrent parsing.
func generateAnonName(currentContext *Node) string {
	anonCountersMutex.Lock()
	defer anonCountersMutex.Unlock()

	if currentContext == nil {
		// Top-level anonymous function (unlikely in Go)
		anonCounters["$global"]++
		return "$anon_" + string(rune(anonCounters["$global"]+'0'))
	}

	// Increment counter for this parent context
	parentName := currentContext.Name
	anonCounters[parentName]++

	// Format as "$anon_N"
	return "$anon_" + string(rune(anonCounters[parentName]+'0'))
}

// parseGoFuncLiteral parses a Go func_literal into a CodeGraph node.
// Returns the node so buildGraphFromAST can set it as currentContext for traversing the closure body.
func parseGoFuncLiteral(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) *Node {
	info := golangpkg.ParseFuncLiteral(tsNode, sourceCode)
	if info == nil {
		return nil
	}

	// Generate anonymous function name scoped to parent context
	anonName := generateAnonName(currentContext)

	methodID := GenerateMethodID("function:"+anonName, info.Params.Names, file, info.LineNumber)
	node := &Node{
		ID:                   methodID,
		Type:                 "func_literal",
		Name:                 anonName,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.Params.Types,
		MethodArgumentsValue: info.Params.Names,
		Modifier:             "private",
		File:                 file,
		isGoSourceFile:       true,
		Language:             "go",
	}
	setGoSourceLocation(node, tsNode, file)
	graph.AddNode(node)

	// Return node to become currentContext for closure body traversal
	return node
}

// parseGoDeferStatement parses a Go defer_statement into a CodeGraph node.
// Does not return a node — defer statements don't set currentContext.
func parseGoDeferStatement(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) {
	info := golangpkg.ParseDeferStatement(tsNode, sourceCode)
	if info == nil {
		return
	}

	// Determine node type based on whether it's a selector expression
	nodeType := "defer_call"

	// Generate unique ID for this defer call
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
		Language:             "go",
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

	// Create edge from parent function to this defer call
	if currentContext != nil {
		graph.AddEdge(currentContext, node)
	}
}

// parseGoGoStatement parses a Go go_statement into a CodeGraph node.
// Does not return a node — go statements don't set currentContext.
func parseGoGoStatement(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) {
	info := golangpkg.ParseGoStatement(tsNode, sourceCode)
	if info == nil {
		return
	}

	// Determine node type
	nodeType := "go_call"

	// Generate unique ID for this goroutine call
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
		Language:             "go",
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

	// Create edge from parent function to this goroutine call
	if currentContext != nil {
		graph.AddEdge(currentContext, node)
	}
}

// parseGoReturnStatement parses a Go return_statement into a CodeGraph node.
// Does not return a node — return statements don't set currentContext.
// Uses GenerateSha256 for ID (NOT GenerateMethodID), following Python pattern.
func parseGoReturnStatement(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	info := golangpkg.ParseReturnStatement(tsNode, sourceCode)
	if info == nil {
		return
	}

	// Generate unique ID using SHA256 (NOT GenerateMethodID)
	uniqueID := "ReturnStmt:" + file + ":" + string(rune(info.LineNumber+'0'))
	stmtID := GenerateSha256(uniqueID)

	// Create a simple ReturnStmt model object
	returnStmt := &model.ReturnStmt{
		Stmt: model.Stmt{
			NodeString: "return " + joinStrings(info.Values),
		},
	}

	node := &Node{
		ID:             stmtID,
		Type:           "ReturnStmt",
		Name:           "return",
		LineNumber:     info.LineNumber,
		ReturnStmt:     returnStmt,
		File:           file,
		isGoSourceFile: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: info.StartByte,
			EndByte:   info.EndByte,
		},
	}

	graph.AddNode(node)
	// NOTE: Do NOT create edges for statement nodes (Python pattern)
}

// joinStrings joins a slice of strings with commas (helper for statement parsing).
func joinStrings(strs []string) string {
	var result strings.Builder
	for i, s := range strs {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(s)
	}
	return result.String()
}

// parseGoForStatement parses a Go for_statement into a CodeGraph node.
// Does not return a node — for statements don't set currentContext.
// Uses GenerateSha256 for ID (NOT GenerateMethodID), following Python pattern.
func parseGoForStatement(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	info := golangpkg.ParseForStatement(tsNode, sourceCode)
	if info == nil {
		return
	}

	// Generate unique ID using SHA256 (NOT GenerateMethodID)
	uniqueID := "ForStmt:" + file + ":" + string(rune(info.LineNumber+'0'))
	stmtID := GenerateSha256(uniqueID)

	// Create node string representation
	var nodeStr string
	if info.IsRange {
		nodeStr = "for " + info.Left + " := range " + info.Right
	} else {
		nodeStr = "for " + info.Init + "; " + info.Condition + "; " + info.Update
	}

	// Create a simple ForStmt model object
	forStmt := &model.ForStmt{
		ConditionalStmt: model.ConditionalStmt{
			Stmt: model.Stmt{
				NodeString: nodeStr,
			},
		},
	}

	node := &Node{
		ID:             stmtID,
		Type:           "ForStmt",
		Name:           "for",
		LineNumber:     info.LineNumber,
		ForStmt:        forStmt,
		File:           file,
		isGoSourceFile: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: info.StartByte,
			EndByte:   info.EndByte,
		},
	}

	graph.AddNode(node)
	// NOTE: Do NOT create edges for statement nodes (Python pattern)
}

// parseGoIfStatement parses a Go if_statement into a CodeGraph node.
// Does not return a node — if statements don't set currentContext.
// Uses GenerateSha256 for ID (NOT GenerateMethodID), following Python pattern.
func parseGoIfStatement(tsNode *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	info := golangpkg.ParseIfStatement(tsNode, sourceCode)
	if info == nil {
		return
	}

	// Generate unique ID using SHA256 (NOT GenerateMethodID)
	uniqueID := "IfStmt:" + file + ":" + string(rune(info.LineNumber+'0'))
	stmtID := GenerateSha256(uniqueID)

	// Create a simple IfStmt model object
	ifStmt := &model.IfStmt{
		ConditionalStmt: model.ConditionalStmt{
			Stmt: model.Stmt{
				NodeString: "if " + info.Condition,
			},
		},
	}

	node := &Node{
		ID:             stmtID,
		Type:           "IfStmt",
		Name:           "if",
		LineNumber:     info.LineNumber,
		IfStmt:         ifStmt,
		File:           file,
		isGoSourceFile: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: info.StartByte,
			EndByte:   info.EndByte,
		},
	}

	graph.AddNode(node)
	// NOTE: Do NOT create edges for statement nodes (Python pattern)
}
