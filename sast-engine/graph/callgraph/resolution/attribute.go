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

// ResolveSelfAttributeCall resolves self.attribute.method() patterns
// This is the core of Phase 3 Task 12 - using extracted attributes to resolve calls.
//
// Algorithm:
//  1. Detect pattern: target starts with "self." and has 2+ dots
//  2. Parse: self.attr.method → attr="attr", method="method"
//  3. Find containing class from callerFQN
//  4. Lookup attribute type in AttributeRegistry
//  5. Resolve method on inferred type
//
// Example:
//
//	Input: self.value.upper (caller: test_chaining.StringBuilder.process)
//	Steps:
//	  1. Parse → attr="value", method="upper"
//	  2. Extract class → test_chaining.StringBuilder
//	  3. Lookup value type → builtins.str
//	  4. Resolve upper on str → builtins.str.upper
//	Output: (builtins.str.upper, true, TypeInfo{builtins.str, 1.0, "self_attribute"})
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

	// Parse the pattern: self.attr.method or self.attr.subattr.method
	parts := strings.Split(target, ".")
	if len(parts) < 3 {
		return "", false, nil
	}

	// For now, handle simple case: self.attr.method (2 levels)
	// TODO: Handle deep chains like self.obj.attr.method
	if len(parts) > 3 {
		attributeFailureStats.DeepChains++
		if len(attributeFailureStats.DeepChainSamples) < 20 {
			attributeFailureStats.DeepChainSamples = append(attributeFailureStats.DeepChainSamples, target)
		}
		return "", false, nil
	}

	attrName := parts[1]
	methodName := parts[2]

	// Step 1: Find the containing class by checking which classes have this method
	classFQN := findClassContainingMethod(callerFQN, typeEngine.Attributes)
	if classFQN == "" {
		attributeFailureStats.ClassNotFound++
		return "", false, nil
	}

	// Step 2: Lookup attribute in AttributeRegistry
	attr := typeEngine.Attributes.GetAttribute(classFQN, attrName)
	if attr == nil {
		attributeFailureStats.AttributeNotFound++
		if len(attributeFailureStats.AttributeNotFoundSamples) < 20 {
			attributeFailureStats.AttributeNotFoundSamples = append(
				attributeFailureStats.AttributeNotFoundSamples,
				fmt.Sprintf("%s (in class %s)", target, classFQN))
		}
		return "", false, nil
	}

	// Step 3: Resolve method on the attribute's type
	attributeTypeFQN := attr.Type.TypeFQN

	// Check if it's a builtin type
	if strings.HasPrefix(attributeTypeFQN, "builtins.") {
		methodFQN := attributeTypeFQN + "." + methodName

		// Verify method exists in builtin registry
		method := builtins.GetMethod(attributeTypeFQN, methodName)
		if method != nil && method.ReturnType != nil {
			return methodFQN, true, &core.TypeInfo{
				TypeFQN:    method.ReturnType.TypeFQN,
				Confidence: float32(attr.Confidence), // Inherit attribute confidence
				Source:     "self_attribute",
			}
		}

		attributeFailureStats.MethodNotInBuiltins++
		return "", false, nil
	}

	// Handle custom class types (user-defined classes).
	// The attribute type is already resolved (e.g., "module.Controller")
	// from variable extraction. Now we need to resolve the method call on that type.
	methodFQN := attributeTypeFQN + "." + methodName

	// Check if method exists in CallGraph.Functions map.
	if callGraph != nil {
		if node := callGraph.Functions[methodFQN]; node != nil {
			// Verify it's actually a callable (method, function, constructor, etc.).
			if node.Type == "method" || node.Type == "function_definition" ||
				node.Type == "constructor" || node.Type == "property" ||
				node.Type == "special_method" {
				return methodFQN, true, &core.TypeInfo{
					TypeFQN:    attributeTypeFQN,
					Confidence: float32(attr.Confidence),
					Source:     "self_attribute_custom_class",
				}
			}
		}
	}

	// Method not found in call graph - collect stats and return unresolved.
	attributeFailureStats.CustomClassUnsupported++
	if len(attributeFailureStats.CustomClassSamples) < 20 {
		attributeFailureStats.CustomClassSamples = append(
			attributeFailureStats.CustomClassSamples,
			fmt.Sprintf("%s (type: %s, method not found: %s)", target, attributeTypeFQN, methodFQN))
	}
	return "", false, nil
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
