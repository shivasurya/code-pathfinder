package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ChainStep represents one step in a method chain.
// Deprecated: Use resolution.ChainStep instead.
type ChainStep = resolution.ChainStep

// ParseChain parses a method chain into individual steps.
// Deprecated: Use resolution.ParseChain instead.
func ParseChain(target string) []resolution.ChainStep {
	return resolution.ParseChain(target)
}

// ResolveChainedCall resolves a method chain by walking each step and tracking types.
// Deprecated: Use resolution.ResolveChainedCall instead.
func ResolveChainedCall(
	target string,
	typeEngine *resolution.TypeInferenceEngine,
	builtins *registry.BuiltinRegistry,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
	callerFQN string,
	currentModule string,
	callGraph *core.CallGraph,
) (string, bool, *core.TypeInfo) {
	return resolution.ResolveChainedCall(target, typeEngine, builtins, moduleRegistry, codeGraph, callerFQN, currentModule, callGraph)
}
