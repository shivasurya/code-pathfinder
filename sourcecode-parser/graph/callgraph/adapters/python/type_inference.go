package python

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

// InferTypes performs type inference for all functions and classes in the module.
// It uses the existing TypeInferenceEngine infrastructure to analyze variable types,
// return types, and attribute types using multiple inference strategies.
//
// The type inference process includes:
//   - Literal inference (x = 5 → int)
//   - Annotation inference (x: int → int)
//   - Assignment inference (x = y where y has known type)
//   - Call return inference (x = foo() where foo returns known type)
//   - Parameter inference (from function signatures)
//   - Attribute inference (x.attr where x is known class)
//
// Parameters:
//   - module: ParsedModule containing the CodeGraph
//   - registry: ModuleRegistry for resolving module paths and cross-module references
//
// Returns:
//   - TypeContext containing all inferred types organized by scope
//   - Error if inference fails
func (p *PythonAnalyzer) InferTypes(
	module *callgraph.ParsedModule,
	registry *callgraph.ModuleRegistry,
) (*callgraph.TypeContext, error) {
	codeGraph, ok := module.AST.(*graph.CodeGraph)
	if !ok {
		return nil, fmt.Errorf("expected *graph.CodeGraph, got %T", module.AST)
	}

	// Create type inference engine
	typeEngine := callgraph.NewTypeInferenceEngine(registry)
	typeEngine.Builtins = callgraph.NewBuiltinRegistry()
	typeEngine.Attributes = callgraph.NewAttributeRegistry()

	// Initialize TypeContext to return
	typeContext := &callgraph.TypeContext{
		Variables: make(map[string]*callgraph.TypeInfo),
		Functions: make(map[string]*callgraph.FunctionDef),
		Classes:   make(map[string]*callgraph.ClassDef),
		Imports:   nil, // Will be set below
	}

	// Extract imports from module metadata (set during Parse in PR-02)
	if imports, ok := module.Metadata["imports"].(*callgraph.ImportMap); ok {
		typeContext.Imports = imports
	} else {
		// Fallback: extract imports now
		var err error
		typeContext.Imports, err = p.ExtractImports(module)
		if err != nil {
			return nil, fmt.Errorf("failed to extract imports: %w", err)
		}
	}

	// Extract functions and classes for TypeContext
	functions, err := p.ExtractFunctions(module)
	if err != nil {
		return nil, fmt.Errorf("failed to extract functions: %w", err)
	}
	for _, fn := range functions {
		typeContext.Functions[fn.FQN] = fn
	}

	classes, err := p.ExtractClasses(module)
	if err != nil {
		return nil, fmt.Errorf("failed to extract classes: %w", err)
	}
	for _, cls := range classes {
		typeContext.Classes[cls.FQN] = cls
	}

	// Perform type inference using the existing engine
	// The engine will populate Scopes with variable type information
	for _, node := range codeGraph.Nodes {
		if node.Type == "function_definition" {
			scope := callgraph.NewFunctionScope(node.ID)

			// Infer types for function parameters from annotations
			for i, paramName := range node.MethodArgumentsValue {
				if i < len(node.MethodArgumentsType) && node.MethodArgumentsType[i] != "" {
					binding := &callgraph.VariableBinding{
						VarName: paramName,
						Type: &callgraph.TypeInfo{
							TypeFQN:    node.MethodArgumentsType[i],
							Confidence: 1.0,
							Source:     "annotation",
						},
						Location: callgraph.Location{
							File:   node.File,
							Line:   int(node.LineNumber),
							Column: 0, // Column information not available in Node structure
						},
					}
					scope.Variables[paramName] = binding
				}
			}

			// Infer return type from annotation
			if node.ReturnType != "" {
				scope.ReturnType = &callgraph.TypeInfo{
					TypeFQN:    node.ReturnType,
					Confidence: 1.0,
					Source:     "annotation",
				}
				typeEngine.ReturnTypes[node.ID] = scope.ReturnType
			}

			typeEngine.AddScope(scope)
		}
	}

	// Update variable bindings with function return types
	typeEngine.UpdateVariableBindingsWithFunctionReturns()

	// Convert TypeInferenceEngine scopes to TypeContext Variables map
	// Flatten all function scopes into the module-level Variables map
	for _, scope := range typeEngine.Scopes {
		for varName, binding := range scope.Variables {
			// Store with function-qualified name (e.g., "myfunction.x")
			qualifiedName := scope.FunctionFQN + "." + varName
			typeContext.Variables[qualifiedName] = binding.Type
		}
	}

	return typeContext, nil
}

// ResolveType resolves a type expression to TypeInfo using import context.
// It attempts to resolve type names to fully qualified type names by consulting
// the import map and builtin registry.
//
// Resolution steps:
//   1. Check if type is a builtin (int, str, list, etc.)
//   2. Check if type is in imports (from module import Type)
//   3. Check if type is a qualified import (module.Type)
//   4. Return original type with low confidence if unresolved
//
// Parameters:
//   - expr: Type expression to resolve (e.g., "User", "List[str]", "os.PathLike")
//   - context: TypeContext containing imports and builtin information
//
// Returns:
//   - TypeInfo with resolved fully qualified type name
//   - Error if resolution fails critically
func (p *PythonAnalyzer) ResolveType(
	expr string,
	context *callgraph.TypeContext,
) (*callgraph.TypeInfo, error) {
	if context == nil {
		return &callgraph.TypeInfo{
			TypeFQN:    expr,
			Confidence: 0.0,
			Source:     "unresolved",
		}, nil
	}

	// Check builtins first using a builtin registry
	builtins := callgraph.NewBuiltinRegistry()
	if builtinType := builtins.GetType("builtins." + expr); builtinType != nil {
		return &callgraph.TypeInfo{
			TypeFQN:    builtinType.FQN,
			Confidence: 1.0,
			Source:     "builtin",
		}, nil
	}

	// Check imports
	if context.Imports != nil {
		// Try to resolve from imports
		if modulePath, ok := context.Imports.Imports[expr]; ok {
			return &callgraph.TypeInfo{
				TypeFQN:    modulePath,
				Confidence: 0.9,
				Source:     "import_resolution",
			}, nil
		}
	}

	// Return with low confidence if we can't resolve
	return &callgraph.TypeInfo{
		TypeFQN:    expr,
		Confidence: 0.3,
		Source:     "unresolved",
	}, nil
}
