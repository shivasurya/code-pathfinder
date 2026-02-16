package extraction

import (
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Go builtin types (Go 1.21+)
var goBuiltinTypes = map[string]bool{
	// Numeric types
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"complex64": true, "complex128": true,

	// String and character types
	"string": true, "byte": true, "rune": true,

	// Boolean
	"bool": true,

	// Special types
	"error": true, "any": true,
}

// IsBuiltinType checks if a type string is a Go builtin type.
//
// Examples:
//   - "int" → true
//   - "string" → true
//   - "User" → false
func IsBuiltinType(typeStr string) bool {
	return goBuiltinTypes[typeStr]
}

// StripPointerPrefix removes the leading * from pointer types.
// Multiple pointer levels are stripped to the base type.
//
// Examples:
//   - "*User" → "User"
//   - "**Config" → "Config"
//   - "User" → "User" (no change)
func StripPointerPrefix(typeStr string) string {
	// Strip all leading asterisks
	for strings.HasPrefix(typeStr, "*") {
		typeStr = strings.TrimPrefix(typeStr, "*")
	}
	return typeStr
}

// ExtractFirstReturnType extracts the first type from multi-return syntax.
//
// Examples:
//   - "(string, error)" → "string"
//   - "(int, bool)" → "int"
//   - "string" → "string" (no change)
//   - "(User, error)" → "User"
func ExtractFirstReturnType(typeStr string) string {
	if !strings.HasPrefix(typeStr, "(") {
		return typeStr
	}

	// Remove outer parentheses: "(string, error)" → "string, error"
	inside := strings.TrimPrefix(strings.TrimSuffix(typeStr, ")"), "(")

	// Split by comma
	parts := strings.Split(inside, ",")
	if len(parts) == 0 {
		return typeStr
	}

	// Return first type, trimmed
	return strings.TrimSpace(parts[0])
}

// ParseGoTypeString parses a Go type string into a TypeInfo object.
//
// Handles multiple Go type patterns:
//   - Builtins: "int", "string", "error" → "builtin.X"
//   - Pointers: "*User" → strip * → "pkg.User"
//   - Multi-return: "(string, error)" → extract first → "builtin.string"
//   - Qualified: "models.User" → resolve package → "github.com/myapp/models.User"
//   - Same-package: "User" → resolve via registry → "github.com/myapp/handlers.User"
//
// Algorithm:
//  1. Normalize and trim whitespace
//  2. Handle multi-return (extract first)
//  3. Strip pointer prefix
//  4. Check if builtin
//  5. Resolve qualified types (with .)
//  6. Resolve same-package types (via registry)
//  7. Fallback to as-is with lower confidence
//
// Parameters:
//   - typeStr: Go type string from node.ReturnType (e.g., "*User", "(string, error)")
//   - registry: Go module registry for package resolution (can be nil)
//   - filePath: Source file path for same-package resolution (can be empty)
//
// Returns:
//   - TypeInfo object with resolved FQN
//   - nil if typeStr is empty
//   - error if parsing fails critically (currently never returns error)
//
// Examples:
//   - ParseGoTypeString("int", nil, "") → TypeInfo{TypeFQN: "builtin.int", Confidence: 1.0}
//   - ParseGoTypeString("*User", registry, "models/user.go") → TypeInfo{TypeFQN: "myapp.models.User", Confidence: 0.95}
//   - ParseGoTypeString("(string, error)", nil, "") → TypeInfo{TypeFQN: "builtin.string", Confidence: 1.0}
func ParseGoTypeString(
	typeStr string,
	registry *core.GoModuleRegistry,
	filePath string,
) (*core.TypeInfo, error) {
	// Handle empty type string
	if typeStr == "" {
		return nil, nil
	}

	// Step 1: Normalize - trim whitespace
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return nil, nil
	}

	// Step 2: Handle multi-return: "(string, error)" → "string"
	// We only track the first return type for assignment inference
	if strings.HasPrefix(typeStr, "(") {
		typeStr = ExtractFirstReturnType(typeStr)
	}

	// Step 3: Strip pointer prefix: "*User" → "User"
	// Method lookup works for both value and pointer receivers
	typeStr = StripPointerPrefix(typeStr)

	// Re-trim after transformations
	typeStr = strings.TrimSpace(typeStr)

	// Step 4: Check if builtin type
	if IsBuiltinType(typeStr) {
		return &core.TypeInfo{
			TypeFQN:    "builtin." + typeStr,
			Confidence: 1.0,
			Source:     "declaration",
		}, nil
	}

	// Step 5: Resolve package-qualified types: "models.User"
	// Contains a dot, so it's a qualified type reference
	if strings.Contains(typeStr, ".") {
		// For now, keep qualified names as-is
		// Full import resolution happens in PR-16 integration
		// This handles cases like "http.Client", "models.User"
		return &core.TypeInfo{
			TypeFQN:    typeStr,
			Confidence: 0.9,
			Source:     "declaration",
		}, nil
	}

	// Step 6: Same-package types: "User" (no package qualifier)
	// Use registry to resolve to full import path
	if filePath != "" && registry != nil {
		dirPath := filepath.Dir(filePath)
		importPath, ok := registry.DirToImport[dirPath]
		if ok {
			// Successfully resolved to import path
			return &core.TypeInfo{
				TypeFQN:    importPath + "." + typeStr,
				Confidence: 0.95,
				Source:     "declaration",
			}, nil
		}
	}

	// Step 7: Fallback - use type string as-is with lower confidence
	// This handles cases where registry lookup failed
	return &core.TypeInfo{
		TypeFQN:    typeStr,
		Confidence: 0.5,
		Source:     "declaration",
	}, nil
}
