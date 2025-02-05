package java

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseIfStatement(node *sitter.Node, sourceCode []byte, file string) *model.Node {
	ifNode := &model.IfStmt{}
	// get the condition of the if statement
	conditionNode := node.Child(1)
	if conditionNode != nil {
		ifNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
	}
	// get the then block of the if statement
	thenNode := node.Child(2)
	if thenNode != nil {
		ifNode.Then = model.Stmt{NodeString: thenNode.Content(sourceCode)}
	}
	// get the else block of the if statement
	elseNode := node.Child(4)
	if elseNode != nil {
		ifNode.Else = model.Stmt{NodeString: elseNode.Content(sourceCode)}
	}

	methodID := fmt.Sprintf("ifstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	ifStmtNode := &model.Node{
		ID:               util.GenerateSha256(methodID),
		Type:             "IfStmt",
		Name:             "IfStmt",
		IsExternal:       true,
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1,
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		IfStmt:           ifNode,
	}

	return ifStmtNode
}
