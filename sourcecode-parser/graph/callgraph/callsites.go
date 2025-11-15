package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ExtractCallSites extracts all function/method call sites from a Python file.
// Deprecated: Use resolution.ExtractCallSites instead.
func ExtractCallSites(filePath string, sourceCode []byte, importMap *core.ImportMap) ([]*core.CallSite, error) {
	return resolution.ExtractCallSites(filePath, sourceCode, importMap)
}
