package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseBreakStatement(node *sitter.Node, sourceCode []byte, file string) *model.BreakStmt {
	breakStmt := &model.BreakStmt{}
	// get identifier if present child
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "identifier" {
			breakStmt.Label = node.Child(i).Content(sourceCode)
		}
	}
	// uniquebreakstmtID := fmt.Sprintf("breakstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return breakStmt
}

func ParseContinueStatement(node *sitter.Node, sourceCode []byte, file string) *model.ContinueStmt {
	continueStmt := &model.ContinueStmt{}
	// get identifier if present child
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "identifier" {
			continueStmt.Label = node.Child(i).Content(sourceCode)
		}
	}
	// uniquecontinueID := fmt.Sprintf("continuestmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return continueStmt
}

func ParseYieldStatement(node *sitter.Node, sourceCode []byte, file string) *model.YieldStmt {
	yieldStmt := &model.YieldStmt{}
	yieldStmtExpr := &model.Expr{NodeString: node.Child(1).Content(sourceCode)}
	yieldStmt.Value = yieldStmtExpr
	//uniqueyieldID := fmt.Sprintf("yield_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return yieldStmt
}

func ParseAssertStatement(node *sitter.Node, sourceCode []byte, file string) *model.AssertStmt {
	assertStmt := &model.AssertStmt{}
	assertStmt.Expr = &model.Expr{NodeString: node.Child(1).Content(sourceCode)}
	if node.Child(3) != nil && node.Child(3).Type() == "string_literal" {
		assertStmt.Message = &model.Expr{NodeString: node.Child(3).Content(sourceCode)}
	}

	//niqueAssertID := fmt.Sprintf("assert_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return assertStmt
}

func ParseReturnStatement(node *sitter.Node, sourceCode []byte, file string) *model.ReturnStmt {
	returnStmt := &model.ReturnStmt{}
	if node.Child(1) != nil {
		returnStmt.Result = &model.Expr{NodeString: node.Child(1).Content(sourceCode)}
	}
	// uniqueReturnID := fmt.Sprintf("return_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return returnStmt
}

func ParseBlockStatement(node *sitter.Node, sourceCode []byte, file string) *model.BlockStmt {
	blockStmt := &model.BlockStmt{}
	for i := 0; i < int(node.ChildCount()); i++ {
		singleBlockStmt := &model.Stmt{}
		singleBlockStmt.NodeString = node.Child(i).Content(sourceCode)
		blockStmt.Stmts = append(blockStmt.Stmts, *singleBlockStmt)
	}

	// uniqueBlockID := fmt.Sprintf("block_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return blockStmt
}

func ParseWhileStatement(node *sitter.Node, sourceCode []byte, file string) *model.WhileStmt {
	whileNode := &model.WhileStmt{}
	// get the condition of the while statement
	conditionNode := node.Child(1)
	if conditionNode != nil {
		whileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	// methodID := fmt.Sprintf("while_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	return whileNode
}

func ParseDoWhileStatement(node *sitter.Node, sourceCode []byte, file string) *model.DoStmt {
	doWhileNode := &model.DoStmt{}
	// get the condition of the while statement
	conditionNode := node.Child(2)
	if conditionNode != nil {
		doWhileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	// methodID := fmt.Sprintf("dowhile_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	return doWhileNode
}

func ParseForLoopStatement(node *sitter.Node, sourceCode []byte, file string) *model.ForStmt {
	forNode := &model.ForStmt{}
	// get the condition of the while statement
	initNode := node.ChildByFieldName("init")
	if initNode != nil {
		forNode.Init = &model.Expr{Node: *initNode, NodeString: initNode.Content(sourceCode)}
	}
	conditionNode := node.ChildByFieldName("condition")
	if conditionNode != nil {
		forNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	incrementNode := node.ChildByFieldName("increment")
	if incrementNode != nil {
		forNode.Increment = &model.Expr{Node: *incrementNode, NodeString: incrementNode.Content(sourceCode)}
	}

	// methodID := fmt.Sprintf("for_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	return forNode
}
