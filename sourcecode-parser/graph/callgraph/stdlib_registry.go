package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// Deprecated: Use core.StdlibRegistry instead.
// This alias will be removed in a future version.
type StdlibRegistry = core.StdlibRegistry

// Deprecated: Use core.Manifest instead.
// This alias will be removed in a future version.
type Manifest = core.Manifest

// Deprecated: Use core.PythonVersionInfo instead.
// This alias will be removed in a future version.
type PythonVersionInfo = core.PythonVersionInfo

// Deprecated: Use core.ModuleEntry instead.
// This alias will be removed in a future version.
type ModuleEntry = core.ModuleEntry

// Deprecated: Use core.RegistryStats instead.
// This alias will be removed in a future version.
type RegistryStats = core.RegistryStats

// Deprecated: Use core.StdlibModule instead.
// This alias will be removed in a future version.
type StdlibModule = core.StdlibModule

// Deprecated: Use core.StdlibFunction instead.
// This alias will be removed in a future version.
type StdlibFunction = core.StdlibFunction

// Deprecated: Use core.FunctionParam instead.
// This alias will be removed in a future version.
type FunctionParam = core.FunctionParam

// Deprecated: Use core.StdlibClass instead.
// This alias will be removed in a future version.
type StdlibClass = core.StdlibClass

// Deprecated: Use core.StdlibConstant instead.
// This alias will be removed in a future version.
type StdlibConstant = core.StdlibConstant

// Deprecated: Use core.StdlibAttribute instead.
// This alias will be removed in a future version.
type StdlibAttribute = core.StdlibAttribute

// NewStdlibRegistry is a convenience wrapper.
// Deprecated: Use core.NewStdlibRegistry instead.
func NewStdlibRegistry() *StdlibRegistry {
	return core.NewStdlibRegistry()
}
