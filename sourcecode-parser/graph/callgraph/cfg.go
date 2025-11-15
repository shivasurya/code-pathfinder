package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/cfg"
)

// Deprecated: Use cfg.BlockType instead.
// This alias will be removed in a future version.
type BlockType = cfg.BlockType

// Deprecated: Use cfg.BlockTypeEntry instead.
// This constant will be removed in a future version.
const BlockTypeEntry = cfg.BlockTypeEntry

// Deprecated: Use cfg.BlockTypeExit instead.
// This constant will be removed in a future version.
const BlockTypeExit = cfg.BlockTypeExit

// Deprecated: Use cfg.BlockTypeNormal instead.
// This constant will be removed in a future version.
const BlockTypeNormal = cfg.BlockTypeNormal

// Deprecated: Use cfg.BlockTypeConditional instead.
// This constant will be removed in a future version.
const BlockTypeConditional = cfg.BlockTypeConditional

// Deprecated: Use cfg.BlockTypeLoop instead.
// This constant will be removed in a future version.
const BlockTypeLoop = cfg.BlockTypeLoop

// Deprecated: Use cfg.BlockTypeSwitch instead.
// This constant will be removed in a future version.
const BlockTypeSwitch = cfg.BlockTypeSwitch

// Deprecated: Use cfg.BlockTypeTry instead.
// This constant will be removed in a future version.
const BlockTypeTry = cfg.BlockTypeTry

// Deprecated: Use cfg.BlockTypeCatch instead.
// This constant will be removed in a future version.
const BlockTypeCatch = cfg.BlockTypeCatch

// Deprecated: Use cfg.BlockTypeFinally instead.
// This constant will be removed in a future version.
const BlockTypeFinally = cfg.BlockTypeFinally

// Deprecated: Use cfg.BasicBlock instead.
// This alias will be removed in a future version.
type BasicBlock = cfg.BasicBlock

// Deprecated: Use cfg.ControlFlowGraph instead.
// This alias will be removed in a future version.
type ControlFlowGraph = cfg.ControlFlowGraph

// Deprecated: Use cfg.NewControlFlowGraph instead.
// This wrapper will be removed in a future version.
func NewControlFlowGraph(functionFQN string) *ControlFlowGraph {
	return cfg.NewControlFlowGraph(functionFQN)
}
