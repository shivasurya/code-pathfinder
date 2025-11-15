package callgraph

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	cgbuilder "github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	cgregistry "github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/resolution"
)

// ImportMapCache is a type alias for backward compatibility.
//
// Deprecated: Use builder.ImportMapCache instead.
// This type alias will be removed in a future version.
type ImportMapCache = cgbuilder.ImportMapCache

// NewImportMapCache creates a new empty import map cache.
//
// Deprecated: Use builder.NewImportMapCache instead.
func NewImportMapCache() *ImportMapCache {
	return cgbuilder.NewImportMapCache()
}

// BuildCallGraph constructs the complete call graph for a Python project.
//
// Deprecated: Use builder.BuildCallGraph instead.
func BuildCallGraph(codeGraph *graph.CodeGraph, registry *core.ModuleRegistry, projectRoot string) (*core.CallGraph, error) {
	return cgbuilder.BuildCallGraph(codeGraph, registry, projectRoot)
}

// resolveCallTarget is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.ResolveCallTarget instead.
func resolveCallTarget(target string, importMap *core.ImportMap, registry *core.ModuleRegistry, currentModule string, codeGraph *graph.CodeGraph, typeEngine *resolution.TypeInferenceEngine, callerFQN string, _ *core.CallGraph) (string, bool, *core.TypeInfo) {
	return cgbuilder.ResolveCallTarget(target, importMap, registry, currentModule, codeGraph, typeEngine, callerFQN, nil)
}

// findFunctionAtLine is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.FindFunctionAtLine instead.
func findFunctionAtLine(root *sitter.Node, lineNumber uint32) *sitter.Node {
	return cgbuilder.FindFunctionAtLine(root, lineNumber)
}

// generateTaintSummaries is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.GenerateTaintSummaries instead.
func generateTaintSummaries(callGraph *core.CallGraph, codeGraph *graph.CodeGraph, registry *core.ModuleRegistry) {
	cgbuilder.GenerateTaintSummaries(callGraph, codeGraph, registry)
}

// Note: detectPythonVersion is defined in python_version_detector.go and delegates to builder package.

// validateStdlibFQN is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.ValidateStdlibFQN instead.
func validateStdlibFQN(fqn string, remoteLoader *cgregistry.StdlibRegistryRemote) bool {
	return cgbuilder.ValidateStdlibFQN(fqn, remoteLoader)
}

// validateFQN is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.ValidateFQN instead.
func validateFQN(fqn string, registry *core.ModuleRegistry) bool {
	return cgbuilder.ValidateFQN(fqn, registry)
}

// indexFunctions is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.IndexFunctions instead.
func indexFunctions(codeGraph *graph.CodeGraph, callGraph *core.CallGraph, registry *core.ModuleRegistry) {
	cgbuilder.IndexFunctions(codeGraph, callGraph, registry)
}

// getFunctionsInFile is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.GetFunctionsInFile instead.
func getFunctionsInFile(codeGraph *graph.CodeGraph, filePath string) []*graph.Node {
	return cgbuilder.GetFunctionsInFile(codeGraph, filePath)
}

// findContainingFunction is a wrapper for backward compatibility with tests.
//
// Deprecated: Use builder.FindContainingFunction instead.
func findContainingFunction(location core.Location, functions []*graph.Node, modulePath string) string {
	return cgbuilder.FindContainingFunction(location, functions, modulePath)
}
