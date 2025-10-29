package callgraph

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ExtractImports extracts all import statements from a Python file and builds an ImportMap.
// It handles three main import styles:
//  1. Simple imports: import module
//  2. From imports: from module import name
//  3. Aliased imports: from module import name as alias
//
// The resulting ImportMap maps local names (aliases or imported names) to their
// fully qualified module paths, enabling later resolution of function calls.
//
// Algorithm:
//  1. Parse source code with tree-sitter Python parser
//  2. Execute tree-sitter query to find all import statements
//  3. Process each import match to extract module paths and aliases
//  4. Build ImportMap with resolved fully qualified names
//
// Parameters:
//   - filePath: absolute path to the Python file being analyzed
//   - sourceCode: contents of the Python file as byte array
//   - registry: module registry for resolving module paths
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
//
//	Result ImportMap:
//	  {
//	    "os": "os",
//	    "sanitize": "myapp.utils.sanitize",
//	    "db_query": "myapp.db.query"
//	  }
func ExtractImports(filePath string, sourceCode []byte, registry *ModuleRegistry) (*ImportMap, error) {
	importMap := NewImportMap(filePath)

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
	traverseForImports(tree.RootNode(), sourceCode, importMap)

	return importMap, nil
}

// traverseForImports recursively traverses the AST to find import statements.
// Uses direct AST traversal instead of queries for better compatibility.
func traverseForImports(node *sitter.Node, sourceCode []byte, importMap *ImportMap) {
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
		processImportFromStatement(node, sourceCode, importMap)
		// Don't recurse into children - we've already processed this import
		return
	}

	// Recursively process children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		traverseForImports(child, sourceCode, importMap)
	}
}

// processImportStatement handles simple import statements: import module [as alias].
// Examples:
//   - import os → "os" = "os"
//   - import os as op → "op" = "os"
func processImportStatement(node *sitter.Node, sourceCode []byte, importMap *ImportMap) {
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
func processImportFromStatement(node *sitter.Node, sourceCode []byte, importMap *ImportMap) {
	// Get the module being imported from
	moduleNameNode := node.ChildByFieldName("module_name")
	if moduleNameNode == nil {
		return
	}

	moduleName := moduleNameNode.Content(sourceCode)

	// The 'name' field might be:
	// 1. A single dotted_name: from os import path
	// 2. A single aliased_import: from os import path as ospath
	// 3. A wildcard_import: from os import *
	//
	// For multiple imports (from json import dumps, loads), tree-sitter
	// creates multiple child nodes, so we need to check all children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		// Skip the module_name node itself - we only want the imported names
		if child == moduleNameNode {
			continue
		}

		// Process each import name/alias
		if child.Type() == "aliased_import" {
			// from module import name as alias
			importNameNode := child.ChildByFieldName("name")
			aliasNode := child.ChildByFieldName("alias")

			if importNameNode != nil && aliasNode != nil {
				importName := importNameNode.Content(sourceCode)
				aliasName := aliasNode.Content(sourceCode)
				fqn := moduleName + "." + importName
				importMap.AddImport(aliasName, fqn)
			}
		} else if child.Type() == "dotted_name" || child.Type() == "identifier" {
			// from module import name
			importName := child.Content(sourceCode)
			fqn := moduleName + "." + importName
			importMap.AddImport(importName, fqn)
		}
	}
}
