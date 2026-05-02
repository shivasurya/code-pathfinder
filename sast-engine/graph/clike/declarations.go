package clike

import sitter "github.com/smacker/go-tree-sitter"

// FunctionInfo holds extracted information from a C or C++ function_definition
// (or function-shaped declaration) node. The structure is identical for both
// languages — the dispatcher in graph/parser_c.go / graph/parser_cpp.go is
// responsible for setting Node.Language and any C++-specific fields (class
// context, namespace) on the resulting graph.Node.
//
// IsDeclaration is true when the node carries no compound_statement body —
// i.e. a forward declaration in a header (`int compute(int);`) rather than
// a definition. Forward-only declarations are still recorded so that callers
// can be linked to them; PR-03's call-graph builder later uses IsDeclaration
// to decide whether the function is callable in this translation unit or
// whether resolution must reach across files.
type FunctionInfo struct {
	Name          string
	ReturnType    string
	ParamNames    []string
	ParamTypes    []string
	IsDeclaration bool
	LineNumber    uint32
}

// FieldInfo holds a single field name + type extracted from a struct, union,
// or class body. Anonymous fields (rare but legal in C11 and C++) carry an
// empty Name; callers may either skip them or synthesize a name from TypeStr.
type FieldInfo struct {
	Name    string
	TypeStr string
}

// ExtractFunctionInfo extracts the function name, return type, parameter
// names, and parameter types from a function_definition node. The same
// implementation works for both C and C++ because tree-sitter exposes the
// same field names ("type", "declarator", "parameters", "body") in both
// grammars.
//
// The C/C++ AST shape is:
//
//	function_definition
//	├── type        ← return type (primitive_type, type_identifier, …)
//	├── declarator  ← function_declarator wrapping the name and parameters
//	│   ├── declarator (identifier or pointer_declarator)  ← name
//	│   └── parameters  ← parameter_list
//	└── body        ← compound_statement (omitted for forward declarations)
//
// Returns nil if node is nil or not a function_definition. Empty parameter
// lists yield empty (non-nil) slices.
func ExtractFunctionInfo(node *sitter.Node, sourceCode []byte) *FunctionInfo {
	if node == nil {
		return nil
	}

	info := &FunctionInfo{
		ParamNames:    []string{},
		ParamTypes:    []string{},
		LineNumber:    node.StartPoint().Row + 1,
		IsDeclaration: node.ChildByFieldName("body") == nil,
	}

	typeNode := node.ChildByFieldName("type")
	declarator := node.ChildByFieldName("declarator")

	// The function name lives at the bottom of the declarator chain, after
	// any pointer_declarator wrappers used for return-type pointers
	// (e.g. char* foo()). The function_declarator itself is reached by
	// walking through pointer_declarator nodes.
	funcDecl := unwrapToFunctionDeclarator(declarator)
	info.ReturnType = ExtractTypeString(typeNode, returnTypeDeclarator(declarator), sourceCode)

	if funcDecl == nil {
		// Best-effort fallback: the node isn't well-formed. Return what we
		// have so the caller can still record a partial function entry.
		return info
	}

	if nameNode := funcDecl.ChildByFieldName("declarator"); nameNode != nil {
		info.Name = nameNode.Content(sourceCode)
	}

	if paramList := funcDecl.ChildByFieldName("parameters"); paramList != nil {
		names, types := ExtractParameters(paramList, sourceCode)
		info.ParamNames = names
		info.ParamTypes = types
	}

	return info
}

// ExtractStructFields walks a field_declaration_list node and returns a
// FieldInfo for every field_declaration child. Bitfields keep the bare type
// (the bit count is dropped) because the type registry does not yet track
// bitfield widths and storing them in the type string would defeat downstream
// type comparison.
//
// Returns nil if list is nil. An empty struct returns an empty (non-nil)
// slice so callers can range without nil-checking the result.
func ExtractStructFields(list *sitter.Node, sourceCode []byte) []FieldInfo {
	if list == nil {
		return nil
	}

	fields := []FieldInfo{}
	for i := 0; i < int(list.NamedChildCount()); i++ {
		child := list.NamedChild(i)
		if child == nil || child.Type() != "field_declaration" {
			continue
		}

		typeNode := child.ChildByFieldName("type")
		declarator := child.ChildByFieldName("declarator")
		typeStr := ExtractTypeString(typeNode, declarator, sourceCode)

		name := fieldDeclaratorName(declarator, sourceCode)
		fields = append(fields, FieldInfo{Name: name, TypeStr: typeStr})
	}
	return fields
}

// unwrapToFunctionDeclarator walks past any pointer_declarator wrappers and
// returns the function_declarator at the centre. Returns nil if no
// function_declarator is reachable.
func unwrapToFunctionDeclarator(node *sitter.Node) *sitter.Node {
	for cur := node; cur != nil; cur = cur.ChildByFieldName("declarator") {
		if cur.Type() == "function_declarator" {
			return cur
		}
	}
	return nil
}

// returnTypeDeclarator returns the chain of pointer_declarator nodes that sit
// between the function_definition and its function_declarator. These
// declarators contribute * suffixes to the return type, not to the parameter
// list. For "char* foo()" the chain is one pointer_declarator deep; for plain
// "int foo()" it is empty (returns nil).
func returnTypeDeclarator(node *sitter.Node) *sitter.Node {
	if node == nil || node.Type() == "function_declarator" {
		return nil
	}
	// Walk the pointer chain into a synthetic declarator that pointerRefSuffix
	// can consume. The shape of the AST already matches what pointerRefSuffix
	// expects, so we can pass node directly: pointerRefSuffix stops as soon
	// as it sees the function_declarator.
	return node
}

// fieldDeclaratorName extracts the bare identifier name from a field
// declarator, stripping pointer / array / reference wrappers. Returns ""
// for anonymous fields (legal in C11 and common with bitfields like
// `int : 3;`) so callers can decide whether to keep or skip them.
func fieldDeclaratorName(declarator *sitter.Node, sourceCode []byte) string {
	for cur := declarator; cur != nil; {
		switch cur.Type() {
		case "field_identifier", "identifier":
			return cur.Content(sourceCode)
		case "pointer_declarator", "array_declarator", "reference_declarator":
			cur = innerDeclarator(cur)
			continue
		}
		return ""
	}
	return ""
}
