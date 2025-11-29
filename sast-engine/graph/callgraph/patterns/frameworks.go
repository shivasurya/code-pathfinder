package patterns

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Framework represents a detected framework.
type Framework struct {
	Name     string
	Version  string
	Category string
}

// DetectFramework detects which framework is used based on imports.
// Returns the first detected framework or nil if none found.
func DetectFramework(importMap *core.ImportMap) *Framework {
	if importMap == nil {
		return nil
	}

	// Check for known frameworks using the core framework definitions
	// Iterate over FQNs (values), not aliases (keys)
	for _, fqn := range importMap.Imports {
		if isKnown, framework := core.IsKnownFramework(fqn); isKnown {
			return &Framework{
				Name:     framework.Name,
				Category: framework.Category,
			}
		}
	}

	return nil
}

// IsKnownFramework checks if import path is a known framework.
// This is a convenience wrapper around core.IsKnownFramework.
func IsKnownFramework(importPath string) bool {
	isKnown, _ := core.IsKnownFramework(importPath)
	return isKnown
}

// GetFrameworkCategory returns the category of a framework given its import path.
// Returns empty string if not a known framework.
func GetFrameworkCategory(importPath string) string {
	return core.GetFrameworkCategory(importPath)
}

// GetFrameworkName returns the name of a framework given its import path.
// Returns empty string if not a known framework.
func GetFrameworkName(importPath string) string {
	return core.GetFrameworkName(importPath)
}
