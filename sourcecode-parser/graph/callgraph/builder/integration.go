package builder

import (
	"time"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// BuildCallGraphFromPath is a convenience function that builds a call graph
// from a project directory path.
//
// It performs all three passes:
//  1. Build module registry
//  2. Parse code graph (uses existing parsed graph)
//  3. Build call graph
//
// Parameters:
//   - codeGraph: the parsed code graph from graph.Initialize()
//   - projectPath: absolute path to project root
//
// Returns:
//   - CallGraph: complete call graph with edges and call sites
//   - ModuleRegistry: module path mappings
//   - error: if any step fails
func BuildCallGraphFromPath(codeGraph *graph.CodeGraph, projectPath string) (*core.CallGraph, *core.ModuleRegistry, error) {
	// Pass 1: Build module registry
	startRegistry := time.Now()
	moduleRegistry, err := registry.BuildModuleRegistry(projectPath)
	if err != nil {
		return nil, nil, err
	}
	elapsedRegistry := time.Since(startRegistry)

	// Pass 2-3: Build call graph (includes import extraction and call site extraction)
	startCallGraph := time.Now()
	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, projectPath)
	if err != nil {
		return nil, nil, err
	}
	elapsedCallGraph := time.Since(startCallGraph)

	// Log timing information
	graph.Log("Module registry built in:", elapsedRegistry)
	graph.Log("Call graph built in:", elapsedCallGraph)

	return callGraph, moduleRegistry, nil
}
