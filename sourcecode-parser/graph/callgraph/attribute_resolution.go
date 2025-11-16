package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ResolveSelfAttributeCall resolves self.attr.method() calls using attribute registry.
// Deprecated: Use resolution.ResolveSelfAttributeCall instead.
func ResolveSelfAttributeCall(
	target string,
	callerFQN string,
	typeEngine *resolution.TypeInferenceEngine,
	builtins *registry.BuiltinRegistry,
	callGraph *core.CallGraph,
) (string, bool, *core.TypeInfo) {
	return resolution.ResolveSelfAttributeCall(target, callerFQN, typeEngine, builtins, callGraph)
}

// PrintAttributeFailureStats prints statistics about attribute resolution failures.
// Deprecated: Use resolution.PrintAttributeFailureStats instead.
func PrintAttributeFailureStats() {
	resolution.PrintAttributeFailureStats()
}

// ResolveAttributePlaceholders resolves __ATTR__ placeholders in call targets.
// Deprecated: Use resolution.ResolveAttributePlaceholders instead.
func ResolveAttributePlaceholders(
	attrRegistry *registry.AttributeRegistry,
	typeEngine *resolution.TypeInferenceEngine,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
) {
	resolution.ResolveAttributePlaceholders(attrRegistry, typeEngine, moduleRegistry, codeGraph)
}
