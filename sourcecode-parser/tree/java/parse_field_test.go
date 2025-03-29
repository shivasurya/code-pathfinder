package java

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

// TestFieldDeclarationStructure tests the structure and behavior of the FieldDeclaration model
func TestFieldDeclarationStructure(t *testing.T) {
	t.Run("Basic field declaration", func(t *testing.T) {
		// Create a field declaration manually to test the structure
		fieldDecl := &model.FieldDeclaration{
			Type:              "int",
			FieldNames:        []string{"count"},
			Visibility:        "private",
			IsStatic:          false,
			IsFinal:           true,
			IsVolatile:        false,
			IsTransient:       false,
			SourceDeclaration: "Test.java",
		}

		// Verify the structure
		assert.Equal(t, "int", fieldDecl.Type)
		assert.Equal(t, []string{"count"}, fieldDecl.FieldNames)
		assert.Equal(t, "private", fieldDecl.Visibility)
		assert.False(t, fieldDecl.IsStatic)
		assert.True(t, fieldDecl.IsFinal)
		assert.False(t, fieldDecl.IsVolatile)
		assert.False(t, fieldDecl.IsTransient)
		assert.Equal(t, "Test.java", fieldDecl.SourceDeclaration)

		// Test the ToString method
		expected := "private final int count;"
		assert.Equal(t, expected, fieldDecl.ToString())
	})

	t.Run("Field declaration with multiple fields", func(t *testing.T) {
		// Create a field declaration with multiple fields
		fieldDecl := &model.FieldDeclaration{
			Type:              "String",
			FieldNames:        []string{"firstName", "lastName"},
			Visibility:        "public",
			IsStatic:          false,
			IsFinal:           false,
			IsVolatile:        false,
			IsTransient:       false,
			SourceDeclaration: "Test.java",
		}

		// Verify the structure
		assert.Equal(t, "String", fieldDecl.Type)
		assert.Equal(t, []string{"firstName", "lastName"}, fieldDecl.FieldNames)
		assert.Equal(t, "public", fieldDecl.Visibility)

		// Test the ToString method
		expected := "public String firstName, lastName;"
		assert.Equal(t, expected, fieldDecl.ToString())
	})

	t.Run("Field declaration with all modifiers", func(t *testing.T) {
		// Create a field declaration with all modifiers
		fieldDecl := &model.FieldDeclaration{
			Type:              "long",
			FieldNames:        []string{"serialVersionUID"},
			Visibility:        "private",
			IsStatic:          true,
			IsFinal:           true,
			IsVolatile:        false,
			IsTransient:       true,
			SourceDeclaration: "Test.java",
		}

		// Verify the structure
		assert.Equal(t, "long", fieldDecl.Type)
		assert.Equal(t, []string{"serialVersionUID"}, fieldDecl.FieldNames)
		assert.Equal(t, "private", fieldDecl.Visibility)
		assert.True(t, fieldDecl.IsStatic)
		assert.True(t, fieldDecl.IsFinal)
		assert.False(t, fieldDecl.IsVolatile)
		assert.True(t, fieldDecl.IsTransient)

		// Test the ToString method
		expected := "private static final transient long serialVersionUID;"
		assert.Equal(t, expected, fieldDecl.ToString())
	})
}
