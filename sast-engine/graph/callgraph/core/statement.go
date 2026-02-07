package core

// StatementType represents the type of statement in the code.
type StatementType string

const (
	// Assignment represents variable assignments: x = expr.
	StatementTypeAssignment StatementType = "assignment"

	// Call represents function/method calls: foo(), obj.method().
	StatementTypeCall StatementType = "call"

	// Return represents return statements: return expr.
	StatementTypeReturn StatementType = "return"

	// If represents conditional statements: if condition: ...
	StatementTypeIf StatementType = "if"

	// For represents loop statements: for x in iterable: ...
	StatementTypeFor StatementType = "for"

	// While represents while loop statements: while condition: ...
	StatementTypeWhile StatementType = "while"

	// With represents context manager statements: with expr as var: ...
	StatementTypeWith StatementType = "with"

	// Try represents exception handling: try: ... except: ...
	StatementTypeTry StatementType = "try"

	// Raise represents exception raising: raise Exception().
	StatementTypeRaise StatementType = "raise"

	// Import represents import statements: import module, from module import name.
	StatementTypeImport StatementType = "import"

	// Expression represents expression statements (calls, attribute access, etc.).
	StatementTypeExpression StatementType = "expression"
)

// Statement represents a single statement in the code with def-use information.
type Statement struct {
	// Type is the kind of statement (assignment, call, return, etc.)
	Type StatementType

	// LineNumber is the source line number for this statement (1-indexed)
	LineNumber uint32

	// Def is the variable being defined by this statement (if any)
	// For assignments: the left-hand side variable
	// For for loops: the loop variable
	// For with statements: the as variable
	// Empty string if no definition
	Def string

	// Uses is the list of variables used/read by this statement
	// For assignments: variables in the right-hand side expression
	// For calls: variables used in arguments
	// For conditions: variables in the condition expression
	Uses []string

	// CallTarget is the function/method being called (if Type == StatementTypeCall)
	// Format: "function_name" for direct calls, "obj.method" for method calls
	// Empty string for non-call statements
	CallTarget string

	// CallArgs are the argument variables passed to the call (if Type == StatementTypeCall)
	// Only includes variable names, not literals
	CallArgs []string

	// NestedStatements contains statements inside this statement's body
	// Used for if/for/while/with/try blocks
	// Empty for simple statements like assignments
	NestedStatements []*Statement

	// ElseBranch contains statements in the else branch (if applicable)
	// Used for if/try statements
	ElseBranch []*Statement
}

// GetDef returns the variable defined by this statement, or empty string if none.
func (s *Statement) GetDef() string {
	return s.Def
}

// GetUses returns the list of variables used by this statement.
func (s *Statement) GetUses() []string {
	return s.Uses
}

// IsCall returns true if this statement is a function/method call.
func (s *Statement) IsCall() bool {
	return s.Type == StatementTypeCall || s.Type == StatementTypeExpression
}

// IsAssignment returns true if this statement is a variable assignment.
func (s *Statement) IsAssignment() bool {
	return s.Type == StatementTypeAssignment
}

// IsControlFlow returns true if this statement is a control flow construct.
func (s *Statement) IsControlFlow() bool {
	switch s.Type {
	case StatementTypeIf, StatementTypeFor, StatementTypeWhile, StatementTypeWith, StatementTypeTry:
		return true
	default:
		return false
	}
}

// HasNestedStatements returns true if this statement contains nested statements.
func (s *Statement) HasNestedStatements() bool {
	return len(s.NestedStatements) > 0 || len(s.ElseBranch) > 0
}

// AllStatements returns a flattened list of this statement and all nested statements.
// Performs depth-first traversal.
func (s *Statement) AllStatements() []*Statement {
	result := make([]*Statement, 0, 1+len(s.NestedStatements)+len(s.ElseBranch))
	result = append(result, s)

	for _, nested := range s.NestedStatements {
		result = append(result, nested.AllStatements()...)
	}

	for _, elseBranch := range s.ElseBranch {
		result = append(result, elseBranch.AllStatements()...)
	}

	return result
}

// DefUseChain represents the def-use relationships for all variables in a function.
type DefUseChain struct {
	// Defs maps variable names to all statements that define them.
	// A variable can have multiple definitions across different code paths.
	Defs map[string][]*Statement

	// Uses maps variable names to all statements that use them.
	// A variable can be used in multiple places.
	Uses map[string][]*Statement
}

// NewDefUseChain creates an empty def-use chain.
func NewDefUseChain() *DefUseChain {
	return &DefUseChain{
		Defs: make(map[string][]*Statement),
		Uses: make(map[string][]*Statement),
	}
}

// AddDef registers a statement as defining a variable.
func (chain *DefUseChain) AddDef(varName string, stmt *Statement) {
	if varName == "" {
		return
	}
	chain.Defs[varName] = append(chain.Defs[varName], stmt)
}

// AddUse registers a statement as using a variable.
func (chain *DefUseChain) AddUse(varName string, stmt *Statement) {
	if varName == "" {
		return
	}
	chain.Uses[varName] = append(chain.Uses[varName], stmt)
}

// GetDefs returns all statements that define a given variable.
// Returns empty slice if variable is never defined.
func (chain *DefUseChain) GetDefs(varName string) []*Statement {
	if defs, ok := chain.Defs[varName]; ok {
		return defs
	}
	return []*Statement{}
}

// GetUses returns all statements that use a given variable.
// Returns empty slice if variable is never used.
func (chain *DefUseChain) GetUses(varName string) []*Statement {
	if uses, ok := chain.Uses[varName]; ok {
		return uses
	}
	return []*Statement{}
}

// IsDefined returns true if the variable has at least one definition.
func (chain *DefUseChain) IsDefined(varName string) bool {
	return len(chain.Defs[varName]) > 0
}

// IsUsed returns true if the variable has at least one use.
func (chain *DefUseChain) IsUsed(varName string) bool {
	return len(chain.Uses[varName]) > 0
}

// AllVariables returns a list of all variable names in the def-use chain.
func (chain *DefUseChain) AllVariables() []string {
	varSet := make(map[string]bool)

	for varName := range chain.Defs {
		varSet[varName] = true
	}

	for varName := range chain.Uses {
		varSet[varName] = true
	}

	result := make([]string, 0, len(varSet))
	for varName := range varSet {
		result = append(result, varName)
	}

	return result
}

// BuildDefUseChains constructs a def-use chain from a list of statements.
// This is a single-pass algorithm that builds an inverted index.
//
// Algorithm:
//  1. Initialize empty Defs and Uses maps
//  2. For each statement:
//     - If stmt.Def is not empty: add stmt to Defs[stmt.Def]
//     - For each variable in stmt.Uses: add stmt to Uses[variable]
//  3. Return DefUseChain
//
// Time complexity: O(n × m)
//
//	where n = number of statements
//	      m = average number of uses per statement
//	Typical: 50 statements × 3 variables = 150 operations (~1 microsecond)
//
// Space complexity: O(v × k)
//
//	where v = number of unique variables
//	      k = average number of defs + uses per variable
//	Typical: 20 variables × 5 references = 100 pointers = 800 bytes
//
// Example:
//
//	statements := []*Statement{
//	    {LineNumber: 1, Def: "x", Uses: []string{}},
//	    {LineNumber: 2, Def: "y", Uses: []string{"x"}},
//	    {LineNumber: 3, Def: "", Uses: []string{"y"}},
//	}
//
//	chain := BuildDefUseChains(statements)
//
//	// Query: where is x defined?
//	xDefs := chain.Defs["x"]  // [stmt1]
//
//	// Query: where is x used?
//	xUses := chain.Uses["x"]  // [stmt2]
func BuildDefUseChains(statements []*Statement) *DefUseChain {
	chain := NewDefUseChain()

	// Single pass: build inverted index
	for _, stmt := range statements {
		// Track definition (single variable per statement)
		if stmt.Def != "" {
			chain.AddDef(stmt.Def, stmt)
		}

		// Track all uses in this statement
		for _, varName := range stmt.Uses {
			chain.AddUse(varName, stmt)
		}
	}

	return chain
}

// DefUseStats contains statistics about the def-use chain (for debugging/diagnostics).
type DefUseStats struct {
	NumVariables       int // Total unique variables
	NumDefs            int // Total definition sites
	NumUses            int // Total use sites
	MaxDefsPerVariable int // Most definitions for a single variable
	MaxUsesPerVariable int // Most uses for a single variable
	UndefinedVariables int // Variables used but never defined (parameters)
	DeadVariables      int // Variables defined but never used
}

// ComputeStats computes statistics about this def-use chain.
// Useful for performance analysis and debugging.
//
// Example:
//
//	stats := chain.ComputeStats()
//	fmt.Printf("Function has %d variables, %d defs, %d uses\n",
//	           stats.NumVariables, stats.NumDefs, stats.NumUses)
func (chain *DefUseChain) ComputeStats() DefUseStats {
	stats := DefUseStats{}

	// Count unique variables
	varSet := make(map[string]bool)
	for varName := range chain.Defs {
		varSet[varName] = true
	}
	for varName := range chain.Uses {
		varSet[varName] = true
	}
	stats.NumVariables = len(varSet)

	// Count total defs and max defs per variable
	for _, defs := range chain.Defs {
		stats.NumDefs += len(defs)
		if len(defs) > stats.MaxDefsPerVariable {
			stats.MaxDefsPerVariable = len(defs)
		}
	}

	// Count total uses and max uses per variable
	for _, uses := range chain.Uses {
		stats.NumUses += len(uses)
		if len(uses) > stats.MaxUsesPerVariable {
			stats.MaxUsesPerVariable = len(uses)
		}
	}

	// Count undefined variables (used but not defined)
	for varName := range chain.Uses {
		if len(chain.Defs[varName]) == 0 {
			stats.UndefinedVariables++
		}
	}

	// Count dead variables (defined but not used)
	for varName := range chain.Defs {
		if len(chain.Uses[varName]) == 0 {
			stats.DeadVariables++
		}
	}

	return stats
}
