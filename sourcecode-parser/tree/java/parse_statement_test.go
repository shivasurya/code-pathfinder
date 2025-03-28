package java

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

func TestParseBreakStatement(t *testing.T) {
	t.Run("Break statement without label", func(t *testing.T) {
		// Create a break statement manually
		breakStmt := &model.BreakStmt{}

		// Verify the structure
		assert.Equal(t, "", breakStmt.Label)
	})

	t.Run("Break statement with label", func(t *testing.T) {
		// Create a break statement with label
		breakStmt := &model.BreakStmt{
			Label: "outerLoop",
		}

		// Verify the structure
		assert.Equal(t, "outerLoop", breakStmt.Label)
		assert.Equal(t, "break (outerLoop)", breakStmt.ToString())
	})
}

func TestParseContinueStatement(t *testing.T) {
	t.Run("Continue statement without label", func(t *testing.T) {
		// Create a continue statement manually
		continueStmt := &model.ContinueStmt{}

		// Verify the structure
		assert.Equal(t, "", continueStmt.Label)
	})

	t.Run("Continue statement with label", func(t *testing.T) {
		// Create a continue statement with label
		continueStmt := &model.ContinueStmt{
			Label: "outerLoop",
		}

		// Verify the structure
		assert.Equal(t, "outerLoop", continueStmt.Label)
		assert.Equal(t, "continue (outerLoop)", continueStmt.ToString())
	})
}

func TestParseYieldStatement(t *testing.T) {
	t.Run("Yield statement with value", func(t *testing.T) {
		// Create a yield statement manually
		yieldStmt := &model.YieldStmt{
			Value: &model.Expr{
				NodeString: "42",
			},
		}

		// Verify the structure
		assert.NotNil(t, yieldStmt.Value)
		assert.Equal(t, "42", yieldStmt.Value.NodeString)
		assert.Equal(t, "yield 42", yieldStmt.ToString())
		assert.Equal(t, yieldStmt.Value, yieldStmt.GetValue())
	})
}

func TestParseAssertStatement(t *testing.T) {
	t.Run("Assert statement without message", func(t *testing.T) {
		// Create an assert statement manually
		assertStmt := &model.AssertStmt{
			Expr: &model.Expr{
				NodeString: "x > 0",
			},
		}

		// Verify the structure
		assert.NotNil(t, assertStmt.Expr)
		assert.Nil(t, assertStmt.Message)
		assert.Equal(t, "x > 0", assertStmt.Expr.NodeString)
		assert.Equal(t, "assert x > 0", assertStmt.ToString())
		assert.Equal(t, assertStmt.Expr, assertStmt.GetExpr())
	})

	t.Run("Assert statement with message", func(t *testing.T) {
		// Create an assert statement with message
		assertStmt := &model.AssertStmt{
			Expr: &model.Expr{
				NodeString: "x > 0",
			},
			Message: &model.Expr{
				NodeString: "\"Value must be positive\"",
			},
		}

		// Verify the structure
		assert.NotNil(t, assertStmt.Expr)
		assert.NotNil(t, assertStmt.Message)
		assert.Equal(t, "x > 0", assertStmt.Expr.NodeString)
		assert.Equal(t, "\"Value must be positive\"", assertStmt.Message.NodeString)
		assert.Equal(t, assertStmt.Message, assertStmt.GetMessage())
	})
}

func TestParseReturnStatement(t *testing.T) {
	t.Run("Return statement without result", func(t *testing.T) {
		// Create a return statement without result
		returnStmt := &model.ReturnStmt{}

		// Verify the structure
		assert.Nil(t, returnStmt.Result)
	})

	t.Run("Return statement with result", func(t *testing.T) {
		// Create a return statement with result
		returnStmt := &model.ReturnStmt{
			Result: &model.Expr{
				NodeString: "true",
			},
		}

		// Verify the structure
		assert.NotNil(t, returnStmt.Result)
		assert.Equal(t, "true", returnStmt.Result.NodeString)
		assert.Equal(t, "return true", returnStmt.ToString())
		assert.Equal(t, returnStmt.Result, returnStmt.GetResult())
	})
}

func TestParseBlockStatement(t *testing.T) {
	t.Run("Empty block statement", func(t *testing.T) {
		// Create an empty block statement
		blockStmt := &model.BlockStmt{}

		// Verify the structure
		assert.Empty(t, blockStmt.Stmts)
		assert.Equal(t, 0, blockStmt.GetNumStmt())
	})

	t.Run("Block statement with statements", func(t *testing.T) {
		// Create a block statement with statements
		blockStmt := &model.BlockStmt{
			Stmts: []model.Stmt{
				{NodeString: "int x = 10;"},
				{NodeString: "return x;"},
			},
		}

		// Verify the structure
		assert.Equal(t, 2, len(blockStmt.Stmts))
		assert.Equal(t, "int x = 10;", blockStmt.Stmts[0].NodeString)
		assert.Equal(t, "return x;", blockStmt.Stmts[1].NodeString)
		assert.Equal(t, 2, blockStmt.GetNumStmt())
		assert.Equal(t, blockStmt.Stmts[0], blockStmt.GetAStmt())
		assert.Equal(t, blockStmt.Stmts[0], blockStmt.GetStmt(0))
		assert.Equal(t, blockStmt.Stmts[1], blockStmt.GetLastStmt())
	})
}

func TestParseWhileStatement(t *testing.T) {
	t.Run("While statement with condition", func(t *testing.T) {
		// Create a while statement manually
		whileStmt := &model.WhileStmt{}
		
		// Set condition using ConditionalStmt field
		whileStmt.Condition = &model.Expr{
			NodeString: "i < 10",
		}

		// Verify the structure
		assert.NotNil(t, whileStmt.Condition)
		assert.Equal(t, "i < 10", whileStmt.Condition.NodeString)
	})
}

func TestParseDoWhileStatement(t *testing.T) {
	t.Run("Do-while statement with condition", func(t *testing.T) {
		// Create a do-while statement manually
		doWhileStmt := &model.DoStmt{}
		
		// Set condition using ConditionalStmt field
		doWhileStmt.Condition = &model.Expr{
			NodeString: "i < 10",
		}

		// Verify the structure
		assert.NotNil(t, doWhileStmt.Condition)
		assert.Equal(t, "i < 10", doWhileStmt.Condition.NodeString)
	})
}

func TestParseForLoopStatement(t *testing.T) {
	t.Run("For loop with init, condition, and increment", func(t *testing.T) {
		// Create a for loop statement manually
		forStmt := &model.ForStmt{}
		
		// Set fields
		forStmt.Init = &model.Expr{
			NodeString: "int i = 0",
		}
		forStmt.Condition = &model.Expr{
			NodeString: "i < 10",
		}
		forStmt.Increment = &model.Expr{
			NodeString: "i++",
		}

		// Verify the structure
		assert.NotNil(t, forStmt.Init)
		assert.NotNil(t, forStmt.Condition)
		assert.NotNil(t, forStmt.Increment)
		assert.Equal(t, "int i = 0", forStmt.Init.NodeString)
		assert.Equal(t, "i < 10", forStmt.Condition.NodeString)
		assert.Equal(t, "i++", forStmt.Increment.NodeString)
	})

	t.Run("For loop with partial components", func(t *testing.T) {
		// Create a for loop with only condition
		forStmt := &model.ForStmt{}
		
		// Set only condition
		forStmt.Condition = &model.Expr{
			NodeString: "i < 10",
		}

		// Verify the structure
		assert.Nil(t, forStmt.Init)
		assert.NotNil(t, forStmt.Condition)
		assert.Nil(t, forStmt.Increment)
		assert.Equal(t, "i < 10", forStmt.Condition.NodeString)
	})
}
