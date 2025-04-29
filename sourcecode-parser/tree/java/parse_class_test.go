package java

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// TestParseClass tests the ParseClass function
func TestParseClass(t *testing.T) {
	t.Run("Basic class with name only", func(t *testing.T) {
		// Setup
		sourceCode := []byte("class TestClass {}")
		className := "TestClass"

		// parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our mocked data
		class := ParseClass(rootNode.Child(0), sourceCode, "TestClass.java")

		// Assertions
		assert.NotNil(t, class)
		assert.Equal(t, className, class.QualifiedName)
		assert.Equal(t, "", class.ClassOrInterface.Package)
		assert.Equal(t, "TestClass.java", class.SourceFile)
		assert.Empty(t, class.Annotations)
		assert.Contains(t, class.Modifiers, "")
		assert.Contains(t, class.SuperTypes, "")
	})

	t.Run("Class with access modifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte("public class PublicClass {}")
		className := "PublicClass"

		// parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Call the function with our mocked data
		class := ParseClass(rootNode.Child(0), sourceCode, "PublicClass.java")

		// Assertions
		assert.NotNil(t, class)
		assert.Equal(t, className, class.QualifiedName)
		assert.Equal(t, "PublicClass.java", class.SourceFile)
		assert.Contains(t, class.Modifiers, "public")
	})

	t.Run("Class with annotation", func(t *testing.T) {
		// Setup
		sourceCode := []byte("@Entity public class EntityClass {}")
		className := "EntityClass"

		// parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())
		node := rootNode.Child(0)

		// Call the function with our mocked data
		class := ParseClass(node, sourceCode, "EntityClass.java")

		// Assertions
		assert.NotNil(t, class)
		assert.Equal(t, className, class.QualifiedName)
		assert.Equal(t, "EntityClass.java", class.SourceFile)
		assert.Contains(t, class.Annotations, "@Entity")
		assert.Contains(t, class.Modifiers, "public")
	})

	t.Run("Class with superclass", func(t *testing.T) {
		// Setup
		sourceCode := []byte("public class ChildClass extends ParentClass implements FileInterface {}")
		className := "ChildClass"
		superClass := "ParentClass"

		// parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())
		node := rootNode.Child(0)

		// Call the function with our mocked data
		class := ParseClass(node, sourceCode, "ChildClass.java")

		// Assertions
		assert.NotNil(t, class)
		assert.Equal(t, className, class.QualifiedName)
		assert.Equal(t, "ChildClass.java", class.SourceFile)
		assert.Contains(t, class.SuperTypes, superClass)
		assert.Contains(t, class.Modifiers, "public")
		assert.Contains(t, class.SuperTypes, "FileInterface")
	})
}

func TestParseObjectCreationExpr(t *testing.T) {
	t.Run("Basic object creation with no arguments", func(t *testing.T) {
		// Setup
		sourceCode := []byte("new SimpleClass()")

		tree := sitter.Parse(sourceCode, java.GetLanguage())

		// The expression_statement node contains the object_creation_expression
		// In a real Java file, this would be inside a method body
		objectCreationNode := findObjectCreationNode(tree)

		// Call the function with our parsed node
		expr := ParseObjectCreationExpr(objectCreationNode, sourceCode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "SimpleClass", expr.ClassName)
		assert.Empty(t, expr.Args)
	})

	t.Run("Object creation with simple arguments", func(t *testing.T) {
		// Setup
		sourceCode := []byte("new Person(\"John\", 30)")

		// Parse source code
		tree := sitter.Parse(sourceCode, java.GetLanguage())

		objectCreationNode := findObjectCreationNode(tree)

		// Call the function
		expr := ParseObjectCreationExpr(objectCreationNode, sourceCode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "Person", expr.ClassName)
		assert.Equal(t, 2, len(expr.Args))
		assert.Equal(t, "\"John\"", expr.Args[0].NodeString)
		assert.Equal(t, "30", expr.Args[1].NodeString)
	})

	t.Run("Object creation with complex arguments", func(t *testing.T) {
		// Setup
		sourceCode := []byte("new Rectangle(10 + 5, height * 2)")

		tree := sitter.Parse(sourceCode, java.GetLanguage())

		objectCreationNode := findObjectCreationNode(tree)

		// Call the function
		expr := ParseObjectCreationExpr(objectCreationNode, sourceCode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "Rectangle", expr.ClassName)
		assert.Equal(t, 2, len(expr.Args))
		assert.Equal(t, "10 + 5", expr.Args[0].NodeString)
		assert.Equal(t, "height * 2", expr.Args[1].NodeString)
	})

	t.Run("Object creation with nested object creation", func(t *testing.T) {
		// Setup
		sourceCode := []byte("new Container(new Content())")

		// Parse source code
		tree := sitter.Parse(sourceCode, java.GetLanguage())

		objectCreationNode := findObjectCreationNode(tree)

		// Call the function
		expr := ParseObjectCreationExpr(objectCreationNode, sourceCode)

		// Assertions
		assert.NotNil(t, expr)
		assert.Equal(t, "Container", expr.ClassName)
		assert.Equal(t, 1, len(expr.Args))
		assert.Equal(t, "new Content()", expr.Args[0].NodeString)
	})
}

// Helper function to find the object_creation_expression node in the tree
func findObjectCreationNode(node *sitter.Node) *sitter.Node {
	if node.Type() == "object_creation_expression" {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findObjectCreationNode(child); found != nil {
			return found
		}
	}

	return nil
}
