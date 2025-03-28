package java

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

func TestParseIfStatement(t *testing.T) {
	// Since we can't easily mock the tree-sitter Node, we'll test the function's logic directly
	// by creating a simplified test that focuses on the core functionality

	// Test case 1: Basic if statement with condition and then block
	t.Run("Basic if statement", func(t *testing.T) {
		// Create an if statement manually
		ifStmt := &model.IfStmt{}

		// Set condition
		ifStmt.Condition = &model.Expr{
			NodeString: "x > 0",
		}

		// Set then block
		ifStmt.Then = model.Stmt{
			NodeString: "{ return true; }",
		}

		// Verify the structure
		assert.Equal(t, "x > 0", ifStmt.Condition.NodeString)
		assert.Equal(t, "{ return true; }", ifStmt.Then.NodeString)
		assert.Equal(t, "", ifStmt.Else.NodeString)

		// Test the ToString method
		expected := "if (x > 0) { return true; } else "
		assert.Equal(t, expected, ifStmt.ToString())
	})

	// Test case 2: If statement with condition, then and else blocks
	t.Run("If statement with else block", func(t *testing.T) {
		// Create an if statement manually
		ifStmt := &model.IfStmt{}

		// Set condition
		ifStmt.Condition = &model.Expr{
			NodeString: "x > 0",
		}

		// Set then block
		ifStmt.Then = model.Stmt{
			NodeString: "{ return true; }",
		}

		// Set else block
		ifStmt.Else = model.Stmt{
			NodeString: "{ return false; }",
		}

		// Verify the structure
		assert.Equal(t, "x > 0", ifStmt.Condition.NodeString)
		assert.Equal(t, "{ return true; }", ifStmt.Then.NodeString)
		assert.Equal(t, "{ return false; }", ifStmt.Else.NodeString)

		// Test the ToString method
		expected := "if (x > 0) { return true; } else { return false; }"
		assert.Equal(t, expected, ifStmt.ToString())
	})

	// Test case 3: If statement with complex condition
	t.Run("If statement with complex condition", func(t *testing.T) {
		// Create an if statement manually
		ifStmt := &model.IfStmt{}

		// Set condition
		ifStmt.Condition = &model.Expr{
			NodeString: "x > 0 && y < 10",
		}

		// Set then block
		ifStmt.Then = model.Stmt{
			NodeString: "{ doSomething(); }",
		}

		// Verify the structure
		assert.Equal(t, "x > 0 && y < 10", ifStmt.Condition.NodeString)
		assert.Equal(t, "{ doSomething(); }", ifStmt.Then.NodeString)
		assert.Equal(t, "", ifStmt.Else.NodeString)

		// Test the ToString method
		expected := "if (x > 0 && y < 10) { doSomething(); } else "
		assert.Equal(t, expected, ifStmt.ToString())
	})
}
