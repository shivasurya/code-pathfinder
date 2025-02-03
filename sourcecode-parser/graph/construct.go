package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	javalang "github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/java"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/smacker/go-tree-sitter/java"

	sitter "github.com/smacker/go-tree-sitter"
	//nolint:all
)

type Node struct {
	ID                   string
	Type                 string
	Name                 string
	CodeSnippet          string
	LineNumber           uint32
	IsExternal           bool
	Modifier             string
	ReturnType           string
	MethodArgumentsType  []string
	MethodArgumentsValue []string
	PackageName          string
	ImportPackage        []string
	SuperClass           string
	Interface            []string
	DataType             string
	Scope                string
	VariableValue        string
	hasAccess            bool
	File                 string
	isJavaSourceFile     bool
	ThrowsExceptions     []string
	Annotation           []string
	JavaDoc              *model.Javadoc
	BinaryExpr           *model.BinaryExpr
	ClassInstanceExpr    *model.ClassInstanceExpr
	IfStmt               *model.IfStmt
	WhileStmt            *model.WhileStmt
	DoStmt               *model.DoStmt
	ForStmt              *model.ForStmt
	BreakStmt            *model.BreakStmt
	ContinueStmt         *model.ContinueStmt
	YieldStmt            *model.YieldStmt
	AssertStmt           *model.AssertStmt
	ReturnStmt           *model.ReturnStmt
	BlockStmt            *model.BlockStmt
	FileNode             *model.File
}

type TreeNode struct {
	Node     *Node
	Children []*TreeNode
	Parent   *TreeNode
}

func (t *TreeNode) AddChild(child *TreeNode) {
	t.Children = append(t.Children, child)
}

func (t *TreeNode) AddChildren(children []*TreeNode) {
	t.Children = append(t.Children, children...)
}

func buildGraphFromAST(node *sitter.Node, sourceCode []byte, currentContext *Node, file string, parentNode *TreeNode) {
	isJavaSourceFile := javalang.IsJavaSourceFile(file)
	switch node.Type() {
	case "block":
		blockNode := javalang.ParseBlockStatement(node, sourceCode)
		uniqueBlockID := fmt.Sprintf("block_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		blockStmtNode := &Node{
			ID:               GenerateSha256(uniqueBlockID),
			Type:             "BlockStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "BlockStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			BlockStmt:        blockNode,
		}
		parentNode.AddChild(&TreeNode{Node: blockStmtNode, Parent: parentNode})
	case "return_statement":
		returnNode := javalang.ParseReturnStatement(node, sourceCode)
		uniqueReturnID := fmt.Sprintf("return_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		returnStmtNode := &Node{
			ID:               GenerateSha256(uniqueReturnID),
			Type:             "ReturnStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "ReturnStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			ReturnStmt:       returnNode,
		}
		parentNode.AddChild(&TreeNode{Node: returnStmtNode, Parent: parentNode})
	case "assert_statement":
		assertNode := javalang.ParseAssertStatement(node, sourceCode)
		uniqueAssertID := fmt.Sprintf("assert_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		assertStmtNode := &Node{
			ID:               GenerateSha256(uniqueAssertID),
			Type:             "AssertStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "AssertStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			AssertStmt:       assertNode,
		}
		parentNode.AddChild(&TreeNode{Node: assertStmtNode, Parent: parentNode})
	case "yield_statement":
		yieldNode := javalang.ParseYieldStatement(node, sourceCode)
		uniqueyieldID := fmt.Sprintf("yield_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		yieldStmtNode := &Node{
			ID:               GenerateSha256(uniqueyieldID),
			Type:             "YieldStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "YieldStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			YieldStmt:        yieldNode,
		}
		parentNode.AddChild(&TreeNode{Node: yieldStmtNode, Parent: parentNode})
	case "break_statement":
		breakNode := javalang.ParseBreakStatement(node, sourceCode)
		uniquebreakstmtID := fmt.Sprintf("breakstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		breakStmtNode := &Node{
			ID:               GenerateSha256(uniquebreakstmtID),
			Type:             "BreakStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "BreakStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			BreakStmt:        breakNode,
		}
		parentNode.AddChild(&TreeNode{Node: breakStmtNode, Parent: parentNode})
	case "continue_statement":
		continueNode := javalang.ParseContinueStatement(node, sourceCode)
		uniquecontinueID := fmt.Sprintf("continuestmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		continueStmtNode := &Node{
			ID:               GenerateSha256(uniquecontinueID),
			Type:             "ContinueStmt",
			LineNumber:       node.StartPoint().Row + 1,
			Name:             "ContinueStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			ContinueStmt:     continueNode,
		}
		parentNode.AddChild(&TreeNode{Node: continueStmtNode, Parent: parentNode})
	case "if_statement":
		ifNode := javalang.ParseIfStatement(node, sourceCode)
		methodID := fmt.Sprintf("ifstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		// add node to graph
		ifStmtNode := &Node{
			ID:               GenerateSha256(methodID),
			Type:             "IfStmt",
			Name:             "IfStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			IfStmt:           ifNode,
		}
		parentNode.AddChild(&TreeNode{Node: ifStmtNode, Parent: parentNode})
	case "while_statement":
		whileNode := model.WhileStmt{}
		// get the condition of the while statement
		conditionNode := node.Child(1)
		if conditionNode != nil {
			whileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
		}
		methodID := fmt.Sprintf("while_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		// add node to graph
		whileStmtNode := &Node{
			ID:               GenerateSha256(methodID),
			Type:             "WhileStmt",
			Name:             "WhileStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			WhileStmt:        &whileNode,
		}
		parentNode.AddChild(&TreeNode{Node: whileStmtNode, Parent: parentNode})
	case "do_statement":
		doWhileNode := model.DoStmt{}
		// get the condition of the while statement
		conditionNode := node.Child(2)
		if conditionNode != nil {
			doWhileNode.Condition = &model.Expr{Node: *conditionNode, NodeString: conditionNode.Content(sourceCode)}
		}
		methodID := fmt.Sprintf("dowhile_stmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
		// add node to graph
		doWhileStmtNode := &Node{
			ID:               GenerateSha256(methodID),
			Type:             "DoStmt",
			Name:             "DoStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			DoStmt:           &doWhileNode,
		}
		parentNode.AddChild(&TreeNode{Node: doWhileStmtNode, Parent: parentNode})
	case "for_statement":
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
		forStmtNode := &Node{
			ID:               GenerateSha256(methodID),
			Type:             "ForStmt",
			Name:             "ForStmt",
			IsExternal:       true,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			ForStmt:          &forNode,
		}
		parentNode.AddChild(&TreeNode{Node: forStmtNode, Parent: parentNode})
	case "binary_expression":
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
			addExpressionNode := &Node{
				ID:               GenerateSha256("add_expression" + node.Content(sourceCode)),
				Type:             "add_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: addExpressionNode, Parent: parentNode})
		case "-":
			var subExpr model.SubExpr
			subExpr.LeftOperand = expressionNode.LeftOperand
			subExpr.RightOperand = expressionNode.RightOperand
			subExpr.Op = expressionNode.Op
			subExpr.BinaryExpr = expressionNode
			subExpressionNode := &Node{
				ID:               GenerateSha256("sub_expression" + node.Content(sourceCode)),
				Type:             "sub_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: subExpressionNode, Parent: parentNode})
		case "*":
			var mulExpr model.MulExpr
			mulExpr.LeftOperand = expressionNode.LeftOperand
			mulExpr.RightOperand = expressionNode.RightOperand
			mulExpr.Op = expressionNode.Op
			mulExpr.BinaryExpr = expressionNode
			mulExpressionNode := &Node{
				ID:               GenerateSha256("mul_expression" + node.Content(sourceCode)),
				Type:             "mul_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: mulExpressionNode, Parent: parentNode})
		case "/":
			var divExpr model.DivExpr
			divExpr.LeftOperand = expressionNode.LeftOperand
			divExpr.RightOperand = expressionNode.RightOperand
			divExpr.Op = expressionNode.Op
			divExpr.BinaryExpr = expressionNode
			divExpressionNode := &Node{
				ID:               GenerateSha256("div_expression" + node.Content(sourceCode)),
				Type:             "div_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: divExpressionNode, Parent: parentNode})
		case ">", "<", ">=", "<=":
			var compExpr model.ComparisonExpr
			compExpr.LeftOperand = expressionNode.LeftOperand
			compExpr.RightOperand = expressionNode.RightOperand
			compExpr.Op = expressionNode.Op
			compExpr.BinaryExpr = expressionNode
			compExpressionNode := &Node{
				ID:               GenerateSha256("comp_expression" + node.Content(sourceCode)),
				Type:             "comp_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: compExpressionNode, Parent: parentNode})
		case "%":
			var RemExpr model.RemExpr
			RemExpr.LeftOperand = expressionNode.LeftOperand
			RemExpr.RightOperand = expressionNode.RightOperand
			RemExpr.Op = expressionNode.Op
			RemExpr.BinaryExpr = expressionNode
			RemExpressionNode := &Node{
				ID:               GenerateSha256("rem_expression" + node.Content(sourceCode)),
				Type:             "rem_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: RemExpressionNode, Parent: parentNode})
		case ">>":
			var RightShiftExpr model.RightShiftExpr
			RightShiftExpr.LeftOperand = expressionNode.LeftOperand
			RightShiftExpr.RightOperand = expressionNode.RightOperand
			RightShiftExpr.Op = expressionNode.Op
			RightShiftExpr.BinaryExpr = expressionNode
			RightShiftExpressionNode := &Node{
				ID:               GenerateSha256("right_shift_expression" + node.Content(sourceCode)),
				Type:             "right_shift_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: RightShiftExpressionNode, Parent: parentNode})
		case "<<":
			var LeftShiftExpr model.LeftShiftExpr
			LeftShiftExpr.LeftOperand = expressionNode.LeftOperand
			LeftShiftExpr.RightOperand = expressionNode.RightOperand
			LeftShiftExpr.Op = expressionNode.Op
			LeftShiftExpr.BinaryExpr = expressionNode
			LeftShiftExpressionNode := &Node{
				ID:               GenerateSha256("left_shift_expression" + node.Content(sourceCode)),
				Type:             "left_shift_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: LeftShiftExpressionNode, Parent: parentNode})
		case "!=":
			var NEExpr model.NEExpr
			NEExpr.LeftOperand = expressionNode.LeftOperand
			NEExpr.RightOperand = expressionNode.RightOperand
			NEExpr.Op = expressionNode.Op
			NEExpr.BinaryExpr = expressionNode
			NEExpressionNode := &Node{
				ID:               GenerateSha256("ne_expression" + node.Content(sourceCode)),
				Type:             "ne_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: NEExpressionNode, Parent: parentNode})
		case "==":
			var EQExpr model.EqExpr
			EQExpr.LeftOperand = expressionNode.LeftOperand
			EQExpr.RightOperand = expressionNode.RightOperand
			EQExpr.Op = expressionNode.Op
			EQExpr.BinaryExpr = expressionNode
			EQExpressionNode := &Node{
				ID:               GenerateSha256("eq_expression" + node.Content(sourceCode)),
				Type:             "eq_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: EQExpressionNode, Parent: parentNode})
		case "&":
			var BitwiseAndExpr model.AndBitwiseExpr
			BitwiseAndExpr.LeftOperand = expressionNode.LeftOperand
			BitwiseAndExpr.RightOperand = expressionNode.RightOperand
			BitwiseAndExpr.Op = expressionNode.Op
			BitwiseAndExpr.BinaryExpr = expressionNode
			BitwiseAndExpressionNode := &Node{
				ID:               GenerateSha256("bitwise_and_expression" + node.Content(sourceCode)),
				Type:             "bitwise_and_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: BitwiseAndExpressionNode, Parent: parentNode})
		case "&&":
			var AndExpr model.AndLogicalExpr
			AndExpr.LeftOperand = expressionNode.LeftOperand
			AndExpr.RightOperand = expressionNode.RightOperand
			AndExpr.Op = expressionNode.Op
			AndExpr.BinaryExpr = expressionNode
			AndExpressionNode := &Node{
				ID:               GenerateSha256("and_expression" + node.Content(sourceCode)),
				Type:             "and_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: AndExpressionNode, Parent: parentNode})
		case "||":
			var OrExpr model.OrLogicalExpr
			OrExpr.LeftOperand = expressionNode.LeftOperand
			OrExpr.RightOperand = expressionNode.RightOperand
			OrExpr.Op = expressionNode.Op
			OrExpr.BinaryExpr = expressionNode
			OrExpressionNode := &Node{
				ID:               GenerateSha256("or_expression" + node.Content(sourceCode)),
				Type:             "or_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: OrExpressionNode, Parent: parentNode})
		case "|":
			var BitwiseOrExpr model.OrBitwiseExpr
			BitwiseOrExpr.LeftOperand = expressionNode.LeftOperand
			BitwiseOrExpr.RightOperand = expressionNode.RightOperand
			BitwiseOrExpr.Op = expressionNode.Op
			BitwiseOrExpr.BinaryExpr = expressionNode
			BitwiseOrExpressionNode := &Node{
				ID:               GenerateSha256("bitwise_or_expression" + node.Content(sourceCode)),
				Type:             "bitwise_or_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: BitwiseOrExpressionNode, Parent: parentNode})
		case ">>>":
			var BitwiseRightShiftExpr model.UnsignedRightShiftExpr
			BitwiseRightShiftExpr.LeftOperand = expressionNode.LeftOperand
			BitwiseRightShiftExpr.RightOperand = expressionNode.RightOperand
			BitwiseRightShiftExpr.Op = expressionNode.Op
			BitwiseRightShiftExpr.BinaryExpr = expressionNode
			BitwiseRightShiftExpressionNode := &Node{
				ID:               GenerateSha256("bitwise_right_shift_expression" + node.Content(sourceCode)),
				Type:             "bitwise_right_shift_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: BitwiseRightShiftExpressionNode, Parent: parentNode})
		case "^":
			var BitwiseXorExpr model.XorBitwiseExpr
			BitwiseXorExpr.LeftOperand = expressionNode.LeftOperand
			BitwiseXorExpr.RightOperand = expressionNode.RightOperand
			BitwiseXorExpr.Op = expressionNode.Op
			BitwiseXorExpr.BinaryExpr = expressionNode
			BitwiseXorExpressionNode := &Node{
				ID:               GenerateSha256("bitwise_xor_expression" + node.Content(sourceCode)),
				Type:             "bitwise_xor_expression",
				Name:             node.Content(sourceCode),
				CodeSnippet:      node.Content(sourceCode),
				LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				BinaryExpr:       &expressionNode,
			}
			parentNode.AddChild(&TreeNode{Node: BitwiseXorExpressionNode, Parent: parentNode})
		}

		invokedNode := &Node{
			ID:               GenerateSha256("binary_expression" + node.Content(sourceCode)),
			Type:             "binary_expression",
			Name:             node.Content(sourceCode),
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1, // Lines start from 0 in the AST
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			BinaryExpr:       &expressionNode,
		}
		parentNode.AddChild(&TreeNode{Node: invokedNode, Parent: parentNode})
	case "method_declaration":
		var javadoc *model.Javadoc
		if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
			commentContent := node.PrevSibling().Content(sourceCode)
			if strings.HasPrefix(commentContent, "/*") {
				javadoc = javalang.ParseJavadocTags(commentContent)
			}
		}
		methodName, methodID := extractMethodName(node, sourceCode, file)
		modifiers := ""
		returnType := ""
		throws := []string{}
		methodArgumentType := []string{}
		methodArgumentValue := []string{}
		annotationMarkers := []string{}

		for i := 0; i < int(node.ChildCount()); i++ {
			childNode := node.Child(i)
			childType := childNode.Type()

			switch childType {
			case "throws":
				// namedChild
				for j := 0; j < int(childNode.NamedChildCount()); j++ {
					namedChild := childNode.NamedChild(j)
					if namedChild.Type() == "type_identifier" {
						throws = append(throws, namedChild.Content(sourceCode))
					}
				}
			case "modifiers":
				modifiers = childNode.Content(sourceCode)
				for j := 0; j < int(childNode.ChildCount()); j++ {
					if childNode.Child(j).Type() == "marker_annotation" {
						annotationMarkers = append(annotationMarkers, childNode.Child(j).Content(sourceCode))
					}
				}
			case "void_type", "type_identifier":
				// get return type of method
				returnType = childNode.Content(sourceCode)
			case "formal_parameters":
				// get method arguments
				for j := 0; j < int(childNode.NamedChildCount()); j++ {
					param := childNode.NamedChild(j)
					if param.Type() == "formal_parameter" {
						// get type of argument and add to method arguments
						paramType := param.Child(0).Content(sourceCode)
						paramValue := param.Child(1).Content(sourceCode)
						methodArgumentType = append(methodArgumentType, paramType)
						methodArgumentValue = append(methodArgumentValue, paramValue)
					}
				}
			}
		}

		invokedNode := &Node{
			ID:                   methodID, // In a real scenario, you would construct a unique ID, possibly using the method signature
			Type:                 "method_declaration",
			Name:                 methodName,
			CodeSnippet:          node.Content(sourceCode),
			LineNumber:           node.StartPoint().Row + 1, // Lines start from 0 in the AST
			Modifier:             javalang.ExtractVisibilityModifier(modifiers),
			ReturnType:           returnType,
			MethodArgumentsType:  methodArgumentType,
			MethodArgumentsValue: methodArgumentValue,
			File:                 file,
			isJavaSourceFile:     isJavaSourceFile,
			ThrowsExceptions:     throws,
			Annotation:           annotationMarkers,
			JavaDoc:              javadoc,
		}
		methodNode := &TreeNode{Node: invokedNode, Parent: parentNode}
		parentNode.AddChild(methodNode)
		currentContext = invokedNode // Update context to the new method

		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			buildGraphFromAST(child, sourceCode, currentContext, file, methodNode)
		}

	case "method_invocation":
		methodName, methodID := extractMethodName(node, sourceCode, file)
		arguments := []string{}
		// get argument list from arguments node iterate for child node
		for i := 0; i < int(node.ChildCount()); i++ {
			if node.Child(i).Type() == "argument_list" {
				argumentsNode := node.Child(i)
				for j := 0; j < int(argumentsNode.ChildCount()); j++ {
					argument := argumentsNode.Child(j)
					switch argument.Type() {
					case "identifier":
						arguments = append(arguments, argument.Content(sourceCode))
					case "string_literal":
						stringliteral := argument.Content(sourceCode)
						stringliteral = strings.TrimPrefix(stringliteral, "\"")
						stringliteral = strings.TrimSuffix(stringliteral, "\"")
						arguments = append(arguments, stringliteral)
					default:
						arguments = append(arguments, argument.Content(sourceCode))
					}
				}
			}
		}

		invokedNode := &Node{
			ID:                   methodID,
			Type:                 "method_invocation",
			Name:                 methodName,
			IsExternal:           true,
			CodeSnippet:          node.Content(sourceCode),
			LineNumber:           node.StartPoint().Row + 1, // Lines start from 0 in the AST
			MethodArgumentsValue: arguments,
			File:                 file,
			isJavaSourceFile:     isJavaSourceFile,
		}
		methodInvocationTreeNode := &TreeNode{Node: invokedNode, Parent: parentNode}
		parentNode.AddChild(methodInvocationTreeNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			buildGraphFromAST(child, sourceCode, currentContext, file, methodInvocationTreeNode)
		}
	case "class_declaration":
		var javadoc *model.Javadoc
		if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
			commentContent := node.PrevSibling().Content(sourceCode)
			if strings.HasPrefix(commentContent, "/*") {
				javadoc = javalang.ParseJavadocTags(commentContent)
			}
		}
		className := node.ChildByFieldName("name").Content(sourceCode)
		packageName := ""
		accessModifier := ""
		superClass := ""
		annotationMarkers := []string{}
		implementedInterface := []string{}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "modifiers" {
				accessModifier = child.Content(sourceCode)
				for j := 0; j < int(child.ChildCount()); j++ {
					if child.Child(j).Type() == "marker_annotation" {
						annotationMarkers = append(annotationMarkers, child.Child(j).Content(sourceCode))
					}
				}
			}
			if child.Type() == "superclass" {
				for j := 0; j < int(child.ChildCount()); j++ {
					if child.Child(j).Type() == "type_identifier" {
						superClass = child.Child(j).Content(sourceCode)
					}
				}
			}
			if child.Type() == "super_interfaces" {
				for j := 0; j < int(child.ChildCount()); j++ {
					// typelist node and then iterate through type_identifier node
					typeList := child.Child(j)
					for k := 0; k < int(typeList.ChildCount()); k++ {
						implementedInterface = append(implementedInterface, typeList.Child(k).Content(sourceCode))
					}
				}
			}
		}

		classNode := &Node{
			ID:               GenerateMethodID(className, []string{}, file),
			Type:             "class_declaration",
			Name:             className,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			PackageName:      packageName,
			Modifier:         javalang.ExtractVisibilityModifier(accessModifier),
			SuperClass:       superClass,
			Interface:        implementedInterface,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			JavaDoc:          javadoc,
			Annotation:       annotationMarkers,
		}
		classTreeNode := &TreeNode{
			Node:     classNode,
			Children: nil,
			Parent:   parentNode,
		}
		parentNode.AddChild(classTreeNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			buildGraphFromAST(child, sourceCode, currentContext, file, classTreeNode)
		}
	case "block_comment":
		// Parse block comments
		if strings.HasPrefix(node.Content(sourceCode), "/*") {
			commentContent := node.Content(sourceCode)
			javadocTags := javalang.ParseJavadocTags(commentContent)

			commentNode := &Node{
				ID:               GenerateMethodID(node.Content(sourceCode), []string{}, file),
				Type:             "block_comment",
				CodeSnippet:      commentContent,
				LineNumber:       node.StartPoint().Row + 1,
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				JavaDoc:          javadocTags,
			}
			parentNode.AddChild(&TreeNode{Node: commentNode, Parent: parentNode})
		}
	case "local_variable_declaration", "field_declaration":
		// Extract variable name, type, and modifiers
		variableName := ""
		variableType := ""
		variableModifier := ""
		variableValue := ""
		hasAccessValue := false
		var scope string
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "variable_declarator":
				variableName = child.Content(sourceCode)
				for j := 0; j < int(child.ChildCount()); j++ {
					if child.Child(j).Type() == "identifier" {
						variableName = child.Child(j).Content(sourceCode)
					}
					// if child type contains =, iterate through and get remaining content
					if child.Child(j).Type() == "=" {
						for k := j + 1; k < int(child.ChildCount()); k++ {
							variableValue += child.Child(k).Content(sourceCode)
						}
					}

				}
				// remove spaces from variable value
				variableValue = strings.ReplaceAll(variableValue, " ", "")
				// remove new line from variable value
				variableValue = strings.ReplaceAll(variableValue, "\n", "")
			case "modifiers":
				variableModifier = child.Content(sourceCode)
			}
			// if child type contains type, get the type of variable
			if strings.Contains(child.Type(), "type") {
				variableType = child.Content(sourceCode)
			}
		}
		if node.Type() == "local_variable_declaration" {
			scope = "local"
			//nolint:all
			// hasAccessValue = hasAccess(node.NextSibling(), variableName, sourceCode)
		} else {
			scope = "field"
		}
		// Create a new node for the variable
		variableNode := &Node{
			ID:               GenerateMethodID(variableName, []string{}, file),
			Type:             "variable_declaration",
			Name:             variableName,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			Modifier:         javalang.ExtractVisibilityModifier(variableModifier),
			DataType:         variableType,
			Scope:            scope,
			VariableValue:    variableValue,
			hasAccess:        hasAccessValue,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
		}
		parentNode.AddChild(&TreeNode{
			Node:     variableNode,
			Children: nil,
			Parent:   parentNode,
		})
	case "object_creation_expression":
		className := ""
		classInstanceExpression := model.ClassInstanceExpr{
			ClassName: "",
			Args:      []*model.Expr{},
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "type_identifier" || child.Type() == "scoped_type_identifier" {
				className = child.Content(sourceCode)
				classInstanceExpression.ClassName = className
			}
			if child.Type() == "argument_list" {
				classInstanceExpression.Args = []*model.Expr{}
				for j := 0; j < int(child.ChildCount()); j++ {
					argType := child.Child(j).Type()
					argumentStopWords := map[string]bool{
						"(": true,
						")": true,
						"{": true,
						"}": true,
						"[": true,
						"]": true,
						",": true,
					}
					if !argumentStopWords[argType] {
						argument := &model.Expr{}
						argument.Type = child.Child(j).Type()
						argument.NodeString = child.Child(j).Content(sourceCode)
						classInstanceExpression.Args = append(classInstanceExpression.Args, argument)
					}
				}
			}
		}

		objectNode := &Node{
			ID:                GenerateMethodID(className, []string{strconv.Itoa(int(node.StartPoint().Row + 1))}, file),
			Type:              "ClassInstanceExpr",
			Name:              className,
			CodeSnippet:       node.Content(sourceCode),
			LineNumber:        node.StartPoint().Row + 1,
			File:              file,
			isJavaSourceFile:  isJavaSourceFile,
			ClassInstanceExpr: &classInstanceExpression,
		}
		parentNode.AddChild(&TreeNode{
			Node:     objectNode,
			Children: nil,
			Parent:   parentNode,
		})
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, currentContext, file, parentNode)
	}
}

//nolint:all
func extractMethodName(node *sitter.Node, sourceCode []byte, filepath string) (string, string) {
	var methodID string

	// if the child node is method_declaration, extract method name, modifiers, parameters, and return type
	var methodName string
	var modifiers, parameters []string

	if node.Type() == "method_declaration" {
		// Iterate over all children of the method_declaration node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "modifiers", "marker_annotation", "annotation":
				// This child is a modifier or annotation, add its content to modifiers
				modifiers = append(modifiers, child.Content(sourceCode)) //nolint:all
			case "identifier":
				// This child is the method name
				methodName = child.Content(sourceCode)
			case "formal_parameters":
				// This child represents formal parameters; iterate through its children
				for j := 0; j < int(child.NamedChildCount()); j++ {
					param := child.NamedChild(j)
					parameters = append(parameters, param.Content(sourceCode))
				}
			}
		}
	}

	// check if type is method_invocation
	// if the child node is method_invocation, extract method name
	if node.Type() == "method_invocation" {
		for j := 0; j < int(node.ChildCount()); j++ {
			child := node.Child(j)
			if child.Type() == "identifier" {
				if methodName == "" {
					methodName = child.Content(sourceCode)
				} else {
					methodName = methodName + "." + child.Content(sourceCode)
				}
			}

			argumentsNode := node.ChildByFieldName("argument_list")
			// add data type of arguments list
			if argumentsNode != nil {
				for k := 0; k < int(argumentsNode.ChildCount()); k++ {
					argument := argumentsNode.Child(k)
					parameters = append(parameters, argument.Child(0).Content(sourceCode))
				}
			}

		}
	}
	content := node.Content(sourceCode)
	lineNumber := int(node.StartPoint().Row) + 1
	columnNumber := int(node.StartPoint().Column) + 1
	// convert to string and merge
	content += " " + strconv.Itoa(lineNumber) + ":" + strconv.Itoa(columnNumber)
	methodID = GenerateMethodID(methodName, parameters, filepath+"/"+content)
	return methodName, methodID
}
func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// append only java files
			if filepath.Ext(path) == ".java" {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func Initialize(directory string) []*TreeNode {
	treeHolder := []*TreeNode{}
	// record start time
	start := time.Now()

	files, err := getFiles(directory)
	if err != nil {
		//nolint:all
		Log("Directory not found:", err)
		return treeHolder
	}

	totalFiles := len(files)
	numWorkers := 5 // Number of concurrent workers
	fileChan := make(chan string, totalFiles)
	treeChan := make(chan *TreeNode, totalFiles)
	statusChan := make(chan string, numWorkers)
	progressChan := make(chan int, totalFiles)
	var wg sync.WaitGroup

	// Worker function
	worker := func(workerID int) {
		// Initialize the parser for each worker
		parser := sitter.NewParser()
		defer parser.Close()

		// Set the language (Java in this case)
		parser.SetLanguage(java.GetLanguage())

		for file := range fileChan {
			fileName := filepath.Base(file)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Reading and parsing code %s\033[0m", workerID, fileName)
			sourceCode, err := readFile(file)
			if err != nil {
				Log("File not found:", err)
				continue
			}
			// Parse the source code
			tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
			if err != nil {
				Log("Error parsing file:", err)
				continue
			}
			//nolint:all
			defer tree.Close()

			rootNode := tree.RootNode()
			localTree := &TreeNode{
				Parent: nil,
				Node: &Node{
					ID:       fileName,
					Type:     "file",
					FileNode: &model.File{File: fileName},
				},
			}
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Building graph and traversing code %s\033[0m", workerID, fileName)
			buildGraphFromAST(rootNode, sourceCode, nil, file, localTree)
			treeHolder = append(treeHolder, localTree)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing file %s\033[0m", workerID, fileName)

			treeChan <- localTree
			progressChan <- 1
		}
		wg.Done()
	}

	// Start workers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i + 1)
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Status updater
	go func() {
		statusLines := make([]string, numWorkers)
		progress := 0
		for {
			select {
			case status, ok := <-statusChan:
				if !ok {
					return
				}
				workerID := int(status[12] - '0')
				statusLines[workerID-1] = status
			case _, ok := <-progressChan:
				if !ok {
					return
				}
				progress++
			}
			fmt.Print("\033[H\033[J") // Clear the screen
			for _, line := range statusLines {
				Log(line)
			}
			Fmt("Progress: %d%%\n", (progress*100)/totalFiles)
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(statusChan)
		close(progressChan)
		close(treeChan)
	}()

	// Print tree structure recursively from treeChan
	// for treeNode := range treeChan {
	// 	printTree(treeNode, 0)
	// }

	end := time.Now()
	elapsed := end.Sub(start)
	Log("Elapsed time: ", elapsed)
	Log("Project parsed successfully")

	return treeHolder
}

// func printTree(node *TreeNode, level int) {
// 	tab := strings.Repeat("\t", level)
// 	fmt.Println(tab+"Value:", node.NodeType)
// 	fmt.Println(tab+"Code:", node.Node.CodeSnippet)
// 	fmt.Println(tab + "-------------------------------------")
// 	for _, child := range node.Children {
// 		printTree(child, level+1)
// 	}
// }
