package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// Deprecated: Use taint.TaintState instead.
// This alias will be removed in a future version.
type TaintState = taint.TaintState

// NewTaintState creates an empty taint state.
// Deprecated: Use taint.NewTaintState instead.
func NewTaintState() *TaintState {
	return taint.NewTaintState()
}

// AnalyzeIntraProceduralTaint performs forward taint analysis on a function.
// Deprecated: Use taint.AnalyzeIntraProceduralTaint instead.
func AnalyzeIntraProceduralTaint(
	functionFQN string,
	statements []*core.Statement,
	defUseChain *core.DefUseChain,
	sources []string,
	sinks []string,
	sanitizers []string,
) *core.TaintSummary {
	return taint.AnalyzeIntraProceduralTaint(functionFQN, statements, defUseChain, sources, sinks, sanitizers)
}
