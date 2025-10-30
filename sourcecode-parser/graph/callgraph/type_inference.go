package callgraph

// TypeInfo represents inferred type information for a variable or expression.
// It tracks the fully qualified type name, confidence level, and how the type was inferred.
type TypeInfo struct {
	TypeFQN    string  // Fully qualified type name (e.g., "builtins.str", "myapp.models.User")
	Confidence float32 // Confidence level from 0.0 to 1.0 (1.0 = certain, 0.5 = heuristic, 0.0 = unknown)
	Source     string  // How the type was inferred (e.g., "literal", "assignment", "annotation")
}

// VariableBinding tracks a variable's type within a scope.
// It captures the variable name, its inferred type, and source location.
type VariableBinding struct {
	VarName      string    // Variable name
	Type         *TypeInfo // Inferred type information
	AssignedFrom string    // FQN of function that assigned this value (if from function call)
	Location     Location  // Source location of the assignment
}

// FunctionScope represents the type environment within a function.
// It tracks variable types and return type for a specific function.
type FunctionScope struct {
	FunctionFQN string                       // Fully qualified name of the function
	Variables   map[string]*VariableBinding  // Variable name -> binding
	ReturnType  *TypeInfo                    // Inferred return type of the function
}

// TypeInferenceEngine manages type inference across the codebase.
// It maintains function scopes, return types, and references to other registries.
type TypeInferenceEngine struct {
	Scopes      map[string]*FunctionScope // Function FQN -> scope
	ReturnTypes map[string]*TypeInfo      // Function FQN -> return type
	Builtins    *BuiltinRegistry          // Builtin types registry
	Registry    *ModuleRegistry           // Module registry reference
}

// NewTypeInferenceEngine creates a new type inference engine.
// The engine is initialized with empty scopes and return types.
//
// Parameters:
//   - registry: module registry for resolving module paths
//
// Returns:
//   - Initialized TypeInferenceEngine
func NewTypeInferenceEngine(registry *ModuleRegistry) *TypeInferenceEngine {
	return &TypeInferenceEngine{
		Scopes:      make(map[string]*FunctionScope),
		ReturnTypes: make(map[string]*TypeInfo),
		Registry:    registry,
	}
}

// GetScope retrieves a function scope by its fully qualified name.
//
// Parameters:
//   - functionFQN: fully qualified name of the function
//
// Returns:
//   - FunctionScope if found, nil otherwise
func (te *TypeInferenceEngine) GetScope(functionFQN string) *FunctionScope {
	return te.Scopes[functionFQN]
}

// AddScope adds or updates a function scope in the engine.
//
// Parameters:
//   - scope: the function scope to add
func (te *TypeInferenceEngine) AddScope(scope *FunctionScope) {
	if scope != nil {
		te.Scopes[scope.FunctionFQN] = scope
	}
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
