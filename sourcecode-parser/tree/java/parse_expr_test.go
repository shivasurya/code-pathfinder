package java

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// TestParseExpr tests the ParseExpr function with different binary expression operators
func TestParseExpr(t *testing.T) {
	t.Run("Addition expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a + b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "+", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have AddExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		addExprNode := childNode.Children[0]
		assert.NotNil(t, addExprNode.Node.AddExpr)
	})

	t.Run("Subtraction expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("x - y")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "-", expr.Op)
		assert.Equal(t, "x", expr.LeftOperand.NodeString)
		assert.Equal(t, "y", expr.RightOperand.NodeString)

		// Check child nodes (should have SubExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		subExprNode := childNode.Children[0]
		assert.NotNil(t, subExprNode.Node.SubExpr)
	})

	t.Run("Multiplication expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a * b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "*", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have MulExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		mulExprNode := childNode.Children[0]
		assert.NotNil(t, mulExprNode.Node.MulExpr)
	})

	t.Run("Division expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a / b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "/", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have DivExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		divExprNode := childNode.Children[0]
		assert.NotNil(t, divExprNode.Node.DivExpr)
	})

	t.Run("Greater than comparison", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a > b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, ">", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have ComparisonExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		compExprNode := childNode.Children[0]
		assert.NotNil(t, compExprNode.Node.ComparisonExpr)
	})

	t.Run("Remainder expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a % b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "%", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have RemExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		remExprNode := childNode.Children[0]
		assert.NotNil(t, remExprNode.Node.RemExpr)
	})

	t.Run("Right shift expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a >> b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, ">>", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have RightShiftExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		rightShiftExprNode := childNode.Children[0]
		assert.NotNil(t, rightShiftExprNode.Node.RightShiftExpr)
	})

	t.Run("Left shift expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a << b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "<<", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have LeftShiftExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		leftShiftExprNode := childNode.Children[0]
		assert.NotNil(t, leftShiftExprNode.Node.LeftShiftExpr)
	})

	t.Run("Not equal expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a != b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "!=", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have NEExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		neExprNode := childNode.Children[0]
		assert.NotNil(t, neExprNode.Node.NEExpr)
	})

	t.Run("Equal expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a == b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "==", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have EQExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		eqExprNode := childNode.Children[0]
		assert.NotNil(t, eqExprNode.Node.EQExpr)
	})

	t.Run("Bitwise AND expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a & b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "&", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have AndBitwiseExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		bitwiseAndExprNode := childNode.Children[0]
		assert.NotNil(t, bitwiseAndExprNode.Node.AndBitwiseExpr)
	})

	t.Run("Logical AND expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a && b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "&&", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have AndLogicalExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		logicalAndExprNode := childNode.Children[0]
		assert.NotNil(t, logicalAndExprNode.Node.AndLogicalExpr)
	})

	t.Run("Logical OR expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a || b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "||", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have OrLogicalExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		logicalOrExprNode := childNode.Children[0]
		assert.NotNil(t, logicalOrExprNode.Node.OrLogicalExpr)
	})

	t.Run("Bitwise OR expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a | b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "|", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
	})

	t.Run("Unsigned right shift expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a >>> b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, ">>>", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have UnsignedRightShiftExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		unsignedRightShiftExprNode := childNode.Children[0]
		assert.NotNil(t, unsignedRightShiftExprNode.Node.UnsignedRightShiftExpr)
	})

	t.Run("Bitwise XOR expression", func(t *testing.T) {
		// Setup
		sourceCode := []byte("a ^ b")

		// Parse source code
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find binary expression node
		binaryExprNode := findBinaryExprNode(rootNode)

		// Create parent node for test
		parentNode := &model.TreeNode{
			Node:     &model.Node{},
			Children: make([]*model.TreeNode, 0),
		}

		// Call the function with our parsed node
		expr := ParseExpr(binaryExprNode, sourceCode, "Test.java", parentNode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "^", expr.Op)
		assert.Equal(t, "a", expr.LeftOperand.NodeString)
		assert.Equal(t, "b", expr.RightOperand.NodeString)

		// Check child nodes (should have XorBitwiseExpr node)
		assert.Equal(t, 1, len(parentNode.Children))
		childNode := parentNode.Children[0]
		assert.NotNil(t, childNode.Node.BinaryExpr)
		assert.Equal(t, 1, len(childNode.Children))
		bitwiseXorExprNode := childNode.Children[0]
		assert.NotNil(t, bitwiseXorExprNode.Node.XorBitwiseExpr)
	})
}

// Helper function to find the binary_expression node in the tree
func findBinaryExprNode(node *sitter.Node) *sitter.Node {
	if node.Type() == "binary_expression" {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findBinaryExprNode(child); found != nil {
			return found
		}
	}

	return nil
}
