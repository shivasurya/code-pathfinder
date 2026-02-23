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

	// Step 4: Initialize stdlib packages
	registry.StdlibPackages = goStdlibSet()

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

// goStdlibSet returns a set of Go standard library packages for Go 1.21.
// Phase 1: Hardcoded map as example.
// Phase 2 (future): Dynamic JSON-based registry downloaded from CDN.
func goStdlibSet() map[string]bool {
	return map[string]bool{
		// Core packages
		"fmt": true, "os": true, "io": true, "errors": true,
		"strings": true, "strconv": true, "bytes": true,
		"unicode": true, "unicode/utf8": true, "bufio": true,

		// Data structures
		"container/heap": true, "container/list": true, "container/ring": true,
		"sort": true, "math": true, "math/big": true, "math/rand": true,
		"math/bits": true, "math/cmplx": true,

		// Networking
		"net": true, "net/http": true, "net/url": true, "net/http/httputil": true,
		"net/http/cookiejar": true, "net/http/fcgi": true, "net/http/httptest": true,
		"net/http/httptrace": true, "net/http/pprof": true,
		"net/mail": true, "net/rpc": true, "net/rpc/jsonrpc": true,
		"net/smtp": true, "net/textproto": true, "net/netip": true,

		// Encoding
		"encoding": true, "encoding/json": true, "encoding/xml": true,
		"encoding/base64": true, "encoding/hex": true, "encoding/csv": true,
		"encoding/gob": true, "encoding/binary": true, "encoding/ascii85": true,
		"encoding/asn1": true, "encoding/pem": true,

		// Time
		"time": true, "time/tzdata": true,

		// File system
		"path": true, "path/filepath": true, "io/fs": true, "io/ioutil": true,

		// Compression
		"compress/bzip2": true, "compress/flate": true, "compress/gzip": true,
		"compress/lzw": true, "compress/zlib": true,

		// Crypto
		"crypto": true, "crypto/aes": true, "crypto/cipher": true,
		"crypto/des": true, "crypto/dsa": true, "crypto/ecdsa": true,
		"crypto/ed25519": true, "crypto/elliptic": true, "crypto/hmac": true,
		"crypto/md5": true, "crypto/rand": true, "crypto/rc4": true,
		"crypto/rsa": true, "crypto/sha1": true, "crypto/sha256": true,
		"crypto/sha512": true, "crypto/subtle": true, "crypto/tls": true,
		"crypto/x509": true, "crypto/x509/pkix": true,

		// Hash
		"hash": true, "hash/adler32": true, "hash/crc32": true,
		"hash/crc64": true, "hash/fnv": true, "hash/maphash": true,

		// Database
		"database/sql": true, "database/sql/driver": true,

		// Context and sync
		"context": true, "sync": true, "sync/atomic": true,

		// Reflection and unsafe
		"reflect": true, "unsafe": true,

		// Runtime
		"runtime": true, "runtime/debug": true, "runtime/metrics": true,
		"runtime/pprof": true, "runtime/trace": true, "runtime/cgo": true,

		// Testing
		"testing": true, "testing/fstest": true, "testing/iotest": true,
		"testing/quick": true,

		// Text processing
		"text/scanner": true, "text/tabwriter": true, "text/template": true,
		"text/template/parse": true, "regexp": true, "regexp/syntax": true,

		// Image
		"image": true, "image/color": true, "image/color/palette": true,
		"image/draw": true, "image/gif": true, "image/jpeg": true,
		"image/png": true,

		// HTML
		"html": true, "html/template": true,

		// Archive
		"archive/tar": true, "archive/zip": true,

		// Index
		"index/suffixarray": true,

		// Go-specific
		"go/ast": true, "go/build": true, "go/constant": true,
		"go/doc": true, "go/format": true, "go/importer": true,
		"go/parser": true, "go/printer": true, "go/scanner": true,
		"go/token": true, "go/types": true, "go/doc/comment": true,
		"go/build/constraint": true,

		// Miscellaneous
		"flag": true, "log": true, "log/syslog": true,
		"mime": true, "mime/multipart": true,
		"mime/quotedprintable": true, "plugin": true, "expvar": true,

		// Embed
		"embed": true,

		// Slices and maps (Go 1.21+)
		"slices": true, "maps": true,

		// Cmp (Go 1.21+)
		"cmp": true,

		// Slog (Go 1.21+)
		"log/slog": true,

		// Debug
		"debug/buildinfo": true, "debug/dwarf": true, "debug/elf": true,
		"debug/gosym": true, "debug/macho": true, "debug/pe": true,
		"debug/plan9obj": true,

		// Internal utilities (exported in some cases)
		"syscall": true, "internal/bytealg": true,
	}
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
