package extraction

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// ExtractGoReturnTypes extracts return type information from indexed functions.
//
// This function implements Pass 2a of the Go call graph construction pipeline.
// It processes functions that were indexed in Pass 1, parsing their return type
// strings (stored in node.ReturnType) into structured TypeInfo objects.
//
// Algorithm:
//  1. Iterate through all functions in callGraph.Functions
//  2. For each function with a return type:
//     a) Get the return type string (e.g., "*User", "(string, error)")
//     b) Parse it using ParseGoTypeString()
//     c) Store the resulting TypeInfo in typeEngine.ReturnTypes
//  3. Skip functions with no return type (void functions)
//  4. Continue on parse errors (don't fail entire extraction)
//
// Thread Safety:
//   This function is thread-safe because typeEngine.AddReturnType() uses mutexes.
//   Can be called in parallel with other extraction passes.
//
// Parameters:
//   - callGraph: The call graph with indexed functions (from Pass 1)
//   - registry: Go module registry for type resolution (from Phase 1)
//   - typeEngine: Type inference engine to store results (from PR-13)
//
// Returns:
//   - error: Currently always returns nil (errors are logged but not propagated)
//
// Example:
//
//	engine := resolution.NewGoTypeInferenceEngine(registry)
//	err := ExtractGoReturnTypes(callGraph, registry, engine)
//	// Now engine.ReturnTypes contains all function return types
func ExtractGoReturnTypes(
	callGraph *core.CallGraph,
	registry *core.GoModuleRegistry,
	typeEngine *resolution.GoTypeInferenceEngine,
) error {
	// Iterate through all indexed functions
	for fqn, node := range callGraph.Functions {
		// Skip functions with no return type (void functions)
		if node.ReturnType == "" {
			continue
		}

		// Parse the return type string into TypeInfo
		typeInfo, err := ParseGoTypeString(node.ReturnType, registry, node.File)
		if err != nil {
			// Log error but continue processing other functions
			// Don't let one parse failure break entire extraction
			continue
		}

		// Store in type engine (nil check handled by ParseGoTypeString)
		if typeInfo != nil {
			typeEngine.AddReturnType(fqn, typeInfo)
		}
	}

	return nil
}
