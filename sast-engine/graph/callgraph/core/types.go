package core

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
)

// Location represents a source code location for tracking call sites.
// This enables precise mapping of where calls occur in the source code.
type Location struct {
	File   string // Absolute path to the source file
	Line   int    // Line number (1-indexed)
	Column int    // Column number (1-indexed)
}

// CallSite represents a function/method call location in the source code.
// It captures both the syntactic information (where the call is) and
// semantic information (what is being called and with what arguments).
type CallSite struct {
	Target        string     // The name of the function being called (e.g., "eval", "utils.sanitize")
	Location      Location   // Where this call occurs in the source code
	Arguments     []Argument // Arguments passed to the call
	Resolved      bool       // Whether we successfully resolved this call to a definition
	TargetFQN     string     // Fully qualified name after resolution (e.g., "myapp.utils.sanitize")
	FailureReason string     // Why resolution failed (empty if Resolved=true)

	// Phase 2: Type inference metadata
	ResolvedViaTypeInference bool    // Was this resolved using type inference?
	InferredType             string  // The inferred type FQN (e.g., "builtins.str", "test.User")
	TypeConfidence           float32 // Confidence score of the type inference (0.0-1.0)
	TypeSource               string  // How type was inferred (e.g., "literal", "return_type", "class_instantiation")
}

// Resolution failure reason categories for diagnostics:
// - "external_framework" - Call to Django, REST framework, pytest, stdlib, etc.
// - "orm_pattern" - Django ORM patterns like Model.objects.filter()
// - "attribute_chain" - Method calls on return values like response.json()
// - "variable_method" - Method calls on variables like value.split()
// - "super_call" - Calls via super() to parent class methods
// - "not_in_imports" - Simple function call not found in imports
// - "unknown" - Unresolved for other reasons

// Argument represents a single argument passed to a function call.
// Tracks both the value/expression and metadata about the argument.
type Argument struct {
	Value      string // The argument expression as a string
	IsVariable bool   // Whether this argument is a variable reference
	Position   int    // Position in the argument list (0-indexed)
}

// CallGraph represents the complete call graph of a program.
// It maps function definitions to their call sites and provides
// both forward (callers → callees) and reverse (callees → callers) edges.
//
// Example:
//
//	Function A calls B and C
//	edges: {"A": ["B", "C"]}
//	reverseEdges: {"B": ["A"], "C": ["A"]}
type CallGraph struct {
	// Forward edges: maps fully qualified function name to list of functions it calls
	// Key: caller FQN (e.g., "myapp.views.get_user")
	// Value: list of callee FQNs (e.g., ["myapp.db.query", "myapp.utils.sanitize"])
	Edges map[string][]string

	// Reverse edges: maps fully qualified function name to list of functions that call it
	// Useful for backward slicing and finding all callers of a function
	// Key: callee FQN
	// Value: list of caller FQNs
	ReverseEdges map[string][]string

	// Detailed call site information for each function
	// Key: caller FQN
	// Value: list of all call sites within that function
	CallSites map[string][]CallSite

	// Map from fully qualified name to the actual function node in the graph
	// This allows quick lookup of function metadata (line number, file, etc.)
	Functions map[string]*graph.Node

	// Taint summaries for each function (intra-procedural analysis results)
	// Key: function FQN
	// Value: TaintSummary with taint flow information
	Summaries map[string]*TaintSummary
}

// NewCallGraph creates and initializes a new CallGraph instance.
// All maps are pre-allocated to avoid nil pointer issues.
func NewCallGraph() *CallGraph {
	return &CallGraph{
		Edges:        make(map[string][]string),
		ReverseEdges: make(map[string][]string),
		CallSites:    make(map[string][]CallSite),
		Functions:    make(map[string]*graph.Node),
		Summaries:    make(map[string]*TaintSummary),
	}
}

// AddEdge adds a directed edge from caller to callee in the call graph.
// Automatically updates both forward and reverse edges.
//
// Parameters:
//   - caller: fully qualified name of the calling function
//   - callee: fully qualified name of the called function
func (cg *CallGraph) AddEdge(caller, callee string) {
	// Add forward edge
	if !contains(cg.Edges[caller], callee) {
		cg.Edges[caller] = append(cg.Edges[caller], callee)
	}

	// Add reverse edge
	if !contains(cg.ReverseEdges[callee], caller) {
		cg.ReverseEdges[callee] = append(cg.ReverseEdges[callee], caller)
	}
}

// AddCallSite adds a call site to the call graph.
// This stores detailed information about where and how a function is called.
//
// Parameters:
//   - caller: fully qualified name of the calling function
//   - callSite: detailed information about the call
func (cg *CallGraph) AddCallSite(caller string, callSite CallSite) {
	cg.CallSites[caller] = append(cg.CallSites[caller], callSite)
}

// GetCallers returns all functions that call the specified function.
// Uses the reverse edges for efficient lookup.
//
// Parameters:
//   - callee: fully qualified name of the function
//
// Returns:
//   - list of caller FQNs, or empty slice if no callers found
func (cg *CallGraph) GetCallers(callee string) []string {
	if callers, ok := cg.ReverseEdges[callee]; ok {
		return callers
	}
	return []string{}
}

// GetCallees returns all functions called by the specified function.
// Uses the forward edges for efficient lookup.
//
// Parameters:
//   - caller: fully qualified name of the function
//
// Returns:
//   - list of callee FQNs, or empty slice if no callees found
func (cg *CallGraph) GetCallees(caller string) []string {
	if callees, ok := cg.Edges[caller]; ok {
		return callees
	}
	return []string{}
}

// ModuleRegistry maintains the mapping between Python file paths and module paths.
// This is essential for resolving imports and building fully qualified names.
//
// Example:
//
//	File: /project/myapp/utils/helpers.py
//	Module: myapp.utils.helpers
type ModuleRegistry struct {
	// Maps fully qualified module path to absolute file path
	// Key: "myapp.utils.helpers"
	// Value: "/absolute/path/to/myapp/utils/helpers.py"
	Modules map[string]string

	// Maps absolute file path to fully qualified module path (reverse of Modules)
	// Key: "/absolute/path/to/myapp/utils/helpers.py"
	// Value: "myapp.utils.helpers"
	// Used for resolving relative imports
	FileToModule map[string]string

	// Maps short module names to all matching file paths (handles ambiguity)
	// Key: "helpers"
	// Value: ["/path/to/myapp/utils/helpers.py", "/path/to/lib/helpers.py"]
	ShortNames map[string][]string

	// Cache for resolved imports to avoid redundant lookups
	// Key: import string (e.g., "utils.helpers")
	// Value: fully qualified module path
	ResolvedImports map[string]string
}

// NewModuleRegistry creates and initializes a new ModuleRegistry instance.
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		Modules:         make(map[string]string),
		FileToModule:    make(map[string]string),
		ShortNames:      make(map[string][]string),
		ResolvedImports: make(map[string]string),
	}
}

// AddModule registers a module in the registry.
// Automatically indexes both the full module path and the short name.
//
// Parameters:
//   - modulePath: fully qualified module path (e.g., "myapp.utils.helpers")
//   - filePath: absolute file path (e.g., "/project/myapp/utils/helpers.py")
func (mr *ModuleRegistry) AddModule(modulePath, filePath string) {
	mr.Modules[modulePath] = filePath
	mr.FileToModule[filePath] = modulePath

	// Extract short name (last component)
	// "myapp.utils.helpers" → "helpers"
	shortName := extractShortName(modulePath)
	if !containsString(mr.ShortNames[shortName], filePath) {
		mr.ShortNames[shortName] = append(mr.ShortNames[shortName], filePath)
	}
}

// GetModulePath returns the file path for a given module, if it exists.
//
// Parameters:
//   - modulePath: fully qualified module path
//
// Returns:
//   - file path and true if found, empty string and false otherwise
func (mr *ModuleRegistry) GetModulePath(modulePath string) (string, bool) {
	filePath, ok := mr.Modules[modulePath]
	return filePath, ok
}

// ImportMap represents the import statements in a single Python file.
// Maps local aliases to fully qualified module paths.
//
// Example:
//
//	File contains: from myapp.utils import sanitize as clean
//	Imports: {"clean": "myapp.utils.sanitize"}
type ImportMap struct {
	FilePath string            // Absolute path to the file containing these imports
	Imports  map[string]string // Maps alias/name to fully qualified module path
}

// NewImportMap creates and initializes a new ImportMap instance.
func NewImportMap(filePath string) *ImportMap {
	return &ImportMap{
		FilePath: filePath,
		Imports:  make(map[string]string),
	}
}

// AddImport adds an import mapping to the import map.
//
// Parameters:
//   - alias: the local name used in the file (e.g., "clean", "sanitize", "utils")
//   - fqn: the fully qualified name (e.g., "myapp.utils.sanitize")
func (im *ImportMap) AddImport(alias, fqn string) {
	im.Imports[alias] = fqn
}

// Resolve looks up the fully qualified name for a local alias.
//
// Parameters:
//   - alias: the local name to resolve
//
// Returns:
//   - fully qualified name and true if found, empty string and false otherwise
func (im *ImportMap) Resolve(alias string) (string, bool) {
	fqn, ok := im.Imports[alias]
	return fqn, ok
}

// Helper function to check if a string slice contains a specific string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function alias for consistency.
func containsString(slice []string, item string) bool {
	return contains(slice, item)
}

// Helper function to extract the last component of a dotted path.
// Example: "myapp.utils.helpers" → "helpers".
func extractShortName(modulePath string) string {
	// Find last dot
	for i := len(modulePath) - 1; i >= 0; i-- {
		if modulePath[i] == '.' {
			return modulePath[i+1:]
		}
	}
	return modulePath
}
