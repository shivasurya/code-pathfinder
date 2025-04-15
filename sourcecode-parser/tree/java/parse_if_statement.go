package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseIfStatement(node *sitter.Node, sourceCode []byte) *model.IfStmt {
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

	// methodID := fmt.Sprintf("ifstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	// add node to graph
	return ifNode
}
