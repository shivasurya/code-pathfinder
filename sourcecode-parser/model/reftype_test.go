package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModifiable(t *testing.T) {
	t.Run("NewModifiable", func(t *testing.T) {
		mods := []string{"public", "static", "final"}
		m := NewModifiable(mods)
		assert.Equal(t, mods, m.Modifiers)
	})

	t.Run("GetAModifier", func(t *testing.T) {
		mods := []string{"public", "static"}
		m := NewModifiable(mods)
		assert.Equal(t, mods, m.GetAModifier())
	})

	t.Run("HasModifier", func(t *testing.T) {
		m := NewModifiable([]string{"public", "static"})
		assert.True(t, m.HasModifier("public"))
		assert.False(t, m.HasModifier("final"))
	})

	t.Run("HasNoModifier", func(t *testing.T) {
		m1 := NewModifiable([]string{})
		assert.True(t, m1.HasNoModifier())

		m2 := NewModifiable([]string{"public"})
		assert.False(t, m2.HasNoModifier())
	})

	t.Run("Modifier Checks", func(t *testing.T) {
		m := NewModifiable([]string{"public", "static", "final", "abstract", "default",
			"native", "private", "protected", "strictfp", "synchronized",
			"transient", "volatile"})

		assert.True(t, m.IsPublic())
		assert.True(t, m.IsStatic())
		assert.True(t, m.IsFinal())
		assert.True(t, m.IsAbstract())
		assert.True(t, m.IsDefault())
		assert.True(t, m.IsNative())
		assert.True(t, m.IsPrivate())
		assert.True(t, m.IsProtected())
		assert.True(t, m.IsStrictfp())
		assert.True(t, m.IsSynchronized())
		assert.True(t, m.IsTransient())
		assert.True(t, m.IsVolatile())
	})

	t.Run("ToString", func(t *testing.T) {
		m1 := NewModifiable([]string{})
		assert.Equal(t, "No Modifiers", m1.ToString())

		m2 := NewModifiable([]string{"public", "static"})
		assert.Equal(t, "public static", m2.ToString())
	})
}

func TestRefType(t *testing.T) {
	resolver := &TypeResolver{
		TypeHierarchy: map[string][]string{
			"Parent": {"Child"},
		},
	}

	refType := NewRefType(
		"com.example.Test",
		"com.example",
		"Test.java",
		true,
		[]string{"Parent"},
		[]string{"field1"},
		[]Method{{Name: "method1", Parameters: []string{"param1"}}},
		[]Method{{Name: "constructor1", Parameters: []string{}}},
		[]string{"NestedType"},
		"",
		false,
		"Lcom/example/Test;",
		resolver,
	)

	t.Run("GetQualifiedName", func(t *testing.T) {
		assert.Equal(t, "com.example.Test", refType.GetQualifiedName())
	})

	t.Run("GetPackage", func(t *testing.T) {
		assert.Equal(t, "com.example", refType.GetPackage())
	})

	t.Run("HasSupertype", func(t *testing.T) {
		assert.True(t, refType.HasSupertype("Parent"))
		assert.False(t, refType.HasSupertype("Unknown"))
	})

	t.Run("DeclaresField", func(t *testing.T) {
		assert.True(t, refType.DeclaresField("field1"))
		assert.False(t, refType.DeclaresField("field2"))
	})

	t.Run("DeclaresMethod", func(t *testing.T) {
		assert.True(t, refType.DeclaresMethod("method1"))
		assert.False(t, refType.DeclaresMethod("method2"))
	})

	t.Run("DeclaresMethodWithParams", func(t *testing.T) {
		assert.True(t, refType.DeclaresMethodWithParams("method1", 1))
		assert.False(t, refType.DeclaresMethodWithParams("method1", 2))
	})

	t.Run("HasMethod", func(t *testing.T) {
		assert.True(t, refType.HasMethod("method1"))
		assert.False(t, refType.HasMethod("method2"))
	})
}

func TestClassOrInterface(t *testing.T) {
	classOrInterface := NewClassOrInterface(
		true,
		[]string{"SubType1", "SubType2"},
		"CompanionObj",
		true,
		true,
	)

	t.Run("GetAPermittedSubtype", func(t *testing.T) {
		assert.Equal(t, []string{"SubType1", "SubType2"}, classOrInterface.GetAPermittedSubtype())
	})

	t.Run("GetCompanionObject", func(t *testing.T) {
		assert.Equal(t, "CompanionObj", classOrInterface.GetCompanionObject())
	})

	t.Run("GetIsSealed", func(t *testing.T) {
		assert.True(t, classOrInterface.GetIsSealed())
	})

	t.Run("GetIsLocal", func(t *testing.T) {
		assert.True(t, classOrInterface.GetIsLocal())
	})

	t.Run("GetIsPackageProtected", func(t *testing.T) {
		assert.True(t, classOrInterface.GetIsPackageProtected())
	})
}

func TestClass(t *testing.T) {
	classOrInterface := ClassOrInterface{
		IsSealed:           true,
		PermittedSubtypes:  []string{"SubType"},
		CompanionObject:    "Companion",
		IsLocal:            false,
		IsPackageProtected: true,
	}

	class := NewClass(
		"TestClass",
		[]string{"@Test", "@Mock"},
		true,
		true,
		classOrInterface,
	)

	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		assert.Equal(t, "Class", class.GetAPrimaryQlClass())
	})

	t.Run("GetAnAnnotation", func(t *testing.T) {
		assert.Equal(t, []string{"@Test", "@Mock"}, class.GetAnAnnotation())
	})

	t.Run("GetIsAnonymous", func(t *testing.T) {
		assert.True(t, class.GetIsAnonymous())
	})

	t.Run("GetIsFileClass", func(t *testing.T) {
		assert.True(t, class.GetIsFileClass())
	})
}
