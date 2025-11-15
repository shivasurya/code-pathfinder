package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// Deprecated: Use core.TypeInfo instead.
type TypeInfo = core.TypeInfo

// Deprecated: Use resolution.VariableBinding instead.
type VariableBinding = resolution.VariableBinding

// Deprecated: Use resolution.FunctionScope instead.
type FunctionScope = resolution.FunctionScope

// Deprecated: Use resolution.TypeInferenceEngine instead.
type TypeInferenceEngine = resolution.TypeInferenceEngine

// NewTypeInferenceEngine creates a new type inference engine.
// Deprecated: Use resolution.NewTypeInferenceEngine instead.
func NewTypeInferenceEngine(registry *core.ModuleRegistry) *TypeInferenceEngine {
	return resolution.NewTypeInferenceEngine(registry)
}

// NewFunctionScope creates a new function scope.
// Deprecated: Use resolution.NewFunctionScope instead.
func NewFunctionScope(functionFQN string) *FunctionScope {
	return resolution.NewFunctionScope(functionFQN)
}
