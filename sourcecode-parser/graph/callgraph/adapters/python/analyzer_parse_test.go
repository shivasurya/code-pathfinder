package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

// TestPythonAnalyzer_Name verifies language identifier.
func TestPythonAnalyzer_Name(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	assert.Equal(t, "python", analyzer.Name())
}

// TestPythonAnalyzer_FileExtensions verifies supported extensions.
func TestPythonAnalyzer_FileExtensions(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	exts := analyzer.FileExtensions()

	assert.Contains(t, exts, ".py")
	assert.Contains(t, exts, ".pyw")
	assert.Contains(t, exts, ".pyi")
	assert.Equal(t, 3, len(exts))
}

// TestPythonAnalyzer_Parse_Simple verifies basic parsing.
func TestPythonAnalyzer_Parse_Simple(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`import os

def hello():
    print("world")
`)

	module, err := analyzer.Parse("test.py", source)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, "test.py", module.FilePath)
	assert.Equal(t, "python", module.Language)
	assert.NotNil(t, module.AST)
	assert.Equal(t, true, module.Metadata["tree-sitter"])
	assert.Equal(t, "3.x", module.Metadata["version"])
}

// TestPythonAnalyzer_Parse_ComplexFile verifies parsing complex Python code.
func TestPythonAnalyzer_Parse_ComplexFile(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`import os
from pathlib import Path

class UserManager:
    def __init__(self):
        self.users = []

    def add_user(self, name: str) -> bool:
        self.users.append(name)
        return True

async def fetch_data():
    pass

@decorator
def decorated_function():
    pass
`)

	module, err := analyzer.Parse("complex.py", source)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, "complex.py", module.FilePath)
	assert.Equal(t, "python", module.Language)
}

// TestPythonAnalyzer_Parse_EmptyFile verifies parsing empty file.
func TestPythonAnalyzer_Parse_EmptyFile(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte("")

	module, err := analyzer.Parse("empty.py", source)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, "empty.py", module.FilePath)
}

// TestPythonAnalyzer_Parse_SyntaxError verifies error handling.
// Note: tree-sitter is resilient and may parse invalid syntax, creating error nodes.
func TestPythonAnalyzer_Parse_InvalidSyntax(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`def broken(::
    pass
`)

	// tree-sitter is resilient and creates partial trees even with syntax errors
	module, err := analyzer.Parse("broken.py", source)

	// ParseSingleFile should still succeed (tree-sitter creates error nodes)
	assert.NoError(t, err)
	assert.NotNil(t, module)
}

// TestPythonAnalyzer_ExtractImports_Simple verifies import extraction.
func TestPythonAnalyzer_ExtractImports_Simple(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`import os
import sys
from pathlib import Path
`)

	module, err := analyzer.Parse("test.py", source)
	assert.NoError(t, err)

	imports, err := analyzer.ExtractImports(module)

	assert.NoError(t, err)
	assert.NotNil(t, imports)

	// Check that imports were extracted
	assert.Contains(t, imports.Imports, "os")
	assert.Contains(t, imports.Imports, "sys")
	assert.Contains(t, imports.Imports, "Path")
}

// TestPythonAnalyzer_ExtractImports_Aliases verifies alias handling.
func TestPythonAnalyzer_ExtractImports_Aliases(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`import os as operating_system
from sys import argv as arguments
`)

	module, _ := analyzer.Parse("test.py", source)
	imports, err := analyzer.ExtractImports(module)

	assert.NoError(t, err)
	assert.NotNil(t, imports)

	// Check aliases were captured
	assert.Contains(t, imports.Imports, "operating_system")
	assert.Contains(t, imports.Imports, "arguments")
}

// TestPythonAnalyzer_ExtractImports_MultipleImports verifies multiple import handling.
func TestPythonAnalyzer_ExtractImports_MultipleImports(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`from json import dumps, loads, JSONDecoder
`)

	module, _ := analyzer.Parse("test.py", source)
	imports, err := analyzer.ExtractImports(module)

	assert.NoError(t, err)
	assert.NotNil(t, imports)

	// Check all imports extracted
	assert.Contains(t, imports.Imports, "dumps")
	assert.Contains(t, imports.Imports, "loads")
	assert.Contains(t, imports.Imports, "JSONDecoder")
}

// TestPythonAnalyzer_ExtractImports_NestedModule verifies nested module imports.
func TestPythonAnalyzer_ExtractImports_NestedModule(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`from foo.bar.baz import qux
`)

	module, _ := analyzer.Parse("test.py", source)
	imports, err := analyzer.ExtractImports(module)

	assert.NoError(t, err)
	assert.NotNil(t, imports)
	assert.Contains(t, imports.Imports, "qux")
}

// TestPythonAnalyzer_ExtractImports_EmptyFile verifies empty file handling.
func TestPythonAnalyzer_ExtractImports_EmptyFile(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte("")

	module, _ := analyzer.Parse("test.py", source)
	imports, err := analyzer.ExtractImports(module)

	assert.NoError(t, err)
	assert.NotNil(t, imports)
	assert.Equal(t, 0, len(imports.Imports))
}

// TestPythonAnalyzer_SupportsFramework verifies framework detection.
func TestPythonAnalyzer_SupportsFramework(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	// Test known frameworks
	assert.True(t, analyzer.SupportsFramework("flask"))
	assert.True(t, analyzer.SupportsFramework("django"))
	assert.True(t, analyzer.SupportsFramework("fastapi"))
	assert.True(t, analyzer.SupportsFramework("pyramid"))
	assert.True(t, analyzer.SupportsFramework("tornado"))
	assert.True(t, analyzer.SupportsFramework("bottle"))
	assert.True(t, analyzer.SupportsFramework("cherrypy"))
	assert.True(t, analyzer.SupportsFramework("aiohttp"))
	assert.True(t, analyzer.SupportsFramework("sanic"))
	assert.True(t, analyzer.SupportsFramework("starlette"))

	// Test case insensitivity
	assert.True(t, analyzer.SupportsFramework("Flask"))
	assert.True(t, analyzer.SupportsFramework("DJANGO"))

	// Test unknown framework
	assert.False(t, analyzer.SupportsFramework("unknown"))
	assert.False(t, analyzer.SupportsFramework("express"))
}

// TestPythonAnalyzer_StubMethods verifies unimplemented methods return errors.
func TestPythonAnalyzer_StubMethods(t *testing.T) {
	analyzer := NewPythonAnalyzer()

	// Test PR-04 stub methods (not yet implemented)
	_, err := analyzer.ExtractCallSites(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Contains(t, err.Error(), "PR-04")

	_, err = analyzer.ExtractStatements(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Contains(t, err.Error(), "PR-04")

	_, err = analyzer.ExtractVariables(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Contains(t, err.Error(), "PR-04")

	_, err = analyzer.AnalyzeTaint(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Contains(t, err.Error(), "PR-04")
}

// TestPythonAnalyzer_InterfaceCompliance verifies analyzer implements LanguageAnalyzer.
func TestPythonAnalyzer_InterfaceCompliance(t *testing.T) {
	var _ callgraph.LanguageAnalyzer = (*PythonAnalyzer)(nil)
	// If this compiles, the interface is implemented correctly
}

// TestPythonAnalyzer_Parse_TypedCode verifies parsing type-annotated Python code.
func TestPythonAnalyzer_Parse_TypedCode(t *testing.T) {
	analyzer := NewPythonAnalyzer()
	source := []byte(`from typing import List, Dict, Optional

def process_data(items: List[str], config: Dict[str, int]) -> Optional[str]:
    if not items:
        return None
    return items[0]
`)

	module, err := analyzer.Parse("typed.py", source)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.Equal(t, "typed.py", module.FilePath)
}
