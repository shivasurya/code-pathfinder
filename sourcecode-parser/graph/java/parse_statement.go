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

func ParseContinueStatement(node *sitter.Node, sourcecode []byte) *model.ContinueStmt {
	continueStmt := &model.ContinueStmt{}
	// get identifier if present child
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "identifier" {
			continueStmt.Label = node.Child(i).Content(sourcecode)
		}
	}
	return continueStmt
}

func ParseYieldStatement(node *sitter.Node, sourcecode []byte) *model.YieldStmt {
	yieldStmt := &model.YieldStmt{}
	yieldStmtExpr := &model.Expr{NodeString: node.Child(1).Content(sourcecode)}
	yieldStmt.Value = yieldStmtExpr
	return yieldStmt
}
