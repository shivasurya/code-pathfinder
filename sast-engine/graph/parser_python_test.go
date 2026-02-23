package graph

import (
	"context"
	"slices"
	"strings"
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

			node := parsePythonFunctionDefinition(funcNode, []byte(tt.code), graph, "test.py", nil)

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
			name:     "Single decorator",
			code:     "@property\ndef get_value(self):\n    return self.value",
			expected: []string{"property"},
		},
		{
			name:     "Multiple decorators",
			code:     "@classmethod\n@cache\ndef compute(cls):\n    pass",
			expected: []string{"classmethod", "cache"},
		},
		{
			name:     "Decorator with arguments",
			code:     "@app.route('/api/users')\ndef get_users():\n    pass",
			expected: []string{"app.route"},
		},
		{
			name:     "No decorators",
			code:     "def simple():\n    pass",
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

	node := parsePythonFunctionDefinition(initNode, []byte(code), graph, "test.py", nil)

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

	node := parsePythonFunctionDefinition(funcNode, []byte(code), graph, "test.py", nil)

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

func TestIsSpecialMethod(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		expected bool
	}{
		// Valid special methods.
		{"__str__", "__str__", true},
		{"__add__", "__add__", true},
		{"__call__", "__call__", true},
		{"__getitem__", "__getitem__", true},
		{"__setitem__", "__setitem__", true},
		{"__len__", "__len__", true},
		{"__repr__", "__repr__", true},
		{"__eq__", "__eq__", true},

		// Invalid - not special methods.
		{"Regular method", "get_value", false},
		{"Private method", "_private", false},
		{"Dunder prefix only", "__init", false},
		{"Dunder suffix only", "init__", false},
		{"Single underscore", "_", false},
		{"Double underscore only", "__", false},
		{"Triple underscore", "___", false},
		{"Empty string", "", false},
		{"__init__ is constructor", "__init__", true}, // Still matches pattern.
		{"Too short", "____", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSpecialMethod(tt.funcName)
			if result != tt.expected {
				t.Errorf("isSpecialMethod(%q) = %v, expected %v", tt.funcName, result, tt.expected)
			}
		})
	}
}

func TestIsInterface(t *testing.T) {
	tests := []struct {
		name         string
		superClasses []string
		expected     bool
	}{
		{"Protocol direct", []string{"Protocol"}, true},
		{"typing.Protocol qualified", []string{"typing.Protocol"}, true},
		{"ABC direct", []string{"ABC"}, true},
		{"abc.ABC qualified", []string{"abc.ABC"}, true},
		{"Multiple with Protocol", []string{"BaseClass", "Protocol"}, true},
		{"Multiple with ABC", []string{"BaseClass", "ABC"}, true},
		{"Custom.Protocol", []string{"mymodule.Protocol"}, true},
		{"Regular class", []string{"BaseClass"}, false},
		{"Empty superclass", []string{}, false},
		{"Nil superclass", nil, false},
		{"Protocol in name but not suffix", []string{"ProtocolBase"}, false},
		{"ABC in name but not suffix", []string{"ABCMeta"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInterface(tt.superClasses)
			if result != tt.expected {
				t.Errorf("isInterface(%v) = %v, expected %v", tt.superClasses, result, tt.expected)
			}
		})
	}
}

func TestIsEnum(t *testing.T) {
	tests := []struct {
		name         string
		superClasses []string
		expected     bool
	}{
		{"Enum direct", []string{"Enum"}, true},
		{"enum.Enum qualified", []string{"enum.Enum"}, true},
		{"IntEnum", []string{"IntEnum"}, true},
		{"enum.IntEnum", []string{"enum.IntEnum"}, true},
		{"Flag", []string{"Flag"}, true},
		{"enum.Flag", []string{"enum.Flag"}, true},
		{"IntFlag", []string{"IntFlag"}, true},
		{"enum.IntFlag", []string{"enum.IntFlag"}, true},
		{"Multiple with Enum", []string{"Mixin", "Enum"}, true},
		{"Regular class", []string{"BaseClass"}, false},
		{"Empty superclass", []string{}, false},
		{"Nil superclass", nil, false},
		{"Enum in name but not suffix", []string{"EnumBase"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEnum(tt.superClasses)
			if result != tt.expected {
				t.Errorf("isEnum(%v) = %v, expected %v", tt.superClasses, result, tt.expected)
			}
		})
	}
}

func TestIsDataclass(t *testing.T) {
	tests := []struct {
		name       string
		decorators []string
		expected   bool
	}{
		{"dataclass decorator", []string{"dataclass"}, true},
		{"dataclasses.dataclass qualified", []string{"dataclasses.dataclass"}, true},
		{"Multiple with dataclass", []string{"frozen", "dataclass"}, true},
		{"Custom.dataclass", []string{"mymodule.dataclass"}, true},
		{"No decorators", []string{}, false},
		{"Nil decorators", nil, false},
		{"Other decorators", []string{"property", "cache"}, false},
		{"dataclass in name but not suffix", []string{"dataclass_utils"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDataclass(tt.decorators)
			if result != tt.expected {
				t.Errorf("isDataclass(%v) = %v, expected %v", tt.decorators, result, tt.expected)
			}
		})
	}
}

func TestParsePythonFunctionDefinition_SpecialMethod(t *testing.T) {
	code := `
class User:
    def __str__(self):
        return f"User: {self.name}"
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

	// Find __str__ method.
	strNode := findNodeByCondition(root, func(n *sitter.Node) bool {
		if n.Type() == "function_definition" {
			nameNode := n.ChildByFieldName("name")
			return nameNode != nil && nameNode.Content([]byte(code)) == "__str__"
		}
		return false
	})

	if strNode == nil {
		t.Fatal("No __str__ method found")
	}

	node := parsePythonFunctionDefinition(strNode, []byte(code), graph, "test.py", nil)

	if node.Type != "special_method" {
		t.Errorf("Expected type 'special_method', got %s", node.Type)
	}
	if node.Name != "__str__" {
		t.Errorf("Expected name '__str__', got %s", node.Name)
	}
}

func TestParsePythonClassDefinition_Interface(t *testing.T) {
	code := `
from typing import Protocol

class Drawable(Protocol):
    def draw(self):
        pass
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

	classNode := findNodeByType(root, "class_definition")
	if classNode == nil {
		t.Fatal("No class_definition node found")
	}

	node := parsePythonClassDefinition(classNode, []byte(code), graph, "test.py")

	if node.Type != "interface" {
		t.Errorf("Expected type 'interface', got %s", node.Type)
	}
	if node.Name != "Drawable" {
		t.Errorf("Expected name 'Drawable', got %s", node.Name)
	}
	if len(node.Interface) == 0 || node.Interface[0] != "Protocol" {
		t.Errorf("Expected superclass 'Protocol', got %v", node.Interface)
	}
}

func TestParsePythonClassDefinition_Enum(t *testing.T) {
	code := `
from enum import Enum

class Color(Enum):
    RED = 1
    GREEN = 2
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

	classNode := findNodeByType(root, "class_definition")
	if classNode == nil {
		t.Fatal("No class_definition node found")
	}

	node := parsePythonClassDefinition(classNode, []byte(code), graph, "test.py")

	if node.Type != "enum" {
		t.Errorf("Expected type 'enum', got %s", node.Type)
	}
	if node.Name != "Color" {
		t.Errorf("Expected name 'Color', got %s", node.Name)
	}
}

func TestParsePythonClassDefinition_Dataclass(t *testing.T) {
	code := `
from dataclasses import dataclass

@dataclass
class Point:
    x: int
    y: int
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

	// Find decorated class.
	classNode := findNodeByCondition(root, func(n *sitter.Node) bool {
		return n.Type() == "class_definition" && n.Parent() != nil && n.Parent().Type() == "decorated_definition"
	})

	if classNode == nil {
		t.Fatal("No decorated class found")
	}

	node := parsePythonClassDefinition(classNode, []byte(code), graph, "test.py")

	if node.Type != "dataclass" {
		t.Errorf("Expected type 'dataclass', got %s", node.Type)
	}
	if node.Name != "Point" {
		t.Errorf("Expected name 'Point', got %s", node.Name)
	}
	if len(node.Annotation) == 0 || node.Annotation[0] != "dataclass" {
		t.Errorf("Expected decorator 'dataclass', got %v", node.Annotation)
	}
}

func TestParsePythonClassDefinition_Regular(t *testing.T) {
	// Ensure regular classes without special inheritance still work.
	code := `
class RegularClass:
    def method(self):
        pass
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

	classNode := findNodeByType(root, "class_definition")
	if classNode == nil {
		t.Fatal("No class_definition node found")
	}

	node := parsePythonClassDefinition(classNode, []byte(code), graph, "test.py")

	if node.Type != "class_definition" {
		t.Errorf("Expected type 'class_definition', got %s", node.Type)
	}
	if node.Name != "RegularClass" {
		t.Errorf("Expected name 'RegularClass', got %s", node.Name)
	}
}

func TestParsePythonClassDefinition_ABC(t *testing.T) {
	code := `
from abc import ABC

class AbstractBase(ABC):
    pass
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

	classNode := findNodeByType(root, "class_definition")
	if classNode == nil {
		t.Fatal("No class_definition node found")
	}

	node := parsePythonClassDefinition(classNode, []byte(code), graph, "test.py")

	if node.Type != "interface" {
		t.Errorf("Expected type 'interface' for ABC, got %s", node.Type)
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

// TestMultipleConstructorsInSameFile verifies that multiple __init__ methods
// in the same file are all indexed (regression test for constructor bug where
// only one constructor per file was being captured).
func TestMultipleConstructorsInSameFile(t *testing.T) {
	code := `
class FirstClass:
    def __init__(self):
        self.name = "first"

class SecondClass:
    def __init__(self, value):
        self.value = value

class ThirdClass:
    def __init__(self):
        pass

class FourthClass:
    def __init__(self, x, y):
        self.x = x
        self.y = y
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
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count constructors in the graph.
	constructorCount := 0
	constructorIDs := make(map[string]bool)
	lineNumbers := make(map[uint32]bool)

	for _, node := range graph.Nodes {
		if node.Type == "constructor" {
			constructorCount++

			// Check for duplicate IDs (would indicate overwriting).
			if constructorIDs[node.ID] {
				t.Errorf("Duplicate constructor ID found: %s", node.ID)
			}
			constructorIDs[node.ID] = true

			// Track line numbers.
			lineNumbers[node.LineNumber] = true
		}
	}

	// Verify all 4 constructors were indexed.
	if constructorCount != 4 {
		t.Errorf("Expected 4 constructors, got %d", constructorCount)
	}

	// Verify all have unique IDs.
	if len(constructorIDs) != 4 {
		t.Errorf("Expected 4 unique constructor IDs, got %d", len(constructorIDs))
	}

	// Verify all have different line numbers.
	if len(lineNumbers) != 4 {
		t.Errorf("Expected 4 different line numbers, got %d", len(lineNumbers))
	}
}

// TestMultipleMethodsWithSameName verifies that methods with the same name
// in different classes are all indexed separately.
func TestMultipleMethodsWithSameName(t *testing.T) {
	code := `
class User:
    def save(self):
        pass

class Product:
    def save(self):
        pass

class Order:
    def save(self):
        pass
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
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count methods named "save".
	saveMethodCount := 0
	saveMethodIDs := make(map[string]bool)

	for _, node := range graph.Nodes {
		if node.Type == "method" && node.Name == "save" {
			saveMethodCount++

			// Check for duplicate IDs.
			if saveMethodIDs[node.ID] {
				t.Errorf("Duplicate ID for save method: %s", node.ID)
			}
			saveMethodIDs[node.ID] = true
		}
	}

	// Verify all 3 save methods were indexed.
	if saveMethodCount != 3 {
		t.Errorf("Expected 3 'save' methods, got %d", saveMethodCount)
	}

	// Verify all have unique IDs.
	if len(saveMethodIDs) != 3 {
		t.Errorf("Expected 3 unique save method IDs, got %d", len(saveMethodIDs))
	}
}

// TestResolveTransitiveEnumInheritance tests that classes inheriting from
// custom enum base classes are properly detected as enums.
func TestResolveTransitiveEnumInheritance(t *testing.T) {
	code := `
from enum import Enum

class CustomEnum(Enum):
    """Custom enum base class."""
    pass

class Operator(CustomEnum):
    """Operator enum - inherits from CustomEnum."""
    ADD = "add"
    SUBTRACT = "subtract"

class Type(CustomEnum):
    """Type enum - inherits from CustomEnum."""
    INTEGER = "int"
    STRING = "str"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Before resolving transitive inheritance.
	customEnumNode := findNodeByNameAndType(graph, "CustomEnum", "enum")
	operatorNode := findNodeByNameAndType(graph, "Operator", "class_definition")
	typeNode := findNodeByNameAndType(graph, "Type", "class_definition")

	if customEnumNode == nil {
		t.Errorf("CustomEnum should be detected as enum (direct inheritance)")
	}
	if operatorNode == nil {
		t.Errorf("Operator should initially be class_definition before resolving")
	}
	if typeNode == nil {
		t.Errorf("Type should initially be class_definition before resolving")
	}

	// Resolve transitive inheritance.
	ResolveTransitiveInheritance(graph)

	// After resolving transitive inheritance.
	operatorNodeAfter := findNodeByNameAndType(graph, "Operator", "enum")
	typeNodeAfter := findNodeByNameAndType(graph, "Type", "enum")

	if operatorNodeAfter == nil {
		t.Errorf("Operator should be detected as enum after resolving transitive inheritance")
	}
	if typeNodeAfter == nil {
		t.Errorf("Type should be detected as enum after resolving transitive inheritance")
	}

	// Verify CustomEnum is still an enum.
	customEnumNodeAfter := findNodeByNameAndType(graph, "CustomEnum", "enum")
	if customEnumNodeAfter == nil {
		t.Errorf("CustomEnum should still be enum after resolving")
	}
}

// TestResolveTransitiveInterfaceInheritance tests that classes inheriting from
// custom interface base classes are properly detected as interfaces.
func TestResolveTransitiveInterfaceInheritance(t *testing.T) {
	code := `
from typing import Protocol

class BaseProtocol(Protocol):
    """Base protocol."""
    pass

class ExtendedProtocol(BaseProtocol):
    """Extended protocol - inherits from BaseProtocol."""
    def method(self) -> None:
        pass
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Resolve transitive inheritance.
	ResolveTransitiveInheritance(graph)

	// Verify both are detected as interfaces.
	baseProtocol := findNodeByNameAndType(graph, "BaseProtocol", "interface")
	extendedProtocol := findNodeByNameAndType(graph, "ExtendedProtocol", "interface")

	if baseProtocol == nil {
		t.Errorf("BaseProtocol should be detected as interface")
	}
	if extendedProtocol == nil {
		t.Errorf("ExtendedProtocol should be detected as interface after resolving transitive inheritance")
	}
}

// TestResolveTransitiveDataclassInheritance tests that classes inheriting from
// dataclass base classes are properly detected as dataclasses.
func TestResolveTransitiveDataclassInheritance(t *testing.T) {
	code := `
from dataclasses import dataclass

@dataclass
class BaseConfig:
    """Base configuration dataclass."""
    debug: bool = False

class AppConfig(BaseConfig):
    """App configuration - inherits from BaseConfig."""
    timeout: int = 30
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Resolve transitive inheritance.
	ResolveTransitiveInheritance(graph)

	// Verify both are detected as dataclasses.
	baseConfig := findNodeByNameAndType(graph, "BaseConfig", "dataclass")
	appConfig := findNodeByNameAndType(graph, "AppConfig", "dataclass")

	if baseConfig == nil {
		t.Errorf("BaseConfig should be detected as dataclass")
	}
	if appConfig == nil {
		t.Errorf("AppConfig should be detected as dataclass after resolving transitive inheritance")
	}
}

// TestResolveTransitiveMultiLevelInheritance tests that multi-level inheritance works.
func TestResolveTransitiveMultiLevelInheritance(t *testing.T) {
	code := `
from enum import Enum

class CustomEnum(Enum):
    """Level 1 - Direct enum."""
    pass

class MiddleEnum(CustomEnum):
    """Level 2 - Inherits from CustomEnum."""
    pass

class LeafEnum(MiddleEnum):
    """Level 3 - Inherits from MiddleEnum."""
    VALUE = 1
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Resolve transitive inheritance.
	ResolveTransitiveInheritance(graph)

	// Verify all three are detected as enums.
	customEnum := findNodeByNameAndType(graph, "CustomEnum", "enum")
	middleEnum := findNodeByNameAndType(graph, "MiddleEnum", "enum")
	leafEnum := findNodeByNameAndType(graph, "LeafEnum", "enum")

	if customEnum == nil {
		t.Errorf("CustomEnum should be detected as enum")
	}
	if middleEnum == nil {
		t.Errorf("MiddleEnum should be detected as enum after resolving transitive inheritance")
	}
	if leafEnum == nil {
		t.Errorf("LeafEnum should be detected as enum after resolving 3-level transitive inheritance")
	}
}

// TestResolveTransitiveInheritanceLabelStudioCase tests the exact case from label-studio.
func TestResolveTransitiveInheritanceLabelStudioCase(t *testing.T) {
	code := `
from enum import Enum

class ConjunctionEnum(Enum):
    """Direct enum - should be detected."""
    AND = "and"
    OR = "or"

class CustomEnum(Enum):
    """Custom enum base class."""
    pass

class Operator(CustomEnum):
    """Operator enum - inherits from CustomEnum."""
    EQUAL = "equal"
    NOT_EQUAL = "not_equal"

class Type(CustomEnum):
    """Type enum - inherits from CustomEnum."""
    STRING = "string"
    NUMBER = "number"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Before resolving, count enums.
	enumCountBefore := 0
	for _, node := range graph.Nodes {
		if node.Type == "enum" {
			enumCountBefore++
		}
	}

	// Should have 2 direct enums (ConjunctionEnum, CustomEnum).
	if enumCountBefore != 2 {
		t.Errorf("Expected 2 enums before resolving, got %d", enumCountBefore)
	}

	// Resolve transitive inheritance.
	ResolveTransitiveInheritance(graph)

	// After resolving, count enums.
	enumCountAfter := 0
	enumNames := []string{}
	for _, node := range graph.Nodes {
		if node.Type == "enum" {
			enumCountAfter++
			enumNames = append(enumNames, node.Name)
		}
	}

	// Should have 4 enums (ConjunctionEnum, CustomEnum, Operator, Type).
	if enumCountAfter != 4 {
		t.Errorf("Expected 4 enums after resolving, got %d: %v", enumCountAfter, enumNames)
	}

	// Verify specific enums.
	if findNodeByNameAndType(graph, "ConjunctionEnum", "enum") == nil {
		t.Errorf("ConjunctionEnum should be enum")
	}
	if findNodeByNameAndType(graph, "CustomEnum", "enum") == nil {
		t.Errorf("CustomEnum should be enum")
	}
	if findNodeByNameAndType(graph, "Operator", "enum") == nil {
		t.Errorf("Operator should be enum after resolving")
	}
	if findNodeByNameAndType(graph, "Type", "enum") == nil {
		t.Errorf("Type should be enum after resolving")
	}
}

// TestResolveTransitiveInheritanceNoChange tests that regular classes are not affected.
func TestResolveTransitiveInheritanceNoChange(t *testing.T) {
	code := `
class BaseClass:
    """Regular base class."""
    pass

class DerivedClass(BaseClass):
    """Regular derived class."""
    pass
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Resolve transitive inheritance.
	ResolveTransitiveInheritance(graph)

	// Verify both are still class_definition.
	baseClass := findNodeByNameAndType(graph, "BaseClass", "class_definition")
	derivedClass := findNodeByNameAndType(graph, "DerivedClass", "class_definition")

	if baseClass == nil {
		t.Errorf("BaseClass should remain as class_definition")
	}
	if derivedClass == nil {
		t.Errorf("DerivedClass should remain as class_definition")
	}
}

// TestClassLevelConstants tests that class-level constants are properly detected.
func TestClassLevelConstants(t *testing.T) {
	code := `
class WebhookAction:
    """Webhook action constants."""
    PROJECT_CREATED = "project_created"
    PROJECT_UPDATED = "project_updated"
    TASK_CREATED = "task_created"

    def get_action(self):
        return self.PROJECT_CREATED
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count class-level constants.
	constantCount := 0
	constantNames := []string{}

	for _, node := range graph.Nodes {
		if node.Type == "constant" && node.Scope == "class" {
			constantCount++
			constantNames = append(constantNames, node.Name)
		}
	}

	// Should have 3 class-level constants.
	if constantCount != 3 {
		t.Errorf("Expected 3 class-level constants, got %d: %v", constantCount, constantNames)
	}

	// Verify specific constants.
	expectedConstants := []string{"PROJECT_CREATED", "PROJECT_UPDATED", "TASK_CREATED"}
	for _, expected := range expectedConstants {
		found := slices.Contains(constantNames, expected)
		if !found {
			t.Errorf("Expected class-level constant %s not found", expected)
		}
	}
}

// TestEnumClassConstants tests that enum class value assignments are detected as constants.
func TestEnumClassConstants(t *testing.T) {
	code := `
from enum import Enum

class ConjunctionEnum(Enum):
    OR = 'or'
    AND = 'and'

class Column(Enum):
    ID = 'id'
    INNER_ID = 'inner_id'
    DATA = 'data'
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count constants inside enum classes.
	constantCount := 0
	constantsByClass := make(map[string][]string)

	for _, node := range graph.Nodes {
		if node.Type == "constant" && node.Scope == "class" {
			constantCount++
			// Group by file location to approximate class membership.
			constantsByClass["all"] = append(constantsByClass["all"], node.Name)
		}
	}

	// Should have 5 enum value constants (OR, AND, ID, INNER_ID, DATA).
	if constantCount != 5 {
		t.Errorf("Expected 5 enum value constants, got %d: %v", constantCount, constantsByClass["all"])
	}

	// Verify specific constants.
	expectedConstants := []string{"OR", "AND", "ID", "INNER_ID", "DATA"}
	for _, expected := range expectedConstants {
		found := slices.Contains(constantsByClass["all"], expected)
		if !found {
			t.Errorf("Expected enum constant %s not found", expected)
		}
	}
}

// TestClassFieldVsConstantDistinction tests that class fields and constants are properly distinguished.
func TestClassFieldVsConstantDistinction(t *testing.T) {
	code := `
class Project:
    """Project model."""
    # Class-level constants (UPPERCASE)
    SEQUENCE = "project_sequence"
    MAX_TASKS = 1000

    # Class-level fields (lowercase/mixed case)
    default_timeout = 30
    _internal_cache = {}

    def __init__(self):
        self.name = "test"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count constants vs fields.
	constantCount := 0
	fieldCount := 0
	constantNames := []string{}
	fieldNames := []string{}

	for _, node := range graph.Nodes {
		if node.Scope == "class" {
			if node.Type == "constant" {
				constantCount++
				constantNames = append(constantNames, node.Name)
			} else if node.Type == "class_field" {
				fieldCount++
				fieldNames = append(fieldNames, node.Name)
			}
		}
	}

	// Should have 2 constants (SEQUENCE, MAX_TASKS).
	if constantCount != 2 {
		t.Errorf("Expected 2 class constants, got %d: %v", constantCount, constantNames)
	}

	// Should have 2 fields (default_timeout, _internal_cache).
	if fieldCount != 2 {
		t.Errorf("Expected 2 class fields, got %d: %v", fieldCount, fieldNames)
	}

	// Verify specific constants.
	if !containsString(constantNames, "SEQUENCE") {
		t.Errorf("Expected constant SEQUENCE not found")
	}
	if !containsString(constantNames, "MAX_TASKS") {
		t.Errorf("Expected constant MAX_TASKS not found")
	}

	// Verify specific fields.
	if !containsString(fieldNames, "default_timeout") {
		t.Errorf("Expected field default_timeout not found")
	}
	if !containsString(fieldNames, "_internal_cache") {
		t.Errorf("Expected field _internal_cache not found")
	}
}

// TestLabelStudioClassConstants tests real examples from label-studio codebase.
func TestLabelStudioClassConstants(t *testing.T) {
	code := `
class ProjectManager:
    """Project manager with counter fields."""
    COUNTER_FIELDS = ['num_tasks', 'num_annotations']
    CACHE_TIMEOUT = 300

class SkillNames:
    """ML skill name constants."""
    TEXT_CLASSIFICATION = "text_classification"
    NAMED_ENTITY_RECOGNITION = "ner"
    IMAGE_SEGMENTATION = "image_segmentation"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count class-level constants.
	constantCount := 0
	constantNames := []string{}

	for _, node := range graph.Nodes {
		if node.Type == "constant" && node.Scope == "class" {
			constantCount++
			constantNames = append(constantNames, node.Name)
		}
	}

	// Should have 5 class-level constants.
	if constantCount != 5 {
		t.Errorf("Expected 5 class-level constants, got %d: %v", constantCount, constantNames)
	}

	// Verify specific constants from both classes.
	expectedConstants := []string{
		"COUNTER_FIELDS", "CACHE_TIMEOUT",
		"TEXT_CLASSIFICATION", "NAMED_ENTITY_RECOGNITION", "IMAGE_SEGMENTATION",
	}
	for _, expected := range expectedConstants {
		if !containsString(constantNames, expected) {
			t.Errorf("Expected class constant %s not found in %v", expected, constantNames)
		}
	}
}

// TestModuleVsClassConstants tests that module and class constants are both detected and distinguished.
func TestModuleVsClassConstants(t *testing.T) {
	code := `
# Module-level constants
API_VERSION = "v1"
MAX_RETRIES = 3

class Settings:
    """Settings class with constants."""
    # Class-level constants
    DEBUG = True
    PORT = 8080
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count constants by scope.
	moduleConstants := 0
	classConstants := 0
	moduleNames := []string{}
	classNames := []string{}

	for _, node := range graph.Nodes {
		if node.Type == "constant" {
			if node.Scope == "module" {
				moduleConstants++
				moduleNames = append(moduleNames, node.Name)
			} else if node.Scope == "class" {
				classConstants++
				classNames = append(classNames, node.Name)
			}
		}
	}

	// Should have 2 module constants.
	if moduleConstants != 2 {
		t.Errorf("Expected 2 module constants, got %d: %v", moduleConstants, moduleNames)
	}

	// Should have 2 class constants.
	if classConstants != 2 {
		t.Errorf("Expected 2 class constants, got %d: %v", classConstants, classNames)
	}

	// Verify module constants.
	if !containsString(moduleNames, "API_VERSION") || !containsString(moduleNames, "MAX_RETRIES") {
		t.Errorf("Module constants incorrect: %v", moduleNames)
	}

	// Verify class constants.
	if !containsString(classNames, "DEBUG") || !containsString(classNames, "PORT") {
		t.Errorf("Class constants incorrect: %v", classNames)
	}
}

// Helper function to check if a slice contains a string.
func containsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// TestSubscriptAssignmentNotIndexed tests that subscript assignments are not indexed as module variables.
func TestSubscriptAssignmentNotIndexed(t *testing.T) {
	code := `
# Simple variable assignments (should be indexed)
DATABASES_ALL = {}
STORAGES = {}
config_dict = {}

# Subscript assignments (should NOT be indexed as variables)
DATABASES_ALL['default'] = {'ENGINE': 'django.db.backends.postgresql'}
STORAGES['default']['BACKEND'] = 'django.core.files.storage.FileSystemStorage'
config_dict['key'] = 'value'
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count module-level variables and constants
	moduleVars := []string{}
	constants := []string{}

	for _, node := range graph.Nodes {
		if node.Scope == "module" {
			if node.Type == "module_variable" {
				moduleVars = append(moduleVars, node.Name)
			} else if node.Type == "constant" {
				constants = append(constants, node.Name)
			}
		}
	}

	// Should have 1 module variable (config_dict)
	if len(moduleVars) != 1 {
		t.Errorf("Expected 1 module_variable, got %d: %v", len(moduleVars), moduleVars)
	}
	if !containsString(moduleVars, "config_dict") {
		t.Errorf("Expected config_dict in module variables, got: %v", moduleVars)
	}

	// Should have 2 constants (DATABASES_ALL, STORAGES)
	if len(constants) != 2 {
		t.Errorf("Expected 2 constants, got %d: %v", len(constants), constants)
	}
	if !containsString(constants, "DATABASES_ALL") {
		t.Errorf("Expected DATABASES_ALL in constants, got: %v", constants)
	}
	if !containsString(constants, "STORAGES") {
		t.Errorf("Expected STORAGES in constants, got: %v", constants)
	}

	// Should NOT have subscript assignments as variables
	allNames := make([]string, 0, len(moduleVars)+len(constants))
	allNames = append(allNames, moduleVars...)
	allNames = append(allNames, constants...)
	for _, name := range allNames {
		if strings.Contains(name, "[") || strings.Contains(name, "'") {
			t.Errorf("Found subscript assignment indexed as variable: %s", name)
		}
	}
}

// TestAttributeAssignmentNotIndexed tests that attribute assignments are not indexed as module variables.
func TestAttributeAssignmentNotIndexed(t *testing.T) {
	code := `
# Simple variable assignment (should be indexed)
settings = object()

# Attribute assignments (should NOT be indexed as variables)
settings.DATA_MANAGER = {}
settings.FEATURE_FLAGS = True
obj.field = "value"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count module-level variables
	moduleVars := []string{}

	for _, node := range graph.Nodes {
		if node.Scope == "module" && node.Type == "module_variable" {
			moduleVars = append(moduleVars, node.Name)
		}
	}

	// Should have 1 module variable (settings)
	if len(moduleVars) != 1 {
		t.Errorf("Expected 1 module_variable, got %d: %v", len(moduleVars), moduleVars)
	}
	if !containsString(moduleVars, "settings") {
		t.Errorf("Expected settings in module variables, got: %v", moduleVars)
	}

	// Should NOT have attribute assignments as variables
	for _, name := range moduleVars {
		if strings.Contains(name, ".") {
			t.Errorf("Found attribute assignment indexed as variable: %s", name)
		}
	}
}

// TestLabelStudioSubscriptCases tests real cases from label-studio codebase.
func TestLabelStudioSubscriptCases(t *testing.T) {
	code := `
# From core/settings/base.py
DATABASES_ALL = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
    }
}

# Subscript assignment (should NOT be indexed)
DATABASES_ALL['default'] = {
    'ENGINE': 'django.db.backends.postgresql',
}

# From core/feature_flags/base.py
store_kwargs = {}

# Subscript assignment (should NOT be indexed)
store_kwargs['redis_opts'] = {'host': 'localhost'}

# From core/settings/base.py
LOGGING = {'handlers': {}}

# Nested subscript assignments (should NOT be indexed)
LOGGING['handlers']['google_cloud_logging'] = {}
LOGGING['root']['level'] = 'INFO'
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count module-level variables and constants
	moduleVars := []string{}
	constants := []string{}

	for _, node := range graph.Nodes {
		if node.Scope == "module" {
			if node.Type == "module_variable" {
				moduleVars = append(moduleVars, node.Name)
			} else if node.Type == "constant" {
				constants = append(constants, node.Name)
			}
		}
	}

	// Should have 1 module variable (store_kwargs)
	if len(moduleVars) != 1 {
		t.Errorf("Expected 1 module_variable, got %d: %v", len(moduleVars), moduleVars)
	}

	// Should have 3 constants (DATABASES_ALL, LOGGING, STORAGES not in this test)
	if len(constants) != 2 {
		t.Errorf("Expected 2 constants, got %d: %v", len(constants), constants)
	}

	// Total module-level should be 3 (not 8 if subscripts were counted)
	totalModuleLevel := len(moduleVars) + len(constants)
	if totalModuleLevel != 3 {
		t.Errorf("Expected 3 total module-level declarations, got %d", totalModuleLevel)
	}

	// Verify no subscript syntax in variable names
	allNames := make([]string, 0, len(moduleVars)+len(constants))
	allNames = append(allNames, moduleVars...)
	allNames = append(allNames, constants...)
	for _, name := range allNames {
		if strings.Contains(name, "[") || strings.Contains(name, "]") || strings.Contains(name, "'") {
			t.Errorf("Subscript assignment incorrectly indexed as variable: %s", name)
		}
	}
}

// TestLabelStudioAttributeCases tests real attribute assignment cases from label-studio.
func TestLabelStudioAttributeCases(t *testing.T) {
	code := `
# From manage.py
from django.core.management.commands import runserver

# Attribute assignment (should NOT be indexed)
runserver.default_port = "8080"

# From data_manager/managers.py
import settings

# Attribute assignment (should NOT be indexed)
settings.DATA_MANAGER_ANNOTATIONS_MAP = {}
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count module-level variables
	moduleVars := []string{}

	for _, node := range graph.Nodes {
		if node.Scope == "module" && node.Type == "module_variable" {
			moduleVars = append(moduleVars, node.Name)
		}
	}

	// Should have 0 module variables (all are attribute assignments)
	if len(moduleVars) != 0 {
		t.Errorf("Expected 0 module_variables, got %d: %v", len(moduleVars), moduleVars)
	}

	// Verify no dotted names
	for _, name := range moduleVars {
		if strings.Contains(name, ".") {
			t.Errorf("Attribute assignment incorrectly indexed as variable: %s", name)
		}
	}
}

// TestMixedAssignmentTypes tests combination of identifier, subscript, and attribute assignments.
func TestMixedAssignmentTypes(t *testing.T) {
	code := `
# Simple identifiers (should be indexed)
simple_var = 123
SIMPLE_CONST = "value"

# Subscript assignments (should NOT be indexed)
data = {}
data['key'] = 'value'

# Attribute assignments (should NOT be indexed)
obj = object()
obj.field = "value"

# More simple identifiers (should be indexed)
another_var = 456
ANOTHER_CONST = "constant"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, _ := parser.ParseCtx(context.TODO(), nil, []byte(code))
	defer tree.Close()

	graph := NewCodeGraph()
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Count module-level variables and constants
	moduleVars := []string{}
	constants := []string{}

	for _, node := range graph.Nodes {
		if node.Scope == "module" {
			if node.Type == "module_variable" {
				moduleVars = append(moduleVars, node.Name)
			} else if node.Type == "constant" {
				constants = append(constants, node.Name)
			}
		}
	}

	// Should have 3 module variables (simple_var, another_var, data, obj)
	if len(moduleVars) != 4 {
		t.Errorf("Expected 4 module_variables, got %d: %v", len(moduleVars), moduleVars)
	}

	// Should have 2 constants (SIMPLE_CONST, ANOTHER_CONST)
	if len(constants) != 2 {
		t.Errorf("Expected 2 constants, got %d: %v", len(constants), constants)
	}

	// Verify expected variables
	expectedVars := []string{"simple_var", "another_var", "data", "obj"}
	for _, expected := range expectedVars {
		if !containsString(moduleVars, expected) {
			t.Errorf("Expected %s in module variables, got: %v", expected, moduleVars)
		}
	}

	// Verify expected constants
	expectedConsts := []string{"SIMPLE_CONST", "ANOTHER_CONST"}
	for _, expected := range expectedConsts {
		if !containsString(constants, expected) {
			t.Errorf("Expected %s in constants, got: %v", expected, constants)
		}
	}

	// Verify no subscript or attribute syntax
	allNames := make([]string, 0, len(moduleVars)+len(constants))
	allNames = append(allNames, moduleVars...)
	allNames = append(allNames, constants...)
	for _, name := range allNames {
		if strings.Contains(name, "[") || strings.Contains(name, ".") {
			t.Errorf("Non-identifier assignment incorrectly indexed: %s", name)
		}
	}
}

// TestNestedFunctionIndexing tests that nested functions are indexed with parent-qualified FQNs.
// This prevents ID collisions when multiple nested functions have the same name.
func TestNestedFunctionIndexing(t *testing.T) {
	sourceCode := []byte(`
def outer_function():
    """Outer function containing nested functions."""
    def inner_function():
        """Inner function inside outer."""
        def deeply_nested():
            """Deeply nested function."""
            return 42
        return deeply_nested()
    return inner_function()

def decorator_factory(action):
    """Decorator factory with nested functions."""
    def decorator(func):
        """Decorator function."""
        def wrapper(*args, **kwargs):
            """Wrapper function."""
            return func(*args, **kwargs)
        return wrapper
    return decorator

def another_decorator_factory(action):
    """Another decorator factory with same nested function names."""
    def decorator(func):
        """Another decorator function."""
        def wrapper(*args, **kwargs):
            """Another wrapper function."""
            return func(*args, **kwargs)
        return wrapper
    return decorator
`)

	graph := NewCodeGraph()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	defer tree.Close()

	buildGraphFromAST(tree.RootNode(), sourceCode, graph, nil, "test.py")

	// Expected functions with qualified names
	expectedFunctions := map[string]int{
		// Top-level functions
		"outer_function":            1,
		"decorator_factory":         1,
		"another_decorator_factory": 1,

		// Nested functions with parent qualification
		"outer_function.inner_function":               1,
		"outer_function.inner_function.deeply_nested": 1,
		"decorator_factory.decorator":                 1,
		"decorator_factory.decorator.wrapper":         1,
		"another_decorator_factory.decorator":         1,
		"another_decorator_factory.decorator.wrapper": 1,
	}

	// Count functions by name
	functionCounts := make(map[string]int)
	for _, node := range graph.Nodes {
		if node.Type == "function_definition" {
			functionCounts[node.Name]++
		}
	}

	// Verify all expected functions are present
	for expectedName, expectedCount := range expectedFunctions {
		actualCount, found := functionCounts[expectedName]
		if !found {
			t.Errorf("Expected function %q not found in graph", expectedName)
		} else if actualCount != expectedCount {
			t.Errorf("Function %q count mismatch: expected %d, got %d", expectedName, expectedCount, actualCount)
		}
	}

	// Verify no unqualified nested functions (collision check)
	for functionName := range functionCounts {
		// These should NOT exist as bare names (should be qualified)
		if functionName == "inner_function" {
			t.Errorf("Nested function 'inner_function' should be qualified as 'outer_function.inner_function'")
		}
		if functionName == "deeply_nested" {
			t.Errorf("Nested function 'deeply_nested' should be qualified as 'outer_function.inner_function.deeply_nested'")
		}
		if functionName == "decorator" {
			t.Errorf("Nested function 'decorator' should be qualified with parent name")
		}
		if functionName == "wrapper" {
			t.Errorf("Nested function 'wrapper' should be qualified with parent name")
		}
	}

	// Verify total function count
	totalFunctions := 0
	for _, count := range functionCounts {
		totalFunctions += count
	}
	expectedTotal := 9
	if totalFunctions != expectedTotal {
		t.Errorf("Total function count mismatch: expected %d, got %d", expectedTotal, totalFunctions)
		t.Logf("Indexed functions: %v", functionCounts)
	}
}

// TestNestedFunctionIDUniqueness tests that nested functions with the same name
// in different parent functions generate unique IDs.
func TestNestedFunctionIDUniqueness(t *testing.T) {
	sourceCode := []byte(`
def parent_a():
    def child():
        pass

def parent_b():
    def child():
        pass
`)

	graph := NewCodeGraph()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	defer tree.Close()

	buildGraphFromAST(tree.RootNode(), sourceCode, graph, nil, "test.py")

	// Find both child functions
	var childA, childB *Node
	for _, node := range graph.Nodes {
		if node.Name == "parent_a.child" {
			childA = node
		}
		if node.Name == "parent_b.child" {
			childB = node
		}
	}

	// Both should exist
	if childA == nil {
		t.Error("parent_a.child not found in graph")
	}
	if childB == nil {
		t.Error("parent_b.child not found in graph")
	}

	// IDs should be different
	if childA != nil && childB != nil {
		if childA.ID == childB.ID {
			t.Errorf("Nested functions with same name should have unique IDs, but both have ID: %s", childA.ID)
		}
		if childA.Name == childB.Name {
			t.Errorf("Nested functions should have parent-qualified names, but both have name: %s", childA.Name)
		}
	}
}

// TestParsePythonFunctionDefinition_ReturnType tests that return type annotations
// are correctly extracted from Python function definitions.
func TestParsePythonFunctionDefinition_ReturnType(t *testing.T) {
	tests := []struct {
		name               string
		code               string
		expectedName       string
		expectedReturnType string
	}{
		{
			name:               "Simple return type",
			code:               "def add(a: int, b: int) -> int:\n    return a + b",
			expectedName:       "add",
			expectedReturnType: "int",
		},
		{
			name:               "String return type",
			code:               "def greet(name: str) -> str:\n    return 'hello ' + name",
			expectedName:       "greet",
			expectedReturnType: "str",
		},
		{
			name:               "None return type",
			code:               "def setup() -> None:\n    pass",
			expectedName:       "setup",
			expectedReturnType: "None",
		},
		{
			name:               "No return type annotation",
			code:               "def process(x):\n    return x",
			expectedName:       "process",
			expectedReturnType: "",
		},
		{
			name:               "Complex return type - union",
			code:               "def safe_divide(a: float, b: float) -> float | None:\n    pass",
			expectedName:       "safe_divide",
			expectedReturnType: "float | None",
		},
		{
			name:               "Generic return type",
			code:               "def get_items() -> list[str]:\n    return []",
			expectedName:       "get_items",
			expectedReturnType: "list[str]",
		},
		{
			name:               "Tuple return type",
			code:               "def get_pair() -> tuple[int, str]:\n    return (1, 'a')",
			expectedName:       "get_pair",
			expectedReturnType: "tuple[int, str]",
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

			funcNode := findNodeByType(root, "function_definition")
			if funcNode == nil {
				t.Fatal("No function_definition node found")
			}

			node := parsePythonFunctionDefinition(funcNode, []byte(tt.code), graph, "test.py", nil)

			if node.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, node.Name)
			}
			if node.ReturnType != tt.expectedReturnType {
				t.Errorf("Expected return type %q, got %q", tt.expectedReturnType, node.ReturnType)
			}
		})
	}
}

// TestParsePythonFunctionDefinition_MethodArgumentsType tests that parameter type
// annotations are correctly extracted into MethodArgumentsType.
func TestParsePythonFunctionDefinition_MethodArgumentsType(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectedName      string
		expectedArgTypes  []string
		expectedArgValues []string
	}{
		{
			name:              "Typed parameters",
			code:              "def add(a: int, b: int) -> int:\n    return a + b",
			expectedName:      "add",
			expectedArgTypes:  []string{"a: int", "b: int"},
			expectedArgValues: []string{"a: int", "b: int"},
		},
		{
			name:              "Mixed typed and untyped",
			code:              "def greet(self, name: str) -> str:\n    pass",
			expectedName:      "greet",
			expectedArgTypes:  []string{"name: str"},
			expectedArgValues: []string{"self", "name: str"},
		},
		{
			name:              "No type annotations",
			code:              "def process(x, y):\n    return x + y",
			expectedName:      "process",
			expectedArgTypes:  nil,
			expectedArgValues: []string{"x", "y"},
		},
		{
			name:              "Complex types",
			code:              "def merge(items: list[str], count: int) -> dict[str, int]:\n    pass",
			expectedName:      "merge",
			expectedArgTypes:  []string{"items: list[str]", "count: int"},
			expectedArgValues: []string{"items: list[str]", "count: int"},
		},
		{
			name:              "Typed default parameter",
			code:              "def connect(host: str, port: int = 8080) -> None:\n    pass",
			expectedName:      "connect",
			expectedArgTypes:  []string{"host: str", "port: int"},
			expectedArgValues: []string{"host: str", "port: int = 8080"},
		},
		{
			name:              "No parameters",
			code:              "def noop() -> None:\n    pass",
			expectedName:      "noop",
			expectedArgTypes:  nil,
			expectedArgValues: []string{},
		},
		{
			name:              "Star args only - no types",
			code:              "def variadic(*args, **kwargs):\n    pass",
			expectedName:      "variadic",
			expectedArgTypes:  nil,
			expectedArgValues: []string{},
		},
		{
			name:              "Typed with star args untyped",
			code:              "def mixed(a: int, *args, **kwargs) -> None:\n    pass",
			expectedName:      "mixed",
			expectedArgTypes:  []string{"a: int"},
			expectedArgValues: []string{"a: int"},
		},
		{
			name:              "Only self - no types",
			code:              "def method(self):\n    pass",
			expectedName:      "method",
			expectedArgTypes:  nil,
			expectedArgValues: []string{"self"},
		},
		{
			name:              "Untyped default only",
			code:              "def greet(name, greeting='Hello'):\n    pass",
			expectedName:      "greet",
			expectedArgTypes:  nil,
			expectedArgValues: []string{"name", "greeting='Hello'"},
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

			funcNode := findNodeByType(root, "function_definition")
			if funcNode == nil {
				t.Fatal("No function_definition node found")
			}

			node := parsePythonFunctionDefinition(funcNode, []byte(tt.code), graph, "test.py", nil)

			if node.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, node.Name)
			}

			// Check MethodArgumentsType
			if tt.expectedArgTypes == nil {
				if len(node.MethodArgumentsType) != 0 {
					t.Errorf("Expected no argument types, got %v", node.MethodArgumentsType)
				}
			} else {
				if len(node.MethodArgumentsType) != len(tt.expectedArgTypes) {
					t.Errorf("Expected %d argument types, got %d: %v",
						len(tt.expectedArgTypes), len(node.MethodArgumentsType), node.MethodArgumentsType)
				} else {
					for i, expected := range tt.expectedArgTypes {
						if node.MethodArgumentsType[i] != expected {
							t.Errorf("Argument type [%d]: expected %q, got %q", i, expected, node.MethodArgumentsType[i])
						}
					}
				}
			}

			// Check MethodArgumentsValue unchanged
			if len(node.MethodArgumentsValue) != len(tt.expectedArgValues) {
				t.Errorf("Expected %d argument values, got %d: %v",
					len(tt.expectedArgValues), len(node.MethodArgumentsValue), node.MethodArgumentsValue)
			} else {
				for i, expected := range tt.expectedArgValues {
					if node.MethodArgumentsValue[i] != expected {
						t.Errorf("Argument value [%d]: expected %q, got %q", i, expected, node.MethodArgumentsValue[i])
					}
				}
			}
		})
	}
}

// TestParsePythonFunctionDefinition_ReturnTypeWithBuildGraph tests that return types
// are populated when parsing through buildGraphFromAST (end-to-end).
func TestParsePythonFunctionDefinition_ReturnTypeWithBuildGraph(t *testing.T) {
	code := `
def add_numbers(a: int, b: int) -> int:
    return a + b

def safe_divide(a: float, b: float) -> float | None:
    if b == 0:
        return None
    return a / b

def no_annotation(x):
    return x

class Calculator:
    def __init__(self, name: str) -> None:
        self.name = name

    def add(self, a: int, b: int) -> int:
        return a + b

    @property
    def display_name(self) -> str:
        return self.name

    def __str__(self) -> str:
        return f"Calculator({self.name})"
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
	buildGraphFromAST(tree.RootNode(), []byte(code), graph, nil, "test.py")

	// Verify return types for each function.
	expectations := map[string]struct {
		returnType string
		argTypes   []string
	}{
		"add_numbers":   {returnType: "int", argTypes: []string{"a: int", "b: int"}},
		"safe_divide":   {returnType: "float | None", argTypes: []string{"a: float", "b: float"}},
		"no_annotation": {returnType: "", argTypes: nil},
		"__init__":      {returnType: "None", argTypes: []string{"name: str"}},
		"add":           {returnType: "int", argTypes: []string{"a: int", "b: int"}},
		"display_name":  {returnType: "str", argTypes: nil},
		"__str__":       {returnType: "str", argTypes: nil},
	}

	for expectedName, expected := range expectations {
		var foundNode *Node
		for _, node := range graph.Nodes {
			if node.Name == expectedName {
				foundNode = node
				break
			}
		}

		if foundNode == nil {
			t.Errorf("Function %s not found in graph", expectedName)
			continue
		}

		if foundNode.ReturnType != expected.returnType {
			t.Errorf("Function %s: expected return type %q, got %q",
				expectedName, expected.returnType, foundNode.ReturnType)
		}

		if expected.argTypes == nil {
			if len(foundNode.MethodArgumentsType) != 0 {
				t.Errorf("Function %s: expected no argument types, got %v",
					expectedName, foundNode.MethodArgumentsType)
			}
		} else {
			if len(foundNode.MethodArgumentsType) != len(expected.argTypes) {
				t.Errorf("Function %s: expected %d argument types, got %d: %v",
					expectedName, len(expected.argTypes), len(foundNode.MethodArgumentsType), foundNode.MethodArgumentsType)
			}
		}
	}
}

// Helper function to find a node by name and type.
func findNodeByNameAndType(graph *CodeGraph, name string, nodeType string) *Node {
	for _, node := range graph.Nodes {
		if node.Name == name && node.Type == nodeType {
			return node
		}
	}
	return nil
}
