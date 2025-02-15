package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseExpr(node *sitter.Node, sourceCode []byte, file string, parentNode *model.TreeNode) *model.BinaryExpr {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	operator := node.ChildByFieldName("operator")
	operatorType := operator.Type()
	expressionNode := &model.BinaryExpr{}
	expressionNode.LeftOperand = &model.Expr{Node: *leftNode, NodeString: leftNode.Content(sourceCode)}
	expressionNode.RightOperand = &model.Expr{Node: *rightNode, NodeString: rightNode.Content(sourceCode)}
	expressionNode.Op = operatorType

	binaryExprNode := &model.TreeNode{Node: &model.Node{BinaryExpr: expressionNode}, Parent: parentNode}
	parentNode.AddChild(binaryExprNode)

	switch operatorType {
	case "+":
		var addExpr model.AddExpr
		addExpr.LeftOperand = expressionNode.LeftOperand
		addExpr.RightOperand = expressionNode.RightOperand
		addExpr.Op = expressionNode.Op
		addExpr.BinaryExpr = *expressionNode
		addExpressionNode := &model.Node{
			AddExpr: &addExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: addExpressionNode, Parent: parentNode})
	case "-":
		var subExpr model.SubExpr
		subExpr.LeftOperand = expressionNode.LeftOperand
		subExpr.RightOperand = expressionNode.RightOperand
		subExpr.Op = expressionNode.Op
		subExpr.BinaryExpr = *expressionNode
		subExpressionNode := &model.Node{
			SubExpr: &subExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: subExpressionNode, Parent: parentNode})
	case "*":
		var mulExpr model.MulExpr
		mulExpr.LeftOperand = expressionNode.LeftOperand
		mulExpr.RightOperand = expressionNode.RightOperand
		mulExpr.Op = expressionNode.Op
		mulExpr.BinaryExpr = *expressionNode
		mulExpressionNode := &model.Node{
			MulExpr: &mulExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: mulExpressionNode, Parent: parentNode})
	case "/":
		var divExpr model.DivExpr
		divExpr.LeftOperand = expressionNode.LeftOperand
		divExpr.RightOperand = expressionNode.RightOperand
		divExpr.Op = expressionNode.Op
		divExpr.BinaryExpr = *expressionNode
		divExpressionNode := &model.Node{
			DivExpr: &divExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: divExpressionNode, Parent: parentNode})
	case ">", "<", ">=", "<=":
		var compExpr model.ComparisonExpr
		compExpr.LeftOperand = expressionNode.LeftOperand
		compExpr.RightOperand = expressionNode.RightOperand
		compExpr.Op = expressionNode.Op
		compExpr.BinaryExpr = *expressionNode
		compExpressionNode := &model.Node{
			ComparisonExpr: &compExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: compExpressionNode, Parent: parentNode})
	case "%":
		var remExpr model.RemExpr
		remExpr.LeftOperand = expressionNode.LeftOperand
		remExpr.RightOperand = expressionNode.RightOperand
		remExpr.Op = expressionNode.Op
		remExpr.BinaryExpr = *expressionNode
		RemExpressionNode := &model.Node{
			RemExpr: &remExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: RemExpressionNode, Parent: parentNode})
	case ">>":
		var rightShiftExpr model.RightShiftExpr
		rightShiftExpr.LeftOperand = expressionNode.LeftOperand
		rightShiftExpr.RightOperand = expressionNode.RightOperand
		rightShiftExpr.Op = expressionNode.Op
		rightShiftExpr.BinaryExpr = *expressionNode
		RightShiftExpressionNode := &model.Node{
			RightShiftExpr: &rightShiftExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: RightShiftExpressionNode, Parent: parentNode})
	case "<<":
		var LeftShiftExpr model.LeftShiftExpr
		LeftShiftExpr.LeftOperand = expressionNode.LeftOperand
		LeftShiftExpr.RightOperand = expressionNode.RightOperand
		LeftShiftExpr.Op = expressionNode.Op
		LeftShiftExpr.BinaryExpr = *expressionNode
		LeftShiftExpressionNode := &model.Node{
			LeftShiftExpr: &LeftShiftExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: LeftShiftExpressionNode, Parent: parentNode})
	case "!=":
		var neExpr model.NEExpr
		neExpr.LeftOperand = expressionNode.LeftOperand
		neExpr.RightOperand = expressionNode.RightOperand
		neExpr.Op = expressionNode.Op
		neExpr.BinaryExpr = *expressionNode
		NEExpressionNode := &model.Node{
			NEExpr: &neExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: NEExpressionNode, Parent: parentNode})
	case "==":
		var EQExpr model.EqExpr
		EQExpr.LeftOperand = expressionNode.LeftOperand
		EQExpr.RightOperand = expressionNode.RightOperand
		EQExpr.Op = expressionNode.Op
		EQExpr.BinaryExpr = *expressionNode
		EQExpressionNode := &model.Node{
			EQExpr: &EQExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: EQExpressionNode, Parent: parentNode})
	case "&":
		var bitWiseAndExpr model.AndBitwiseExpr
		bitWiseAndExpr.LeftOperand = expressionNode.LeftOperand
		bitWiseAndExpr.RightOperand = expressionNode.RightOperand
		bitWiseAndExpr.Op = expressionNode.Op
		bitWiseAndExpr.BinaryExpr = *expressionNode
		BitwiseAndExpressionNode := &model.Node{
			AndBitwiseExpr: &bitWiseAndExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: BitwiseAndExpressionNode, Parent: parentNode})
	case "&&":
		var andExpr model.AndLogicalExpr
		andExpr.LeftOperand = expressionNode.LeftOperand
		andExpr.RightOperand = expressionNode.RightOperand
		andExpr.Op = expressionNode.Op
		andExpr.BinaryExpr = *expressionNode
		AndExpressionNode := &model.Node{
			AndLogicalExpr: &andExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: AndExpressionNode, Parent: parentNode})
	case "||":
		var OrExpr model.OrLogicalExpr
		OrExpr.LeftOperand = expressionNode.LeftOperand
		OrExpr.RightOperand = expressionNode.RightOperand
		OrExpr.Op = expressionNode.Op
		OrExpr.BinaryExpr = *expressionNode
		OrExpressionNode := &model.Node{
			OrLogicalExpr: &OrExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: OrExpressionNode, Parent: parentNode})
	case "|":
		var BitwiseOrExpr model.OrBitwiseExpr
		BitwiseOrExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseOrExpr.RightOperand = expressionNode.RightOperand
		BitwiseOrExpr.Op = expressionNode.Op
		BitwiseOrExpr.BinaryExpr = *expressionNode
		BitwiseOrExpressionNode := &model.Node{
			BinaryExpr: expressionNode,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: BitwiseOrExpressionNode, Parent: parentNode})
	case ">>>":
		var BitwiseRightShiftExpr model.UnsignedRightShiftExpr
		BitwiseRightShiftExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseRightShiftExpr.RightOperand = expressionNode.RightOperand
		BitwiseRightShiftExpr.Op = expressionNode.Op
		BitwiseRightShiftExpr.BinaryExpr = *expressionNode
		BitwiseRightShiftExpressionNode := &model.Node{
			UnsignedRightShiftExpr: &BitwiseRightShiftExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: BitwiseRightShiftExpressionNode, Parent: parentNode})
	case "^":
		var BitwiseXorExpr model.XorBitwiseExpr
		BitwiseXorExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseXorExpr.RightOperand = expressionNode.RightOperand
		BitwiseXorExpr.Op = expressionNode.Op
		BitwiseXorExpr.BinaryExpr = *expressionNode
		BitwiseXorExpressionNode := &model.Node{
			XorBitwiseExpr: &BitwiseXorExpr,
		}
		binaryExprNode.AddChild(&model.TreeNode{Node: BitwiseXorExpressionNode, Parent: parentNode})
	}

	return expressionNode
}
