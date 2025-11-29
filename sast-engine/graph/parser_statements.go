package graph

import (
	"fmt"

	javalang "github.com/shivasurya/code-pathfinder/sast-engine/graph/java"
	"github.com/shivasurya/code-pathfinder/sast-engine/model"
	sitter "github.com/smacker/go-tree-sitter"
)

// parseBlockStatement parses block statements.
func parseBlockStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) {
	blockNode := javalang.ParseBlockStatement(node, sourceCode)
	uniqueBlockID := fmt.Sprintf("block_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	blockStmtNode := &Node{
		ID:               GenerateSha256(uniqueBlockID),
		Type:             "BlockStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "BlockStmt",
		IsExternal:       true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		BlockStmt:        blockNode,
	}
	graph.AddNode(blockStmtNode)
}

// parseReturnStatement parses return statements (Java or Python).
func parseReturnStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJava, isPython bool) {
	if isPython {
		parsePythonReturnStatement(node, sourceCode, graph, file)
	} else if isJava {
		returnNode := javalang.ParseReturnStatement(node, sourceCode)
		uniqueReturnID := fmt.Sprintf("return_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		returnStmtNode := &Node{
			ID:               GenerateSha256(uniqueReturnID),
			Type:             "ReturnStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "ReturnStmt",
			IsExternal:       true,
			SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
			File:             file,
			isJavaSourceFile: isJava,
			ReturnStmt:       returnNode,
		}
		graph.AddNode(returnStmtNode)
	}
}

// parseBreakStatement parses break statements (Java or Python).
func parseBreakStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJava, isPython bool) {
	if isPython {
		parsePythonBreakStatement(node, sourceCode, graph, file)
	} else if isJava {
		breakNode := javalang.ParseBreakStatement(node, sourceCode)
		uniquebreakstmtID := fmt.Sprintf("breakstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		breakStmtNode := &Node{
			ID:               GenerateSha256(uniquebreakstmtID),
			Type:             "BreakStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "BreakStmt",
			IsExternal:       true,
			SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
			File:             file,
			isJavaSourceFile: isJava,
			BreakStmt:        breakNode,
		}
		graph.AddNode(breakStmtNode)
	}
}

// parseContinueStatement parses continue statements (Java or Python).
func parseContinueStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJava, isPython bool) {
	if isPython {
		parsePythonContinueStatement(node, sourceCode, graph, file)
	} else if isJava {
		continueNode := javalang.ParseContinueStatement(node, sourceCode)
		uniquecontinueID := fmt.Sprintf("continuestmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		continueStmtNode := &Node{
			ID:               GenerateSha256(uniquecontinueID),
			Type:             "ContinueStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "ContinueStmt",
			IsExternal:       true,
			SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
			File:             file,
			isJavaSourceFile: isJava,
			ContinueStmt:     continueNode,
		}
		graph.AddNode(continueStmtNode)
	}
}

// parseAssertStatement parses assert statements (Java or Python).
func parseAssertStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJava, isPython bool) {
	if isPython {
		parsePythonAssertStatement(node, sourceCode, graph, file)
	} else if isJava {
		assertNode := javalang.ParseAssertStatement(node, sourceCode)
		uniqueAssertID := fmt.Sprintf("assert_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		assertStmtNode := &Node{
			ID:               GenerateSha256(uniqueAssertID),
			Type:             "AssertStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "AssertStmt",
			IsExternal:       true,
			SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
			File:             file,
			isJavaSourceFile: isJava,
			AssertStmt:       assertNode,
		}
		graph.AddNode(assertStmtNode)
	}
}

// parseYieldStatement parses yield statements (Java only).
func parseYieldStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) {
	yieldNode := javalang.ParseYieldStatement(node, sourceCode)
	uniqueyieldID := fmt.Sprintf("yield_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	yieldStmtNode := &Node{
		ID:               GenerateSha256(uniqueyieldID),
		Type:             "YieldStmt",
		LineNumber:       node.StartPoint().Row + 1,
		Name:             "YieldStmt",
		IsExternal:       true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		YieldStmt:        yieldNode,
	}
	graph.AddNode(yieldStmtNode)
}

// parseIfStatement parses if statements.
func parseIfStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) {
	ifNode := model.IfStmt{}
	conditionNode := node.Child(1)
	if conditionNode != nil {
		ifNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	thenNode := node.Child(2)
	if thenNode != nil {
		ifNode.Then = model.Stmt{NodeString: thenNode.Content(sourceCode)}
	}
	elseNode := node.Child(4)
	if elseNode != nil {
		ifNode.Else = model.Stmt{NodeString: elseNode.Content(sourceCode)}
	}

	methodID := fmt.Sprintf("ifstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	ifStmtNode := &Node{
		ID:               GenerateSha256(methodID),
		Type:             "IfStmt",
		Name:             "IfStmt",
		IsExternal:       true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		IfStmt:           &ifNode,
	}
	graph.AddNode(ifStmtNode)
}

// parseWhileStatement parses while statements.
func parseWhileStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) {
	whileNode := model.WhileStmt{}
	conditionNode := node.Child(1)
	if conditionNode != nil {
		whileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	methodID := fmt.Sprintf("while_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	whileStmtNode := &Node{
		ID:               GenerateSha256(methodID),
		Type:             "WhileStmt",
		Name:             "WhileStmt",
		IsExternal:       true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		WhileStmt:        &whileNode,
	}
	graph.AddNode(whileStmtNode)
}

// parseDoStatement parses do-while statements.
func parseDoStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) {
	doWhileNode := model.DoStmt{}
	conditionNode := node.Child(2)
	if conditionNode != nil {
		doWhileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	methodID := fmt.Sprintf("dowhile_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	doWhileStmtNode := &Node{
		ID:               GenerateSha256(methodID),
		Type:             "DoStmt",
		Name:             "DoStmt",
		IsExternal:       true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		DoStmt:           &doWhileNode,
	}
	graph.AddNode(doWhileStmtNode)
}

// parseForStatement parses for statements.
func parseForStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isJavaSourceFile bool) {
	forNode := model.ForStmt{}
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
	forStmtNode := &Node{
		ID:               GenerateSha256(methodID),
		Type:             "ForStmt",
		Name:             "ForStmt",
		IsExternal:       true,
		SourceLocation: &SourceLocation{File: file, StartByte: node.StartByte(), EndByte: node.EndByte()},
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		isJavaSourceFile: isJavaSourceFile,
		ForStmt:          &forNode,
	}
	graph.AddNode(forStmtNode)
}
