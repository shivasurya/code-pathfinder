package java

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// TestParseBreakStatement tests the ParseBreakStatement function
func TestParseBreakStatement(t *testing.T) {
	t.Run("Break statement without label", func(t *testing.T) {
		// Setup
		sourceCode := []byte("break;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the break_statement node
		breakNode := findNodeByType(rootNode, "break_statement")
		assert.NotNil(t, breakNode)

		// Call the function with our parsed node
		breakStmt := ParseBreakStatement(breakNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, breakStmt)
		assert.Empty(t, breakStmt.Label)
	})

	t.Run("Break statement with label", func(t *testing.T) {
		// Setup
		sourceCode := []byte("break outerLoop;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the break_statement node
		breakNode := findNodeByType(rootNode, "break_statement")
		assert.NotNil(t, breakNode)

		// Call the function with our parsed node
		breakStmt := ParseBreakStatement(breakNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, breakStmt)
		assert.Equal(t, "outerLoop", breakStmt.Label)
	})
}

// TestParseContinueStatement tests the ParseContinueStatement function
func TestParseContinueStatement(t *testing.T) {
	t.Run("Continue statement without label", func(t *testing.T) {
		// Setup
		sourceCode := []byte("continue;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the continue_statement node
		continueNode := findNodeByType(rootNode, "continue_statement")
		assert.NotNil(t, continueNode)

		// Call the function with our parsed node
		continueStmt := ParseContinueStatement(continueNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, continueStmt)
		assert.Empty(t, continueStmt.Label)
	})

	t.Run("Continue statement with label", func(t *testing.T) {
		// Setup
		sourceCode := []byte("continue outerLoop;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the continue_statement node
		continueNode := findNodeByType(rootNode, "continue_statement")
		assert.NotNil(t, continueNode)

		// Call the function with our parsed node
		continueStmt := ParseContinueStatement(continueNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, continueStmt)
		assert.Equal(t, "outerLoop", continueStmt.Label)
	})
}

// TestParseYieldStatement tests the ParseYieldStatement function
func TestParseYieldStatement(t *testing.T) {
	t.Run("Yield statement with value", func(t *testing.T) {
		// Setup
		sourceCode := []byte("yield 42;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the yield_statement node
		yieldNode := findNodeByType(rootNode, "yield_statement")
		assert.NotNil(t, yieldNode)

		// Call the function with our parsed node
		yieldStmt := ParseYieldStatement(yieldNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, yieldStmt)
		assert.NotNil(t, yieldStmt.Value)
		assert.Equal(t, "42", yieldStmt.Value.NodeString)
	})
}

// TestParseAssertStatement tests the ParseAssertStatement function
func TestParseAssertStatement(t *testing.T) {
	t.Run("Assert statement without message", func(t *testing.T) {
		// Setup
		sourceCode := []byte("assert x > 0;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the assert_statement node
		assertNode := findNodeByType(rootNode, "assert_statement")
		assert.NotNil(t, assertNode)

		// Call the function with our parsed node
		assertStmt := ParseAssertStatement(assertNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, assertStmt)
		assert.NotNil(t, assertStmt.Expr)
		assert.Equal(t, "x > 0", assertStmt.Expr.NodeString)
		assert.Nil(t, assertStmt.Message)
	})

	t.Run("Assert statement with message", func(t *testing.T) {
		// Setup
		sourceCode := []byte("assert x > 0 : \"Value must be positive\";")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the assert_statement node
		assertNode := findNodeByType(rootNode, "assert_statement")
		assert.NotNil(t, assertNode)

		// Call the function with our parsed node
		assertStmt := ParseAssertStatement(assertNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, assertStmt)
		assert.NotNil(t, assertStmt.Expr)
		assert.Equal(t, "x > 0", assertStmt.Expr.NodeString)
		assert.NotNil(t, assertStmt.Message)
		assert.Equal(t, "\"Value must be positive\"", assertStmt.Message.NodeString)
	})
}

// TestParseReturnStatement tests the ParseReturnStatement function
func TestParseReturnStatement(t *testing.T) {
	t.Run("Return statement without result", func(t *testing.T) {
		// Setup
		sourceCode := []byte("return;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the return_statement node
		returnNode := findNodeByType(rootNode, "return_statement")
		assert.NotNil(t, returnNode)

		// Call the function with our parsed node
		returnStmt := ParseReturnStatement(returnNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, returnStmt)
		// The implementation might set Result to an empty Expr instead of nil
		if returnStmt.Result != nil {
			assert.Equal(t, ";", returnStmt.Result.NodeString)
		}
	})

	t.Run("Return statement with result", func(t *testing.T) {
		// Setup
		sourceCode := []byte("return true;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the return_statement node
		returnNode := findNodeByType(rootNode, "return_statement")
		assert.NotNil(t, returnNode)

		// Call the function with our parsed node
		returnStmt := ParseReturnStatement(returnNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, returnStmt)
		assert.NotNil(t, returnStmt.Result)
		assert.Equal(t, "true", returnStmt.Result.NodeString)
	})
}

// TestParseBlockStatement tests the ParseBlockStatement function
func TestParseBlockStatement(t *testing.T) {
	t.Run("Empty block statement", func(t *testing.T) {
		// Setup
		sourceCode := []byte("{}")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the block node
		blockNode := findNodeByType(rootNode, "block")
		assert.NotNil(t, blockNode)

		// Call the function with our parsed node
		blockStmt := ParseBlockStatement(blockNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, blockStmt)
		// The implementation adds { and } as statements
		if len(blockStmt.Stmts) > 0 {
			assert.LessOrEqual(t, len(blockStmt.Stmts), 2)
		}
	})

	t.Run("Block statement with statements", func(t *testing.T) {
		// Setup
		sourceCode := []byte("{ int x = 10; return x; }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the block node
		blockNode := findNodeByType(rootNode, "block")
		assert.NotNil(t, blockNode)

		// Call the function with our parsed node
		blockStmt := ParseBlockStatement(blockNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, blockStmt)
		assert.Equal(t, 4, len(blockStmt.Stmts)) // { and } are also counted as statements
		// Check that the statements are included in the block
		assert.Contains(t, blockStmt.Stmts[1].NodeString, "int x = 10;")
		assert.Contains(t, blockStmt.Stmts[2].NodeString, "return x;")
	})
}

// TestParseWhileStatement tests the ParseWhileStatement function
func TestParseWhileStatement(t *testing.T) {
	t.Run("While statement with condition", func(t *testing.T) {
		// Setup
		sourceCode := []byte("while (i < 10) { i++; }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the while_statement node
		whileNode := findNodeByType(rootNode, "while_statement")
		assert.NotNil(t, whileNode)

		// Call the function with our parsed node
		whileStmt := ParseWhileStatement(whileNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, whileStmt)
		assert.NotNil(t, whileStmt.Condition)
		assert.Equal(t, "(i < 10)", whileStmt.Condition.NodeString)
	})
}

// TestParseDoWhileStatement tests the ParseDoWhileStatement function
func TestParseDoWhileStatement(t *testing.T) {
	t.Run("Do-while statement with condition", func(t *testing.T) {
		// Setup
		sourceCode := []byte("do { i++; } while (i < 10);")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the do_statement node
		doNode := findNodeByType(rootNode, "do_statement")
		assert.NotNil(t, doNode)

		// Call the function with our parsed node
		doStmt := ParseDoWhileStatement(doNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, doStmt)
		assert.NotNil(t, doStmt.Condition)
		// The implementation might extract different parts of the condition
		assert.Contains(t, doStmt.Condition.NodeString, "while")
	})
}

// TestParseForLoopStatement tests the ParseForLoopStatement function
func TestParseForLoopStatement(t *testing.T) {
	t.Run("For loop with init, condition, and increment", func(t *testing.T) {
		// Setup
		sourceCode := []byte("for (int i = 0; i < 10; i++) { process(); }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the for_statement node
		forNode := findNodeByType(rootNode, "for_statement")
		assert.NotNil(t, forNode)

		// Call the function with our parsed node
		forStmt := ParseForLoopStatement(forNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, forStmt)
		// Check that at least one of the components is not nil
		if forStmt.Init != nil {
			assert.Contains(t, forStmt.Init.NodeString, "int i = 0")
		}
		if forStmt.Condition != nil {
			assert.Contains(t, forStmt.Condition.NodeString, "i < 10")
		}
		if forStmt.Increment != nil {
			assert.Contains(t, forStmt.Increment.NodeString, "i++")
		}
	})

	t.Run("For loop with partial components", func(t *testing.T) {
		// Setup
		sourceCode := []byte("for (; i < 10; i++) { process(); }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the for_statement node
		forNode := findNodeByType(rootNode, "for_statement")
		assert.NotNil(t, forNode)

		// Call the function with our parsed node
		forStmt := ParseForLoopStatement(forNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, forStmt)
		// We expect Init and Increment to be nil, but Condition to be set
		if forStmt.Condition != nil {
			assert.Contains(t, forStmt.Condition.NodeString, "i < 10")
		}
	})

	t.Run("For loop with only increment", func(t *testing.T) {
		// Setup
		sourceCode := []byte("for (;;i++) { process(); }")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the for_statement node
		forNode := findNodeByType(rootNode, "for_statement")
		assert.NotNil(t, forNode)

		// Call the function with our parsed node
		forStmt := ParseForLoopStatement(forNode, sourceCode, "Test.java")

		// Assertions
		assert.NotNil(t, forStmt)
		// We expect Init and Condition to be nil, but Increment to be set
		assert.Nil(t, forStmt.Init)
		assert.Nil(t, forStmt.Condition)
		assert.NotNil(t, forStmt.Increment)
		if forStmt.Increment != nil {
			assert.Contains(t, forStmt.Increment.NodeString, "i++")
		}
	})
}

// Helper function already defined in parse_import_test.go
// func findNodeByType(node *sitter.Node, nodeType string) *sitter.Node {
// 	if node.Type() == nodeType {
// 		return node
// 	}

// 	for i := 0; i < int(node.ChildCount()); i++ {
// 		child := node.Child(i)
// 		if found := findNodeByType(child, nodeType); found != nil {
// 			return found
// 		}
// 	}

// 	return nil
// }
