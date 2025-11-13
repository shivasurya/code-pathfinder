package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
)

// ExtractPythonImports extracts all import statements from a Python CodeGraph.
// This is a convenience wrapper around the existing ExtractImports function,
// made available for the Python adapter with a Python-specific name.
//
// Handles:
//   - import foo
//   - import foo.bar
//   - from foo import bar
//   - from foo import bar as baz
//   - from foo import *
//   - from . import module (relative imports)
//   - from .. import module (relative imports)
//
// Parameters:
//   - codeGraph: the parsed Python code graph containing import nodes
//   - filePath: absolute path to the Python file
//   - sourceCode: contents of the Python file as byte array
//   - registry: module registry for resolving relative imports
//
// Returns:
//   - ImportMap: map of local names to fully qualified module paths
//   - error: if parsing fails
func ExtractPythonImports(codeGraph *graph.CodeGraph, filePath string, sourceCode []byte, registry *ModuleRegistry) (*ImportMap, error) {
	// Delegate to the existing ExtractImports function
	// This avoids code duplication and maintains consistency with the legacy path
	return ExtractImports(filePath, sourceCode, registry)
}
