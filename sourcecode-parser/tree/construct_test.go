package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/db"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/stretchr/testify/assert"
)

func setupTestData(t *testing.T) (string, *db.StorageNode) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create sample Java files
	sampleCode := []byte(`
	package com.example;
	
	import java.util.List;
	
	/**
	 * Sample class documentation
	 */
	public class TestClass {
		private int count;
		
		public void testMethod() {
			int localVar = 0;
			if (count > 0) {
				while (localVar < count) {
					localVar++;
				}
			}
			assert localVar >= 0 : "Local variable must be non-negative";
		}
		
		public void complexMethod() {
			for (int i = 0; i < 10; i++) {
				if (i % 2 == 0) {
					continue;
				}
				doSomething();
			}
		}
		
		private void doSomething() {
			Object obj = new Object();
			count = count + 1;
		}
	}
	`)

	testFile := filepath.Join(tempDir, "TestClass.java")
	err := os.WriteFile(testFile, sampleCode, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize storage node
	storageNode := db.NewStorageNode(tempDir)
	return tempDir, storageNode
}

func TestInitialize(t *testing.T) {
	tempDir, storageNode := setupTestData(t)

	// Test Initialize function
	trees := Initialize(tempDir, storageNode)

	// Verify the results
	assert.NotEmpty(t, trees, "Should return non-empty tree slice")
	assert.Equal(t, 1, len(trees), "Should process one file")

	// Verify the root node
	root := trees[0]
	assert.NotNil(t, root)
	assert.Equal(t, "File", root.Node.NodeType)
	assert.Equal(t, "TestClass.java", root.Node.FileNode.File)
}

func TestBuildQLTreeFromAST(t *testing.T) {
	// Setup parser
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	sampleCode := []byte("public class Test { private int x; }")
	tree, err := parser.ParseCtx(context.Background(), nil, sampleCode)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()

	// Create parent node and storage node
	parentNode := &model.TreeNode{
		Node: &model.Node{
			NodeType: "File",
			FileNode: &model.File{File: "Test.java"},
		},
	}

	tempDir := t.TempDir()
	storageNode := db.NewStorageNode(tempDir)

	// Test buildQLTreeFromAST
	buildQLTreeFromAST(tree.RootNode(), sampleCode, "Test.java", parentNode, storageNode)

	// Verify the results
	assert.NotEmpty(t, parentNode.Children)
	assert.Equal(t, "ClassDeclaration", parentNode.Children[0].Node.NodeType)
}

func TestGetFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"Test1.java",
		"Test2.java",
		"NotAJavaFile.txt",
	}

	for _, file := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, file), []byte(""), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test getFiles
	files, err := getFiles(tempDir)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files), "Should only find .java files")
}

func TestReadFile(t *testing.T) {
	tempDir := t.TempDir()
	testContent := []byte("test content")
	testFile := filepath.Join(tempDir, "test.txt")

	// Create test file
	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test readFile
	content, err := readFile(testFile)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Test with non-existent file
	content, err = readFile(filepath.Join(tempDir, "nonexistent.txt"))
	assert.Error(t, err)
	assert.Nil(t, content)
}
