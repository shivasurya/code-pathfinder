package extraction

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// ExtractVariableAssignments extracts variable assignments from a Python file
// and populates the type inference engine with inferred types.
//
// Algorithm:
//  1. Parse source code with tree-sitter Python parser
//  2. Traverse AST to find assignment statements
//  3. For each assignment:
//     - Extract variable name
//     - Infer type from RHS (literal, function call, or method call)
//     - Create VariableBinding with inferred type
//     - Add binding to function scope
//
// Parameters:
//   - filePath: absolute path to the Python file
//   - sourceCode: contents of the file as byte array
//   - typeEngine: type inference engine to populate
//   - registry: module registry for resolving module paths
//   - builtinRegistry: builtin types registry for literal inference
//   - importMap: import mappings for resolving class instantiations from imports
//
// Returns:
//   - error: if parsing fails
//
// Note: Class context is tracked during AST traversal by detecting class_definition nodes.
// This enables building class-qualified FQNs (module.ClassName.methodName) that match
// the FQNs created during function indexing (Pass 1).
func ExtractVariableAssignments(
	filePath string,
	sourceCode []byte,
	typeEngine *resolution.TypeInferenceEngine,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) error {
	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return err
	}
	defer tree.Close()

	// Get module FQN for this file
	modulePath, exists := registry.FileToModule[filePath]
	if !exists {
		// If file not in registry, skip (e.g., external files)
		return nil
	}

	// Traverse AST to find assignments
	// Class context is tracked during traversal by detecting class_definition nodes
	traverseForAssignments(
		tree.RootNode(),
		sourceCode,
		filePath,
		modulePath,
		"",
		"",
		typeEngine,
		registry,
		builtinRegistry,
		importMap,
	)

	return nil
}

// traverseForAssignments recursively traverses AST to find assignment statements.
//
// Parameters:
//   - node: current AST node
//   - sourceCode: source code bytes
//   - filePath: file path for locations
//   - modulePath: module FQN
//   - currentFunction: current function FQN (empty if module-level)
//   - currentClass: current class name (empty if not in a class)
//   - typeEngine: type inference engine
//   - builtinRegistry: builtin types registry
//   - importMap: import mappings for resolving class instantiations
func traverseForAssignments(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	currentClass string,
	typeEngine *resolution.TypeInferenceEngine,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) {
	if node == nil {
		return
	}

	nodeType := node.Type()

	// Update context when entering a class definition
	if nodeType == "class_definition" {
		className := extractClassName(node, sourceCode)
		if className != "" {
			currentClass = className
		}
	}

	// Update context when entering function/method
	if nodeType == "function_definition" {
		functionName := extractFunctionName(node, sourceCode)
		if functionName != "" {
			// Match Pass 1's FQN scheme exactly so call-site lookups
			// (resolveCallTarget → typeEngine.GetScope(callerFQN)) find the
			// bindings created here.
			//
			//   Top-level function f                  → module.f
			//   Direct method m of class C            → module.C.m
			//   Nested fn inner inside top-level outer → module.outer.inner
			//   Nested fn helper inside method m of C  → module.m.helper
			//                                            (NOT class-qualified —
			//                                             nested fns are typed
			//                                             function_definition,
			//                                             not method)
			switch {
			case currentClass != "" && currentFunction == "":
				currentFunction = fmt.Sprintf("%s.%s.%s", modulePath, currentClass, functionName)
			case currentFunction != "":
				// Nested: prepend the parent's qualified name (no class part).
				parent := strings.TrimPrefix(currentFunction, modulePath+".")
				if currentClass != "" {
					parent = strings.TrimPrefix(parent, currentClass+".")
				}
				currentFunction = modulePath + "." + parent + "." + functionName
			default:
				currentFunction = modulePath + "." + functionName
			}

			// Ensure scope exists for this function
			if typeEngine.GetScope(currentFunction) == nil {
				typeEngine.AddScope(resolution.NewFunctionScope(currentFunction))
			}

			// Extract typed parameters as variable bindings so method calls
			// on them (e.g., `bundle.extract()` where `bundle: tarfile.TarFile`)
			// can be resolved by Phase B receiver-type matching.
			processTypedParameters(
				node,
				sourceCode,
				filePath,
				currentFunction,
				typeEngine,
				builtinRegistry,
				importMap,
			)
		}
	}

	// Process assignment statements
	if nodeType == "assignment" {
		processAssignment(
			node,
			sourceCode,
			filePath,
			modulePath,
			currentFunction,
			typeEngine,
			registry,
			builtinRegistry,
			importMap,
		)
	}

	// Process `with ... as var` bindings (Gap 19): treat each `as_pattern`
	// the same as a plain assignment so Phase A's stdlib resolver can resolve
	// patterns like `with tarfile.open(p) as tar:`.
	if nodeType == "with_statement" {
		processWithStatement(
			node,
			sourceCode,
			filePath,
			modulePath,
			currentFunction,
			typeEngine,
			registry,
			builtinRegistry,
			importMap,
		)
	}

	// Recurse to children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		traverseForAssignments(
			child,
			sourceCode,
			filePath,
			modulePath,
			currentFunction,
			currentClass,
			typeEngine,
			registry,
			builtinRegistry,
			importMap,
		)
	}
}

// processAssignment extracts type information from an assignment statement.
//
// Handles:
//   - var = "literal" (literal inference)
//   - var = func() (return type inference - Task 2 Phase 1)
//   - var = obj.method() (method return type - Task 2 Phase 1)
//
// Parameters:
//   - node: assignment AST node
//   - sourceCode: source code bytes
//   - filePath: file path for location
//   - modulePath: module FQN
//   - currentFunction: current function FQN (empty if module-level)
//   - typeEngine: type inference engine
//   - builtinRegistry: builtin types registry
//   - importMap: import mappings for resolving class instantiations
func processAssignment(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	typeEngine *resolution.TypeInferenceEngine,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) {
	// Assignment node structure:
	//   assignment
	//     left: identifier or pattern
	//     "="
	//     right: expression

	var leftNode *sitter.Node
	var rightNode *sitter.Node

	// Find left and right sides of assignment
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "pattern_list" {
			leftNode = child
		} else if child.Type() != "=" && rightNode == nil {
			// Right side is the first non-"=" expression node
			if child.Type() != "identifier" && child.Type() != "pattern_list" {
				rightNode = child
			}
		}
	}

	if leftNode == nil || rightNode == nil {
		return
	}

	// Extract variable name
	varName := leftNode.Content(sourceCode)
	varName = strings.TrimSpace(varName)

	// Skip pattern assignments (tuple unpacking) for now
	if leftNode.Type() == "pattern_list" {
		return
	}

	// Infer type from right side
	typeInfo := inferTypeFromExpression(rightNode, sourceCode, modulePath, registry, builtinRegistry, importMap)
	if typeInfo == nil {
		return
	}

	// Create variable binding
	binding := &resolution.VariableBinding{
		VarName: varName,
		Type:    typeInfo,
		Location: resolution.Location{
			File:   filePath,
			Line:   leftNode.StartPoint().Row + 1,
			Column: leftNode.StartPoint().Column + 1,
		},
	}

	// If RHS is a call, track the function that assigned this
	if rightNode.Type() == "call" {
		calleeName := extractCalleeName(rightNode, sourceCode)
		if calleeName != "" {
			binding.AssignedFrom = calleeName
		}
	}

	// Add to function scope or module-level scope
	scopeFQN := currentFunction
	if scopeFQN == "" {
		// Module-level variable - use module path as scope name
		scopeFQN = modulePath
	}

	scope := typeEngine.GetScope(scopeFQN)
	if scope == nil {
		scope = resolution.NewFunctionScope(scopeFQN)
		typeEngine.AddScope(scope)
	}

	scope.Variables[varName] = append(scope.Variables[varName], binding)
}

// processTypedParameters walks a function definition's parameter list and
// adds a typed VariableBinding for each `typed_parameter` /
// `typed_default_parameter`. This enables receiver-type matching for code
// like `def f(bundle: tarfile.TarFile): bundle.extract(...)`.
//
// Type resolution rules:
//   - Strip Optional[T] / Union[T, None] / T | None → T
//   - Dotted names (e.g., tarfile.TarFile) → use as-is (already qualified)
//   - Bare identifiers → look up in importMap; if found, use FQN
//   - Builtin names (int, str, list, ...) → builtins.<name>
//   - Otherwise → use the stripped name as-is (best-effort)
func processTypedParameters(
	funcNode *sitter.Node,
	sourceCode []byte,
	filePath string,
	currentFunction string,
	typeEngine *resolution.TypeInferenceEngine,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) {
	params := funcNode.ChildByFieldName("parameters")
	if params == nil {
		return
	}

	scope := typeEngine.GetScope(currentFunction)
	if scope == nil {
		return
	}

	for i := 0; i < int(params.ChildCount()); i++ {
		param := params.Child(i)
		if param == nil {
			continue
		}
		if param.Type() != "typed_parameter" && param.Type() != "typed_default_parameter" {
			continue
		}

		// Locate the parameter name (identifier).
		// typed_parameter: identifier, ":", type
		// typed_default_parameter: identifier, ":", type, "=", default_value
		var identNode *sitter.Node
		for j := 0; j < int(param.ChildCount()); j++ {
			c := param.Child(j)
			if c != nil && c.Type() == "identifier" {
				identNode = c
				break
			}
		}
		if identNode == nil {
			continue
		}
		paramName := strings.TrimSpace(identNode.Content(sourceCode))
		if paramName == "" || paramName == "self" || paramName == "cls" {
			continue
		}

		typeNode := param.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}

		typeFQN := resolveParamType(typeNode.Content(sourceCode), importMap, builtinRegistry)
		if typeFQN == "" {
			continue
		}

		binding := &resolution.VariableBinding{
			VarName: paramName,
			Type: &core.TypeInfo{
				TypeFQN:    typeFQN,
				Confidence: 0.95,
				Source:     "param_annotation",
			},
			Location: resolution.Location{
				File:   filePath,
				Line:   identNode.StartPoint().Row + 1,
				Column: identNode.StartPoint().Column + 1,
			},
		}
		scope.Variables[paramName] = append(scope.Variables[paramName], binding)
	}
}

// resolveParamType normalizes a parameter annotation source string to an FQN.
func resolveParamType(annotation string, importMap *core.ImportMap, builtinRegistry *registry.BuiltinRegistry) string {
	trimmed := strings.TrimSpace(annotation)
	// Forward references: `def f(x: "MyClass")` — strip surrounding quotes.
	if len(trimmed) >= 2 {
		first, last := trimmed[0], trimmed[len(trimmed)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
	}
	stripped := stripTypeHintWrappers(trimmed)
	if stripped == "" || stripped == "None" {
		return ""
	}
	// Drop generic args: `dict[str, int]` → `dict`.
	if idx := strings.Index(stripped, "["); idx > 0 {
		stripped = strings.TrimSpace(stripped[:idx])
	}

	// Already-dotted name → assume fully qualified (matches Langflow's
	// `tarfile.TarFile` pattern). The first segment may itself be an
	// imported alias; if so, expand it via importMap.
	if strings.Contains(stripped, ".") {
		if importMap != nil {
			head, rest, _ := strings.Cut(stripped, ".")
			if fqn, ok := importMap.Resolve(head); ok {
				return fqn + "." + rest
			}
		}
		return stripped
	}

	// Bare identifier — try import map first.
	if importMap != nil {
		if fqn, ok := importMap.Resolve(stripped); ok {
			return fqn
		}
	}

	// Builtin? Normalize to builtins.<name>.
	if builtinRegistry != nil {
		if t := builtinRegistry.GetType("builtins." + stripped); t != nil {
			return "builtins." + stripped
		}
	}

	return stripped
}

// processWithStatement extracts `with ... as var` bindings as if they were
// plain assignments, feeding them through the same inferTypeFromExpression
// pipeline. Handles single, multiple, and `async with` items. Tuple-pattern
// aliases (`with f() as (a, b):`) are skipped, mirroring processAssignment.
func processWithStatement(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	modulePath string,
	currentFunction string,
	typeEngine *resolution.TypeInferenceEngine,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) {
	// AST shape: with_statement → with_clause → (with_item | as_pattern)+
	// Each as_pattern has fields: value (context-manager expr) and alias (target).
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() != "with_clause" && child.Type() != "with_item" {
			continue
		}

		for j := 0; j < int(child.NamedChildCount()); j++ {
			item := child.NamedChild(j)
			if item == nil {
				continue
			}

			asPat := item
			if item.Type() == "with_item" {
				for k := 0; k < int(item.NamedChildCount()); k++ {
					inner := item.NamedChild(k)
					if inner != nil && inner.Type() == "as_pattern" {
						asPat = inner
						break
					}
				}
			}
			if asPat == nil || asPat.Type() != "as_pattern" {
				continue
			}

			// tree-sitter-python exposes only the `alias` field on as_pattern;
			// the value is the first named child.
			aliasNode := asPat.ChildByFieldName("alias")
			if aliasNode == nil {
				if n := int(asPat.NamedChildCount()); n >= 2 {
					aliasNode = asPat.NamedChild(n - 1)
				}
			}
			var valueNode *sitter.Node
			if asPat.NamedChildCount() > 0 {
				valueNode = asPat.NamedChild(0)
			}
			if aliasNode == nil || valueNode == nil {
				continue
			}

			varName := strings.TrimSpace(aliasNode.Content(sourceCode))
			// Skip tuple/list unpacking (e.g. `as (a, b)`) — same as processAssignment.
			if varName == "" || strings.ContainsAny(varName, "(),[] \t\n") {
				continue
			}

			typeInfo := inferTypeFromExpression(valueNode, sourceCode, modulePath, registry, builtinRegistry, importMap)
			if typeInfo == nil {
				continue
			}

			binding := &resolution.VariableBinding{
				VarName: varName,
				Type:    typeInfo,
				Location: resolution.Location{
					File:   filePath,
					Line:   aliasNode.StartPoint().Row + 1,
					Column: aliasNode.StartPoint().Column + 1,
				},
			}
			if valueNode.Type() == "call" {
				if calleeName := extractCalleeName(valueNode, sourceCode); calleeName != "" {
					binding.AssignedFrom = calleeName
				}
			}

			scopeFQN := currentFunction
			if scopeFQN == "" {
				scopeFQN = modulePath
			}
			scope := typeEngine.GetScope(scopeFQN)
			if scope == nil {
				scope = resolution.NewFunctionScope(scopeFQN)
				typeEngine.AddScope(scope)
			}
			scope.Variables[varName] = append(scope.Variables[varName], binding)
		}
	}
}

// inferTypeFromExpression infers the type of an expression.
//
// Currently handles:
//   - Literals (strings, numbers, lists, dicts, etc.)
//   - Function calls (creates placeholders or resolves class instantiations)
//
// Parameters:
//   - node: expression AST node
//   - sourceCode: source code bytes
//   - modulePath: module FQN
//   - registry: module registry for class resolution
//   - builtinRegistry: builtin types registry
//   - importMap: import mappings for resolving class instantiations from imports
//
// Returns:
//   - TypeInfo if type can be inferred, nil otherwise
func inferTypeFromExpression(
	node *sitter.Node,
	sourceCode []byte,
	modulePath string,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) *core.TypeInfo {
	if node == nil {
		return nil
	}

	nodeType := node.Type()

	// Handle function calls - try class instantiation first, then create placeholder
	if nodeType == "call" {
		// First, try to resolve as class instantiation (e.g., User(), HttpResponse())
		// This handles PascalCase patterns immediately without creating placeholders
		//
		// CROSS-FILE IMPORT RESOLUTION:
		// Use the provided importMap (from file's actual imports) to resolve class names
		// from other modules. This enables patterns like:
		//   from module_a import Calculator
		//   calc = Calculator()  # ← resolves to module_a.Calculator (not local)
		//   result = calc.add()  # ← resolves to module_a.Calculator.add
		//
		// Edge cases handled:
		// - Inline object creation: Calculator().add(1, 2)
		// - Multi-line chained calls: Calculator()\n    .add(1, 2)\n    .get_result()
		// - Null/empty importMap (tests): Falls back to heuristic resolution
		classType := resolution.ResolveClassInstantiation(node, sourceCode, modulePath, importMap, registry)
		if classType != nil {
			return classType
		}

		// Not a class instantiation - create placeholder for function call
		// This will be resolved later by UpdateVariableBindingsWithFunctionReturns()
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" || child.Type() == "attribute" {
				calleeName := extractCalleeName(child, sourceCode)
				if calleeName != "" {
					return &core.TypeInfo{
						TypeFQN:    "call:" + calleeName,
						Confidence: 0.5, // Medium confidence - will be refined later
						Source:     "function_call_placeholder",
					}
				}
			}
		}
	}

	// Handle boolean operators (or, and)
	// Supports conditional patterns: x = param or Class()
	if nodeType == "boolean_operator" {
		return inferFromBooleanOp(node, sourceCode, modulePath, registry, builtinRegistry, importMap)
	}

	// Handle literals
	switch nodeType {
	case "string", "concatenated_string":
		return &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "integer":
		return &core.TypeInfo{
			TypeFQN:    "builtins.int",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "float":
		return &core.TypeInfo{
			TypeFQN:    "builtins.float",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "true", "false":
		return &core.TypeInfo{
			TypeFQN:    "builtins.bool",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "none":
		return &core.TypeInfo{
			TypeFQN:    "builtins.NoneType",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "list":
		return &core.TypeInfo{
			TypeFQN:    "builtins.list",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "dictionary":
		return &core.TypeInfo{
			TypeFQN:    "builtins.dict",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "set":
		return &core.TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "tuple":
		return &core.TypeInfo{
			TypeFQN:    "builtins.tuple",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	// For non-literals, try to infer from builtin registry
	// This handles edge cases where tree-sitter node types don't match exactly
	literal := node.Content(sourceCode)
	return builtinRegistry.InferLiteralType(literal)
}

// inferFromBooleanOp infers type from boolean operator expressions.
//
// Handles conditional patterns for local variables:
//   - x or Y(): Prefer right operand (concrete value over None/falsy)
//   - x and Y(): Prefer left operand
//   - x or y or Z(): Recursively process nested operators
//
// Examples:
//   - config or Settings() → type: Settings (confidence: 0.855)
//   - x or None → type: NoneType (confidence: 0.95)
//   - enabled and "active" → type: str (confidence: 0.93)
//
// Parameters:
//   - node: boolean_operator AST node
//   - sourceCode: source code bytes
//   - modulePath: module FQN for class resolution
//   - registry: module registry
//   - builtinRegistry: builtin types registry
//   - importMap: import mappings
//
// Returns:
//   - TypeInfo with inferred type and adjusted confidence, or nil if cannot infer
func inferFromBooleanOp(
	node *sitter.Node,
	sourceCode []byte,
	modulePath string,
	registry *core.ModuleRegistry,
	builtinRegistry *registry.BuiltinRegistry,
	importMap *core.ImportMap,
) *core.TypeInfo {
	// Find operator and operands by traversing children
	var operator string
	var leftNode, rightNode *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "or", "and":
			operator = child.Type()
		default:
			// Capture first two non-operator nodes as operands
			if leftNode == nil {
				leftNode = child
			} else if rightNode == nil {
				rightNode = child
			}
		}
	}

	if leftNode == nil || operator == "" {
		return nil
	}

	// For "or": prefer right operand (concrete value over None/falsy)
	// Example: config or Settings() → infer Settings
	if operator == "or" && rightNode != nil {
		rightType := inferTypeFromExpression(rightNode, sourceCode, modulePath, registry, builtinRegistry, importMap)
		if rightType != nil && rightType.TypeFQN != "builtins.NoneType" {
			// Apply confidence penalty for conditional pattern
			rightType.Confidence *= 0.95
			rightType.Source = "boolean_or_" + rightType.Source
			return rightType
		}

		// Fallback to left operand if right is None or cannot be inferred
		leftType := inferTypeFromExpression(leftNode, sourceCode, modulePath, registry, builtinRegistry, importMap)
		if leftType != nil {
			leftType.Confidence *= 0.90
			leftType.Source = "boolean_or_" + leftType.Source
			return leftType
		}
	}

	// For "and": prefer left operand
	// Example: enabled and "active" → infer from "active"
	if operator == "and" {
		leftType := inferTypeFromExpression(leftNode, sourceCode, modulePath, registry, builtinRegistry, importMap)
		if leftType != nil {
			leftType.Confidence *= 0.93
			leftType.Source = "boolean_and_" + leftType.Source
			return leftType
		}
	}

	return nil
}

// extractFunctionName extracts the function name from a function_definition node.
func extractFunctionName(node *sitter.Node, sourceCode []byte) string {
	if node.Type() != "function_definition" {
		return ""
	}

	// Find the identifier node (function name)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return child.Content(sourceCode)
		}
	}

	return ""
}

// extractCalleeName extracts the name of a called function/method from an AST node.
func extractCalleeName(node *sitter.Node, sourceCode []byte) string {
	nodeType := node.Type()

	switch nodeType {
	case "identifier":
		// Simple function call: foo()
		return node.Content(sourceCode)

	case "attribute":
		// Method call: obj.method() or obj.attr.method()
		// The attribute node has 'object' and 'attribute' fields
		objectNode := node.ChildByFieldName("object")
		attributeNode := node.ChildByFieldName("attribute")

		if objectNode != nil && attributeNode != nil {
			// Recursively extract object name (could be nested)
			objectName := extractCalleeName(objectNode, sourceCode)
			attributeName := attributeNode.Content(sourceCode)

			if objectName != "" && attributeName != "" {
				return objectName + "." + attributeName
			}
		}

	case "call":
		// Chained call: foo()() or obj.method()()
		// For now, just extract the outer call's function
		return node.Content(sourceCode)
	}

	return ""
}

