package resolution

import (
	"context"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ExtractImports extracts all import statements from a Python file and builds an ImportMap.
// It handles four main import styles:
//  1. Simple imports: import module
//  2. From imports: from module import name
//  3. Aliased imports: from module import name as alias
//  4. Relative imports: from . import module, from .. import module
//
// The resulting ImportMap maps local names (aliases or imported names) to their
// fully qualified module paths, enabling later resolution of function calls.
//
// Algorithm:
//  1. Parse source code with tree-sitter Python parser
//  2. Traverse AST to find all import statements
//  3. Process each import to extract module paths and aliases
//  4. Resolve relative imports using module registry
//  5. Build ImportMap with resolved fully qualified names
//
// Parameters:
//   - filePath: absolute path to the Python file being analyzed
//   - sourceCode: contents of the Python file as byte array
//   - registry: module registry for resolving module paths and relative imports
//
// Returns:
//   - ImportMap: map of local names to fully qualified module paths
//   - error: if parsing fails or source is invalid
//
// Example:
//
//	Source code:
//	  import os
//	  from myapp.utils import sanitize
//	  from myapp.db import query as db_query
//	  from . import helper
//	  from ..config import settings
//
//	Result ImportMap:
//	  {
//	    "os": "os",
//	    "sanitize": "myapp.utils.sanitize",
//	    "db_query": "myapp.db.query",
//	    "helper": "myapp.submodule.helper",
//	    "settings": "myapp.config.settings"
//	  }
func ExtractImports(filePath string, sourceCode []byte, registry *core.ModuleRegistry) (*core.ImportMap, error) {
	importMap := core.NewImportMap(filePath)

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	// Traverse AST to find import statements
	traverseForImports(tree.RootNode(), sourceCode, importMap, filePath, registry)

	return importMap, nil
}

// traverseForImports recursively traverses the AST to find import statements.
// Uses direct AST traversal instead of queries for better compatibility.
func traverseForImports(node *sitter.Node, sourceCode []byte, importMap *core.ImportMap, filePath string, registry *core.ModuleRegistry) {
	if node == nil {
		return
	}

	nodeType := node.Type()

	// Process import statements
	switch nodeType {
	case "import_statement":
		processImportStatement(node, sourceCode, importMap)
		// Don't recurse into children - we've already processed this import
		return
	case "import_from_statement":
		processImportFromStatement(node, sourceCode, importMap, filePath, registry)
		// Don't recurse into children - we've already processed this import
		return
	}

	// Recursively process children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		traverseForImports(child, sourceCode, importMap, filePath, registry)
	}
}

// processImportStatement handles simple import statements: import module [as alias].
// Examples:
//   - import os → "os" = "os"
//   - import os as op → "op" = "os"
func processImportStatement(node *sitter.Node, sourceCode []byte, importMap *core.ImportMap) {
	// Look for 'name' field which contains the import
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}

	// Check if it's an aliased import
	if nameNode.Type() == "aliased_import" {
		// import module as alias
		moduleNode := nameNode.ChildByFieldName("name")
		aliasNode := nameNode.ChildByFieldName("alias")

		if moduleNode != nil && aliasNode != nil {
			moduleName := moduleNode.Content(sourceCode)
			aliasName := aliasNode.Content(sourceCode)
			importMap.AddImport(aliasName, moduleName)
		}
	} else if nameNode.Type() == "dotted_name" {
		// Simple import: import module
		moduleName := nameNode.Content(sourceCode)
		importMap.AddImport(moduleName, moduleName)
	}
}

// processImportFromStatement handles from-import statements: from module import name [as alias].
// Examples:
//   - from os import path → "path" = "os.path"
//   - from os import path as ospath → "ospath" = "os.path"
//   - from json import dumps, loads → "dumps" = "json.dumps", "loads" = "json.loads"
//   - from . import module → "module" = "currentpackage.module"
//   - from .. import module → "module" = "parentpackage.module"
func processImportFromStatement(node *sitter.Node, sourceCode []byte, importMap *core.ImportMap, filePath string, registry *core.ModuleRegistry) {
	var moduleName string

	// Check for relative imports first
	// Tree-sitter creates a 'relative_import' node for imports starting with dots
	// This node contains import_prefix (the dots) and optionally a dotted_name
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "relative_import" {
			// Found relative import - extract dot count and module suffix
			dotCount := 0
			var moduleSuffix string

			// Look for import_prefix child (contains the dots)
			for j := 0; j < int(child.NamedChildCount()); j++ {
				subchild := child.NamedChild(j)
				if subchild.Type() == "import_prefix" {
					// Count dots in prefix
					dotCount = strings.Count(subchild.Content(sourceCode), ".")
				} else if subchild.Type() == "dotted_name" {
					// This is the module path after dots (e.g., "utils" in "..utils")
					moduleSuffix = subchild.Content(sourceCode)
				}
			}

			// Ensure we found dots - if not, this isn't a valid relative import
			if dotCount > 0 {
				// Resolve relative import to absolute module path
				moduleName = resolveRelativeImport(filePath, dotCount, moduleSuffix, registry)
			}
			break
		}
	}

	// If not a relative import, check for absolute import (module_name field)
	if moduleName == "" {
		moduleNameNode := node.ChildByFieldName("module_name")
		if moduleNameNode != nil {
			moduleName = moduleNameNode.Content(sourceCode)
			// Normalize project-internal imports to include project root
			moduleName = normalizeProjectImport(moduleName, filePath, registry)
		} else {
			return
		}
	}

	// The 'name' field might be:
	// 1. A single dotted_name: from os import path
	// 2. A single aliased_import: from os import path as ospath
	// 3. A wildcard_import: from os import *
	//
	// For multiple imports (from json import dumps, loads), tree-sitter
	// creates multiple child nodes, so we need to check all children
	moduleNameNode := node.ChildByFieldName("module_name")
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		// Skip nodes we don't want to process as imported names
		childType := child.Type()
		if childType == "from" || childType == "import" || childType == "(" || childType == ")" ||
			childType == "," || childType == "relative_import" || child == moduleNameNode {
			continue
		}

		// Process each import name/alias
		if childType == "aliased_import" {
			// from module import name as alias
			importNameNode := child.ChildByFieldName("name")
			aliasNode := child.ChildByFieldName("alias")

			if importNameNode != nil && aliasNode != nil {
				importName := importNameNode.Content(sourceCode)
				aliasName := aliasNode.Content(sourceCode)
				fqn := moduleName + "." + importName
				importMap.AddImport(aliasName, fqn)
			}
		} else if childType == "dotted_name" || childType == "identifier" {
			// from module import name
			importName := child.Content(sourceCode)
			fqn := moduleName + "." + importName
			importMap.AddImport(importName, fqn)
		}
	}
}

// resolveRelativeImport resolves a relative import to an absolute module path.
//
// Python relative imports use dot notation to navigate the package hierarchy:
//   - Single dot (.)  refers to the current package
//   - Two dots (..)   refers to the parent package
//   - Three dots (...) refers to the grandparent package
//
// Algorithm:
//  1. Get the current file's module path from the registry
//  2. Navigate up the package hierarchy based on dot count
//  3. Append the module suffix if present
//  4. Return the resolved absolute module path
//
// Parameters:
//   - filePath: absolute path to the file containing the relative import
//   - dotCount: number of leading dots in the import (1 for ".", 2 for "..", etc.)
//   - moduleSuffix: the module path after the dots (e.g., "utils" in "from ..utils import foo")
//   - registry: module registry for resolving file paths to module paths
//
// Returns:
//   - Resolved absolute module path
//
// Examples:
//
//	File: /project/myapp/submodule/helper.py (module: myapp.submodule.helper)
//	- resolveRelativeImport(..., 1, "utils", registry)   → "myapp.submodule.utils"
//	- resolveRelativeImport(..., 2, "config", registry)  → "myapp.config"
//	- resolveRelativeImport(..., 1, "", registry)        → "myapp.submodule"
//	- resolveRelativeImport(..., 3, "db", registry)      → "myapp.db" (if grandparent is myapp)
func resolveRelativeImport(filePath string, dotCount int, moduleSuffix string, registry *core.ModuleRegistry) string {
	// Get the current file's module path from the reverse map
	currentModule, found := registry.FileToModule[filePath]
	if !found {
		// Fallback: if not in registry, return the suffix or empty
		return moduleSuffix
	}

	// Split the module path into components
	// For "myapp.submodule.helper", we get ["myapp", "submodule", "helper"]
	parts := strings.Split(currentModule, ".")

	// For a file, the last component is the module name itself, not a package
	// So we need to remove it before navigating up
	if len(parts) > 0 {
		parts = parts[:len(parts)-1] // Remove the file's module name
	}

	// Navigate up the hierarchy based on dot count
	// Single dot (.) = current package (no change)
	// Two dots (..) = parent package (go up 1 level)
	// Three dots (...) = grandparent package (go up 2 levels)
	levelsUp := min(dotCount-1,
		// Can't go up more levels than available - clamp to root
		len(parts))

	if levelsUp > 0 {
		parts = parts[:len(parts)-levelsUp]
	}

	// Build the base module path
	var baseModule string
	if len(parts) > 0 {
		baseModule = strings.Join(parts, ".")
	}

	// Append the module suffix if present
	if moduleSuffix != "" {
		if baseModule != "" {
			return baseModule + "." + moduleSuffix
		}
		return moduleSuffix
	}

	return baseModule
}

// normalizeProjectImport normalizes project-internal imports to include the project root.
//
// Python imports can be:
//  1. Third-party imports (django.db.models) - already absolute, no normalization needed
//  2. Project-internal imports (data_manager.utils) - relative to project root, needs normalization
//
// This function distinguishes between these cases by:
//  1. Checking if moduleName already exists in the registry (already absolute)
//  2. If not, extracting the project root from the current file's module path
//  3. Checking if projectRoot + "." + moduleName exists in the registry
//  4. If found, returning the normalized path; otherwise, returning original (third-party)
//
// Parameters:
//   - moduleName: the module name extracted from the import statement (e.g., "data_manager.prepare_params")
//   - filePath: absolute path to the file containing the import
//   - registry: module registry for resolving module paths
//
// Returns:
//   - Normalized module path with project root prepended if it's project-internal
//
// Examples:
//
//	File: label_studio/data_manager/functions.py (module: label_studio.data_manager.functions)
//	- normalizeProjectImport("data_manager.prepare_params", ..., registry)
//	  → "label_studio.data_manager.prepare_params" (project-internal)
//	- normalizeProjectImport("django.db.models", ..., registry)
//	  → "django.db.models" (third-party, unchanged)
//	- normalizeProjectImport("rest_framework.views", ..., registry)
//	  → "rest_framework.views" (third-party, unchanged)
func normalizeProjectImport(moduleName string, filePath string, registry *core.ModuleRegistry) string {
	// If moduleName is empty, return as-is
	if moduleName == "" {
		return moduleName
	}

	// Check if this module already exists in the registry (already absolute)
	if _, exists := registry.Modules[moduleName]; exists {
		return moduleName
	}

	// Get the current file's module path from the registry
	currentModule, found := registry.FileToModule[filePath]
	if !found {
		// If file not in registry, can't normalize - return as-is
		return moduleName
	}

	// Extract the project root (first component of current module path)
	// For "label_studio.data_manager.functions", project root is "label_studio"
	// Note: strings.Split always returns at least one element, even for empty strings
	parts := strings.Split(currentModule, ".")
	projectRoot := parts[0]

	// Try prepending project root to see if it's a project-internal import
	normalizedPath := projectRoot + "." + moduleName
	if _, exists := registry.Modules[normalizedPath]; exists {
		return normalizedPath
	}

	// Not found in registry even with project root - must be third-party
	// Return original module name
	return moduleName
}
