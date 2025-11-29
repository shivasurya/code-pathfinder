package builder

import (
	"log"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
)

// GenerateTaintSummaries analyzes all Python functions for taint flows.
// This is Pass 5 of the call graph building process.
//
// For each function:
//  1. Extract statements from AST
//  2. Build def-use chains
//  3. Analyze intra-procedural taint
//  4. Store TaintSummary in callGraph.Summaries
//
// Parameters:
//   - callGraph: the call graph being built (will be populated with summaries)
//   - codeGraph: the parsed AST nodes (currently unused, reserved for future use)
//   - registry: module registry (currently unused, reserved for future use)
func GenerateTaintSummaries(callGraph *core.CallGraph, codeGraph *graph.CodeGraph, registry *core.ModuleRegistry) {
	_ = codeGraph  // Reserved for future use
	_ = registry   // Reserved for future use
	analyzed := 0
	total := len(callGraph.Functions)

	// Iterate over all indexed functions
	for funcFQN, funcNode := range callGraph.Functions {
		// Read source code for this function's file
		sourceCode, err := ReadFileBytes(funcNode.File)
		if err != nil {
			log.Printf("Warning: failed to read file %s for taint analysis: %v", funcNode.File, err)
			continue
		}

		// Parse the Python file to get AST
		tree, err := extraction.ParsePythonFile(sourceCode)
		if err != nil {
			log.Printf("Warning: failed to parse %s for taint analysis: %v", funcNode.File, err)
			continue
		}

		// Find the function node in the AST by line number
		functionNode := FindFunctionAtLine(tree.RootNode(), funcNode.LineNumber)
		if functionNode == nil {
			log.Printf("Warning: could not find function %s at line %d", funcFQN, funcNode.LineNumber)
			if tree != nil {
				tree.Close()
			}
			continue
		}

		// Step 1: Extract statements from function
		statements, err := extraction.ExtractStatements(funcNode.File, sourceCode, functionNode)
		if err != nil {
			log.Printf("Warning: failed to extract statements from %s: %v", funcFQN, err)
			if tree != nil {
				tree.Close()
			}
			continue
		}

		// Step 2: Build def-use chains
		defUseChain := core.BuildDefUseChains(statements)

		// Step 3: Analyze intra-procedural taint
		// For MVP: use empty sources/sinks/sanitizers (will be populated from patterns in PR #6)
		summary := taint.AnalyzeIntraProceduralTaint(
			funcFQN,
			statements,
			defUseChain,
			[]string{}, // sources - will come from patterns
			[]string{}, // sinks - will come from patterns
			[]string{}, // sanitizers - will come from patterns
		)

		// Step 4: Store summary
		callGraph.Summaries[funcFQN] = summary

		analyzed++

		// Report progress every 1000 functions
		if analyzed%1000 == 0 {
			log.Printf("Analyzed %d/%d functions...", analyzed, total)
		}

		// Clean up tree-sitter tree
		if tree != nil {
			tree.Close()
		}
	}
}
