package java

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/model"
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

func ParseAssertStatement(node *sitter.Node, sourcecode []byte) *model.AssertStmt {
	assertStmt := &model.AssertStmt{}
	assertStmt.Expr = &model.Expr{NodeString: node.Child(1).Content(sourcecode)}
	if node.Child(3) != nil && node.Child(3).Type() == "string_literal" {
		assertStmt.Message = &model.Expr{NodeString: node.Child(3).Content(sourcecode)}
	}
	return assertStmt
}

func ParseReturnStatement(node *sitter.Node, sourcecode []byte) *model.ReturnStmt {
	returnStmt := &model.ReturnStmt{}
	if node.Child(1) != nil {
		returnStmt.Result = &model.Expr{NodeString: node.Child(1).Content(sourcecode)}
	}
	return returnStmt
}

func ParseBlockStatement(node *sitter.Node, sourcecode []byte) *model.BlockStmt {
	blockStmt := &model.BlockStmt{}
	for i := 0; i < int(node.ChildCount()); i++ {
		singleBlockStmt := &model.Stmt{}
		singleBlockStmt.NodeString = node.Child(i).Content(sourcecode)
		blockStmt.Stmts = append(blockStmt.Stmts, *singleBlockStmt)
	}
	return blockStmt
}
