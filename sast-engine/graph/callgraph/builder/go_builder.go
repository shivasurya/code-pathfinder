package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// CallSiteInternal represents a function call location during graph construction.
// This is an internal structure used during Pass 2 and Pass 3 before final resolution.
type CallSiteInternal struct {
	CallerFQN    string // FQN of the function containing this call
	CallerFile   string // File path where the call occurs
	CallLine     uint32 // Line number of the call
	FunctionName string // Simple name of the function being called (e.g., "Println", "Helper")
	ObjectName   string // Package/object name for qualified calls (e.g., "fmt", "utils"), empty for simple calls
	Arguments    []string
}

// BuildGoCallGraph constructs the call graph for a Go project using a 5-pass algorithm.
//
// Pass 1: Index functions from CodeGraph → populate CallGraph.Functions
// Pass 2a: Extract return types from all functions (PR-14)
// Pass 2b: Extract variable assignments from all functions (PR-15)
// Pass 3: Extract call sites from call_expression nodes → create CallSiteInternal list
// Pass 4: Resolve call targets to FQNs → add edges to CallGraph
//
// Parameters:
//   - codeGraph: the existing code graph with parsed AST nodes from PR-06
//   - registry: Go module registry from PR-07 with import path mappings
//   - typeEngine: Go type inference engine for Phase 2 type tracking (PR-14/PR-15)
//
// Returns:
//   - CallGraph: complete call graph with resolved edges and type information
//   - error: if any critical step fails
func BuildGoCallGraph(codeGraph *graph.CodeGraph, registry *core.GoModuleRegistry, typeEngine *resolution.GoTypeInferenceEngine) (*core.CallGraph, error) {
	callGraph := core.NewCallGraph()

	// Store type engine in call graph for MCP tool access
	if typeEngine != nil {
		callGraph.GoTypeEngine = typeEngine
	}

	// Build import map cache for each source file
	// Map: filePath → GoImportMap
	importMaps := make(map[string]*core.GoImportMap)

	for _, node := range codeGraph.Nodes {
		if node.File != "" && importMaps[node.File] == nil {
			sourceCode, err := os.ReadFile(node.File)
			if err != nil {
				continue // Skip files we can't read
			}

			importMap, err := resolution.ExtractGoImports(node.File, sourceCode, registry)
			if err != nil {
				continue // Skip files with import extraction errors
			}

			importMaps[node.File] = importMap
		}
	}

	// Pass 1: Index all function definitions
	fmt.Fprintf(os.Stderr, "  Pass 1: Indexing functions...\n")
	functionContext := indexGoFunctions(codeGraph, callGraph, registry)
	fmt.Fprintf(os.Stderr, "    Indexed %d functions\n", len(callGraph.Functions))

	// Pass 2a: Extract return types from all indexed Go functions
	// Only run if typeEngine is provided (not nil)
	// ExtractGoReturnTypes operates on already-indexed functions in callGraph
	if typeEngine != nil {
		fmt.Fprintf(os.Stderr, "  Pass 2a: Extracting return types...\n")
		_ = extraction.ExtractGoReturnTypes(callGraph, registry, typeEngine)

		// Pass 2b: Extract variable assignments from all Go source files (parallel)
		// Collect all Go source files
		goFiles := make(map[string]bool)
		for _, node := range codeGraph.Nodes {
			if node.File != "" && strings.HasSuffix(node.File, ".go") {
				goFiles[node.File] = true
			}
		}

		totalFiles := len(goFiles)
		fmt.Fprintf(os.Stderr, "  Pass 2b: Extracting variable assignments (%d files)...\n", totalFiles)

		// Determine optimal worker count (same pattern as Python builder)
		numWorkers := getOptimalWorkerCount()

		// Create job queue for parallel processing
		varJobs := make(chan string, 100)
		var varProcessed atomic.Int64
		var wg sync.WaitGroup

		// Start workers for variable assignment extraction
		for range numWorkers {
			wg.Go(func() {
				for filePath := range varJobs {
					sourceCode, err := ReadFileBytes(filePath)
					if err != nil {
						continue // Skip files we can't read
					}

					// Extract variable assignments for this file
					// ExtractGoVariableAssignments is thread-safe (uses mutex internally)
					_ = extraction.ExtractGoVariableAssignments(filePath, sourceCode, typeEngine, registry, importMaps[filePath])

					// Progress tracking
					count := varProcessed.Add(1)
					if count%50 == 0 || count == int64(totalFiles) {
						percentage := float64(count) / float64(totalFiles) * 100
						fmt.Fprintf(os.Stderr, "\r    Variable assignments: %d/%d (%.1f%%)", count, totalFiles, percentage)
					}
				}
			})
		}

		// Queue all Go files for variable assignment extraction
		for filePath := range goFiles {
			varJobs <- filePath
		}
		close(varJobs)
		wg.Wait()

		// Final progress line
		if totalFiles > 0 {
			fmt.Fprintf(os.Stderr, "\r    Variable assignments: %d/%d (100.0%%)\n", totalFiles, totalFiles)
		}
	}

	// Pass 3: Extract call sites from call_expression nodes
	fmt.Fprintf(os.Stderr, "  Pass 3: Extracting call sites...\n")
	callSites := extractGoCallSitesFromCodeGraph(codeGraph, callGraph)
	fmt.Fprintf(os.Stderr, "    Found %d call sites\n", len(callSites))

	// Pass 4: Resolve call targets and add edges
	fmt.Fprintf(os.Stderr, "  Pass 4: Resolving call targets...\n")
	totalCallSites := len(callSites)
	resolvedCount := 0

	for i, callSite := range callSites {
		importMap := importMaps[callSite.CallerFile]
		if importMap == nil {
			// No import map - can still resolve builtins and same-package calls
			importMap = core.NewGoImportMap(callSite.CallerFile)
		}

		targetFQN, resolved := resolveGoCallTarget(callSite, importMap, registry, functionContext, typeEngine, callGraph)

		if resolved {
			resolvedCount++
			// Add edge from caller to callee
			callGraph.AddEdge(callSite.CallerFQN, targetFQN)

			// Add detailed call site information
			callGraph.AddCallSite(callSite.CallerFQN, core.CallSite{
				Target: callSite.FunctionName,
				Location: core.Location{
					File: callSite.CallerFile,
					Line: int(callSite.CallLine),
				},
				Resolved:  true,
				TargetFQN: targetFQN,
			})
		} else {
			// Record unresolved call for diagnostics
			callGraph.AddCallSite(callSite.CallerFQN, core.CallSite{
				Target: callSite.FunctionName,
				Location: core.Location{
					File: callSite.CallerFile,
					Line: int(callSite.CallLine),
				},
				Resolved:      false,
				FailureReason: "unresolved_go_call",
			})
		}

		// Progress tracking
		if (i+1)%500 == 0 || i+1 == totalCallSites {
			percentage := float64(i+1) / float64(totalCallSites) * 100
			resolutionRate := float64(resolvedCount) / float64(i+1) * 100
			fmt.Fprintf(os.Stderr, "\r    Call targets: %d/%d (%.1f%%) - %.1f%% resolved",
				i+1, totalCallSites, percentage, resolutionRate)
		}
	}

	// Final summary
	if totalCallSites > 0 {
		finalResolutionRate := float64(resolvedCount) / float64(totalCallSites) * 100
		fmt.Fprintf(os.Stderr, "\r    Call targets: %d/%d (100.0%%) - %.1f%% resolved\n",
			totalCallSites, totalCallSites, finalResolutionRate)
	}

	return callGraph, nil
}

// indexGoFunctions indexes all function definitions from the CodeGraph.
// This is Pass 1 of the 3-pass algorithm.
//
// Handles:
//   - function_definition: package-level functions
//   - method: methods with receivers
//   - func_literal: anonymous functions/closures
//
// Returns:
//   - functionContext: map from simple name to list of nodes for resolution
func indexGoFunctions(codeGraph *graph.CodeGraph, callGraph *core.CallGraph, registry *core.GoModuleRegistry) map[string][]*graph.Node {
	functionContext := make(map[string][]*graph.Node)

	totalNodes := len(codeGraph.Nodes)
	processed := 0
	indexed := 0

	for _, node := range codeGraph.Nodes {
		processed++

		// Progress tracking every 5000 nodes
		if processed%5000 == 0 {
			percentage := float64(processed) / float64(totalNodes) * 100
			fmt.Fprintf(os.Stderr, "\r    Scanning nodes: %d/%d (%.1f%%) - %d functions found",
				processed, totalNodes, percentage, indexed)
		}

		// Only index Go function-like nodes
		if node.Type != "function_declaration" && node.Type != "method" && node.Type != "func_literal" {
			continue
		}

		// Build FQN using module registry
		fqn := buildGoFQN(node, codeGraph, registry)

		// Add to CallGraph.Functions
		callGraph.Functions[fqn] = node

		// Add to function context for name-based lookup
		functionContext[node.Name] = append(functionContext[node.Name], node)
		indexed++
	}

	// Final line
	if totalNodes > 0 {
		fmt.Fprintf(os.Stderr, "\r    Scanning nodes: %d/%d (100.0%%) - %d functions found\n",
			totalNodes, totalNodes, indexed)
	}

	return functionContext
}

// extractGoCallSitesFromCodeGraph extracts call sites from call_expression nodes.
// This is Pass 3 of the 5-pass algorithm.
//
// Reuses call_expression nodes created in PR-06 to avoid AST re-parsing.
// Converts each call node to a CallSiteInternal struct for resolution.
//
// Returns:
//   - list of CallSiteInternal structs ready for resolution in Pass 4
func extractGoCallSitesFromCodeGraph(codeGraph *graph.CodeGraph, callGraph *core.CallGraph) []*CallSiteInternal {
	callSites := make([]*CallSiteInternal, 0)

	// Build reverse map: node ID → FQN (O(n) once instead of O(n²) per call)
	nodeIDToFQN := make(map[string]string, len(callGraph.Functions))
	for fqn, funcNode := range callGraph.Functions {
		nodeIDToFQN[funcNode.ID] = fqn
	}

	totalNodes := len(codeGraph.Nodes)
	processed := 0
	callNodesFound := 0

	for _, node := range codeGraph.Nodes {
		processed++

		// Progress tracking every 5000 nodes
		if processed%5000 == 0 {
			percentage := float64(processed) / float64(totalNodes) * 100
			fmt.Fprintf(os.Stderr, "\r    Scanning for calls: %d/%d (%.1f%%) - %d calls found",
				processed, totalNodes, percentage, callNodesFound)
		}

		// Go call nodes are either "call" or "method_expression"
		if node.Type != "call" && node.Type != "method_expression" {
			continue
		}

		callNodesFound++

		// Extract function name and object name
		// Function name is in node.Name
		// Object name is in node.Interface[0] for method calls
		functionName := node.Name
		var objectName string
		if len(node.Interface) > 0 {
			objectName = node.Interface[0]
		}

		// Find containing function to get caller FQN
		containingFunc := findContainingGoFunction(node, codeGraph)
		var callerFQN string
		if containingFunc != nil {
			// Fast O(1) lookup using reverse map
			callerFQN = nodeIDToFQN[containingFunc.ID]
		}

		callSite := &CallSiteInternal{
			CallerFQN:    callerFQN,
			CallerFile:   node.File,
			CallLine:     node.LineNumber,
			FunctionName: functionName,
			ObjectName:   objectName,
		}

		callSites = append(callSites, callSite)
	}

	// Final line
	if totalNodes > 0 {
		fmt.Fprintf(os.Stderr, "\r    Scanning for calls: %d/%d (100.0%%) - %d calls found\n",
			totalNodes, totalNodes, callNodesFound)
	}

	return callSites
}

// resolveGoCallTarget resolves a call site to a fully qualified name.
// This is Pass 4 of the 5-pass algorithm.
//
// Resolution patterns:
//
//	1a. Qualified import call: fmt.Println → resolve "fmt" via imports → "fmt.Println"
//	1b. Variable method call: user.Save() → resolve "user" via typeEngine → "pkg.User.Save" (PR-17)
//	2. Same-package call: Helper() → find in functionContext → "github.com/myapp/utils.Helper"
//	3. Builtin call: append() → "builtin.append"
//	4. Unresolved: return false
//
// Parameters:
//   - callSite: the call site to resolve
//   - importMap: imports for the caller's file (from PR-07)
//   - registry: module registry with stdlib information
//   - functionContext: map from simple name to nodes
//   - typeEngine: Go type inference engine for variable type lookup (PR-14/PR-15/PR-16)
//   - callGraph: call graph for method existence verification (PR-16)
//
// Returns:
//   - targetFQN: the resolved fully qualified name
//   - resolved: true if resolution succeeded
func resolveGoCallTarget(
	callSite *CallSiteInternal,
	importMap *core.GoImportMap,
	registry *core.GoModuleRegistry,
	functionContext map[string][]*graph.Node,
	typeEngine *resolution.GoTypeInferenceEngine,
	callGraph *core.CallGraph,
) (string, bool) {
	// Pattern 1a: Qualified call (pkg.Func or obj.Method)
	if callSite.ObjectName != "" {
		// Try import resolution first (existing pattern)
		importPath, ok := importMap.Resolve(callSite.ObjectName)
		if ok {
			// Successfully resolved import path
			targetFQN := importPath + "." + callSite.FunctionName
			return targetFQN, true
		}

		// Pattern 1b: Variable-based method resolution (PR-17)
		// If import resolution failed, try resolving as variable.method()
		// Example: user.Save() where user is a variable of type *User
		if typeEngine != nil && callGraph != nil && callSite.CallerFQN != "" {
			scope := typeEngine.GetScope(callSite.CallerFQN)
			if scope != nil {
				binding := scope.GetVariable(callSite.ObjectName)
				if binding != nil && binding.Type != nil {
					// Build method FQN: "pkg.Type.Method"
					// Handle pointer types: *User -> User (methods are defined on the type, not the pointer in FQN)
					typeFQN := binding.Type.TypeFQN
					if after, ok0 := strings.CutPrefix(typeFQN, "*"); ok0 {
						typeFQN = after
					}

					methodFQN := typeFQN + "." + callSite.FunctionName

					// Verify method exists in callGraph before returning
					// This prevents false positives from unindexed methods
					if callGraph.Functions[methodFQN] != nil {
						return methodFQN, true
					}
				}
			}
		}

		// Import not found and variable not found - unresolved
		return "", false
	}

	// Pattern 2: Same-package call (simple function name)
	candidates := functionContext[callSite.FunctionName]
	for _, candidate := range candidates {
		// Check if candidate is in the same package as caller
		if isSameGoPackage(callSite.CallerFile, candidate.File) {
			// Build FQN for this candidate
			candidateFQN := buildGoFQN(candidate, nil, registry)
			return candidateFQN, true
		}
	}

	// Pattern 3: Builtin function
	if isBuiltin(callSite.FunctionName) {
		return "builtin." + callSite.FunctionName, true
	}

	// Pattern 4: Unresolved
	return "", false
}

// buildGoFQN constructs a fully qualified name for a Go function, method, or closure.
//
// FQN formats:
//   - Package function: "github.com/myapp/handlers.HandleRequest"
//   - Method: "github.com/myapp/models.Server.Start"
//   - Closure: "github.com/myapp/handlers.HandleRequest.$anon_1"
//
// Parameters:
//   - node: the function node (function_declaration, method, func_literal)
//   - codeGraph: the code graph for parent lookup
//   - registry: module registry for import path mapping
//
// Returns:
//   - fully qualified name string
func buildGoFQN(node *graph.Node, codeGraph *graph.CodeGraph, registry *core.GoModuleRegistry) string {
	// Get directory path for this file
	dirPath := filepath.Dir(node.File)

	// Convert to absolute path if relative (for registry lookup)
	if !filepath.IsAbs(dirPath) {
		if absPath, err := filepath.Abs(dirPath); err == nil {
			dirPath = absPath
		}
	}

	// Look up import path from registry
	importPath, ok := registry.DirToImport[dirPath]
	if !ok {
		// Fallback: use file name
		importPath = filepath.Base(dirPath)
	}

	switch node.Type {
	case "function_declaration":
		// Package-level function: module.Function
		return importPath + "." + node.Name

	case "method":
		// Method: module.Receiver.Method
		// Receiver type is stored in node.Interface[0]
		if len(node.Interface) > 0 && node.Interface[0] != "" {
			return importPath + "." + node.Interface[0] + "." + node.Name
		}
		// Fallback if no receiver type
		return importPath + "." + node.Name

	case "func_literal":
		// Closure: parentFQN.$anon_N
		// Find parent function
		parent := findParentGoFunction(node, codeGraph)
		if parent != nil {
			parentFQN := buildGoFQN(parent, codeGraph, registry)
			return parentFQN + "." + node.Name // Name is already "$anon_N" from PR-06
		}
		// Orphaned closure - shouldn't happen but handle gracefully
		return importPath + "." + node.Name

	default:
		return importPath + "." + node.Name
	}
}

// findContainingGoFunction finds the function/method/closure that contains a given call node.
// Walks parent edges in the CodeGraph to find the first function-like ancestor.
//
// Returns:
//   - Node pointer to the containing function, or nil if no containing function found
func findContainingGoFunction(callNode *graph.Node, codeGraph *graph.CodeGraph) *graph.Node {
	// Build parent map from CodeGraph edges
	parentMap := make(map[string]*graph.Node)
	for _, node := range codeGraph.Nodes {
		for _, edge := range node.OutgoingEdges {
			parentMap[edge.To.ID] = node
		}
	}

	// Walk up the parent chain
	current := callNode
	for {
		parent := parentMap[current.ID]
		if parent == nil {
			break
		}

		// Check if parent is a function-like node
		if parent.Type == "function_declaration" || parent.Type == "method" || parent.Type == "func_literal" {
			return parent
		}

		current = parent
	}

	return nil
}

// findParentGoFunction finds the immediate parent function for a closure.
// Used by buildGoFQN for closure FQN generation.
func findParentGoFunction(closureNode *graph.Node, codeGraph *graph.CodeGraph) *graph.Node {
	// Build parent map
	parentMap := make(map[string]*graph.Node)
	for _, node := range codeGraph.Nodes {
		for _, edge := range node.OutgoingEdges {
			parentMap[edge.To.ID] = node
		}
	}

	// Walk up to find parent function
	current := closureNode
	for {
		parent := parentMap[current.ID]
		if parent == nil {
			return nil
		}

		if parent.Type == "function_declaration" || parent.Type == "method" || parent.Type == "func_literal" {
			return parent
		}

		current = parent
	}
}

// isBuiltin returns true if the function name is a Go builtin.
// Go has 15 builtin functions that are always available.
func isBuiltin(name string) bool {
	builtins := map[string]bool{
		"append":  true,
		"cap":     true,
		"close":   true,
		"complex": true,
		"copy":    true,
		"delete":  true,
		"imag":    true,
		"len":     true,
		"make":    true,
		"new":     true,
		"panic":   true,
		"print":   true,
		"println": true,
		"real":    true,
		"recover": true,
	}
	return builtins[name]
}

// isSameGoPackage returns true if two file paths belong to the same Go package.
// In Go, a package is all files in the same directory.
func isSameGoPackage(file1, file2 string) bool {
	dir1 := filepath.Dir(file1)
	dir2 := filepath.Dir(file2)
	return dir1 == dir2
}
