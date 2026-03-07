package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
	cgregistry "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// getOptimalWorkerCount determines the optimal number of parallel workers.
// It balances performance with resource consumption to avoid overwhelming systems.
//
// Algorithm:
//  1. Start with available CPU cores
//  2. Leave 1-2 cores for OS/other processes
//  3. Cap at 16 workers (diminishing returns + memory concerns)
//  4. Minimum 2 workers (ensure some parallelism)
//  5. Respect PATHFINDER_MAX_WORKERS env var if set
//
// Returns:
//   - Number of workers to use (2-16)
func getOptimalWorkerCount() int {
	// Check for user override
	if envWorkers := os.Getenv("PATHFINDER_MAX_WORKERS"); envWorkers != "" {
		if count, err := strconv.Atoi(envWorkers); err == nil && count > 0 {
			// Respect user setting but cap at 32 for safety
			if count > 32 {
				count = 32
			}
			return count
		}
	}

	// Get available CPU cores
	cpuCount := runtime.NumCPU()

	// Conservative approach: use 75% of cores, leave some for OS
	workers := min(
		// Apply bounds
		// Minimum parallelism
		max(

			int(float64(cpuCount)*0.75),

			2),
		// Cap at 16 (memory/connection limits)
		16)

	return workers
}

// BuildCallGraph constructs the complete call graph for a Python project.
// This is Pass 3 of the 3-pass algorithm:
//   - Pass 1: BuildModuleRegistry - map files to modules
//   - Pass 2: ExtractImports + ExtractCallSites - parse imports and calls
//   - Pass 3: BuildCallGraph - resolve calls and build graph
//
// Algorithm:
//  1. For each Python file in the project:
//     a. Extract imports to build ImportMap
//     b. Extract call sites from AST
//     c. Extract function definitions from main graph
//  2. For each call site:
//     a. Resolve target name using ImportMap
//     b. Find target function definition in registry
//     c. Add edge from caller to callee
//     d. Store detailed call site information
//
// Parameters:
//   - codeGraph: the existing code graph with parsed AST nodes
//   - registry: module registry mapping files to modules
//   - projectRoot: absolute path to project root
//
// Returns:
//   - CallGraph: complete call graph with edges and call sites
//   - error: if any step fails
//
// Example:
//
//	Given:
//	  File: myapp/views.py
//	    def get_user():
//	        sanitize(data)  # call to myapp.utils.sanitize
//
//	Creates:
//	  edges: {"myapp.views.get_user": ["myapp.utils.sanitize"]}
//	  reverseEdges: {"myapp.utils.sanitize": ["myapp.views.get_user"]}
//	  callSites: {"myapp.views.get_user": [CallSite{Target: "sanitize", ...}]}
func BuildCallGraph(codeGraph *graph.CodeGraph, registry *core.ModuleRegistry, projectRoot string, logger *output.Logger) (*core.CallGraph, error) {
	callGraph := core.NewCallGraph()

	// Initialize import map cache for performance
	// This avoids re-parsing imports from the same file multiple times
	importCache := NewImportMapCache()

	// Initialize type inference engine
	typeEngine := resolution.NewTypeInferenceEngine(registry)
	typeEngine.Builtins = cgregistry.NewBuiltinRegistry()

	// Phase 3 Task 12: Initialize attribute registry for tracking class attributes
	typeEngine.Attributes = cgregistry.NewAttributeRegistry()

	// PR #3: Detect Python version and load stdlib registry from remote CDN
	pythonVersion := DetectPythonVersion(projectRoot)
	logger.Debug("Detected Python version: %s", pythonVersion)

	// Create remote registry loader
	remoteLoader := cgregistry.NewStdlibRegistryRemote(
		"https://assets.codepathfinder.dev/registries",
		pythonVersion,
	)

	// Load manifest from CDN
	err := remoteLoader.LoadManifest(logger)
	if err != nil {
		logger.Warning("Failed to load stdlib registry from CDN: %v", err)
		// Continue without stdlib resolution - not a fatal error
	} else {
		// Create adapter to satisfy existing StdlibRegistry interface
		stdlibRegistry := &core.StdlibRegistry{
			Modules:  make(map[string]*core.StdlibModule),
			Manifest: remoteLoader.Manifest,
		}

		// The remote loader will lazy-load modules as needed
		// We store a reference to it for on-demand loading
		typeEngine.StdlibRegistry = stdlibRegistry
		typeEngine.StdlibRemote = remoteLoader

		logger.Statistic("Loaded stdlib manifest from CDN: %d modules available", remoteLoader.ModuleCount())
	}

	// PR #4: Load third-party type registry from CDN (non-fatal on failure)
	thirdPartyLoader := cgregistry.NewThirdPartyRegistryRemote(
		"https://assets.codepathfinder.dev/registries",
	)
	if err := thirdPartyLoader.LoadManifest(logger); err != nil {
		logger.Warning("Failed to load third-party registry: %v", err)
	} else {
		typeEngine.ThirdPartyRemote = thirdPartyLoader
		logger.Statistic("Third-party manifest: %d packages available", thirdPartyLoader.ModuleCount())
	}

	// Phase 1: Build class context map for class-qualified FQN generation
	// This maps file locations to class names, allowing us to determine
	// which class a method belongs to based on its byte range.
	// We build this once and reuse it throughout call graph construction.
	classContext := buildClassContext(codeGraph)

	// First, index all function definitions from the code graph
	// This builds the Functions map for quick lookup
	indexFunctions(codeGraph, callGraph, registry)

	// Index typed parameters as standalone symbols from the indexed functions
	indexParameters(callGraph)

	// Phase 2 Task 9: Extract return types from all functions (first pass - PARALLELIZED)
	logger.Debug("Extracting return types from %d modules (parallel)...", len(registry.Modules))

	type returnJob struct {
		modulePath string
		filePath   string
	}

	returnJobs := make(chan returnJob, 100)
	var returnMutex sync.Mutex
	allReturnStatements := make([]*resolution.ReturnStatement, 0)
	allFunctionsWithReturnValues := make(map[string]bool)
	var processedFiles atomic.Int64
	numWorkers := getOptimalWorkerCount()
	var wg sync.WaitGroup

	logger.Debug("Using %d parallel workers for callgraph construction", numWorkers)

	// Start workers for return type extraction
	for range numWorkers {
		wg.Go(func() {
			for job := range returnJobs {
				sourceCode, err := ReadFileBytes(job.filePath)
				if err != nil {
					continue
				}

				// Extract imports using cache (needed for class instantiation resolution)
				importMap, err := importCache.GetOrExtract(job.filePath, sourceCode, registry)
				if err != nil {
					continue
				}

				// Store ImportMap for later use in attribute placeholder resolution (P0 fix)
				typeEngine.AddImportMap(job.filePath, importMap)

				returns, functionsWithReturns, err := resolution.ExtractReturnTypes(job.filePath, sourceCode, job.modulePath, typeEngine.Builtins, importMap)
				if err != nil {
					continue
				}

				returnMutex.Lock()
				if len(returns) > 0 {
					allReturnStatements = append(allReturnStatements, returns...)
				}
				for fqn := range functionsWithReturns {
					allFunctionsWithReturnValues[fqn] = true
				}
				returnMutex.Unlock()

				// Progress tracking
				count := processedFiles.Add(1)
				if count%1000 == 0 {
					logger.Debug("Processed %d/%d files for return types", count, len(registry.Modules))
				}
			}
		})
	}

	// Queue all Python files
	for modulePath, filePath := range registry.Modules {
		if !strings.HasSuffix(filePath, ".py") {
			continue
		}
		returnJobs <- returnJob{modulePath, filePath}
	}
	close(returnJobs)
	wg.Wait()

	logger.Debug("Completed return type extraction: %d files processed", processedFiles.Load())

	// Merge return types and add to engine
	mergedReturns := resolution.MergeReturnTypes(allReturnStatements)
	typeEngine.AddReturnTypesToEngine(mergedReturns)

	// Back-populate inferred return types to function nodes and detect void functions
	populateInferredReturnTypes(callGraph, typeEngine, allFunctionsWithReturnValues, logger)

	// PR #5: Pre-load third-party modules that appear in project imports.
	// ImportMaps are populated during return type extraction above.
	// Pre-fetching here avoids per-call-site CDN downloads during Pass 4.
	preloadThirdPartyModules(typeEngine, logger)

	// Phase 2 Task 8: Extract ALL variable assignments BEFORE resolving calls (second pass - PARALLELIZED)
	logger.Debug("Extracting variable assignments (parallel)...")

	varJobs := make(chan string, 100)
	var varProcessed atomic.Int64
	wg = sync.WaitGroup{}

	// Start workers for variable assignment extraction
	for range numWorkers {
		wg.Go(func() {
			for filePath := range varJobs {
				sourceCode, err := ReadFileBytes(filePath)
				if err != nil {
					continue
				}

				// Extract imports using cache (needed for class instantiation resolution)
				importMap, err := importCache.GetOrExtract(filePath, sourceCode, registry)
				if err != nil {
					continue
				}

				// Store ImportMap for later use in attribute placeholder resolution (P0 fix)
				typeEngine.AddImportMap(filePath, importMap)

				// Extract variable assignments - typeEngine methods are mutex-protected internally
				// Class context is tracked during AST traversal to build class-qualified FQNs (matching Pass 1)
				_ = extraction.ExtractVariableAssignments(filePath, sourceCode, typeEngine, registry, typeEngine.Builtins, importMap)

				// Progress tracking
				count := varProcessed.Add(1)
				if count%1000 == 0 {
					logger.Debug("Processed %d files for variable assignments", count)
				}
			}
		})
	}

	// Queue all Python files
	for _, filePath := range registry.Modules {
		if !strings.HasSuffix(filePath, ".py") {
			continue
		}
		varJobs <- filePath
	}
	close(varJobs)
	wg.Wait()

	logger.Debug("Completed variable assignment extraction: %d files processed", varProcessed.Load())

	// Resolve var: placeholders in return types using scope variable lookups.
	// Must happen AFTER variable extraction (scopes populated) and BEFORE call: resolution.
	typeEngine.ResolveReturnVariableReferences()

	// Phase 2 Task 8: Resolve call: placeholders with return types
	// This MUST happen before we start resolving call sites!
	typeEngine.UpdateVariableBindingsWithFunctionReturns()

	// PR #6: Resolve remaining call: placeholders using third-party type registry
	// Must run AFTER UpdateVariableBindingsWithFunctionReturns (userland first).
	resolveThirdPartyVariableBindings(typeEngine, logger)

	// Phase 3 Task 12: Extract class attributes (third pass - PARALLELIZED)
	logger.Debug("Extracting class attributes (parallel)...")

	attrJobs := make(chan returnJob, 100) // Reuse returnJob struct
	var attrProcessed atomic.Int64
	wg = sync.WaitGroup{}

	// Start workers for class attribute extraction
	for range numWorkers {
		wg.Go(func() {
			for job := range attrJobs {
				sourceCode, err := ReadFileBytes(job.filePath)
				if err != nil {
					continue
				}

				// Extract class attributes - AttributeRegistry methods are mutex-protected
				_ = extraction.ExtractClassAttributes(job.filePath, sourceCode, job.modulePath, typeEngine, typeEngine.Attributes)

				// Progress tracking
				count := attrProcessed.Add(1)
				if count%1000 == 0 {
					logger.Debug("Processed %d files for class attributes", count)
				}
			}
		})
	}

	// Queue all Python files
	for modulePath, filePath := range registry.Modules {
		if !strings.HasSuffix(filePath, ".py") {
			continue
		}
		attrJobs <- returnJob{modulePath, filePath}
	}
	close(attrJobs)
	wg.Wait()

	logger.Debug("Completed class attribute extraction: %d files processed", attrProcessed.Load())

	// Phase 3 Task 12: Resolve placeholder types in attributes (Pass 3)
	resolution.ResolveAttributePlaceholders(typeEngine.Attributes, typeEngine, registry, codeGraph)

	// PR #7: Resolve parent classes and propagate inherited parameter types
	resolveParentClassInheritance(codeGraph, callGraph, registry, typeEngine, logger)

	// Process each Python file in the project (fourth pass for call site resolution - PARALLELIZED)
	logger.Debug("Resolving call sites (parallel)...")

	callSiteJobs := make(chan returnJob, 100)
	var callGraphMutex sync.Mutex // Protect callGraph modifications
	var callSiteProcessed atomic.Int64
	wg = sync.WaitGroup{}

	// Start workers for call site resolution
	for range numWorkers {
		wg.Go(func() {
			for job := range callSiteJobs {
				// Read source code for parsing
				sourceCode, err := ReadFileBytes(job.filePath)
				if err != nil {
					continue
				}

				// Extract imports using cache (cache is thread-safe)
				importMap, err := importCache.GetOrExtract(job.filePath, sourceCode, registry)
				if err != nil {
					continue
				}

				// Store ImportMap for later use in attribute placeholder resolution (P0 fix)
				typeEngine.AddImportMap(job.filePath, importMap)

				// Extract all call sites from this file
				callSites, err := resolution.ExtractCallSites(job.filePath, sourceCode, importMap)
				if err != nil {
					continue
				}

				// Get all function definitions in this file
				fileFunctions := getFunctionsInFile(codeGraph, job.filePath)

				// Process each call site to resolve targets and build edges
				for _, callSite := range callSites {
					// Phase 1: Find the caller function containing this call site
					// Now with class context for class-qualified FQNs
					callerFQN := findContainingFunction(callSite.Location, fileFunctions, job.modulePath, classContext)
					if callerFQN == "" {
						callerFQN = job.modulePath
					}

					// Resolve the call target to a fully qualified name
					targetFQN, resolved, typeInfo := resolveCallTarget(callSite.Target, importMap, registry, job.modulePath, codeGraph, typeEngine, callerFQN, callGraph, logger)

					// Update call site with resolution information
					callSite.TargetFQN = targetFQN
					callSite.Resolved = resolved

					// Phase 2 Task 10: Populate type inference metadata
					if typeInfo != nil {
						callSite.ResolvedViaTypeInference = true
						callSite.InferredType = typeInfo.TypeFQN
						callSite.TypeConfidence = typeInfo.Confidence
						callSite.TypeSource = typeInfo.Source
					}

					// If resolution failed, categorize the failure reason
					if !resolved {
						callSite.FailureReason = categorizeResolutionFailure(callSite.Target, targetFQN, typeEngine)
					}

					// CRITICAL: Lock callGraph modifications (shared state)
					callGraphMutex.Lock()
					callGraph.AddCallSite(callerFQN, *callSite)
					if resolved {
						callGraph.AddEdge(callerFQN, targetFQN)
					}
					callGraphMutex.Unlock()
				}

				// Progress tracking
				count := callSiteProcessed.Add(1)
				if count%1000 == 0 {
					logger.Debug("Processed %d files for call sites", count)
				}
			}
		})
	}

	// Queue all Python files
	for modulePath, filePath := range registry.Modules {
		if !strings.HasSuffix(filePath, ".py") {
			continue
		}
		callSiteJobs <- returnJob{modulePath, filePath}
	}
	close(callSiteJobs)
	wg.Wait()

	logger.Debug("Completed call site resolution: %d files processed", callSiteProcessed.Load())

	// Phase 3 Task 12: Print attribute failure analysis (debug mode only)
	resolution.PrintAttributeFailureStats(logger)

	// Pass 5: Generate taint summaries for all functions
	logger.Debug("Generating taint summaries...")
	GenerateTaintSummaries(callGraph, codeGraph, registry)
	logger.Statistic("Generated taint summaries for %d functions", len(callGraph.Summaries))

	// Store attribute registry for symbol search and type inference
	callGraph.Attributes = typeEngine.Attributes

	// Store type engine for module variable type lookups in MCP tools
	callGraph.TypeEngine = typeEngine

	return callGraph, nil
}

// preloadThirdPartyModules scans all collected ImportMaps and pre-fetches
// third-party modules that appear in project imports. This avoids per-call-site
// CDN downloads during call resolution (Pass 4).
func preloadThirdPartyModules(typeEngine *resolution.TypeInferenceEngine, logger *output.Logger) {
	if typeEngine.ThirdPartyRemote == nil {
		return
	}

	loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote)
	if !ok || loader == nil {
		return
	}

	// Collect unique top-level module names from all import maps
	seen := make(map[string]bool)

	typeEngine.ForEachImportMap(func(_ string, importMap *core.ImportMap) {
		for _, fqn := range importMap.Imports {
			// Extract top-level package name (e.g., "requests" from "requests.sessions")
			topLevel := fqn
			if idx := strings.Index(fqn, "."); idx > 0 {
				topLevel = fqn[:idx]
			}
			if !seen[topLevel] && loader.HasModule(topLevel) {
				seen[topLevel] = true
			}
		}
	})

	if len(seen) == 0 {
		return
	}

	// Pre-fetch each matched module
	var loaded int
	for moduleName := range seen {
		if _, err := loader.GetModule(moduleName, logger); err != nil {
			logger.Warning("Failed to pre-load third-party module %s: %v", moduleName, err)
		} else {
			loaded++
		}
	}

	if loaded > 0 {
		logger.Statistic("Pre-loaded %d third-party modules from typeshed", loaded)
	}
}

// IndexFunctions builds the Functions map in the call graph.
// Extracts all function definitions from the code graph and maps them by FQN.
//
// Parameters:
//   - codeGraph: the parsed code graph
//   - callGraph: the call graph being built
//   - registry: module registry for resolving file paths to modules
func IndexFunctions(codeGraph *graph.CodeGraph, callGraph *core.CallGraph, registry *core.ModuleRegistry) {
	indexFunctions(codeGraph, callGraph, registry)
}

// indexFunctions is the internal implementation of IndexFunctions.
func indexFunctions(codeGraph *graph.CodeGraph, callGraph *core.CallGraph, registry *core.ModuleRegistry) {
	// First pass: Build class context map (file+line → class name)
	classContext := buildClassContext(codeGraph)

	for _, node := range codeGraph.Nodes {
		// Only index function/method definitions (Java and Python types)
		// Java: method_declaration
		// Python: function_definition, method, constructor, property, special_method
		if node.Type != "method_declaration" && node.Type != "function_definition" &&
			node.Type != "method" && node.Type != "constructor" &&
			node.Type != "property" && node.Type != "special_method" {
			continue
		}

		// Get the module path for this function's file
		modulePath, ok := registry.FileToModule[node.File]
		if !ok {
			continue
		}

		// Build fully qualified name with class context if applicable
		fqn := buildFQN(modulePath, node, classContext)
		callGraph.Functions[fqn] = node
	}
}

// IndexParameters extracts typed parameters from indexed functions and stores them
// as standalone symbols in callGraph.Parameters. This is the exported wrapper.
func IndexParameters(callGraph *core.CallGraph) {
	indexParameters(callGraph)
}

// indexParameters iterates all indexed functions and extracts their typed parameters
// into the Parameters map. Each parameter with a type annotation becomes a ParameterSymbol
// keyed by its FQN (e.g., "myapp.auth.validate_user.username").
//
// Parameters named "self" and "cls" are skipped as they don't carry useful type information.
func indexParameters(callGraph *core.CallGraph) {
	for fqn, node := range callGraph.Functions {
		if len(node.MethodArgumentsType) == 0 {
			continue
		}
		for _, paramStr := range node.MethodArgumentsType {
			parts := strings.SplitN(paramStr, ": ", 2)
			if len(parts) != 2 {
				continue
			}
			paramName := parts[0]
			paramType := parts[1]

			// Skip self and cls parameters — they don't carry useful type information.
			if paramName == "self" || paramName == "cls" {
				continue
			}

			paramFQN := fqn + "." + paramName
			callGraph.Parameters[paramFQN] = &core.ParameterSymbol{
				Name:           paramName,
				TypeAnnotation: paramType,
				ParentFQN:      fqn,
				File:           node.File,
				Line:           node.LineNumber,
			}
		}
	}
}

// buildClassContext creates a map of file locations to class names.
// This allows us to determine which class a method belongs to based on its location.
func buildClassContext(codeGraph *graph.CodeGraph) map[string]string {
	classCtx := make(map[string]string)

	// Find all class definitions
	for _, node := range codeGraph.Nodes {
		if node.Type == "class_definition" || node.Type == "interface" ||
			node.Type == "enum" || node.Type == "dataclass" {
			// For each class, we need to know its byte range
			// Methods/constructors within this range belong to this class
			if node.SourceLocation != nil {
				// Store class name by file + start/end bytes
				key := fmt.Sprintf("%s:%d:%d", node.File, node.SourceLocation.StartByte, node.SourceLocation.EndByte)
				classCtx[key] = node.Name
			}
		}
	}

	return classCtx
}

// buildFQN constructs the fully qualified name for a function/method node.
// For methods: module.ClassName.methodName
// For functions: module.functionName.
func buildFQN(modulePath string, node *graph.Node, classContext map[string]string) string {
	// For methods/constructors/properties, try to find the containing class
	if node.Type == "method" || node.Type == "constructor" ||
		node.Type == "property" || node.Type == "special_method" {
		// Find which class this method belongs to
		className := findContainingClass(node, classContext)
		if className != "" {
			return fmt.Sprintf("%s.%s.%s", modulePath, className, node.Name)
		}
	}

	// For top-level functions or if class not found, use simple FQN
	return modulePath + "." + node.Name
}

// findContainingClass determines which class a node belongs to based on its location.
func findContainingClass(node *graph.Node, classContext map[string]string) string {
	if node.SourceLocation == nil {
		return ""
	}

	// Check if this node is within any class's byte range
	for key, className := range classContext {
		// Parse key format: "/path/to/file.py:startByte:endByte"
		// Use strings.LastIndex to find the last two colons (for byte ranges)
		lastColon := strings.LastIndex(key, ":")
		if lastColon == -1 {
			continue
		}
		secondLastColon := strings.LastIndex(key[:lastColon], ":")
		if secondLastColon == -1 {
			continue
		}

		// Extract components
		file := key[:secondLastColon]
		classStartStr := key[secondLastColon+1 : lastColon]
		classEndStr := key[lastColon+1:]

		// Parse byte positions
		classStart, err1 := strconv.ParseUint(classStartStr, 10, 32)
		classEnd, err2 := strconv.ParseUint(classEndStr, 10, 32)
		if err1 != nil || err2 != nil {
			continue
		}

		// Check if node is within this class's byte range
		if file == node.File &&
			node.SourceLocation.StartByte >= uint32(classStart) &&
			node.SourceLocation.EndByte <= uint32(classEnd) {
			return className
		}
	}

	return ""
}

// GetFunctionsInFile returns all function definitions in a specific file.
//
// Parameters:
//   - codeGraph: the parsed code graph
//   - filePath: absolute path to the file
//
// Returns:
//   - List of function/method nodes in the file, sorted by line number
func GetFunctionsInFile(codeGraph *graph.CodeGraph, filePath string) []*graph.Node {
	return getFunctionsInFile(codeGraph, filePath)
}

// getFunctionsInFile is the internal implementation of GetFunctionsInFile.
func getFunctionsInFile(codeGraph *graph.CodeGraph, filePath string) []*graph.Node {
	var functions []*graph.Node

	for _, node := range codeGraph.Nodes {
		if node.File == filePath &&
			(node.Type == "method_declaration" || node.Type == "function_definition" ||
				node.Type == "method" || node.Type == "constructor" ||
				node.Type == "property" || node.Type == "special_method") {
			functions = append(functions, node)
		}
	}

	return functions
}

// FindContainingFunction finds the function that contains a given call site location.
// Uses line numbers to determine which function a call belongs to.
//
// Algorithm:
//  1. Iterate through all functions in the file
//  2. Find function with the highest line number that's still <= call line
//  3. Return the FQN of that function (class-qualified for methods)
//
// Parameters:
//   - location: source location of the call site
//   - functions: all function definitions in the file
//   - modulePath: module path of the file
//   - classContext: map of file locations to class names (for class-qualified FQNs)
//
// Returns:
//   - Fully qualified name of the containing function, or empty if not found
//
// Examples:
//   - Module-level function: "myapp.process"
//   - Instance method: "myapp.User.save"
//   - Nested class method: "myapp.Outer.Inner.method"
func FindContainingFunction(location core.Location, functions []*graph.Node, modulePath string, classContext map[string]string) string {
	return findContainingFunction(location, functions, modulePath, classContext)
}

// findContainingFunction is the internal implementation of FindContainingFunction.
func findContainingFunction(location core.Location, functions []*graph.Node, modulePath string, classContext map[string]string) string {
	// In Python, module-level code has no indentation (column == 1)
	// If the call site is at column 1, it's module-level, not inside any function
	if location.Column == 1 {
		return ""
	}

	var bestMatch *graph.Node
	var bestLine uint32

	for _, fn := range functions {
		// Check if call site is after this function definition
		if uint32(location.Line) >= fn.LineNumber {
			// Keep track of the closest preceding function
			if bestMatch == nil || fn.LineNumber > bestLine {
				bestMatch = fn
				bestLine = fn.LineNumber
			}
		}
	}

	if bestMatch != nil {
		// Phase 1: Build class-qualified FQN for methods
		// For methods/constructors/properties/special_methods, include class name
		if bestMatch.Type == "method" || bestMatch.Type == "constructor" ||
			bestMatch.Type == "property" || bestMatch.Type == "special_method" {
			className := findContainingClass(bestMatch, classContext)
			if className != "" {
				return fmt.Sprintf("%s.%s.%s", modulePath, className, bestMatch.Name)
			}
		}

		// For module-level functions or if class not found
		return modulePath + "." + bestMatch.Name
	}

	return ""
}

// categorizeResolutionFailure determines why a call target failed to resolve.
// This enables diagnostic reporting to understand resolution gaps.
//
// Categories:
//   - "external_framework" - Known external frameworks (Django, REST, pytest, stdlib)
//   - "orm_pattern" - Django ORM patterns (Model.objects.*, queryset.*)
//   - "attribute_chain" - Method calls on objects/return values
//   - "variable_method" - Method calls that appear to be on variables
//   - "super_call" - Calls via super() mechanism
//   - "not_in_imports" - Simple name not found in imports
//   - "unknown" - Other unresolved patterns
//
// Parameters:
//   - target: original call target string (e.g., "models.ForeignKey")
//   - targetFQN: resolved fully qualified name (e.g., "django.db.models.ForeignKey")
//
// Returns:
//   - category string describing the failure reason
func categorizeResolutionFailure(target, targetFQN string, typeEngine *resolution.TypeInferenceEngine) string {
	// Check third-party registry dynamically (no hardcoded framework lists)
	if typeEngine != nil && strings.Contains(targetFQN, ".") {
		moduleName, _ := splitModuleAndName(targetFQN)
		if typeEngine.ThirdPartyRemote != nil {
			if loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote); ok {
				if loader.HasModule(moduleName) {
					return "external_framework"
				}
			}
		}
		// Check stdlib registry dynamically
		if typeEngine.StdlibRemote != nil {
			if loader, ok := typeEngine.StdlibRemote.(*cgregistry.StdlibRegistryRemote); ok {
				if loader.HasModule(moduleName) {
					return "stdlib_unresolved"
				}
			}
		}
	}

	// Check for Django ORM patterns
	if strings.Contains(target, ".objects.") ||
		strings.HasSuffix(target, ".objects") ||
		(strings.Contains(target, ".") && (strings.HasSuffix(target, ".filter") ||
			strings.HasSuffix(target, ".get") ||
			strings.HasSuffix(target, ".create") ||
			strings.HasSuffix(target, ".update") ||
			strings.HasSuffix(target, ".delete") ||
			strings.HasSuffix(target, ".all") ||
			strings.HasSuffix(target, ".first") ||
			strings.HasSuffix(target, ".last") ||
			strings.HasSuffix(target, ".count") ||
			strings.HasSuffix(target, ".exists"))) {
		return "orm_pattern"
	}

	// Check for super() calls
	if strings.HasPrefix(target, "super(") || strings.HasPrefix(target, "super.") {
		return "super_call"
	}

	// Check for attribute chains (has dots, looks like obj.method())
	// Heuristic: lowercase first component likely means variable/object
	if before, _, ok := strings.Cut(target, "."); ok {
		firstComponent := before
		// If starts with lowercase and not a known module pattern, likely attribute chain
		if len(firstComponent) > 0 && firstComponent[0] >= 'a' && firstComponent[0] <= 'z' {
			// Could be variable method or attribute chain
			// Check common variable-like patterns
			if firstComponent == "self" || firstComponent == "cls" ||
				firstComponent == "request" || firstComponent == "response" ||
				firstComponent == "queryset" || firstComponent == "user" ||
				firstComponent == "obj" || firstComponent == "value" ||
				firstComponent == "data" || firstComponent == "result" {
				return "variable_method"
			}
			return "attribute_chain"
		}
	}

	// Simple name (no dots) - not in imports
	if !strings.Contains(target, ".") {
		return "not_in_imports"
	}

	// Everything else
	return "unknown"
}

// Python built-in functions that should not be resolved as module functions.
var pythonBuiltins = map[string]bool{
	"eval":       true,
	"exec":       true,
	"input":      true,
	"raw_input":  true,
	"compile":    true,
	"__import__": true,
}

// ResolveCallTarget resolves a call target name to a fully qualified name.
// This is the core resolution logic that handles:
//   - Direct function calls: sanitize() → myapp.utils.sanitize
//   - Method calls: obj.method() → (unresolved, needs type inference)
//   - Imported functions: from utils import sanitize; sanitize() → myapp.utils.sanitize
//   - Qualified calls: utils.sanitize() → myapp.utils.sanitize
//
// Algorithm:
//  1. Check if target is a simple name (no dots)
//     a. Look up in import map
//     b. If found, return FQN from import
//     c. If not found, try to find in same module
//  2. If target has dots (qualified name)
//     a. Split into base and rest
//     b. Resolve base using import map
//     c. Append rest to get full FQN
//  3. If all else fails, check if it exists in the registry
//
// Parameters:
//   - target: the call target name (e.g., "sanitize", "utils.sanitize", "obj.method")
//   - importMap: import mappings for the current file
//   - registry: module registry for validation
//   - currentModule: the module containing this call
//   - codeGraph: the parsed code graph for validation
//   - typeEngine: type inference engine
//   - callerFQN: fully qualified name of the calling function
//   - callGraph: the call graph being built
//
// Returns:
//   - Fully qualified name of the target
//   - Boolean indicating if resolution was successful
//   - TypeInfo if resolved via type inference
//
// Examples:
//
//	target="sanitize", imports={"sanitize": "myapp.utils.sanitize"}
//	  → "myapp.utils.sanitize", true, nil
//
//	target="utils.sanitize", imports={"utils": "myapp.utils"}
//	  → "myapp.utils.sanitize", true, nil
//
//	target="obj.method", imports={}
//	  → "obj.method", false, nil  (needs type inference)
func ResolveCallTarget(target string, importMap *core.ImportMap, registry *core.ModuleRegistry, currentModule string, codeGraph *graph.CodeGraph, typeEngine *resolution.TypeInferenceEngine, callerFQN string, callGraph *core.CallGraph, logger *output.Logger) (string, bool, *core.TypeInfo) {
	return resolveCallTarget(target, importMap, registry, currentModule, codeGraph, typeEngine, callerFQN, callGraph, logger)
}

// resolveCallTarget is the internal implementation of ResolveCallTarget.
func resolveCallTarget(target string, importMap *core.ImportMap, registry *core.ModuleRegistry, currentModule string, codeGraph *graph.CodeGraph, typeEngine *resolution.TypeInferenceEngine, callerFQN string, callGraph *core.CallGraph, logger *output.Logger) (string, bool, *core.TypeInfo) {
	// Backward compatibility: if typeEngine or callerFQN not provided, skip type inference
	if typeEngine == nil || callerFQN == "" {
		fqn, resolved := resolveCallTargetLegacy(target, importMap, registry, currentModule, codeGraph)
		return fqn, resolved, nil
	}

	// Phase 3 Task 11: Check for method chaining BEFORE other resolution
	// Chains have pattern "()." indicating call followed by attribute access
	if strings.Contains(target, ").") {
		chainFQN, chainResolved, chainType := resolution.ResolveChainedCall(
			target,
			typeEngine,
			typeEngine.Builtins,
			registry,
			codeGraph,
			callerFQN,
			currentModule,
			callGraph,
		)
		if chainResolved {
			return chainFQN, true, chainType
		}
		// Chain parsing attempted but failed - fall through to regular resolution
	}

	// Phase 3 Task 12: Check for self.attribute.method() patterns BEFORE self.method()
	// Pattern: self.attr.method (2+ dots starting with self.)
	if strings.HasPrefix(target, "self.") && strings.Count(target, ".") >= 2 {
		attrFQN, attrResolved, attrType := resolution.ResolveSelfAttributeCall(
			target,
			callerFQN,
			typeEngine,
			typeEngine.Builtins,
			callGraph,
		)
		if attrResolved {
			return attrFQN, true, attrType
		}

		// PR #7: Fallback — check parent class attributes via third-party registry
		// For self.attr.method where attr isn't in child class, try parent classes
		if typeEngine != nil && typeEngine.ThirdPartyRemote != nil && codeGraph != nil {
			attrParts := strings.Split(target, ".")
			if len(attrParts) == 3 {
				attrName := attrParts[1]
				methodOnAttr := attrParts[2]
				callerParts := strings.Split(callerFQN, ".")
				if len(callerParts) >= 3 {
					callerClassName := callerParts[len(callerParts)-2]
					callerClassFQN := currentModule + "." + callerClassName
					fqn, resolved, typeInfo := resolveInheritedSelfAttrMethod(
						callerClassFQN, attrName, methodOnAttr,
						codeGraph, registry, typeEngine, logger,
					)
					if resolved {
						return fqn, true, typeInfo
					}
				}
			}
		}
		// Attribute resolution attempted but failed - fall through
	}

	// Phase 3: Handle super().method() calls - resolve to parent class method
	if after, ok := strings.CutPrefix(target, "super()."); ok {
		methodName := after

		// Extract current class name from callerFQN
		// callerFQN format: "module.ClassName.methodName"
		parts := strings.Split(callerFQN, ".")

		if len(parts) >= 3 {
			// Current class info
			className := parts[len(parts)-2]
			currentClassFQN := currentModule + "." + className

			// PR #7: Find the class node and resolve parent via imports + third-party
			if codeGraph != nil {
				for _, node := range codeGraph.Nodes {
					if node.Type != "class_definition" && node.Type != "dataclass" {
						continue
					}
					if node.File == "" || len(node.Interface) == 0 {
						continue
					}
					modulePath, ok := registry.FileToModule[node.File]
					if !ok {
						continue
					}
					nodeFQN := modulePath + "." + node.Name
					if nodeFQN != currentClassFQN {
						continue
					}

					// Found the class — resolve each parent and check for the method
					for _, superClassName := range node.Interface {
						parentFQN := resolution.ResolveParentClassFQN(
							currentClassFQN, superClassName, node.File,
							typeEngine, registry,
						)
						if parentFQN == "" {
							continue
						}

						// Check userland first
						parentMethodFQN := parentFQN + "." + methodName
						if callGraph != nil && callGraph.Functions[parentMethodFQN] != nil {
							return parentMethodFQN, true, nil
						}

						// Check third-party registry
						if typeEngine != nil && typeEngine.ThirdPartyRemote != nil {
							if loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote); ok {
								tpModule, tpClass := splitModuleAndName(parentFQN)
								if tpClass != "" && loader.HasModule(tpModule) {
									method := findThirdPartyClassMethod(loader, tpModule, tpClass, methodName, logger)
									if method != nil {
										return parentFQN + "." + methodName, true, nil
									}
								}
							}
						}
					}
					break
				}
			}

			// Heuristic fallback: Try common parent class names (Base suffix)
			if callGraph != nil {
				parentMethodFQN := currentModule + "." + className + "Base." + methodName
				if callGraph.Functions[parentMethodFQN] != nil {
					return parentMethodFQN, true, nil
				}
			}

			// Try module-level function as last resort
			moduleFQN := currentModule + "." + methodName
			if callGraph != nil && callGraph.Functions[moduleFQN] != nil {
				return moduleFQN, true, nil
			}

			// Return unresolved with descriptive FQN
			return currentModule + ".super()." + methodName, false, nil
		}

		// Can't extract class from callerFQN
		return "super()." + methodName, false, nil
	}

	// Phase 2: Handle self.method() calls - resolve to current class method
	if after, ok := strings.CutPrefix(target, "self."); ok {
		methodName := after

		// Phase 2: Extract class name from callerFQN for class-qualified lookup
		// callerFQN format: "module.ClassName.methodName" for methods
		//                   "module.functionName" for module-level functions
		parts := strings.Split(callerFQN, ".")

		// If callerFQN has 3+ parts, it's a class method
		// parts = ["module", "ClassName", "methodName"] or more for nested modules
		if len(parts) >= 3 {
			// Extract class name (second-to-last part)
			// For "module.ClassName.methodName" → className = "ClassName"
			// For "app.models.User.save" → className = "User"
			className := parts[len(parts)-2]

			// Build class-qualified FQN: module.ClassName.methodName
			classQualifiedFQN := currentModule + "." + className + "." + methodName

			// Try class-qualified lookup first
			if validateFQN(classQualifiedFQN, registry) {
				return classQualifiedFQN, true, nil
			}

			// Check if target exists in Functions map (more reliable than validateFQN)
			if callGraph != nil && callGraph.Functions[classQualifiedFQN] != nil {
				return classQualifiedFQN, true, nil
			}
		}

		// Fall back to module-level method (backward compatibility)
		// This handles cases where method might be at module level or
		// when class extraction fails
		moduleFQN := currentModule + "." + methodName
		if validateFQN(moduleFQN, registry) {
			return moduleFQN, true, nil
		}

		// Check Functions map for module-level
		if callGraph != nil && callGraph.Functions[moduleFQN] != nil {
			return moduleFQN, true, nil
		}

		// Return unresolved but with module prefix
		return moduleFQN, false, nil
	}

	// Handle simple names (no dots)
	if !strings.Contains(target, ".") {
		// Check if it's a Python built-in
		if pythonBuiltins[target] {
			// Return as builtins.function for pattern matching
			return "builtins." + target, true, nil
		}

		// Try to resolve through imports
		if fqn, ok := importMap.Resolve(target); ok {
			// Validate if it exists in registry
			if validateFQN(fqn, registry) {
				return fqn, true, nil
			}
			// Check stdlib for imported names (e.g., from os import getcwd)
			if typeEngine != nil && typeEngine.StdlibRemote != nil {
				if remoteLoader, ok := typeEngine.StdlibRemote.(*cgregistry.StdlibRegistryRemote); ok {
					if validateStdlibFQN(fqn, remoteLoader, logger) {
						return fqn, true, nil
					}
				}
			}
			// Check third-party for imported names (e.g., from requests import Session)
			if typeEngine != nil && typeEngine.ThirdPartyRemote != nil {
				if loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote); ok {
					if validateThirdPartyFQN(fqn, loader, logger) {
						return fqn, true, nil
					}
				}
			}
			// Check callGraph.Functions directly (may differ from registry module keys)
			if callGraph != nil && callGraph.Functions[fqn] != nil {
				return fqn, true, nil
			}
			// Fix: strip leading package prefix and retry
			// Import FQNs use full package path (e.g., label_studio.core.utils.params.get_env)
			// but registry keys are relative to project root (e.g., core.utils.params.get_env)
			if strippedFQN, ok := resolveWithPrefixStripping(fqn, registry, callGraph); ok {
				return strippedFQN, true, nil
			}
			return fqn, false, nil
		}

		// Not in imports - might be in same module
		sameLevelFQN := currentModule + "." + target
		if validateFQN(sameLevelFQN, registry) {
			return sameLevelFQN, true, nil
		}

		// Can't resolve - return as-is
		return target, false, nil
	}

	// Handle qualified names (with dots)
	parts := strings.SplitN(target, ".", 2)
	base := parts[0]
	rest := parts[1]

	// Phase 2 Task 9: Try type inference for variable.method() calls
	if typeEngine != nil && callerFQN != "" {
		// Try function scope first, then fall back to module scope
		var binding *resolution.VariableBinding

		// Check function scope first
		functionScope := typeEngine.GetScope(callerFQN)
		if functionScope != nil {
			if b := functionScope.GetVariable(base); b != nil {
				binding = b
			}
		}

		// If not found in function scope, try module scope
		if binding == nil {
			moduleScope := typeEngine.GetScope(currentModule)
			if moduleScope != nil {
				if b := moduleScope.GetVariable(base); b != nil {
					binding = b
				}
			}
		}

		if binding != nil {
			// Check if variable has type information
			if binding.Type != nil {
				typeFQN := binding.Type.TypeFQN

				// Skip placeholders (call:, var:) - not yet resolved
				if strings.HasPrefix(typeFQN, "call:") || strings.HasPrefix(typeFQN, "var:") {
					// Continue to legacy resolution
				} else {
					// Check if it's a builtin type
					if typeEngine.Builtins != nil && strings.HasPrefix(typeFQN, "builtins.") {
						method := typeEngine.Builtins.GetMethod(typeFQN, rest)
						if method != nil {
							// Resolved to builtin method - return with type info
							return typeFQN + "." + rest, true, binding.Type
						}
					}

					// Phase 3: Enhanced instance.method() resolution
					// Check if it's a project type (user-defined class/method)
					methodFQN := typeFQN + "." + rest

					// Phase 3: Try Functions map first with class-qualified FQN
					// This is more reliable than codeGraph.Nodes for class methods
					if callGraph != nil {
						if node := callGraph.Functions[methodFQN]; node != nil {
							// Found in Functions map with class-qualified FQN
							if node.Type == "method" || node.Type == "function_definition" ||
								node.Type == "constructor" || node.Type == "property" ||
								node.Type == "special_method" {
								return methodFQN, true, binding.Type
							}
						}
					}

					// Validate method exists in code graph (fallback)
					if codeGraph != nil {
						if node, ok := codeGraph.Nodes[methodFQN]; ok {
							if node.Type == "method_declaration" || node.Type == "function_definition" {
								// Resolved via code graph validation - return with type info
								return methodFQN, true, binding.Type
							}
						}

						// Legacy: Python class methods stored at module level
						// Try stripping the class name and looking for module.method
						// This is for backward compatibility with older indexing
						lastDot := strings.LastIndex(typeFQN, ".")
						if lastDot >= 0 {
							modulePart := typeFQN[:lastDot]
							className := typeFQN[lastDot+1:]

							// Check if it looks like a Python class (PascalCase)
							if len(className) > 0 && className[0] >= 'A' && className[0] <= 'Z' {
								pythonMethodFQN := modulePart + "." + rest
								if callGraph != nil {
									if node, ok := callGraph.Functions[pythonMethodFQN]; ok {
										if node.Type == "method_declaration" || node.Type == "function_definition" {
											// Resolved via Python module-level method lookup
											return pythonMethodFQN, true, binding.Type
										}
									}
								}
							}
						}
					}

					// PR #6: Check third-party type registry for method on typed variable
					if typeEngine.ThirdPartyRemote != nil {
						if loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote); ok {
							tpModule, tpClass := splitModuleAndName(typeFQN)
							if tpClass != "" && loader.HasModule(tpModule) {
								method := findThirdPartyClassMethod(loader, tpModule, tpClass, rest, logger)
								if method != nil {
									return methodFQN, true, &core.TypeInfo{
										TypeFQN:    typeFQN,
										Confidence: binding.Type.Confidence,
										Source:     "typeshed",
									}
								}
							}
						}
					}

					// Heuristic: If type has good confidence (>= 0.7), assume method exists
					if binding.Type.Confidence >= 0.7 {
						// Resolved via confidence heuristic - return with type info
						return methodFQN, true, binding.Type
					}

				}
			}
		}
	}

	// Try to resolve base through imports
	if baseFQN, ok := importMap.Resolve(base); ok {
		fullFQN := baseFQN + "." + rest
		// Check if it's an ORM pattern (before validateFQN, since ORM methods don't exist in source)
		if ormFQN, resolved := resolution.ResolveORMCall(target, currentModule, registry, codeGraph); resolved {
			return ormFQN, true, nil
		}
		// PR #3: Check stdlib registry before user project registry
		if typeEngine != nil && typeEngine.StdlibRemote != nil {
			if remoteLoader, ok := typeEngine.StdlibRemote.(*cgregistry.StdlibRegistryRemote); ok {
				if validateStdlibFQN(fullFQN, remoteLoader, logger) {
					return fullFQN, true, nil
				}
			}
		}
		// PR #6: Check third-party registry before user project registry
		if typeEngine != nil && typeEngine.ThirdPartyRemote != nil {
			if loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote); ok {
				if validateThirdPartyFQN(fullFQN, loader, logger) {
					return fullFQN, true, nil
				}
			}
		}
		if validateFQN(fullFQN, registry) {
			return fullFQN, true, nil
		}
		// Check callGraph.Functions directly
		if callGraph != nil && callGraph.Functions[fullFQN] != nil {
			return fullFQN, true, nil
		}
		// Fix: strip leading package prefix and retry
		if strippedFQN, ok := resolveWithPrefixStripping(fullFQN, registry, callGraph); ok {
			return strippedFQN, true, nil
		}
		return fullFQN, false, nil
	}

	// Base not in imports - might be module-level access
	// Try current module
	fullFQN := currentModule + "." + target
	if validateFQN(fullFQN, registry) {
		return fullFQN, true, nil
	}

	// Before giving up, check if it's an ORM pattern (Django, SQLAlchemy, etc.)
	// ORM methods are dynamically generated at runtime and won't be in source
	if ormFQN, resolved := resolution.ResolveORMCall(target, currentModule, registry, codeGraph); resolved {
		return ormFQN, true, nil
	}

	// PR #3: Last resort - check if target is a stdlib call (e.g., os.path.join)
	// This handles cases where stdlib modules are imported directly (import os.path)
	if typeEngine != nil && typeEngine.StdlibRemote != nil {
		if remoteLoader, ok := typeEngine.StdlibRemote.(*cgregistry.StdlibRegistryRemote); ok {
			if validateStdlibFQN(target, remoteLoader, logger) {
				return target, true, nil
			}
		}
	}

	// PR #6: Last resort - check third-party registry
	if typeEngine != nil && typeEngine.ThirdPartyRemote != nil {
		if loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote); ok {
			if validateThirdPartyFQN(target, loader, logger) {
				return target, true, nil
			}
		}
	}

	// Can't resolve - return as-is
	return target, false, nil
}

// stdlibModuleAliases maps platform-specific module aliases to their canonical names.
// For example, os.path is posixpath on Unix/Linux/Mac and ntpath on Windows.
var stdlibModuleAliases = map[string]string{
	"os.path": "posixpath", // On POSIX systems (Unix, Linux, macOS)
	// Note: On Windows, os.path would be ntpath, but we default to POSIX
	// since most development happens on Unix-like systems
}

// ValidateStdlibFQN checks if a fully qualified name is a stdlib function.
// Supports module.function, module.submodule.function, and module.Class patterns.
// Handles platform-specific module aliases (e.g., os.path -> posixpath).
// Uses lazy loading via remote registry to download modules on-demand.
//
// Examples:
//
//	"os.getcwd" - returns true if os.getcwd exists in stdlib
//	"os.path.join" - returns true if posixpath.join exists in stdlib (alias resolution)
//	"json.dumps" - returns true if json.dumps exists in stdlib
//
// Parameters:
//   - fqn: fully qualified name to check
//   - remoteLoader: remote stdlib registry loader
//   - logger: structured logger for warnings
//
// Returns:
//   - true if FQN is a stdlib function or class
func ValidateStdlibFQN(fqn string, remoteLoader *cgregistry.StdlibRegistryRemote, logger *output.Logger) bool {
	return validateStdlibFQN(fqn, remoteLoader, logger)
}

// validateStdlibFQN is the internal implementation of ValidateStdlibFQN.
func validateStdlibFQN(fqn string, remoteLoader *cgregistry.StdlibRegistryRemote, logger *output.Logger) bool {
	if remoteLoader == nil {
		return false
	}

	// Split FQN into parts: os.path.join -> ["os", "path", "join"]
	parts := strings.Split(fqn, ".")
	if len(parts) < 2 {
		return false
	}

	// Try different module combinations
	// For "os.path.join", try:
	//   1. module="os.path", function="join" (with alias resolution)
	//   2. module="os", function="path.join"
	//   3. module="os", function="path" (submodule)

	// Try longest match first (os.path)
	for i := len(parts) - 1; i >= 1; i-- {
		moduleName := strings.Join(parts[:i], ".")
		functionName := parts[i]

		// Check if this module is an alias (e.g., os.path -> posixpath)
		if canonicalName, isAlias := stdlibModuleAliases[moduleName]; isAlias {
			moduleName = canonicalName
		}

		// Lazy load module from remote registry
		module, err := remoteLoader.GetModule(moduleName, logger)
		if err != nil {
			logger.Warning("Failed to load stdlib module %s: %v", moduleName, err)
			continue
		}
		if module == nil {
			continue
		}

		// Check if it's a function
		if _, ok := module.Functions[functionName]; ok {
			return true
		}

		// Check if it's a class
		if _, ok := module.Classes[functionName]; ok {
			return true
		}

		// Check if it's a constant
		if _, ok := module.Constants[functionName]; ok {
			return true
		}

		// Check if it's an attribute
		if _, ok := module.Attributes[functionName]; ok {
			return true
		}
	}

	return false
}

// splitModuleAndName splits a fully qualified name at the first dot.
// The first part is the top-level package name, the rest is the sub-path.
//
// Examples:
//
//	"requests.Response" -> ("requests", "Response")
//	"django.http.HttpRequest" -> ("django", "http.HttpRequest")
//	"requests" -> ("requests", "")
func splitModuleAndName(fqn string) (string, string) {
	if idx := strings.Index(fqn, "."); idx > 0 {
		return fqn[:idx], fqn[idx+1:]
	}
	return fqn, ""
}

// validateThirdPartyFQN checks if a fully qualified name exists in the third-party registry.
// Checks functions, classes, and constants in the top-level module.
// Handles Python re-exports: e.g., requests.get is stored as api.get in the CDN.
// Also handles sub-module path mismatches via suffix matching and progressive stripping.
func validateThirdPartyFQN(fqn string, loader *cgregistry.ThirdPartyRegistryRemote, logger *output.Logger) bool {
	moduleName, name := splitModuleAndName(fqn)
	if name == "" || !loader.HasModule(moduleName) {
		return false
	}
	module, err := loader.GetModule(moduleName, logger)
	if err != nil || module == nil {
		return false
	}
	if _, ok := module.Functions[name]; ok {
		return true
	}
	if _, ok := module.Classes[name]; ok {
		return true
	}
	if _, ok := module.Constants[name]; ok {
		return true
	}
	// Extract the last component for suffix matching
	// e.g., "db.models.ForeignKey" → lastComponent = "ForeignKey"
	// This handles re-exports where django.db.models.ForeignKey is stored as db.models.fields.related.ForeignKey
	lastComponent := name
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		lastComponent = name[idx+1:]
	}
	lastSuffix := "." + lastComponent
	for funcName := range module.Functions {
		if funcName == lastComponent || strings.HasSuffix(funcName, lastSuffix) {
			return true
		}
	}
	for className := range module.Classes {
		if className == lastComponent || strings.HasSuffix(className, lastSuffix) {
			return true
		}
	}
	return false
}

// findThirdPartyFunction looks up a function in the third-party registry.
// Handles Python re-exports where functions are stored with sub-module paths.
// E.g., requests.get is stored as api.get because it's defined in requests/api.pyi.
func findThirdPartyFunction(loader *cgregistry.ThirdPartyRegistryRemote, moduleName, funcName string, logger *output.Logger) *core.StdlibFunction {
	fn := loader.GetFunction(moduleName, funcName, logger)
	if fn != nil {
		return fn
	}
	// Suffix match for re-exported functions
	module, err := loader.GetModule(moduleName, logger)
	if err != nil || module == nil {
		return nil
	}
	suffix := "." + funcName
	for name, f := range module.Functions {
		if strings.HasSuffix(name, suffix) {
			return f
		}
	}
	return nil
}

// findThirdPartyClass looks up a class in the third-party registry.
// Handles sub-module prefixed class names from return types.
// E.g., return type "requests.api.Response" → class stored as "Response".
func findThirdPartyClass(loader *cgregistry.ThirdPartyRegistryRemote, moduleName, className string, logger *output.Logger) *core.StdlibClass {
	cls := loader.GetClass(moduleName, className, logger)
	if cls != nil {
		return cls
	}
	// Strip sub-module prefix: "api.Response" → "Response"
	if idx := strings.LastIndex(className, "."); idx >= 0 {
		shortName := className[idx+1:]
		return loader.GetClass(moduleName, shortName, logger)
	}
	return nil
}

// findThirdPartyClassMethod looks up a class method, handling sub-module prefixed class names.
func findThirdPartyClassMethod(loader *cgregistry.ThirdPartyRegistryRemote, moduleName, className, methodName string, logger *output.Logger) *core.StdlibFunction {
	method := loader.GetClassMethod(moduleName, className, methodName, logger)
	if method != nil {
		return method
	}
	// Strip sub-module prefix from class name
	if idx := strings.LastIndex(className, "."); idx >= 0 {
		shortName := className[idx+1:]
		return loader.GetClassMethod(moduleName, shortName, methodName, logger)
	}
	return nil
}

// resolveInheritedSelfAttrMethod resolves self.attr.method() when attr comes from a parent class.
// For example, in a Django View subclass: self.request.GET.get("q") → request is from parent View.
func resolveInheritedSelfAttrMethod(
	classFQN string,
	attrName string,
	methodName string,
	codeGraph *graph.CodeGraph,
	registry *core.ModuleRegistry,
	typeEngine *resolution.TypeInferenceEngine,
	logger *output.Logger,
) (string, bool, *core.TypeInfo) {
	if typeEngine.ThirdPartyRemote == nil {
		return "", false, nil
	}
	loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote)
	if !ok || loader == nil {
		return "", false, nil
	}

	// Find the class node to get its parent classes
	for _, node := range codeGraph.Nodes {
		if node.Type != "class_definition" && node.Type != "dataclass" {
			continue
		}
		if len(node.Interface) == 0 || node.File == "" {
			continue
		}
		modulePath, ok := registry.FileToModule[node.File]
		if !ok {
			continue
		}
		if modulePath+"."+node.Name != classFQN {
			continue
		}

		// Try each parent class
		for _, superClassName := range node.Interface {
			parentFQN := resolution.ResolveParentClassFQN(
				classFQN, superClassName, node.File, typeEngine, registry,
			)
			if parentFQN == "" {
				continue
			}

			// Get the attribute type from the parent class
			attrType := resolution.ResolveInheritedSelfAttribute(
				parentFQN, attrName, typeEngine.ThirdPartyRemote, logger,
			)
			if attrType == nil || attrType.TypeFQN == "" {
				continue
			}

			// Now resolve the method on the attribute type
			attrTypeFQN := attrType.TypeFQN
			attrModule, attrClass := splitModuleAndName(attrTypeFQN)
			if attrClass != "" && loader.HasModule(attrModule) {
				method := findThirdPartyClassMethod(loader, attrModule, attrClass, methodName, logger)
				if method != nil {
					resolvedFQN := attrTypeFQN + "." + methodName
					return resolvedFQN, true, &core.TypeInfo{
						TypeFQN:    attrTypeFQN,
						Confidence: attrType.Confidence,
						Source:     "parent_class_attribute",
					}
				}
			}
		}
		break
	}

	return "", false, nil
}

// resolveParentClassInheritance iterates over class_definition nodes, resolves parent
// class FQNs via imports, and propagates parameter types from parent methods to child overrides.
// For example: class TestView(View) → resolves View to django.views.View → propagates
// request: django.http.HttpRequest from View.get to TestView.get.
func resolveParentClassInheritance(
	codeGraph *graph.CodeGraph,
	callGraph *core.CallGraph,
	registry *core.ModuleRegistry,
	typeEngine *resolution.TypeInferenceEngine,
	logger *output.Logger,
) {
	if typeEngine.ThirdPartyRemote == nil {
		return
	}

	loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote)
	if !ok || loader == nil {
		return
	}

	classContext := buildClassContext(codeGraph)
	_ = classContext

	var resolved, total int

	for _, node := range codeGraph.Nodes {
		if node.Type != "class_definition" && node.Type != "dataclass" {
			continue
		}
		if len(node.Interface) == 0 {
			continue
		}

		// Build class FQN
		modulePath, ok := registry.FileToModule[node.File]
		if !ok {
			continue
		}
		classFQN := modulePath + "." + node.Name
		total++

		// Resolve each parent class
		for _, superClassName := range node.Interface {
			parentFQN := resolution.ResolveParentClassFQN(
				classFQN, superClassName, node.File,
				typeEngine, registry,
			)
			if parentFQN == "" {
				continue
			}

			// Check if parent is in third-party registry
			parentModule, parentClass := splitModuleAndName(parentFQN)
			if parentClass == "" || !loader.HasModule(parentModule) {
				continue
			}

			// Verify class exists in CDN (with re-export suffix matching)
			cls := findThirdPartyClass(loader, parentModule, parentClass, logger)
			if cls == nil {
				continue
			}

			resolved++
			logger.Debug("Resolved parent class: %s -> %s", classFQN+"."+superClassName, parentFQN)

			// Propagate parameter types to all child methods that override parent methods
			for childFQN, childNode := range callGraph.Functions {
				// Check if this function belongs to the current class
				if !strings.HasPrefix(childFQN, classFQN+".") {
					continue
				}
				if childNode.Type != "method" && childNode.Type != "function_definition" &&
					childNode.Type != "constructor" && childNode.Type != "special_method" {
					continue
				}

				// Extract method name from FQN (last segment)
				methodName := childFQN[len(classFQN)+1:]
				if strings.Contains(methodName, ".") {
					continue // Skip nested classes/methods
				}

				resolution.PropagateParentParamTypes(
					childFQN, parentFQN, methodName,
					typeEngine, typeEngine.ThirdPartyRemote, logger,
				)
			}
		}
	}

	if total > 0 {
		logger.Debug("Parent class resolution: %d/%d classes resolved via third-party", resolved, total)
	}
}

// resolveThirdPartyVariableBindings resolves remaining call: placeholders using third-party registry.
// Must be called AFTER UpdateVariableBindingsWithFunctionReturns().
// Runs in two phases:
//
//	Phase A: Module-level function/constructor calls (e.g., call:requests.get)
//	Phase B: Instance method calls on variables resolved in Phase A (e.g., call:session.get)
func resolveThirdPartyVariableBindings(typeEngine *resolution.TypeInferenceEngine, logger *output.Logger) {
	if typeEngine.ThirdPartyRemote == nil {
		return
	}
	loader, ok := typeEngine.ThirdPartyRemote.(*cgregistry.ThirdPartyRegistryRemote)
	if !ok || loader == nil {
		return
	}

	// Phase A: Resolve direct module.function calls
	for _, scope := range typeEngine.Scopes {
		for varName, bindings := range scope.Variables {
			for i, binding := range bindings {
				if binding == nil || binding.Type == nil || !strings.HasPrefix(binding.Type.TypeFQN, "call:") {
					continue
				}
				funcName := strings.TrimPrefix(binding.Type.TypeFQN, "call:")
				if !strings.Contains(funcName, ".") {
					continue
				}
				// Skip instance method calls (receiver is a variable) — handle in Phase B
				parts := strings.SplitN(funcName, ".", 2)
				if scope.GetVariable(parts[0]) != nil {
					continue
				}

				moduleName, name := splitModuleAndName(funcName)
				if !loader.HasModule(moduleName) {
					continue
				}

				// Try as module-level function (e.g., requests.get → may be stored as api.get)
				fn := findThirdPartyFunction(loader, moduleName, name, logger)
				if fn != nil && fn.ReturnType != "" {
					scope.Variables[varName][i].Type = &core.TypeInfo{
						TypeFQN:    fn.ReturnType,
						Confidence: binding.Type.Confidence * fn.Confidence * 0.95,
						Source:     "typeshed",
					}
					scope.Variables[varName][i].AssignedFrom = funcName
					continue
				}

				// Try as constructor (e.g., requests.Session)
				cls := findThirdPartyClass(loader, moduleName, name, logger)
				if cls != nil {
					scope.Variables[varName][i].Type = &core.TypeInfo{
						TypeFQN:    funcName,
						Confidence: binding.Type.Confidence * 0.95,
						Source:     "typeshed",
					}
					scope.Variables[varName][i].AssignedFrom = funcName
					continue
				}
			}
		}
	}

	// Phase B: Resolve instance method calls on variables resolved in Phase A
	for _, scope := range typeEngine.Scopes {
		for varName, bindings := range scope.Variables {
			for i, binding := range bindings {
				if binding == nil || binding.Type == nil || !strings.HasPrefix(binding.Type.TypeFQN, "call:") {
					continue
				}
				funcName := strings.TrimPrefix(binding.Type.TypeFQN, "call:")
				if !strings.Contains(funcName, ".") {
					continue
				}

				parts := strings.SplitN(funcName, ".", 2)
				receiver := parts[0]
				methodName := parts[1]

				receiverBinding := scope.GetVariable(receiver)
				if receiverBinding == nil || receiverBinding.Type == nil ||
					strings.HasPrefix(receiverBinding.Type.TypeFQN, "call:") {
					continue
				}

				tpModule, tpClass := splitModuleAndName(receiverBinding.Type.TypeFQN)
				if tpClass == "" || !loader.HasModule(tpModule) {
					continue
				}

				method := findThirdPartyClassMethod(loader, tpModule, tpClass, methodName, logger)
				if method != nil && method.ReturnType != "" {
					scope.Variables[varName][i].Type = &core.TypeInfo{
						TypeFQN:    method.ReturnType,
						Confidence: binding.Type.Confidence * method.Confidence * 0.9,
						Source:     "typeshed",
					}
					scope.Variables[varName][i].AssignedFrom = receiverBinding.Type.TypeFQN + "." + methodName
				}
			}
		}
	}
}

// ValidateFQN checks if a fully qualified name exists in the registry.
// Handles both module names and function names within modules.
//
// Examples:
//
//	"myapp.utils" - checks if module exists
//	"myapp.utils.sanitize" - checks if module "myapp.utils" exists
//
// Parameters:
//   - fqn: fully qualified name to validate
//   - registry: module registry
//
// Returns:
//   - true if FQN is valid (module or function in existing module)
func ValidateFQN(fqn string, registry *core.ModuleRegistry) bool {
	return validateFQN(fqn, registry)
}

// validateFQN is the internal implementation of ValidateFQN.
func validateFQN(fqn string, registry *core.ModuleRegistry) bool {
	// Check if it's a module
	if _, ok := registry.Modules[fqn]; ok {
		return true
	}

	// Check if parent module exists (for module-level functions/classes)
	// "myapp.utils.sanitize" → check if "myapp.utils" exists
	lastDot := strings.LastIndex(fqn, ".")
	if lastDot > 0 {
		parentModule := fqn[:lastDot]
		if _, ok := registry.Modules[parentModule]; ok {
			return true
		}

		// For class methods (module.ClassName.method_name), parent is ClassName
		// which won't be in registry. Check grandparent (module level).
		// "adapter.UserAdapter.to_domain_model" → check if "adapter" exists
		secondLastDot := strings.LastIndex(parentModule, ".")
		if secondLastDot > 0 {
			grandparentModule := parentModule[:secondLastDot]
			if _, ok := registry.Modules[grandparentModule]; ok {
				return true
			}
		}
	}

	return false
}

// resolveWithPrefixStripping tries progressively stripping leading package components
// from an FQN and re-checking against the registry and callGraph.Functions.
//
// This fixes the common mismatch where imports use the full package path
// (e.g., "label_studio.core.utils.params.get_env") but the registry and callGraph
// key modules relative to the project root (e.g., "core.utils.params.get_env").
func resolveWithPrefixStripping(fqn string, registry *core.ModuleRegistry, callGraph *core.CallGraph) (string, bool) {
	remaining := fqn
	for {
		idx := strings.Index(remaining, ".")
		if idx < 0 {
			break
		}
		remaining = remaining[idx+1:]
		// Need at least one dot remaining (module.function)
		if !strings.Contains(remaining, ".") {
			break
		}
		if validateFQN(remaining, registry) {
			return remaining, true
		}
		if callGraph != nil && callGraph.Functions[remaining] != nil {
			return remaining, true
		}
	}
	return fqn, false
}

// resolveCallTargetLegacy is the old resolution logic without type inference.
// Used for backward compatibility with existing tests.
func resolveCallTargetLegacy(target string, importMap *core.ImportMap, registry *core.ModuleRegistry, currentModule string, codeGraph *graph.CodeGraph) (string, bool) {
	// Handle self.method() calls - resolve to current module
	if after, ok := strings.CutPrefix(target, "self."); ok {
		methodName := after
		// Resolve to module.method
		moduleFQN := currentModule + "." + methodName
		// Validate exists
		if validateFQN(moduleFQN, registry) {
			return moduleFQN, true
		}
		// Return unresolved but with module prefix
		return moduleFQN, false
	}

	// Handle simple names (no dots)
	if !strings.Contains(target, ".") {
		// Check if it's a Python built-in
		if pythonBuiltins[target] {
			// Return as builtins.function for pattern matching
			return "builtins." + target, true
		}

		// Try to resolve through imports
		if fqn, ok := importMap.Resolve(target); ok {
			// Validate if it exists in registry
			resolved := validateFQN(fqn, registry)
			return fqn, resolved
		}

		// Not in imports - might be in same module
		sameLevelFQN := currentModule + "." + target
		if validateFQN(sameLevelFQN, registry) {
			return sameLevelFQN, true
		}

		// Can't resolve - return as-is
		return target, false
	}

	// Handle qualified names (with dots)
	parts := strings.SplitN(target, ".", 2)
	base := parts[0]
	rest := parts[1]

	// Try to resolve base through imports
	if baseFQN, ok := importMap.Resolve(base); ok {
		fullFQN := baseFQN + "." + rest
		// Check if it's an ORM pattern (before validateFQN, since ORM methods don't exist in source)
		if ormFQN, resolved := resolution.ResolveORMCall(target, currentModule, registry, codeGraph); resolved {
			return ormFQN, true
		}
		if validateFQN(fullFQN, registry) {
			return fullFQN, true
		}
		return fullFQN, false
	}

	// Base not in imports - might be module-level access
	// Try current module
	fullFQN := currentModule + "." + target
	if validateFQN(fullFQN, registry) {
		return fullFQN, true
	}

	// Before giving up, check if it's an ORM pattern (Django, SQLAlchemy, etc.)
	// ORM methods are dynamically generated at runtime and won't be in source
	if ormFQN, resolved := resolution.ResolveORMCall(target, currentModule, registry, codeGraph); resolved {
		return ormFQN, true
	}

	// Can't resolve - return as-is
	return target, false
}

// DetectPythonVersion infers Python version from project files.
// It checks in order:
//  1. .python-version file
//  2. pyproject.toml [tool.poetry.dependencies] or [project] requires-python
//  3. Defaults to "3.14"
//
// Parameters:
//   - projectPath: absolute path to the project root
//
// Returns:
//   - Python version string (e.g., "3.14", "3.11", "3.9")
func DetectPythonVersion(projectPath string) string {
	return detectPythonVersionInternal(projectPath)
}

// detectPythonVersionInternal is the implementation - extracted from python_version_detector.go.
func detectPythonVersionInternal(projectPath string) string {
	// 1. Check .python-version file
	if version := readPythonVersionFile(projectPath); version != "" {
		return version
	}

	// 2. Check pyproject.toml
	if version := parsePyprojectToml(projectPath); version != "" {
		return version
	}

	// 3. Default to 3.11 (most widely used stable version)
	return "3.11"
}

// Helper functions for DetectPythonVersion.
func readPythonVersionFile(projectPath string) string {
	versionFile := filepath.Join(projectPath, ".python-version")
	data, err := ReadFileBytes(versionFile)
	if err != nil {
		return ""
	}

	version := strings.TrimSpace(string(data))
	return extractMajorMinor(version)
}

func parsePyprojectToml(projectPath string) string {
	// Import the functionality from cgregistry which has the full implementation
	// For now, we'll use a simplified version
	tomlFile := filepath.Join(projectPath, "pyproject.toml")
	data, err := ReadFileBytes(tomlFile)
	if err != nil {
		return ""
	}

	// Very simple regex-free parsing - just look for version numbers
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		// Check for requires-python or python = patterns
		if strings.Contains(line, "requires-python") || strings.Contains(line, "python") {
			// Extract version number pattern (e.g., 3.11, 3.9, etc.)
			parts := strings.FieldsSeq(line)
			for part := range parts {
				part = strings.Trim(part, `"'>=<~^`)
				if strings.Contains(part, ".") && len(part) >= 3 && len(part) <= 5 {
					// Check if it looks like a version (starts with digit)
					if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
						return extractMajorMinor(part)
					}
				}
			}
		}
	}

	return ""
}

func extractMajorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return ""
}

// populateInferredReturnTypes back-populates inferred return types from the TypeInferenceEngine
// to function Node.ReturnType fields and detects void functions (no return <expr>).
//
// For each Python function/method in the call graph:
//  1. If it already has an annotation-based ReturnType, skip it
//  2. If TypeEngine has an inferred return type with sufficient confidence, use it
//  3. If no inferred type AND the function has no return <expr> statements, mark as "None" (void)
func populateInferredReturnTypes(
	callGraph *core.CallGraph,
	typeEngine *resolution.TypeInferenceEngine,
	functionsWithReturnValues map[string]bool,
	logger *output.Logger,
) {
	populated := 0
	voidDetected := 0

	for fqn, node := range callGraph.Functions {
		// Only process Python functions (skip Java etc.)
		if !strings.HasSuffix(node.File, ".py") {
			continue
		}

		// Skip if already has annotation-based return type
		if node.ReturnType != "" {
			continue
		}

		// Try to use inferred return type from TypeEngine
		typeInfo, ok := typeEngine.GetReturnType(fqn)
		if ok && typeInfo != nil && typeInfo.Confidence >= 0.5 {
			// Skip unresolved placeholders
			if strings.HasPrefix(typeInfo.TypeFQN, "call:") || strings.HasPrefix(typeInfo.TypeFQN, "var:") {
				// Function has return expressions but we couldn't resolve the type — leave empty
				continue
			}
			node.ReturnType = NormalizeReturnType(typeInfo.TypeFQN)
			populated++
			continue
		}

		// No inferred type — check if this is a void function
		// A function is void if it has NO return <expr> statements at all
		if !functionsWithReturnValues[fqn] {
			node.ReturnType = "None"
			voidDetected++
		}
		// If it HAS return expressions but we couldn't infer the type, leave empty (honest unknown)
	}

	logger.Debug("Populated %d inferred return types, detected %d void functions", populated, voidDetected)
}

// NormalizeReturnType converts fully-qualified builtin type names to their short form.
// This normalizes the internal representation (e.g., "builtins.str") to the user-facing
// form (e.g., "str") that matches what developers write in annotations.
func NormalizeReturnType(typeFQN string) string {
	switch typeFQN {
	case "builtins.str":
		return "str"
	case "builtins.int":
		return "int"
	case "builtins.float":
		return "float"
	case "builtins.bool":
		return "bool"
	case "builtins.list":
		return "list"
	case "builtins.dict":
		return "dict"
	case "builtins.set":
		return "set"
	case "builtins.tuple":
		return "tuple"
	case "builtins.NoneType":
		return "None"
	case "builtins.bytes":
		return "bytes"
	case "builtins.complex":
		return "complex"
	case "builtins.Generator":
		return "Generator"
	default:
		return typeFQN
	}
}
