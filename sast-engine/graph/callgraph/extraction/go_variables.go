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
		"", // currentClassName (empty at start)
		typeEngine,
		registry,
		importMap,
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
	currentClassName string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
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
			}
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
			currentClassName,
			typeEngine,
			registry,
			importMap,
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

// processShortVarDeclaration processes a short_var_declaration node.
// Placeholder for now, will be implemented in next step.
func processShortVarDeclaration(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
) {
	// TODO: Implement in Step 4
}

// processAssignmentStatement processes an assignment_statement node.
// Placeholder for now, will be implemented in next step.
func processAssignmentStatement(
	node *sitter.Node,
	sourceCode []byte,
	filePath string,
	functionFQN string,
	typeEngine *resolution.GoTypeInferenceEngine,
	registry *core.GoModuleRegistry,
	importMap *core.GoImportMap,
) {
	// TODO: Implement in Step 4
}
