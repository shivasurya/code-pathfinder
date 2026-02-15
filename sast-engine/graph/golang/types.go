package golang

import sitter "github.com/smacker/go-tree-sitter"

// TypeInfo holds extracted information from a Go type declaration.
// Used by the dispatcher in parser_golang.go to create graph.Node instances.
type TypeInfo struct {
	Name       string   // "Server", "Handler", "UserID"
	Kind       string   // "struct", "interface", "alias"
	Visibility string   // "public" or "private"
	LineNumber uint32
	StartByte  uint32   // byte range from type_spec node
	EndByte    uint32
	Fields     []string // struct fields: ["Host: string", "Logger"]
	Methods    []string // interface methods: ["Handle() error", "io.Reader"]
}

// ParseTypeDeclaration extracts type information from a Go type_declaration node.
// Returns a slice because grouped declarations (type ( A int; B string )) produce
// multiple types from a single type_declaration node.
func ParseTypeDeclaration(node *sitter.Node, sourceCode []byte) []*TypeInfo {
	var types []*TypeInfo

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)

		switch child.Type() {
		case "type_spec":
			info := parseTypeSpec(child, sourceCode)
			if info != nil {
				types = append(types, info)
			}
		case "type_alias":
			info := parseTypeAlias(child, sourceCode)
			if info != nil {
				types = append(types, info)
			}
		}
	}

	return types
}

// parseTypeSpec extracts type info from a type_spec node (e.g., type Server struct{}).
func parseTypeSpec(spec *sitter.Node, sourceCode []byte) *TypeInfo {
	name := ""
	nameNode := spec.ChildByFieldName("name")
	if nameNode != nil {
		name = nameNode.Content(sourceCode)
	}

	info := &TypeInfo{
		Name:       name,
		Visibility: DetermineVisibility(name),
		LineNumber: spec.StartPoint().Row + 1,
		StartByte:  spec.StartByte(),
		EndByte:    spec.EndByte(),
	}

	typeNode := spec.ChildByFieldName("type")
	if typeNode == nil {
		info.Kind = "alias"
		return info
	}

	switch typeNode.Type() {
	case "struct_type":
		info.Kind = "struct"
		info.Fields = ExtractStructFields(typeNode, sourceCode)
	case "interface_type":
		info.Kind = "interface"
		info.Methods = ExtractInterfaceMethods(typeNode, sourceCode)
	default:
		info.Kind = "alias"
	}

	return info
}

// parseTypeAlias extracts type info from a type_alias node (e.g., type A = int).
func parseTypeAlias(alias *sitter.Node, sourceCode []byte) *TypeInfo {
	name := ""
	nameNode := alias.ChildByFieldName("name")
	if nameNode != nil {
		name = nameNode.Content(sourceCode)
	}

	return &TypeInfo{
		Name:       name,
		Kind:       "alias",
		Visibility: DetermineVisibility(name),
		LineNumber: alias.StartPoint().Row + 1,
		StartByte:  alias.StartByte(),
		EndByte:    alias.EndByte(),
	}
}
