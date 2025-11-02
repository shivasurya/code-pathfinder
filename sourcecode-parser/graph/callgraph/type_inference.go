package callgraph

import "strings"

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
	Scopes         map[string]*FunctionScope  // Function FQN -> scope
	ReturnTypes    map[string]*TypeInfo       // Function FQN -> return type
	Builtins       *BuiltinRegistry           // Builtin types registry
	Registry       *ModuleRegistry            // Module registry reference
	Attributes     *AttributeRegistry         // Class attributes registry (Phase 3 Task 12)
	StdlibRegistry *StdlibRegistry            // Python stdlib registry (PR #2)
	StdlibRemote   *StdlibRegistryRemote      // Remote loader for lazy module loading (PR #3)
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

// ResolveVariableType resolves the type of a variable assignment from a function call.
// It looks up the return type of the called function and propagates it with confidence decay.
//
// Parameters:
//   - assignedFrom: Function FQN that was called
//   - confidence: Base confidence from assignment
//
// Returns:
//   - TypeInfo with propagated type, or nil if function has no return type
func (te *TypeInferenceEngine) ResolveVariableType(
	assignedFrom string,
	confidence float32,
) *TypeInfo {
	// Look up return type of the function
	returnType, ok := te.ReturnTypes[assignedFrom]
	if !ok {
		return nil
	}

	// Reduce confidence slightly for propagation
	propagatedConfidence := returnType.Confidence * confidence * 0.95

	return &TypeInfo{
		TypeFQN:    returnType.TypeFQN,
		Confidence: propagatedConfidence,
		Source:     "function_call_propagation",
	}
}

// UpdateVariableBindingsWithFunctionReturns resolves "call:funcName" placeholders.
// It iterates through all scopes and replaces placeholder types with actual return types.
//
// This enables inter-procedural type propagation:
//   user = create_user()  # Initially typed as "call:create_user"
//   # After update, typed as "test.User" based on create_user's return type
func (te *TypeInferenceEngine) UpdateVariableBindingsWithFunctionReturns() {
	for _, scope := range te.Scopes {
		for varName, binding := range scope.Variables {
			if binding.Type != nil && strings.HasPrefix(binding.Type.TypeFQN, "call:") {
				// Extract function name from "call:funcName"
				funcName := strings.TrimPrefix(binding.Type.TypeFQN, "call:")

				// Build FQN for the function call
				var funcFQN string

				// Check if funcName already contains dots (e.g., "logging.getLogger", "MySerializer")
				if strings.Contains(funcName, ".") {
					// Already qualified (e.g., imported module.function)
					// Try as-is first
					funcFQN = funcName
				} else {
					// Simple name - need to qualify with current scope
					lastDotIndex := strings.LastIndex(scope.FunctionFQN, ".")
					if lastDotIndex >= 0 {
						// Function scope: strip function name, add called function
						funcFQN = scope.FunctionFQN[:lastDotIndex+1] + funcName
					} else {
						// Module-level scope
						modulePath := scope.FunctionFQN
						funcFQN = modulePath + "." + funcName
					}
				}

				// Resolve type
				resolvedType := te.ResolveVariableType(funcFQN, binding.Type.Confidence)
				if resolvedType != nil {
					scope.Variables[varName].Type = resolvedType
					scope.Variables[varName].AssignedFrom = funcFQN
				}
			}
		}
	}
}
