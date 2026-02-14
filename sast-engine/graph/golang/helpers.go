package golang

import sitter "github.com/smacker/go-tree-sitter"

// GoParams holds extracted Go parameter information.
// Used by every declaration parser in PR-03 through PR-06.
type GoParams struct {
	Names []string // Parameter names (e.g., ["w", "r"])
	Types []string // Parameter types as "name: type" pairs (e.g., ["w: http.ResponseWriter", "r: *http.Request"])
}

// ExtractParameters extracts parameter names and types from a Go parameter_list node.
//
// Handles Go's grouped parameter syntax where multiple names share a type:
//
//	func Foo(a, b int, c string) → Names=["a","b","c"], Types=["a: int","b: int","c: string"]
//
// Also handles variadic parameters:
//
//	func Foo(args ...string) → Names=["args"], Types=["args: ...string"]
//
// Returns empty GoParams if paramList is nil.
func ExtractParameters(paramList *sitter.Node, sourceCode []byte) GoParams {
	result := GoParams{}
	if paramList == nil {
		return result
	}

	for i := 0; i < int(paramList.NamedChildCount()); i++ {
		param := paramList.NamedChild(i)
		if param.Type() != "parameter_declaration" && param.Type() != "variadic_parameter_declaration" {
			continue
		}

		// Extract the type node (last named child that is a type)
		typeNode := param.ChildByFieldName("type")
		paramType := ""
		if typeNode != nil {
			paramType = typeNode.Content(sourceCode)
		}

		// For variadic params, prefix with "..."
		isVariadic := param.Type() == "variadic_parameter_declaration"
		if isVariadic && paramType != "" {
			paramType = "..." + paramType
		}

		// Extract all identifier children (names before the type)
		var names []string
		for j := 0; j < int(param.NamedChildCount()); j++ {
			child := param.NamedChild(j)
			if child.Type() == "identifier" {
				names = append(names, child.Content(sourceCode))
			}
		}

		// If no explicit names (e.g., func(int, string)), use empty name
		if len(names) == 0 && paramType != "" {
			result.Names = append(result.Names, "")
			result.Types = append(result.Types, paramType)
			continue
		}

		// All names share the same type
		for _, name := range names {
			result.Names = append(result.Names, name)
			if paramType != "" {
				result.Types = append(result.Types, name+": "+paramType)
			}
		}
	}

	return result
}

// ExtractReturnType extracts the return type string from a Go function result node.
//
// Handles:
//   - Single type: "int" → "int"
//   - Multiple returns: "(string, error)" → "(string, error)"
//   - Named returns: "(n int, err error)" → "(n int, err error)"
//   - No return: nil → ""
func ExtractReturnType(resultNode *sitter.Node, sourceCode []byte) string {
	if resultNode == nil {
		return ""
	}
	return resultNode.Content(sourceCode)
}

// ExtractReceiverType extracts the receiver base type from a Go method declaration.
// Strips pointer indirection to return the underlying type name.
//
// Examples:
//
//	(s *Server) → "Server"
//	(s Server)  → "Server"
//	nil         → ""
func ExtractReceiverType(receiverNode *sitter.Node, sourceCode []byte) string {
	if receiverNode == nil {
		return ""
	}

	for i := 0; i < int(receiverNode.NamedChildCount()); i++ {
		param := receiverNode.NamedChild(i)
		if param.Type() != "parameter_declaration" {
			continue
		}

		typeNode := param.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}

		// Handle pointer receiver: *Server
		if typeNode.Type() == "pointer_type" {
			for j := 0; j < int(typeNode.NamedChildCount()); j++ {
				child := typeNode.NamedChild(j)
				if child.Type() == "type_identifier" {
					return child.Content(sourceCode)
				}
			}
		}

		// Value receiver: Server
		if typeNode.Type() == "type_identifier" {
			return typeNode.Content(sourceCode)
		}
	}

	return ""
}

// ExtractStructFields extracts field names and types from a Go struct_type node.
// Returns fields as "Name: Type" strings. Embedded types appear as just "Type".
func ExtractStructFields(structNode *sitter.Node, sourceCode []byte) []string {
	var fields []string
	if structNode == nil {
		return fields
	}

	// Find the field_declaration_list
	var fieldList *sitter.Node
	for i := 0; i < int(structNode.NamedChildCount()); i++ {
		child := structNode.NamedChild(i)
		if child.Type() == "field_declaration_list" {
			fieldList = child
			break
		}
	}
	if fieldList == nil {
		return fields
	}

	for i := 0; i < int(fieldList.NamedChildCount()); i++ {
		field := fieldList.NamedChild(i)
		if field.Type() != "field_declaration" {
			continue
		}

		nameNode := field.ChildByFieldName("name")
		typeNode := field.ChildByFieldName("type")

		if nameNode != nil && typeNode != nil {
			// Named field: Name string
			fields = append(fields, nameNode.Content(sourceCode)+": "+typeNode.Content(sourceCode))
		} else if typeNode != nil {
			// Embedded type: just the type name
			fields = append(fields, typeNode.Content(sourceCode))
		}
	}

	return fields
}

// ExtractInterfaceMethods extracts method signatures from a Go interface_type node.
// Also includes embedded interface types.
func ExtractInterfaceMethods(interfaceNode *sitter.Node, sourceCode []byte) []string {
	var methods []string
	if interfaceNode == nil {
		return methods
	}

	for i := 0; i < int(interfaceNode.NamedChildCount()); i++ {
		child := interfaceNode.NamedChild(i)
		switch child.Type() {
		case "method_spec", "method_elem":
			methods = append(methods, child.Content(sourceCode))
		case "type_elem", "type_identifier", "qualified_type":
			// Embedded interface (e.g., io.Reader) or type constraint
			methods = append(methods, child.Content(sourceCode))
		}
	}

	return methods
}

// DetermineVisibility returns "public" or "private" based on Go's
// capitalization convention: exported names start with an uppercase letter.
func DetermineVisibility(name string) string {
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		return "public"
	}
	return "private"
}

// IsInitFunction returns true if the function name is "init",
// which has special semantics in Go (auto-called at package initialization).
func IsInitFunction(name string) bool {
	return name == "init"
}

// IsGoKeyword checks if a name is a Go keyword, predeclared identifier,
// predeclared type, or builtin function.
func IsGoKeyword(name string) bool {
	keywords := map[string]bool{
		// Go keywords (25)
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
		// Predeclared identifiers
		"true": true, "false": true, "nil": true, "iota": true,
		// Predeclared types
		"bool": true, "byte": true, "complex64": true, "complex128": true,
		"error": true, "float32": true, "float64": true, "int": true,
		"int8": true, "int16": true, "int32": true, "int64": true,
		"rune": true, "string": true, "uint": true, "uint8": true,
		"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
		"any": true, "comparable": true,
		// Builtin functions (18)
		"append": true, "cap": true, "clear": true, "close": true,
		"complex": true, "copy": true, "delete": true, "imag": true,
		"len": true, "make": true, "max": true, "min": true,
		"new": true, "panic": true, "print": true, "println": true,
		"real": true, "recover": true,
	}
	return keywords[name]
}
