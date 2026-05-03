package builder

import (
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// ReadFileBytes reads a file and returns its contents as a byte slice.
// Helper function for reading source code.
//
// Parameters:
//   - filePath: path to the file (can be relative or absolute)
//
// Returns:
//   - File contents as byte slice
//   - error if file cannot be read
func ReadFileBytes(filePath string) ([]byte, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(absPath)
}

// FindFunctionAtLine searches for a function definition at the specified line number.
// Returns the tree-sitter node for the function, or nil if not found.
//
// This function recursively traverses the AST tree to find a function or method
// definition node at the given line number.
//
// Parameters:
//   - root: the root tree-sitter node to search from
//   - lineNumber: the line number to search for (1-indexed)
//
// Returns:
//   - tree-sitter node for the function definition, or nil if not found
func FindFunctionAtLine(root *sitter.Node, lineNumber uint32) *sitter.Node {
	if root == nil {
		return nil
	}

	// Check if this node is a function definition at the target line
	if (root.Type() == "function_definition" || root.Type() == "method_declaration") &&
		root.StartPoint().Row+1 == lineNumber {
		return root
	}

	// Recursively search children
	for i := 0; i < int(root.ChildCount()); i++ {
		if result := FindFunctionAtLine(root.Child(i), lineNumber); result != nil {
			return result
		}
	}

	return nil
}

// findGoNodeByByteRange finds a Go function or method node matching the given byte range.
// Analogous to FindFunctionAtLine (which searches Python by line number).
//
// Go nodes use SourceLocation{StartByte, EndByte} set by setGoSourceLocation
// in graph/parser_golang.go. This function matches on those byte offsets.
//
// Note: Go uses "function_declaration" (not Python's "function_definition")
// and "method_declaration" for methods with receivers.
//
// Parameters:
//   - root: the root tree-sitter node to search from
//   - startByte: expected StartByte of the function node
//   - endByte: expected EndByte of the function node
//
// Returns:
//   - tree-sitter node for the function/method, or nil if not found
func findGoNodeByByteRange(root *sitter.Node, startByte, endByte uint32) *sitter.Node {
	if root == nil {
		return nil
	}

	if (root.Type() == "function_declaration" || root.Type() == "method_declaration") &&
		root.StartByte() == startByte && root.EndByte() == endByte {
		return root
	}

	for i := 0; i < int(root.ChildCount()); i++ {
		if result := findGoNodeByByteRange(root.Child(i), startByte, endByte); result != nil {
			return result
		}
	}

	return nil
}

// splitGoTypeFQN splits a Go type FQN into import path and type name.
// Uses the LAST dot as separator since import paths can contain dots.
//
// Examples:
//
//	"database/sql.DB"             → ("database/sql", "DB", true)
//	"net/http.Request"            → ("net/http", "Request", true)
//	"github.com/lib/pq.Connector" → ("github.com/lib/pq", "Connector", true)
//	"fmt.Stringer"                → ("fmt", "Stringer", true)
//	"error"                       → ("", "", false)
func splitGoTypeFQN(typeFQN string) (importPath, typeName string, ok bool) {
	if typeFQN == "" {
		return "", "", false
	}

	lastDot := strings.LastIndex(typeFQN, ".")
	if lastDot < 0 || lastDot == len(typeFQN)-1 {
		return "", "", false
	}

	return typeFQN[:lastDot], typeFQN[lastDot+1:], true
}

// resolveGoTypeFQN resolves a short Go type name to a fully qualified import path
// using the file's import map.
//
// Examples (with importMap: "http" → "net/http", "sql" → "database/sql"):
//
//	"http.Request"  → "net/http.Request"
//	"sql.DB"        → "database/sql.DB"
//	"MyStruct"      → "MyStruct"  (unqualified — returned as-is)
//	"redis.Client"  → "redis.Client"  (unknown alias — returned as-is)
func resolveGoTypeFQN(shortType string, importMap *core.GoImportMap) string {
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

// resolveFieldType converts a raw struct field type string (as stored in the
// CDN registry) to a TypeFQN suitable for method resolution.
//
// The CDN stores field types relative to the package they belong to, so a
// field typed "Header" in "net/http" means "net/http.Header". A field typed
// "io.ReadCloser" is already package-qualified and returned as-is. Pointer
// prefixes are stripped since method lookup works on the base type.
//
// Examples (pkgPath = "net/http"):
//
//	"Header"        → "net/http.Header"
//	"*url.URL"      → "net/url.URL"  (the caller must resolve "url" → "net/url")
//	"io.ReadCloser" → "io.ReadCloser"
//	"string"        → "" (builtin — not useful for method resolution)
func resolveFieldType(rawType, pkgPath string) string {
	t := strings.TrimPrefix(rawType, "*")
	t = strings.TrimPrefix(t, "[]") // drop slice prefix; focus on element type
	if t == "" {
		return ""
	}
	// Skip builtins and function types — not useful for method resolution.
	if strings.HasPrefix(t, "func") || strings.HasPrefix(t, "chan") || strings.HasPrefix(t, "map") {
		return ""
	}
	// Already package-qualified (e.g., "io.ReadCloser", "url.URL")
	if strings.Contains(t, ".") {
		return t // caller may further resolve "url" → "net/url" if needed
	}
	// Check if it's a Go builtin type — skip those.
	builtins := map[string]bool{
		"bool": true, "byte": true, "error": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true,
		"uint64": true, "string": true, "rune": true, "float32": true,
		"float64": true, "any": true,
	}
	if builtins[t] {
		return ""
	}
	// Unqualified name — belongs to the same package (e.g., "Header" in "net/http").
	return pkgPath + "." + t
}

// findMethodInPackageInterfaces scans all types in the given stdlib package for
// any interface type that declares the named method. Returns the FQN of the
// method on the first matching interface (e.g., "testing.TB.Fatalf").
//
// This handles promoted methods that do not appear directly on a concrete type
// in the CDN data (e.g., testing.T.Fatalf is promoted from testing.common but
// the CDN only lists testing.T's direct methods). Since T implements TB and TB
// declares Fatalf, this function finds testing.TB.Fatalf and returns it.
//
// Returns ("", false) when no interface in the package has the method.
func findMethodInPackageInterfaces(
	loader core.GoStdlibLoader,
	importPath, methodName string,
) (methodFQN string, found bool) {
	pkg, err := loader.GetPackage(importPath)
	if err != nil || pkg == nil {
		return "", false
	}

	for typeName, typ := range pkg.Types {
		if typ.Kind != "interface" {
			continue
		}
		if _, hasMethod := typ.Methods[methodName]; hasMethod {
			return importPath + "." + typeName + "." + methodName, true
		}
	}
	return "", false
}
