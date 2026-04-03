package builder

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// parsedFile holds a cached parsed tree and source code for a Go source file.
type parsedFile struct {
	tree       *sitter.Tree
	sourceCode []byte
}

// GenerateGoTaintSummaries populates taint-analysis data for all Go functions.
// This is the Go equivalent of GenerateTaintSummaries (taint.go:26) for Python.
//
// For each Go function in callGraph.Functions:
//  1. Read source file via ReadFileBytes (helpers.go:19)
//  2. Parse with Go tree-sitter grammar (golang.GetLanguage)
//  3. Find function AST node via findGoNodeByByteRange (helpers.go)
//  4. Extract statements via extraction.ExtractGoStatements (PR-01)
//  5. Build def-use chains via core.BuildDefUseChains
//  6. Analyze intra-procedural taint via taint.AnalyzeIntraProceduralTaint
//  7. Store results in callGraph.Statements, callGraph.Summaries
//
// Parameters include typeEngine, registry, and importMaps for forward compatibility
// with PR-05 (type enrichment pass) — unused in this PR but signature is stable.
func GenerateGoTaintSummaries(
	callGraph *core.CallGraph,
	codeGraph *graph.CodeGraph,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMaps map[string]*core.GoImportMap,
) {
	_ = typeEngine // Reserved for PR-05 type enrichment
	_ = importMaps // Reserved for PR-05 type enrichment

	// Cache parsed trees per file to avoid re-parsing the same file
	// for multiple functions in the same source file.
	fileCache := make(map[string]*parsedFile)

	// Clean up all cached trees at the end.
	defer func() {
		for _, pf := range fileCache {
			if pf.tree != nil {
				pf.tree.Close()
			}
		}
	}()

	analyzed := 0

	for funcFQN, funcNode := range callGraph.Functions {
		// Only process Go functions — skip Python, Java, etc.
		if funcNode.Language != "go" {
			continue
		}

		// SourceLocation is required to find the function in the AST.
		if funcNode.SourceLocation == nil {
			continue
		}

		// Get or parse the source file.
		pf, ok := fileCache[funcNode.File]
		if !ok {
			sourceCode, err := ReadFileBytes(funcNode.File)
			if err != nil {
				log.Printf("Warning: failed to read file %s for Go taint analysis: %v", funcNode.File, err)
				continue
			}

			parser := sitter.NewParser()
			parser.SetLanguage(golang.GetLanguage())

			tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
			parser.Close()
			if err != nil {
				log.Printf("Warning: failed to parse %s for Go taint analysis: %v", funcNode.File, err)
				continue
			}

			pf = &parsedFile{tree: tree, sourceCode: sourceCode}
			fileCache[funcNode.File] = pf
		}

		// Find the function node in the AST by byte range.
		// Go uses SourceLocation{StartByte, EndByte} set by setGoSourceLocation in parser_golang.go.
		functionASTNode := findGoNodeByByteRange(
			pf.tree.RootNode(),
			funcNode.SourceLocation.StartByte,
			funcNode.SourceLocation.EndByte,
		)
		if functionASTNode == nil {
			continue
		}

		// Step 1: Extract statements from function body.
		statements, err := extraction.ExtractGoStatements(funcNode.File, pf.sourceCode, functionASTNode)
		if err != nil {
			log.Printf("Warning: failed to extract Go statements from %s: %v", funcFQN, err)
			continue
		}

		// Store statements for demand-driven dataflow analysis (Tier 2 feed).
		callGraph.Statements[funcFQN] = statements

		// NOTE: CFG building is added in PR-03. For now, skip CFG (Tier 1 unavailable).

		// NOTE: Type enrichment is added in PR-05. For now, statements have raw
		// variable-prefixed AttributeAccess/CallChain (e.g., "r.URL.Path" not
		// "net/http.Request.URL.Path").

		// Step 2: Build def-use chains.
		defUseChain := core.BuildDefUseChains(statements)

		// Step 3: Analyze intra-procedural taint.
		// Empty sources/sinks/sanitizers — DataflowExecutor provides these at query time.
		summary := taint.AnalyzeIntraProceduralTaint(
			funcFQN,
			statements,
			defUseChain,
			[]string{}, // sources — from rule at execution time
			[]string{}, // sinks — from rule at execution time
			[]string{}, // sanitizers — from rule at execution time
		)

		// Step 4: Store summary.
		callGraph.Summaries[funcFQN] = summary

		analyzed++
	}

	if analyzed > 0 {
		fmt.Fprintf(os.Stderr, "  Pass 5: Generated Go taint summaries for %d functions\n", analyzed)
	}

	// Phase 2: Extract package-level variable declarations into synthetic init scopes.
	extractGoPackageLevelVars(callGraph, fileCache, registry, codeGraph)
}

// extractGoPackageLevelVars scans Go source files for top-level var_declaration nodes
// and creates synthetic function scopes to hold their statements.
//
// For each file with package-level vars, creates:
//   - FQN: "<importPath>.init$vars" (e.g., "testapp.init$vars")
//   - Statements: one per var_spec with Def, Uses, CallTarget, CallChain
//   - A synthetic graph.Node entry in callGraph.Functions
//
// This allows inter-procedural analysis to detect taint flowing from
// package-level sources (like os.Getenv) into function-level sinks.
func extractGoPackageLevelVars(callGraph *core.CallGraph, fileCache map[string]*parsedFile, registry *core.GoModuleRegistry, codeGraph *graph.CodeGraph) {
	// Collect all Go source files — some files (e.g., config.go with only var
	// declarations) may not have functions and thus aren't in fileCache yet.
	goFiles := make(map[string]bool)
	for _, node := range codeGraph.Nodes {
		if node.Language == "go" && node.File != "" {
			goFiles[node.File] = true
		}
	}

	// Parse any Go files not already in the cache.
	for filePath := range goFiles {
		if _, ok := fileCache[filePath]; ok {
			continue
		}
		sourceCode, err := ReadFileBytes(filePath)
		if err != nil {
			continue
		}
		parser := sitter.NewParser()
		parser.SetLanguage(golang.GetLanguage())
		tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
		parser.Close()
		if err != nil {
			continue
		}
		fileCache[filePath] = &parsedFile{tree: tree, sourceCode: sourceCode}
	}

	// Track which packages we've already processed.
	processedPackages := make(map[string]bool)

	for filePath, pf := range fileCache {
		if pf.tree == nil {
			continue
		}

		root := pf.tree.RootNode()
		if root == nil {
			continue
		}

		// Resolve the module import path for this file's directory.
		// Uses registry.DirToImport (same as buildGoFQN in go_builder.go).
		var packagePath string
		if registry != nil {
			dirPath := filepath.Dir(filePath)
			if !filepath.IsAbs(dirPath) {
				if abs, err := filepath.Abs(dirPath); err == nil {
					dirPath = abs
				}
			}
			if importPath, ok := registry.DirToImport[dirPath]; ok {
				packagePath = importPath
			}
		}

		// Fallback: extract short package name from package_clause.
		if packagePath == "" {
			for i := 0; i < int(root.ChildCount()); i++ {
				child := root.Child(i)
				if child != nil && child.Type() == "package_clause" {
					for j := 0; j < int(child.ChildCount()); j++ {
						gc := child.Child(j)
						if gc != nil && gc.Type() == "package_identifier" {
							packagePath = gc.Content(pf.sourceCode)
							break
						}
					}
					break
				}
			}
		}

		if packagePath == "" {
			continue
		}

		initFQN := packagePath + ".init$vars"
		processedPackages[initFQN] = true

		// Find all top-level var_declaration nodes.
		var pkgVarStmts []*core.Statement
		for i := 0; i < int(root.ChildCount()); i++ {
			child := root.Child(i)
			if child == nil || child.Type() != "var_declaration" {
				continue
			}

			varStmts := extraction.ExtractGoVarDeclFromNode(child, pf.sourceCode)
			for _, stmt := range varStmts {
				stmt.LineNumber = uint32(child.StartPoint().Row + 1) //nolint:unconvert
				pkgVarStmts = append(pkgVarStmts, stmt)
			}
		}

		if len(pkgVarStmts) == 0 {
			continue
		}

		// Store statements in synthetic scope.
		if existing, ok := callGraph.Statements[initFQN]; ok {
			callGraph.Statements[initFQN] = append(existing, pkgVarStmts...)
		} else {
			callGraph.Statements[initFQN] = pkgVarStmts
		}

		// Create a synthetic function node so inter-procedural analysis can find it.
		if _, exists := callGraph.Functions[initFQN]; !exists {
			callGraph.Functions[initFQN] = &graph.Node{
				ID:       "init$vars:" + packagePath,
				Type:     "init_function",
				Name:     "init$vars",
				Language: "go",
				File:     filePath,
			}
		}

		// Build def-use chains and summary for the synthetic scope.
		allStmts := callGraph.Statements[initFQN]
		defUseChain := core.BuildDefUseChains(allStmts)
		summary := taint.AnalyzeIntraProceduralTaint(
			initFQN,
			allStmts,
			defUseChain,
			[]string{}, []string{}, []string{},
		)
		callGraph.Summaries[initFQN] = summary
	}
}
