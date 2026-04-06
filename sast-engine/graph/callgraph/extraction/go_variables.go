package extraction

import (
	"context"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	golangpkg "github.com/shivasurya/code-pathfinder/sast-engine/graph/golang"
)

// ExtractGoVariableAssignments extracts variable assignments from a Go file
// and populates the type inference engine with inferred types.
//
// This function implements Pass 2b of the Go call graph construction pipeline.
// It processes variable assignments by inferring types from RHS expressions and
// storing the bindings in function scopes.
//
// Algorithm:
//  1. Parse source code with tree-sitter Go parser
//  2. Track function context during AST traversal
//  3. Find assignment nodes (short_var_declaration, assignment_statement)
//  4. For each assignment:
//     a) Extract LHS variable name(s)
//     b) Infer type from RHS expression:
//        - Function call: Look up return type in engine
//        - Literal: Infer builtin type
//        - Variable ref: Copy type from scope
//        - Struct literal: Extract type name
//     c) Create GoVariableBinding
//     d) Add binding to function scope
//
// RHS Type Inference Patterns:
//   - GetUser() → Look up GetUser return type
//   - "Alice" → builtin.string
//   - 42 → builtin.int
//   - true → builtin.bool
//   - user → Copy user's type from scope
//   - User{} → Extract User type
//   - &User{} → Extract User type (strip &)
//
// Parameters:
//   - filePath: Absolute path to the Go source file
//   - sourceCode: Contents of the file as byte array
//   - typeEngine: Type inference engine with return types populated (from Pass 2a)
//   - registry: Go module registry for package resolution
//   - importMap: Import mappings for resolving qualified type names
//
// Returns:
//   - error: If parsing fails or other critical errors occur
//
// Example:
//
//	engine := resolution.NewGoTypeInferenceEngine(registry)
//	// After Pass 2a (return type extraction)
//	err := ExtractGoVariableAssignments(filePath, sourceCode, engine, registry, importMap)
//	// Now engine.Scopes contains variable bindings for each function
//
// Thread Safety:
//   This function is thread-safe because typeEngine operations use mutexes.
//   Can be called in parallel for different files.
func ExtractGoVariableAssignments(
	filePath string,
	sourceCode []byte,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) error {
	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return err
	}
	defer tree.Close()

	// Get package path for this file
	dirPath := filepath.Dir(filePath)

	// Convert to absolute path if relative (for registry lookup)
	if !filepath.IsAbs(dirPath) {
		if absPath, err := filepath.Abs(dirPath); err == nil {
			dirPath = absPath
		}
	}

	packagePath, exists := registry.DirToImport[dirPath]
	if !exists {
		// File not in registry (e.g., external dependency), skip
		return nil
	}

	// Traverse AST to find variable assignments
	traverseForVariableAssignments(
		tree.RootNode(),
		sourceCode,
		filePath,
		packagePath,
		"", // currentFunctionFQN (empty at start)
		typeEngine,
		registry,
		importMap,
		callGraph,
	)

	return nil
}

// traverseForVariableAssignments recursively traverses the AST to find variable assignments.
// Tracks function context to properly scope variable bindings.
func traverseForVariableAssignments(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	packagePath string,
	currentFunctionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) {
	if node == nil {
		return
	}

	nodeType := node.Type()

	// Track function context
	switch nodeType {
	case "function_declaration":
		// Regular function: packagePath.FunctionName
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			funcName := nameNode.Content(sourceCode)
			currentFunctionFQN = packagePath + "." + funcName
		}

	case "method_declaration":
		// Method: packagePath.ClassName.MethodName
		nameNode := node.ChildByFieldName("name")
		receiverNode := node.ChildByFieldName("receiver")
		if nameNode != nil && receiverNode != nil {
			methodName := nameNode.Content(sourceCode)
			// Extract receiver type
			receiverType := extractReceiverType(receiverNode, sourceCode)
			if receiverType != "" {
				currentFunctionFQN = packagePath + "." + receiverType + "." + methodName

				// Add receiver variable as a typed binding so that calls like s.foo()
				// inside the method body can be resolved via type inference.
				receiverName := extractReceiverName(receiverNode, sourceCode)
				if receiverName != "" {
					scope := typeEngine.GetScope(currentFunctionFQN)
					if scope == nil {
						scope = resolution.NewGoFunctionScope(currentFunctionFQN)
						typeEngine.AddScope(scope)
					}
					scope.AddVariable(&resolution.GoVariableBinding{
						VarName: receiverName,
						Type: &core.TypeInfo{
							TypeFQN:    packagePath + "." + receiverType,
							Confidence: 0.95,
							Source:     "receiver_declaration",
						},
					})
				}
			}
		}

	case "var_declaration":
		// Handle explicit variable declaration: var sb strings.Builder
		if currentFunctionFQN != "" {
			processVarDeclaration(
				node,
				sourceCode,
				filePath,
				currentFunctionFQN,
				typeEngine,
				registry,
				importMap,
				callGraph,
			)
		}

	case "short_var_declaration":
		// Handle short variable declaration: x := value
		if currentFunctionFQN != "" {
			processShortVarDeclaration(
				node,
				sourceCode,
				filePath,
				currentFunctionFQN,
				typeEngine,
				registry,
				importMap,
				callGraph,
			)
		}

	case "assignment_statement":
		// Handle regular assignment: x = value
		if currentFunctionFQN != "" {
			processAssignmentStatement(
				node,
				sourceCode,
				filePath,
				currentFunctionFQN,
				typeEngine,
				registry,
				importMap,
				callGraph,
			)
		}
	}

	// Recursively traverse children
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		traverseForVariableAssignments(
			child,
			sourceCode,
			filePath,
			packagePath,
			currentFunctionFQN,
			typeEngine,
			registry,
			importMap,
			callGraph,
		)
	}
}

// extractReceiverType extracts the type name from a receiver node.
// Handles both value and pointer receivers: (u User) or (u *User).
func extractReceiverType(receiverNode *sitter.Node, sourceCode []byte) string {
	// Receiver is a parameter_list with one parameter
	for i := 0; i < int(receiverNode.NamedChildCount()); i++ {
		param := receiverNode.NamedChild(i)
		if param.Type() == "parameter_declaration" {
			typeNode := param.ChildByFieldName("type")
			if typeNode != nil {
				typeName := typeNode.Content(sourceCode)
				// Strip pointer prefix if present
				typeName = strings.TrimPrefix(typeName, "*")
				return typeName
			}
		}
	}
	return ""
}

// extractReceiverName extracts the variable name from a receiver node.
// For `func (s *Store) Method()` it returns "s".
// Returns "" when the receiver is unnamed (e.g. `func (*Store) Method()`).
func extractReceiverName(receiverNode *sitter.Node, sourceCode []byte) string {
	for i := 0; i < int(receiverNode.NamedChildCount()); i++ {
		param := receiverNode.NamedChild(i)
		if param.Type() == "parameter_declaration" {
			nameNode := param.ChildByFieldName("name")
			if nameNode != nil {
				return nameNode.Content(sourceCode)
			}
		}
	}
	return ""
}

// processVarDeclaration processes a var_declaration node.
// Handles: var sb strings.Builder, var x, y int, var ( a T; b U ).
func processVarDeclaration(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		switch child.Type() {
		case "var_spec":
			processVarSpec(child, sourceCode, filePath, functionFQN, typeEngine, registry, importMap, callGraph)
		case "var_spec_list":
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() == "var_spec" {
					processVarSpec(spec, sourceCode, filePath, functionFQN, typeEngine, registry, importMap, callGraph)
				}
			}
		}
	}
}

// processVarSpec processes a single var_spec node inside a var_declaration.
func processVarSpec(
	spec *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) {
	// Collect variable names (may be multiple: var x, y int)
	var names []string
	for i := 0; i < int(spec.NamedChildCount()); i++ {
		child := spec.NamedChild(i)
		if child.Type() == "identifier" {
			name := child.Content(sourceCode)
			if name != "_" {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		return
	}

	var typeInfo *core.TypeInfo

	// Priority 1: Explicit type annotation (var sb strings.Builder)
	typeNode := spec.ChildByFieldName("type")
	if typeNode != nil {
		typeStr := strings.TrimPrefix(typeNode.Content(sourceCode), "*")
		typeFQN := extractionResolveGoTypeFQN(typeStr, importMap)
		// Unqualified type — belongs to same package
		if typeFQN == typeStr && !strings.Contains(typeFQN, ".") {
			dirPath := filepath.Dir(filePath)
			if pkgPath, ok := registry.DirToImport[dirPath]; ok {
				typeFQN = pkgPath + "." + typeFQN
			}
		}
		if typeFQN != "" && typeFQN != typeStr {
			typeInfo = &core.TypeInfo{
				TypeFQN:    typeFQN,
				Confidence: 0.9,
				Source:     "var_declaration",
			}
		}
	}

	// Priority 2: Infer from RHS value expression (var x = someFunc())
	if typeInfo == nil {
		valueNode := spec.ChildByFieldName("value")
		if valueNode != nil {
			typeInfo = inferTypeFromRHS(valueNode, sourceCode, filePath, functionFQN, typeEngine, registry, importMap, callGraph)
		}
	}

	if typeInfo == nil {
		return
	}

	scope := typeEngine.GetScope(functionFQN)
	if scope == nil {
		scope = resolution.NewGoFunctionScope(functionFQN)
		typeEngine.AddScope(scope)
	}
	for _, name := range names {
		scope.AddVariable(&resolution.GoVariableBinding{
			VarName: name,
			Type:    typeInfo,
			Location: resolution.Location{
				File: filePath,
				Line: spec.StartPoint().Row + 1,
			},
		})
	}
}

// processShortVarDeclaration processes a short_var_declaration node.
// Extracts variable names and infers types from RHS.
func processShortVarDeclaration(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) {
	// Use existing helper to extract variable info
	varInfos := golangpkg.ParseShortVarDeclaration(node, sourceCode)
	if len(varInfos) == 0 {
		return
	}

	// Get RHS node for type inference
	rhsNode := node.ChildByFieldName("right")
	if rhsNode == nil {
		return
	}

	// For multi-assignment (x, y := foo()), all variables get same type
	// For single assignment (x := foo()), just one variable
	for _, varInfo := range varInfos {
		// Skip blank identifier
		if varInfo.Name == "_" {
			continue
		}

		// Infer type from RHS
		typeInfo := inferTypeFromRHS(
			rhsNode,
			sourceCode,
			filePath,
			functionFQN,
			typeEngine,
			registry,
			importMap,
			callGraph,
		)

		if typeInfo == nil {
			// Could not infer type, skip
			continue
		}

		// Create variable binding
		binding := &resolution.GoVariableBinding{
			VarName:      varInfo.Name,
			Type:         typeInfo,
			AssignedFrom: varInfo.Value,
			Location: resolution.Location{
				File: filePath,
				Line: varInfo.LineNumber,
			},
		}

		// Get or create function scope
		scope := typeEngine.GetScope(functionFQN)
		if scope == nil {
			scope = resolution.NewGoFunctionScope(functionFQN)
			typeEngine.AddScope(scope)
		}

		// Add variable binding to scope
		scope.AddVariable(binding)
	}
}

// processAssignmentStatement processes an assignment_statement node.
// Handles reassignments: x = value.
func processAssignmentStatement(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) {
	// Use existing helper to extract variable info
	varInfos := golangpkg.ParseAssignment(node, sourceCode)
	if len(varInfos) == 0 {
		return
	}

	// Get RHS node for type inference
	rhsNode := node.ChildByFieldName("right")
	if rhsNode == nil {
		return
	}

	// Process each LHS variable
	for _, varInfo := range varInfos {
		// Infer type from RHS
		typeInfo := inferTypeFromRHS(
			rhsNode,
			sourceCode,
			filePath,
			functionFQN,
			typeEngine,
			registry,
			importMap,
			callGraph,
		)

		if typeInfo == nil {
			// Could not infer type, skip
			continue
		}

		// Create variable binding (allows multiple bindings for reassignments)
		binding := &resolution.GoVariableBinding{
			VarName:      varInfo.Name,
			Type:         typeInfo,
			AssignedFrom: varInfo.Value,
			Location: resolution.Location{
				File: filePath,
				Line: varInfo.LineNumber,
			},
		}

		// Get or create function scope
		scope := typeEngine.GetScope(functionFQN)
		if scope == nil {
			scope = resolution.NewGoFunctionScope(functionFQN)
			typeEngine.AddScope(scope)
		}

		// Add variable binding to scope
		scope.AddVariable(binding)
	}
}

// inferTypeFromRHS infers the type from a RHS expression node.
// Returns nil if type cannot be inferred.
//
// Handles:
//   - Function calls: Look up return type
//   - Literals: Return builtin type
//   - Variable refs: Copy type from scope
//   - Struct literals: Extract type name
//   - Address-of operator: Recurse on operand
func inferTypeFromRHS(
	rhsNode *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) *core.TypeInfo {
	if rhsNode == nil {
		return nil
	}

	nodeType := rhsNode.Type()

	// Handle different RHS patterns
	switch nodeType {
	// String literals
	case "interpreted_string_literal", "raw_string_literal":
		return &core.TypeInfo{
			TypeFQN:    "builtin.string",
			Confidence: 1.0,
			Source:     "literal",
		}

	// Numeric literals
	case "int_literal":
		return &core.TypeInfo{
			TypeFQN:    "builtin.int",
			Confidence: 1.0,
			Source:     "literal",
		}

	case "float_literal":
		return &core.TypeInfo{
			TypeFQN:    "builtin.float64",
			Confidence: 1.0,
			Source:     "literal",
		}

	case "imaginary_literal":
		return &core.TypeInfo{
			TypeFQN:    "builtin.complex128",
			Confidence: 1.0,
			Source:     "literal",
		}

	// Boolean literals
	case "true", "false":
		return &core.TypeInfo{
			TypeFQN:    "builtin.bool",
			Confidence: 1.0,
			Source:     "literal",
		}

	// Rune literal
	case "rune_literal":
		return &core.TypeInfo{
			TypeFQN:    "builtin.rune",
			Confidence: 1.0,
			Source:     "literal",
		}

	// Nil literal
	case "nil":
		return &core.TypeInfo{
			TypeFQN:    "builtin.nil",
			Confidence: 1.0,
			Source:     "literal",
		}

	// Function call - look up return type
	case "call_expression":
		if result := inferTypeFromFunctionCall(rhsNode, sourceCode, filePath, typeEngine, registry, importMap); result != nil {
			return result
		}
		// Param-aware fallback: if RHS is obj.Method() and obj is a function parameter,
		// resolve the method's return type via StdlibLoader / ThirdPartyLoader.
		return inferTypeFromParamMethodCall(rhsNode, sourceCode, functionFQN, callGraph, registry, importMap)

	// Variable reference - copy type from scope
	case "identifier":
		varName := rhsNode.Content(sourceCode)
		return inferTypeFromVariable(varName, functionFQN, typeEngine)

	// Struct literal - extract type name
	case "composite_literal":
		return inferTypeFromCompositeLiteral(
			rhsNode,
			sourceCode,
			filePath,
			registry,
		)

	// Unary expression - handle address-of operator
	case "unary_expression":
		return inferTypeFromUnaryExpression(
			rhsNode,
			sourceCode,
			filePath,
			functionFQN,
			typeEngine,
			registry,
			importMap,
			callGraph,
		)

	// Expression list - for multi-assignment, get first element
	case "expression_list":
		if rhsNode.NamedChildCount() > 0 {
			firstChild := rhsNode.NamedChild(0)
			return inferTypeFromRHS(
				firstChild,
				sourceCode,
				filePath,
				functionFQN,
				typeEngine,
				registry,
				importMap,
				callGraph,
			)
		}
		return nil

	default:
		return nil
	}
}

// inferTypeFromFunctionCall infers type from a function call expression.
// Looks up the function's return type in the TypeInferenceEngine.
// Falls back to the stdlib registry when the engine has no entry (Pass 2a only
// covers project-defined functions).
func inferTypeFromFunctionCall(
	callNode *sitter.Node,
	sourceCode []byte,
	filePath string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
) *core.TypeInfo {
	// Extract function name from call_expression
	functionNode := callNode.ChildByFieldName("function")
	if functionNode == nil {
		return nil
	}

	funcName := extractGoFunctionName(functionNode, sourceCode, importMap)
	if funcName == "" {
		return nil
	}

	// For unqualified function names (no package prefix), qualify with current package
	if !strings.Contains(funcName, ".") {
		// Get current package path
		dirPath := filepath.Dir(filePath)
		if packagePath, ok := registry.DirToImport[dirPath]; ok {
			funcName = packagePath + "." + funcName
		}
	}

	// Look up return type in engine (populated by Pass 2a)
	if typeInfo, ok := typeEngine.GetReturnType(funcName); ok {
		return typeInfo
	}

	// Fallback: attempt stdlib lookup for cross-package stdlib calls.
	// funcName is "importPath.FuncName" when the importMap resolved the qualifier,
	// e.g., "net/http.NewRequest", "fmt.Sprintf".
	if idx := strings.LastIndex(funcName, "."); idx > 0 {
		importPath := funcName[:idx]
		fnName := funcName[idx+1:]
		if ti := inferTypeFromStdlibFunction(importPath, fnName, registry); ti != nil {
			return ti
		}
		// Fallback: attempt third-party lookup for non-stdlib cross-package calls.
		if ti := inferTypeFromThirdPartyFunction(importPath, fnName, registry); ti != nil {
			return ti
		}
	}

	// Function not found or has no return type
	return nil
}

// inferTypeFromStdlibFunction looks up the primary return type of a Go stdlib
// function using the StdlibLoader attached to the registry.
//
// Parameters:
//   - importPath: full stdlib import path (e.g., "net/http", "fmt")
//   - funcName: exported function name (e.g., "NewRequest", "Sprintf")
//   - registry: Go module registry (provides StdlibLoader)
//
// Returns nil when the loader is unavailable, the import path is not a stdlib
// package, the function is not found, or there is no usable return type.
func inferTypeFromStdlibFunction(importPath, funcName string, registry *core.GoModuleRegistry) *core.TypeInfo {
	if registry.StdlibLoader == nil {
		return nil
	}
	if !registry.StdlibLoader.ValidateStdlibImport(importPath) {
		return nil
	}
	fn, err := registry.StdlibLoader.GetFunction(importPath, funcName)
	if err != nil || fn == nil || len(fn.Returns) == 0 {
		return nil
	}
	// Use the first non-error, non-empty return value.
	for _, ret := range fn.Returns {
		if ret.Type == "" || ret.Type == "error" {
			continue
		}
		typeFQN := normalizeStdlibReturnType(ret.Type, importPath)
		if typeFQN == "" {
			continue
		}
		return &core.TypeInfo{
			TypeFQN:    typeFQN,
			Confidence: 0.9,
			Source:     "stdlib_registry",
		}
	}
	return nil
}

// normalizeStdlibReturnType converts a raw stdlib return type string into a TypeFQN.
//
// rawType is the type string as stored in the registry JSON (e.g., "*Request",
// "string", "error", "io.Reader"). importPath is the package the function belongs
// to, used to qualify unqualified type names.
//
// Examples:
//   - "*Request",  "net/http"  → "net/http.Request"
//   - "File",      "os"        → "os.File"
//   - "string",    "fmt"       → "builtin.string"
//   - "error",     "os"        → "builtin.error"
//   - "io.Reader", "net/http"  → "io.Reader"
//   - "[]byte",    "os"        → "builtin.byte"
func normalizeStdlibReturnType(rawType, importPath string) string {
	t := rawType
	// Strip pointer and slice qualifiers.
	t = strings.TrimPrefix(t, "*")
	t = strings.TrimPrefix(t, "[]")
	if t == "" {
		return ""
	}
	// Handle builtin types.
	switch t {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error":
		return "builtin." + t
	}
	// Cross-package reference (e.g., "io.Reader", "os.FileInfo"):
	// leave as-is; the caller can resolve the package alias in a future pass.
	if strings.Contains(t, ".") {
		return t
	}
	// Unqualified type name — belongs to the function's own package.
	return importPath + "." + t
}

// inferTypeFromThirdPartyFunction looks up the primary return type of a Go
// third-party function using the ThirdPartyLoader attached to the registry.
func inferTypeFromThirdPartyFunction(importPath, funcName string, registry *core.GoModuleRegistry) *core.TypeInfo {
	if registry.ThirdPartyLoader == nil {
		return nil
	}
	if !registry.ThirdPartyLoader.ValidateImport(importPath) {
		return nil
	}
	fn, err := registry.ThirdPartyLoader.GetFunction(importPath, funcName)
	if err != nil || fn == nil || len(fn.Returns) == 0 {
		return nil
	}
	for _, ret := range fn.Returns {
		if ret.Type == "" || ret.Type == "error" {
			continue
		}
		typeFQN := normalizeStdlibReturnType(ret.Type, importPath)
		if typeFQN == "" {
			continue
		}
		return &core.TypeInfo{
			TypeFQN:    typeFQN,
			Confidence: 0.85,
			Source:     "thirdparty_local",
		}
	}
	return nil
}

// inferTypeFromParamMethodCall resolves the return type of a method call where the
// receiver is a function parameter (e.g. r.FormValue("id") when r is *http.Request).
//
// This is the param-aware fallback in inferTypeFromRHS: it fires only when the
// standard inferTypeFromFunctionCall path returned nil (i.e., the receiver is not
// a package alias and not tracked as a :=-variable in the scope).
//
// Resolution order follows the Check 2 / Check 2.5 precedence:
//  1. StdlibLoader — for stdlib types (e.g. net/http.Request)
//  2. ThirdPartyLoader — for vendored/GOMODCACHE types (e.g. gin.Context)
func inferTypeFromParamMethodCall(
	callNode *sitter.Node,
	sourceCode []byte,
	functionFQN string,
	callGraph *core.CallGraph,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
) *core.TypeInfo {
	if callGraph == nil || callNode == nil {
		return nil
	}

	// Must be a selector_expression receiver: obj.Method(...)
	funcNode := callNode.ChildByFieldName("function")
	if funcNode == nil || funcNode.Type() != "selector_expression" {
		return nil
	}

	operandNode := funcNode.ChildByFieldName("operand")
	fieldNode := funcNode.ChildByFieldName("field")
	if operandNode == nil || fieldNode == nil {
		return nil
	}

	objectName := operandNode.Content(sourceCode)
	methodName := fieldNode.Content(sourceCode)

	// If the operand is a known package alias, it was already handled by inferTypeFromFunctionCall.
	if importMap != nil {
		if _, ok := importMap.Imports[objectName]; ok {
			return nil
		}
	}

	// Look up the enclosing function's parameter list.
	callerNode, ok := callGraph.Functions[functionFQN]
	if !ok || callerNode == nil {
		return nil
	}

	for i, paramName := range callerNode.MethodArgumentsValue {
		if paramName != objectName || i >= len(callerNode.MethodArgumentsType) {
			continue
		}

		typeStr := callerNode.MethodArgumentsType[i]
		// Strip "name: " prefix that the parser sometimes prepends.
		if colonIdx := strings.Index(typeStr, ": "); colonIdx >= 0 {
			typeStr = typeStr[colonIdx+2:]
		}
		// Strip pointer qualifier — we look up the base type.
		typeStr = strings.TrimPrefix(typeStr, "*")

		// Resolve short qualifier (e.g. "http.Request" → "net/http.Request").
		paramTypeFQN := extractionResolveGoTypeFQN(typeStr, importMap)

		importPath, typeName, split := extractionSplitGoTypeFQN(paramTypeFQN)
		if !split {
			continue
		}

		// Check StdlibLoader first, then ThirdPartyLoader.
		var method *core.GoStdlibFunction
		if registry.StdlibLoader != nil {
			if t, err := registry.StdlibLoader.GetType(importPath, typeName); err == nil && t != nil {
				method = t.Methods[methodName]
			}
		}
		if method == nil && registry.ThirdPartyLoader != nil {
			if t, err := registry.ThirdPartyLoader.GetType(importPath, typeName); err == nil && t != nil {
				method = t.Methods[methodName]
			}
		}
		if method == nil || len(method.Returns) == 0 {
			continue
		}

		for _, ret := range method.Returns {
			if ret.Type == "" || ret.Type == "error" {
				continue
			}
			return &core.TypeInfo{
				TypeFQN:    normalizeStdlibReturnType(ret.Type, importPath),
				Confidence: 0.85,
				Source:     "method_return_type",
			}
		}
	}

	return nil
}

// extractionSplitGoTypeFQN splits a fully-qualified Go type name into its package
// import path and type name.  Duplicated from builder/helpers.go to avoid an
// import cycle (extraction → builder → extraction).
func extractionSplitGoTypeFQN(typeFQN string) (importPath, typeName string, ok bool) {
	if typeFQN == "" {
		return "", "", false
	}
	lastDot := strings.LastIndex(typeFQN, ".")
	if lastDot < 0 || lastDot == len(typeFQN)-1 {
		return "", "", false
	}
	return typeFQN[:lastDot], typeFQN[lastDot+1:], true
}

// extractionResolveGoTypeFQN resolves a short Go type name to a fully-qualified
// import path using the file's import map.  Duplicated from builder/helpers.go to
// avoid an import cycle.
func extractionResolveGoTypeFQN(shortType string, importMap *core.GoImportMap) string {
	if shortType == "" || importMap == nil {
		return shortType
	}
	dotIdx := strings.Index(shortType, ".")
	if dotIdx < 0 {
		return shortType
	}
	alias := shortType[:dotIdx]
	rest := shortType[dotIdx+1:]
	importPath, ok := importMap.Resolve(alias)
	if !ok {
		return shortType
	}
	return importPath + "." + rest
}

// extractGoFunctionName extracts the function name from a function node.
// Handles:
//   - Simple calls: foo()
//   - Qualified calls: pkg.Foo()
//   - Method calls: obj.Method() (returns Method)
func extractGoFunctionName(
	funcNode *sitter.Node,
	sourceCode []byte,
	importMap *core.GoImportMap,
) string {
	nodeType := funcNode.Type()

	switch nodeType {
	case "identifier":
		// Simple function call: foo()
		return funcNode.Content(sourceCode)

	case "selector_expression":
		// Qualified call: pkg.Foo() or obj.Method()
		// Get the selector (Foo or Method)
		fieldNode := funcNode.ChildByFieldName("field")
		if fieldNode == nil {
			return ""
		}

		fieldName := fieldNode.Content(sourceCode)

		// Get the operand (pkg or obj)
		operandNode := funcNode.ChildByFieldName("operand")
		if operandNode == nil {
			return fieldName
		}

		operandName := operandNode.Content(sourceCode)

		// Check if operand is a package name in imports
		if importMap != nil {
			if importPath, ok := importMap.Imports[operandName]; ok {
				// It's a package: return importPath.FunctionName
				return importPath + "." + fieldName
			}
		}

		// Could be a method call (obj.Method) or unknown qualified call
		// Return just the method name for now
		// The actual resolution will happen in PR-17
		return fieldName

	default:
		return ""
	}
}

// inferTypeFromVariable infers type by looking up the variable in the current function scope.
// Returns the most recent binding for the variable.
func inferTypeFromVariable(
	varName string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
) *core.TypeInfo {
	// Get function scope
	scope := typeEngine.GetScope(functionFQN)
	if scope == nil {
		return nil
	}

	// Get variable bindings
	bindings, ok := scope.Variables[varName]
	if !ok || len(bindings) == 0 {
		return nil
	}

	// Return most recent binding
	return bindings[len(bindings)-1].Type
}

// inferTypeFromCompositeLiteral infers type from a composite literal (struct literal).
// Handles: User{...}, &Config{...}, pkg.Type{...}.
func inferTypeFromCompositeLiteral(
	literalNode *sitter.Node,
	sourceCode []byte,
	filePath string,
	registry *core.GoModuleRegistry,
) *core.TypeInfo {
	// Get type node from composite literal
	typeNode := literalNode.ChildByFieldName("type")
	if typeNode == nil {
		return nil
	}

	typeName := typeNode.Content(sourceCode)

	// Parse the type name using existing parser from PR-14
	typeInfo, err := ParseGoTypeString(typeName, registry, filePath)
	if err != nil {
		return nil
	}
	return typeInfo
}

// inferTypeFromUnaryExpression infers type from a unary expression.
// Primarily handles address-of operator: &User{...}.
func inferTypeFromUnaryExpression(
	unaryNode *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
	callGraph *core.CallGraph,
) *core.TypeInfo {
	// Check operator
	operatorNode := unaryNode.ChildByFieldName("operator")
	if operatorNode == nil {
		return nil
	}

	operator := operatorNode.Content(sourceCode)

	// Get operand
	operandNode := unaryNode.ChildByFieldName("operand")
	if operandNode == nil {
		return nil
	}

	switch operator {
	case "&":
		// Address-of operator: &User{...}
		// Infer type from operand (the result will be the same type,
		// pointer handling is done by ParseGoTypeString which strips *)
		return inferTypeFromRHS(
			operandNode,
			sourceCode,
			filePath,
			functionFQN,
			typeEngine,
			registry,
			importMap,
			callGraph,
		)

	default:
		// Other unary operators (!, -, +, etc.) - not handling for now
		return nil
	}
}
