package resolution

import (
	"maps"
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// GoVariableBinding represents a variable's type information at a specific assignment.
// Multiple bindings can exist for the same variable (reassignment tracking).
//
// Example:
//
//	user := GetUser(123)  // Creates binding with type from GetUser's return type
//
// Supports reassignment:
//
//	user := GetUser(1)    // Binding 1
//	user = NewUser()      // Binding 2 (latest)
type GoVariableBinding struct {
	// Variable name (e.g., "user", "config", "result")
	VarName string

	// Inferred type information
	Type *core.TypeInfo

	// FQN of function that assigned this value, or "literal" for constants
	AssignedFrom string

	// Source location of assignment
	Location Location
}

// GoFunctionScope tracks variable type bindings within a single function.
// Variables can have multiple bindings due to reassignment - latest binding
// is always at the end of the slice.
//
// Example:
//
//	scope := NewGoFunctionScope("github.com/myapp/handlers.HandleRequest")
//	scope.AddVariable(&GoVariableBinding{VarName: "user", ...})
//	binding := scope.GetVariable("user")  // Returns latest binding
type GoFunctionScope struct {
	// Function FQN (e.g., "github.com/myapp/handlers.HandleRequest")
	FunctionFQN string

	// Variable name → bindings (multiple bindings for reassignment)
	// Latest binding is always last in the slice
	Variables map[string][]*GoVariableBinding
}

// NewGoFunctionScope creates a new function scope.
func NewGoFunctionScope(functionFQN string) *GoFunctionScope {
	return &GoFunctionScope{
		FunctionFQN: functionFQN,
		Variables:   make(map[string][]*GoVariableBinding),
	}
}

// AddVariable adds a variable binding to the scope.
// Supports multiple bindings for reassignment (latest is last in slice).
func (s *GoFunctionScope) AddVariable(binding *GoVariableBinding) {
	if binding == nil {
		return
	}
	s.Variables[binding.VarName] = append(s.Variables[binding.VarName], binding)
}

// GetVariable retrieves the latest binding for a variable.
// Returns nil if variable not found.
func (s *GoFunctionScope) GetVariable(varName string) *GoVariableBinding {
	bindings := s.Variables[varName]
	if len(bindings) == 0 {
		return nil
	}
	// Return latest binding (last in slice)
	return bindings[len(bindings)-1]
}

// HasVariable checks if a variable exists in the scope.
func (s *GoFunctionScope) HasVariable(varName string) bool {
	return len(s.Variables[varName]) > 0
}

// GetAllBindings returns all bindings for a variable (for reassignment analysis).
// Useful for debugging and understanding variable evolution.
func (s *GoFunctionScope) GetAllBindings(varName string) []*GoVariableBinding {
	return s.Variables[varName]
}

// GoTypeInferenceEngine manages type information for Go code.
// Thread-safe implementation for parallel extraction.
//
// Architecture:
//   - Scopes: Map function FQN → GoFunctionScope (per-function variable tracking)
//   - ReturnTypes: Map function FQN → TypeInfo (return type for each function)
//   - Registry: Go module registry for resolving import paths
//
// Thread Safety:
//
//	All public methods use RWMutex for safe concurrent access during parallel
//	file processing in Pass 2a and Pass 2b.
//
// Example:
//
//	engine := NewGoTypeInferenceEngine(registry)
//	engine.AddReturnType("myapp.GetUser", &core.TypeInfo{...})
//	scope := NewGoFunctionScope("myapp.HandleRequest")
//	engine.AddScope(scope)
type GoTypeInferenceEngine struct {
	// Function FQN → variable scopes
	Scopes map[string]*GoFunctionScope

	// Function FQN → return type
	ReturnTypes map[string]*core.TypeInfo

	// Go module registry (from Phase 1)
	Registry *core.GoModuleRegistry

	// Thread-safe access
	scopeMutex sync.RWMutex
	typeMutex  sync.RWMutex
}

// NewGoTypeInferenceEngine creates an initialized type inference engine.
func NewGoTypeInferenceEngine(registry *core.GoModuleRegistry) *GoTypeInferenceEngine {
	return &GoTypeInferenceEngine{
		Scopes:      make(map[string]*GoFunctionScope),
		ReturnTypes: make(map[string]*core.TypeInfo),
		Registry:    registry,
	}
}

// ===== Scope Management =====

// GetScope retrieves a function scope (thread-safe read).
// Returns nil if scope not found.
func (e *GoTypeInferenceEngine) GetScope(functionFQN string) *GoFunctionScope {
	e.scopeMutex.RLock()
	defer e.scopeMutex.RUnlock()
	return e.Scopes[functionFQN]
}

// AddScope stores a function scope (thread-safe write).
// Ignores nil scopes.
func (e *GoTypeInferenceEngine) AddScope(scope *GoFunctionScope) {
	if scope == nil {
		return
	}
	e.scopeMutex.Lock()
	defer e.scopeMutex.Unlock()
	e.Scopes[scope.FunctionFQN] = scope
}

// HasScope checks if a scope exists for a function.
func (e *GoTypeInferenceEngine) HasScope(functionFQN string) bool {
	e.scopeMutex.RLock()
	defer e.scopeMutex.RUnlock()
	_, exists := e.Scopes[functionFQN]
	return exists
}

// GetAllScopes returns all function scopes (for testing/debugging).
// Returns a copy to prevent external modification.
func (e *GoTypeInferenceEngine) GetAllScopes() map[string]*GoFunctionScope {
	e.scopeMutex.RLock()
	defer e.scopeMutex.RUnlock()

	scopes := make(map[string]*GoFunctionScope)
	maps.Copy(scopes, e.Scopes)
	return scopes
}

// ===== Return Type Management =====

// GetReturnType retrieves return type for a function (thread-safe read).
// Returns (typeInfo, true) if found, (nil, false) otherwise.
func (e *GoTypeInferenceEngine) GetReturnType(functionFQN string) (*core.TypeInfo, bool) {
	e.typeMutex.RLock()
	defer e.typeMutex.RUnlock()
	typeInfo, ok := e.ReturnTypes[functionFQN]
	return typeInfo, ok
}

// AddReturnType stores return type for a function (thread-safe write).
// Ignores nil type info.
func (e *GoTypeInferenceEngine) AddReturnType(functionFQN string, typeInfo *core.TypeInfo) {
	if typeInfo == nil {
		return
	}
	e.typeMutex.Lock()
	defer e.typeMutex.Unlock()
	e.ReturnTypes[functionFQN] = typeInfo
}

// HasReturnType checks if a return type exists for a function.
func (e *GoTypeInferenceEngine) HasReturnType(functionFQN string) bool {
	e.typeMutex.RLock()
	defer e.typeMutex.RUnlock()
	_, exists := e.ReturnTypes[functionFQN]
	return exists
}

// GetAllReturnTypes returns all return types (for testing/debugging).
// Returns a copy to prevent external modification.
func (e *GoTypeInferenceEngine) GetAllReturnTypes() map[string]*core.TypeInfo {
	e.typeMutex.RLock()
	defer e.typeMutex.RUnlock()

	types := make(map[string]*core.TypeInfo)
	maps.Copy(types, e.ReturnTypes)
	return types
}
