package clike

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ExtractTypeString assembles a complete C/C++ type string from the AST nodes
// produced by tree-sitter for a declaration. The result includes type
// qualifiers (const / volatile / restrict), the base type (primitive_type,
// type_identifier, qualified_identifier, or template_type), and any pointer
// or reference suffixes derived from the declarator chain.
//
// The function is invoked with the "type" node from a parameter_declaration,
// declaration, or field_declaration; the caller passes in the matching
// "declarator" node to pick up the * / & / [] suffixes that tree-sitter
// places on the declarator side of the AST. Either argument may be nil:
// a nil typeNode produces "void" (the empty-return-type convention used by
// tree-sitter's C grammar), and a nil declarator simply means no suffixes.
//
// Examples (typeNode + declarator → result):
//
//	primitive_type "int"                    + identifier "x"        → "int"
//	primitive_type "char"                   + pointer_declarator    → "char*"
//	primitive_type "int"                    + pointer_declarator(2) → "int**"
//	type_identifier "FILE"                  + pointer_declarator    → "FILE*"
//	primitive_type "int" with const         + identifier "x"        → "const int"
//	qualified_identifier "std::string"      + reference_declarator  → "std::string&"
//	template_type "std::vector<int>"        + identifier "v"        → "std::vector<int>"
//	primitive_type "long" with unsigned     + identifier "n"        → "unsigned long"
//
// The helper is whitespace-conservative: qualifiers are joined with single
// spaces and pointer/reference suffixes are appended without spaces, which
// matches the canonical form used by every other type registry in the
// codebase (Python typeshed, Go types.go, Java fully-qualified names).
func ExtractTypeString(typeNode, declarator *sitter.Node, sourceCode []byte) string {
	if typeNode == nil {
		return "void"
	}

	// Qualifiers (const, volatile, restrict, unsigned, signed) live as
	// sibling type_qualifier / sized_type_specifier nodes on the parent.
	qualifiers := collectTypeQualifiers(typeNode, sourceCode)

	base := baseTypeString(typeNode, sourceCode)
	suffix := pointerRefSuffix(declarator, sourceCode)

	if len(qualifiers) == 0 {
		return base + suffix
	}
	return strings.Join(qualifiers, " ") + " " + base + suffix
}

// baseTypeString returns the human-readable base type from typeNode, without
// any qualifiers or pointer/reference suffixes.
//
// Tree-sitter emits five shapes in the spots we call this from —
// primitive_type, type_identifier, sized_type_specifier, qualified_identifier
// ("std::string"), and template_type ("vector<int>") — and all of them
// serialise verbatim from source, so a single content fetch suffices.
func baseTypeString(typeNode *sitter.Node, sourceCode []byte) string {
	return strings.TrimSpace(typeNode.Content(sourceCode))
}

// collectTypeQualifiers walks the parent of typeNode looking for qualifier
// siblings (type_qualifier nodes such as "const", "volatile", "restrict",
// and the C-specific signedness markers "unsigned" / "signed" expressed as
// sized_type_specifier siblings on certain grammars). Order is preserved
// from source.
func collectTypeQualifiers(typeNode *sitter.Node, sourceCode []byte) []string {
	parent := typeNode.Parent()
	if parent == nil {
		return nil
	}

	var quals []string
	for i := 0; i < int(parent.NamedChildCount()); i++ {
		sib := parent.NamedChild(i)
		if sib == nil || sib.Equal(typeNode) {
			continue
		}
		if sib.Type() == "type_qualifier" {
			quals = append(quals, strings.TrimSpace(sib.Content(sourceCode)))
		}
	}
	return quals
}

// pointerRefSuffix walks down a declarator chain and emits one * for each
// pointer_declarator and one & for each reference_declarator (C++ only).
// The traversal stops at the first non-pointer/reference node, which is
// usually the identifier that names the entity being declared.
//
// tree-sitter nests pointer declarators left-to-right, so "int**" appears
// as pointer_declarator(pointer_declarator(identifier)); collecting them
// inside-out produces the correct "**" suffix order.
func pointerRefSuffix(declarator *sitter.Node, _ []byte) string {
	suffix := ""
	for cur := declarator; cur != nil; {
		switch cur.Type() {
		case "pointer_declarator", "abstract_pointer_declarator":
			suffix += "*"
		case "reference_declarator", "abstract_reference_declarator":
			suffix += "&"
		default:
			return suffix
		}
		next := innerDeclarator(cur)
		if next == nil {
			return suffix
		}
		cur = next
	}
	return suffix
}

// innerDeclarator returns the next declarator inside a wrapper such as
// pointer_declarator, reference_declarator, or array_declarator. The C
// grammar exposes this as a field-named "declarator" child, but the C++
// reference_declarator (and several abstract_* variants) does not name the
// inner child — in that case we fall back to the first non-anonymous named
// child, which is the wrapped declarator or identifier in every grammar
// rule that uses these node types.
func innerDeclarator(wrapper *sitter.Node) *sitter.Node {
	if wrapper == nil {
		return nil
	}
	if c := wrapper.ChildByFieldName("declarator"); c != nil {
		return c
	}
	for i := 0; i < int(wrapper.NamedChildCount()); i++ {
		c := wrapper.NamedChild(i)
		if c == nil {
			continue
		}
		// Skip type-side qualifiers that occasionally appear inside
		// declarators (e.g., const inside `int * const p`).
		if c.Type() == "type_qualifier" {
			continue
		}
		return c
	}
	return nil
}
