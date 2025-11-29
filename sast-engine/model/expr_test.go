package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinaryExpr(t *testing.T) {
	leftExpr := &Expr{Kind: 0}
	rightExpr := &Expr{Kind: 0}
	binaryExpr := &BinaryExpr{
		Op:           "+",
		LeftOperand:  leftExpr,
		RightOperand: rightExpr,
	}

	t.Run("GetLeftOperand", func(t *testing.T) {
		assert.Equal(t, leftExpr, binaryExpr.GetLeftOperand())
	})

	t.Run("GetRightOperand", func(t *testing.T) {
		assert.Equal(t, rightExpr, binaryExpr.GetRightOperand())
	})

	t.Run("GetOp", func(t *testing.T) {
		assert.Equal(t, "+", binaryExpr.GetOp())
	})

	t.Run("GetKind", func(t *testing.T) {
		assert.Equal(t, 1, binaryExpr.GetKind())
	})

	t.Run("GetAnOperand", func(t *testing.T) {
		assert.Equal(t, leftExpr, binaryExpr.GetAnOperand())
	})

	t.Run("HasOperands", func(t *testing.T) {
		assert.True(t, binaryExpr.HasOperands(leftExpr, rightExpr))
		assert.False(t, binaryExpr.HasOperands(rightExpr, leftExpr))
	})
}

func TestAddExpr(t *testing.T) {
	addExpr := &AddExpr{
		BinaryExpr: BinaryExpr{Op: "+"},
		op:         "+",
	}

	assert.Equal(t, "+", addExpr.GetOp())
}

func TestComparisonExpr(t *testing.T) {
	compExpr := &ComparisonExpr{}

	assert.Nil(t, compExpr.GetGreaterThanOperand())
	assert.Nil(t, compExpr.GetLessThanOperand())
	assert.True(t, compExpr.IsStrict())
}

func TestExpr(t *testing.T) {
	expr := &Expr{Kind: 42}

	t.Run("GetAChildExpr", func(t *testing.T) {
		assert.Equal(t, expr, expr.GetAChildExpr())
	})

	t.Run("GetChildExpr", func(t *testing.T) {
		assert.Equal(t, expr, expr.GetChildExpr(0))
	})

	t.Run("GetNumChildExpr", func(t *testing.T) {
		assert.Equal(t, int64(1), expr.GetNumChildExpr())
	})

	t.Run("GetKind", func(t *testing.T) {
		assert.Equal(t, 42, expr.GetKind())
	})
}

func TestExprParent(t *testing.T) {
	parent := &ExprParent{}

	assert.Nil(t, parent.GetAChildExpr())
	assert.Nil(t, parent.GetChildExpr(0))
	assert.Equal(t, int64(0), parent.GetNumChildExpr())
}

func TestClassInstanceExpr(t *testing.T) {
	t.Run("GetClassName", func(t *testing.T) {
		testCases := []struct {
			name     string
			expr     *ClassInstanceExpr
			expected string
		}{
			{
				name:     "Normal class name",
				expr:     &ClassInstanceExpr{ClassName: "MyClass"},
				expected: "MyClass",
			},
			{
				name:     "Empty class name",
				expr:     &ClassInstanceExpr{ClassName: ""},
				expected: "",
			},
			{
				name:     "Class name with special characters",
				expr:     &ClassInstanceExpr{ClassName: "My_Class$123"},
				expected: "My_Class$123",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.expr.GetClassName()
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}
