package java

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseBreakStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	breakStmt := &model.BreakStmt{}
	// get identifier if present child
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "identifier" {
			breakStmt.Label = node.Child(i).Content(sourceCode)
		}
	}
	uniquebreakstmtID := fmt.Sprintf("breakstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	breakStmtNode := &model.Node{
		ID:               util.GenerateSha256(uniquebreakstmtID),
		Type:             "BreakStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "BreakStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		BreakStmt:        breakStmt,
	}
	return breakStmtNode
}

func ParseContinueStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	continueStmt := &model.ContinueStmt{}
	// get identifier if present child
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.Child(i).Type() == "identifier" {
			continueStmt.Label = node.Child(i).Content(sourceCode)
		}
	}
	uniquecontinueID := fmt.Sprintf("continuestmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	continueStmtNode := &model.Node{
		ID:               util.GenerateSha256(uniquecontinueID),
		Type:             "ContinueStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "ContinueStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		ContinueStmt:     continueStmt,
	}
	return continueStmtNode
}

func ParseYieldStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	yieldStmt := &model.YieldStmt{}
	yieldStmtExpr := &model.Expr{NodeString: node.Child(1).Content(sourceCode)}
	yieldStmt.Value = yieldStmtExpr
	uniqueyieldID := fmt.Sprintf("yield_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	yieldStmtNode := &model.Node{
		ID:               util.GenerateSha256(uniqueyieldID),
		Type:             "YieldStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "YieldStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		YieldStmt:        yieldStmt,
	}
	return yieldStmtNode
}

func ParseAssertStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	assertStmt := &model.AssertStmt{}
	assertStmt.Expr = &model.Expr{NodeString: node.Child(1).Content(sourceCode)}
	if node.Child(3) != nil && node.Child(3).Type() == "string_literal" {
		assertStmt.Message = &model.Expr{NodeString: node.Child(3).Content(sourceCode)}
	}

	uniqueAssertID := fmt.Sprintf("assert_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	assertStmtNode := &model.Node{
		ID:               util.GenerateSha256(uniqueAssertID),
		Type:             "AssertStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "AssertStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		AssertStmt:       assertStmt,
	}
	return assertStmtNode
}

func ParseReturnStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	returnStmt := &model.ReturnStmt{}
	if node.Child(1) != nil {
		returnStmt.Result = &model.Expr{NodeString: node.Child(1).Content(sourceCode)}
	}
	uniqueReturnID := fmt.Sprintf("return_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	returnStmtNode := &model.Node{
		ID:               util.GenerateSha256(uniqueReturnID),
		Type:             "ReturnStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "ReturnStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		ReturnStmt:       returnStmt,
	}
	return returnStmtNode
}

func ParseBlockStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	blockStmt := &model.BlockStmt{}
	for i := 0; i < int(node.ChildCount()); i++ {
		singleBlockStmt := &model.Stmt{}
		singleBlockStmt.NodeString = node.Child(i).Content(sourceCode)
		blockStmt.Stmts = append(blockStmt.Stmts, *singleBlockStmt)
	}

	uniqueBlockID := fmt.Sprintf("block_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	blockStmtNode := &model.Node{
		ID:               util.GenerateSha256(uniqueBlockID),
		Type:             "BlockStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "BlockStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		BlockStmt:        blockStmt,
	}
	return blockStmtNode
}

func ParseWhileStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	whileNode := model.WhileStmt{}
	// get the condition of the while statement
	conditionNode := node.Child(1)
	if conditionNode != nil {
		whileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	methodID := fmt.Sprintf("while_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	whileStmtNode := &model.Node{
		ID:               util.GenerateSha256(methodID),
		Type:             "WhileStmt",
		Name:             "WhileStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		WhileStmt:        &whileNode,
	}
	return whileStmtNode
}

func ParseDoWhileStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	doWhileNode := model.DoStmt{}
	// get the condition of the while statement
	conditionNode := node.Child(2)
	if conditionNode != nil {
		doWhileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	methodID := fmt.Sprintf("dowhile_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	doWhileStmtNode := &model.Node{
		ID:               util.GenerateSha256(methodID),
		Type:             "DoStmt",
		Name:             "DoStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		DoStmt:           &doWhileNode,
	}
	return doWhileStmtNode
}

func ParseForLoopStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	forNode := model.ForStmt{}
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

	methodID := fmt.Sprintf("for_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	forStmtNode := &model.Node{
		ID:               util.GenerateSha256(methodID),
		Type:             "ForStmt",
		Name:             "ForStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		ForStmt:          &forNode,
	}
	return forStmtNode
}
