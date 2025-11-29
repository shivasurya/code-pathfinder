package extraction

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// ExtractStatements extracts all statements from a Python function body.
// It processes assignments, calls, and returns to build def-use chains.
// Returns a slice of Statement objects or an error if parsing fails.
func ExtractStatements(filePath string, sourceCode []byte, functionNode *sitter.Node) ([]*core.Statement, error) {
	if functionNode == nil {
		return nil, fmt.Errorf("function node is nil")
	}

	bodyNode := functionNode.ChildByFieldName("body")
	if bodyNode == nil {
		// Empty function or no body
		return []*core.Statement{}, nil
	}

	var statements []*core.Statement

	// Iterate through all children of the body
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		stmtNode := bodyNode.Child(i)
		if stmtNode == nil {
			continue
		}

		// Python wraps many statements in expression_statement nodes
		// We need to unwrap them to get to the actual statement
		actualNode := stmtNode
		if stmtNode.Type() == "expression_statement" {
			// Get the first child which is the actual expression
			firstChild := stmtNode.Child(0)
			if firstChild != nil {
				actualNode = firstChild
			}
		}

		var stmt *core.Statement

		switch actualNode.Type() {
		case "assignment":
			stmt = extractAssignment(actualNode, sourceCode)

		case "augmented_assignment":
			stmt = extractAugmentedAssignment(actualNode, sourceCode)

		case "call":
			// Standalone call without assignment
			stmt = extractCall(actualNode, sourceCode)

		case "return_statement":
			stmt = extractReturn(actualNode, sourceCode)

		// Skip control flow statements (requires path sensitivity)
		case "if_statement", "while_statement", "for_statement", "with_statement", "try_statement":
			continue

		default:
			// Skip unknown statement types
			continue
		}

		if stmt != nil {
			// Set line number from the statement node
			stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
			statements = append(statements, stmt)
		}
	}

	return statements, nil
}

// extractAssignment processes assignment statements like "x = expr".
// Returns a Statement with Defs for LHS and Uses for RHS identifiers.
func extractAssignment(node *sitter.Node, sourceCode []byte) *core.Statement {
	if node == nil {
		return nil
	}

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")

	if leftNode == nil || rightNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeAssignment,
		Uses: []string{},
	}

	// Extract all identifiers from LHS (handles tuple unpacking)
	leftType := leftNode.Type()

	switch leftType {
	case "identifier":
		// Simple assignment: x = expr
		name := string(leftNode.Content(sourceCode)) //nolint:unconvert
		if !isKeyword(name) {
			stmt.Def = name
		}

	case "pattern_list", "tuple_pattern":
		// Tuple unpacking: x, y = expr
		// Skip tuple unpacking (not supported - requires multiple defs)
		return nil

	case "attribute":
		// Attribute assignment: obj.attr = expr
		// We skip these as they don't define local variables
		return nil

	case "subscript":
		// Subscript assignment: arr[i] = expr
		// We skip these as they don't define local variables
		return nil

	default:
		// Unknown LHS type, skip conservatively
		return nil
	}

	// Store RHS expression in CallTarget
	stmt.CallTarget = string(rightNode.Content(sourceCode)) //nolint:unconvert

	// Extract all identifiers from RHS
	rightType := rightNode.Type()

	if rightType == "call" {
		// Assignment from call: x = foo()
		callStmt := extractCall(rightNode, sourceCode)
		if callStmt != nil {
			// Use call's uses
			stmt.Uses = callStmt.Uses
		}
	} else {
		// Assignment from expression: x = y + z
		stmt.Uses = extractIdentifiers(rightNode, sourceCode)
	}

	return stmt
}

// extractAugmentedAssignment processes augmented assignments like "x += expr".
// Returns a Statement with both Def and Use for the target variable.
func extractAugmentedAssignment(node *sitter.Node, sourceCode []byte) *core.Statement {
	if node == nil {
		return nil
	}

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")

	if leftNode == nil || rightNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeAssignment,
		Uses: []string{},
	}

	// For augmented assignment, LHS is both defined and used
	leftType := leftNode.Type()

	switch leftType {
	case "identifier":
		name := string(leftNode.Content(sourceCode)) //nolint:unconvert
		if !isKeyword(name) {
			stmt.Def = name
			stmt.Uses = append(stmt.Uses, name)
		}

	case "attribute", "subscript":
		// obj.attr += expr or arr[i] += expr
		// Extract identifiers from the expression
		leftIds := extractIdentifiers(leftNode, sourceCode)
		stmt.Uses = append(stmt.Uses, leftIds...)
		// No def for attributes/subscripts
		if len(stmt.Uses) == 0 {
			return nil
		}

	default:
		return nil
	}

	// Extract identifiers from RHS
	rightIds := extractIdentifiers(rightNode, sourceCode)
	stmt.Uses = append(stmt.Uses, rightIds...)

	return stmt
}

// extractCall processes function/method calls.
// Returns a Statement with Uses for call arguments and CallTarget.
func extractCall(callNode *sitter.Node, sourceCode []byte) *core.Statement {
	if callNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeCall,
		Uses: []string{},
	}

	// Extract call target (function/method name)
	functionNode := callNode.ChildByFieldName("function")
	if functionNode != nil {
		stmt.CallTarget = extractCallTarget(functionNode, sourceCode)

		// For nested calls, add the function name to Uses (conservative approach)
		targetIds := extractIdentifiers(functionNode, sourceCode)
		stmt.Uses = append(stmt.Uses, targetIds...)
	}

	// Extract arguments
	argumentsNode := callNode.ChildByFieldName("arguments")
	if argumentsNode != nil {
		// CallArgs contains literal argument values
		stmt.CallArgs = extractCallArgs(argumentsNode, sourceCode)

		// Uses contains all identifiers from arguments (recursive extraction)
		argIds := extractIdentifiersFromArgs(argumentsNode, sourceCode)
		stmt.Uses = append(stmt.Uses, argIds...)
	}

	return stmt
}

// extractCallTarget extracts the function/method name from a call expression.
// Handles: foo, obj.method, obj.method1.method2.
func extractCallTarget(functionNode *sitter.Node, sourceCode []byte) string {
	if functionNode == nil {
		return ""
	}

	switch functionNode.Type() {
	case "identifier":
		// Simple call: foo()
		return string(functionNode.Content(sourceCode)) //nolint:unconvert

	case "attribute":
		// Method call: obj.method() or obj.method1.method2()
		// Extract just the method name (rightmost identifier)
		attrNode := functionNode.ChildByFieldName("attribute")
		if attrNode != nil {
			return string(attrNode.Content(sourceCode)) //nolint:unconvert
		}
		return string(functionNode.Content(sourceCode)) //nolint:unconvert

	default:
		// Complex expression, return full content
		return string(functionNode.Content(sourceCode)) //nolint:unconvert
	}
}

// extractIdentifiersFromArgs extracts all identifiers from call arguments recursively.
// Used for the Uses field to track all variables referenced.
func extractIdentifiersFromArgs(argumentsNode *sitter.Node, sourceCode []byte) []string {
	if argumentsNode == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var identifiers []string

	// Iterate through all argument children
	for i := 0; i < int(argumentsNode.ChildCount()); i++ {
		argNode := argumentsNode.Child(i)
		if argNode == nil {
			continue
		}

		// Skip punctuation
		if argNode.Type() == "," || argNode.Type() == "(" || argNode.Type() == ")" {
			continue
		}

		// Handle keyword arguments: arg=value
		if argNode.Type() == "keyword_argument" {
			valueNode := argNode.ChildByFieldName("value")
			if valueNode != nil {
				ids := extractIdentifiers(valueNode, sourceCode)
				for _, id := range ids {
					if !seen[id] {
						seen[id] = true
						identifiers = append(identifiers, id)
					}
				}
			}
			continue
		}

		// Extract identifiers from the argument expression
		ids := extractIdentifiers(argNode, sourceCode)
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				identifiers = append(identifiers, id)
			}
		}
	}

	return identifiers
}

// extractCallArgs extracts all values used in call arguments (identifiers and literals).
// Returns a deduplicated list of argument values.
func extractCallArgs(argumentsNode *sitter.Node, sourceCode []byte) []string {
	if argumentsNode == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var args []string

	// Iterate through all argument children
	for i := 0; i < int(argumentsNode.ChildCount()); i++ {
		argNode := argumentsNode.Child(i)
		if argNode == nil {
			continue
		}

		// Skip punctuation (, and )
		if argNode.Type() == "," || argNode.Type() == "(" || argNode.Type() == ")" {
			continue
		}

		// Handle keyword arguments: arg=value
		if argNode.Type() == "keyword_argument" {
			valueNode := argNode.ChildByFieldName("value")
			if valueNode != nil {
				// Include the full value (identifier or literal)
				value := string(valueNode.Content(sourceCode)) //nolint:unconvert
				if !seen[value] {
					seen[value] = true
					args = append(args, value)
				}
			}
			continue
		}

		// Regular positional argument (identifier or literal)
		value := string(argNode.Content(sourceCode)) //nolint:unconvert
		if !seen[value] {
			seen[value] = true
			args = append(args, value)
		}
	}

	return args
}

// extractReturn processes return statements.
// Returns a Statement with Uses for returned identifiers.
func extractReturn(node *sitter.Node, sourceCode []byte) *core.Statement {
	if node == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeReturn,
		Uses: []string{},
	}

	// Check if there's a return value
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		// Skip the "return" keyword itself
		if child.Type() == "return" {
			continue
		}

		// Store the return expression in CallTarget
		stmt.CallTarget = string(child.Content(sourceCode)) //nolint:unconvert

		// Extract identifiers from the return expression
		ids := extractIdentifiers(child, sourceCode)
		stmt.Uses = append(stmt.Uses, ids...)
	}

	return stmt
}

// extractIdentifiers recursively extracts all identifiers from an AST node.
// Returns a deduplicated list of identifier names (filters out keywords).
func extractIdentifiers(node *sitter.Node, sourceCode []byte) []string {
	if node == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var identifiers []string

	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Type() == "identifier" {
			name := string(n.Content(sourceCode)) //nolint:unconvert
			if !isKeyword(name) && !seen[name] {
				seen[name] = true
				identifiers = append(identifiers, name)
			}
			return
		}

		// Recursively visit children
		for i := 0; i < int(n.ChildCount()); i++ {
			visit(n.Child(i))
		}
	}

	visit(node)
	return identifiers
}

// isKeyword checks if a name is a Python keyword.
// Keywords should not be treated as variables in def-use chains.
func isKeyword(name string) bool {
	keywords := map[string]bool{
		"False": true, "None": true, "True": true,
		"and": true, "as": true, "assert": true, "async": true, "await": true,
		"break": true, "class": true, "continue": true, "def": true, "del": true,
		"elif": true, "else": true, "except": true, "finally": true, "for": true,
		"from": true, "global": true, "if": true, "import": true, "in": true,
		"is": true, "lambda": true, "nonlocal": true, "not": true, "or": true,
		"pass": true, "raise": true, "return": true, "try": true, "while": true,
		"with": true, "yield": true,
		"self": true, // Filter out self references
	}
	return keywords[name]
}

// ParsePythonFile parses a Python source file using tree-sitter.
// Returns the parsed tree or an error.
func ParsePythonFile(sourceCode []byte) (*sitter.Tree, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python code: %w", err)
	}

	if tree == nil {
		return nil, fmt.Errorf("tree-sitter returned nil tree")
	}

	return tree, nil
}
