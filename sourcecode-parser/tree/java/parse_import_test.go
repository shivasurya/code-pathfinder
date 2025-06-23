package java

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

// TestParseImportDeclaration tests the ParseImportDeclaration function
func TestParseImportDeclaration(t *testing.T) {
	t.Run("Simple import", func(t *testing.T) {
		// Setup
		sourceCode := []byte("import java.util.List;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the import_declaration node
		importNode := findNodeByType(rootNode, "import_declaration")
		assert.NotNil(t, importNode)

		// Call the function with our parsed node
		importType := ParseImportDeclaration(importNode, sourceCode, "Sample.java")

		// Assertions
		assert.NotNil(t, importType)
		assert.Equal(t, "java.util.List", importType.ImportedType)
		assert.Equal(t, "Sample.java", importType.SourceDeclaration)
	})

	t.Run("Import with static keyword", func(t *testing.T) {
		// Setup
		sourceCode := []byte("import static java.util.Collections.sort;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the import_declaration node
		importNode := findNodeByType(rootNode, "import_declaration")
		assert.NotNil(t, importNode)

		// Call the function with our parsed node
		importType := ParseImportDeclaration(importNode, sourceCode, "Sample.java")

		// Assertions
		assert.NotNil(t, importType)
		assert.Equal(t, "java.util.Collections.sort", importType.ImportedType)
		assert.Equal(t, "Sample.java", importType.SourceDeclaration)
	})

	t.Run("Import with wildcard", func(t *testing.T) {
		// Setup
		sourceCode := []byte("import java.util.*;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the import_declaration node
		importNode := findNodeByType(rootNode, "import_declaration")
		assert.NotNil(t, importNode)

		// Call the function with our parsed node
		importType := ParseImportDeclaration(importNode, sourceCode, "Sample.java")

		// Assertions
		assert.NotNil(t, importType)
		assert.Equal(t, "java.util", importType.ImportedType)
		assert.Equal(t, "Sample.java", importType.SourceDeclaration)
	})

	t.Run("Import with single identifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte("import String;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the import_declaration node
		importNode := findNodeByType(rootNode, "import_declaration")
		assert.NotNil(t, importNode)

		// Call the function with our parsed node
		importType := ParseImportDeclaration(importNode, sourceCode, "Sample.java")

		// Assertions
		assert.NotNil(t, importType)
		assert.Equal(t, "String", importType.ImportedType)
		assert.Equal(t, "Sample.java", importType.SourceDeclaration)
	})
}

// TestParsePackageDeclaration tests the ParsePackageDeclaration function
func TestParsePackageDeclaration(t *testing.T) {
	t.Run("Simple package declaration", func(t *testing.T) {
		// Setup
		sourceCode := []byte("package com.example;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the package_declaration node
		packageNode := findNodeByType(rootNode, "package_declaration")
		assert.NotNil(t, packageNode)

		// Call the function with our parsed node
		pkg := ParsePackageDeclaration(packageNode, sourceCode)

		// Assertions
		assert.NotNil(t, pkg)
		assert.Equal(t, "com.example", pkg.QualifiedName)
	})

	t.Run("Package declaration with multiple levels", func(t *testing.T) {
		// Setup
		sourceCode := []byte("package org.example.project.subpackage;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the package_declaration node
		packageNode := findNodeByType(rootNode, "package_declaration")
		assert.NotNil(t, packageNode)

		// Call the function with our parsed node
		pkg := ParsePackageDeclaration(packageNode, sourceCode)

		// Assertions
		assert.NotNil(t, pkg)
		assert.Equal(t, "org.example.project.subpackage", pkg.QualifiedName)
	})

	t.Run("Package declaration with single identifier", func(t *testing.T) {
		// Setup
		sourceCode := []byte("package example;")

		// Parse source code to get the node
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the package_declaration node
		packageNode := findNodeByType(rootNode, "package_declaration")
		assert.NotNil(t, packageNode)

		// Call the function with our parsed node
		pkg := ParsePackageDeclaration(packageNode, sourceCode)

		// Assertions
		assert.NotNil(t, pkg)
		assert.Equal(t, "example", pkg.QualifiedName)
	})
}

// TestIsIdentifier tests the isIdentifier function
func TestIsIdentifier(t *testing.T) {
	// Since we can't easily create tree-sitter nodes with specific types for testing,
	// we'll test the function by creating a simple Java code that produces the node types we need

	t.Run("Test with identifier", func(t *testing.T) {
		// Parse a simple identifier
		sourceCode := []byte("public class Test { }")
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Find the class name identifier node
		classNode := findNodeByType(rootNode, "class_declaration")
		identifierNode := classNode.ChildByFieldName("name")

		// Test the function
		assert.True(t, isIdentifier(identifierNode))
	})

	t.Run("Test with non-identifier", func(t *testing.T) {
		// Parse code with a non-identifier node
		sourceCode := []byte("public class Test { }")
		rootNode := sitter.Parse(sourceCode, java.GetLanguage())

		// Use the class_declaration node itself (not an identifier)
		classNode := findNodeByType(rootNode, "class_declaration")

		// Test the function
		assert.False(t, isIdentifier(classNode))
	})
}

// Helper function to find a node by its type in the tree
func findNodeByType(node *sitter.Node, nodeType string) *sitter.Node {
	if node.Type() == nodeType {
		return node
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findNodeByType(child, nodeType); found != nil {
			return found
		}
	}

	return nil
}
