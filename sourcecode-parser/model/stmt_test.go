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
