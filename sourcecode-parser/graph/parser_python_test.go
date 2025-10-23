package graph

import (
	"context"
	"testing"

	queryparser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

func TestParsePythonFunctionDefinition(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedName   string
		expectedParams int
	}{
		{
			name:           "Simple function",
			code:           "def hello():\n    pass",
			expectedName:   "hello",
			expectedParams: 0,
		},
		{
			name:           "Function with parameters",
			code:           "def add(x, y):\n    return x + y",
			expectedName:   "add",
			expectedParams: 2,
		},
		{
			name:           "Function with default parameters",
			code:           "def greet(name, msg='Hello'):\n    print(msg, name)",
			expectedName:   "greet",
			expectedParams: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(python.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			defer tree.Close()

			graph := NewCodeGraph()
			root := tree.RootNode()
			
			// Find function_definition node
			var funcNode *sitter.Node
			for i := 0; i < int(root.NamedChildCount()); i++ {
				child := root.NamedChild(i)
				if child.Type() == "function_definition" {
					funcNode = child
					break
				}
			}

			if funcNode == nil {
				t.Fatal("No function_definition node found")
			}

			node := parsePythonFunctionDefinition(funcNode, []byte(tt.code), graph, "test.py")

			if node.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, node.Name)
			}
			if len(node.MethodArgumentsValue) != tt.expectedParams {
				t.Errorf("Expected %d params, got %d", tt.expectedParams, len(node.MethodArgumentsValue))
			}
			if !node.isPythonSourceFile {
				t.Error("Expected isPythonSourceFile to be true")
			}
		})
	}
}

func TestParsePythonClassDefinition(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedName  string
		expectedBases int
	}{
		{
			name:          "Simple class",
			code:          "class MyClass:\n    pass",
			expectedName:  "MyClass",
			expectedBases: 0,
		},
		{
			name:          "Class with inheritance",
			code:          "class Child(Parent):\n    pass",
			expectedName:  "Child",
			expectedBases: 1,
		},
		{
			name:          "Class with multiple inheritance",
			code:          "class Multi(Base1, Base2):\n    pass",
			expectedName:  "Multi",
			expectedBases: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sitter.NewParser()
			parser.SetLanguage(python.GetLanguage())
			defer parser.Close()

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.code))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			defer tree.Close()

			graph := NewCodeGraph()
			root := tree.RootNode()
			
			var classNode *sitter.Node
			for i := 0; i < int(root.NamedChildCount()); i++ {
				child := root.NamedChild(i)
				if child.Type() == "class_definition" {
					classNode = child
					break
				}
			}

			if classNode == nil {
				t.Fatal("No class_definition node found")
			}

			parsePythonClassDefinition(classNode, []byte(tt.code), graph, "test.py")

			if len(graph.Nodes) == 0 {
				t.Fatal("No nodes added to graph")
			}

			var node *Node
			for _, n := range graph.Nodes {
				if n.Type == "class_definition" {
					node = n
					break
				}
			}

			if node == nil {
				t.Fatal("No class node found in graph")
			}

			if node.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, node.Name)
			}
			if len(node.Interface) != tt.expectedBases {
				t.Errorf("Expected %d bases, got %d", tt.expectedBases, len(node.Interface))
			}
		})
	}
}

func TestParsePythonAssignment(t *testing.T) {
	code := "x = 42"
	
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer tree.Close()

	graph := NewCodeGraph()
	root := tree.RootNode()
	
	var assignNode *sitter.Node
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "expression_statement" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				subchild := child.NamedChild(j)
				if subchild.Type() == "assignment" {
					assignNode = subchild
					break
				}
			}
		}
	}

	if assignNode == nil {
		t.Fatal("No assignment node found")
	}

	parsePythonAssignment(assignNode, []byte(code), graph, "test.py")

	if len(graph.Nodes) == 0 {
		t.Fatal("No nodes added to graph")
	}

	var node *Node
	for _, n := range graph.Nodes {
		if n.Type == "variable_assignment" {
			node = n
			break
		}
	}

	if node == nil {
		t.Fatal("No variable assignment node found")
	}

	if node.Name != "x" {
		t.Errorf("Expected variable name 'x', got %s", node.Name)
	}
	if node.VariableValue != "42" {
		t.Errorf("Expected value '42', got %s", node.VariableValue)
	}
}

// TestPythonEndToEndQueryIntegration tests the full flow from parsing Python code to querying it.
func TestPythonEndToEndQueryIntegration(t *testing.T) {
	pythonCode := `
class Calculator:
    """A simple calculator class"""

    def add(self, x, y):
        """Add two numbers"""
        return x + y

    def subtract(self, x, y):
        """Subtract two numbers"""
        return x - y

def process_data(data):
    """Process data function"""
    return data * 2
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(pythonCode))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer tree.Close()

	graph := NewCodeGraph()
	root := tree.RootNode()
	buildGraphFromAST(root, []byte(pythonCode), graph, nil, "test.py")

	t.Run("Query Python classes", func(t *testing.T) {
		query := queryparser.Query{
			SelectList: []queryparser.SelectList{{Entity: "class_definition", Alias: "c"}},
			Expression: "c.getName() == \"Calculator\"",
			Condition: []string{
				"c.getName()==\"Calculator\"",
			},
		}

		resultSet, _ := QueryEntities(graph, query)
		if len(resultSet) != 1 {
			t.Errorf("Expected 1 class, got %d", len(resultSet))
		}

		if len(resultSet) > 0 && resultSet[0][0].Name != "Calculator" {
			t.Errorf("Expected class name 'Calculator', got %s", resultSet[0][0].Name)
		}
	})

	t.Run("Query Python functions", func(t *testing.T) {
		query := queryparser.Query{
			SelectList: []queryparser.SelectList{{Entity: "function_definition", Alias: "f"}},
			Expression: "f.getName() == \"process_data\"",
			Condition: []string{
				"f.getName()==\"process_data\"",
			},
		}

		resultSet, _ := QueryEntities(graph, query)
		if len(resultSet) != 1 {
			t.Errorf("Expected 1 function, got %d", len(resultSet))
		}

		if len(resultSet) > 0 && resultSet[0][0].Name != "process_data" {
			t.Errorf("Expected function name 'process_data', got %s", resultSet[0][0].Name)
		}
	})

	t.Run("Query all Python functions", func(t *testing.T) {
		query := queryparser.Query{
			SelectList: []queryparser.SelectList{{Entity: "function_definition", Alias: "f"}},
			Expression: "",
			Condition:  []string{},
		}

		resultSet, _ := QueryEntities(graph, query)
		// Should find: add, subtract, process_data (3 functions total)
		if len(resultSet) < 3 {
			t.Errorf("Expected at least 3 functions, got %d", len(resultSet))
		}
	})

	t.Run("Query Python functions with argument filtering", func(t *testing.T) {
		query := queryparser.Query{
			SelectList: []queryparser.SelectList{{Entity: "function_definition", Alias: "f"}},
			Expression: "len(f.getArgumentName()) == 3",
			Condition: []string{
				"len(f.getArgumentName())==3",
			},
		}

		resultSet, _ := QueryEntities(graph, query)
		// Should find: add and subtract (both have self, x, y)
		if len(resultSet) != 2 {
			t.Errorf("Expected 2 functions with 3 arguments, got %d", len(resultSet))
		}
	})

	t.Run("Verify toString method works", func(t *testing.T) {
		query := queryparser.Query{
			SelectList: []queryparser.SelectList{{Entity: "class_definition", Alias: "c"}},
			SelectOutput: []queryparser.SelectOutput{
				{Type: "variable", SelectEntity: "c"},
			},
			Expression: "c.getName() == \"Calculator\"",
			Condition: []string{
				"c.getName()==\"Calculator\"",
			},
		}

		resultSet, output := QueryEntities(graph, query)
		if len(resultSet) != 1 {
			t.Errorf("Expected 1 class, got %d", len(resultSet))
		}

		if len(output) != 1 {
			t.Errorf("Expected 1 output, got %d", len(output))
		}

		// Verify that output doesn't contain error about missing toString
		if len(output) > 0 && len(output[0]) > 0 {
			outputStr, ok := output[0][0].(string)
			if !ok || outputStr == "" {
				t.Error("Expected non-empty string output from toString")
			}
		}
	})
}
