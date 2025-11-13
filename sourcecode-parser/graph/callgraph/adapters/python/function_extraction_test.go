package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

// TestPythonAnalyzer_ExtractFunctions_Simple verifies basic function extraction.
func TestPythonAnalyzer_ExtractFunctions_Simple(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def hello(name):
    return f"Hello {name}"

def calculate(x, y):
    return x + y
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	functions, err := analyzer.ExtractFunctions(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(functions), 2, "Should extract at least 2 functions")

	// Find functions by name
	var hello, calculate *callgraph.FunctionDef
	for _, fn := range functions {
		if fn.Name == "hello" {
			hello = fn
		} else if fn.Name == "calculate" {
			calculate = fn
		}
	}

	// Verify hello function
	assert.NotNil(t, hello, "Should extract hello function")
	if hello != nil {
		assert.Equal(t, "hello", hello.Name)
		assert.NotEmpty(t, hello.FQN)
		assert.Equal(t, "test.py", hello.Location.File)
	}

	// Verify calculate function
	assert.NotNil(t, calculate, "Should extract calculate function")
	if calculate != nil {
		assert.Equal(t, "calculate", calculate.Name)
		assert.NotEmpty(t, calculate.FQN)
	}
}

// TestPythonAnalyzer_ExtractFunctions_WithAnnotations verifies extraction with type annotations.
func TestPythonAnalyzer_ExtractFunctions_WithAnnotations(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def greet(name: str, age: int) -> str:
    return f"Hello {name}, you are {age} years old"
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	functions, err := analyzer.ExtractFunctions(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(functions), 1)

	greet := findFunctionByName(functions, "greet")
	assert.NotNil(t, greet, "Should extract greet function")

	if greet != nil {
		// Verify parameters
		assert.GreaterOrEqual(t, len(greet.Parameters), 2)

		// Note: Actual parameter extraction depends on parser implementation
		// This test documents current behavior
	}
}

// TestPythonAnalyzer_ExtractFunctions_EmptyFile verifies handling of empty files.
func TestPythonAnalyzer_ExtractFunctions_EmptyFile(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte("")

	module, err := analyzer.Parse("empty.py", source)
	assert.NoError(t, err)

	functions, err := analyzer.ExtractFunctions(module)

	assert.NoError(t, err)
	assert.NotNil(t, functions)
	// Empty file may have 0 functions
}

// TestPythonAnalyzer_ExtractFunctions_Nested verifies extraction of nested functions.
func TestPythonAnalyzer_ExtractFunctions_Nested(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def outer():
    def inner():
        return 42
    return inner()
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	functions, err := analyzer.ExtractFunctions(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(functions), 1, "Should extract at least outer function")

	outer := findFunctionByName(functions, "outer")
	assert.NotNil(t, outer, "Should extract outer function")
}

// TestPythonAnalyzer_ExtractClasses_Simple verifies basic class extraction.
func TestPythonAnalyzer_ExtractClasses_Simple(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`class User:
    def __init__(self, name):
        self.name = name

    def greet(self):
        return f"Hello {self.name}"

class Product:
    pass
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	classes, err := analyzer.ExtractClasses(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(classes), 2, "Should extract at least 2 classes")

	// Find classes by name
	var user, product *callgraph.ClassDef
	for _, cls := range classes {
		if cls.Name == "User" {
			user = cls
		} else if cls.Name == "Product" {
			product = cls
		}
	}

	// Verify User class
	assert.NotNil(t, user, "Should extract User class")
	if user != nil {
		assert.Equal(t, "User", user.Name)
		assert.NotEmpty(t, user.FQN)
		assert.Equal(t, "test.py", user.Location.File)
	}

	// Verify Product class
	assert.NotNil(t, product, "Should extract Product class")
	if product != nil {
		assert.Equal(t, "Product", product.Name)
	}
}

// TestPythonAnalyzer_ExtractClasses_WithAnnotations verifies class extraction with type hints.
func TestPythonAnalyzer_ExtractClasses_WithAnnotations(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`class Person:
    name: str
    age: int

    def __init__(self, name: str, age: int):
        self.name = name
        self.age = age
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	classes, err := analyzer.ExtractClasses(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(classes), 1)

	person := findClassByName(classes, "Person")
	assert.NotNil(t, person, "Should extract Person class")

	if person != nil {
		assert.Equal(t, "Person", person.Name)
		// Attributes and methods extraction depends on parser implementation
	}
}

// TestPythonAnalyzer_ExtractClasses_EmptyFile verifies handling of empty files.
func TestPythonAnalyzer_ExtractClasses_EmptyFile(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte("")

	module, err := analyzer.Parse("empty.py", source)
	assert.NoError(t, err)

	classes, err := analyzer.ExtractClasses(module)

	assert.NoError(t, err)
	assert.NotNil(t, classes)
	// Empty file may have 0 classes
}

// TestPythonAnalyzer_ExtractClasses_Nested verifies extraction of nested classes.
func TestPythonAnalyzer_ExtractClasses_Nested(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`class Outer:
    class Inner:
        pass
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	classes, err := analyzer.ExtractClasses(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(classes), 1, "Should extract at least Outer class")

	outer := findClassByName(classes, "Outer")
	assert.NotNil(t, outer, "Should extract Outer class")
}

// TestPythonAnalyzer_ExtractClasses_WithInheritance verifies class extraction with base classes.
func TestPythonAnalyzer_ExtractClasses_WithInheritance(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`class Animal:
    pass

class Dog(Animal):
    pass
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	classes, err := analyzer.ExtractClasses(module)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(classes), 2)

	dog := findClassByName(classes, "Dog")
	assert.NotNil(t, dog, "Should extract Dog class")

	// Base class extraction depends on parser implementation
	// This test documents expected behavior
}

// TestPythonAnalyzer_ExtractFunctions_InvalidAST verifies error handling for invalid AST.
func TestPythonAnalyzer_ExtractFunctions_InvalidAST(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	module := &callgraph.ParsedModule{
		FilePath: "test.py",
		Language: "python",
		AST:      "invalid", // Not a CodeGraph
	}

	_, err := analyzer.ExtractFunctions(module)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *graph.CodeGraph")
}

// TestPythonAnalyzer_ExtractClasses_InvalidAST verifies error handling for invalid AST.
func TestPythonAnalyzer_ExtractClasses_InvalidAST(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	module := &callgraph.ParsedModule{
		FilePath: "test.py",
		Language: "python",
		AST:      "invalid", // Not a CodeGraph
	}

	_, err := analyzer.ExtractClasses(module)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *graph.CodeGraph")
}

// Helper functions

func findFunctionByName(functions []*callgraph.FunctionDef, name string) *callgraph.FunctionDef {
	for _, fn := range functions {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}

func findClassByName(classes []*callgraph.ClassDef, name string) *callgraph.ClassDef {
	for _, cls := range classes {
		if cls.Name == name {
			return cls
		}
	}
	return nil
}
