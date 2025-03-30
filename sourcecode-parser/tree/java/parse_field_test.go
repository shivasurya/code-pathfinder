package java

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// Helper function to find field_declaration node in the parse tree
func findFieldDeclarationNode(node *sitter.Node) *sitter.Node {
	if node.Type() == "field_declaration" {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findFieldDeclarationNode(child); found != nil {
			return found
		}
	}

	return nil
}

// TestParseField tests the ParseField function
func TestParseField(t *testing.T) {
	t.Run("Basic field with private modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private int count;
		}`)

		// Parse the code
		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)
		assert.NotNil(t, fieldNode, "Field declaration node should be found")

		// Call the function with our parsed node
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "int", field.Type)
		assert.Equal(t, []string{"count"}, field.FieldNames)
		assert.Equal(t, "private", field.Visibility)
		assert.False(t, field.IsStatic)
		assert.False(t, field.IsFinal)
		assert.False(t, field.IsVolatile)
		assert.False(t, field.IsTransient)
		assert.Equal(t, "TestClass.java", field.SourceDeclaration)
	})

	t.Run("Field with public modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			public String name;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "String", field.Type)
		assert.Equal(t, []string{"name"}, field.FieldNames)
		assert.Equal(t, "public", field.Visibility)
	})

	t.Run("Field with protected modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			protected double value;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "double", field.Type)
		assert.Equal(t, []string{"value"}, field.FieldNames)
		assert.Equal(t, "protected", field.Visibility)
	})

	t.Run("Field with static modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			public static final int MAX_VALUE = 100;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "int", field.Type)
		assert.Equal(t, []string{"MAX_VALUE"}, field.FieldNames)
		assert.Equal(t, "public", field.Visibility)
		assert.True(t, field.IsStatic)
		assert.True(t, field.IsFinal)
	})

	t.Run("Field with multiple modifiers", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private static final transient long serialVersionUID = 1L;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "long", field.Type)
		assert.Equal(t, []string{"serialVersionUID"}, field.FieldNames)
		assert.Equal(t, "private", field.Visibility)
		assert.True(t, field.IsStatic)
		assert.True(t, field.IsFinal)
		assert.True(t, field.IsTransient)
		assert.False(t, field.IsVolatile)
	})

	t.Run("Field with volatile modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private volatile boolean running;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "boolean", field.Type)
		assert.Equal(t, []string{"running"}, field.FieldNames)
		assert.Equal(t, "private", field.Visibility)
		assert.False(t, field.IsStatic)
		assert.False(t, field.IsFinal)
		assert.True(t, field.IsVolatile)
		assert.False(t, field.IsTransient)
	})

	t.Run("Field with multiple variable names", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private String firstName, lastName;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "String", field.Type)
		assert.Equal(t, []string{"firstName", "lastName"}, field.FieldNames)
		assert.Equal(t, "private", field.Visibility)
	})

	t.Run("Field with initialization", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private int counter = 0;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "int", field.Type)
		assert.Equal(t, []string{"counter"}, field.FieldNames)
		assert.Equal(t, "private", field.Visibility)
	})

	t.Run("Field with no explicit visibility modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			int defaultVisibility;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "int", field.Type)
		assert.Equal(t, []string{"defaultVisibility"}, field.FieldNames)
		assert.Equal(t, "", field.Visibility) // Default package-private visibility
	})

	t.Run("Field with complex type", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private List<String> items;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)

		// Call the function
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, field)
		assert.Equal(t, "List<String>", field.Type)
		assert.Equal(t, []string{"items"}, field.FieldNames)
		assert.Equal(t, "private", field.Visibility)
	})
}

// TestParseFieldToString tests the ToString method of the FieldDeclaration model
func TestParseFieldToString(t *testing.T) {
	t.Run("Basic field with private modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private int count;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Test the ToString method
		expected := "private int count;"
		assert.Equal(t, expected, field.ToString())
	})

	t.Run("Field with multiple modifiers", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			public static final String CONSTANT = "value";
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Test the ToString method
		expected := "public static final String CONSTANT;"
		assert.Equal(t, expected, field.ToString())
	})

	t.Run("Field with multiple variable names", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private String firstName, lastName;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Test the ToString method
		expected := "private String firstName, lastName;"
		assert.Equal(t, expected, field.ToString())
	})

	t.Run("Field with all modifiers", func(t *testing.T) {
		// Setup
		sourceCode := []byte(`class TestClass {
			private static final volatile transient long id;
		}`)

		tree := sitter.Parse(sourceCode, java.GetLanguage())
		fieldNode := findFieldDeclarationNode(tree)
		field := ParseField(fieldNode, sourceCode, "TestClass.java")

		// Test the ToString method
		expected := "private static final volatile transient long id;"
		assert.Equal(t, expected, field.ToString())
	})
}
