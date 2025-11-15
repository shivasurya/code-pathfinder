package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// Deprecated: Use core.ClassAttribute instead.
// This alias will be removed in a future version.
type ClassAttribute = core.ClassAttribute

// Deprecated: Use core.ClassAttributes instead.
// This alias will be removed in a future version.
type ClassAttributes = core.ClassAttributes

// Deprecated: Use registry.AttributeRegistry instead.
// This alias will be removed in a future version.
type AttributeRegistry = registry.AttributeRegistry

// NewAttributeRegistry creates a new empty AttributeRegistry.
// Deprecated: Use registry.NewAttributeRegistry instead.
func NewAttributeRegistry() *AttributeRegistry {
	return registry.NewAttributeRegistry()
}
