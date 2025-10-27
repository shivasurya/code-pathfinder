package callgraph

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
)

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
//   Given:
//     File: myapp/views.py
//       def get_user():
//           sanitize(data)  # call to myapp.utils.sanitize
//
//   Creates:
//     edges: {"myapp.views.get_user": ["myapp.utils.sanitize"]}
//     reverseEdges: {"myapp.utils.sanitize": ["myapp.views.get_user"]}
//     callSites: {"myapp.views.get_user": [CallSite{Target: "sanitize", ...}]}
func BuildCallGraph(codeGraph *graph.CodeGraph, registry *ModuleRegistry, projectRoot string) (*CallGraph, error) {
	callGraph := NewCallGraph()

	// First, index all function definitions from the code graph
	// This builds the Functions map for quick lookup
	indexFunctions(codeGraph, callGraph, registry)

	// Process each Python file in the project
	for modulePath, filePath := range registry.Modules {
		// Skip non-Python files
		if !strings.HasSuffix(filePath, ".py") {
			continue
		}

		// Read source code for parsing
		sourceCode, err := readFileBytes(filePath)
		if err != nil {
			// Skip files we can't read
			continue
		}

		// Extract imports to build ImportMap for this file
		importMap, err := ExtractImports(filePath, sourceCode, registry)
		if err != nil {
			// Skip files with import errors
			continue
		}

		// Extract all call sites from this file
		callSites, err := ExtractCallSites(filePath, sourceCode, importMap)
		if err != nil {
			// Skip files with call site extraction errors
			continue
		}

		// Get all function definitions in this file
		fileFunctions := getFunctionsInFile(codeGraph, filePath)

		// Process each call site to resolve targets and build edges
		for _, callSite := range callSites {
			// Find the caller function containing this call site
			callerFQN := findContainingFunction(callSite.Location, fileFunctions, modulePath)
			if callerFQN == "" {
				// Call at module level - use module name as caller
				callerFQN = modulePath
			}

			// Resolve the call target to a fully qualified name
			targetFQN, resolved := resolveCallTarget(callSite.Target, importMap, registry, modulePath)

			// Update call site with resolution information
			callSite.TargetFQN = targetFQN
			callSite.Resolved = resolved

			// Add call site to graph (dereference pointer)
			callGraph.AddCallSite(callerFQN, *callSite)

			// Add edge if we successfully resolved the target
			if resolved {
				callGraph.AddEdge(callerFQN, targetFQN)
			}
		}
	}

	return callGraph, nil
}

// indexFunctions builds the Functions map in the call graph.
// Extracts all function definitions from the code graph and maps them by FQN.
//
// Parameters:
//   - codeGraph: the parsed code graph
//   - callGraph: the call graph being built
//   - registry: module registry for resolving file paths to modules
func indexFunctions(codeGraph *graph.CodeGraph, callGraph *CallGraph, registry *ModuleRegistry) {
	for _, node := range codeGraph.Nodes {
		// Only index function/method definitions
		if node.Type != "method_declaration" && node.Type != "function_definition" {
			continue
		}

		// Get the module path for this function's file
		modulePath, ok := registry.FileToModule[node.File]
		if !ok {
			continue
		}

		// Build fully qualified name: module.function
		fqn := modulePath + "." + node.Name
		callGraph.Functions[fqn] = node
	}
}

// getFunctionsInFile returns all function definitions in a specific file.
//
// Parameters:
//   - codeGraph: the parsed code graph
//   - filePath: absolute path to the file
//
// Returns:
//   - List of function/method nodes in the file, sorted by line number
func getFunctionsInFile(codeGraph *graph.CodeGraph, filePath string) []*graph.Node {
	var functions []*graph.Node

	for _, node := range codeGraph.Nodes {
		if node.File == filePath &&
			(node.Type == "method_declaration" || node.Type == "function_definition") {
			functions = append(functions, node)
		}
	}

	return functions
}

// findContainingFunction finds the function that contains a given call site location.
// Uses line numbers to determine which function a call belongs to.
//
// Algorithm:
//  1. Iterate through all functions in the file
//  2. Find function with the highest line number that's still <= call line
//  3. Return the FQN of that function
//
// Parameters:
//   - location: source location of the call site
//   - functions: all function definitions in the file
//   - modulePath: module path of the file
//
// Returns:
//   - Fully qualified name of the containing function, or empty if not found
func findContainingFunction(location Location, functions []*graph.Node, modulePath string) string {
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
		return modulePath + "." + bestMatch.Name
	}

	return ""
}

// resolveCallTarget resolves a call target name to a fully qualified name.
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
//
// Returns:
//   - Fully qualified name of the target
//   - Boolean indicating if resolution was successful
//
// Examples:
//   target="sanitize", imports={"sanitize": "myapp.utils.sanitize"}
//     → "myapp.utils.sanitize", true
//
//   target="utils.sanitize", imports={"utils": "myapp.utils"}
//     → "myapp.utils.sanitize", true
//
//   target="obj.method", imports={}
//     → "obj.method", false  (needs type inference)

// Python built-in functions that should not be resolved as module functions.
var pythonBuiltins = map[string]bool{
	"eval":       true,
	"exec":       true,
	"input":      true,
	"raw_input":  true,
	"compile":    true,
	"__import__": true,
}

func resolveCallTarget(target string, importMap *ImportMap, registry *ModuleRegistry, currentModule string) (string, bool) {
	// Handle self.method() calls - resolve to current module
	if strings.HasPrefix(target, "self.") {
		methodName := strings.TrimPrefix(target, "self.")
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
			// Found in imports - return the FQN
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

	// Can't resolve - return as-is
	return target, false
}

// validateFQN checks if a fully qualified name exists in the registry.
// Handles both module names and function names within modules.
//
// Examples:
//   "myapp.utils" - checks if module exists
//   "myapp.utils.sanitize" - checks if module "myapp.utils" exists
//
// Parameters:
//   - fqn: fully qualified name to validate
//   - registry: module registry
//
// Returns:
//   - true if FQN is valid (module or function in existing module)
func validateFQN(fqn string, registry *ModuleRegistry) bool {
	// Check if it's a module
	if _, ok := registry.Modules[fqn]; ok {
		return true
	}

	// Check if parent module exists (for functions)
	// "myapp.utils.sanitize" → check if "myapp.utils" exists
	lastDot := strings.LastIndex(fqn, ".")
	if lastDot > 0 {
		parentModule := fqn[:lastDot]
		if _, ok := registry.Modules[parentModule]; ok {
			return true
		}
	}

	return false
}

// readFileBytes reads a file and returns its contents as a byte slice.
// Helper function for reading source code.
func readFileBytes(filePath string) ([]byte, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(absPath)
}
