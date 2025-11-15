package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// Deprecated: Use registry.BuiltinMethod instead.
// This alias will be removed in a future version.
type BuiltinMethod = registry.BuiltinMethod

// Deprecated: Use registry.BuiltinType instead.
// This alias will be removed in a future version.
type BuiltinType = registry.BuiltinType

// Deprecated: Use registry.BuiltinRegistry instead.
// This alias will be removed in a future version.
type BuiltinRegistry = registry.BuiltinRegistry

// NewBuiltinRegistry creates and initializes a registry with Python builtin types.
// Deprecated: Use registry.NewBuiltinRegistry instead.
func NewBuiltinRegistry() *BuiltinRegistry {
	return registry.NewBuiltinRegistry()
}
