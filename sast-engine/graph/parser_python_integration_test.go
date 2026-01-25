package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPythonSymbolTypes_Integration is an integration test that parses a comprehensive
// Python file and verifies all 12 symbol types are correctly detected.
func TestPythonSymbolTypes_Integration(t *testing.T) {
	// Get the fixture file path
	fixturePath, err := filepath.Abs("testdata/python_symbols")
	assert.NoError(t, err, "Should resolve fixture path")

	// Parse the fixture project
	codeGraph := Initialize(fixturePath, nil)

	// Count symbols by type
	typeCounts := make(map[string]int)
	typeNames := make(map[string][]string)

	for _, node := range codeGraph.Nodes {
		typeCounts[node.Type]++
		typeNames[node.Type] = append(typeNames[node.Type], node.Name)
	}

	// Define expected symbol types and minimum counts
	expectedTypes := map[string]struct {
		minCount    int
		description string
	}{
		// Function types
		"function_definition": {2, "Module-level functions"},
		"method":              {3, "Instance methods inside classes"},
		"constructor":         {1, "__init__ methods"},  // Only ComprehensiveClass and Rectangle have __init__
		"property":            {2, "@property decorated methods"},
		"special_method":      {8, "Magic methods (__str__, __add__, etc.)"},

		// Class types
		"class_definition": {1, "Regular classes"},
		"interface":        {2, "Protocol/ABC classes"},
		"enum":             {3, "Enum/IntEnum/Flag classes"},
		"dataclass":        {2, "@dataclass decorated classes"},

		// Variable types
		"module_variable": {2, "Module-level lowercase variables"},
		"constant":        {4, "UPPERCASE constants (module + class level)"},
		"class_field":     {1, "Class-level lowercase attributes"},
	}

	// Verify each expected type
	t.Run("All symbol types detected", func(t *testing.T) {
		for symbolType, expected := range expectedTypes {
			count := typeCounts[symbolType]
			assert.GreaterOrEqual(t, count, expected.minCount,
				"Expected at least %d %s (%s), got %d. Found: %v",
				expected.minCount, symbolType, expected.description, count, typeNames[symbolType])
		}
	})

	// Verify specific symbol names to ensure correct classification
	t.Run("Specific symbols correctly classified", func(t *testing.T) {
		// Check constants
		assert.Contains(t, typeNames["constant"], "MAX_CONNECTIONS", "MAX_CONNECTIONS should be a constant")
		assert.Contains(t, typeNames["constant"], "API_KEY", "API_KEY should be a constant")
		assert.Contains(t, typeNames["constant"], "DEFAULT_TIMEOUT", "DEFAULT_TIMEOUT should be a constant")

		// Check module variables
		assert.Contains(t, typeNames["module_variable"], "version", "version should be a module_variable")
		assert.Contains(t, typeNames["module_variable"], "debug_mode", "debug_mode should be a module_variable")

		// Check functions
		assert.Contains(t, typeNames["function_definition"], "module_level_function", "module_level_function should be function_definition")
		assert.Contains(t, typeNames["function_definition"], "another_function", "another_function should be function_definition")

		// Check interfaces
		assert.Contains(t, typeNames["interface"], "Drawable", "Drawable should be an interface (Protocol)")
		assert.Contains(t, typeNames["interface"], "Storage", "Storage should be an interface (ABC)")

		// Check enums
		assert.Contains(t, typeNames["enum"], "Color", "Color should be an enum")
		assert.Contains(t, typeNames["enum"], "Priority", "Priority should be an enum")
		assert.Contains(t, typeNames["enum"], "Flags", "Flags should be an enum")

		// Check dataclasses
		assert.Contains(t, typeNames["dataclass"], "Point", "Point should be a dataclass")
		assert.Contains(t, typeNames["dataclass"], "Rectangle", "Rectangle should be a dataclass")

		// Check constructors
		assert.Contains(t, typeNames["constructor"], "__init__", "All classes should have __init__ as constructor")

		// Check properties
		assert.Contains(t, typeNames["property"], "name_property", "name_property should be a property")
		assert.Contains(t, typeNames["property"], "value_property", "value_property should be a property")

		// Check methods (NOT constructors, properties, or special methods)
		assert.Contains(t, typeNames["method"], "regular_method", "regular_method should be a method")
		assert.Contains(t, typeNames["method"], "another_method", "another_method should be a method")
		assert.Contains(t, typeNames["method"], "area", "area (in dataclass) should be a method")

		// Check special methods
		assert.Contains(t, typeNames["special_method"], "__str__", "__str__ should be a special_method")
		assert.Contains(t, typeNames["special_method"], "__add__", "__add__ should be a special_method")
		assert.Contains(t, typeNames["special_method"], "__len__", "__len__ should be a special_method")
		assert.Contains(t, typeNames["special_method"], "__eq__", "__eq__ should be a special_method")
		assert.Contains(t, typeNames["special_method"], "__call__", "__call__ should be a special_method")
		assert.Contains(t, typeNames["special_method"], "__repr__", "__repr__ should be a special_method")

		// Check class fields vs constants
		assert.Contains(t, typeNames["class_field"], "class_variable", "class_variable should be a class_field")
		assert.Contains(t, typeNames["constant"], "MAX_SIZE", "MAX_SIZE should be a constant (class-level UPPERCASE)")
	})

	// Verify method vs function_definition distinction
	t.Run("Method vs function_definition distinction", func(t *testing.T) {
		// Module-level functions should be function_definition
		for _, name := range []string{"module_level_function", "another_function"} {
			assert.Contains(t, typeNames["function_definition"], name,
				"%s is module-level, should be function_definition not method", name)
			assert.NotContains(t, typeNames["method"], name,
				"%s should not be classified as method", name)
		}

		// Functions inside classes should be method (unless constructor, property, or special)
		for _, name := range []string{"regular_method", "another_method", "area"} {
			assert.Contains(t, typeNames["method"], name,
				"%s is inside a class, should be method not function_definition", name)
			assert.NotContains(t, typeNames["function_definition"], name,
				"%s should not be classified as function_definition", name)
		}

		// Constructor should NOT be method
		assert.NotContains(t, typeNames["method"], "__init__",
			"__init__ should be constructor, not method")

		// Special methods should NOT be method
		for _, name := range []string{"__str__", "__add__", "__len__"} {
			assert.NotContains(t, typeNames["method"], name,
				"%s should be special_method, not method", name)
		}

		// Properties should NOT be method
		for _, name := range []string{"name_property", "value_property"} {
			assert.NotContains(t, typeNames["method"], name,
				"%s should be property, not method", name)
		}
	})

	// Print summary for debugging
	t.Logf("\n=== Integration Test Summary ===")
	t.Logf("Total nodes parsed: %d", len(codeGraph.Nodes))
	t.Logf("\nSymbol type counts:")
	for symbolType, expected := range expectedTypes {
		count := typeCounts[symbolType]
		status := "✅"
		if count < expected.minCount {
			status = "❌"
		}
		t.Logf("  %s %-20s: %d (expected >= %d) - %s",
			status, symbolType, count, expected.minCount, expected.description)
	}
}

// TestMethodDistinction_Context tests that methods are correctly distinguished
// from functions based on their parent context (class vs module).
func TestMethodDistinction_Context(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedType   string
		expectedName   string
		shouldBeMethod bool
	}{
		{
			name: "Module-level function",
			code: `
def my_function():
    pass
`,
			expectedType:   "function_definition",
			expectedName:   "my_function",
			shouldBeMethod: false,
		},
		{
			name: "Method in class",
			code: `
class MyClass:
    def my_method(self):
        pass
`,
			expectedType:   "method",
			expectedName:   "my_method",
			shouldBeMethod: true,
		},
		{
			name: "Method in interface",
			code: `
from typing import Protocol

class MyInterface(Protocol):
    def my_method(self):
        pass
`,
			expectedType:   "method",
			expectedName:   "my_method",
			shouldBeMethod: true,
		},
		{
			name: "Method in enum",
			code: `
from enum import Enum

class MyEnum(Enum):
    VALUE = 1

    def my_method(self):
        pass
`,
			expectedType:   "method",
			expectedName:   "my_method",
			shouldBeMethod: true,
		},
		{
			name: "Method in dataclass",
			code: `
from dataclasses import dataclass

@dataclass
class MyDataclass:
    x: int

    def my_method(self):
        pass
`,
			expectedType:   "method",
			expectedName:   "my_method",
			shouldBeMethod: true,
		},
		{
			name: "Constructor not method",
			code: `
class MyClass:
    def __init__(self):
        pass
`,
			expectedType:   "constructor",
			expectedName:   "__init__",
			shouldBeMethod: false,
		},
		{
			name: "Property not method",
			code: `
class MyClass:
    @property
    def my_prop(self):
        pass
`,
			expectedType:   "property",
			expectedName:   "my_prop",
			shouldBeMethod: false,
		},
		{
			name: "Special method not method",
			code: `
class MyClass:
    def __str__(self):
        pass
`,
			expectedType:   "special_method",
			expectedName:   "__str__",
			shouldBeMethod: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.py")

			// Write test code to file
			err := os.WriteFile(testFile, []byte(tt.code), 0644)
			assert.NoError(t, err)

			// Parse the file
			codeGraph := Initialize(tmpDir, nil)

			// Find the node with the expected name
			var foundNode *Node
			for _, node := range codeGraph.Nodes {
				if node.Name == tt.expectedName {
					foundNode = node
					break
				}
			}

			assert.NotNil(t, foundNode, "Should find node with name %s", tt.expectedName)
			assert.Equal(t, tt.expectedType, foundNode.Type,
				"Node %s should have type %s, got %s",
				tt.expectedName, tt.expectedType, foundNode.Type)

			// Verify it's not incorrectly classified as method
			if !tt.shouldBeMethod {
				assert.NotEqual(t, "method", foundNode.Type,
					"Node %s should NOT be classified as method", tt.expectedName)
			}
		})
	}
}
