package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// Deprecated: Use core.TaintInfo instead.
// This alias will be removed in a future version.
type TaintInfo = core.TaintInfo

// Deprecated: Use core.TaintSummary instead.
// This alias will be removed in a future version.
type TaintSummary = core.TaintSummary

// NewTaintSummary is a convenience wrapper.
// Deprecated: Use core.NewTaintSummary instead.
func NewTaintSummary(functionFQN string) *TaintSummary {
	return core.NewTaintSummary(functionFQN)
}
