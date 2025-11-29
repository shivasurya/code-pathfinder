package python

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseReturnStatement(node *sitter.Node, sourcecode []byte) *model.ReturnStmt {
	returnStmt := &model.ReturnStmt{}
	// Python return statements can have 0 or more return values
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "return" {
			returnStmt.Result = &model.Expr{NodeString: child.Content(sourcecode)}
			break
		}
	}
	return returnStmt
}

func ParseBreakStatement(node *sitter.Node, sourcecode []byte) *model.BreakStmt {
	breakStmt := &model.BreakStmt{}
	// Python break statements don't have labels
	return breakStmt
}

func ParseContinueStatement(node *sitter.Node, sourcecode []byte) *model.ContinueStmt {
	continueStmt := &model.ContinueStmt{}
	// Python continue statements don't have labels
	return continueStmt
}

func ParseAssertStatement(node *sitter.Node, sourcecode []byte) *model.AssertStmt {
	assertStmt := &model.AssertStmt{}
	// Python assert has condition and optional message
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "assert" && child.Type() != "," {
			if assertStmt.Expr == nil {
				assertStmt.Expr = &model.Expr{NodeString: child.Content(sourcecode)}
			} else if assertStmt.Message == nil {
				assertStmt.Message = &model.Expr{NodeString: child.Content(sourcecode)}
			}
		}
	}
	return assertStmt
}

func ParseBlockStatement(node *sitter.Node, sourcecode []byte) *model.BlockStmt {
	blockStmt := &model.BlockStmt{}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		singleBlockStmt := &model.Stmt{}
		singleBlockStmt.NodeString = child.Content(sourcecode)
		blockStmt.Stmts = append(blockStmt.Stmts, *singleBlockStmt)
	}
	return blockStmt
}

func ParseYieldStatement(node *sitter.Node, sourcecode []byte) *model.YieldStmt {
	yieldStmt := &model.YieldStmt{}
	// Python yield can be in yield_expression
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "yield" && child.Type() != "from" {
			yieldStmt.Value = &model.Expr{NodeString: child.Content(sourcecode)}
			break
		}
	}
	return yieldStmt
}
