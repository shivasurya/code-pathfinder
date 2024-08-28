package graph

import (
	"fmt"
	"testing"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

func TestQueryEntities(t *testing.T) {
	graph := NewCodeGraph()
	node1 := &Node{ID: "abcd", Type: "method_declaration", Name: "testMethod", Modifier: "public"}
	node2 := &Node{ID: "cdef", Type: "class_declaration", Name: "TestClass", Modifier: "private"}
	graph.AddNode(node1)
	graph.AddNode(node2)

	tests := []struct {
		name     string
		query    parser.Query
		expected int
	}{
		{
			name: "Query with expression",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == 'public'",
			},
			expected: 1,
		},
		{
			name: "Query with no results",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == 'private'",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QueryEntities(graph, tt.query)
			fmt.Println(result)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

func TestFilterEntities(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		query    parser.Query
		expected bool
	}{
		{
			name: "Filter method by visibility",
			node: &Node{Type: "method_declaration", Modifier: "public"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == 'public'",
			},
			expected: true,
		},
		{
			name: "Filter class by name",
			node: &Node{Type: "class_declaration", Name: "TestClass"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_declaration", Alias: "cd"}},
				Expression: "cd.getName() == 'TestClass'",
			},
			expected: true,
		},
		{
			name: "Filter method by return type",
			node: &Node{Type: "method_declaration", ReturnType: "void"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getReturnType() == 'void'",
			},
			expected: true,
		},
		{
			name: "Filter variable by data type",
			node: &Node{Type: "variable_declaration", DataType: "int"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "variable_declaration", Alias: "vd"}},
				Expression: "vd.getVariableDataType() == 'int'",
			},
			expected: true,
		},
		{
			name: "Filter with complex expression",
			node: &Node{Type: "method_declaration", Modifier: "public", ReturnType: "String", Name: "getName"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == 'public' && md.getReturnType() == 'String' && md.getName() == 'getName'",
			},
			expected: true,
		},
		{
			name: "Filter with false condition",
			node: &Node{Type: "method_declaration", Modifier: "private"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == 'public'",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterEntities([]*Node{tt.node}, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateProxyEnv(t *testing.T) {
	node := &Node{
		Type:                 "method_declaration",
		Name:                 "testMethod",
		Modifier:             "public",
		ReturnType:           "String",
		MethodArgumentsType:  []string{"int", "boolean"},
		MethodArgumentsValue: []string{"arg1", "arg2"},
		ThrowsExceptions:     []string{"IOException"},
		JavaDoc:              &model.Javadoc{},
	}

	query := parser.Query{
		SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
	}

	env := generateProxyEnv(node, query)
	assert.NotNil(t, env)
	assert.Contains(t, env, "md")
	methodEnv := env["md"].(map[string]interface{})

	assert.NotNil(t, methodEnv["getVisibility"])
	assert.NotNil(t, methodEnv["getAnnotation"])
	assert.NotNil(t, methodEnv["getReturnType"])
	assert.NotNil(t, methodEnv["getName"])
	assert.NotNil(t, methodEnv["getArgumentType"])
	assert.NotNil(t, methodEnv["getArgumentName"])
	assert.NotNil(t, methodEnv["getThrowsType"])
	assert.NotNil(t, methodEnv["getDoc"])

	visibility := methodEnv["getVisibility"].(func() string)()
	assert.Equal(t, "public", visibility)

	name := methodEnv["getName"].(func() string)()
	assert.Equal(t, "testMethod", name)

	returnType := methodEnv["getReturnType"].(func() string)()
	assert.Equal(t, "String", returnType)

	argTypes := methodEnv["getArgumentType"].(func() []string)()
	assert.Equal(t, []string{"int", "boolean"}, argTypes)

	argNames := methodEnv["getArgumentName"].(func() []string)()
	assert.Equal(t, []string{"arg1", "arg2"}, argNames)

	throwsTypes := methodEnv["getThrowsType"].(func() []string)()
	assert.Equal(t, []string{"IOException"}, throwsTypes)
}
