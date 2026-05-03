package extraction

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
)

// ExtractCStatements walks a C function body and produces one
// *core.Statement per recognised top-level construct (declaration,
// expression, return, if/for/while/do/switch). The result feeds the
// CFG builder (PR-10) and the future variable-dependency graph.
//
// Forward declarations and prototypes (no body) yield (nil, nil) — the
// caller can iterate without nil checks.
//
// The function is a thin wrapper around the shared clikeExtractor; C
// and C++ share every dispatcher except for the keyword filter and a
// handful of C++-only AST nodes (`throw_statement`, `try_statement`,
// `for_range_loop`).
func ExtractCStatements(filePath string, sourceCode []byte, functionNode *sitter.Node) ([]*core.Statement, error) {
	if functionNode == nil {
		return nil, nil
	}
	e := &clikeExtractor{
		filePath:  filePath,
		src:       sourceCode,
		isKeyword: clike.IsCKeyword,
	}
	return e.extractFunctionBody(functionNode), nil
}
