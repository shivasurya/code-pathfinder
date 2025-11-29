package resolution

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Location represents a source code location.
type Location struct {
	File       string // File path
	Line       uint32 // Line number
	Column     uint32 // Column number
	StartByte  uint32 // Starting byte offset
	EndByte    uint32 // Ending byte offset
}

// VariableBinding tracks a variable's type within a scope.
// It captures the variable name, its inferred type, and source location.
type VariableBinding struct {
	VarName      string          // Variable name
	Type         *core.TypeInfo  // Inferred type information
	AssignedFrom string          // FQN of function that assigned this value (if from function call)
	Location     Location        // Source location of the assignment
}

// FunctionScope represents the type environment within a function.
// It tracks variable types and return type for a specific function.
type FunctionScope struct {
	FunctionFQN string                       // Fully qualified name of the function
	Variables   map[string]*VariableBinding  // Variable name -> binding
	ReturnType  *core.TypeInfo               // Inferred return type of the function
}

// NewFunctionScope creates a new function scope with initialized maps.
//
// Parameters:
//   - functionFQN: fully qualified name of the function
//
// Returns:
//   - Initialized FunctionScope
func NewFunctionScope(functionFQN string) *FunctionScope {
	return &FunctionScope{
		FunctionFQN: functionFQN,
		Variables:   make(map[string]*VariableBinding),
	}
}

// AddVariable adds or updates a variable binding in the scope.
//
// Parameters:
//   - binding: the variable binding to add
func (fs *FunctionScope) AddVariable(binding *VariableBinding) {
	if binding != nil && binding.VarName != "" {
		fs.Variables[binding.VarName] = binding
	}
}

// GetVariable retrieves a variable binding by name.
//
// Parameters:
//   - varName: the variable name to look up
//
// Returns:
//   - VariableBinding if found, nil otherwise
func (fs *FunctionScope) GetVariable(varName string) *VariableBinding {
	return fs.Variables[varName]
}

// HasVariable checks if a variable exists in the scope.
//
// Parameters:
//   - varName: the variable name to check
//
// Returns:
//   - true if the variable exists, false otherwise
func (fs *FunctionScope) HasVariable(varName string) bool {
	_, exists := fs.Variables[varName]
	return exists
}
