package graph

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	pythonlang "github.com/shivasurya/code-pathfinder/sast-engine/graph/python"
	sitter "github.com/smacker/go-tree-sitter"
)

// extractDecorators extracts decorators from a decorated_definition node.
// Returns a list of decorator names (e.g., ["property", "staticmethod"]).
func extractDecorators(node *sitter.Node, sourceCode []byte) []string {
	var decorators []string

	// Check if this is a decorated_definition.
	if node.Type() != "decorated_definition" {
		return decorators
	}

	// Iterate through children to find decorator nodes.
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "decorator" {
			decoratorText := child.Content(sourceCode)
			// Remove @ symbol and extract name.
			decoratorText = strings.TrimPrefix(decoratorText, "@")
			// Handle decorators with arguments like @decorator(args).
			if idx := strings.Index(decoratorText, "("); idx != -1 {
				decoratorText = decoratorText[:idx]
			}
			decoratorText = strings.TrimSpace(decoratorText)
			decorators = append(decorators, decoratorText)
		}
	}

	return decorators
}

// hasDecorator checks if a list of decorators contains a specific decorator.
func hasDecorator(decorators []string, name string) bool {
	return slices.Contains(decorators, name)
}

// isConstantName checks if a variable name follows Python constant naming convention.
// Constants are typically all uppercase with underscores (e.g., MAX_SIZE, API_KEY).
func isConstantName(name string) bool {
	if name == "" {
		return false
	}

	// Must contain at least one letter.
	hasLetter := false
	for _, r := range name {
		if unicode.IsLetter(r) {
			hasLetter = true
			if unicode.IsLower(r) {
				// Contains lowercase letter - not a constant.
				return false
			}
		} else if r != '_' && !unicode.IsDigit(r) {
			// Contains non-alphanumeric characters (except underscore).
			return false
		}
	}

	return hasLetter
}

// isConstructor checks if a function is a Python constructor (__init__).
func isConstructor(functionName string) bool {
	return functionName == "__init__"
}

// isSpecialMethod checks if a function is a Python special/magic method.
// Special methods are surrounded by double underscores (e.g., __str__, __add__).
func isSpecialMethod(functionName string) bool {
	if len(functionName) < 5 {
		// Minimum length for __x__ is 5 characters.
		return false
	}
	return strings.HasPrefix(functionName, "__") && strings.HasSuffix(functionName, "__")
}

// isInterface checks if a class is a Python interface (Protocol or ABC).
// Checks if any base class is "Protocol", "ABC", or ends with those names.
func isInterface(superClasses []string) bool {
	for _, base := range superClasses {
		// Direct inheritance from Protocol or ABC.
		if base == "Protocol" || base == "ABC" {
			return true
		}
		// Qualified names like typing.Protocol or abc.ABC.
		if strings.HasSuffix(base, ".Protocol") || strings.HasSuffix(base, ".ABC") {
			return true
		}
	}
	return false
}

// isEnum checks if a class is a Python Enum.
// Checks if any base class is "Enum" or qualified like enum.Enum.
func isEnum(superClasses []string) bool {
	for _, base := range superClasses {
		if base == "Enum" || base == "IntEnum" || base == "Flag" || base == "IntFlag" {
			return true
		}
		if strings.HasSuffix(base, ".Enum") || strings.HasSuffix(base, ".IntEnum") ||
			strings.HasSuffix(base, ".Flag") || strings.HasSuffix(base, ".IntFlag") {
			return true
		}
	}
	return false
}

// isDataclass checks if a class has the @dataclass decorator.
func isDataclass(decorators []string) bool {
	for _, d := range decorators {
		if d == "dataclass" || strings.HasSuffix(d, ".dataclass") {
			return true
		}
	}
	return false
}

// parsePythonFunctionDefinition parses Python function definitions.
// Handles decorators to distinguish between regular functions, methods, properties, and constructors.
// Distinguishes methods (functions inside classes) from module-level functions.
func parsePythonFunctionDefinition(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) *Node {
	// Extract function name and parameters
	functionName := ""
	parameters := []string{}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		functionName = nameNode.Content(sourceCode)
	}

	parametersNode := node.ChildByFieldName("parameters")
	var methodArgumentsType []string
	if parametersNode != nil {
		for i := 0; i < int(parametersNode.NamedChildCount()); i++ {
			param := parametersNode.NamedChild(i)
			switch param.Type() {
			case "identifier", "typed_parameter", "default_parameter", "typed_default_parameter":
				parameters = append(parameters, param.Content(sourceCode))
			}
			// Extract typed parameters for MethodArgumentsType in "name: type" format.
			switch param.Type() {
			case "typed_parameter":
				// "a: int" → use full content directly.
				methodArgumentsType = append(methodArgumentsType, param.Content(sourceCode))
			case "typed_default_parameter":
				// "port: int = 8080" → extract "port: int" (name + type, skip default).
				nameNode := param.ChildByFieldName("name")
				typeNode := param.ChildByFieldName("type")
				if nameNode != nil && typeNode != nil {
					methodArgumentsType = append(methodArgumentsType, nameNode.Content(sourceCode)+": "+typeNode.Content(sourceCode))
				}
			}
		}
	}

	// Extract return type annotation (e.g., "-> int" produces "int").
	returnType := ""
	returnTypeNode := node.ChildByFieldName("return_type")
	if returnTypeNode != nil {
		returnType = returnTypeNode.Content(sourceCode)
	}

	// Determine node type based on function characteristics.
	nodeType := "function_definition"

	// Check if function is inside a class (method vs function).
	isInsideClass := currentContext != nil && (currentContext.Type == "class_definition" ||
		currentContext.Type == "interface" ||
		currentContext.Type == "enum" ||
		currentContext.Type == "dataclass")

	// Check if function is nested inside another function.
	isNestedFunction := currentContext != nil && (currentContext.Type == "function_definition" ||
		currentContext.Type == "method" ||
		currentContext.Type == "property" ||
		currentContext.Type == "constructor" ||
		currentContext.Type == "special_method")

	// Build qualified function name for nested functions.
	// Nested functions should have FQNs like parent.child.grandchild.
	qualifiedFunctionName := functionName
	if isNestedFunction && currentContext.Name != "" {
		qualifiedFunctionName = currentContext.Name + "." + functionName
	}

	// Determine function type based on characteristics (priority order).
	switch {
	case isConstructor(functionName):
		nodeType = "constructor"
	case isSpecialMethod(functionName):
		// Special methods like __str__, __add__, __call__.
		nodeType = "special_method"
	case isInsideClass:
		// Regular function inside a class is a method.
		nodeType = "method"
	}

	// Check for decorators (parent might be decorated_definition).
	var decorators []string
	if node.Parent() != nil && node.Parent().Type() == "decorated_definition" {
		decorators = extractDecorators(node.Parent(), sourceCode)

		// If function has @property decorator, mark it as property type.
		if hasDecorator(decorators, "property") {
			nodeType = "property"
		}
	}

	lineNumber := node.StartPoint().Row + 1
	methodID := GenerateMethodID("function:"+qualifiedFunctionName, parameters, file, lineNumber)
	functionNode := &Node{
		ID:   methodID,
		Type: nodeType,
		Name: qualifiedFunctionName,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:           lineNumber,
		ReturnType:           returnType,
		MethodArgumentsType:  methodArgumentsType,
		MethodArgumentsValue: parameters,
		Annotation:           decorators,
		File:                 file,
		isPythonSourceFile:   true,
		Language:             "python",
	}
	graph.AddNode(functionNode)
	return functionNode
}

// parsePythonClassDefinition parses Python class definitions.
// Returns the class node to be used as context for nested definitions.
// Detects interfaces (Protocol/ABC), enums, and dataclasses.
func parsePythonClassDefinition(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) *Node {
	// Extract class name and bases
	className := ""
	superClasses := []string{}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		className = nameNode.Content(sourceCode)
	}

	superclassNode := node.ChildByFieldName("superclasses")
	if superclassNode != nil {
		for i := 0; i < int(superclassNode.NamedChildCount()); i++ {
			superClass := superclassNode.NamedChild(i)
			if superClass.Type() == "identifier" || superClass.Type() == "attribute" {
				superClasses = append(superClasses, superClass.Content(sourceCode))
			}
		}
	}

	// Determine class type based on inheritance and decorators.
	classType := "class_definition"

	// Check for interface (Protocol or ABC).
	if isInterface(superClasses) {
		classType = "interface"
	} else if isEnum(superClasses) {
		// Check for enum.
		classType = "enum"
	}

	// Check for decorators (parent might be decorated_definition).
	var decorators []string
	if node.Parent() != nil && node.Parent().Type() == "decorated_definition" {
		decorators = extractDecorators(node.Parent(), sourceCode)

		// Check if this is a dataclass.
		if isDataclass(decorators) {
			classType = "dataclass"
		}
	}

	classLineNumber := node.StartPoint().Row + 1
	classNode := &Node{
		ID:   GenerateMethodID("class:"+className, []string{}, file, classLineNumber),
		Type: classType,
		Name: className,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:         classLineNumber,
		Interface:          superClasses,
		Annotation:         decorators,
		File:               file,
		isPythonSourceFile: true,
		Language:           "python",
	}
	graph.AddNode(classNode)
	return classNode
}

// parsePythonCall parses Python function calls.
func parsePythonCall(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *Node, file string) {
	// Python function calls
	callName := ""
	arguments := []string{}

	functionNode := node.ChildByFieldName("function")
	if functionNode != nil {
		callName = functionNode.Content(sourceCode)
	}

	argumentsNode := node.ChildByFieldName("arguments")
	if argumentsNode != nil {
		for i := 0; i < int(argumentsNode.NamedChildCount()); i++ {
			arg := argumentsNode.NamedChild(i)
			arguments = append(arguments, arg.Content(sourceCode))
		}
	}

	callLineNumber := node.StartPoint().Row + 1
	callID := GenerateMethodID(callName, arguments, file, callLineNumber)
	callNode := &Node{
		ID:         callID,
		Type:       "call",
		Name:       callName,
		IsExternal: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:           callLineNumber,
		MethodArgumentsValue: arguments,
		File:                 file,
		isPythonSourceFile:   true,
		Language:             "python",
	}
	graph.AddNode(callNode)
	if currentContext != nil {
		graph.AddEdge(currentContext, callNode)
	}
}

// parsePythonReturnStatement parses Python return statements.
func parsePythonReturnStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	returnNode := pythonlang.ParseReturnStatement(node, sourceCode)
	uniqueReturnID := fmt.Sprintf("return_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	returnStmtNode := &Node{
		ID:         GenerateSha256(uniqueReturnID),
		Type:       "ReturnStmt",
		LineNumber: node.StartPoint().Row + 1,
		Name:       "ReturnStmt",
		IsExternal: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		Language:           "python",
		ReturnStmt:         returnNode,
	}
	graph.AddNode(returnStmtNode)
}

// parsePythonBreakStatement parses Python break statements.
func parsePythonBreakStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	breakNode := pythonlang.ParseBreakStatement(node, sourceCode)
	uniquebreakstmtID := fmt.Sprintf("breakstmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	breakStmtNode := &Node{
		ID:         GenerateSha256(uniquebreakstmtID),
		Type:       "BreakStmt",
		LineNumber: node.StartPoint().Row + 1,
		Name:       "BreakStmt",
		IsExternal: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		Language:           "python",
		BreakStmt:          breakNode,
	}
	graph.AddNode(breakStmtNode)
}

// parsePythonContinueStatement parses Python continue statements.
func parsePythonContinueStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	continueNode := pythonlang.ParseContinueStatement(node, sourceCode)
	uniquecontinueID := fmt.Sprintf("continuestmt_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	continueStmtNode := &Node{
		ID:         GenerateSha256(uniquecontinueID),
		Type:       "ContinueStmt",
		LineNumber: node.StartPoint().Row + 1,
		Name:       "ContinueStmt",
		IsExternal: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		Language:           "python",
		ContinueStmt:       continueNode,
	}
	graph.AddNode(continueStmtNode)
}

// parsePythonAssertStatement parses Python assert statements.
func parsePythonAssertStatement(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	assertNode := pythonlang.ParseAssertStatement(node, sourceCode)
	uniqueAssertID := fmt.Sprintf("assert_%d_%d_%s", node.StartPoint().Row+1, node.StartPoint().Column+1, file)
	assertStmtNode := &Node{
		ID:         GenerateSha256(uniqueAssertID),
		Type:       "AssertStmt",
		LineNumber: node.StartPoint().Row + 1,
		Name:       "AssertStmt",
		IsExternal: true,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		File:               file,
		isPythonSourceFile: true,
		Language:           "python",
		AssertStmt:         assertNode,
	}
	graph.AddNode(assertStmtNode)
}

// parsePythonYieldExpression parses Python yield expressions.
func parsePythonYieldExpression(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	// Handle yield expressions in Python
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "yield" {
			yieldNode := pythonlang.ParseYieldStatement(child, sourceCode)
			uniqueyieldID := fmt.Sprintf("yield_%d_%d_%s", child.StartPoint().Row+1, child.StartPoint().Column+1, file)
			yieldStmtNode := &Node{
				ID:         GenerateSha256(uniqueyieldID),
				Type:       "YieldStmt",
				LineNumber: child.StartPoint().Row + 1,
				Name:       "YieldStmt",
				IsExternal: true,
				SourceLocation: &SourceLocation{
					File:      file,
					StartByte: child.StartByte(),
					EndByte:   child.EndByte(),
				},
				File:               file,
				isPythonSourceFile: true,
				Language:           "python",
				YieldStmt:          yieldNode,
			}
			graph.AddNode(yieldStmtNode)
			break
		}
	}
}

// parsePythonAssignment parses Python variable assignments.
// Distinguishes between local variables, module-level variables, and constants.
// Only processes simple identifier assignments, skipping subscript and attribute assignments.
func parsePythonAssignment(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) {
	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return
	}

	// Only process simple identifier assignments (e.g., "x = 1", "CONSTANT = 'value'").
	// Skip subscript assignments (e.g., "dict['key'] = value", "DATABASES_ALL['default'] = {}").
	// Skip attribute assignments (e.g., "obj.field = value", "settings.DATA_MANAGER = {}").
	if leftNode.Type() == "subscript" || leftNode.Type() == "attribute" {
		return
	}

	variableName := leftNode.Content(sourceCode)

	variableValue := ""
	rightNode := node.ChildByFieldName("right")
	if rightNode != nil {
		variableValue = rightNode.Content(sourceCode)
	}

	// Determine variable type and scope.
	nodeType := "variable_assignment"
	scope := "local"

	// Check if this is a module-level variable (no current context).
	if currentContext == nil {
		scope = "module"
		nodeType = "module_variable"

		// Check if it's a constant (uppercase naming convention).
		if isConstantName(variableName) {
			nodeType = "constant"
		}
	} else if currentContext.Type == "class_definition" || currentContext.Type == "enum" ||
		currentContext.Type == "interface" || currentContext.Type == "dataclass" {
		// Assignment inside a class (including enum/interface/dataclass) but outside any method
		// is a class-level variable.
		scope = "class"
		nodeType = "class_field"

		// Check if it's a class-level constant (uppercase naming convention).
		if isConstantName(variableName) {
			nodeType = "constant"
		}
	}

	varLineNumber := node.StartPoint().Row + 1
	variableNode := &Node{
		ID:   GenerateMethodID(variableName, []string{}, file, varLineNumber),
		Type: nodeType,
		Name: variableName,
		SourceLocation: &SourceLocation{
			File:      file,
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
		},
		LineNumber:         varLineNumber,
		VariableValue:      variableValue,
		Scope:              scope,
		File:               file,
		isPythonSourceFile: true,
		Language:           "python",
	}
	graph.AddNode(variableNode)
}

// ResolveTransitiveInheritance resolves transitive inheritance for Python classes.
// This fixes the issue where classes inheriting from custom enum/interface/dataclass
// base classes are not properly detected.
//
// Example:
//
//	class CustomEnum(Enum):  # Detected as enum (direct inheritance)
//	    pass
//
//	class Operator(CustomEnum):  # NOT detected without this fix (transitive)
//	    pass
//
// After this function, Operator will also be marked as "enum".
func ResolveTransitiveInheritance(codeGraph *CodeGraph) {
	// Build a map of class name → class node for quick lookup.
	classMap := make(map[string]*Node)
	for _, node := range codeGraph.Nodes {
		if node.Type == "class_definition" || node.Type == "interface" ||
			node.Type == "enum" || node.Type == "dataclass" {
			classMap[node.Name] = node
		}
	}

	// Track which classes have been processed to avoid infinite loops.
	processed := make(map[string]bool)

	// Helper function to check if a class transitively inherits from a specific type.
	var inheritsFrom func(className string, targetType string) bool
	inheritsFrom = func(className string, targetType string) bool {
		// Prevent infinite recursion.
		if processed[className] {
			return false
		}
		processed[className] = true
		defer func() { processed[className] = false }()

		// Look up the class.
		classNode, exists := classMap[className]
		if !exists {
			return false
		}

		// If this class is already the target type, return true.
		if classNode.Type == targetType {
			return true
		}

		// Check all base classes.
		for _, baseClass := range classNode.Interface {
			// Extract just the class name (handle qualified names like typing.Protocol).
			parts := strings.Split(baseClass, ".")
			baseName := parts[len(parts)-1]

			// Recursively check if the base class is or inherits from the target type.
			if inheritsFrom(baseName, targetType) {
				return true
			}
		}

		return false
	}

	// Update class types based on transitive inheritance.
	for _, node := range codeGraph.Nodes {
		if node.Type != "class_definition" {
			continue
		}

		// Check if this class transitively inherits from enum.
		processed = make(map[string]bool) // Reset for each check.
		for _, baseClass := range node.Interface {
			parts := strings.Split(baseClass, ".")
			baseName := parts[len(parts)-1]

			if inheritsFrom(baseName, "enum") {
				node.Type = "enum"
				break
			}
		}

		// Check if this class transitively inherits from interface.
		if node.Type == "class_definition" {
			processed = make(map[string]bool)
			for _, baseClass := range node.Interface {
				parts := strings.Split(baseClass, ".")
				baseName := parts[len(parts)-1]

				if inheritsFrom(baseName, "interface") {
					node.Type = "interface"
					break
				}
			}
		}

		// Check if this class transitively inherits from dataclass.
		if node.Type == "class_definition" {
			processed = make(map[string]bool)
			for _, baseClass := range node.Interface {
				parts := strings.Split(baseClass, ".")
				baseName := parts[len(parts)-1]

				if inheritsFrom(baseName, "dataclass") {
					node.Type = "dataclass"
					break
				}
			}
		}
	}
}
