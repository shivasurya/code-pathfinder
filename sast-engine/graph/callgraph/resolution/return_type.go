package resolution

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
)

// ReturnStatement represents a return statement in a function.
type ReturnStatement struct {
	FunctionFQN string
	ReturnType  *core.TypeInfo
	Location    Location
}

// ExtractReturnTypes analyzes return statements in all functions in a file.
func ExtractReturnTypes(
	filePath string,
	sourceCode []byte,
	modulePath string,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) ([]*ReturnStatement, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var returns []*ReturnStatement
	traverseForReturns(tree.RootNode(), sourceCode, filePath, modulePath, "", &returns, builtinRegistry, importMap)

	return returns, nil
}

func traverseForReturns(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	returns *[]*ReturnStatement,
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
		traverseForReturns(node.Child(i), sourceCode, filePath, modulePath, newFunction, returns, builtinRegistry, importMap)
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
	case "string":
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

	for funcFQN, typeInfo := range returnTypes {
		te.ReturnTypes[funcFQN] = typeInfo
	}
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
