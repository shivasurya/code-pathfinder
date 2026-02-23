package extraction

import (
	"context"
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution/strategies"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ExtractClassAttributes extracts all class attributes from a Python file
// This is Pass 1 & 2 of the attribute extraction algorithm:
//
//	Pass 1: Extract class metadata (FQN, methods, file path)
//	Pass 2: Extract attribute assignments (self.attr = value)
//
// Algorithm:
//  1. Parse file with tree-sitter
//  2. Find all class definitions
//  3. For each class:
//     a. Create ClassAttributes entry
//     b. Collect method names
//     c. Scan for self.attr assignments
//     d. Infer types using 6 strategies
//
// Parameters:
//   - filePath: absolute path to Python file
//   - sourceCode: file contents
//   - modulePath: fully qualified module path (e.g., "myapp.models")
//   - typeEngine: type inference engine with return types and variables
//   - registry: attribute registry to populate
//
// Returns:
//   - error if parsing fails
func ExtractClassAttributes(
	filePath string,
	sourceCode []byte,
	modulePath string,
	typeEngine *resolution.TypeInferenceEngine,
	attrRegistry *registry.AttributeRegistry,
) error {
	// Parse file with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filePath, err)
	}
	defer tree.Close()

	root := tree.RootNode()

	// Find all class definitions in file
	classes := findClassNodes(root, sourceCode)

	for _, classNode := range classes {
		className := extractClassName(classNode, sourceCode)
		if className == "" {
			continue
		}

		// Build fully qualified class name
		classFQN := modulePath + "." + className

		// Create ClassAttributes entry
		classAttrs := &core.ClassAttributes{
			ClassFQN:   classFQN,
			Attributes: make(map[string]*core.ClassAttribute),
			Methods:    []string{},
			FilePath:   filePath,
		}

		// Pass 1: Extract method names
		methodNodes := findMethodNodes(classNode, sourceCode)
		for _, methodNode := range methodNodes {
			methodName := extractMethodName(methodNode, sourceCode)
			if methodName != "" {
				methodFQN := classFQN + "." + methodName
				classAttrs.Methods = append(classAttrs.Methods, methodFQN)
			}
		}

		// Pass 2: Extract attribute assignments
		attributeMap := extractAttributeAssignments(
			classNode,
			sourceCode,
			classFQN,
			filePath,
			typeEngine,
		)

		classAttrs.Attributes = attributeMap

		// Add to registry
		attrRegistry.AddClassAttributes(classAttrs)
	}

	return nil
}

// findClassNodes finds all class_definition nodes in the AST.
func findClassNodes(node *sitter.Node, _ []byte) []*sitter.Node {
	classes := make([]*sitter.Node, 0)

	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		if n.Type() == "class_definition" {
			classes = append(classes, n)
		}

		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(i)
			if child != nil {
				traverse(child)
			}
		}
	}

	traverse(node)
	return classes
}

// extractClassName extracts the class name from a class_definition node.
func extractClassName(classNode *sitter.Node, sourceCode []byte) string {
	// class_definition has structure:
	//   class <identifier> [(bases)] : <block>
	//   The identifier is the second child (after "class" keyword)

	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child == nil {
			continue
		}

		if child.Type() == "identifier" {
			return child.Content(sourceCode)
		}
	}

	return ""
}

// findMethodNodes finds all function_definition nodes within a class.
func findMethodNodes(classNode *sitter.Node, _ []byte) []*sitter.Node {
	methods := make([]*sitter.Node, 0)

	// Find the block node
	var blockNode *sitter.Node
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child != nil && child.Type() == "block" {
			blockNode = child
			break
		}
	}

	if blockNode == nil {
		return methods
	}

	// Find function_definition nodes in the block
	for i := 0; i < int(blockNode.ChildCount()); i++ {
		child := blockNode.Child(i)
		if child != nil && child.Type() == "function_definition" {
			methods = append(methods, child)
		}
	}

	return methods
}

// extractMethodName extracts the method name from a function_definition node.
func extractMethodName(methodNode *sitter.Node, sourceCode []byte) string {
	for i := 0; i < int(methodNode.ChildCount()); i++ {
		child := methodNode.Child(i)
		if child != nil && child.Type() == "identifier" {
			return child.Content(sourceCode)
		}
	}
	return ""
}

// extractAttributeAssignments extracts all self.attr = value assignments from a class
// This implements the 6 type inference strategies:
//  1. Literal values: self.name = "John" → builtins.str
//  2. Class instantiation: self.user = User() → myapp.User
//  3. Function returns: self.result = calculate() → lookup return type
//  4. Constructor parameters: def __init__(self, user: User) → User
//  5. Attribute copy: self.my_obj = other.obj → lookup other.obj
//  6. Type annotations: self.value: str = None → builtins.str
func extractAttributeAssignments(
	classNode *sitter.Node,
	sourceCode []byte,
	_ string,
	filePath string,
	typeEngine *resolution.TypeInferenceEngine,
) map[string]*core.ClassAttribute {
	attributes := make(map[string]*core.ClassAttribute)

	// Find all method blocks in the class
	methods := findMethodNodes(classNode, sourceCode)

	for _, methodNode := range methods {
		methodName := extractMethodName(methodNode, sourceCode)

		// Find assignments in method body
		assignments := findSelfAttributeAssignments(methodNode, sourceCode)

		for _, assignment := range assignments {
			attrName := assignment.AttributeName

			// Infer type using the 6 strategies
			typeInfo := inferAttributeType(
				assignment,
				sourceCode,
				typeEngine,
				methodNode,
				filePath,
			)

			if typeInfo != nil {
				attr := &core.ClassAttribute{
					Name:       attrName,
					Type:       typeInfo,
					AssignedIn: methodName,
					Location: &graph.SourceLocation{
						File:      filePath,
						StartByte: assignment.Node.StartByte(),
						EndByte:   assignment.Node.EndByte(),
					},
					Confidence: float64(typeInfo.Confidence),
				}

				// If attribute already exists, keep the one with higher confidence
				existing, exists := attributes[attrName]
				if !exists || attr.Confidence > existing.Confidence {
					attributes[attrName] = attr
				}
			}
		}
	}

	return attributes
}

// AttributeAssignment represents a self.attr = value assignment.
type AttributeAssignment struct {
	AttributeName string       // Name of the attribute (e.g., "value", "user")
	RightSide     *sitter.Node // AST node of the right-hand side expression
	Node          *sitter.Node // Full assignment node
}

// findSelfAttributeAssignments finds all self.attr = value patterns in a method.
func findSelfAttributeAssignments(methodNode *sitter.Node, sourceCode []byte) []AttributeAssignment {
	assignments := make([]AttributeAssignment, 0)

	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		// Look for assignment nodes
		if n.Type() == "assignment" {
			// Check if left side is self.attr
			leftNode := n.ChildByFieldName("left")
			rightNode := n.ChildByFieldName("right")

			if leftNode != nil && rightNode != nil {
				// Check if left is attribute (self.attr)
				if leftNode.Type() == "attribute" {
					// Get object and attribute
					objectNode := leftNode.ChildByFieldName("object")
					attrNode := leftNode.ChildByFieldName("attribute")

					if objectNode != nil && attrNode != nil {
						objectName := objectNode.Content(sourceCode)
						attrName := attrNode.Content(sourceCode)

						// Check if object is "self"
						if objectName == "self" {
							assignments = append(assignments, AttributeAssignment{
								AttributeName: attrName,
								RightSide:     rightNode,
								Node:          n,
							})
						}
					}
				}
			}
		}

		// Recurse to children
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(i)
			if child != nil {
				traverse(child)
			}
		}
	}

	traverse(methodNode)
	return assignments
}

// inferAttributeType infers the type of an attribute using 6 strategies.
func inferAttributeType(
	assignment AttributeAssignment,
	sourceCode []byte,
	typeEngine *resolution.TypeInferenceEngine,
	methodNode *sitter.Node,
	filePath string,
) *core.TypeInfo {
	rightNode := assignment.RightSide

	// Strategy 1: Literal values (confidence: 1.0)
	if typeInfo := inferFromLiteral(rightNode, sourceCode); typeInfo != nil {
		return typeInfo
	}

	// Strategy 2: Constructor parameters (confidence: 0.95) - Run before class instantiation
	// This strategy handles typed parameters like "controller: Optional[Controller] = None"
	if typeInfo := inferFromConstructorParam(assignment, methodNode, sourceCode, typeEngine); typeInfo != nil {
		return typeInfo
	}

	// Strategy 3: Class instantiation (confidence: 0.9)
	// Fallback for untyped parameters like "controller=None" with "controller or Controller()"
	if typeInfo := inferFromClassInstantiation(rightNode, sourceCode, typeEngine); typeInfo != nil {
		return typeInfo
	}

	// Strategy 3b: Inline instantiation with chaining (confidence: 0.85)
	// Handles Controller().configure() or Builder().set_x().set_y() patterns
	if typeInfo := inferFromInlineInstantiation(rightNode, sourceCode, typeEngine, filePath); typeInfo != nil {
		return typeInfo
	}

	// Strategy 4: Function call returns (confidence: 0.8)
	if typeInfo := inferFromFunctionCall(rightNode, sourceCode, typeEngine); typeInfo != nil {
		return typeInfo
	}

	// Strategy 5: Attribute copy (confidence: 0.85)
	if typeInfo := inferFromAttributeCopy(rightNode, sourceCode, typeEngine); typeInfo != nil {
		return typeInfo
	}

	// Strategy 6: Type annotations (confidence: 1.0)
	// Note: This is now handled by Strategy 2 (Constructor parameters)

	// Unknown type
	return nil
}

// Strategy 1: Infer type from literal values.
func inferFromLiteral(node *sitter.Node, _ []byte) *core.TypeInfo {
	nodeType := node.Type()

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
	case "tuple":
		return &core.TypeInfo{
			TypeFQN:    "builtins.tuple",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "set":
		return &core.TypeInfo{
			TypeFQN:    "builtins.set",
			Confidence: 1.0,
			Source:     "literal",
		}
	case "none":
		return &core.TypeInfo{
			TypeFQN:    "builtins.NoneType",
			Confidence: 1.0,
			Source:     "literal",
		}
	}

	return nil
}

// Strategy 2: Infer type from class instantiation.
//
//nolint:unparam // typeEngine kept for strategy interface consistency and recursive call
func inferFromClassInstantiation(node *sitter.Node, sourceCode []byte, typeEngine *resolution.TypeInferenceEngine) *core.TypeInfo {
	// Handle boolean operators: extract class from right side of "or"
	// e.g., "controller or Controller()" → extract "Controller()"
	if node.Type() == "boolean_operator" {
		// Find the operator and operands
		var leftNode, rightNode *sitter.Node
		var foundOr bool

		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}

			switch child.Type() {
			case "or":
				foundOr = true
			case "identifier", "call", "attribute", "boolean_operator":
				if leftNode == nil {
					leftNode = child
				} else if rightNode == nil {
					rightNode = child
				}
			}
		}

		// For "or" operator, prefer right side (concrete value over None/falsy)
		if foundOr && rightNode != nil {
			return inferFromClassInstantiation(rightNode, sourceCode, typeEngine)
		}

		return nil
	}

	if node.Type() != "call" {
		return nil
	}

	// Get the function being called
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}

	// Simple identifier (e.g., User())
	if funcNode.Type() == "identifier" {
		className := funcNode.Content(sourceCode)

		// Check if it's a known class (starts with uppercase by convention)
		if len(className) > 0 && className[0] >= 'A' && className[0] <= 'Z' {
			return &core.TypeInfo{
				TypeFQN:    "class:" + className, // Placeholder, will be resolved later
				Confidence: 0.9,
				Source:     "class_instantiation_attribute",
			}
		}
	}

	return nil
}

// moduleRegistryAdapter adapts core.ModuleRegistry to strategies.ModuleRegistryInterface.
// Needed because GetModulePath signatures differ:
// - core.ModuleRegistry.GetModulePath(string) (string, bool)
// - strategies.ModuleRegistryInterface.GetModulePath(string) string.
type moduleRegistryAdapter struct {
	registry *core.ModuleRegistry
}

func (a *moduleRegistryAdapter) GetModulePath(filePath string) string {
	// Use FileToModule map to convert file path to module path
	// Note: ModuleRegistry.GetModulePath() does the opposite (module -> file)
	if a.registry.FileToModule != nil {
		if modulePath, ok := a.registry.FileToModule[filePath]; ok {
			return modulePath
		}
	}
	return ""
}

func (a *moduleRegistryAdapter) ResolveImport(importPath string, fromFile string) (string, bool) {
	// ModuleRegistry doesn't have ResolveImport method, return not found
	// ChainStrategy will still work without import resolution
	return "", false
}

// Strategy 3b: Infer type from inline instantiation with chaining.
// Handles patterns like Controller().configure() or Builder().set_x().set_y()
// Uses ChainStrategy to resolve the chain and extract the base class type.
// Confidence: 0.85 (heuristic-based fluent interface detection).
func inferFromInlineInstantiation(
	node *sitter.Node,
	sourceCode []byte,
	typeEngine *resolution.TypeInferenceEngine,
	filePath string,
) *core.TypeInfo {
	// Defensive checks: ensure registries are available
	if typeEngine == nil || typeEngine.Attributes == nil || typeEngine.Registry == nil {
		return nil
	}

	// Create inference context for ChainStrategy
	// Note: BuiltinRegistry is set to nil because registry.BuiltinRegistry
	// doesn't implement GetMethodReturnType method required by the interface.
	// ModuleRegistry uses an adapter to match the interface signature.
	ctx := &strategies.InferenceContext{
		SourceCode:      sourceCode,
		FilePath:        filePath,
		Store:           resolution.NewTypeStore(),
		AttrRegistry:    typeEngine.Attributes,
		ModuleRegistry:  &moduleRegistryAdapter{registry: typeEngine.Registry},
		BuiltinRegistry: nil, // Interface mismatch: missing GetMethodReturnType
	}

	// Use ChainStrategy to resolve inline instantiation patterns
	chainStrat := strategies.NewChainStrategy()
	if !chainStrat.CanHandle(node, ctx) {
		return nil
	}

	// Synthesize the type through the chain
	resolvedType, confidence := chainStrat.Synthesize(node, ctx)

	// Extract concrete type FQN
	if concrete, ok := core.ExtractConcreteType(resolvedType); ok {
		typeFQN := concrete.FQN()

		// If the FQN is just a class name without module prefix (e.g., "Controller"),
		// try to resolve it to the full FQN by searching the AttributeRegistry.
		// This handles cross-file class references where ChainStrategy returns the
		// short name because it doesn't have import resolution.
		if !strings.Contains(typeFQN, ".") && typeEngine.Attributes != nil {
			if fullFQN := resolveClassShortName(typeFQN, typeEngine.Attributes); fullFQN != "" {
				typeFQN = fullFQN
			}
		}

		return &core.TypeInfo{
			TypeFQN:    typeFQN,
			Confidence: float32(confidence),
			Source:     "inline_instantiation",
		}
	}

	return nil
}

// resolveClassShortName attempts to resolve a short class name (e.g., "Controller")
// to its full FQN by searching the AttributeRegistry for matching classes.
// Returns the full FQN if found, empty string otherwise.
func resolveClassShortName(shortName string, attrRegistry *registry.AttributeRegistry) string {
	if attrRegistry == nil {
		return ""
	}

	// Get all registered classes and search for ones matching the short name
	// We look for FQNs that end with ".<shortName>" (e.g., "controller.Controller")
	var matches []string
	suffix := "." + shortName

	// Iterate through all classes in the registry
	// Note: AttributeRegistry doesn't expose a list of all classes directly,
	// so we need to use the internal Classes map
	for classFQN := range attrRegistry.Classes {
		if strings.HasSuffix(classFQN, suffix) {
			matches = append(matches, classFQN)
		}
	}

	// If we found exactly one match, return it
	if len(matches) == 1 {
		return matches[0]
	}

	// If multiple matches, we can't disambiguate without import information
	// Return empty string to keep the short name
	return ""
}

// Strategy 4: Infer type from function call returns.
func inferFromFunctionCall(node *sitter.Node, sourceCode []byte, _ *resolution.TypeInferenceEngine) *core.TypeInfo {
	if node.Type() != "call" {
		return nil
	}

	// Get the function being called
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}

	// Simple function call (lowercase by convention)
	if funcNode.Type() == "identifier" {
		funcName := funcNode.Content(sourceCode)

		// Check if it's lowercase (function, not class)
		if len(funcName) > 0 && funcName[0] >= 'a' && funcName[0] <= 'z' {
			// Try to lookup return type
			// For now, use placeholder - will be resolved in Pass 3
			return &core.TypeInfo{
				TypeFQN:    "call:" + funcName,
				Confidence: 0.8,
				Source:     "function_call_attribute",
			}
		}
	}

	return nil
}

// Strategy 4: Infer type from constructor parameters.
func inferFromConstructorParam(
	assignment AttributeAssignment,
	methodNode *sitter.Node,
	sourceCode []byte,
	_ *resolution.TypeInferenceEngine,
) *core.TypeInfo {
	// Check if we're in __init__
	methodName := extractMethodName(methodNode, sourceCode)
	if methodName != "__init__" {
		return nil
	}

	// Extract parameter name from RHS (handles both simple identifier and boolean operators)
	paramName := extractParamNameFromRHS(assignment.RightSide, sourceCode)
	if paramName == "" {
		return nil
	}

	// Get function parameters
	params := methodNode.ChildByFieldName("parameters")
	if params == nil {
		return nil
	}

	// Find matching parameter with type annotation
	for i := 0; i < int(params.ChildCount()); i++ {
		param := params.Child(i)
		if param == nil {
			continue
		}
		// Handle both typed_parameter and typed_default_parameter
		// typed_default_parameter is used when parameter has both type annotation AND default value
		// e.g., controller: Optional[Controller] = None
		if param.Type() != "typed_parameter" && param.Type() != "typed_default_parameter" {
			continue
		}

		// Get parameter name
		// Note: typed_parameter uses field name, but typed_default_parameter uses positional access
		identNode := param.ChildByFieldName("identifier")
		if identNode == nil && param.ChildCount() > 0 {
			// For typed_default_parameter, identifier is first child
			firstChild := param.Child(0)
			if firstChild != nil && firstChild.Type() == "identifier" {
				identNode = firstChild
			}
		}
		if identNode == nil {
			continue
		}

		if identNode.Content(sourceCode) == paramName {
			// Get type annotation
			typeNode := param.ChildByFieldName("type")
			if typeNode == nil {
				continue
			}

			typeName := typeNode.Content(sourceCode)

			// Strip type hint wrappers (Optional[], Union[], etc.) to get the actual class name
			strippedTypeName := stripTypeHintWrappers(typeName)

			// Adjust confidence based on assignment pattern
			// Simple identifier (self.x = param): 0.95
			// Boolean operator (self.x = param or Class()): 0.92
			confidence := float32(0.95)
			if assignment.RightSide.Type() == "boolean_operator" {
				confidence = 0.92
			}

			return &core.TypeInfo{
				TypeFQN:    "param:" + strippedTypeName, // Placeholder, will be resolved
				Confidence: confidence,
				Source:     "constructor_param",
			}
		}
	}

	return nil
}

// Strategy 5: Infer type from attribute copy (self.obj = other.attr).
func inferFromAttributeCopy(node *sitter.Node, _ []byte, _ *resolution.TypeInferenceEngine) *core.TypeInfo {
	// Check if right side is attribute access
	if node.Type() != "attribute" {
		return nil
	}

	// For now, return placeholder - this would need class attribute lookup
	// which creates circular dependency (need attributes to infer attributes)
	// This is a future enhancement
	return nil
}

// stripTypeHintWrappers removes Python type hint wrappers to extract the core class name.
//
// Handles common patterns:
//   - Optional[ClassName] → ClassName
//   - Union[ClassName, None] → ClassName
//   - ClassName | None → ClassName
//   - ClassName → ClassName (no change)
//
// This enables type annotations like "Optional[Controller]" to resolve to "Controller"
// for ImportMap lookup to work correctly.
//
// Examples:
//   - "Optional[Controller]" → "Controller"
//   - "Union[Handler, None]" → "Handler"
//   - "Controller | None" → "Controller"
//   - "Controller" → "Controller"
func stripTypeHintWrappers(typeName string) string {
	typeName = strings.TrimSpace(typeName)

	// Handle Optional[ClassName] → extract ClassName
	if strings.HasPrefix(typeName, "Optional[") && strings.HasSuffix(typeName, "]") {
		inner := typeName[len("Optional[") : len(typeName)-1]
		return strings.TrimSpace(inner)
	}

	// Handle Union[ClassName, None] or Union[None, ClassName] → extract ClassName
	if strings.HasPrefix(typeName, "Union[") && strings.HasSuffix(typeName, "]") {
		inner := typeName[len("Union[") : len(typeName)-1]
		parts := strings.SplitSeq(inner, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part != "None" && part != "" {
				return part
			}
		}
	}

	// Handle ClassName | None or None | ClassName → extract ClassName
	if strings.Contains(typeName, "|") {
		parts := strings.SplitSeq(typeName, "|")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part != "None" && part != "" {
				return part
			}
		}
	}

	// No wrapper, return as-is
	return typeName
}

// extractParamNameFromRHS extracts parameter name from RHS expression, handling:
// - Simple identifier: "controller"
// - Boolean operator: "controller or Controller()"
//
// This supports conditional assignment patterns used in dependency injection:
//
//	def __init__(self, controller: Controller | None = None):
//	    self.controller = controller or Controller()
//
// Returns the parameter name if found, empty string otherwise.
func extractParamNameFromRHS(node *sitter.Node, sourceCode []byte) string {
	switch node.Type() {
	case "identifier":
		return node.Content(sourceCode)

	case "boolean_operator":
		return extractParamFromBooleanOp(node, sourceCode)

	default:
		return ""
	}
}

// extractParamFromBooleanOp handles boolean operator patterns (or, and).
//
// Supported patterns:
//   - "param or Class()": Extract "param" if right is instantiation
//   - "param and handler": Extract "param" from left operand
//   - "a or b or Class()": Extract leftmost identifier "a"
//
// Returns parameter name if pattern matches, empty string otherwise.
func extractParamFromBooleanOp(node *sitter.Node, sourceCode []byte) string {
	// Find operator ("or" or "and") and operands
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
		case "identifier", "call", "attribute", "boolean_operator":
			if leftNode == nil {
				leftNode = child
			} else if rightNode == nil {
				rightNode = child
			}
		}
	}

	if leftNode == nil || operator == "" {
		return ""
	}

	// For "or": param or Class()
	if operator == "or" {
		// Left should be identifier (param name)
		if leftNode.Type() == "identifier" {
			paramName := leftNode.Content(sourceCode)

			// Validate right is instantiation (call node)
			if rightNode != nil && rightNode.Type() == "call" {
				return paramName
			}

			// Handle nested: param or other or Class()
			// Still return leftmost param name
			if rightNode != nil && rightNode.Type() == "boolean_operator" {
				return paramName
			}
		}
	}

	// For "and": param and param.method()
	if operator == "and" {
		if leftNode.Type() == "identifier" {
			return leftNode.Content(sourceCode)
		}
	}

	return ""
}
