package golang

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ClosureInfo represents a parsed Go func_literal (anonymous function/closure).
type ClosureInfo struct {
	Params     GoParams
	ReturnType string
	LineNumber uint32
	StartByte  uint32
	EndByte    uint32
}

// ParseFuncLiteral parses a Go func_literal node into a ClosureInfo.
// Example: func(x int) int { return x + 1 }.
func ParseFuncLiteral(node *sitter.Node, sourceCode []byte) *ClosureInfo {
	if node == nil || node.Type() != "func_literal" {
		return nil
	}

	info := &ClosureInfo{
		LineNumber: uint32(node.StartPoint().Row) + 1,
		StartByte:  node.StartByte(),
		EndByte:    node.EndByte(),
	}

	// Extract parameters from field "parameters"
	paramList := node.ChildByFieldName("parameters")
	if paramList != nil {
		info.Params = ExtractParameters(paramList, sourceCode)
	}

	// Extract return type from field "result"
	resultNode := node.ChildByFieldName("result")
	if resultNode != nil {
		info.ReturnType = resultNode.Content(sourceCode)
	} else {
		info.ReturnType = ""
	}

	return info
}

// ParseDeferStatement parses a Go defer_statement node into a CallInfo.
// Example: defer f.Close()
// Returns the CallInfo for the deferred call, which the dispatcher will mark as defer_call.
func ParseDeferStatement(node *sitter.Node, sourceCode []byte) *CallInfo {
	if node == nil || node.Type() != "defer_statement" {
		return nil
	}

	// Find the call_expression child
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "call_expression" {
			// Parse as regular call expression
			return ParseCallExpression(child, sourceCode)
		}
	}

	return nil
}

// ParseGoStatement parses a Go go_statement node into a CallInfo.
// Example: go handler(conn)
// Returns the CallInfo for the goroutine call, which the dispatcher will mark as go_call.
func ParseGoStatement(node *sitter.Node, sourceCode []byte) *CallInfo {
	if node == nil || node.Type() != "go_statement" {
		return nil
	}

	// Find the call_expression child
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "call_expression" {
			// Parse as regular call expression
			return ParseCallExpression(child, sourceCode)
		}
	}

	return nil
}
