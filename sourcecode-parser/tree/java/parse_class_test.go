package java

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

// TestClassInstanceExprStructure tests the structure and behavior of the ClassInstanceExpr model
func TestClassInstanceExprStructure(t *testing.T) {
	t.Run("Basic class instance expression", func(t *testing.T) {
		// Create a class instance expression manually
		expr := &model.ClassInstanceExpr{
			ClassName: "java.util.ArrayList",
			Args:      []*model.Expr{},
		}

		// Verify the structure
		assert.Equal(t, "java.util.ArrayList", expr.ClassName)
		assert.Empty(t, expr.Args)

		// Test the GetClassName method
		assert.Equal(t, "java.util.ArrayList", expr.GetClassName())

		// Test the GetNumArgs method
		assert.Equal(t, 0, expr.GetNumArgs())
	})

	t.Run("Class instance expression with arguments", func(t *testing.T) {
		// Create a class instance expression with arguments
		expr := &model.ClassInstanceExpr{
			ClassName: "java.util.HashMap",
			Args: []*model.Expr{
				{NodeString: "10"},
				{NodeString: "0.75f"},
			},
		}

		// Verify the structure
		assert.Equal(t, "java.util.HashMap", expr.ClassName)
		assert.Len(t, expr.Args, 2)
		assert.Equal(t, "10", expr.Args[0].NodeString)
		assert.Equal(t, "0.75f", expr.Args[1].NodeString)

		// Test the GetArgs method
		args := expr.GetArgs()
		assert.Len(t, args, 2)

		// Test the GetArg method
		assert.Equal(t, "10", expr.GetArg(0).NodeString)
		assert.Equal(t, "0.75f", expr.GetArg(1).NodeString)

		// Test the GetNumArgs method
		assert.Equal(t, 2, expr.GetNumArgs())
	})
}
