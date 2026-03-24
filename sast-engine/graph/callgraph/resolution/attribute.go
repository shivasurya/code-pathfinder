package resolution

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
)

// FailureStats tracks why attribute chain resolution fails.
type FailureStats struct {
	TotalAttempts          int
	NotSelfPrefix          int
	DeepChains             int // 3+ levels
	ClassNotFound          int
	AttributeNotFound      int
	MethodNotInBuiltins    int
	CustomClassUnsupported int

	// Pattern samples for analysis
	DeepChainSamples         []string
	AttributeNotFoundSamples []string
	CustomClassSamples       []string
}

var attributeFailureStats = &FailureStats{
	DeepChainSamples:         make([]string, 0, 20),
	AttributeNotFoundSamples: make([]string, 0, 20),
	CustomClassSamples:       make([]string, 0, 20),
}

// maxChainDepth limits the number of intermediate attributes in a chain walk.
// Real-world Python rarely exceeds 4 levels (self.app.db.session.execute).
// This prevents pathological chains from causing excessive work.
const maxChainDepth = 6

// ResolveSelfAttributeCall resolves self.attribute.method() patterns with
// support for arbitrary chain depth (e.g., self.obj.attr.method()).
//
// Algorithm:
//  1. Detect pattern: target starts with "self." and has 2+ dots
//  2. Parse: self.attr₁.attr₂...attrN.method → chain=[attr₁..attrN], method
//  3. Find containing class from callerFQN
//  4. Walk the chain: for each attribute, look up its type and advance
//  5. Resolve the final method on the terminal type
//
// Examples:
//
//	2-level: self.value.upper → chain=["value"], method="upper"
//	3-level: self.core.config.get → chain=["core","config"], method="get"
//	4-level: self.app.db.session.execute → chain=["app","db","session"], method="execute"
//
// Parameters:
//   - target: call target string (e.g., "self.value.upper")
//   - callerFQN: fully qualified name of calling function
//   - typeEngine: type inference engine with attribute registry
//   - builtins: builtin registry for method lookup
//   - callGraph: call graph for class lookup
//
// Returns:
//   - resolvedFQN: fully qualified method name
//   - resolved: true if resolution succeeded
//   - typeInfo: inferred type information
func ResolveSelfAttributeCall(
	target string,
	callerFQN string,
	typeEngine *TypeInferenceEngine,
	builtins *registry.BuiltinRegistry,
	callGraph *core.CallGraph,
) (string, bool, *core.TypeInfo) {
	attributeFailureStats.TotalAttempts++

	// Check if this is a self.attr.method pattern
	if !strings.HasPrefix(target, "self.") {
		attributeFailureStats.NotSelfPrefix++
		return "", false, nil
	}

	// Count dots - need at least 2 (self.attr.method)
	dotCount := strings.Count(target, ".")
	if dotCount < 2 {
		return "", false, nil
	}

	// Parse the pattern: self.attr₁[.attr₂...].method
	parts := strings.Split(target, ".")
	if len(parts) < 3 {
		return "", false, nil
	}

	// Extract attribute chain and final method name.
	// parts[0] = "self", parts[1..n-1] = attribute chain, parts[n] = method
	attrChain := parts[1 : len(parts)-1] // e.g., ["core", "config"]
	methodName := parts[len(parts)-1]    // e.g., "get"

	// Enforce depth limit to prevent pathological chains
	if len(attrChain) > maxChainDepth {
		attributeFailureStats.DeepChains++
		if len(attributeFailureStats.DeepChainSamples) < 20 {
			attributeFailureStats.DeepChainSamples = append(attributeFailureStats.DeepChainSamples, target)
		}
		return "", false, nil
	}

	// Step 1: Find the containing class by checking which classes have this method
	classFQN := findClassContainingMethod(callerFQN, typeEngine.Attributes)
	if classFQN == "" {
		attributeFailureStats.ClassNotFound++
		return "", false, nil
	}

	// Step 2: Walk the attribute chain iteratively.
	// Start from the containing class and resolve each attribute's type.
	currentTypeFQN := classFQN
	var lastAttrConfidence float64
	visited := make(map[string]bool) // Cycle detection

	for _, attrName := range attrChain {
		// Cycle detection: if we've seen this type before, stop
		if visited[currentTypeFQN] {
			attributeFailureStats.AttributeNotFound++
			if len(attributeFailureStats.AttributeNotFoundSamples) < 20 {
				attributeFailureStats.AttributeNotFoundSamples = append(
					attributeFailureStats.AttributeNotFoundSamples,
					fmt.Sprintf("%s (circular ref at type %s)", target, currentTypeFQN))
			}
			return "", false, nil
		}
		visited[currentTypeFQN] = true

		attr := typeEngine.Attributes.GetAttribute(currentTypeFQN, attrName)
		if attr == nil {
			attributeFailureStats.AttributeNotFound++
			if len(attributeFailureStats.AttributeNotFoundSamples) < 20 {
				attributeFailureStats.AttributeNotFoundSamples = append(
					attributeFailureStats.AttributeNotFoundSamples,
					fmt.Sprintf("%s (attr %q not found in %s)", target, attrName, currentTypeFQN))
			}
			return "", false, nil
		}

		if attr.Type == nil {
			attributeFailureStats.AttributeNotFound++
			return "", false, nil
		}

		lastAttrConfidence = attr.Confidence
		currentTypeFQN = attr.Type.TypeFQN

		// Resolve placeholder types like "class:Config" inline
		if strings.HasPrefix(currentTypeFQN, "class:") {
			className := strings.TrimPrefix(currentTypeFQN, "class:")
			resolved := resolveClassNameForChain(className, classFQN, typeEngine, callGraph)
			if resolved != "" {
				currentTypeFQN = resolved
			}
			// If unresolved, continue with the placeholder — it may still match
		}
	}

	// Step 3: Resolve the final method on the terminal type
	return resolveMethodOnType(currentTypeFQN, methodName, lastAttrConfidence, builtins, callGraph)
}

// resolveMethodOnType resolves a method call on a given type FQN.
// Checks builtin registry first, then custom class methods in the call graph.
func resolveMethodOnType(
	typeFQN string,
	methodName string,
	attrConfidence float64,
	builtins *registry.BuiltinRegistry,
	callGraph *core.CallGraph,
) (string, bool, *core.TypeInfo) {
	// Check if it's a builtin type
	if strings.HasPrefix(typeFQN, "builtins.") {
		methodFQN := typeFQN + "." + methodName

		// Verify method exists in builtin registry
		method := builtins.GetMethod(typeFQN, methodName)
		if method != nil && method.ReturnType != nil {
			return methodFQN, true, &core.TypeInfo{
				TypeFQN:    method.ReturnType.TypeFQN,
				Confidence: float32(attrConfidence),
				Source:     "self_attribute",
			}
		}

		attributeFailureStats.MethodNotInBuiltins++
		return "", false, nil
	}

	// Handle custom class types (user-defined classes).
	methodFQN := typeFQN + "." + methodName

	if callGraph != nil {
		// Exact lookup first
		if node := callGraph.Functions[methodFQN]; node != nil {
			if isCallableNode(node) {
				return methodFQN, true, &core.TypeInfo{
					TypeFQN:    typeFQN,
					Confidence: float32(attrConfidence),
					Source:     "self_attribute_custom_class",
				}
			}
		}

		// Suffix fallback: handles FQN mismatches from relative imports.
		// The attribute registry may store "config.parser.ConfigParser" while the
		// callgraph stores "myapp.config.parser.ConfigParser" (with full module prefix).
		// We match on "ClassName.method" suffix to bridge this gap.
		suffix := extractClassMethodSuffix(typeFQN, methodName)
		if suffix != "" {
			for fqn, node := range callGraph.Functions {
				if strings.HasSuffix(fqn, "."+suffix) && isCallableNode(node) {
					// Use the callgraph's FQN (the authoritative one)
					resolvedTypeFQN := strings.TrimSuffix(fqn, "."+methodName)
					return fqn, true, &core.TypeInfo{
						TypeFQN:    resolvedTypeFQN,
						Confidence: float32(attrConfidence * 0.85), // slight penalty for fuzzy match
						Source:     "self_attribute_custom_class",
					}
				}
			}
		}
	}

	// Method not found — collect stats
	attributeFailureStats.CustomClassUnsupported++
	if len(attributeFailureStats.CustomClassSamples) < 20 {
		attributeFailureStats.CustomClassSamples = append(
			attributeFailureStats.CustomClassSamples,
			fmt.Sprintf("method %s not found on type %s", methodName, typeFQN))
	}
	return "", false, nil
}

// isCallableNode checks if a graph node represents a callable symbol.
func isCallableNode(node *graph.Node) bool {
	return node != nil && (node.Type == "method" || node.Type == "function_definition" ||
		node.Type == "constructor" || node.Type == "property" ||
		node.Type == "special_method")
}

// extractClassMethodSuffix extracts "ClassName.method" from a full type FQN.
// e.g., "config.parser.ConfigParser" + "get" → "ConfigParser.get".
func extractClassMethodSuffix(typeFQN, methodName string) string {
	lastDot := strings.LastIndex(typeFQN, ".")
	if lastDot == -1 {
		// typeFQN is just a class name (no module prefix)
		return typeFQN + "." + methodName
	}
	className := typeFQN[lastDot+1:]
	return className + "." + methodName
}

// resolveClassNameForChain resolves a "class:ClassName" placeholder during chain walking.
// Uses ImportMap, same-module lookup, and module registry (same as ResolveAttributePlaceholders).
func resolveClassNameForChain(
	className string,
	contextClassFQN string,
	typeEngine *TypeInferenceEngine,
	callGraph *core.CallGraph,
) string {
	if typeEngine == nil {
		return ""
	}

	// Try resolving via the existing resolveClassName (uses ImportMap, same-module, short names)
	modulePath := getModuleFromClassFQN(contextClassFQN)
	candidateFQN := modulePath + "." + className

	// Check call graph for the class — if any function key starts with candidateFQN+".",
	// the class exists in the codebase.
	if callGraph != nil {
		prefix := candidateFQN + "."
		for fqn := range callGraph.Functions {
			if strings.HasPrefix(fqn, prefix) {
				return candidateFQN
			}
		}
	}

	// Check attribute registry — if the class has registered attributes, it exists
	if typeEngine.Attributes != nil && typeEngine.Attributes.HasClass(candidateFQN) {
		return candidateFQN
	}

	return ""
}

// PrintAttributeFailureStats prints detailed statistics about attribute chain failures.
// Only prints if debug mode is enabled via the provided logger.
func PrintAttributeFailureStats(logger interface{ IsDebug() bool }) {
	// Don't print if not in debug mode or no attempts made
	if logger != nil && !logger.IsDebug() {
		return
	}
	if attributeFailureStats.TotalAttempts == 0 {
		return
	}

	fmt.Printf("\n[ATTR_FAILURE_ANALYSIS] Self-Attribute Resolution Attempts\n")
	fmt.Printf("========================================================\n")
	fmt.Printf("Total attempts:              %d\n", attributeFailureStats.TotalAttempts)
	fmt.Printf("\nFailure Breakdown:\n")
	fmt.Printf("  Not self prefix:           %d (%.1f%%)\n",
		attributeFailureStats.NotSelfPrefix,
		float64(attributeFailureStats.NotSelfPrefix)*100/float64(attributeFailureStats.TotalAttempts))
	fmt.Printf("  Deep chains (3+ levels):   %d (%.1f%%)\n",
		attributeFailureStats.DeepChains,
		float64(attributeFailureStats.DeepChains)*100/float64(attributeFailureStats.TotalAttempts))
	fmt.Printf("  Class not found:           %d (%.1f%%)\n",
		attributeFailureStats.ClassNotFound,
		float64(attributeFailureStats.ClassNotFound)*100/float64(attributeFailureStats.TotalAttempts))
	fmt.Printf("  Attribute not found:       %d (%.1f%%)\n",
		attributeFailureStats.AttributeNotFound,
		float64(attributeFailureStats.AttributeNotFound)*100/float64(attributeFailureStats.TotalAttempts))
	fmt.Printf("  Method not in builtins:    %d (%.1f%%)\n",
		attributeFailureStats.MethodNotInBuiltins,
		float64(attributeFailureStats.MethodNotInBuiltins)*100/float64(attributeFailureStats.TotalAttempts))
	fmt.Printf("  Custom class unsupported:  %d (%.1f%%)\n",
		attributeFailureStats.CustomClassUnsupported,
		float64(attributeFailureStats.CustomClassUnsupported)*100/float64(attributeFailureStats.TotalAttempts))

	// Print sample patterns
	if len(attributeFailureStats.DeepChainSamples) > 0 {
		fmt.Printf("\nDeep chain samples (first 10):\n")
		for i, sample := range attributeFailureStats.DeepChainSamples {
			if i >= 10 {
				break
			}
			fmt.Printf("  - %s\n", sample)
		}
	}

	if len(attributeFailureStats.AttributeNotFoundSamples) > 0 {
		fmt.Printf("\nAttribute not found samples (first 10):\n")
		for i, sample := range attributeFailureStats.AttributeNotFoundSamples {
			if i >= 10 {
				break
			}
			fmt.Printf("  - %s\n", sample)
		}
	}

	if len(attributeFailureStats.CustomClassSamples) > 0 {
		fmt.Printf("\nCustom class samples (first 10):\n")
		for i, sample := range attributeFailureStats.CustomClassSamples {
			if i >= 10 {
				break
			}
			fmt.Printf("  - %s\n", sample)
		}
	}

	fmt.Printf("========================================================\n\n")
}

// findClassContainingMethod finds which class contains a given method
// Since Python function FQNs don't include the class name, we need to search
// the attribute registry to find which class has this method in its Methods list.
//
// The caller FQN is "module.method" (e.g., "test_chaining.upper")
// The Methods list has "module.ClassName.method" (e.g., "test_chaining.StringBuilder.upper")
//
// We need to match by checking if the class's method ends with the caller's method.
//
// Parameters:
//   - methodFQN: fully qualified method name without class (e.g., "test_chaining.upper")
//   - registry: attribute registry with class information
//
// Returns:
//   - class FQN if found, empty string otherwise
func findClassContainingMethod(methodFQN string, registry *registry.AttributeRegistry) string {
	// Extract method name from FQN (last part after final dot)
	methodName := methodFQN
	if lastDot := strings.LastIndex(methodFQN, "."); lastDot != -1 {
		methodName = methodFQN[lastDot+1:]
	}

	// Iterate over all classes and check if they have this method
	for _, classFQN := range registry.GetAllClasses() {
		classAttrs := registry.GetClassAttributes(classFQN)
		if classAttrs == nil {
			continue
		}

		// Check if this method is in the class's method list
		// Methods in the list are fully qualified with class name
		expectedMethodFQN := classFQN + "." + methodName
		if slices.Contains(classAttrs.Methods, expectedMethodFQN) {
			return classFQN
		}
	}

	return ""
}

// ResolveAttributePlaceholders resolves placeholder types in the attribute registry
// Placeholders are created during extraction when we can't determine the exact type:
//   - class:User → resolve to fully qualified class name
//   - call:calculate → resolve to function return type
//   - param:User → resolve to fully qualified class name
//
// This is Pass 3 of the attribute extraction algorithm.
//
// Parameters:
//   - registry: attribute registry with placeholder types
//   - typeEngine: type inference engine with return types
//   - moduleRegistry: module registry for resolving class names
//   - codeGraph: code graph for finding class definitions
func ResolveAttributePlaceholders(
	registry *registry.AttributeRegistry,
	typeEngine *TypeInferenceEngine,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
) {
	// Iterate over all classes and attributes
	for _, classFQN := range registry.GetAllClasses() {
		classAttrs := registry.GetClassAttributes(classFQN)
		if classAttrs == nil {
			continue
		}

		for attrName, attr := range classAttrs.Attributes {
			if attr.Type == nil {
				continue
			}

			originalType := attr.Type.TypeFQN

			// Resolve placeholder types
			switch {
			case strings.HasPrefix(originalType, "class:"):
				// class:User → try to resolve to full FQN
				className := strings.TrimPrefix(originalType, "class:")
				resolvedFQN := resolveClassName(className, classFQN, moduleRegistry, codeGraph, classAttrs.FilePath, typeEngine)
				if resolvedFQN != "" {
					attr.Type.TypeFQN = resolvedFQN
					attr.Type.Confidence = 0.9 // High confidence for resolved classes
				}
			case strings.HasPrefix(originalType, "call:"):
				// call:func → lookup return type
				funcName := strings.TrimPrefix(originalType, "call:")
				// Try to find function in same module
				modulePath := getModuleFromClassFQN(classFQN)
				funcFQN := modulePath + "." + funcName

				if returnType, exists := typeEngine.GetReturnType(funcFQN); exists && returnType != nil {
					attr.Type.TypeFQN = returnType.TypeFQN
					attr.Type.Confidence = returnType.Confidence * 0.8 // Decay confidence
					attr.Type.Source = "function_call_attribute"
				}
			case strings.HasPrefix(originalType, "param:"):
				// param:User → resolve type annotation
				typeName := strings.TrimPrefix(originalType, "param:")
				resolvedFQN := resolveClassName(typeName, classFQN, moduleRegistry, codeGraph, classAttrs.FilePath, typeEngine)
				if resolvedFQN != "" {
					attr.Type.TypeFQN = resolvedFQN
					attr.Type.Confidence = 0.95 // Very high confidence for annotations
				}
			}

			// Update attribute with resolved type
			classAttrs.Attributes[attrName] = attr
		}
	}
}

// resolveClassName resolves a class name to its fully qualified name.
// Uses module registry, code graph, and ImportMap to find the class definition.
//
// P0 Fix: Now uses ImportMap (if available) to resolve imported class names correctly.
func resolveClassName(
	className string,
	contextClassFQN string,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
	filePath string,
	typeEngine *TypeInferenceEngine,
) string {
	// P0 Fix: Try ImportMap first (most accurate for imported classes)
	if filePath != "" && typeEngine != nil {
		if importMap := typeEngine.GetImportMap(filePath); importMap != nil {
			if resolvedFQN, ok := importMap.Resolve(className); ok {
				return resolvedFQN
			}
		}
	}

	// Get the module of the context class
	modulePath := getModuleFromClassFQN(contextClassFQN)

	// Try same module next
	candidateFQN := modulePath + "." + className
	if classExists(candidateFQN, codeGraph) {
		return candidateFQN
	}

	// Try short name lookup in module registry
	if paths, ok := moduleRegistry.ShortNames[className]; ok && len(paths) > 0 {
		// Use first match (could be improved with import analysis)
		filePathMatch := paths[0]
		if modulePath, ok := moduleRegistry.FileToModule[filePathMatch]; ok {
			return modulePath + "." + className
		}
	}

	// Not found
	return ""
}

// classExists checks if a class exists in the code graph.
func classExists(classFQN string, codeGraph *graph.CodeGraph) bool {
	// Look for class_declaration nodes with matching name
	for _, node := range codeGraph.Nodes {
		if node.Type == "class_declaration" && node.Name == classFQN {
			return true
		}
	}
	return false
}

// getModuleFromClassFQN extracts the module path from a class FQN
// Examples:
//
//	test_chaining.StringBuilder → test_chaining
//	myapp.models.User → myapp.models
func getModuleFromClassFQN(classFQN string) string {
	parts := strings.Split(classFQN, ".")
	if len(parts) < 2 {
		return classFQN
	}
	// Return all but the last part (class name)
	return strings.Join(parts[:len(parts)-1], ".")
}
