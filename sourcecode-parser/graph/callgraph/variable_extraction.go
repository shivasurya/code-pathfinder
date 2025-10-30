package callgraph

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ExtractVariableAssignments extracts variable assignments from a Python file
// and populates the type inference engine with inferred types.
//
// Algorithm:
//  1. Parse source code with tree-sitter Python parser
//  2. Traverse AST to find assignment statements
//  3. For each assignment:
//     - Extract variable name
//     - Infer type from RHS (literal, function call, or method call)
//     - Create VariableBinding with inferred type
//     - Add binding to function scope
//
// Parameters:
//   - filePath: absolute path to the Python file
//   - sourceCode: contents of the file as byte array
//   - typeEngine: type inference engine to populate
//   - registry: module registry for resolving module paths
//   - builtinRegistry: builtin types registry for literal inference
//
// Returns:
//   - error: if parsing fails
func ExtractVariableAssignments(
	filePath string,
	sourceCode []byte,
	typeEngine *TypeInferenceEngine,
	registry *ModuleRegistry,
	builtinRegistry *BuiltinRegistry,
) error {
	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return err
	}
	defer tree.Close()

	// Get module FQN for this file
	modulePath, exists := registry.FileToModule[filePath]
	if !exists {
		// If file not in registry, skip (e.g., external files)
		return nil
	}

	// Traverse AST to find assignments
	traverseForAssignments(
		tree.RootNode(),
		sourceCode,
		filePath,
		modulePath,
		"",
		typeEngine,
		builtinRegistry,
	)

	return nil
}

// traverseForAssignments recursively traverses AST to find assignment statements.
//
// Parameters:
//   - node: current AST node
//   - sourceCode: source code bytes
//   - filePath: file path for locations
//   - modulePath: module FQN
//   - currentFunction: current function FQN (empty if module-level)
//   - typeEngine: type inference engine
//   - builtinRegistry: builtin types registry
func traverseForAssignments(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	typeEngine *TypeInferenceEngine,
	builtinRegistry *BuiltinRegistry,
) {
	if node == nil {
		return
	}

	nodeType := node.Type()

	// Update context when entering function/method
	if nodeType == "function_definition" {
		functionName := extractFunctionName(node, sourceCode)
		if functionName != "" {
			if currentFunction == "" {
				// Module-level function
				currentFunction = modulePath + "." + functionName
			} else {
				// Nested function
				currentFunction = currentFunction + "." + functionName
			}

			// Ensure scope exists for this function
			if typeEngine.GetScope(currentFunction) == nil {
				typeEngine.AddScope(NewFunctionScope(currentFunction))
			}
		}
	}

	// Process assignment statements
	if nodeType == "assignment" {
		processAssignment(
			node,
			sourceCode,
			filePath,
			currentFunction,
			typeEngine,
			builtinRegistry,
		)
	}

	// Recurse to children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		traverseForAssignments(
			child,
			sourceCode,
			filePath,
			modulePath,
			currentFunction,
			typeEngine,
			builtinRegistry,
		)
	}
}

// processAssignment extracts type information from an assignment statement.
//
// Handles:
//   - var = "literal" (literal inference)
//   - var = func() (return type inference - Task 2 Phase 1)
//   - var = obj.method() (method return type - Task 2 Phase 1)
//
// Parameters:
//   - node: assignment AST node
//   - sourceCode: source code bytes
//   - filePath: file path for location
//   - currentFunction: current function FQN
//   - typeEngine: type inference engine
//   - builtinRegistry: builtin types registry
func processAssignment(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	currentFunction string,
	typeEngine *TypeInferenceEngine,
	builtinRegistry *BuiltinRegistry,
) {
	// Assignment node structure:
	//   assignment
	//     left: identifier or pattern
	//     "="
	//     right: expression

	var leftNode *sitter.Node
	var rightNode *sitter.Node

	// Find left and right sides of assignment
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "pattern_list" {
			leftNode = child
		} else if child.Type() != "=" && rightNode == nil {
			// Right side is the first non-"=" expression node
			if child.Type() != "identifier" && child.Type() != "pattern_list" {
				rightNode = child
			}
		}
	}

	if leftNode == nil || rightNode == nil {
		return
	}

	// Extract variable name
	varName := leftNode.Content(sourceCode)
	varName = strings.TrimSpace(varName)

	// Skip pattern assignments (tuple unpacking) for now
	if leftNode.Type() == "pattern_list" {
		return
	}

	// Infer type from right side
	typeInfo := inferTypeFromExpression(rightNode, sourceCode, builtinRegistry)
	if typeInfo == nil {
		return
	}

	// Create variable binding
	binding := &VariableBinding{
		VarName: varName,
		Type:    typeInfo,
		Location: Location{
			File:   filePath,
			Line:   int(leftNode.StartPoint().Row) + 1,
			Column: int(leftNode.StartPoint().Column) + 1,
		},
	}

	// If RHS is a call, track the function that assigned this
	if rightNode.Type() == "call" {
		calleeName := extractCalleeName(rightNode, sourceCode)
		if calleeName != "" {
			binding.AssignedFrom = calleeName
		}
	}

	// Add to function scope or create module-level scope
	functionFQN := currentFunction
	if functionFQN == "" {
		// Module-level variable - use a special scope
		// For simplicity, we'll skip module-level for now
		return
	}

	scope := typeEngine.GetScope(functionFQN)
	if scope == nil {
		scope = NewFunctionScope(functionFQN)
		typeEngine.AddScope(scope)
	}

	scope.Variables[varName] = binding
}

// inferTypeFromExpression infers the type of an expression.
//
// Currently handles:
//   - Literals (strings, numbers, lists, dicts, etc.)
//   - Future: function calls, method calls (Task 2 Phase 1)
//
// Parameters:
//   - node: expression AST node
//   - sourceCode: source code bytes
//   - builtinRegistry: builtin types registry
//
// Returns:
//   - TypeInfo if type can be inferred, nil otherwise
func inferTypeFromExpression(
	node *sitter.Node,
	sourceCode []byte,
	builtinRegistry *BuiltinRegistry,
) *TypeInfo {
	if node == nil {
		return nil
	}

	nodeType := node.Type()

	// Handle literals
	switch nodeType {
	case "string", "concatenated_string":
		return &TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "integer":
		return &TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "float":
		return &TypeInfo{
			TypeFQN:    "builtins.float",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "true", "false":
		return &TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "none":
		return &TypeInfo{
			TypeFQN:    "builtins.NoneType",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "list":
		return &TypeInfo{
			TypeFQN:    "builtins.list",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "dictionary":
		return &TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "set":
		return &TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "tuple":
		return &TypeInfo{
			TypeFQN:    "builtins.tuple",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// For non-literals, try to infer from builtin registry
	// This handles edge cases where tree-sitter node types don't match exactly
	literal := node.Content(sourceCode)
	return builtinRegistry.InferLiteralType(literal)
}

// extractFunctionName extracts the function name from a function_definition node.
func extractFunctionName(node *sitter.Node, sourceCode []byte) string {
	if node.Type() != "function_definition" {
		return ""
	}

	// Find the identifier node (function name)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return child.Content(sourceCode)
		}
	}

	return ""
}

