package python

import (
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

func init() {
	// Register Python analyzer with the global registry
	// This must happen after builder.go init() creates the registry
	if callgraph.GetGlobalLanguageRegistry() != nil {
		callgraph.GetGlobalLanguageRegistry().Register(NewPythonAnalyzer())
	}
}

// PythonAnalyzer implements the LanguageAnalyzer interface for Python.
// It delegates to existing Python parsing and extraction logic while
// providing a clean interface for the multi-language architecture.
type PythonAnalyzer struct{}

// NewPythonAnalyzer creates a new Python language analyzer.
func NewPythonAnalyzer() *PythonAnalyzer {
	return &PythonAnalyzer{}
}

// Name returns the language identifier.
func (p *PythonAnalyzer) Name() string {
	return "python"
}

// FileExtensions returns supported file extensions.
func (p *PythonAnalyzer) FileExtensions() []string {
	return []string{".py", ".pyw", ".pyi"} // .py, .pyw (Windows), .pyi (stubs)
}

// Parse converts Python source code to an AST using tree-sitter.
// It wraps the existing graph.ParseSingleFile() logic for single-file parsing.
func (p *PythonAnalyzer) Parse(filePath string, source []byte) (*callgraph.ParsedModule, error) {
	// Delegate to existing parsing infrastructure
	codeGraph, err := graph.ParseSingleFile(filePath, source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python file: %w", err)
	}

	// Wrap in language-agnostic ParsedModule
	// Store the source code in metadata so ExtractImports can use it without re-reading
	return &callgraph.ParsedModule{
		FilePath: filePath,
		Language: "python",
		AST:      codeGraph, // Store CodeGraph in AST field
		Metadata: map[string]interface{}{
			"tree-sitter": true,
			"version":     "3.x", // Can detect version from AST if needed
			"source":      source, // Store source for ExtractImports
		},
	}, nil
}

// ExtractImports extracts all import statements from the module.
// It delegates to the extracted ExtractPythonImports function in python_imports.go.
func (p *PythonAnalyzer) ExtractImports(module *callgraph.ParsedModule) (*callgraph.ImportMap, error) {
	// Extract CodeGraph from ParsedModule.AST
	codeGraph, ok := module.AST.(*graph.CodeGraph)
	if !ok {
		return nil, fmt.Errorf("expected *graph.CodeGraph, got %T", module.AST)
	}

	// Get source from metadata (stored during Parse)
	var source []byte
	if sourceData, ok := module.Metadata["source"]; ok {
		source, ok = sourceData.([]byte)
		if !ok {
			return nil, fmt.Errorf("source in metadata is not []byte")
		}
	} else {
		// Fallback: try to read from file if source not in metadata
		var err error
		source, err = graph.ReadFileContent(module.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read source file: %w", err)
		}
	}

	// Delegate to extracted import logic
	// Note: We create a temporary registry here. In the full implementation (PR-05),
	// this will be passed from the builder's registry
	registry := callgraph.NewModuleRegistry()
	return callgraph.ExtractPythonImports(codeGraph, module.FilePath, source, registry)
}

// ExtractFunctions extracts all function definitions from the module.
// Not implemented in Phase 1 - will be added in PR-03.
func (p *PythonAnalyzer) ExtractFunctions(module *callgraph.ParsedModule) ([]*callgraph.FunctionDef, error) {
	return nil, fmt.Errorf("not implemented (PR-03)")
}

// ExtractClasses extracts all class definitions from the module.
// Not implemented in Phase 1 - will be added in PR-03.
func (p *PythonAnalyzer) ExtractClasses(module *callgraph.ParsedModule) ([]*callgraph.ClassDef, error) {
	return nil, fmt.Errorf("not implemented (PR-03)")
}

// InferTypes performs type inference using Python-specific rules.
// Not implemented in Phase 1 - will be added in PR-03.
func (p *PythonAnalyzer) InferTypes(module *callgraph.ParsedModule, registry *callgraph.ModuleRegistry) (*callgraph.TypeContext, error) {
	return nil, fmt.Errorf("not implemented (PR-03)")
}

// ExtractCallSites returns all function calls within a function.
// Not implemented in Phase 1 - will be added in PR-04.
func (p *PythonAnalyzer) ExtractCallSites(fn *callgraph.FunctionDef) ([]*callgraph.CallSite, error) {
	return nil, fmt.Errorf("not implemented (PR-04)")
}

// ExtractStatements returns all statements within a function.
// Not implemented in Phase 1 - will be added in PR-04.
func (p *PythonAnalyzer) ExtractStatements(fn *callgraph.FunctionDef) ([]*callgraph.Statement, error) {
	return nil, fmt.Errorf("not implemented (PR-04)")
}

// ExtractVariables returns all variable declarations/assignments.
// Not implemented in Phase 1 - will be added in PR-04.
func (p *PythonAnalyzer) ExtractVariables(fn *callgraph.FunctionDef) ([]*callgraph.Variable, error) {
	return nil, fmt.Errorf("not implemented (PR-04)")
}

// AnalyzeTaint performs taint analysis on function CFG.
// Not implemented in Phase 1 - will be added in PR-04.
func (p *PythonAnalyzer) AnalyzeTaint(fn *callgraph.FunctionDef, cfg *callgraph.CFG) (*callgraph.TaintSummary, error) {
	return nil, fmt.Errorf("not implemented (PR-04)")
}

// ResolveType resolves a type expression to TypeInfo.
// Not implemented in Phase 1 - will be added in PR-03.
func (p *PythonAnalyzer) ResolveType(expr string, context *callgraph.TypeContext) (*callgraph.TypeInfo, error) {
	return nil, fmt.Errorf("not implemented (PR-03)")
}

// SupportsFramework checks if the analyzer supports a framework.
func (p *PythonAnalyzer) SupportsFramework(name string) bool {
	// Python frameworks we detect
	frameworks := map[string]bool{
		"flask":     true,
		"django":    true,
		"fastapi":   true,
		"pyramid":   true,
		"tornado":   true,
		"bottle":    true,
		"cherrypy":  true,
		"aiohttp":   true,
		"sanic":     true,
		"starlette": true,
	}
	return frameworks[strings.ToLower(name)]
}
