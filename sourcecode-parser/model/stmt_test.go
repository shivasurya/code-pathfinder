package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhileStmt(t *testing.T) {
	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		whileStmt := &WhileStmt{}
		assert.Equal(t, "WhileStmt", whileStmt.GetAPrimaryQlClass())
	})

	t.Run("GetCondition", func(t *testing.T) {
		condition := &Expr{NodeString: "x < 10"}
		whileStmt := &WhileStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: condition,
			},
		}
		assert.Equal(t, condition, whileStmt.GetCondition())
	})

	t.Run("GetHalsteadID", func(t *testing.T) {
		whileStmt := &WhileStmt{}
		assert.Equal(t, 0, whileStmt.GetHalsteadID())
	})

	t.Run("GetStmt", func(t *testing.T) {
		stmt := Stmt{NodeString: "x++"}
		whileStmt := &WhileStmt{
			ConditionalStmt: ConditionalStmt{
				Stmt: stmt,
			},
		}
		assert.Equal(t, stmt, whileStmt.GetStmt())
	})

	t.Run("GetPP", func(t *testing.T) {
		whileStmt := &WhileStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: &Expr{NodeString: "x < 10"},
				Stmt:      Stmt{NodeString: "x++"},
			},
		}
		expected := "while (x < 10) x++"
		assert.Equal(t, expected, whileStmt.GetPP())
	})

	t.Run("ToString", func(t *testing.T) {
		whileStmt := &WhileStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: &Expr{NodeString: "x < 10"},
				Stmt:      Stmt{NodeString: "x++"},
			},
		}
		expected := "while (x < 10) x++"
		assert.Equal(t, expected, whileStmt.ToString())
	})
}

func TestForStmt(t *testing.T) {
	t.Run("GetPrimaryQlClass", func(t *testing.T) {
		forStmt := &ForStmt{}
		assert.Equal(t, "ForStmt", forStmt.GetPrimaryQlClass())
	})

	t.Run("GetAnInit", func(t *testing.T) {
		init := &Expr{NodeString: "int i = 0"}
		forStmt := &ForStmt{Init: init}
		assert.Equal(t, init, forStmt.GetAnInit())
	})

	t.Run("GetCondition", func(t *testing.T) {
		condition := &Expr{NodeString: "i < 10"}
		forStmt := &ForStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: condition,
			},
		}
		assert.Equal(t, condition, forStmt.GetCondition())
	})

	t.Run("GetAnUpdate", func(t *testing.T) {
		increment := &Expr{NodeString: "i++"}
		forStmt := &ForStmt{Increment: increment}
		assert.Equal(t, increment, forStmt.GetAnUpdate())
	})

	t.Run("ToString", func(t *testing.T) {
		forStmt := &ForStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: &Expr{NodeString: "i < 10"},
				Stmt:      Stmt{NodeString: "System.out.println(i)"},
			},
			Init:      &Expr{NodeString: "int i = 0"},
			Increment: &Expr{NodeString: "i++"},
		}
		expected := "for (int i = 0; i < 10; i++) System.out.println(i)"
		assert.Equal(t, expected, forStmt.ToString())
	})
}

func TestIfStmt(t *testing.T) {
	t.Run("GetCondition", func(t *testing.T) {
		condition := &Expr{NodeString: "x > 0"}
		ifStmt := &IfStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: condition,
			},
		}
		assert.Equal(t, condition, ifStmt.GetCondition())
	})

	t.Run("GetElse", func(t *testing.T) {
		elseStmt := Stmt{NodeString: "return false"}
		ifStmt := &IfStmt{
			Else: elseStmt,
		}
		assert.Equal(t, &elseStmt, ifStmt.GetElse())
	})

	t.Run("GetThen", func(t *testing.T) {
		thenStmt := Stmt{NodeString: "return true"}
		ifStmt := &IfStmt{
			Then: thenStmt,
		}
		assert.Equal(t, &thenStmt, ifStmt.GetThen())
	})

	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		ifStmt := &IfStmt{}
		assert.Equal(t, "ifStmt", ifStmt.GetAPrimaryQlClass())
	})

	t.Run("GetPP and ToString", func(t *testing.T) {
		ifStmt := &IfStmt{
			ConditionalStmt: ConditionalStmt{
				Condition: &Expr{NodeString: "x > 0"},
			},
			Then: Stmt{NodeString: "return true"},
			Else: Stmt{NodeString: "return false"},
		}
		expected := "if (x > 0) return true else return false"
		assert.Equal(t, expected, ifStmt.GetPP())
		assert.Equal(t, expected, ifStmt.ToString())
	})
}

func TestContinueStmt(t *testing.T) {
	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		continueStmt := &ContinueStmt{}
		assert.Equal(t, "ContinueStmt", continueStmt.GetAPrimaryQlClass())
	})

	t.Run("GetHalsteadID", func(t *testing.T) {
		continueStmt := &ContinueStmt{}
		assert.Equal(t, 0, continueStmt.GetHalsteadID())
	})

	t.Run("GetPP with label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "outerLoop",
		}
		expected := "continue (outerLoop)"
		assert.Equal(t, expected, continueStmt.GetPP())
	})

	t.Run("GetPP without label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "",
		}
		expected := "continue ()"
		assert.Equal(t, expected, continueStmt.GetPP())
	})

	t.Run("ToString with label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "innerLoop",
		}
		expected := "continue (innerLoop)"
		assert.Equal(t, expected, continueStmt.ToString())
	})

	t.Run("ToString without label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "",
		}
		expected := "continue ()"
		assert.Equal(t, expected, continueStmt.ToString())
	})

	t.Run("hasLabel with label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "loop1",
		}
		assert.True(t, continueStmt.hasLabel())
	})

	t.Run("hasLabel without label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "",
		}
		assert.False(t, continueStmt.hasLabel())
	})

	t.Run("GetLabel with label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "loop2",
		}
		assert.Equal(t, "loop2", continueStmt.GetLabel())
	})

	t.Run("GetLabel without label", func(t *testing.T) {
		continueStmt := &ContinueStmt{
			Label: "",
		}
		assert.Equal(t, "", continueStmt.GetLabel())
	})
}

func TestYieldStmt(t *testing.T) {
	t.Run("ToString with non-empty value", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "42"},
		}
		assert.Equal(t, "yield 42", yieldStmt.ToString())
	})

	t.Run("ToString with empty value", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: ""},
		}
		assert.Equal(t, "yield ", yieldStmt.ToString())
	})

	t.Run("ToString with complex expression", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "a + b * c"},
		}
		assert.Equal(t, "yield a + b * c", yieldStmt.ToString())
	})

	t.Run("ToString with string literal", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "\"hello world\""},
		}
		assert.Equal(t, "yield \"hello world\"", yieldStmt.ToString())
	})
}

func TestYieldStmt_GetValue(t *testing.T) {
	t.Run("GetValue with non-nil value", func(t *testing.T) {
		expr := &Expr{NodeString: "42"}
		yieldStmt := &YieldStmt{
			Value: expr,
		}
		assert.Equal(t, expr, yieldStmt.GetValue())
	})

	t.Run("GetValue with nil value", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: nil,
		}
		assert.Nil(t, yieldStmt.GetValue())
	})

	t.Run("GetValue with complex expression", func(t *testing.T) {
		expr := &Expr{NodeString: "foo() + bar(x, y)"}
		yieldStmt := &YieldStmt{
			Value: expr,
		}
		assert.Equal(t, expr, yieldStmt.GetValue())
	})

	t.Run("GetValue preserves expression reference", func(t *testing.T) {
		expr := &Expr{NodeString: "someValue"}
		yieldStmt := &YieldStmt{
			Value: expr,
		}
		retrievedExpr := yieldStmt.GetValue()
		expr.NodeString = "modifiedValue"
		assert.Equal(t, "modifiedValue", retrievedExpr.NodeString)
	})
}

func TestYieldStmt_GetHalsteadID(t *testing.T) {
	t.Run("Returns zero for empty yield statement", func(t *testing.T) {
		yieldStmt := &YieldStmt{}
		assert.Equal(t, 0, yieldStmt.GetHalsteadID())
	})

	t.Run("Returns zero for yield with simple value", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "42"},
		}
		assert.Equal(t, 0, yieldStmt.GetHalsteadID())
	})

	t.Run("Returns zero for yield with complex expression", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "a + b * c"},
		}
		assert.Equal(t, 0, yieldStmt.GetHalsteadID())
	})

	t.Run("Returns zero for yield with method call", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "calculateValue()"},
		}
		assert.Equal(t, 0, yieldStmt.GetHalsteadID())
	})
}

func TestYieldStmt_GetPP(t *testing.T) {
	t.Run("GetPP with numeric value", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "42"},
		}
		assert.Equal(t, "yield 42", yieldStmt.GetPP())
	})

	t.Run("GetPP with method call", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "getValue()"},
		}
		assert.Equal(t, "yield getValue()", yieldStmt.GetPP())
	})

	t.Run("GetPP with complex expression", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "x + y * (z - 1)"},
		}
		assert.Equal(t, "yield x + y * (z - 1)", yieldStmt.GetPP())
	})

	t.Run("GetPP with empty expression", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: ""},
		}
		assert.Equal(t, "yield ", yieldStmt.GetPP())
	})

	t.Run("GetPP with string literal", func(t *testing.T) {
		yieldStmt := &YieldStmt{
			Value: &Expr{NodeString: "\"test string\""},
		}
		assert.Equal(t, "yield \"test string\"", yieldStmt.GetPP())
	})
}

func TestAssertStmt_GetMessage(t *testing.T) {
	t.Run("GetMessage with non-nil message", func(t *testing.T) {
		message := &Expr{NodeString: "Expected value to be positive"}
		assertStmt := &AssertStmt{
			Message: message,
		}
		assert.Equal(t, message, assertStmt.GetMessage())
	})

	t.Run("GetMessage with nil message", func(t *testing.T) {
		assertStmt := &AssertStmt{
			Message: nil,
		}
		assert.Nil(t, assertStmt.GetMessage())
	})

	t.Run("GetMessage with complex expression", func(t *testing.T) {
		message := &Expr{NodeString: "String.format(\"Value %d should be greater than zero\", value)"}
		assertStmt := &AssertStmt{
			Message: message,
		}
		assert.Equal(t, message, assertStmt.GetMessage())
	})

	t.Run("GetMessage preserves expression reference", func(t *testing.T) {
		message := &Expr{NodeString: "Initial message"}
		assertStmt := &AssertStmt{
			Message: message,
		}
		retrievedMessage := assertStmt.GetMessage()
		message.NodeString = "Modified message"
		assert.Equal(t, "Modified message", retrievedMessage.NodeString)
	})
}

func TestReturnStmt_GetPP(t *testing.T) {
	t.Run("GetPP with numeric value", func(t *testing.T) {
		returnStmt := &ReturnStmt{
			Result: &Expr{NodeString: "42"},
		}
		assert.Equal(t, "return 42", returnStmt.GetPP())
	})

	t.Run("GetPP with string literal", func(t *testing.T) {
		returnStmt := &ReturnStmt{
			Result: &Expr{NodeString: "\"hello world\""},
		}
		assert.Equal(t, "return \"hello world\"", returnStmt.GetPP())
	})

	t.Run("GetPP with method call", func(t *testing.T) {
		returnStmt := &ReturnStmt{
			Result: &Expr{NodeString: "getValue()"},
		}
		assert.Equal(t, "return getValue()", returnStmt.GetPP())
	})

	t.Run("GetPP with complex expression", func(t *testing.T) {
		returnStmt := &ReturnStmt{
			Result: &Expr{NodeString: "x + y * (z - 1)"},
		}
		assert.Equal(t, "return x + y * (z - 1)", returnStmt.GetPP())
	})

	t.Run("GetPP with empty expression", func(t *testing.T) {
		returnStmt := &ReturnStmt{
			Result: &Expr{NodeString: ""},
		}
		assert.Equal(t, "return ", returnStmt.GetPP())
	})

	t.Run("GetPP with boolean expression", func(t *testing.T) {
		returnStmt := &ReturnStmt{
			Result: &Expr{NodeString: "x > 0 && y < 10"},
		}
		assert.Equal(t, "return x > 0 && y < 10", returnStmt.GetPP())
	})
}

func TestBlockStmt(t *testing.T) {
	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		blockStmt := &BlockStmt{}
		assert.Equal(t, "BlockStmt", blockStmt.GetAPrimaryQlClass())
	})

	t.Run("GetHalsteadID", func(t *testing.T) {
		blockStmt := &BlockStmt{}
		assert.Equal(t, 0, blockStmt.GetHalsteadID())
	})

	t.Run("GetStmt with valid index", func(t *testing.T) {
		stmt1 := Stmt{NodeString: "x = 1"}
		stmt2 := Stmt{NodeString: "y = 2"}
		blockStmt := &BlockStmt{
			Stmts: []Stmt{stmt1, stmt2},
		}
		assert.Equal(t, stmt2, blockStmt.GetStmt(1))
	})

	t.Run("GetAStmt with non-empty block", func(t *testing.T) {
		stmt1 := Stmt{NodeString: "x = 1"}
		stmt2 := Stmt{NodeString: "y = 2"}
		blockStmt := &BlockStmt{
			Stmts: []Stmt{stmt1, stmt2},
		}
		assert.Equal(t, stmt1, blockStmt.GetAStmt())
	})

	t.Run("GetNumStmt with multiple statements", func(t *testing.T) {
		blockStmt := &BlockStmt{
			Stmts: []Stmt{
				{NodeString: "x = 1"},
				{NodeString: "y = 2"},
				{NodeString: "z = 3"},
			},
		}
		assert.Equal(t, 3, blockStmt.GetNumStmt())
	})

	t.Run("GetNumStmt with empty block", func(t *testing.T) {
		blockStmt := &BlockStmt{
			Stmts: []Stmt{},
		}
		assert.Equal(t, 0, blockStmt.GetNumStmt())
	})

	t.Run("GetLastStmt with multiple statements", func(t *testing.T) {
		stmt1 := Stmt{NodeString: "x = 1"}
		stmt2 := Stmt{NodeString: "y = 2"}
		stmt3 := Stmt{NodeString: "z = 3"}
		blockStmt := &BlockStmt{
			Stmts: []Stmt{stmt1, stmt2, stmt3},
		}
		assert.Equal(t, stmt3, blockStmt.GetLastStmt())
	})
}
