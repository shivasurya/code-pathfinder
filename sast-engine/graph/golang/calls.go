package golang

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// CallInfo represents a parsed Go call expression (function call, method call, or package call).
type CallInfo struct {
	FunctionName string   // "Println", "Method", "foo"
	ObjectName   string   // "fmt", "obj", "" for simple calls
	Arguments    []string // argument source code strings
	IsSelector   bool     // true for obj.Method() or pkg.Func(), false for foo()
	LineNumber   uint32
	StartByte    uint32
	EndByte      uint32
}

// ParseCallExpression parses a Go call_expression node into a CallInfo.
// Handles simple calls (foo()), method calls (obj.Method()), package calls (pkg.Func()).
func ParseCallExpression(node *sitter.Node, sourceCode []byte) *CallInfo {
	if node == nil || node.Type() != "call_expression" {
		return nil
	}

	info := &CallInfo{
		LineNumber: node.StartPoint().Row + 1,
		StartByte:  node.StartByte(),
		EndByte:    node.EndByte(),
	}

	// Extract function name (field: "function")
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}

	switch funcNode.Type() {
	case "identifier":
		// Simple call: foo()
		info.FunctionName = funcNode.Content(sourceCode)
		info.IsSelector = false

	case "selector_expression":
		// Method or package call: obj.Method() or pkg.Func()
		obj, field := ParseSelectorExpression(funcNode, sourceCode)
		info.ObjectName = obj
		info.FunctionName = field
		info.IsSelector = true

	case "func_literal":
		// IIFE (immediately invoked function expression)
		// This will be handled separately as a func_literal node
		info.FunctionName = ""
		info.IsSelector = false

	default:
		// Unknown function type, extract content as fallback
		info.FunctionName = funcNode.Content(sourceCode)
		info.IsSelector = false
	}

	// Extract arguments (field: "arguments")
	argsNode := node.ChildByFieldName("arguments")
	if argsNode != nil && argsNode.Type() == "argument_list" {
		info.Arguments = extractArguments(argsNode, sourceCode)
	}

	return info
}

// ParseSelectorExpression extracts the object and field from a selector_expression.
// Returns (object, field) as strings.
// Example: "fmt.Println" returns ("fmt", "Println").
func ParseSelectorExpression(node *sitter.Node, sourceCode []byte) (object string, field string) {
	if node == nil || node.Type() != "selector_expression" {
		return "", ""
	}

	// Field: "operand"
	operandNode := node.ChildByFieldName("operand")
	if operandNode != nil {
		object = operandNode.Content(sourceCode)
	}

	// Field: "field"
	fieldNode := node.ChildByFieldName("field")
	if fieldNode != nil {
		field = fieldNode.Content(sourceCode)
	}

	return object, field
}

// extractArguments iterates the named children of an argument_list node
// and returns their source code as strings.
func extractArguments(argsNode *sitter.Node, sourceCode []byte) []string {
	// Initialize with empty slice (not nil) for consistent behavior
	arguments := []string{}

	for i := 0; i < int(argsNode.NamedChildCount()); i++ {
		child := argsNode.NamedChild(i)
		if child != nil {
			arguments = append(arguments, child.Content(sourceCode))
		}
	}

	return arguments
}
