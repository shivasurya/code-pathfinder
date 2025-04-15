package model

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestVariable(t *testing.T) {
	assigned := []string{"10", "20"}
	v := NewVariable("x", "int", "local", "10", assigned, "line 1")

	assert.Equal(t, "x", v.Name)
	assert.Equal(t, "int", v.Type)
	assert.Equal(t, "local", v.Scope)
	assert.Equal(t, "10", v.Initializer)
	assert.Equal(t, assigned, v.GetAnAssignedValue())
	assert.Equal(t, "10", v.GetInitializer())
	assert.Equal(t, "int", v.GetType())
	assert.Equal(t, "int x = 10;", v.PP())
}

func TestLocalScopeVariable(t *testing.T) {
	lsv := NewLocalScopeVariable("y", "string", "parameter", "MyFunc", "line 2")

	assert.Equal(t, "y", lsv.Name)
	assert.Equal(t, "string", lsv.Type)
	assert.Equal(t, "parameter", lsv.Scope)
	assert.Equal(t, "MyFunc", lsv.DeclaredIn)
	assert.Equal(t, "line 2", lsv.SourceDeclaration)
	assert.Equal(t, "MyFunc", lsv.GetCallable())
}

func TestLocalVariableDecl(t *testing.T) {
	lvd := NewLocalVariableDecl(
		"z", "float64", "Compute", "float64 z = 3.14;", "3.14", "block1", "line 3",
	)

	assert.Equal(t, "z", lvd.Name)
	assert.Equal(t, "float64", lvd.Type)
	assert.Equal(t, "Compute", lvd.Callable)
	assert.Equal(t, "float64 z = 3.14;", lvd.DeclExpr)
	assert.Equal(t, "3.14", lvd.Initializer)
	assert.Equal(t, "block1", lvd.ParentScope)
	assert.Equal(t, "line 3", lvd.SourceDeclaration)

	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		assert.Equal(t, "LocalVariableDecl", lvd.GetAPrimaryQlClass())
	})
	t.Run("GetCallable", func(t *testing.T) {
		assert.Equal(t, "Compute", lvd.GetCallable())
	})
	t.Run("GetDeclExpr", func(t *testing.T) {
		assert.Equal(t, "float64 z = 3.14;", lvd.GetDeclExpr())
	})
	t.Run("GetEnclosingCallable", func(t *testing.T) {
		assert.Equal(t, "Compute", lvd.GetEnclosingCallable())
	})
	t.Run("GetInitializer", func(t *testing.T) {
		assert.Equal(t, "3.14", lvd.GetInitializer())
	})
	t.Run("GetParent", func(t *testing.T) {
		assert.Equal(t, "block1", lvd.GetParent())
	})
	t.Run("GetType", func(t *testing.T) {
		assert.Equal(t, "float64", lvd.GetType())
	})
	t.Run("ToString with initializer", func(t *testing.T) {
		assert.Equal(t, "float64 z = 3.14;", lvd.ToString())
	})

	lvdNoInit := NewLocalVariableDecl("a", "bool", "Check", "bool a;", "", "block2", "line 4")
	t.Run("ToString without initializer", func(t *testing.T) {
		assert.Equal(t, "bool a;", lvdNoInit.ToString())
	})
}
