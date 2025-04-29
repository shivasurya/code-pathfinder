package graph

import (
	"testing"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/db"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

func TestQueryEntities_MethodDeclarations(t *testing.T) {
	// Setup test database
	storageNode := &db.StorageNode{}

	// Add test method declarations
	methods := []*model.Method{
		{
			Name:           "testMethod1",
			QualifiedName:  "com.example.TestClass.testMethod1",
			ReturnType:     "void",
			Visibility:     "public",
			Parameters:     []string{"String", "int"},
			ParameterNames: []string{"param1", "param2"},
		},
		{
			Name:          "testMethod2",
			QualifiedName: "com.example.TestClass.testMethod2",
			ReturnType:    "String",
			Visibility:    "private",
			IsStatic:      true,
		},
	}

	for _, method := range methods {
		storageNode.AddMethodDecl(method)
	}

	// Test case 1: Query all methods
	t.Run("query all methods", func(t *testing.T) {
		query := parser.Query{
			SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
			ExpressionTree: &parser.ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getName()",
					Alias:  "md",
					Entity: "method_declaration",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "\"testMethod1\"",
				},
			},
		}

		nodes, output := QueryEntities(storageNode, query)

		assert.Equal(t, 1, len(nodes), "Should find 1 method")
		assert.NotNil(t, output, "Output should not be nil")
	})

	// Test case 2: Query with filter
	t.Run("query with filter", func(t *testing.T) {
		query := parser.Query{
			SelectList: []parser.SelectList{{Entity: "method_declaration"}},
			ExpressionTree: &parser.ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getVisibility()",
					Alias:  "md",
					Entity: "method_declaration",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "\"public\"",
				},
			},
			SelectOutput: []parser.SelectOutput{
				{SelectEntity: "md.getName()", Type: "method_chain"},
				{SelectEntity: "md.getVisibility()", Type: "method_chain"},
			},
		}

		nodes, output := QueryEntities(storageNode, query)

		assert.Equal(t, 1, len(nodes), "Should find 1 public method")
		assert.NotNil(t, output, "Output should not be nil")
		assert.Equal(t, 1, len(output), "Should have 1 output row")
		assert.Equal(t, "testMethod1", output[0][0], "First method should be testMethod1")
		assert.Equal(t, "public", output[0][1], "First method should be public")
	})
}

func TestQueryEntities_ClassDeclarations(t *testing.T) {
	// Setup test database
	storageNode := &db.StorageNode{}

	// Add test class declarations
	classes := []*model.Class{
		{
			ClassOrInterface: model.ClassOrInterface{
				RefType: model.RefType{
					QualifiedName: "com.example.TestClass1",
					Package:       "com.example",
				},
			},
			ClassID: "1",
		},
		{
			ClassOrInterface: model.ClassOrInterface{
				RefType: model.RefType{
					QualifiedName: "com.example.TestClass2",
					Package:       "com.example",
				},
			},
			ClassID: "2",
		},
	}

	for _, class := range classes {
		storageNode.AddClassDecl(class)
	}

	t.Run("query all classes", func(t *testing.T) {
		query := parser.Query{
			SelectList: []parser.SelectList{{Entity: "class_declaration", Alias: "cd"}},
			ExpressionTree: &parser.ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getName()",
					Alias:  "cd",
					Entity: "class_declaration",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "\"com.example.TestClass1\"",
				},
			},
		}

		nodes, output := QueryEntities(storageNode, query)

		assert.Equal(t, 1, len(nodes), "Should find 1 class")
		assert.NotNil(t, nodes, "Nodes should not be nil")
		assert.NotNil(t, output, "Output should not be nil")
	})
}

func TestQueryEntities_EmptyQuery(t *testing.T) {
	storageNode := &db.StorageNode{}

	t.Run("empty query returns nil", func(t *testing.T) {
		query := parser.Query{}
		nodes, output := QueryEntities(storageNode, query)

		assert.Nil(t, nodes, "Nodes should be nil for empty query")
		assert.Nil(t, output, "Output should be nil for empty query")
	})
}
