package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// BuildModuleRegistry is a convenience wrapper.
// Deprecated: Use registry.BuildModuleRegistry instead.
func BuildModuleRegistry(rootPath string) (*core.ModuleRegistry, error) {
	return registry.BuildModuleRegistry(rootPath)
}
