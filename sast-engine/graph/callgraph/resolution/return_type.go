package resolution

import (
	"context"
	"maps"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ReturnStatement represents a return statement in a function.
type ReturnStatement struct {
	FunctionFQN string
	ReturnType  *core.TypeInfo
	Location    Location
}

// ExtractReturnTypes analyzes return statements in all functions in a file.
// Returns:
//   - []*ReturnStatement: return statements with inferred types
//   - map[string]bool: set of function FQNs that have at least one `return <expr>` statement
//     (used to distinguish void functions from functions with uninferrable returns)
func ExtractReturnTypes(
	filePath string,
	sourceCode []byte,
	modulePath string,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) ([]*ReturnStatement, map[string]bool, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, nil, err
	}
	defer tree.Close()

	var returns []*ReturnStatement
	functionsWithReturnValues := make(map[string]bool)
	traverseForReturns(tree.RootNode(), sourceCode, filePath, modulePath, "", &returns, functionsWithReturnValues, builtinRegistry, importMap)

	return returns, functionsWithReturnValues, nil
}

func traverseForReturns(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	returns *[]*ReturnStatement,
	functionsWithReturnValues map[string]bool,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) {
	if node == nil {
		return
	}

	// Track when we enter a class or function
	newFunction := currentFunction

	// Track class definitions to build class-qualified FQNs for methods
	if node.Type() == "class_definition" {
		className := extractClassNameFromNode(node, sourceCode)
		if className != "" {
			if currentFunction == "" {
				// Top-level class
				newFunction = modulePath + "." + className
			} else {
				// Nested class
				newFunction = currentFunction + "." + className
			}
		}
	}

	// Track function definitions (both module-level and methods)
	if node.Type() == "function_definition" {
		funcName := extractFunctionNameFromNode(node, sourceCode)
		if funcName != "" {
			if currentFunction == "" {
				// Module-level function
				newFunction = modulePath + "." + funcName
			} else {
				// Method inside a class or nested function
				newFunction = currentFunction + "." + funcName
			}
		}
	}

	// Look for return statements
	if node.Type() == "return_statement" && newFunction != "" {
		// Get the return value (skip the "return" keyword)
		var valueNode *sitter.Node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			// Skip the "return" keyword and any whitespace
			if child.Type() != "return" {
				valueNode = child
				break
			}
		}

		if valueNode != nil {
			// Track that this function has at least one return <expr> statement
			functionsWithReturnValues[newFunction] = true

			returnType := inferReturnType(valueNode, sourceCode, modulePath, builtinRegistry, importMap)
			if returnType != nil {
				stmt := &ReturnStatement{
					FunctionFQN: newFunction,
					ReturnType:  returnType,
					Location: Location{
						File:   filePath,
						Line:   node.StartPoint().Row + 1,
						Column: node.StartPoint().Column + 1,
					},
				}
				*returns = append(*returns, stmt)
			}
		}
	}

	// Recurse with updated function context
	for i := 0; i < int(node.ChildCount()); i++ {
		traverseForReturns(node.Child(i), sourceCode, filePath, modulePath, newFunction, returns, functionsWithReturnValues, builtinRegistry, importMap)
	}
}

func inferReturnType(
	node *sitter.Node,
	sourceCode []byte,
	modulePath string,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) *core.TypeInfo {
	if node == nil {
		return nil
	}

	nodeType := node.Type()

	switch nodeType {
	case "string", "concatenated_string":
		return &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "integer":
		return &core.TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "float":
		return &core.TypeInfo{
			TypeFQN:    "builtins.float",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "true", "false", "True", "False":
		return &core.TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "list":
		return &core.TypeInfo{
			TypeFQN:    "builtins.list",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "dictionary":
		return &core.TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "set":
		return &core.TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "tuple":
		return &core.TypeInfo{
			TypeFQN:    "builtins.tuple",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "none":
		return &core.TypeInfo{
			TypeFQN:    "builtins.NoneType",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "call":
		// Try class instantiation first (Task 7)
		//
		// CROSS-FILE IMPORT RESOLUTION:
		// Use provided importMap to resolve class instantiations from imports in return statements.
		// This enables patterns like:
		//   from models import User
		//   def create_user():
		//       return User()  # ← resolves to models.User (not local)
		//
		// Edge cases handled:
		// - Inline object creation in returns: return Calculator().get_result()
		// - Multi-line returns with chained calls
		// - Null/empty importMap (tests): Falls back to heuristic resolution
		classType := ResolveClassInstantiation(node, sourceCode, modulePath, importMap, nil)
		if classType != nil {
			return classType
		}

		// Return type from function call - will be enhanced in later tasks
		// The first child is usually the function being called
		var functionNode *sitter.Node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() != "argument_list" && child.Type() != "(" && child.Type() != ")" {
				functionNode = child
				break
			}
		}

		if functionNode != nil {
			funcName := functionNode.Content(sourceCode)

			// Check if it's a builtin type constructor
			if builtinRegistry != nil {
				// Try with builtins. prefix
				builtinType := builtinRegistry.GetType("builtins." + funcName)
				if builtinType != nil {
					return &core.TypeInfo{
						TypeFQN:    builtinType.FQN,
						Confidence: 0.9,
						Source:     "return_builtin_constructor",
					}
				}
			}

			// Placeholder for function calls - will be resolved later
			return &core.TypeInfo{
				TypeFQN:    "call:" + funcName,
				Confidence: 0.3,
				Source:     "return_function_call",
			}
		}

	case "identifier":
		// Return variable - needs scope lookup (Phase 2 Task 8)
		varName := node.Content(sourceCode)
		return &core.TypeInfo{
			TypeFQN:    "var:" + varName,
			Confidence: 0.2,
			Source:     "return_variable",
		}

	case "not_operator":
		// `return not x` always produces bool
		return &core.TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "return_not_operator",
		}

	case "comparison_operator":
		// `return x > 5`, `return a == b`, `return x in items`
		return &core.TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "return_comparison",
		}

	case "boolean_operator":
		// `return x or y`, `return a and b`
		// Try to infer from operands — both sides often have the same type
		if node.ChildCount() >= 3 {
			left := inferReturnType(node.Child(0), sourceCode, modulePath, builtinRegistry, importMap)
			right := inferReturnType(node.Child(2), sourceCode, modulePath, builtinRegistry, importMap)

			// Filter out placeholders (var: and call: prefixes)
			leftConcrete := left != nil && !strings.HasPrefix(left.TypeFQN, "var:") && !strings.HasPrefix(left.TypeFQN, "call:")
			rightConcrete := right != nil && !strings.HasPrefix(right.TypeFQN, "var:") && !strings.HasPrefix(right.TypeFQN, "call:")

			if leftConcrete && rightConcrete && left.TypeFQN == right.TypeFQN {
				return &core.TypeInfo{
					TypeFQN:    left.TypeFQN,
					Confidence: left.Confidence * 0.9,
					Source:     "return_boolean_operator",
				}
			}
			// If only one side has a concrete type, use it with lower confidence
			if leftConcrete {
				return &core.TypeInfo{
					TypeFQN:    left.TypeFQN,
					Confidence: left.Confidence * 0.7,
					Source:     "return_boolean_operator",
				}
			}
			if rightConcrete {
				return &core.TypeInfo{
					TypeFQN:    right.TypeFQN,
					Confidence: right.Confidence * 0.7,
					Source:     "return_boolean_operator",
				}
			}
		}

	case "conditional_expression":
		// `return x if cond else y` — tree-sitter children: [body, "if", condition, "else", orelse]
		if node.ChildCount() >= 5 {
			trueExpr := inferReturnType(node.Child(0), sourceCode, modulePath, builtinRegistry, importMap)
			falseExpr := inferReturnType(node.Child(4), sourceCode, modulePath, builtinRegistry, importMap)
			if trueExpr != nil && falseExpr != nil && trueExpr.TypeFQN == falseExpr.TypeFQN {
				return &core.TypeInfo{
					TypeFQN:    trueExpr.TypeFQN,
					Confidence: trueExpr.Confidence * 0.9,
					Source:     "return_conditional",
				}
			}
			// If both sides differ, take the higher confidence one
			if trueExpr != nil && falseExpr != nil {
				if trueExpr.Confidence >= falseExpr.Confidence {
					return &core.TypeInfo{
						TypeFQN:    trueExpr.TypeFQN,
						Confidence: trueExpr.Confidence * 0.6,
						Source:     "return_conditional",
					}
				}
				return &core.TypeInfo{
					TypeFQN:    falseExpr.TypeFQN,
					Confidence: falseExpr.Confidence * 0.6,
					Source:     "return_conditional",
				}
			}
			if trueExpr != nil {
				return trueExpr
			}
			if falseExpr != nil {
				return falseExpr
			}
		}

	case "parenthesized_expression":
		// `return (expr)` — unwrap and infer inner expression
		if node.ChildCount() >= 1 {
			// Find the inner expression (skip parentheses)
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() != "(" && child.Type() != ")" {
					return inferReturnType(child, sourceCode, modulePath, builtinRegistry, importMap)
				}
			}
		}

	case "unary_operator":
		// `return -x` — check the operator
		if node.ChildCount() >= 2 {
			op := node.Child(0).Content(sourceCode)
			if op == "-" || op == "+" || op == "~" {
				// Unary numeric operators — infer from operand
				operand := inferReturnType(node.Child(1), sourceCode, modulePath, builtinRegistry, importMap)
				if operand != nil && (operand.TypeFQN == "builtins.int" || operand.TypeFQN == "builtins.float") {
					return operand
				}
				// Default to int for unary - on unknown operand
				if op == "-" || op == "+" {
					return &core.TypeInfo{
						TypeFQN:    "builtins.int",
						Confidence: 0.5,
						Source:     "return_unary_operator",
					}
				}
			}
		}

	case "list_comprehension":
		return &core.TypeInfo{
			TypeFQN:    "builtins.list",
			Confidence: 1.0,
			Source:     "return_comprehension",
		}

	case "dictionary_comprehension":
		return &core.TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "return_comprehension",
		}

	case "set_comprehension":
		return &core.TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "return_comprehension",
		}

	case "generator_expression":
		return &core.TypeInfo{
			TypeFQN:    "builtins.Generator",
			Confidence: 0.9,
			Source:     "return_generator",
		}
	}

	return nil
}

// MergeReturnTypes combines multiple return statements for same function.
// Takes the highest confidence return type.
func MergeReturnTypes(statements []*ReturnStatement) map[string]*core.TypeInfo {
	merged := make(map[string]*core.TypeInfo)

	for _, stmt := range statements {
		existing, ok := merged[stmt.FunctionFQN]
		if !ok {
			merged[stmt.FunctionFQN] = stmt.ReturnType
			continue
		}

		// If new type has higher confidence, use it
		if stmt.ReturnType.Confidence > existing.Confidence {
			merged[stmt.FunctionFQN] = stmt.ReturnType
		}
	}

	return merged
}

// AddReturnTypesToEngine populates TypeInferenceEngine with return types.
// Thread-safe for concurrent writes.
func (te *TypeInferenceEngine) AddReturnTypesToEngine(returnTypes map[string]*core.TypeInfo) {
	te.typeMutex.Lock()
	defer te.typeMutex.Unlock()

	maps.Copy(te.ReturnTypes, returnTypes)
}

// isPascalCase checks if a string is in PascalCase (likely a class name).
func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}

	// First character must be uppercase letter
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}

	// Single character uppercase is considered PascalCase (e.g., "U")
	if len(s) == 1 {
		return true
	}

	// Must not be all caps (constants are UPPER_SNAKE_CASE)
	allCaps := true
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			allCaps = false
			break
		}
	}

	return !allCaps
}

// ResolveClassInstantiation attempts to resolve class instantiation patterns.
func ResolveClassInstantiation(
	callNode *sitter.Node,
	sourceCode []byte,
	modulePath string,
	importMap *core.ImportMap,
	registry *core.ModuleRegistry,
) *core.TypeInfo {
	if callNode == nil || callNode.Type() != "call" {
		return nil
	}

	// Get the function node (what's being called)
	var functionNode *sitter.Node
	for i := 0; i < int(callNode.ChildCount()); i++ {
		child := callNode.Child(i)
		if child.Type() != "argument_list" && child.Type() != "(" && child.Type() != ")" {
			functionNode = child
			break
		}
	}

	if functionNode == nil {
		return nil
	}

	funcName := functionNode.Content(sourceCode)

	// Check for attribute access (e.g., models.User())
	if strings.Contains(funcName, ".") {
		parts := strings.Split(funcName, ".")
		className := parts[len(parts)-1]

		// Last part should be PascalCase
		if !isPascalCase(className) {
			return nil
		}

		// Try to resolve through imports
		if importMap != nil {
			basePart := strings.Join(parts[:len(parts)-1], ".")
			resolvedModule, ok := importMap.Resolve(basePart)
			if ok && resolvedModule != "" {
				return &core.TypeInfo{
					TypeFQN:    resolvedModule + "." + className,
					Confidence: 0.9,
					Source:     "class_instantiation_import",
				}
			}
		}

		// Heuristic: assume it's a class in same module or submodule
		return &core.TypeInfo{
			TypeFQN:    modulePath + "." + funcName,
			Confidence: 0.7,
			Source:     "class_instantiation_heuristic",
		}
	}

	// Simple name (e.g., User())
	if isPascalCase(funcName) {
		// Check imports first
		if importMap != nil {
			resolvedFQN, ok := importMap.Resolve(funcName)
			if ok && resolvedFQN != "" {
				return &core.TypeInfo{
					TypeFQN:    resolvedFQN,
					Confidence: 0.95,
					Source:     "class_instantiation_import",
				}
			}
		}

		// Check if class exists in module registry
		classFQN := modulePath + "." + funcName
		if registry != nil {
			// Simplified check - in real implementation, would verify class exists
			// For now, use heuristic
			return &core.TypeInfo{
				TypeFQN:    classFQN,
				Confidence: 0.8,
				Source:     "class_instantiation_local",
			}
		}

		return &core.TypeInfo{
			TypeFQN:    classFQN,
			Confidence: 0.6,
			Source:     "class_instantiation_guess",
		}
	}

	return nil
}

// extractFunctionNameFromNode extracts the function name from a function_definition node.
func extractFunctionNameFromNode(node *sitter.Node, sourceCode []byte) string {
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

func extractClassNameFromNode(node *sitter.Node, sourceCode []byte) string {
	if node.Type() != "class_definition" {
		return ""
	}

	// Find the identifier node (class name)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return child.Content(sourceCode)
		}
	}

	return ""
}
