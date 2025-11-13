package callgraph

// FunctionDef represents a language-agnostic function definition.
type FunctionDef struct {
	Name       string      // Simple name (e.g., "calculate")
	FQN        string      // Fully qualified name (e.g., "module.Class.calculate")
	Parameters []Parameter // Function parameters
	ReturnType *TypeInfo   // Return type information
	Body       interface{} // Language-specific AST body
	Location   Location    // Source location
}

// ClassDef represents a language-agnostic class/struct definition.
type ClassDef struct {
	Name        string               // Simple name
	FQN         string               // Fully qualified name
	Methods     []*FunctionDef       // All methods
	Attributes  map[string]*TypeInfo // Class attributes/fields
	BaseClasses []string             // Parent classes/interfaces
	Location    Location             // Source location
}

// Parameter represents a function parameter.
type Parameter struct {
	Name         string      // Parameter name
	Type         *TypeInfo   // Parameter type
	DefaultValue interface{} // Default value (nil if none)
	Position     int         // Parameter position (0-indexed)
}

// Variable represents a variable declaration or assignment.
type Variable struct {
	Name     string    // Variable name
	Type     *TypeInfo // Variable type
	Scope    string    // "local", "global", "nonlocal"
	Location Location  // Source location
}

// TypeContext holds type information for a module.
type TypeContext struct {
	Variables map[string]*TypeInfo    // Variable name → type
	Functions map[string]*FunctionDef // Function name → definition
	Classes   map[string]*ClassDef    // Class name → definition
	Imports   *ImportMap              // Import map
}

// CFG represents a Control Flow Graph for a function.
// This is a placeholder type that will be fully implemented in Phase 4.
// For now, it's defined to satisfy the LanguageAnalyzer interface contract.
type CFG struct {
	// Placeholder - will be implemented in Phase 4
}
