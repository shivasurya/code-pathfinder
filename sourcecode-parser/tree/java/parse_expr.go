package java

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	util "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"
	sitter "github.com/smacker/go-tree-sitter"
)

func ParseExpr(node *sitter.Node, sourceCode []byte, file string, parentNode *model.TreeNode) *model.Node {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	operator := node.ChildByFieldName("operator")
	operatorType := operator.Type()
	expressionNode := model.BinaryExpr{}
	expressionNode.LeftOperand = &model.Expr{Node: *leftNode, NodeString: leftNode.Content(sourceCode)}
	expressionNode.RightOperand = &model.Expr{Node: *rightNode, NodeString: rightNode.Content(sourceCode)}
	expressionNode.Op = operatorType
	switch operatorType {
	case "+":
		var addExpr model.AddExpr
		addExpr.LeftOperand = expressionNode.LeftOperand
		addExpr.RightOperand = expressionNode.RightOperand
		addExpr.Op = expressionNode.Op
		addExpr.BinaryExpr = expressionNode
		addExpressionNode := &model.Node{
			ID:               util.GenerateSha256("add_expression" + node.Content(sourceCode)),
			Type:             "add_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: addExpressionNode, Parent: parentNode})
	case "-":
		var subExpr model.SubExpr
		subExpr.LeftOperand = expressionNode.LeftOperand
		subExpr.RightOperand = expressionNode.RightOperand
		subExpr.Op = expressionNode.Op
		subExpr.BinaryExpr = expressionNode
		subExpressionNode := &model.Node{
			ID:               util.GenerateSha256("sub_expression" + node.Content(sourceCode)),
			Type:             "sub_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: subExpressionNode, Parent: parentNode})
	case "*":
		var mulExpr model.MulExpr
		mulExpr.LeftOperand = expressionNode.LeftOperand
		mulExpr.RightOperand = expressionNode.RightOperand
		mulExpr.Op = expressionNode.Op
		mulExpr.BinaryExpr = expressionNode
		mulExpressionNode := &model.Node{
			ID:               util.GenerateSha256("mul_expression" + node.Content(sourceCode)),
			Type:             "mul_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: mulExpressionNode, Parent: parentNode})
	case "/":
		var divExpr model.DivExpr
		divExpr.LeftOperand = expressionNode.LeftOperand
		divExpr.RightOperand = expressionNode.RightOperand
		divExpr.Op = expressionNode.Op
		divExpr.BinaryExpr = expressionNode
		divExpressionNode := &model.Node{
			ID:               util.GenerateSha256("div_expression" + node.Content(sourceCode)),
			Type:             "div_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: divExpressionNode, Parent: parentNode})
	case ">", "<", ">=", "<=":
		var compExpr model.ComparisonExpr
		compExpr.LeftOperand = expressionNode.LeftOperand
		compExpr.RightOperand = expressionNode.RightOperand
		compExpr.Op = expressionNode.Op
		compExpr.BinaryExpr = expressionNode
		compExpressionNode := &model.Node{
			ID:               util.GenerateSha256("comp_expression" + node.Content(sourceCode)),
			Type:             "comp_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: compExpressionNode, Parent: parentNode})
	case "%":
		var RemExpr model.RemExpr
		RemExpr.LeftOperand = expressionNode.LeftOperand
		RemExpr.RightOperand = expressionNode.RightOperand
		RemExpr.Op = expressionNode.Op
		RemExpr.BinaryExpr = expressionNode
		RemExpressionNode := &model.Node{
			ID:               util.GenerateSha256("rem_expression" + node.Content(sourceCode)),
			Type:             "rem_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: RemExpressionNode, Parent: parentNode})
	case ">>":
		var RightShiftExpr model.RightShiftExpr
		RightShiftExpr.LeftOperand = expressionNode.LeftOperand
		RightShiftExpr.RightOperand = expressionNode.RightOperand
		RightShiftExpr.Op = expressionNode.Op
		RightShiftExpr.BinaryExpr = expressionNode
		RightShiftExpressionNode := &model.Node{
			ID:               util.GenerateSha256("right_shift_expression" + node.Content(sourceCode)),
			Type:             "right_shift_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: RightShiftExpressionNode, Parent: parentNode})
	case "<<":
		var LeftShiftExpr model.LeftShiftExpr
		LeftShiftExpr.LeftOperand = expressionNode.LeftOperand
		LeftShiftExpr.RightOperand = expressionNode.RightOperand
		LeftShiftExpr.Op = expressionNode.Op
		LeftShiftExpr.BinaryExpr = expressionNode
		LeftShiftExpressionNode := &model.Node{
			ID:               util.GenerateSha256("left_shift_expression" + node.Content(sourceCode)),
			Type:             "left_shift_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: LeftShiftExpressionNode, Parent: parentNode})
	case "!=":
		var NEExpr model.NEExpr
		NEExpr.LeftOperand = expressionNode.LeftOperand
		NEExpr.RightOperand = expressionNode.RightOperand
		NEExpr.Op = expressionNode.Op
		NEExpr.BinaryExpr = expressionNode
		NEExpressionNode := &model.Node{
			ID:               util.GenerateSha256("ne_expression" + node.Content(sourceCode)),
			Type:             "ne_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: NEExpressionNode, Parent: parentNode})
	case "==":
		var EQExpr model.EqExpr
		EQExpr.LeftOperand = expressionNode.LeftOperand
		EQExpr.RightOperand = expressionNode.RightOperand
		EQExpr.Op = expressionNode.Op
		EQExpr.BinaryExpr = expressionNode
		EQExpressionNode := &model.Node{
			ID:               util.GenerateSha256("eq_expression" + node.Content(sourceCode)),
			Type:             "eq_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: EQExpressionNode, Parent: parentNode})
	case "&":
		var BitwiseAndExpr model.AndBitwiseExpr
		BitwiseAndExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseAndExpr.RightOperand = expressionNode.RightOperand
		BitwiseAndExpr.Op = expressionNode.Op
		BitwiseAndExpr.BinaryExpr = expressionNode
		BitwiseAndExpressionNode := &model.Node{
			ID:               util.GenerateSha256("bitwise_and_expression" + node.Content(sourceCode)),
			Type:             "bitwise_and_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: BitwiseAndExpressionNode, Parent: parentNode})
	case "&&":
		var AndExpr model.AndLogicalExpr
		AndExpr.LeftOperand = expressionNode.LeftOperand
		AndExpr.RightOperand = expressionNode.RightOperand
		AndExpr.Op = expressionNode.Op
		AndExpr.BinaryExpr = expressionNode
		AndExpressionNode := &model.Node{
			ID:               util.GenerateSha256("and_expression" + node.Content(sourceCode)),
			Type:             "and_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: AndExpressionNode, Parent: parentNode})
	case "||":
		var OrExpr model.OrLogicalExpr
		OrExpr.LeftOperand = expressionNode.LeftOperand
		OrExpr.RightOperand = expressionNode.RightOperand
		OrExpr.Op = expressionNode.Op
		OrExpr.BinaryExpr = expressionNode
		OrExpressionNode := &model.Node{
			ID:               util.GenerateSha256("or_expression" + node.Content(sourceCode)),
			Type:             "or_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: OrExpressionNode, Parent: parentNode})
	case "|":
		var BitwiseOrExpr model.OrBitwiseExpr
		BitwiseOrExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseOrExpr.RightOperand = expressionNode.RightOperand
		BitwiseOrExpr.Op = expressionNode.Op
		BitwiseOrExpr.BinaryExpr = expressionNode
		BitwiseOrExpressionNode := &model.Node{
			ID:               util.GenerateSha256("bitwise_or_expression" + node.Content(sourceCode)),
			Type:             "bitwise_or_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: BitwiseOrExpressionNode, Parent: parentNode})
	case ">>>":
		var BitwiseRightShiftExpr model.UnsignedRightShiftExpr
		BitwiseRightShiftExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseRightShiftExpr.RightOperand = expressionNode.RightOperand
		BitwiseRightShiftExpr.Op = expressionNode.Op
		BitwiseRightShiftExpr.BinaryExpr = expressionNode
		BitwiseRightShiftExpressionNode := &model.Node{
			ID:               util.GenerateSha256("bitwise_right_shift_expression" + node.Content(sourceCode)),
			Type:             "bitwise_right_shift_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: BitwiseRightShiftExpressionNode, Parent: parentNode})
	case "^":
		var BitwiseXorExpr model.XorBitwiseExpr
		BitwiseXorExpr.LeftOperand = expressionNode.LeftOperand
		BitwiseXorExpr.RightOperand = expressionNode.RightOperand
		BitwiseXorExpr.Op = expressionNode.Op
		BitwiseXorExpr.BinaryExpr = expressionNode
		BitwiseXorExpressionNode := &model.Node{
			ID:               util.GenerateSha256("bitwise_xor_expression" + node.Content(sourceCode)),
			Type:             "bitwise_xor_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			IsJavaSourceFile: IsJavaSourceFile(file),
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&model.TreeNode{Node: BitwiseXorExpressionNode, Parent: parentNode})
	}

	invokedNode := &model.Node{
		ID:               util.GenerateSha256("binary_expression" + node.Content(sourceCode)),
		Type:             "binary_expression",
		Name:             node.Content(sourceCode),
		CodeSnippet:      node.Content(sourceCode),
		LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
		File:             file,
		IsJavaSourceFile: IsJavaSourceFile(file),
		BinaryExpr:       &expressionNode,
	}
	return invokedNode
}
