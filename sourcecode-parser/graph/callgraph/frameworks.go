package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// Deprecated: Use core.FrameworkDefinition instead.
// This alias will be removed in a future version.
type FrameworkDefinition = core.FrameworkDefinition

// LoadFrameworks is a convenience wrapper.
// Deprecated: Use core.LoadFrameworks instead.
func LoadFrameworks() []FrameworkDefinition {
	return core.LoadFrameworks()
}

// IsKnownFramework is a convenience wrapper.
// Deprecated: Use core.IsKnownFramework instead.
func IsKnownFramework(fqn string) (bool, *FrameworkDefinition) {
	return core.IsKnownFramework(fqn)
}

// GetFrameworkCategory is a convenience wrapper.
// Deprecated: Use core.GetFrameworkCategory instead.
func GetFrameworkCategory(fqn string) string {
	return core.GetFrameworkCategory(fqn)
}

// GetFrameworkName is a convenience wrapper.
// Deprecated: Use core.GetFrameworkName instead.
func GetFrameworkName(fqn string) string {
	return core.GetFrameworkName(fqn)
}
