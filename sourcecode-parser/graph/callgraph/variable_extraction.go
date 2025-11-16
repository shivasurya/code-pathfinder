package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/extraction"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ExtractVariableAssignments extracts variable assignments from a Python file.
// Deprecated: Use extraction.ExtractVariableAssignments instead.
func ExtractVariableAssignments(
	filePath string,
	sourceCode []byte,
	typeEngine *resolution.TypeInferenceEngine,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
) error {
	return extraction.ExtractVariableAssignments(filePath, sourceCode, typeEngine, registry, builtinRegistry)
}
