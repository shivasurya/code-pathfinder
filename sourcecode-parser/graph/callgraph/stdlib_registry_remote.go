package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// Deprecated: Use registry.StdlibRegistryRemote instead.
// This alias will be removed in a future version.
type StdlibRegistryRemote = registry.StdlibRegistryRemote

// NewStdlibRegistryRemote creates a new remote registry loader.
// Deprecated: Use registry.NewStdlibRegistryRemote instead.
func NewStdlibRegistryRemote(baseURL, pythonVersion string) *StdlibRegistryRemote {
	return registry.NewStdlibRegistryRemote(baseURL, pythonVersion)
}
