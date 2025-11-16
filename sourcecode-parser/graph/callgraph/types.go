package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// Deprecated: Use core.Location instead.
// This alias will be removed in a future version.
type Location = core.Location

// Deprecated: Use core.CallSite instead.
// This alias will be removed in a future version.
type CallSite = core.CallSite

// Deprecated: Use core.Argument instead.
// This alias will be removed in a future version.
type Argument = core.Argument

// Deprecated: Use core.CallGraph instead.
// This alias will be removed in a future version.
type CallGraph = core.CallGraph

// Deprecated: Use core.ModuleRegistry instead.
// This alias will be removed in a future version.
type ModuleRegistry = core.ModuleRegistry

// Deprecated: Use core.ImportMap instead.
// This alias will be removed in a future version.
type ImportMap = core.ImportMap

// NewCallGraph is a convenience wrapper.
// Deprecated: Use core.NewCallGraph instead.
func NewCallGraph() *CallGraph {
	return core.NewCallGraph()
}

// NewModuleRegistry is a convenience wrapper.
// Deprecated: Use core.NewModuleRegistry instead.
func NewModuleRegistry() *ModuleRegistry {
	return core.NewModuleRegistry()
}

// NewImportMap is a convenience wrapper.
// Deprecated: Use core.NewImportMap instead.
func NewImportMap(filePath string) *ImportMap {
	return core.NewImportMap(filePath)
}

// Helper functions for internal use within callgraph package
// These are kept here for backward compatibility with other files in the package

// contains checks if a string slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// extractShortName extracts the last component of a dotted path.
// Example: "myapp.utils.helpers" → "helpers".
func extractShortName(modulePath string) string {
	// Find last dot
	for i := len(modulePath) - 1; i >= 0; i-- {
		if modulePath[i] == '.' {
			return modulePath[i+1:]
		}
	}
	return modulePath
}
