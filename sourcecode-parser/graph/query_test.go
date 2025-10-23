package graph

import (
	"fmt"
	"testing"
	"time"

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
				Expression: "md.getVisibility() == \"public\"",
				Condition: []string{
					"md.getVisibility()==\"public\"",
				},
			},
			expected: 1,
		},
		{
			name: "Query with no results",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == \"private\"",
				Condition: []string{
					"md.getVisibility()==\"private\"",
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultSet, output := QueryEntities(graph, tt.query)
			fmt.Println(resultSet)
			fmt.Println(output)
			assert.Equal(t, tt.expected, len(resultSet))
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
				Expression: "md.getVisibility() == \"public\"",
			},
			expected: true,
		},
		{
			name: "Filter class by name",
			node: &Node{Type: "class_declaration", Name: "TestClass"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_declaration", Alias: "cd"}},
				Expression: "cd.getName() == \"TestClass\"",
			},
			expected: true,
		},
		{
			name: "Filter method by return type",
			node: &Node{Type: "method_declaration", ReturnType: "void"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getReturnType() == \"void\"",
			},
			expected: true,
		},
		{
			name: "Filter variable by data type",
			node: &Node{Type: "variable_declaration", DataType: "int"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "variable_declaration", Alias: "vd"}},
				Expression: "vd.getVariableDataType() == \"int\"",
			},
			expected: true,
		},
		{
			name: "Filter with complex expression",
			node: &Node{Type: "method_declaration", Modifier: "public", ReturnType: "String", Name: "getName"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == \"public\" && md.getReturnType() == \"String\" && md.getName() == \"getName\"",
			},
			expected: true,
		},
		{
			name: "Filter with false condition",
			node: &Node{Type: "method_declaration", Modifier: "private"},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "md"}},
				Expression: "md.getVisibility() == \"public\"",
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

func TestCartesianProduct(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]interface{}
		expected [][]interface{}
	}{
		{
			name:     "Empty input",
			input:    [][]interface{}{},
			expected: [][]interface{}{{}},
		},
		{
			name:     "Single set",
			input:    [][]interface{}{{1, 2, 3}},
			expected: [][]interface{}{{1}, {2}, {3}},
		},
		{
			name:     "Two sets",
			input:    [][]interface{}{{1, 2}, {"a", "b"}},
			expected: [][]interface{}{{1, "a"}, {2, "a"}, {1, "b"}, {2, "b"}},
		},
		{
			name:  "Three sets",
			input: [][]interface{}{{1, 2}, {"a", "b"}, {true, false}},
			expected: [][]interface{}{
				{1, "a", true}, {2, "a", true},
				{1, "b", true}, {2, "b", true},
				{1, "a", false}, {2, "a", false},
				{1, "b", false}, {2, "b", false},
			},
		},
		{
			name:     "Mixed types",
			input:    [][]interface{}{{1, "x"}, {true, 3.14}},
			expected: [][]interface{}{{1, true}, {"x", true}, {1, 3.14}, {"x", 3.14}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cartesianProduct(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCartesianProductLargeInput(t *testing.T) {
	input := [][]interface{}{
		{1, 2, 3, 4, 5},
		{"a", "b", "c", "d", "e"},
		{true, false},
	}
	result := cartesianProduct(input)
	assert.Equal(t, 50, len(result))
	assert.Equal(t, 3, len(result[0]))
}

func TestCartesianProductPerformance(t *testing.T) {
	input := make([][]interface{}, 10)
	for i := range input {
		input[i] = make([]interface{}, 5)
		for j := range input[i] {
			input[i][j] = j
		}
	}

	start := time.Now()
	result := cartesianProduct(input)
	duration := time.Since(start)

	assert.Equal(t, 9765625, len(result))
	assert.Less(t, duration, 10*time.Second)
}

// TestPythonClassDefinitionSupport tests that Python class_definition entities are properly supported.
func TestPythonClassDefinitionSupport(t *testing.T) {
	graph := NewCodeGraph()
	node := &Node{
		ID:                 "python_class_1",
		Type:               "class_definition",
		Name:               "Calculator",
		Interface:          []string{"object"},
		isPythonSourceFile: true,
	}
	graph.AddNode(node)

	tests := []struct {
		name     string
		query    parser.Query
		expected int
	}{
		{
			name: "Query Python class by name",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_definition", Alias: "c"}},
				Expression: "c.getName() == \"Calculator\"",
				Condition: []string{
					"c.getName()==\"Calculator\"",
				},
			},
			expected: 1,
		},
		{
			name: "Query all Python classes",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_definition", Alias: "c"}},
				Expression: "",
				Condition:  []string{},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultSet, output := QueryEntities(graph, tt.query)
			assert.Equal(t, tt.expected, len(resultSet))
			assert.NotNil(t, output)
		})
	}
}

// TestPythonFunctionDefinitionSupport tests that Python function_definition entities are properly supported.
func TestPythonFunctionDefinitionSupport(t *testing.T) {
	graph := NewCodeGraph()
	node := &Node{
		ID:                   "python_func_1",
		Type:                 "function_definition",
		Name:                 "process_data",
		MethodArgumentsValue: []string{"data", "options"},
		isPythonSourceFile:   true,
	}
	graph.AddNode(node)

	tests := []struct {
		name     string
		query    parser.Query
		expected int
	}{
		{
			name: "Query Python function by name",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "function_definition", Alias: "f"}},
				Expression: "f.getName() == \"process_data\"",
				Condition: []string{
					"f.getName()==\"process_data\"",
				},
			},
			expected: 1,
		},
		{
			name: "Query all Python functions",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "function_definition", Alias: "f"}},
				Expression: "",
				Condition:  []string{},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultSet, output := QueryEntities(graph, tt.query)
			assert.Equal(t, tt.expected, len(resultSet))
			assert.NotNil(t, output)
		})
	}
}

// TestGenerateProxyEnvForPythonClass tests that proxy environment is correctly generated for Python classes.
func TestGenerateProxyEnvForPythonClass(t *testing.T) {
	node := &Node{
		Type:               "class_definition",
		Name:               "TestClass",
		Interface:          []string{"BaseClass", "Mixin"},
		isPythonSourceFile: true,
	}

	query := parser.Query{
		SelectList: []parser.SelectList{{Entity: "class_definition", Alias: "c"}},
	}

	env := generateProxyEnv(node, query)
	assert.NotNil(t, env)
	assert.Contains(t, env, "c")
	classEnv := env["c"].(map[string]interface{})

	// Verify all expected methods are present
	assert.NotNil(t, classEnv["getName"])
	assert.NotNil(t, classEnv["getInterface"])
	assert.NotNil(t, classEnv["toString"])

	// Test getName
	name := classEnv["getName"].(func() string)()
	assert.Equal(t, "TestClass", name)

	// Test getInterface
	interfaces := classEnv["getInterface"].(func() []string)()
	assert.Equal(t, []string{"BaseClass", "Mixin"}, interfaces)

	// Test toString
	toString := classEnv["toString"].(func() string)()
	assert.Contains(t, toString, "TestClass")
	assert.Contains(t, toString, "class_definition")
}

// TestGenerateProxyEnvForPythonFunction tests that proxy environment is correctly generated for Python functions.
func TestGenerateProxyEnvForPythonFunction(t *testing.T) {
	node := &Node{
		Type:                 "function_definition",
		Name:                 "calculate",
		MethodArgumentsValue: []string{"x", "y", "z"},
		isPythonSourceFile:   true,
	}

	query := parser.Query{
		SelectList: []parser.SelectList{{Entity: "function_definition", Alias: "f"}},
	}

	env := generateProxyEnv(node, query)
	assert.NotNil(t, env)
	assert.Contains(t, env, "f")
	funcEnv := env["f"].(map[string]interface{})

	// Verify all expected methods are present
	assert.NotNil(t, funcEnv["getName"])
	assert.NotNil(t, funcEnv["getArgumentName"])
	assert.NotNil(t, funcEnv["toString"])

	// Test getName
	name := funcEnv["getName"].(func() string)()
	assert.Equal(t, "calculate", name)

	// Test getArgumentName
	argNames := funcEnv["getArgumentName"].(func() []string)()
	assert.Equal(t, []string{"x", "y", "z"}, argNames)

	// Test toString
	toString := funcEnv["toString"].(func() string)()
	assert.Contains(t, toString, "calculate")
	assert.Contains(t, toString, "function_definition")
}

// TestFilterPythonEntities tests filtering of Python entities.
func TestFilterPythonEntities(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		query    parser.Query
		expected bool
	}{
		{
			name: "Filter Python class by name",
			node: &Node{
				Type:               "class_definition",
				Name:               "Calculator",
				isPythonSourceFile: true,
			},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_definition", Alias: "c"}},
				Expression: "c.getName() == \"Calculator\"",
			},
			expected: true,
		},
		{
			name: "Filter Python function by name",
			node: &Node{
				Type:               "function_definition",
				Name:               "process_data",
				isPythonSourceFile: true,
			},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "function_definition", Alias: "f"}},
				Expression: "f.getName() == \"process_data\"",
			},
			expected: true,
		},
		{
			name: "Filter Python class with false condition",
			node: &Node{
				Type:               "class_definition",
				Name:               "TestClass",
				isPythonSourceFile: true,
			},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_definition", Alias: "c"}},
				Expression: "c.getName() == \"DifferentClass\"",
			},
			expected: false,
		},
		{
			name: "Filter Python function by arguments",
			node: &Node{
				Type:                 "function_definition",
				Name:                 "add",
				MethodArgumentsValue: []string{"x", "y"},
				isPythonSourceFile:   true,
			},
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "function_definition", Alias: "f"}},
				Expression: "len(f.getArgumentName()) == 2",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterEntities([]*Node{tt.node}, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMixedJavaAndPythonEntities tests querying both Java and Python entities.
func TestMixedJavaAndPythonEntities(t *testing.T) {
	graph := NewCodeGraph()

	// Add Java class
	javaClass := &Node{
		ID:                 "java_class_1",
		Type:               "class_declaration",
		Name:               "JavaClass",
		Modifier:           "public",
		isJavaSourceFile:   true,
	}
	graph.AddNode(javaClass)

	// Add Python class
	pythonClass := &Node{
		ID:                 "python_class_1",
		Type:               "class_definition",
		Name:               "PythonClass",
		isPythonSourceFile: true,
	}
	graph.AddNode(pythonClass)

	// Add Java method
	javaMethod := &Node{
		ID:                 "java_method_1",
		Type:               "method_declaration",
		Name:               "javaMethod",
		Modifier:           "public",
		isJavaSourceFile:   true,
	}
	graph.AddNode(javaMethod)

	// Add Python function
	pythonFunc := &Node{
		ID:                 "python_func_1",
		Type:               "function_definition",
		Name:               "python_function",
		isPythonSourceFile: true,
	}
	graph.AddNode(pythonFunc)

	tests := []struct {
		name     string
		query    parser.Query
		expected int
	}{
		{
			name: "Query only Java classes",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_declaration", Alias: "c"}},
			},
			expected: 1,
		},
		{
			name: "Query only Python classes",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "class_definition", Alias: "c"}},
			},
			expected: 1,
		},
		{
			name: "Query only Java methods",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "method_declaration", Alias: "m"}},
			},
			expected: 1,
		},
		{
			name: "Query only Python functions",
			query: parser.Query{
				SelectList: []parser.SelectList{{Entity: "function_definition", Alias: "f"}},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultSet, _ := QueryEntities(graph, tt.query)
			assert.Equal(t, tt.expected, len(resultSet))
		})
	}
}
