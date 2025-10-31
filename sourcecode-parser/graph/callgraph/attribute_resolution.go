package callgraph

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
)

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
//  Input: self.value.upper (caller: test_chaining.StringBuilder.process)
//  Steps:
//    1. Parse → attr="value", method="upper"
//    2. Extract class → test_chaining.StringBuilder
//    3. Lookup value type → builtins.str
//    4. Resolve upper on str → builtins.str.upper
//  Output: (builtins.str.upper, true, TypeInfo{builtins.str, 1.0, "self_attribute"})
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
	builtins *BuiltinRegistry,
	callGraph *CallGraph,
) (string, bool, *TypeInfo) {
	// Check if this is a self.attr.method pattern
	if !strings.HasPrefix(target, "self.") {
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
		return "", false, nil
	}

	attrName := parts[1]
	methodName := parts[2]

	// Step 1: Find the containing class by checking which classes have this method
	classFQN := findClassContainingMethod(callerFQN, typeEngine.Attributes)
	if classFQN == "" {
		return "", false, nil
	}

	// Step 2: Lookup attribute in AttributeRegistry
	attr := typeEngine.Attributes.GetAttribute(classFQN, attrName)
	if attr == nil {
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
			return methodFQN, true, &TypeInfo{
				TypeFQN:    method.ReturnType.TypeFQN,
				Confidence: float32(attr.Confidence), // Inherit attribute confidence
				Source:     "self_attribute",
			}
		}

		return "", false, nil
	}

	// TODO: Handle custom class types (class:User → myapp.User)
	// This requires resolving placeholders and class method lookup
	return "", false, nil
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
func findClassContainingMethod(methodFQN string, registry *AttributeRegistry) string {
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
		for _, method := range classAttrs.Methods {
			if method == expectedMethodFQN {
				return classFQN
			}
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
	registry *AttributeRegistry,
	typeEngine *TypeInferenceEngine,
	moduleRegistry *ModuleRegistry,
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
			if strings.HasPrefix(originalType, "class:") {
				// class:User → try to resolve to full FQN
				className := strings.TrimPrefix(originalType, "class:")
				resolvedFQN := resolveClassName(className, classFQN, moduleRegistry, codeGraph)
				if resolvedFQN != "" {
					attr.Type.TypeFQN = resolvedFQN
					attr.Type.Confidence = 0.9 // High confidence for resolved classes
				}
			} else if strings.HasPrefix(originalType, "call:") {
				// call:func → lookup return type
				funcName := strings.TrimPrefix(originalType, "call:")
				// Try to find function in same module
				modulePath := getModuleFromClassFQN(classFQN)
				funcFQN := modulePath + "." + funcName

				if returnType, exists := typeEngine.ReturnTypes[funcFQN]; exists && returnType != nil {
					attr.Type.TypeFQN = returnType.TypeFQN
					attr.Type.Confidence = returnType.Confidence * 0.8 // Decay confidence
					attr.Type.Source = "function_call_attribute"
				}
			} else if strings.HasPrefix(originalType, "param:") {
				// param:User → resolve type annotation
				typeName := strings.TrimPrefix(originalType, "param:")
				resolvedFQN := resolveClassName(typeName, classFQN, moduleRegistry, codeGraph)
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

// resolveClassName resolves a class name to its fully qualified name
// Uses module registry and code graph to find the class definition
func resolveClassName(
	className string,
	contextClassFQN string,
	moduleRegistry *ModuleRegistry,
	codeGraph *graph.CodeGraph,
) string {
	// Get the module of the context class
	modulePath := getModuleFromClassFQN(contextClassFQN)

	// Try same module first
	candidateFQN := modulePath + "." + className
	if classExists(candidateFQN, codeGraph) {
		return candidateFQN
	}

	// Try short name lookup in module registry
	if paths, ok := moduleRegistry.ShortNames[className]; ok && len(paths) > 0 {
		// Use first match (could be improved with import analysis)
		filePath := paths[0]
		if modulePath, ok := moduleRegistry.FileToModule[filePath]; ok {
			return modulePath + "." + className
		}
	}

	// Not found
	return ""
}

// classExists checks if a class exists in the code graph
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
//   test_chaining.StringBuilder → test_chaining
//   myapp.models.User → myapp.models
func getModuleFromClassFQN(classFQN string) string {
	parts := strings.Split(classFQN, ".")
	if len(parts) < 2 {
		return classFQN
	}
	// Return all but the last part (class name)
	return strings.Join(parts[:len(parts)-1], ".")
}
