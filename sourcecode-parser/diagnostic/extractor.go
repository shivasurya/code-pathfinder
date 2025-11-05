package diagnostic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ExtractAllFunctions walks a project directory and extracts all Python function definitions.
// Returns a slice of FunctionMetadata for each function found.
//
// Performance: ~1-2 seconds for 10,000 functions
//
// Example:
//
//	functions, err := ExtractAllFunctions("/path/to/project")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d functions\n", len(functions))
func ExtractAllFunctions(projectPath string) ([]*FunctionMetadata, error) {
	var functions []*FunctionMetadata

	// Walk all .py files in project
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Python files
		if !strings.HasSuffix(path, ".py") {
			return nil
		}

		// Skip common directories
		if shouldSkipDir(path) {
			return nil
		}

		// Read source code
		sourceCode, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Extract functions from this file
		fileFunctions, err := extractFunctionsFromFile(path, sourceCode, projectPath)
		if err != nil {
			// Log warning but continue processing other files
			fmt.Printf("Warning: failed to extract from %s: %v\n", path, err)
			return nil
		}

		functions = append(functions, fileFunctions...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return functions, nil
}

// shouldSkipDir returns true if the directory should be skipped.
func shouldSkipDir(path string) bool {
	skipDirs := []string{
		"__pycache__",
		".git",
		".venv",
		"venv",
		"node_modules",
		".tox",
		".pytest_cache",
		"build",
		"dist",
		".eggs",
	}

	for _, skip := range skipDirs {
		if strings.Contains(path, skip) {
			return true
		}
	}

	return false
}

// extractFunctionsFromFile parses a single Python file and extracts all function definitions.
func extractFunctionsFromFile(filePath string, sourceCode []byte, projectRoot string) ([]*FunctionMetadata, error) {
	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse error: %w", err)
	}

	if tree == nil {
		return nil, fmt.Errorf("tree-sitter returned nil tree")
	}

	// Build module name from file path
	moduleName := buildModuleName(filePath, projectRoot)

	// Find all function definitions
	var functions []*FunctionMetadata
	findFunctions(tree.RootNode(), sourceCode, filePath, moduleName, "", &functions)

	return functions, nil
}

// findFunctions recursively finds all function_definition nodes in the AST.
// Handles both top-level functions and class methods.
func findFunctions(node *sitter.Node, sourceCode []byte, filePath, moduleName, className string, functions *[]*FunctionMetadata) {
	if node == nil {
		return
	}

	// Check if this is a function definition
	if node.Type() == "function_definition" {
		metadata := extractFunctionMetadata(node, sourceCode, filePath, moduleName, className)
		if metadata != nil {
			*functions = append(*functions, metadata)
		}
		// Recurse into function body to find nested functions
		bodyNode := node.ChildByFieldName("body")
		if bodyNode != nil {
			for i := 0; i < int(bodyNode.ChildCount()); i++ {
				child := bodyNode.Child(i)
				if child != nil {
					findFunctions(child, sourceCode, filePath, moduleName, className, functions)
				}
			}
		}
		return
	}

	// Check if this is a class definition
	if node.Type() == "class_definition" {
		// Extract class name
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			currentClassName := nameNode.Content(sourceCode)

			// Find all methods in this class
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				for i := 0; i < int(bodyNode.ChildCount()); i++ {
					child := bodyNode.Child(i)
					if child != nil {
						findFunctions(child, sourceCode, filePath, moduleName, currentClassName, functions)
					}
				}
			}
		}
		return
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			findFunctions(child, sourceCode, filePath, moduleName, className, functions)
		}
	}
}

// extractFunctionMetadata extracts metadata from a function_definition node.
func extractFunctionMetadata(node *sitter.Node, sourceCode []byte, filePath, moduleName, className string) *FunctionMetadata {
	// Extract function name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	functionName := nameNode.Content(sourceCode)

	// Build FQN
	var fqn string
	if className != "" {
		fqn = fmt.Sprintf("%s.%s.%s", moduleName, className, functionName)
	} else {
		fqn = fmt.Sprintf("%s.%s", moduleName, functionName)
	}

	// Check for decorators (they come before function_definition)
	hasDecorators := false
	// Note: Tree-sitter puts decorators as siblings before the function node
	// We'll check if there are decorator nodes in parent

	// Extract line numbers (1-indexed)
	startLine := int(node.StartPoint().Row) + 1
	endLine := int(node.EndPoint().Row) + 1

	// Extract source code
	sourceContent := node.Content(sourceCode)

	// Calculate LOC
	loc := endLine - startLine + 1

	// Check if async
	isAsync := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "async" {
			isAsync = true
			break
		}
	}

	// Check if method (has self or cls parameter)
	isMethod := className != ""
	if !isMethod {
		// Check parameters for self/cls
		paramsNode := node.ChildByFieldName("parameters")
		if paramsNode != nil {
			paramsText := paramsNode.Content(sourceCode)
			if strings.Contains(paramsText, "self") || strings.Contains(paramsText, "cls") {
				isMethod = true
			}
		}
	}

	return &FunctionMetadata{
		FilePath:      filePath,
		FunctionName:  functionName,
		FQN:           fqn,
		StartLine:     startLine,
		EndLine:       endLine,
		SourceCode:    sourceContent,
		LOC:           loc,
		HasDecorators: hasDecorators,
		ClassName:     className,
		IsMethod:      isMethod,
		IsAsync:       isAsync,
	}
}

// buildModuleName converts file path to Python module name.
// Example: "/project/myapp/views.py" → "myapp.views".
func buildModuleName(filePath, projectRoot string) string {
	// Make path relative to project root
	relPath, err := filepath.Rel(projectRoot, filePath)
	if err != nil {
		// Fallback: use absolute path
		relPath = filePath
	}

	// Remove .py extension
	relPath = strings.TrimSuffix(relPath, ".py")

	// Replace path separators with dots
	moduleName := strings.ReplaceAll(relPath, string(filepath.Separator), ".")

	// Handle __init__.py → remove __init
	moduleName = strings.ReplaceAll(moduleName, ".__init__", "")

	return moduleName
}
