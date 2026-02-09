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
// Variables stores multiple bindings per variable name to support reassignment tracking.
type FunctionScope struct {
	FunctionFQN string                         // Fully qualified name of the function
	Variables   map[string][]*VariableBinding  // Variable name -> bindings (per-assignment)
	ReturnType  *core.TypeInfo                 // Inferred return type of the function
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
		Variables:   make(map[string][]*VariableBinding),
	}
}

// AddVariable appends a variable binding in the scope.
// Multiple bindings per variable name are preserved for reassignment tracking.
//
// Parameters:
//   - binding: the variable binding to add
func (fs *FunctionScope) AddVariable(binding *VariableBinding) {
	if binding != nil && binding.VarName != "" {
		fs.Variables[binding.VarName] = append(fs.Variables[binding.VarName], binding)
	}
}

// GetVariable retrieves the last variable binding by name.
// Returns the most recent binding, which preserves backward compatibility
// for callers that expect a single binding per variable.
//
// Parameters:
//   - varName: the variable name to look up
//
// Returns:
//   - Last VariableBinding if found, nil otherwise
func (fs *FunctionScope) GetVariable(varName string) *VariableBinding {
	bindings := fs.Variables[varName]
	if len(bindings) == 0 {
		return nil
	}
	return bindings[len(bindings)-1]
}

// GetVariableAtLine retrieves a variable binding at a specific line.
// Used for line-aware type lookup (e.g., when a variable is reassigned with different types).
//
// Parameters:
//   - varName: the variable name to look up
//   - line: the line number to match
//
// Returns:
//   - VariableBinding at the specified line, nil if not found
func (fs *FunctionScope) GetVariableAtLine(varName string, line uint32) *VariableBinding {
	bindings := fs.Variables[varName]
	for _, binding := range bindings {
		if binding != nil && binding.Location.Line == line {
			return binding
		}
	}
	return nil
}

// HasVariable checks if a variable exists in the scope.
//
// Parameters:
//   - varName: the variable name to check
//
// Returns:
//   - true if the variable exists, false otherwise
func (fs *FunctionScope) HasVariable(varName string) bool {
	bindings := fs.Variables[varName]
	return len(bindings) > 0
}
