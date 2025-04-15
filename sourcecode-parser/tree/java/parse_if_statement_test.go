package java

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// TestParseIfStatement tests the ParseIfStatement function
func TestParseIfStatement(t *testing.T) {
	t.Run("Basic if statement with then block only", func(t *testing.T) {
		// Setup
		sourceCode := []byte("if (x > 0) { doSomething(); }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the if_statement node
		ifNode := findIfStatementNode(rootNode)
		assert.NotNil(t, ifNode)

		// Call the function with our parsed node
		ifStmt := ParseIfStatement(ifNode, sourceCode)

		// Assertions
		assert.NotNil(t, ifStmt)
		assert.NotNil(t, ifStmt.Condition)
		assert.Equal(t, "(x > 0)", ifStmt.Condition.NodeString)
		assert.NotEmpty(t, ifStmt.Then.NodeString)
		assert.Equal(t, "{ doSomething(); }", ifStmt.Then.NodeString)
		assert.Empty(t, ifStmt.Else.NodeString)
	})

	t.Run("If statement with else block", func(t *testing.T) {
		// Setup
		sourceCode := []byte("if (x <= 0) { return false; } else { return true; }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the if_statement node
		ifNode := findIfStatementNode(rootNode)
		assert.NotNil(t, ifNode)

		// Call the function with our parsed node
		ifStmt := ParseIfStatement(ifNode, sourceCode)

		// Assertions
		assert.NotNil(t, ifStmt)
		assert.NotNil(t, ifStmt.Condition)
		assert.Equal(t, "(x <= 0)", ifStmt.Condition.NodeString)
		assert.NotEmpty(t, ifStmt.Then.NodeString)
		assert.Equal(t, "{ return false; }", ifStmt.Then.NodeString)
		assert.NotEmpty(t, ifStmt.Else.NodeString)
		assert.Equal(t, "{ return true; }", ifStmt.Else.NodeString)
	})

	t.Run("If statement with complex condition", func(t *testing.T) {
		// Setup
		sourceCode := []byte("if (x > 0 && y < 10 || z == 5) { process(); }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the if_statement node
		ifNode := findIfStatementNode(rootNode)
		assert.NotNil(t, ifNode)

		// Call the function with our parsed node
		ifStmt := ParseIfStatement(ifNode, sourceCode)

		// Assertions
		assert.NotNil(t, ifStmt)
		assert.NotNil(t, ifStmt.Condition)
		assert.Equal(t, "(x > 0 && y < 10 || z == 5)", ifStmt.Condition.NodeString)
		assert.NotEmpty(t, ifStmt.Then.NodeString)
		assert.Equal(t, "{ process(); }", ifStmt.Then.NodeString)
		assert.Empty(t, ifStmt.Else.NodeString)
	})

	t.Run("If statement with else-if chain", func(t *testing.T) {
		// Setup
		sourceCode := []byte("if (score >= 90) { grade = 'A'; } else if (score >= 80) { grade = 'B'; }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the if_statement node
		ifNode := findIfStatementNode(rootNode)
		assert.NotNil(t, ifNode)

		// Call the function with our parsed node
		ifStmt := ParseIfStatement(ifNode, sourceCode)

		// Assertions
		assert.NotNil(t, ifStmt)
		assert.NotNil(t, ifStmt.Condition)
		assert.Equal(t, "(score >= 90)", ifStmt.Condition.NodeString)
		assert.NotEmpty(t, ifStmt.Then.NodeString)
		assert.Equal(t, "{ grade = 'A'; }", ifStmt.Then.NodeString)
		assert.NotEmpty(t, ifStmt.Else.NodeString)
		// The else block contains another if statement
		assert.Contains(t, ifStmt.Else.NodeString, "if (score >= 80)")
	})
}

// Helper function to find the if_statement node in the tree
func findIfStatementNode(node *sitter.Node) *sitter.Node {
	if node.Type() == "if_statement" {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findIfStatementNode(child); found != nil {
			return found
		}
	}

	return nil
}
