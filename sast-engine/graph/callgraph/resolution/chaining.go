package resolution

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
)

// ChainStep represents a single step in a method chain.
// For example, in "obj.method1().method2()", there are 2 steps:
//   - Step 1: obj.method1() → returns some type
//   - Step 2: result.method2() → returns some type
type ChainStep struct {
	Expression string         // The full expression for this step (e.g., "create_builder()")
	MethodName string         // Just the method/function name (e.g., "create_builder")
	IsCall     bool           // True if this step is a function call (has parentheses)
	Type       *core.TypeInfo // Resolved type after this step
}

// ParseChain parses a method chain into individual steps.
//
// Examples:
//   - "create_builder().append()" → ["create_builder()", "append()"]
//   - "text.strip().upper().split()" → ["text.strip()", "upper()", "split()"]
//   - "obj.attr.method()" → ["obj.attr.method()"] (not a chain, just nested attribute)
//
// A chain is identified by the pattern "().": a call followed by more method access.
//
// Parameters:
//   - target: the full target string from call site
//
// Returns:
//   - []ChainStep: parsed chain steps, or nil if not a chain
func ParseChain(target string) []ChainStep {
	// Quick check: is this a chain?
	// A chain has the pattern "()." somewhere (call followed by attribute access)
	if !strings.Contains(target, ").") {
		return nil
	}

	var steps []ChainStep
	i := 0

	// Walk through the target string character by character
	for i < len(target) {
		// Find the start of the next call (either start of string or after a dot)
		start := i

		// Find the matching closing parenthesis for this call
		// We need to track nested parentheses and quotes
		parenDepth := 0
		inString := false
		stringChar := rune(0)
		callEnd := -1

		for j := i; j < len(target); j++ {
			ch := rune(target[j])

			// Handle string literals
			if ch == '"' || ch == '\'' {
				if !inString {
					inString = true
					stringChar = ch
				} else if ch == stringChar && (j == 0 || target[j-1] != '\\') {
					inString = false
				}
				continue
			}

			if inString {
				continue
			}

			// Track parentheses depth
			if ch == '(' {
				parenDepth++
			} else if ch == ')' {
				parenDepth--
				if parenDepth == 0 {
					// Found matching close paren
					callEnd = j
					break
				}
			}
		}

		if callEnd == -1 {
			// No closing paren found - take the rest as last step
			step := parseStep(target[start:])
			if step != nil {
				steps = append(steps, *step)
			}
			break
		}

		// Extract this step (up to and including the ")")
		stepExpr := target[start : callEnd+1]
		step := parseStep(stepExpr)
		if step != nil {
			steps = append(steps, *step)
		}

		// Check if there's a dot after the closing paren
		if callEnd+1 < len(target) && target[callEnd+1] == '.' {
			// Move past the "."
			i = callEnd + 2
		} else {
			// No more chain - we're done
			break
		}
	}

	// Only return if we found multiple steps (actual chain)
	if len(steps) <= 1 {
		return nil
	}

	return steps
}

// parseStep parses a single step expression.
func parseStep(expr string) *ChainStep {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}

	step := &ChainStep{
		Expression: expr,
		IsCall:     strings.Contains(expr, "("),
	}

	// Extract method name
	if step.IsCall {
		// Find the opening paren to separate name from arguments
		parenIndex := strings.Index(expr, "(")
		if parenIndex > 0 {
			nameWithPath := expr[:parenIndex]
			// Take the last part after any dots
			parts := strings.Split(nameWithPath, ".")
			step.MethodName = parts[len(parts)-1]
		} else {
			// Malformed - shouldn't happen
			step.MethodName = expr
		}
	} else {
		// For non-call steps, the whole expression is the name
		step.MethodName = expr
	}

	return step
}

// ResolveChainedCall resolves a method chain by walking each step and tracking types.
//
// Algorithm:
//  1. Parse chain into individual steps
//  2. Resolve first step:
//     - If it's a call: resolve as function call, get return type
//     - If it's a variable: look up type in scopes
//  3. For each subsequent step:
//     - Use previous step's type to resolve method
//     - Get method's return type from builtins or return type registry
//     - Track confidence through the chain (multiply confidences)
//  4. Return final type and resolution status
//
// Parameters:
//   - target: the full target string (e.g., "create_builder().append().upper()")
//   - typeEngine: type inference engine with scopes and return types
//   - builtins: builtin registry for builtin method lookups
//   - registry: module registry for validation
//   - codeGraph: code graph for function lookups
//   - callerFQN: FQN of the calling function (for scope lookups)
//   - currentModule: current module path
//   - callGraph: call graph for function lookups
//
// Returns:
//   - targetFQN: the fully qualified name of the final call
//   - resolved: true if chain was successfully resolved
//   - typeInfo: type information for the final result
func ResolveChainedCall(
	target string,
	typeEngine *TypeInferenceEngine,
	builtins *registry.BuiltinRegistry,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
	callerFQN string,
	currentModule string,
	callGraph *core.CallGraph,
) (string, bool, *core.TypeInfo) {
	// Parse the chain
	steps := ParseChain(target)
	if steps == nil || len(steps) == 0 {
		// Not a chain
		return "", false, nil
	}

	// Track current type through the chain
	var currentType *core.TypeInfo
	var lastResolvedFQN string
	confidence := 1.0

	// Resolve each step in the chain
	for i, step := range steps {
		if i == 0 {
			// First step: resolve as variable or function call
			firstType, firstFQN, ok := resolveFirstChainStep(
				step,
				typeEngine,
				callerFQN,
				currentModule,
				moduleRegistry,
				callGraph,
			)
			if !ok {
				// Can't resolve first step - chain fails
				return target, false, nil
			}
			currentType = firstType
			lastResolvedFQN = firstFQN
			if currentType != nil {
				confidence *= float64(currentType.Confidence)
			}
		} else {
			// Subsequent steps: resolve as method on current type
			if currentType == nil {
				// Lost type information - can't continue chain
				return target, false, nil
			}

			methodType, methodFQN, ok := resolveChainMethod(
				step,
				currentType,
				builtins,
				typeEngine,
				moduleRegistry,
				callGraph,
			)
			if !ok {
				// Can't resolve this method - chain fails
				return target, false, nil
			}

			currentType = methodType
			lastResolvedFQN = methodFQN
			if currentType != nil {
				confidence *= float64(currentType.Confidence)
			}
		}
	}

	// Successfully resolved entire chain
	if currentType != nil {
		// Update confidence based on chain length
		finalConfidence := float32(confidence)
		return lastResolvedFQN, true, &core.TypeInfo{
			TypeFQN:    currentType.TypeFQN,
			Confidence: finalConfidence,
			Source:     "method_chain",
		}
	}

	return lastResolvedFQN, true, nil
}

// resolveFirstChainStep resolves the first step in a chain.
// This can be a variable or a function call.
func resolveFirstChainStep(
	step ChainStep,
	typeEngine *TypeInferenceEngine,
	callerFQN string,
	currentModule string,
	_ *core.ModuleRegistry,
	callGraph *core.CallGraph,
) (*core.TypeInfo, string, bool) {
	if step.IsCall {
		// It's a function call - look up return type
		funcName := step.MethodName

		// Try to build FQN for this function
		// First check if it's in the current module
		funcFQN := currentModule + "." + funcName

		// Look up return type
		if typeEngine != nil {
			if returnType, exists := typeEngine.ReturnTypes[funcFQN]; exists {
				// Skip unresolved placeholders - they won't help with chaining
				if !strings.HasPrefix(returnType.TypeFQN, "call:") &&
					!strings.HasPrefix(returnType.TypeFQN, "var:") {
					return returnType, funcFQN, true
				}
			}
		}

		// Try checking call graph functions
		if callGraph != nil {
			if _, exists := callGraph.Functions[funcFQN]; exists {
				// Function exists, but no return type info
				// Look up return type again (might be in registry)
				if typeEngine != nil {
					if returnType, exists := typeEngine.ReturnTypes[funcFQN]; exists {
						// Skip placeholders
						if !strings.HasPrefix(returnType.TypeFQN, "call:") &&
							!strings.HasPrefix(returnType.TypeFQN, "var:") {
							return returnType, funcFQN, true
						}
					}
				}
			}
		}

		// Can't resolve function with concrete type
		return nil, "", false
	}

	// It's a variable - look up in scopes
	varName := step.MethodName

	// Check function scope first
	functionScope := typeEngine.GetScope(callerFQN)
	if functionScope != nil {
		if binding, exists := functionScope.Variables[varName]; exists {
			if binding.Type != nil {
				return binding.Type, varName, true
			}
		}
	}

	// Check module scope
	moduleScope := typeEngine.GetScope(currentModule)
	if moduleScope != nil {
		if binding, exists := moduleScope.Variables[varName]; exists {
			if binding.Type != nil {
				return binding.Type, varName, true
			}
		}
	}

	// Can't resolve variable
	return nil, "", false
}

// resolveChainMethod resolves a method call on a known type.
// This handles both builtin methods and user-defined methods.
func resolveChainMethod(
	step ChainStep,
	currentType *core.TypeInfo,
	builtins *registry.BuiltinRegistry,
	typeEngine *TypeInferenceEngine,
	_ *core.ModuleRegistry,
	callGraph *core.CallGraph,
) (*core.TypeInfo, string, bool) {
	if currentType == nil {
		return nil, "", false
	}

	typeFQN := currentType.TypeFQN
	methodName := step.MethodName

	// Skip unresolved placeholders
	if strings.HasPrefix(typeFQN, "call:") || strings.HasPrefix(typeFQN, "var:") {
		return nil, "", false
	}

	// Check builtin methods first
	if builtins != nil && strings.HasPrefix(typeFQN, "builtins.") {
		method := builtins.GetMethod(typeFQN, methodName)
		if method != nil && method.ReturnType != nil {
			methodFQN := typeFQN + "." + methodName
			return method.ReturnType, methodFQN, true
		}
	}

	// Check user-defined methods
	// For user-defined classes, the method FQN is: module.ClassName.methodName
	// But in Python, methods are stored at module level: module.methodName
	methodFQN := typeFQN + "." + methodName

	// Try Python-style lookup first (module.method instead of module.Class.method)
	// This is more common in Python codebases
	lastDot := strings.LastIndex(typeFQN, ".")
	var pythonMethodFQN string
	if lastDot >= 0 {
		modulePart := typeFQN[:lastDot]
		pythonMethodFQN = modulePart + "." + methodName

		// Check if method exists in call graph
		if callGraph != nil {
			if _, exists := callGraph.Functions[pythonMethodFQN]; exists {
				// Method exists - check for return type
				if typeEngine != nil {
					if returnType, exists := typeEngine.ReturnTypes[pythonMethodFQN]; exists {
						// If return type is unresolved (var:self, call:something), assume fluent interface
						if strings.HasPrefix(returnType.TypeFQN, "var:") || strings.HasPrefix(returnType.TypeFQN, "call:") {
							// Fluent interface pattern - method returns same type
							return &core.TypeInfo{
								TypeFQN:    currentType.TypeFQN,
								Confidence: currentType.Confidence * 0.9, // Slightly reduce confidence
								Source:     "method_chain_fluent",
							}, pythonMethodFQN, true
						}
						// Concrete return type found
						return returnType, pythonMethodFQN, true
					}
				}
				// Method exists but no return type - assume fluent interface
				return &core.TypeInfo{
					TypeFQN:    currentType.TypeFQN,
					Confidence: currentType.Confidence * 0.85,
					Source:     "method_chain_fluent",
				}, pythonMethodFQN, true
			}
		}

		// Try return type lookup even if not in call graph
		if typeEngine != nil {
			if returnType, exists := typeEngine.ReturnTypes[pythonMethodFQN]; exists {
				// Check for unresolved placeholders
				if strings.HasPrefix(returnType.TypeFQN, "var:") || strings.HasPrefix(returnType.TypeFQN, "call:") {
					return &core.TypeInfo{
						TypeFQN:    currentType.TypeFQN,
						Confidence: currentType.Confidence * 0.9,
						Source:     "method_chain_fluent",
					}, pythonMethodFQN, true
				}
				return returnType, pythonMethodFQN, true
			}
		}
	}

	// Try direct method FQN lookup (less common in Python)
	if typeEngine != nil {
		if returnType, exists := typeEngine.ReturnTypes[methodFQN]; exists {
			if strings.HasPrefix(returnType.TypeFQN, "var:") || strings.HasPrefix(returnType.TypeFQN, "call:") {
				return &core.TypeInfo{
					TypeFQN:    currentType.TypeFQN,
					Confidence: currentType.Confidence * 0.9,
					Source:     "method_chain_fluent",
				}, methodFQN, true
			}
			return returnType, methodFQN, true
		}
	}

	// Check if method exists in call graph (direct lookup)
	if callGraph != nil {
		if _, exists := callGraph.Functions[methodFQN]; exists {
			// Method exists - assume fluent interface
			return &core.TypeInfo{
				TypeFQN:    currentType.TypeFQN,
				Confidence: currentType.Confidence * 0.85,
				Source:     "method_chain_fluent",
			}, methodFQN, true
		}
	}

	// Heuristic: if original type has high confidence (>= 0.7) and method looks valid,
	// assume it exists and returns same type (fluent interface pattern)
	if currentType.Confidence >= 0.7 && methodName != "" {
		return &core.TypeInfo{
			TypeFQN:    currentType.TypeFQN,
			Confidence: currentType.Confidence * 0.8,
			Source:     "method_chain_heuristic",
		}, methodFQN, true
	}

	// Can't resolve method
	return nil, "", false
}
