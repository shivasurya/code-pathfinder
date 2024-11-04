package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseBreakStatement(node *sitter.Node, sourcecode []byte) *model.BreakStmt {
	breakStmt := &model.BreakStmt{}
	// get identifier if present child
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "identifier" {
			breakStmt.Label = node.Child(i).Content(sourcecode)
		}
	}
	return breakStmt
}
