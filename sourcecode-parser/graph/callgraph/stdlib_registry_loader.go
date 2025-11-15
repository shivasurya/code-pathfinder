package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// Deprecated: Use registry.StdlibRegistryLoader instead.
// This alias will be removed in a future version.
type StdlibRegistryLoader = registry.StdlibRegistryLoader

// NewStdlibRegistryLoader creates a new stdlib registry loader.
// Deprecated: Use registry.StdlibRegistryLoader directly.
func NewStdlibRegistryLoader(registryPath string) *StdlibRegistryLoader {
	return &registry.StdlibRegistryLoader{
		RegistryPath: registryPath,
	}
}
