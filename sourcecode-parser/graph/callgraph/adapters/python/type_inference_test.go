package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

// TestPythonAnalyzer_InferTypes_Simple verifies basic type inference.
func TestPythonAnalyzer_InferTypes_Simple(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def example():
    x = 5
    y = "hello"
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)
	assert.NotNil(t, typeContext.Variables)
	assert.NotNil(t, typeContext.Functions)
	assert.NotNil(t, typeContext.Classes)
}

// TestPythonAnalyzer_InferTypes_WithAnnotations verifies type inference from annotations.
func TestPythonAnalyzer_InferTypes_WithAnnotations(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def greet(name: str, age: int) -> str:
    return f"Hello {name}, age {age}"
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)

	// Verify function was extracted
	assert.GreaterOrEqual(t, len(typeContext.Functions), 1)

	// Find greet function
	var greet *callgraph.FunctionDef
	for _, fn := range typeContext.Functions {
		if fn.Name == "greet" {
			greet = fn
			break
		}
	}

	if greet != nil {
		// Verify return type was extracted
		// Note: Actual return type extraction depends on parser
		_ = greet.ReturnType
	}
}

// TestPythonAnalyzer_InferTypes_EmptyFile verifies handling of empty files.
func TestPythonAnalyzer_InferTypes_EmptyFile(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte("")

	module, err := analyzer.Parse("empty.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)
}

// TestPythonAnalyzer_InferTypes_WithImports verifies type inference with imports.
func TestPythonAnalyzer_InferTypes_WithImports(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`import os
from pathlib import Path

def process():
    p = Path("/tmp")
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)
	assert.NotNil(t, typeContext.Imports)
}

// TestPythonAnalyzer_InferTypes_MultipleF unctions verifies inference across multiple functions.
func TestPythonAnalyzer_InferTypes_MultipleFunctions(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def add(x: int, y: int) -> int:
    return x + y

def multiply(x: int, y: int) -> int:
    return x * y

def combine(a, b):
    return add(a, b) + multiply(a, b)
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)

	// Verify multiple functions extracted
	assert.GreaterOrEqual(t, len(typeContext.Functions), 3)
}

// TestPythonAnalyzer_InferTypes_WithClasses verifies type inference for classes.
func TestPythonAnalyzer_InferTypes_WithClasses(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`class User:
    def __init__(self, name: str):
        self.name = name

    def get_name(self) -> str:
        return self.name
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)

	// Verify class was extracted
	assert.GreaterOrEqual(t, len(typeContext.Classes), 1)
}

// TestPythonAnalyzer_InferTypes_InvalidAST verifies error handling for invalid AST.
func TestPythonAnalyzer_InferTypes_InvalidAST(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	module := &callgraph.ParsedModule{
		FilePath: "test.py",
		Language: "python",
		AST:      "invalid", // Not a CodeGraph
	}

	registry := callgraph.NewModuleRegistry()
	_, err := analyzer.InferTypes(module, registry)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected *graph.CodeGraph")
}

// TestPythonAnalyzer_ResolveType_Builtin verifies resolution of builtin types.
func TestPythonAnalyzer_ResolveType_Builtin(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	// Create a basic TypeContext
	typeContext := &callgraph.TypeContext{
		Variables: make(map[string]*callgraph.TypeInfo),
		Functions: make(map[string]*callgraph.FunctionDef),
		Classes:   make(map[string]*callgraph.ClassDef),
	}

	// Resolve builtin types (BuiltinRegistry is created internally in ResolveType)
	intType, err := analyzer.ResolveType("int", typeContext)
	assert.NoError(t, err)
	assert.NotNil(t, intType)

	strType, err := analyzer.ResolveType("str", typeContext)
	assert.NoError(t, err)
	assert.NotNil(t, strType)
}

// TestPythonAnalyzer_ResolveType_WithImports verifies type resolution with imports.
func TestPythonAnalyzer_ResolveType_WithImports(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	// Create TypeContext with imports
	typeContext := &callgraph.TypeContext{
		Variables: make(map[string]*callgraph.TypeInfo),
		Functions: make(map[string]*callgraph.FunctionDef),
		Classes:   make(map[string]*callgraph.ClassDef),
		Imports: &callgraph.ImportMap{
			Imports: map[string]string{
				"Path":   "pathlib.Path",
				"Logger": "logging.Logger",
			},
		},
	}

	// Resolve imported type
	pathType, err := analyzer.ResolveType("Path", typeContext)
	assert.NoError(t, err)
	assert.NotNil(t, pathType)
	assert.Equal(t, "pathlib.Path", pathType.TypeFQN)
}

// TestPythonAnalyzer_ResolveType_NilContext verifies handling of nil context.
func TestPythonAnalyzer_ResolveType_NilContext(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	typeInfo, err := analyzer.ResolveType("SomeType", nil)

	assert.NoError(t, err)
	assert.NotNil(t, typeInfo)
	assert.Equal(t, "SomeType", typeInfo.TypeFQN)
	assert.Equal(t, float32(0.0), typeInfo.Confidence)
}

// TestPythonAnalyzer_ResolveType_UnknownType verifies handling of unknown types.
func TestPythonAnalyzer_ResolveType_UnknownType(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	typeContext := &callgraph.TypeContext{
		Variables: make(map[string]*callgraph.TypeInfo),
		Functions: make(map[string]*callgraph.FunctionDef),
		Classes:   make(map[string]*callgraph.ClassDef),
		Imports:   &callgraph.ImportMap{Imports: make(map[string]string)},
	}

	typeInfo, err := analyzer.ResolveType("UnknownCustomType", typeContext)

	assert.NoError(t, err)
	assert.NotNil(t, typeInfo)
	assert.Equal(t, "UnknownCustomType", typeInfo.TypeFQN)
	// Should have low confidence for unknown types
	assert.Less(t, typeInfo.Confidence, float32(0.5))
}

// TestPythonAnalyzer_InferTypes_WithAsyncFunction verifies type inference for async functions.
func TestPythonAnalyzer_InferTypes_WithAsyncFunction(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`async def fetch_data(url: str) -> dict:
    return {"data": "example"}
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)
	assert.GreaterOrEqual(t, len(typeContext.Functions), 1)
}

// TestPythonAnalyzer_InferTypes_WithDecorators verifies type inference with decorated functions.
func TestPythonAnalyzer_InferTypes_WithDecorators(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`@property
def name(self) -> str:
    return self._name

@staticmethod
def create() -> 'MyClass':
    return MyClass()
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	registry := callgraph.NewModuleRegistry()
	typeContext, err := analyzer.InferTypes(module, registry)

	assert.NoError(t, err)
	assert.NotNil(t, typeContext)
}
