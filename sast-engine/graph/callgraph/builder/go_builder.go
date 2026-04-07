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
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
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
func BuildGoCallGraph(codeGraph *graph.CodeGraph, registry *core.GoModuleRegistry, typeEngine *resolution.GoTypeInferenceEngine, logger *output.Logger) (*core.CallGraph, error) {
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
	functionContext := indexGoFunctions(codeGraph, callGraph, registry, typeEngine)
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
					_ = extraction.ExtractGoVariableAssignments(filePath, sourceCode, typeEngine, registry, importMaps[filePath], callGraph)

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

	// Pre-index package-level variables for Source 3 lookup in resolveGoCallTarget.
	// Without this, Source 3 scans all 225k+ nodes for every unresolved method call.
	// Key: filepath.Dir(file) + "::" + varName → node
	pkgVarIndex := buildPkgVarIndex(codeGraph)

	// Pre-index struct field types for Source 4 lookup (chained field access: a.Field.Method()).
	// Key: "pkgPath.TypeName.FieldName" → resolved field type FQN
	callGraph.GoStructFieldIndex = buildStructFieldIndex(codeGraph, registry, importMaps)

	totalCallSites := len(callSites)

	// pass4Result holds the fully computed output for one call site, produced by
	// a worker goroutine and later applied to the CallGraph sequentially.
	// Keeping resolution and write separate means workers touch zero shared state.
	type pass4Result struct {
		callerFQN string
		targetFQN string // empty when unresolved
		resolved  bool
		callSite  core.CallSite
	}

	// Stage 1: Resolve call sites in parallel.
	//
	// Safety proof — every shared structure accessed inside this stage is either:
	//   • Immutable by Pass 4 (importMaps, callGraph.Functions,
	//     callGraph.GoStructFieldIndex, functionContext, pkgVarIndex,
	//     registry.DirToImport): written only during earlier passes, never here.
	//   • Protected by sync.RWMutex (typeEngine.GetScope/GetVariable,
	//     StdlibLoader.ValidateStdlibImport/GetType,
	//     ThirdPartyLoader.ValidateImport/GetType): concurrent reads are safe.
	//   • Worker-local (importMap fallback allocation, callSite value copy,
	//     local variables, result slice): not shared at all.
	//
	// The ONLY writes in Pass 4 (AddEdge, AddCallSite) happen in Stage 2.
	numWorkers := getOptimalWorkerCount()
	shardResults := make([][]pass4Result, numWorkers)
	chunkSize := (totalCallSites + numWorkers - 1) / numWorkers

	var resolveWg sync.WaitGroup
	var processedCount atomic.Int64

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, totalCallSites)
		if start >= totalCallSites {
			break
		}
		shardIdx := w // capture for closure

		resolveWg.Add(1)
		go func() {
			defer resolveWg.Done()
			local := make([]pass4Result, 0, end-start)

			for _, callSite := range callSites[start:end] {
				importMap := importMaps[callSite.CallerFile]
				if importMap == nil {
					// No import map - can still resolve builtins and same-package calls
					importMap = core.NewGoImportMap(callSite.CallerFile)
				}

				targetFQN, resolved, isStdlib, resolveSource := resolveGoCallTarget(
					callSite, importMap, registry, functionContext, typeEngine,
					callGraph, pkgVarIndex, logger)

				var cs core.CallSite
				if resolved {
					// Populate type inference metadata from parameter types or variable bindings.
					// NOTE: this block is intentionally preserved verbatim from the original
					// sequential loop — do not refactor or deduplicate with resolveGoCallTarget.
					var inferredType string
					var typeConfidence float32
					var typeSource string
					var wasTypeResolved bool

					if callSite.ObjectName != "" {
						// Source 1: Function parameter types
						if callerNode, exists := callGraph.Functions[callSite.CallerFQN]; exists {
							for pi, paramName := range callerNode.MethodArgumentsValue {
								if paramName == callSite.ObjectName && pi < len(callerNode.MethodArgumentsType) {
									typeStr := callerNode.MethodArgumentsType[pi]
									if colonIdx := strings.Index(typeStr, ": "); colonIdx >= 0 {
										typeStr = typeStr[colonIdx+2:]
									}
									typeStr = strings.TrimPrefix(typeStr, "*")
									im := importMaps[callSite.CallerFile]
									inferredType = resolveGoTypeFQN(typeStr, im)
									typeConfidence = 0.95
									typeSource = "go_function_parameter"
									wasTypeResolved = true
									break
								}
							}
						}

						// Source 2: Local variable type bindings from GoTypeInferenceEngine
						if !wasTypeResolved && typeEngine != nil {
							scope := typeEngine.GetScope(callSite.CallerFQN)
							if scope != nil {
								binding := scope.GetVariable(callSite.ObjectName)
								if binding != nil && binding.Type != nil {
									typeFQN := binding.Type.TypeFQN
									if after, ok := strings.CutPrefix(typeFQN, "*"); ok {
										typeFQN = after
									}
									inferredType = typeFQN
									typeConfidence = binding.Type.Confidence
									typeSource = "go_variable_binding"
									wasTypeResolved = true
								}
							}
						}
					}

					// Propagate resolve source from the resolver (e.g. third-party).
					// This overrides the type-inference source so stats correctly attribute
					// calls resolved via GoThirdPartyLoader rather than counting them as
					// user-code resolutions.
					if resolveSource != "" {
						typeSource = resolveSource
					}

					// Convert CallSiteInternal.Arguments to core.Argument structs.
					args := buildCallSiteArguments(callSite.Arguments)

					cs = core.CallSite{
						Target: callSite.FunctionName,
						Location: core.Location{
							File: callSite.CallerFile,
							Line: int(callSite.CallLine),
						},
						Arguments:                args,
						Resolved:                 true,
						TargetFQN:                targetFQN,
						IsStdlib:                 isStdlib,
						ResolvedViaTypeInference: wasTypeResolved,
						InferredType:             inferredType,
						TypeConfidence:           typeConfidence,
						TypeSource:               typeSource,
					}
				} else {
					args := buildCallSiteArguments(callSite.Arguments)

					// Record unresolved call for diagnostics
					cs = core.CallSite{
						Target: callSite.FunctionName,
						Location: core.Location{
							File: callSite.CallerFile,
							Line: int(callSite.CallLine),
						},
						Arguments:     args,
						Resolved:      false,
						FailureReason: "unresolved_go_call",
					}
				}

				local = append(local, pass4Result{
					callerFQN: callSite.CallerFQN,
					targetFQN: targetFQN,
					resolved:  resolved,
					callSite:  cs,
				})

				// Progress tracking (atomic — safe from multiple goroutines).
				// Uses the same \r overwrite pattern as Pass 2b.
				count := processedCount.Add(1)
				if count%500 == 0 || count == int64(totalCallSites) {
					percentage := float64(count) / float64(totalCallSites) * 100
					fmt.Fprintf(os.Stderr, "\r    Call targets: %d/%d (%.1f%%)",
						count, totalCallSites, percentage)
				}
			}
			shardResults[shardIdx] = local
		}()
	}
	resolveWg.Wait()

	// Stage 2: Apply results sequentially — single goroutine, zero locking needed.
	// AddEdge and AddCallSite have no mutex; keeping writes here ensures safety.
	resolvedCount := 0
	stdlibCount := 0
	for _, shard := range shardResults {
		for _, r := range shard {
			if r.resolved {
				resolvedCount++
				if r.callSite.IsStdlib {
					stdlibCount++
				}
				callGraph.AddEdge(r.callerFQN, r.targetFQN)
			}
			callGraph.AddCallSite(r.callerFQN, r.callSite)
		}
	}

	// Final summary
	if totalCallSites > 0 {
		finalResolutionRate := float64(resolvedCount) / float64(totalCallSites) * 100
		fmt.Fprintf(os.Stderr, "\r    Call targets: %d/%d (100.0%%) - %.1f%% resolved\n",
			totalCallSites, totalCallSites, finalResolutionRate)
		if stdlibCount > 0 && resolvedCount > 0 {
			stdlibRate := float64(stdlibCount) / float64(resolvedCount) * 100
			fmt.Fprintf(os.Stderr, "    Stdlib calls: %d (%.1f%% of resolved)\n",
				stdlibCount, stdlibRate)
		}
	}

	// Pass 5: Generate taint summaries for all Go functions.
	// Populates callGraph.Statements and callGraph.Summaries (Tier 2 feed for DataflowExecutor).
	// CFG population (Tier 1) is added in PR-03. Type enrichment in PR-05.
	GenerateGoTaintSummaries(callGraph, codeGraph, typeEngine, registry, importMaps)

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
func indexGoFunctions(codeGraph *graph.CodeGraph, callGraph *core.CallGraph, registry *core.GoModuleRegistry, typeEngine *resolution.GoTypeInferenceEngine) map[string][]*graph.Node {
	functionContext := make(map[string][]*graph.Node)

	// Build parent map once for closure FQN construction (func_literal nodes need their parent).
	// Without this, buildGoFQN → findParentGoFunction would rebuild the map for every closure.
	parentMap := buildParentMap(codeGraph)

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
		fqn := buildGoFQN(node, parentMap, registry)

		// Add to CallGraph.Functions
		callGraph.Functions[fqn] = node

		// Eagerly create scope so Pattern 1b Source 2 always finds one.
		// Guard with GetScope == nil so Pass 2b bindings are not overwritten.
		if typeEngine != nil && typeEngine.GetScope(fqn) == nil {
			typeEngine.AddScope(resolution.NewGoFunctionScope(fqn))
		}

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

	// Build parent map once here so findContainingGoFunction doesn't rebuild it
	// for every call node (was O(call_nodes × total_nodes) before this fix).
	parentMap := buildParentMap(codeGraph)

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
		containingFunc := findContainingGoFunction(node, parentMap)
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
			Arguments:    node.MethodArgumentsValue, // argument expressions from AST
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
//   - isStdlib: true when the target is a Go standard library function
//   - resolveSource: "thirdparty_local" when resolved via GoThirdPartyLoader; "" otherwise
func resolveGoCallTarget(
	callSite *CallSiteInternal,
	importMap *core.GoImportMap,
	registry *core.GoModuleRegistry,
	functionContext map[string][]*graph.Node,
	typeEngine *resolution.GoTypeInferenceEngine,
	callGraph *core.CallGraph,
	pkgVarIndex map[string]*graph.Node,
	logger *output.Logger,
) (string, bool, bool, string) {
	// Pattern 1a: Qualified call (pkg.Func or obj.Method)
	if callSite.ObjectName != "" {
		// Try import resolution first (existing pattern)
		importPath, ok := importMap.Resolve(callSite.ObjectName)
		if ok {
			// Successfully resolved import path; check if it is a stdlib package.
			targetFQN := importPath + "." + callSite.FunctionName
			isStdlib := registry.StdlibLoader != nil &&
				registry.StdlibLoader.ValidateStdlibImport(importPath)
			if !isStdlib && registry != nil && registry.ThirdPartyLoader != nil &&
				registry.ThirdPartyLoader.ValidateImport(importPath) {
				return targetFQN, true, false, "thirdparty_local"
			}
			return targetFQN, true, isStdlib, ""
		}

		// Pattern 1b: Variable-based method resolution (PR-17 + Approach C)
		// If import resolution failed, try resolving as variable.method()
		// Example: db.Query(sql) where db is *sql.DB, r.FormValue() where r is *http.Request
		if callGraph != nil && callSite.CallerFQN != "" {
			// Source 1: Function parameter types (MethodArgumentsValue/Type).
			// GoTypeInferenceEngine does NOT track function parameters — only := and = assignments.
			// So we check the caller function's parameter list first.
			var typeFQN string
			if callerNode, exists := callGraph.Functions[callSite.CallerFQN]; exists {
				for i, paramName := range callerNode.MethodArgumentsValue {
					if paramName == callSite.ObjectName && i < len(callerNode.MethodArgumentsType) {
						typeStr := callerNode.MethodArgumentsType[i]
						// Go parser stores types as "name: type" (e.g., "r: *http.Request").
						if colonIdx := strings.Index(typeStr, ": "); colonIdx >= 0 {
							typeStr = typeStr[colonIdx+2:]
						}
						typeStr = strings.TrimPrefix(typeStr, "*")
						// Resolve short type via import map: "http.Request" → "net/http.Request"
						typeFQN = resolveGoTypeFQN(typeStr, importMap)
						break
					}
				}
			}

			// Source 2: Local variable types from GoTypeInferenceEngine (from := and = assignments).
			if typeFQN == "" && typeEngine != nil {
				scope := typeEngine.GetScope(callSite.CallerFQN)
				if scope != nil {
					binding := scope.GetVariable(callSite.ObjectName)
					if binding != nil && binding.Type != nil {
						typeFQN = binding.Type.TypeFQN
						typeFQN = strings.TrimPrefix(typeFQN, "*")
					} else if logger != nil && logger.IsDebug() {
						logger.Debug("[debug-1b] %s.%s: scope found but no binding for %q", callSite.CallerFQN, callSite.FunctionName, callSite.ObjectName)
					}
				} else if logger != nil && logger.IsDebug() {
					logger.Debug("[debug-1b] %s.%s: no scope for %q", callSite.CallerFQN, callSite.FunctionName, callSite.CallerFQN)
				}
			}

			// Source 3: Package-level variable types from CodeGraph nodes.
			// Covers `var globalDB *sql.DB` at package scope — not tracked by
			// GoTypeInferenceEngine (which only processes := / = assignments in
			// function bodies). Only fires when Source 1 and Source 2 both fail.
			// Uses pre-built pkgVarIndex (O(1)) instead of a full node scan (O(N)).
			if typeFQN == "" && pkgVarIndex != nil {
				key := filepath.Dir(callSite.CallerFile) + "::" + callSite.ObjectName
				if varNode, ok := pkgVarIndex[key]; ok {
					typeStr := strings.TrimPrefix(varNode.DataType, "*")
					typeFQN = resolveGoTypeFQN(typeStr, importMap)
				}
			}

			// Source 4: Struct field access (a.Field.Method()).
			// Fires only when ObjectName is "root.Field" and Sources 1-3 all failed.
			// Looks up the root variable's type via Sources 1-3, then resolves the
			// field's type from the pre-built struct field index (S4-Source4a) or
			// from the CDN for stdlib types (S4-Source4b).
			if typeFQN == "" && callGraph != nil {
				dotIdx := strings.Index(callSite.ObjectName, ".")
				if dotIdx > 0 {
					rootName := callSite.ObjectName[:dotIdx]
					fieldName := callSite.ObjectName[dotIdx+1:]
					// Only handle simple one-level access; skip chained dots or method calls.
					if !strings.Contains(fieldName, ".") && !strings.Contains(fieldName, "(") {
						var rootTypeFQN string

						// S4-Source1: function parameters
						if callerNode, exists := callGraph.Functions[callSite.CallerFQN]; exists {
							for i, paramName := range callerNode.MethodArgumentsValue {
								if paramName == rootName && i < len(callerNode.MethodArgumentsType) {
									typeStr := callerNode.MethodArgumentsType[i]
									if ci := strings.Index(typeStr, ": "); ci >= 0 {
										typeStr = typeStr[ci+2:]
									}
									rootTypeFQN = resolveGoTypeFQN(strings.TrimPrefix(typeStr, "*"), importMap)
									break
								}
							}
						}
						// S4-Source2: scope variable binding
						if rootTypeFQN == "" && typeEngine != nil {
							scope := typeEngine.GetScope(callSite.CallerFQN)
							if scope != nil {
								if b := scope.GetVariable(rootName); b != nil && b.Type != nil {
									rootTypeFQN = strings.TrimPrefix(b.Type.TypeFQN, "*")
								}
							}
						}
						// S4-Source3: package-level variable
						if rootTypeFQN == "" && pkgVarIndex != nil {
							key := filepath.Dir(callSite.CallerFile) + "::" + rootName
							if varNode, ok := pkgVarIndex[key]; ok {
								rootTypeFQN = resolveGoTypeFQN(strings.TrimPrefix(varNode.DataType, "*"), importMap)
							}
						}

						if rootTypeFQN != "" {
							if ft, ok := callGraph.GoStructFieldIndex[rootTypeFQN+"."+fieldName]; ok {
								typeFQN = ft
							}
							// S4-Source4b: Stdlib struct field lookup (lazy, via CDN).
							// Covers stdlib types like net/http.Request.Header → net/http.Header.
							// Only runs when user-code struct index missed the field and the
							// root type comes from a known stdlib package.
							if typeFQN == "" && registry != nil && registry.StdlibLoader != nil {
								if pkgPath, typeName, ok := splitGoTypeFQN(rootTypeFQN); ok &&
									registry.StdlibLoader.ValidateStdlibImport(pkgPath) {
									if stdlibType, err := registry.StdlibLoader.GetType(pkgPath, typeName); err == nil && stdlibType != nil {
										for _, f := range stdlibType.Fields {
											if f.Name == fieldName {
												typeFQN = resolveFieldType(f.Type, pkgPath)
												// resolveFieldType may return a short-qualified
												// type like "url.URL" when the CDN stores the
												// field using the owner package's import alias.
												// Expand using the calling file's importMap first
												// (e.g., "url" → "net/url" if the file imports it).
												if typeFQN != "" && strings.Contains(typeFQN, ".") && !strings.Contains(typeFQN, "/") {
													if expanded := resolveGoTypeFQN(typeFQN, importMap); strings.Contains(expanded, "/") {
														typeFQN = expanded
													}
												}
												break
											}
										}
									}
								}
							}
						}
					}
				}
			}

			if typeFQN != "" {
				methodFQN := typeFQN + "." + callSite.FunctionName

				// Check 1: Method exists in user code
				if callGraph.Functions[methodFQN] != nil {
					return methodFQN, true, false, ""
				}

				// Check 2 (Approach C): Validate method via StdlibLoader
				if registry != nil && registry.StdlibLoader != nil {
					importPath, typeName, ok := splitGoTypeFQN(typeFQN)
					if ok && registry.StdlibLoader.ValidateStdlibImport(importPath) {
						stdlibType, err := registry.StdlibLoader.GetType(importPath, typeName)
						if err == nil && stdlibType != nil {
							if _, hasMethod := stdlibType.Methods[callSite.FunctionName]; hasMethod {
								return methodFQN, true, true, "" // resolved via stdlib
							}
						}
						// Check 2b: Method not found directly on the type — scan the same
						// package for an interface that declares it. This covers promoted
						// methods whose CDN entry does not list them on the concrete type
						// (e.g., testing.T.Fatalf is promoted from testing.common but
						// testing.TB.Fatalf is present; T implements TB).
						if ifaceFQN, found := findMethodInPackageInterfaces(
							registry.StdlibLoader, importPath, callSite.FunctionName,
						); found {
							return ifaceFQN, true, true, "" // resolved via stdlib interface
						}
					}
				}

				// Check 2.5: Validate method via ThirdPartyLoader (vendor/GOMODCACHE)
				if registry != nil && registry.ThirdPartyLoader != nil {
					importPath, typeName, ok := splitGoTypeFQN(typeFQN)
					if ok {
						// Skip if already checked as stdlib
						isStdlib := registry.StdlibLoader != nil &&
							registry.StdlibLoader.ValidateStdlibImport(importPath)
						if !isStdlib && registry.ThirdPartyLoader.ValidateImport(importPath) {
							tpType, err := registry.ThirdPartyLoader.GetType(importPath, typeName)
							if err == nil && tpType != nil {
								if _, hasMethod := tpType.Methods[callSite.FunctionName]; hasMethod {
									return methodFQN, true, false, "thirdparty_local" // resolved via third-party
								}
							}
						}
					}
				}

				// Check 3: Promoted method via struct embedding
				if promotedFQN, resolved, isStdlib := resolvePromotedMethod(
					typeFQN, callSite.FunctionName, registry,
				); resolved {
					return promotedFQN, true, isStdlib, ""
				}

				// Check 4: Unvalidated best-effort — only for verifiably complete FQNs.
				// typeFQN must contain "/" (a real multi-segment module path), or be
				// the built-in "error" interface, or a CGO type ("C.something").
				// Incomplete FQNs like "Chunk" or "blob.Chunk" are rejected here to
				// prevent false positives from low-confidence type bindings.
				if strings.Contains(typeFQN, "/") ||
					typeFQN == "error" ||
					strings.HasPrefix(typeFQN, "C.") {
					return methodFQN, true, false, ""
				}
			}
		}

		// Import not found and variable not found - unresolved
		return "", false, false, ""
	}

	// Pattern 2: Same-package call (simple function name)
	candidates := functionContext[callSite.FunctionName]
	for _, candidate := range candidates {
		// Check if candidate is in the same package as caller
		if isSameGoPackage(callSite.CallerFile, candidate.File) {
			// Build FQN for this candidate
			candidateFQN := buildGoFQN(candidate, nil, registry)
			return candidateFQN, true, false, ""
		}
	}

	// Pattern 3: Builtin function
	if isBuiltin(callSite.FunctionName) {
		return "builtin." + callSite.FunctionName, true, false, ""
	}

	// Pattern 4: Unresolved
	return "", false, false, ""
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
func buildGoFQN(node *graph.Node, parentMap map[string]*graph.Node, registry *core.GoModuleRegistry) string {
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
		// Find parent function using the pre-built parentMap
		parent := findParentGoFunction(node, parentMap)
		if parent != nil {
			parentFQN := buildGoFQN(parent, parentMap, registry)
			return parentFQN + "." + node.Name // Name is already "$anon_N" from PR-06
		}
		// Orphaned closure - shouldn't happen but handle gracefully
		return importPath + "." + node.Name

	default:
		return importPath + "." + node.Name
	}
}

// buildParentMap constructs a reverse-edge map (child ID → parent node) from the CodeGraph.
// Build this once and pass it to findContainingGoFunction / findParentGoFunction to avoid
// rebuilding it O(N) times inside loops over call nodes.
func buildParentMap(codeGraph *graph.CodeGraph) map[string]*graph.Node {
	parentMap := make(map[string]*graph.Node, len(codeGraph.Nodes))
	for _, node := range codeGraph.Nodes {
		for _, edge := range node.OutgoingEdges {
			parentMap[edge.To.ID] = node
		}
	}
	return parentMap
}

// buildPkgVarIndex builds a lookup table for package-level variables.
// Key: filepath.Dir(file) + "::" + varName
// Value: the module_variable node (only nodes with a non-empty DataType are included).
//
// This replaces the O(N) linear scan in resolveGoCallTarget Source 3 with an O(1) lookup.
func buildPkgVarIndex(codeGraph *graph.CodeGraph) map[string]*graph.Node {
	index := make(map[string]*graph.Node)
	for _, node := range codeGraph.Nodes {
		if node.Type != "module_variable" || node.DataType == "" {
			continue
		}
		key := filepath.Dir(node.File) + "::" + node.Name
		index[key] = node
	}
	return index
}

// buildStructFieldIndex builds a flat index of struct field → field type FQN for all
// struct_definition nodes in user code.
// Key:   "pkgPath.TypeName.FieldName"   (e.g. "myapp.models.Attention.KNorm")
// Value: resolved field type FQN         (e.g. "myapp.nn.Linear")
//
// Used by resolveGoCallTarget Source 4 to resolve chained field access: a.Field.Method().
func buildStructFieldIndex(codeGraph *graph.CodeGraph, registry *core.GoModuleRegistry, importMaps map[string]*core.GoImportMap) map[string]string {
	index := make(map[string]string)
	for _, node := range codeGraph.Nodes {
		if node.Type != "struct_definition" || node.Language != "go" || node.File == "" {
			continue
		}
		dirPath := filepath.Dir(node.File)
		pkgPath, ok := registry.DirToImport[dirPath]
		if !ok {
			continue
		}
		typeFQN := pkgPath + "." + node.Name
		importMap := importMaps[node.File]

		for _, field := range node.Interface {
			// Field format stored by parser: "FieldName: TypeStr"
			colonIdx := strings.Index(field, ": ")
			if colonIdx < 0 {
				continue // embedded type, skip
			}
			fieldName := field[:colonIdx]
			typeStr := strings.TrimPrefix(field[colonIdx+2:], "*")
			if typeStr == "" {
				continue
			}
			// Resolve to FQN via importMap
			fieldTypeFQN := resolveGoTypeFQN(typeStr, importMap)
			// Unqualified — same package
			if fieldTypeFQN == typeStr && !strings.Contains(fieldTypeFQN, ".") {
				fieldTypeFQN = pkgPath + "." + typeStr
			}
			if fieldTypeFQN != "" {
				index[typeFQN+"."+fieldName] = fieldTypeFQN
			}
		}
	}
	return index
}

// findContainingGoFunction finds the function/method/closure that contains a given call node.
// Walks parent edges using the pre-built parentMap to find the first function-like ancestor.
//
// parentMap must be built once via buildParentMap before iterating call nodes.
//
// Returns:
//   - Node pointer to the containing function, or nil if no containing function found
func findContainingGoFunction(callNode *graph.Node, parentMap map[string]*graph.Node) *graph.Node {
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
// parentMap must be pre-built via buildParentMap.
func findParentGoFunction(closureNode *graph.Node, parentMap map[string]*graph.Node) *graph.Node {
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

// goBuiltins is the set of Go builtin function names and predeclared type names
// that syntactically look like function calls (e.g. int(x), float64(x)).
// Allocated once at package init.
var goBuiltins = map[string]bool{
	// Builtin functions
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
	// Predeclared numeric/string types used as type-conversion expressions.
	// In Go, T(x) is syntactically identical to a call expression, so the
	// call-site extractor captures these as plain function calls.
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uintptr":    true,
	"float32":    true,
	"float64":    true,
	"complex64":  true,
	"complex128": true,
	"string":     true,
	"byte":       true,
	"rune":       true,
	"bool":       true,
	"error":      true,
}

// isBuiltin returns true if the function name is a Go builtin.
func isBuiltin(name string) bool {
	return goBuiltins[name]
}

// isSameGoPackage returns true if two file paths belong to the same Go package.
// In Go, a package is all files in the same directory.
func isSameGoPackage(file1, file2 string) bool {
	dir1 := filepath.Dir(file1)
	dir2 := filepath.Dir(file2)
	return dir1 == dir2
}

// buildCallSiteArguments converts CallSiteInternal.Arguments ([]string) to
// core.Argument structs with Value, IsVariable, and Position.
func buildCallSiteArguments(argNames []string) []core.Argument {
	if len(argNames) == 0 {
		return nil
	}
	args := make([]core.Argument, len(argNames))
	for i, name := range argNames {
		args[i] = core.Argument{
			Value:      name,
			IsVariable: !isGoLiteral(name),
			Position:   i,
		}
	}
	return args
}

// isGoLiteral checks if an argument value looks like a literal (not a variable).
func isGoLiteral(value string) bool {
	if value == "" {
		return true
	}
	// Quoted strings
	if (value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'') ||
		(value[0] == '`' && value[len(value)-1] == '`') {
		return true
	}
	// Numbers
	if value[0] >= '0' && value[0] <= '9' {
		return true
	}
	// Go keyword literals
	if value == "true" || value == "false" || value == "nil" {
		return true
	}
	return false
}

// resolvePromotedMethod checks if a method exists on an embedded type.
// Go struct embedding promotes all methods of the embedded type.
//
// Example: type MyHandler struct { *sql.DB }
//
//	MyHandler doesn't have Query(), but *sql.DB does (promoted).
//	h.Query(sql) → resolves to "database/sql.DB.Query"
func resolvePromotedMethod(
	typeFQN string,
	methodName string,
	registry *core.GoModuleRegistry,
) (string, bool, bool) {
	if registry == nil || registry.StdlibLoader == nil {
		return "", false, false
	}

	importPath, typeName, ok := splitGoTypeFQN(typeFQN)
	if !ok {
		return "", false, false
	}

	stdlibType, err := registry.StdlibLoader.GetType(importPath, typeName)
	if err != nil || stdlibType == nil {
		return "", false, false
	}

	return resolvePromotedMethodFromFields(stdlibType.Fields, methodName, registry)
}

// resolvePromotedMethodFromFields walks embedded struct fields to find promoted methods.
// Separated for testability without StdlibLoader.
func resolvePromotedMethodFromFields(
	fields []*core.GoStructField,
	methodName string,
	registry *core.GoModuleRegistry,
) (string, bool, bool) {
	for _, field := range fields {
		if field.Name != "" {
			continue // skip named fields, only check embedded (anonymous) fields
		}
		embeddedTypeFQN := strings.TrimPrefix(field.Type, "*")

		embImport, embType, ok := splitGoTypeFQN(embeddedTypeFQN)
		if !ok {
			continue
		}

		if registry != nil && registry.StdlibLoader != nil {
			embStdlibType, err := registry.StdlibLoader.GetType(embImport, embType)
			if err == nil && embStdlibType != nil {
				if _, hasMethod := embStdlibType.Methods[methodName]; hasMethod {
					return embeddedTypeFQN + "." + methodName, true, true
				}
			}
		}
	}

	return "", false, false
}
