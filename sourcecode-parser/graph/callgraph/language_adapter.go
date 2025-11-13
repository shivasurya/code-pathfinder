package callgraph

// LanguageAnalyzer is the core interface all language implementations must satisfy.
// It abstracts the 4-pass static analysis algorithm:
// 1. Parse: Source code → AST
// 2. Type Inference: Extract imports, functions, classes, infer types
// 3. Call Graph: Extract call sites, statements, variables
// 4. Taint Analysis: Analyze data flow for security vulnerabilities.
type LanguageAnalyzer interface {
	// Name returns the language name (e.g., "python", "go", "rust")
	Name() string

	// FileExtensions returns the file extensions for this language (e.g., [".py"], [".go"], [".rs"])
	FileExtensions() []string

	// Parse converts source code to language-specific AST
	Parse(filePath string, source []byte) (*ParsedModule, error)

	// ExtractImports returns all import statements from the module
	ExtractImports(module *ParsedModule) (*ImportMap, error)

	// ExtractFunctions returns all function/method definitions
	ExtractFunctions(module *ParsedModule) ([]*FunctionDef, error)

	// ExtractClasses returns all class/struct definitions
	ExtractClasses(module *ParsedModule) ([]*ClassDef, error)

	// InferTypes performs type inference using language-specific rules
	InferTypes(module *ParsedModule, registry *ModuleRegistry) (*TypeContext, error)

	// ExtractCallSites returns all function calls within a function
	ExtractCallSites(fn *FunctionDef) ([]*CallSite, error)

	// ExtractStatements returns all statements within a function
	ExtractStatements(fn *FunctionDef) ([]*Statement, error)

	// ExtractVariables returns all variable declarations/assignments
	ExtractVariables(fn *FunctionDef) ([]*Variable, error)

	// AnalyzeTaint performs taint analysis on function CFG
	AnalyzeTaint(fn *FunctionDef, cfg *CFG) (*TaintSummary, error)

	// ResolveType resolves a type expression to TypeInfo
	ResolveType(expr string, context *TypeContext) (*TypeInfo, error)

	// SupportsFramework checks if the analyzer supports a framework (e.g., "flask", "django", "gin")
	SupportsFramework(name string) bool
}

// ParsedModule represents a language-agnostic parsed file.
type ParsedModule struct {
	FilePath string                 // Path to the source file
	Language string                 // Language name (e.g., "python", "go")
	AST      interface{}            // Language-specific AST root
	Metadata map[string]interface{} // Language-specific metadata
}

// LanguageRegistry manages registered language analyzers.
type LanguageRegistry struct {
	languages map[string]LanguageAnalyzer // name → analyzer
	byExt     map[string]LanguageAnalyzer // extension → analyzer
}

// NewLanguageRegistry creates an empty registry.
func NewLanguageRegistry() *LanguageRegistry {
	return &LanguageRegistry{
		languages: make(map[string]LanguageAnalyzer),
		byExt:     make(map[string]LanguageAnalyzer),
	}
}

// Register adds a language analyzer to the registry.
func (r *LanguageRegistry) Register(analyzer LanguageAnalyzer) {
	name := analyzer.Name()
	r.languages[name] = analyzer

	// Register all file extensions
	for _, ext := range analyzer.FileExtensions() {
		r.byExt[ext] = analyzer
	}
}

// GetByExtension returns the analyzer for a file extension.
func (r *LanguageRegistry) GetByExtension(ext string) (LanguageAnalyzer, bool) {
	analyzer, ok := r.byExt[ext]
	return analyzer, ok
}

// GetByName returns the analyzer by language name.
func (r *LanguageRegistry) GetByName(name string) (LanguageAnalyzer, bool) {
	analyzer, ok := r.languages[name]
	return analyzer, ok
}
