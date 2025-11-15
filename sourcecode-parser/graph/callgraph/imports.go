package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ExtractImports extracts all import statements from a Python file and builds an ImportMap.
// Deprecated: Use resolution.ExtractImports instead.
func ExtractImports(filePath string, sourceCode []byte, registry *core.ModuleRegistry) (*core.ImportMap, error) {
	return resolution.ExtractImports(filePath, sourceCode, registry)
}
