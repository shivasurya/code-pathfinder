package resolution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
)

// BuildGoModuleRegistry builds a registry mapping directories to Go import paths.
// It parses go.mod to extract the module path, then walks the directory tree to build
// bidirectional mappings between directories and import paths.
//
// Parameters:
//   - projectRoot: absolute path to the project root (contains go.mod)
//
// Returns:
//   - populated GoModuleRegistry or error if go.mod is missing/invalid
func BuildGoModuleRegistry(projectRoot string) (*core.GoModuleRegistry, error) {
	registry := core.NewGoModuleRegistry()

	// Step 1: Parse go.mod to get module path and Go version
	modulePath, goVersion, err := parseGoMod(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	registry.ModulePath = modulePath
	registry.GoVersion = goVersion

	// Step 2: Get absolute path
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}

	// Step 3: Walk directory tree to build import path mappings
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process directories
		if !info.IsDir() {
			return nil
		}

		// Skip excluded directories
		if shouldSkipGoDirectory(info.Name()) {
			return filepath.SkipDir
		}

		// Calculate relative path from root
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		// Build import path
		var importPath string
		if relPath == "." {
			// Root directory maps to module path
			importPath = modulePath
		} else {
			// Subpackage: module path + relative path
			// Convert Windows backslashes to forward slashes
			normalizedRel := filepath.ToSlash(relPath)
			importPath = modulePath + "/" + normalizedRel
		}

		// Add bidirectional mapping
		registry.DirToImport[path] = importPath
		registry.ImportToDir[importPath] = path

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	return registry, nil
}

// ExtractGoImports extracts import statements from a Go source file.
// It parses the file's AST to find all import declarations and builds a mapping
// from local names (or aliases) to full import paths.
//
// Parameters:
//   - filePath: absolute path to the Go source file
//   - sourceCode: the file's source code as bytes
//   - registry: the Go module registry (currently unused but kept for consistency)
//
// Returns:
//   - GoImportMap containing all imports, or error if parsing fails
func ExtractGoImports(filePath string, sourceCode []byte, registry *core.GoModuleRegistry) (*core.GoImportMap, error) {
	importMap := core.NewGoImportMap(filePath)

	// Parse with tree-sitter
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}
	defer tree.Close()

	rootNode := tree.RootNode()

	// Step 1: Extract package name
	for i := 0; i < int(rootNode.NamedChildCount()); i++ {
		child := rootNode.NamedChild(i)
		if child.Type() == "package_clause" {
			// Find package_identifier child
			for j := 0; j < int(child.NamedChildCount()); j++ {
				pkgNode := child.NamedChild(j)
				if pkgNode.Type() == "package_identifier" {
					importMap.PackageName = pkgNode.Content(sourceCode)
					break
				}
			}
			break
		}
	}

	// Step 2: Traverse AST to find imports
	traverseForGoImports(rootNode, sourceCode, importMap)

	return importMap, nil
}

// traverseForGoImports recursively traverses the AST to find import declarations.
func traverseForGoImports(node *sitter.Node, sourceCode []byte, importMap *core.GoImportMap) {
	if node == nil {
		return
	}

	nodeType := node.Type()

	// Process import declarations
	if nodeType == "import_declaration" {
		processGoImportDeclaration(node, sourceCode, importMap)
		return // Don't recurse into children
	}

	// Recursively traverse children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		traverseForGoImports(child, sourceCode, importMap)
	}
}

// processGoImportDeclaration processes an import_declaration node.
// An import_declaration contains either:
//   - Direct import_spec nodes (simple imports)
//   - An import_spec_list containing import_spec nodes (grouped imports)
func processGoImportDeclaration(node *sitter.Node, sourceCode []byte, importMap *core.GoImportMap) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)

		switch child.Type() {
		case "import_spec":
			// Direct import: import "fmt"
			processImportSpec(child, sourceCode, importMap)
		case "import_spec_list":
			// Grouped imports: import ( ... )
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() == "import_spec" {
					processImportSpec(spec, sourceCode, importMap)
				}
			}
		}
	}
}

// processImportSpec processes a single import_spec node.
// Handles: simple imports, aliased imports, dot imports, and side-effect imports.
//
// Examples:
//   - import "fmt"                    → {"fmt": "fmt"}
//   - import h "net/http"             → {"h": "net/http"}
//   - import . "fmt"                  → {".": "fmt"}
//   - import _ "github.com/lib/pq"    → {"_": "github.com/lib/pq"}
func processImportSpec(node *sitter.Node, sourceCode []byte, importMap *core.GoImportMap) {
	var localName string
	var importPath string

	// Find import path (always an interpreted_string_literal)
	pathNode := node.ChildByFieldName("path")
	if pathNode != nil {
		// Remove surrounding quotes
		rawPath := pathNode.Content(sourceCode)
		importPath = strings.Trim(rawPath, `"`)
	}

	// Check for alias (name field)
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		// Explicit alias: h, ., or _
		localName = nameNode.Content(sourceCode)
	} else {
		// No alias - use default (last segment of import path)
		localName = extractLocalName(importPath)
	}

	// Add to imports map
	if importPath != "" {
		importMap.AddImport(localName, importPath)
	}
}

// parseGoMod extracts the module path from go.mod file.
//
// Parameters:
//   - projectRoot: absolute path to the project root
//
// Returns:
//   - module path (e.g., "github.com/example/myapp") or error if not found
func parseGoMod(projectRoot string) (modulePath string, goVersion string, err error) {
	goModPath := filepath.Join(projectRoot, "go.mod")

	content, readErr := os.ReadFile(goModPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", "", fmt.Errorf("go.mod not found in %s", projectRoot)
		}
		return "", "", readErr
	}

	// Parse go.mod line by line
	lines := strings.SplitSeq(string(content), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)

		// Extract module path
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				modulePath = parts[1]
			}
		}

		// Extract Go version
		if strings.HasPrefix(line, "go ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				goVersion = parts[1]
			}
		}
	}

	if modulePath == "" {
		return "", "", fmt.Errorf("module declaration not found in go.mod")
	}

	// goVersion is optional, so we don't error if it's missing
	return modulePath, goVersion, nil
}

// ============================================================================
// GoImportResolver — dynamic stdlib import classification
// ============================================================================

// ImportType classifies a Go import path.
type ImportType int

const (
	ImportUnknown    ImportType = iota
	ImportStdlib                // Go standard library (e.g., "fmt", "net/http")
	ImportThirdParty            // External module (e.g., "github.com/gorilla/mux")
	ImportLocal                 // Same module (e.g., "github.com/myapp/handlers" or "./utils")
)

// GoImportResolver classifies Go import paths as stdlib, third-party, or local.
// It uses the registry's StdlibLoader for dynamic, version-aware stdlib detection,
// falling back to a heuristic (no domain in path) when the loader is unavailable.
//
// Example:
//
//	resolver := NewGoImportResolver(registry)
//	if resolver.isStdlibImport("net/http") { ... }
//	kind := resolver.ClassifyImport("github.com/gorilla/mux")
type GoImportResolver struct {
	registry *core.GoModuleRegistry
}

// NewGoImportResolver creates a GoImportResolver backed by the given module registry.
// registry may be nil; in that case all classification falls back to the heuristic.
func NewGoImportResolver(registry *core.GoModuleRegistry) *GoImportResolver {
	return &GoImportResolver{registry: registry}
}

// isStdlibImport reports whether importPath belongs to the Go standard library.
// It uses StdlibLoader.ValidateStdlibImport when available, otherwise
// delegates to the offline heuristic.
func (r *GoImportResolver) isStdlibImport(importPath string) bool {
	if r.registry != nil && r.registry.StdlibLoader != nil {
		return r.registry.StdlibLoader.ValidateStdlibImport(importPath)
	}
	return r.isStdlibImportFallback(importPath)
}

// isStdlibImportFallback is the offline heuristic used when no StdlibLoader is
// available. Stdlib packages never contain a "." (domain separator) in their
// import path and are not prefixed with "internal/".
//
// Examples:
//
//	"fmt"          → true   (stdlib)
//	"net/http"     → true   (stdlib)
//	"github.com/x" → false  (has dot → third-party)
//	"internal/foo" → false  (internal package)
func (r *GoImportResolver) isStdlibImportFallback(importPath string) bool {
	if strings.HasPrefix(importPath, "internal/") {
		return false
	}
	return !strings.Contains(importPath, ".")
}

// ClassifyImport categorises a single import path.
func (r *GoImportResolver) ClassifyImport(importPath string) ImportType {
	if r.isStdlibImport(importPath) {
		return ImportStdlib
	}
	// Relative imports are always local.
	if strings.HasPrefix(importPath, ".") {
		return ImportLocal
	}
	// Imports that share the current module's path are local.
	if r.registry != nil && r.registry.ModulePath != "" &&
		strings.HasPrefix(importPath, r.registry.ModulePath) {
		return ImportLocal
	}
	return ImportThirdParty
}

// ResolveImports classifies each import path in the given slice.
func (r *GoImportResolver) ResolveImports(imports []string) map[string]ImportType {
	result := make(map[string]ImportType, len(imports))
	for _, importPath := range imports {
		result[importPath] = r.ClassifyImport(importPath)
	}
	return result
}

// shouldSkipGoDirectory returns true if the directory should be skipped during traversal.
func shouldSkipGoDirectory(dirName string) bool {
	skipDirs := map[string]bool{
		"vendor":       true, // Vendored dependencies
		"testdata":     true, // Test fixtures
		".git":         true, // Version control
		".svn":         true,
		".hg":          true,
		"node_modules": true, // If Go project has frontend
		"dist":         true, // Build output
		"build":        true,
		"_build":       true,
		".vscode":      true, // IDE files
		".idea":        true,
		"tmp":          true, // Temporary files
		"temp":         true,
		"__pycache__":  true, // If mixed project
		".DS_Store":    true, // macOS metadata
	}
	return skipDirs[dirName]
}

// extractLocalName extracts the default local name from an import path.
// Returns the last segment of the path.
//
// Examples:
//   - "fmt" → "fmt"
//   - "net/http" → "http"
//   - "github.com/myapp/handlers" → "handlers"
func extractLocalName(importPath string) string {
	// Handle empty path
	if importPath == "" {
		return ""
	}

	// Find last slash
	lastSlash := strings.LastIndex(importPath, "/")
	if lastSlash == -1 {
		// No slash - return whole path
		return importPath
	}

	// Return segment after last slash
	return importPath[lastSlash+1:]
}
