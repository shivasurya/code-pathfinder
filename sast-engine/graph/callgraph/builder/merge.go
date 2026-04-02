package builder

import (
	"maps"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// MergeCallGraphs adds all entries from src into dst.
// This enables combining Go and Python call graphs into a unified graph.
//
// FQN namespaces prevent collisions:
//
//	Go:     "github.com/myapp/handlers.HandleRequest"
//	Python: "myapp.handlers.handle_request"
//	Java:   "com.myapp.handlers.HandleRequest"
//
// All maps are merged by appending src entries to dst:
//   - Functions: Direct assignment (FQNs are unique)
//   - CallSites: Append sites from src to dst
//   - Edges: Append callees from src to dst
//   - ReverseEdges: Append callers from src to dst
//
// Parameters:
//   - dst: destination call graph (e.g., Python call graph)
//   - src: source call graph (e.g., Go call graph)
//
// Note: This function modifies dst in-place. No return value.
func MergeCallGraphs(dst, src *core.CallGraph) {
	// Merge Functions map
	maps.Copy(dst.Functions, src.Functions)

	// Merge CallSites map
	for caller, sites := range src.CallSites {
		dst.CallSites[caller] = append(dst.CallSites[caller], sites...)
	}

	// Merge Edges map (forward call graph)
	for caller, callees := range src.Edges {
		dst.Edges[caller] = append(dst.Edges[caller], callees...)
	}

	// Merge ReverseEdges map (reverse call graph)
	for callee, callers := range src.ReverseEdges {
		dst.ReverseEdges[callee] = append(dst.ReverseEdges[callee], callers...)
	}

	// Merge dataflow analysis data (Statements, CFGs, CFGBlockStatements, Summaries).
	// These maps are keyed by FQN — Go and Python FQN namespaces are disjoint,
	// so maps.Copy is safe (no key collisions).
	maps.Copy(dst.Statements, src.Statements)
	maps.Copy(dst.CFGs, src.CFGs)
	maps.Copy(dst.CFGBlockStatements, src.CFGBlockStatements)
	maps.Copy(dst.Summaries, src.Summaries)
}
