package python

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

// ExtractFunctions converts CodeGraph nodes to language-agnostic FunctionDef structures.
// It iterates through all nodes in the CodeGraph and extracts function definitions
// (Type="function_definition"), converting them to the FunctionDef type introduced in PR-01.
//
// Parameters:
//   - module: ParsedModule containing CodeGraph in AST field
//
// Returns:
//   - Slice of FunctionDef structures representing all functions in the module
//   - Error if AST is not a CodeGraph or extraction fails
func (p *PythonAnalyzer) ExtractFunctions(
	module *callgraph.ParsedModule,
) ([]*callgraph.FunctionDef, error) {
	codeGraph, ok := module.AST.(*graph.CodeGraph)
	if !ok {
		return nil, fmt.Errorf("expected *graph.CodeGraph, got %T", module.AST)
	}

	functions := make([]*callgraph.FunctionDef, 0)

	// Iterate through all nodes to find function definitions
	for _, node := range codeGraph.Nodes {
		if node.Type == "function_definition" {
			functionDef := &callgraph.FunctionDef{
				Name:       node.Name,
				FQN:        node.ID, // Use node ID as FQN
				Parameters: extractParameters(node),
				ReturnType: extractReturnType(node),
				Body:       node, // Keep original Node for later passes
				Location: callgraph.Location{
					File:   module.FilePath,
					Line:   int(node.LineNumber),
					Column: 0, // Column information not available in Node structure
				},
			}

			functions = append(functions, functionDef)
		}
	}

	return functions, nil
}

// ExtractClasses converts CodeGraph nodes to language-agnostic ClassDef structures.
// It iterates through all nodes in the CodeGraph and extracts class definitions
// (Type="class_definition"), converting them to the ClassDef type introduced in PR-01.
//
// Parameters:
//   - module: ParsedModule containing CodeGraph in AST field
//
// Returns:
//   - Slice of ClassDef structures representing all classes in the module
//   - Error if AST is not a CodeGraph or extraction fails
func (p *PythonAnalyzer) ExtractClasses(
	module *callgraph.ParsedModule,
) ([]*callgraph.ClassDef, error) {
	codeGraph, ok := module.AST.(*graph.CodeGraph)
	if !ok {
		return nil, fmt.Errorf("expected *graph.CodeGraph, got %T", module.AST)
	}

	classes := make([]*callgraph.ClassDef, 0)

	// Iterate through all nodes to find class definitions
	for _, node := range codeGraph.Nodes {
		if node.Type == "class_definition" {
			classDef := &callgraph.ClassDef{
				Name:        node.Name,
				FQN:         node.ID, // Use node ID as FQN
				Methods:     extractMethods(codeGraph, node),
				Attributes:  extractAttributes(node),
				BaseClasses: extractBaseClasses(node),
				Location: callgraph.Location{
					File:   module.FilePath,
					Line:   int(node.LineNumber),
					Column: 0, // Column information not available in Node structure
				},
			}

			classes = append(classes, classDef)
		}
	}

	return classes, nil
}

// extractParameters converts Node.ArgumentTypes and ArgumentNames to Parameter structs.
// It handles type annotations if available, creating TypeInfo structures for each parameter.
//
// Parameters:
//   - funcNode: Node representing a function definition
//
// Returns:
//   - Slice of Parameter structures with name, position, and type information
func extractParameters(funcNode *graph.Node) []callgraph.Parameter {
	params := make([]callgraph.Parameter, 0)

	// Handle case where MethodArgumentsValue and MethodArgumentsType are available
	for i, argName := range funcNode.MethodArgumentsValue {
		parameter := callgraph.Parameter{
			Name:     argName,
			Position: i,
		}

		// Extract type if available
		if i < len(funcNode.MethodArgumentsType) && funcNode.MethodArgumentsType[i] != "" {
			parameter.Type = &callgraph.TypeInfo{
				TypeFQN:    funcNode.MethodArgumentsType[i],
				Confidence: 1.0, // Type annotation = high confidence
				Source:     "annotation",
			}
		}

		params = append(params, parameter)
	}

	return params
}

// extractReturnType extracts return type annotation from function node.
// Returns nil if no return type annotation is present.
//
// Parameters:
//   - funcNode: Node representing a function definition
//
// Returns:
//   - TypeInfo if return type annotation exists, nil otherwise
func extractReturnType(funcNode *graph.Node) *callgraph.TypeInfo {
	if funcNode.ReturnType == "" {
		return nil
	}

	return &callgraph.TypeInfo{
		TypeFQN:    funcNode.ReturnType,
		Confidence: 1.0, // Type annotation = high confidence
		Source:     "annotation",
	}
}

// extractMethods extracts all methods from a class node by finding child function nodes.
// It searches through edges to find all functions that belong to this class.
//
// Parameters:
//   - codeGraph: Complete code graph with all nodes and edges
//   - classNode: Node representing the class definition
//
// Returns:
//   - Slice of FunctionDef structures representing all methods in the class
func extractMethods(codeGraph *graph.CodeGraph, classNode *graph.Node) []*callgraph.FunctionDef {
	methods := make([]*callgraph.FunctionDef, 0)

	// Find all edges from this class to child function nodes
	for _, edge := range codeGraph.Edges {
		if edge.From.ID == classNode.ID && edge.To.Type == "function_definition" {
			methodNode := edge.To
			methodDef := &callgraph.FunctionDef{
				Name:       methodNode.Name,
				FQN:        methodNode.ID,
				Parameters: extractParameters(methodNode),
				ReturnType: extractReturnType(methodNode),
				Body:       methodNode,
				Location: callgraph.Location{
					File:   methodNode.File,
					Line:   int(methodNode.LineNumber),
					Column: 0, // Column information not available in Node structure
				},
			}

			methods = append(methods, methodDef)
		}
	}

	return methods
}

// extractAttributes extracts class attributes with type information.
// Currently returns empty map as attribute extraction is not fully implemented
// in the existing CodeGraph structure. Will be enhanced in later PRs.
//
// Parameters:
//   - classNode: Node representing the class definition
//
// Returns:
//   - Map of attribute names to TypeInfo (currently empty)
func extractAttributes(classNode *graph.Node) map[string]*callgraph.TypeInfo {
	attributes := make(map[string]*callgraph.TypeInfo)

	// Note: Full attribute extraction requires traversing class body nodes
	// This is a placeholder implementation
	// TODO: Implement full attribute extraction in future PR

	return attributes
}

// extractBaseClasses extracts parent class names from class node.
// Currently returns empty slice as base class information is not stored
// in the current Node structure. Will be enhanced in later PRs.
//
// Parameters:
//   - classNode: Node representing the class definition
//
// Returns:
//   - Slice of base class names (currently empty)
func extractBaseClasses(classNode *graph.Node) []string {
	baseClasses := make([]string, 0)

	// Note: Base class information not currently stored in Node structure
	// This is a placeholder implementation
	// TODO: Implement base class extraction in future PR

	return baseClasses
}
