package callgraph

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ReturnStatement represents a return statement in a function.
type ReturnStatement struct {
	FunctionFQN string
	ReturnType  *TypeInfo
	Location    Location
}

// ExtractReturnTypes analyzes return statements in all functions in a file.
func ExtractReturnTypes(
	filePath string,
	sourceCode []byte,
	modulePath string,
	builtinRegistry *BuiltinRegistry,
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
	traverseForReturns(tree.RootNode(), sourceCode, filePath, modulePath, "", &returns, builtinRegistry)

	return returns, nil
}

func traverseForReturns(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	returns *[]*ReturnStatement,
	builtinRegistry *BuiltinRegistry,
) {
	if node == nil {
		return
	}

	// Track when we enter a function
	newFunction := currentFunction
	if node.Type() == "function_definition" {
		funcName := extractFunctionNameFromNode(node, sourceCode)
		if funcName != "" {
			if currentFunction == "" {
				// Module-level function
				newFunction = modulePath + "." + funcName
			} else {
				// Nested function
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
			returnType := inferReturnType(valueNode, sourceCode, modulePath, builtinRegistry)
			if returnType != nil {
				stmt := &ReturnStatement{
					FunctionFQN: newFunction,
					ReturnType:  returnType,
					Location: Location{
						File:   filePath,
						Line:   int(node.StartPoint().Row) + 1,
						Column: int(node.StartPoint().Column) + 1,
					},
				}
				*returns = append(*returns, stmt)
			}
		}
	}

	// Recurse with updated function context
	for i := 0; i < int(node.ChildCount()); i++ {
		traverseForReturns(node.Child(i), sourceCode, filePath, modulePath, newFunction, returns, builtinRegistry)
	}
}

func inferReturnType(
	node *sitter.Node,
	sourceCode []byte,
	_ string, // modulePath - reserved for future use
	builtinRegistry *BuiltinRegistry,
) *TypeInfo {
	if node == nil {
		return nil
	}

	nodeType := node.Type()

	switch nodeType {
	case "string":
		return &TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "integer":
		return &TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "float":
		return &TypeInfo{
			TypeFQN:    "builtins.float",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "true", "false", "True", "False":
		return &TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "list":
		return &TypeInfo{
			TypeFQN:    "builtins.list",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "dictionary":
		return &TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "set":
		return &TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "tuple":
		return &TypeInfo{
			TypeFQN:    "builtins.tuple",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "none":
		return &TypeInfo{
			TypeFQN:    "builtins.NoneType",
			Confidence: 1.0,
			Source:     "return_literal",
		}

	case "call":
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
					return &TypeInfo{
						TypeFQN:    builtinType.FQN,
						Confidence: 0.9,
						Source:     "return_builtin_constructor",
					}
				}
			}

			// Placeholder for function calls - will be resolved later
			return &TypeInfo{
				TypeFQN:    "call:" + funcName,
				Confidence: 0.3,
				Source:     "return_function_call",
			}
		}

	case "identifier":
		// Return variable - needs scope lookup (Phase 2 Task 8)
		varName := node.Content(sourceCode)
		return &TypeInfo{
			TypeFQN:    "var:" + varName,
			Confidence: 0.2,
			Source:     "return_variable",
		}
	}

	return nil
}

// MergeReturnTypes combines multiple return statements for same function.
// Takes the highest confidence return type.
func MergeReturnTypes(statements []*ReturnStatement) map[string]*TypeInfo {
	merged := make(map[string]*TypeInfo)

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
func (te *TypeInferenceEngine) AddReturnTypesToEngine(returnTypes map[string]*TypeInfo) {
	for funcFQN, typeInfo := range returnTypes {
		te.ReturnTypes[funcFQN] = typeInfo
	}
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
