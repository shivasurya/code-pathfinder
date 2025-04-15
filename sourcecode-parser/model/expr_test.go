package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinaryExpr(t *testing.T) {
	leftExpr := &Expr{Kind: 0, NodeString: "left"}
	rightExpr := &Expr{Kind: 0, NodeString: "right"}
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

	t.Run("GetLeftOperandString", func(t *testing.T) {
		assert.Equal(t, "left", binaryExpr.GetLeftOperandString())
	})

	t.Run("GetRightOperandString", func(t *testing.T) {
		assert.Equal(t, "right", binaryExpr.GetRightOperandString())
	})

	t.Run("ToString", func(t *testing.T) {
		str := binaryExpr.ToString()
		assert.Contains(t, str, "BinaryExpr(")
		assert.Contains(t, str, "+")
		assert.Contains(t, str, "left")
		assert.Contains(t, str, "right")
	})
}

func TestAddExpr(t *testing.T) {
	addExpr := &AddExpr{
		BinaryExpr: BinaryExpr{Op: "+"},
		op:         "+",
	}

	assert.Equal(t, "+", addExpr.GetOp())
}

func TestOtherBinaryExprTypes_GetOp(t *testing.T) {
	types := []struct {
		name string
		expr interface{ GetOp() string }
		expected string
	}{
		{"SubExpr", &SubExpr{op: "-"}, "-"},
		{"DivExpr", &DivExpr{op: "/"}, "/"},
		{"MulExpr", &MulExpr{op: "*"}, "*"},
		{"RemExpr", &RemExpr{op: "%"}, "%"},
		{"EqExpr", &EqExpr{op: "=="}, "=="},
		{"NEExpr", &NEExpr{op: "!="}, "!="},
		{"GTExpr", &GTExpr{op: ">"}, ">"},
		{"GEExpr", &GEExpr{op: ">="}, ">="},
		{"LTExpr", &LTExpr{op: "<"}, "<"},
		{"LEExpr", &LEExpr{op: "<="}, "<="},
		{"AndBitwiseExpr", &AndBitwiseExpr{op: "&"}, "&"},
		{"OrBitwiseExpr", &OrBitwiseExpr{op: "|"}, "|"},
		{"LeftShiftExpr", &LeftShiftExpr{op: "<<"}, "<<"},
		{"RightShiftExpr", &RightShiftExpr{op: ">>"}, ">>"},
		{"UnsignedRightShiftExpr", &UnsignedRightShiftExpr{op: ">>>"}, ">>>"},
		{"AndLogicalExpr", &AndLogicalExpr{op: "&&"}, "&&"},
		{"OrLogicalExpr", &OrLogicalExpr{op: "||"}, "||"},
	}
	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.expr.GetOp())
		})
	}
}

func TestComparisonExpr(t *testing.T) {
	compExpr := &ComparisonExpr{}

	assert.Nil(t, compExpr.GetGreaterThanOperand())
	assert.Nil(t, compExpr.GetLessThanOperand())
	assert.True(t, compExpr.IsStrict())
}

func TestExpr(t *testing.T) {
	expr := &Expr{Kind: 42, NodeString: "foo"}

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

	t.Run("String", func(t *testing.T) {
		assert.Equal(t, "Expr(foo)", expr.String())
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
			{"Normal class name", &ClassInstanceExpr{ClassName: "MyClass"}, "MyClass"},
			{"Empty class name", &ClassInstanceExpr{ClassName: ""}, ""},
			{"Class name with special characters", &ClassInstanceExpr{ClassName: "My_Class$123"}, "My_Class$123"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.expr.GetClassName()
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}
