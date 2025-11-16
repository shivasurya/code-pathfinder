package callgraph

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/extraction"
)

// ExtractStatements extracts all statements from a Python function body.
// Deprecated: Use extraction.ExtractStatements instead.
func ExtractStatements(filePath string, sourceCode []byte, functionNode *sitter.Node) ([]*core.Statement, error) {
	return extraction.ExtractStatements(filePath, sourceCode, functionNode)
}

// ParsePythonFile parses a Python source file using tree-sitter.
// Deprecated: Use extraction.ParsePythonFile instead.
func ParsePythonFile(sourceCode []byte) (*sitter.Tree, error) {
	return extraction.ParsePythonFile(sourceCode)
}
