package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/extraction"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ExtractClassAttributes extracts class attributes from Python file.
// Deprecated: Use extraction.ExtractClassAttributes instead.
func ExtractClassAttributes(
	filePath string,
	sourceCode []byte,
	modulePath string,
	typeEngine *resolution.TypeInferenceEngine,
	attrRegistry *registry.AttributeRegistry,
) error {
	return extraction.ExtractClassAttributes(filePath, sourceCode, modulePath, typeEngine, attrRegistry)
}
