package resolution

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// ExtractCallSites extracts all function/method call sites from a Python file.
// It traverses the AST to find call expressions and builds CallSite objects
// with caller context, callee information, and arguments.
//
// Algorithm:
//  1. Parse source code with tree-sitter Python parser
//  2. Traverse AST to find call expressions
//  3. For each call, extract:
//     - Caller function/method (containing context)
//     - Callee name (function/method being called)
//     - Arguments (positional and keyword)
//     - Source location (file, line, column)
//  4. Build CallSite objects for each call
//
// Parameters:
//   - filePath: absolute path to the Python file being analyzed
//   - sourceCode: contents of the Python file as byte array
//   - importMap: import mappings for resolving qualified names
//
// Returns:
//   - []CallSite: list of all call sites found in the file
//   - error: if parsing fails or source is invalid
//
// Example:
//
//	Source code:
//	  def process_data():
//	      result = sanitize(data)
//	      db.query(result)
//
//	Extracts CallSites:
//	  [
//	    {Caller: "process_data", Callee: "sanitize", Args: ["data"]},
//	    {Caller: "process_data", Callee: "db.query", Args: ["result"]}
//	  ]
func ExtractCallSites(filePath string, sourceCode []byte, importMap *core.ImportMap) ([]*core.CallSite, error) {
	var callSites []*core.CallSite

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	// Traverse AST to find call expressions
	// We need to track the current function/method context as we traverse
	traverseForCalls(tree.RootNode(), sourceCode, filePath, importMap, "", &callSites)

	return callSites, nil
}

// traverseForCalls recursively traverses the AST to find call expressions.
// It maintains the current function/method context (caller) as it traverses.
//
// Parameters:
//   - node: current AST node being processed
//   - sourceCode: source code bytes for extracting node content
//   - filePath: file path for source location
//   - importMap: import mappings for resolving names
//   - currentContext: name of the current function/method containing this code
//   - callSites: accumulator for discovered call sites
func traverseForCalls(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	importMap *core.ImportMap,
	currentContext string,
	callSites *[]*core.CallSite,
) {
	if node == nil {
		return
	}

	nodeType := node.Type()

	// Update context when entering a function or method definition
	newContext := currentContext
	if nodeType == "function_definition" {
		// Extract function name
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			newContext = nameNode.Content(sourceCode)
		}
	}

	// Process call expressions
	if nodeType == "call" {
		callSite := processCallExpression(node, sourceCode, filePath, importMap, currentContext)
		if callSite != nil {
			*callSites = append(*callSites, callSite)
		}
	}

	// Recursively process children with updated context
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		traverseForCalls(child, sourceCode, filePath, importMap, newContext, callSites)
	}
}

// processCallExpression processes a call expression node and extracts CallSite information.
//
// Call expression structure in tree-sitter:
//   - function: the callable being invoked (identifier, attribute, etc.)
//   - arguments: argument_list containing positional and keyword arguments
//
// Examples:
//   - foo() → function="foo", arguments=[]
//   - obj.method(x) → function="obj.method", arguments=["x"]
//   - func(a, b=2) → function="func", arguments=["a", "b=2"]
//
// Parameters:
//   - node: call expression AST node
//   - sourceCode: source code bytes
//   - filePath: file path for location
//   - importMap: import mappings for resolving names
//   - caller: name of the function containing this call
//
// Returns:
//   - CallSite: extracted call site information, or nil if extraction fails
func processCallExpression(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	_ *core.ImportMap, // Will be used in Pass 3 for call resolution
	_ string,          // caller - Will be used in Pass 3 for call resolution
) *core.CallSite {
	// Get the function being called
	functionNode := node.ChildByFieldName("function")
	if functionNode == nil {
		return nil
	}

	// Extract callee name (handles identifiers, attributes, etc.)
	callee := extractCalleeName(functionNode, sourceCode)
	if callee == "" {
		return nil
	}

	// Get arguments
	argumentsNode := node.ChildByFieldName("arguments")
	var args []*core.Argument
	if argumentsNode != nil {
		args = extractArguments(argumentsNode, sourceCode)
	}

	// Create source location
	location := &core.Location{
		File:   filePath,
		Line:   int(node.StartPoint().Row) + 1, // tree-sitter is 0-indexed
		Column: int(node.StartPoint().Column) + 1,
	}

	return &core.CallSite{
		Target:    callee,
		Location:  *location,
		Arguments: convertArgumentsToSlice(args),
		Resolved:  false,
		TargetFQN: "", // Will be set during resolution phase
	}
}

// extractCalleeName extracts the name of the callable from a function node.
// Handles different node types:
//   - identifier: simple function name (e.g., "foo")
//   - attribute: method call (e.g., "obj.method", "obj.attr.method")
//
// Parameters:
//   - node: function node from call expression
//   - sourceCode: source code bytes
//
// Returns:
//   - Fully qualified callee name
func extractCalleeName(node *sitter.Node, sourceCode []byte) string {
	nodeType := node.Type()

	switch nodeType {
	case "identifier":
		// Simple function call: foo()
		return node.Content(sourceCode)

	case "attribute":
		// Method call: obj.method() or obj.attr.method()
		// The attribute node has 'object' and 'attribute' fields
		objectNode := node.ChildByFieldName("object")
		attributeNode := node.ChildByFieldName("attribute")

		if objectNode != nil && attributeNode != nil {
			// Recursively extract object name (could be nested)
			objectName := extractCalleeName(objectNode, sourceCode)
			attributeName := attributeNode.Content(sourceCode)

			if objectName != "" && attributeName != "" {
				return objectName + "." + attributeName
			}
		}

	case "call":
		// Chained call: Class()() or obj.method()()
		// Extract the function being called, not the entire call expression with arguments
		// This prevents FQN pollution like "module.Class(args).method" instead of "module.Class.method"
		functionNode := node.ChildByFieldName("function")
		if functionNode != nil {
			// Recursively extract the function name (could be nested: obj.Class())
			return extractCalleeName(functionNode, sourceCode)
		}
		// Fallback if no function field found
		return node.Content(sourceCode)
	}

	// For other node types, return the full content
	return node.Content(sourceCode)
}

// extractArguments extracts all arguments from an argument_list node.
// Handles both positional and keyword arguments.
//
// Note: The Argument struct doesn't distinguish between positional and keyword arguments.
// For keyword arguments (name=value), we store them as "name=value" in the Value field.
//
// Examples:
//   - (a, b, c) → [Arg{Value: "a", Position: 0}, Arg{Value: "b", Position: 1}, ...]
//   - (x, y=2, z=foo) → [Arg{Value: "x", Position: 0}, Arg{Value: "y=2", Position: 1}, ...]
//
// Parameters:
//   - argumentsNode: argument_list AST node
//   - sourceCode: source code bytes
//
// Returns:
//   - List of Argument objects
func extractArguments(argumentsNode *sitter.Node, sourceCode []byte) []*core.Argument {
	var args []*core.Argument

	// Iterate through all children of argument_list
	for i := 0; i < int(argumentsNode.NamedChildCount()); i++ {
		child := argumentsNode.NamedChild(i)
		if child == nil {
			continue
		}

		// For all argument types, just extract the full content
		// This handles both positional and keyword arguments
		arg := &core.Argument{
			Value:      child.Content(sourceCode),
			IsVariable: child.Type() == "identifier",
			Position:   i,
		}
		args = append(args, arg)
	}

	return args
}

// convertArgumentsToSlice converts a slice of Argument pointers to a slice of Argument values.
func convertArgumentsToSlice(args []*core.Argument) []core.Argument {
	result := make([]core.Argument, len(args))
	for i, arg := range args {
		if arg != nil {
			result[i] = *arg
		}
	}
	return result
}
