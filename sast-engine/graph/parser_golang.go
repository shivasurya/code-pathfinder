package graph

import (
	golangpkg "github.com/shivasurya/code-pathfinder/sast-engine/graph/golang"
	sitter "github.com/smacker/go-tree-sitter"
)

// Ensure golangpkg import is used. PR-03+ will add real dispatch functions.
var _ = golangpkg.DetermineVisibility

// setGoSourceLocation sets the SourceLocation on a Node from a tree-sitter node.
// Every Go node created by PR-03+ must call this to enable call graph construction,
// which uses StartByte/EndByte ranges to determine function containment.
func setGoSourceLocation(node *Node, tsNode *sitter.Node, file string) {
	node.SourceLocation = &SourceLocation{
		File:      file,
		StartByte: tsNode.StartByte(),
		EndByte:   tsNode.EndByte(),
	}
}
