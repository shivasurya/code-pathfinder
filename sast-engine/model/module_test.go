package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModule(t *testing.T) {
	cu := &CompilationUnit{}
	directive := Directive{Directive: "requires java.base"}
	module := &Module{
		Cu:     cu,
		Di:     directive,
		Name:   "com.example.module",
		isOpen: true,
	}

	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		assert.Equal(t, "Module", module.GetAPrimaryQlClass())
	})

	t.Run("GetACompilationUnit", func(t *testing.T) {
		assert.Equal(t, cu, module.GetACompilationUnit())
	})

	t.Run("GetName", func(t *testing.T) {
		assert.Equal(t, "com.example.module", module.GetName())
	})

	t.Run("ToString", func(t *testing.T) {
		assert.Equal(t, "com.example.module", module.ToString())
	})

	t.Run("GetDirective", func(t *testing.T) {
		assert.Equal(t, &directive, module.GetDirective())
	})

	t.Run("IsOpen", func(t *testing.T) {
		assert.True(t, module.IsOpen())
	})
}

func TestModuleWithEmptyValues(t *testing.T) {
	module := &Module{}

	t.Run("Empty module properties", func(t *testing.T) {
		assert.Nil(t, module.GetACompilationUnit())
		assert.Empty(t, module.GetName())
		assert.Empty(t, module.ToString())
		assert.False(t, module.IsOpen())
	})
}

func TestDirective(t *testing.T) {
	t.Run("ToString with value", func(t *testing.T) {
		directive := &Directive{Directive: "requires transitive java.sql"}
		assert.Equal(t, "requires transitive java.sql", directive.ToString())
	})

	t.Run("ToString empty", func(t *testing.T) {
		directive := &Directive{}
		assert.Empty(t, directive.ToString())
	})
}
