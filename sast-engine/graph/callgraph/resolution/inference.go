package resolution

import (
	"strings"
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
)

// TypeInferenceEngine manages type inference across the codebase.
// It maintains function scopes, return types, and references to other registries.
// Thread-safe for concurrent access via mutex protection.
type TypeInferenceEngine struct {
	Scopes         map[string]*FunctionScope    // Function FQN -> scope
	ReturnTypes    map[string]*core.TypeInfo    // Function FQN -> return type
	Builtins       *registry.BuiltinRegistry    // Builtin types registry
	Registry       *core.ModuleRegistry         // Module registry reference
	Attributes     *registry.AttributeRegistry  // Class attributes registry (Phase 3 Task 12)
	StdlibRegistry *core.StdlibRegistry         // Python stdlib registry (PR #2)
	StdlibRemote   interface{}                  // Remote loader for lazy module loading (PR #3)
	scopeMutex     sync.RWMutex                 // Protects Scopes map for concurrent access
	typeMutex      sync.RWMutex                 // Protects ReturnTypes map for concurrent access
}

// StdlibRegistryRemote will be defined in registry package.
// For now, use an interface or accept nil.
type StdlibRegistryRemote interface{}

// NewTypeInferenceEngine creates a new type inference engine.
// The engine is initialized with empty scopes and return types.
//
// Parameters:
//   - registry: module registry for resolving module paths
//
// Returns:
//   - Initialized TypeInferenceEngine
func NewTypeInferenceEngine(registry *core.ModuleRegistry) *TypeInferenceEngine {
	return &TypeInferenceEngine{
		Scopes:      make(map[string]*FunctionScope),
		ReturnTypes: make(map[string]*core.TypeInfo),
		Registry:    registry,
	}
}

// GetScope retrieves a function scope by its fully qualified name.
// Thread-safe for concurrent reads.
//
// Parameters:
//   - functionFQN: fully qualified name of the function
//
// Returns:
//   - FunctionScope if found, nil otherwise
func (te *TypeInferenceEngine) GetScope(functionFQN string) *FunctionScope {
	te.scopeMutex.RLock()
	defer te.scopeMutex.RUnlock()
	return te.Scopes[functionFQN]
}

// AddScope adds or updates a function scope in the engine.
// Thread-safe for concurrent writes.
//
// Parameters:
//   - scope: the function scope to add
func (te *TypeInferenceEngine) AddScope(scope *FunctionScope) {
	if scope != nil {
		te.scopeMutex.Lock()
		defer te.scopeMutex.Unlock()
		te.Scopes[scope.FunctionFQN] = scope
	}
}

// GetReturnType retrieves a function's return type.
// Thread-safe for concurrent reads.
//
// Parameters:
//   - functionFQN: fully qualified name of the function
//
// Returns:
//   - TypeInfo if found, nil otherwise
//   - bool indicating whether the type was found
func (te *TypeInferenceEngine) GetReturnType(functionFQN string) (*core.TypeInfo, bool) {
	te.typeMutex.RLock()
	defer te.typeMutex.RUnlock()
	typeInfo, ok := te.ReturnTypes[functionFQN]
	return typeInfo, ok
}

// ResolveVariableType resolves the type of a variable assignment from a function call.
// It looks up the return type of the called function and propagates it with confidence decay.
// Thread-safe for concurrent reads.
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
) *core.TypeInfo {
	// Look up return type of the function (thread-safe read)
	returnType, ok := te.GetReturnType(assignedFrom)
	if !ok {
		return nil
	}

	// Reduce confidence slightly for propagation
	propagatedConfidence := returnType.Confidence * confidence * 0.95

	return &core.TypeInfo{
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
