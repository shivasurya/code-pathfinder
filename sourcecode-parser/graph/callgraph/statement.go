package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// Deprecated: Use core.StatementType instead.
// This alias will be removed in a future version.
type StatementType = core.StatementType

const (
	// Deprecated: Use core.StatementTypeAssignment instead.
	StatementTypeAssignment = core.StatementTypeAssignment

	// Deprecated: Use core.StatementTypeCall instead.
	StatementTypeCall = core.StatementTypeCall

	// Deprecated: Use core.StatementTypeReturn instead.
	StatementTypeReturn = core.StatementTypeReturn

	// Deprecated: Use core.StatementTypeIf instead.
	StatementTypeIf = core.StatementTypeIf

	// Deprecated: Use core.StatementTypeFor instead.
	StatementTypeFor = core.StatementTypeFor

	// Deprecated: Use core.StatementTypeWhile instead.
	StatementTypeWhile = core.StatementTypeWhile

	// Deprecated: Use core.StatementTypeWith instead.
	StatementTypeWith = core.StatementTypeWith

	// Deprecated: Use core.StatementTypeTry instead.
	StatementTypeTry = core.StatementTypeTry

	// Deprecated: Use core.StatementTypeRaise instead.
	StatementTypeRaise = core.StatementTypeRaise

	// Deprecated: Use core.StatementTypeImport instead.
	StatementTypeImport = core.StatementTypeImport

	// Deprecated: Use core.StatementTypeExpression instead.
	StatementTypeExpression = core.StatementTypeExpression
)

// Deprecated: Use core.Statement instead.
// This alias will be removed in a future version.
type Statement = core.Statement

// Deprecated: Use core.DefUseChain instead.
// This alias will be removed in a future version.
type DefUseChain = core.DefUseChain

// Deprecated: Use core.DefUseStats instead.
// This alias will be removed in a future version.
type DefUseStats = core.DefUseStats

// NewDefUseChain is a convenience wrapper.
// Deprecated: Use core.NewDefUseChain instead.
func NewDefUseChain() *DefUseChain {
	return core.NewDefUseChain()
}

// BuildDefUseChains is a convenience wrapper.
// Deprecated: Use core.BuildDefUseChains instead.
func BuildDefUseChains(statements []*Statement) *DefUseChain {
	return core.BuildDefUseChains(statements)
}
