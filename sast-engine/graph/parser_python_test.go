package graph

import (
	"context"
	"testing"

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

	parsePythonAssignment(assignNode, []byte(code), graph, "test.py", nil)

	if len(graph.Nodes) == 0 {
		t.Fatal("No nodes added to graph")
	}

	var node *Node
	for _, n := range graph.Nodes {
		if n.Name == "x" {
			node = n
			break
		}
	}

	if node == nil {
		t.Fatal("No variable assignment node found")
	}

	// Module-level lowercase variable should be module_variable.
	if node.Type != "module_variable" {
		t.Errorf("Expected type 'module_variable', got %s", node.Type)
	}
	if node.Name != "x" {
		t.Errorf("Expected variable name 'x', got %s", node.Name)
	}
	if node.VariableValue != "42" {
		t.Errorf("Expected value '42', got %s", node.VariableValue)
	}
}

func TestExtractDecorators(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "Single decorator",
			code: "@property\ndef get_value(self):\n    return self.value",
			expected: []string{"property"},
		},
		{
			name: "Multiple decorators",
			code: "@classmethod\n@cache\ndef compute(cls):\n    pass",
			expected: []string{"classmethod", "cache"},
		},
		{
			name: "Decorator with arguments",
			code: "@app.route('/api/users')\ndef get_users():\n    pass",
			expected: []string{"app.route"},
		},
		{
			name: "No decorators",
			code: "def simple():\n    pass",
			expected: []string{},
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

			root := tree.RootNode()
			var decoratedNode *sitter.Node

			// Find decorated_definition or function_definition node.
			for i := 0; i < int(root.NamedChildCount()); i++ {
				child := root.NamedChild(i)
				if child.Type() == "decorated_definition" {
					decoratedNode = child
					break
				}
			}

			var decorators []string
			if decoratedNode != nil {
				decorators = extractDecorators(decoratedNode, []byte(tt.code))
			}

			if len(decorators) != len(tt.expected) {
				t.Errorf("Expected %d decorators, got %d", len(tt.expected), len(decorators))
			}

			for i, expected := range tt.expected {
				if i >= len(decorators) {
					t.Errorf("Missing decorator at index %d: %s", i, expected)
					continue
				}
				if decorators[i] != expected {
					t.Errorf("Expected decorator %s, got %s", expected, decorators[i])
				}
			}
		})
	}
}

func TestExtractDecorators_NotDecoratedDefinition(t *testing.T) {
	// Test with non-decorated node type.
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	code := "def simple():\n    pass"
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
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

	// Passing function_definition instead of decorated_definition.
	decorators := extractDecorators(funcNode, []byte(code))
	if len(decorators) != 0 {
		t.Errorf("Expected 0 decorators for non-decorated node, got %d", len(decorators))
	}
}

func TestHasDecorator(t *testing.T) {
	tests := []struct {
		name       string
		decorators []string
		search     string
		expected   bool
	}{
		{"Found - property", []string{"property", "cache"}, "property", true},
		{"Found - cache", []string{"property", "cache"}, "cache", true},
		{"Not found", []string{"property", "cache"}, "staticmethod", false},
		{"Empty list", []string{}, "property", false},
		{"Nil list", nil, "property", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasDecorator(tt.decorators, tt.search)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsConstantName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		{"Simple constant", "MAX_SIZE", true},
		{"Single word constant", "API", true},
		{"Constant with numbers", "API_V2", true},
		{"Lowercase", "max_size", false},
		{"Mixed case", "MaxSize", false},
		{"Camel case", "maxSize", false},
		{"Starting with underscore", "_PRIVATE_MAX", true},
		{"Only underscores", "___", false},
		{"Empty string", "", false},
		{"Only digits", "123", false},
		{"With special chars", "MAX-SIZE", false},
		{"Leading underscore lowercase", "_private", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConstantName(tt.varName)
			if result != tt.expected {
				t.Errorf("isConstantName(%q) = %v, expected %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestIsConstructor(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		expected bool
	}{
		{"Constructor", "__init__", true},
		{"Regular method", "get_value", false},
		{"Similar name", "__init", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConstructor(tt.funcName)
			if result != tt.expected {
				t.Errorf("isConstructor(%q) = %v, expected %v", tt.funcName, result, tt.expected)
			}
		})
	}
}

// findNodeByType recursively searches for a node of a specific type.
func findNodeByType(node *sitter.Node, nodeType string) *sitter.Node {
	if node.Type() == nodeType {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		result := findNodeByType(node.Child(i), nodeType)
		if result != nil {
			return result
		}
	}
	return nil
}

// findNodeByCondition recursively searches for a node matching a condition.
func findNodeByCondition(node *sitter.Node, condition func(*sitter.Node) bool) *sitter.Node {
	if condition(node) {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		result := findNodeByCondition(node.Child(i), condition)
		if result != nil {
			return result
		}
	}
	return nil
}

func TestParsePythonFunctionDefinition_Constructor(t *testing.T) {
	code := `
class User:
    def __init__(self, name):
        self.name = name
`
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

	// Find __init__ function.
	initNode := findNodeByCondition(root, func(n *sitter.Node) bool {
		if n.Type() == "function_definition" {
			nameNode := n.ChildByFieldName("name")
			return nameNode != nil && nameNode.Content([]byte(code)) == "__init__"
		}
		return false
	})

	if initNode == nil {
		t.Fatal("No __init__ method found")
	}

	node := parsePythonFunctionDefinition(initNode, []byte(code), graph, "test.py")

	if node.Type != "constructor" {
		t.Errorf("Expected type 'constructor', got %s", node.Type)
	}
	if node.Name != "__init__" {
		t.Errorf("Expected name '__init__', got %s", node.Name)
	}
}

func TestParsePythonFunctionDefinition_Property(t *testing.T) {
	code := `
class User:
    @property
    def name(self):
        return self._name
`
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

	// Find decorated function.
	funcNode := findNodeByCondition(root, func(n *sitter.Node) bool {
		return n.Type() == "function_definition" && n.Parent() != nil && n.Parent().Type() == "decorated_definition"
	})

	if funcNode == nil {
		t.Fatal("No decorated function found")
	}

	node := parsePythonFunctionDefinition(funcNode, []byte(code), graph, "test.py")

	if node.Type != "property" {
		t.Errorf("Expected type 'property', got %s", node.Type)
	}
	if len(node.Annotation) == 0 || node.Annotation[0] != "property" {
		t.Errorf("Expected decorator 'property', got %v", node.Annotation)
	}
}

func TestParsePythonAssignment_ModuleLevel(t *testing.T) {
	code := "version = '1.0.0'"

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

	assignNode := findNodeByType(root, "assignment")

	if assignNode == nil {
		t.Fatal("No assignment node found")
	}

	// No context = module level.
	parsePythonAssignment(assignNode, []byte(code), graph, "test.py", nil)

	var node *Node
	for _, n := range graph.Nodes {
		if n.Name == "version" {
			node = n
			break
		}
	}

	if node == nil {
		t.Fatal("No variable node found")
	}

	if node.Type != "module_variable" {
		t.Errorf("Expected type 'module_variable', got %s", node.Type)
	}
	if node.Scope != "module" {
		t.Errorf("Expected scope 'module', got %s", node.Scope)
	}
}

func TestParsePythonAssignment_Constant(t *testing.T) {
	code := "MAX_CONNECTIONS = 100"

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
	assignNode = findNodeByType(root, "assignment")

	if assignNode == nil {
		t.Fatal("No assignment node found")
	}

	parsePythonAssignment(assignNode, []byte(code), graph, "test.py", nil)

	var node *Node
	for _, n := range graph.Nodes {
		if n.Name == "MAX_CONNECTIONS" {
			node = n
			break
		}
	}

	if node == nil {
		t.Fatal("No variable node found")
	}

	if node.Type != "constant" {
		t.Errorf("Expected type 'constant', got %s", node.Type)
	}
}

func TestParsePythonAssignment_ClassField(t *testing.T) {
	code := `
class Config:
    debug = True
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer tree.Close()

	graph := NewCodeGraph()

	// Create a class context.
	classContext := &Node{
		Type: "class_definition",
		Name: "Config",
	}

	root := tree.RootNode()
	var assignNode *sitter.Node
	assignNode = findNodeByType(root, "assignment")

	if assignNode == nil {
		t.Fatal("No assignment node found")
	}

	parsePythonAssignment(assignNode, []byte(code), graph, "test.py", classContext)

	var node *Node
	for _, n := range graph.Nodes {
		if n.Name == "debug" {
			node = n
			break
		}
	}

	if node == nil {
		t.Fatal("No variable node found")
	}

	if node.Type != "class_field" {
		t.Errorf("Expected type 'class_field', got %s", node.Type)
	}
	if node.Scope != "class" {
		t.Errorf("Expected scope 'class', got %s", node.Scope)
	}
}

func TestParsePythonAssignment_Local(t *testing.T) {
	code := `
def process():
    result = calculate()
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	defer tree.Close()

	graph := NewCodeGraph()

	// Create a function context.
	funcContext := &Node{
		Type: "function_definition",
		Name: "process",
	}

	root := tree.RootNode()
	var assignNode *sitter.Node
	assignNode = findNodeByType(root, "assignment")

	if assignNode == nil {
		t.Fatal("No assignment node found")
	}

	parsePythonAssignment(assignNode, []byte(code), graph, "test.py", funcContext)

	var node *Node
	for _, n := range graph.Nodes {
		if n.Name == "result" {
			node = n
			break
		}
	}

	if node == nil {
		t.Fatal("No variable node found")
	}

	if node.Type != "variable_assignment" {
		t.Errorf("Expected type 'variable_assignment', got %s", node.Type)
	}
	if node.Scope != "local" {
		t.Errorf("Expected scope 'local', got %s", node.Scope)
	}
}

func TestParsePythonClassDefinition_ReturnsNode(t *testing.T) {
	code := "class TestClass:\n    pass"

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

	node := parsePythonClassDefinition(classNode, []byte(code), graph, "test.py")

	if node == nil {
		t.Fatal("parsePythonClassDefinition should return a node")
	}
	if node.Type != "class_definition" {
		t.Errorf("Expected type 'class_definition', got %s", node.Type)
	}
	if node.Name != "TestClass" {
		t.Errorf("Expected name 'TestClass', got %s", node.Name)
	}
}

