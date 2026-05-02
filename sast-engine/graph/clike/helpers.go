package clike

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CallInfo describes a single C/C++ call_expression. The dispatcher in
// graph/parser_c.go (PR-03) and graph/parser_cpp.go (PR-04) uses this
// structure to decide which call_resolution strategy to apply: free function
// calls go through the file-scope index, qualified calls go through the
// namespace index, and method calls (the obj.foo() and obj->foo() shapes)
// require the receiver type to resolve.
type CallInfo struct {
	// Target is the source-level callee text. For free functions this is
	// the bare name ("malloc"); for qualified calls it includes the
	// namespace chain ("std::move", "mylib::Socket::connect"); for method
	// and arrow calls it is the field/method name only ("free", "size").
	Target string

	// Args holds each argument expression as raw source text. Argument
	// parsing is deferred to a later PR — at this layer we only need the
	// arity and approximate text for diagnostics.
	Args []string

	// IsMethod is true for the field_expression shapes:
	//   obj.foo()    (dot operator)
	//   ptr->foo()   (arrow operator)
	IsMethod bool

	// IsArrow distinguishes -> from . on a method call. Both set IsMethod;
	// IsArrow=true additionally tells the resolver that Receiver is
	// pointer-typed, which matters for member access through smart
	// pointers and forward-declared classes.
	IsArrow bool

	// IsQualified is true for namespace-qualified calls like
	// std::move(x) or mylib::ns::func(a). When true, Target already
	// contains the full chain.
	IsQualified bool

	// Receiver is the source-level expression on the left of '.' or '->'
	// for method calls, empty otherwise. It is captured raw (not resolved)
	// because the type-inference step that turns it into a class FQN runs
	// in a later pass.
	Receiver string
}

// ExtractParameters extracts the parameter names and types from a
// parameter_list node. The two slices are returned as parallel arrays
// (names[i] corresponds to types[i]) to match the convention used by
// graph/golang/helpers.go and the Java parser.
//
// The extractor handles every shape produced by tree-sitter for C and C++:
//
//   - parameter_declaration:                int x          → "x", "int"
//   - parameter_declaration with pointer:   char* buf      → "buf", "char*"
//   - parameter_declaration with reference: const T& v     → "v", "const T&"
//   - parameter_declaration without name:   int            → "", "int"
//   - optional_parameter_declaration:       int x = 0      → "x", "int"
//   - variadic_parameter:                   ...            → "...", "..."
//
// Returns empty (non-nil) slices when paramList is nil or empty.
func ExtractParameters(paramList *sitter.Node, sourceCode []byte) (names []string, types []string) {
	names = []string{}
	types = []string{}
	if paramList == nil {
		return names, types
	}

	for i := 0; i < int(paramList.NamedChildCount()); i++ {
		param := paramList.NamedChild(i)
		if param == nil {
			continue
		}

		switch param.Type() {
		case "parameter_declaration", "optional_parameter_declaration":
			name, typ := extractSingleParameter(param, sourceCode)
			names = append(names, name)
			types = append(types, typ)
		case "variadic_parameter", "variadic_parameter_declaration", "...":
			names = append(names, "...")
			types = append(types, "...")
		}
	}
	return names, types
}

// extractSingleParameter pulls the name and type from a single parameter_declaration.
func extractSingleParameter(param *sitter.Node, sourceCode []byte) (string, string) {
	typeNode := param.ChildByFieldName("type")
	declarator := param.ChildByFieldName("declarator")
	typeStr := ExtractTypeString(typeNode, declarator, sourceCode)
	name := parameterDeclaratorName(declarator, sourceCode)
	return name, typeStr
}

// parameterDeclaratorName walks a parameter declarator chain to find the
// bare identifier name, stripping pointer / reference / array wrappers.
// Returns "" when the parameter is unnamed (legal in C and common in
// abstract declarators used for casts).
func parameterDeclaratorName(declarator *sitter.Node, sourceCode []byte) string {
	for cur := declarator; cur != nil; {
		switch cur.Type() {
		case "identifier":
			return cur.Content(sourceCode)
		case "pointer_declarator", "reference_declarator", "array_declarator":
			cur = innerDeclarator(cur)
			continue
		}
		// Abstract declarators (abstract_pointer_declarator etc.) and any
		// other shape have no identifier we can extract.
		return ""
	}
	return ""
}

// ExtractCallInfo extracts the callee, arguments, and call shape from a
// call_expression node. The function returns nil when node is nil or not a
// call_expression so callers can pass it through unchecked AST traversals.
//
// tree-sitter's C / C++ call_expression has two named fields:
//
//	function   ← identifier, field_expression, qualified_identifier, …
//	arguments  ← argument_list (named children are the args)
//
// The function field's node type is what determines IsMethod / IsArrow /
// IsQualified — see the cases inside.
func ExtractCallInfo(node *sitter.Node, sourceCode []byte) *CallInfo {
	if node == nil || node.Type() != "call_expression" {
		return nil
	}

	info := &CallInfo{Args: []string{}}

	if fn := node.ChildByFieldName("function"); fn != nil {
		populateCallTarget(info, fn, sourceCode)
	}
	if argList := node.ChildByFieldName("arguments"); argList != nil {
		for i := 0; i < int(argList.NamedChildCount()); i++ {
			if arg := argList.NamedChild(i); arg != nil {
				info.Args = append(info.Args, arg.Content(sourceCode))
			}
		}
	}
	return info
}

// populateCallTarget classifies the function expression and writes the
// derived shape flags / target / receiver back into info.
func populateCallTarget(info *CallInfo, fn *sitter.Node, sourceCode []byte) {
	switch fn.Type() {
	case "identifier":
		info.Target = fn.Content(sourceCode)

	case "field_expression":
		// obj.method() or obj->method().
		// tree-sitter exposes "argument" (the receiver) and "field" (the
		// method name); the access kind is the operator child between them.
		info.IsMethod = true
		if recv := fn.ChildByFieldName("argument"); recv != nil {
			info.Receiver = recv.Content(sourceCode)
		}
		if field := fn.ChildByFieldName("field"); field != nil {
			info.Target = field.Content(sourceCode)
		}
		info.IsArrow = strings.Contains(fn.Content(sourceCode), "->")

	case "qualified_identifier":
		info.IsQualified = true
		info.Target = strings.TrimSpace(fn.Content(sourceCode))

	default:
		// Fallback covers function pointers, lambdas, parenthesized
		// expressions, etc. We record the raw text so downstream code can
		// still match on the source form even when we cannot classify it.
		info.Target = strings.TrimSpace(fn.Content(sourceCode))
	}
}

// cKeywords is the canonical set of C reserved words plus a handful of
// common constants that statement extraction should never report as
// referenced variables. The list spans C89 through C23.
//
// Bool/null constants (NULL, EOF, true, false, nullptr) are included here
// because real C code references them as identifiers; treating them as
// keywords prevents def-use chains from carrying noise entries.
var cKeywords = map[string]bool{
	// C89/C90
	"auto": true, "break": true, "case": true, "char": true, "const": true,
	"continue": true, "default": true, "do": true, "double": true, "else": true,
	"enum": true, "extern": true, "float": true, "for": true, "goto": true,
	"if": true, "int": true, "long": true, "register": true, "return": true,
	"short": true, "signed": true, "sizeof": true, "static": true, "struct": true,
	"switch": true, "typedef": true, "union": true, "unsigned": true, "void": true,
	"volatile": true, "while": true,
	// C99
	"restrict": true, "inline": true, "_Bool": true,
	"_Complex": true, "_Imaginary": true,
	// C11
	"_Alignas": true, "_Alignof": true, "_Atomic": true, "_Generic": true,
	"_Noreturn": true, "_Static_assert": true, "_Thread_local": true,
	// C23
	"bool": true, "true": true, "false": true,
	"nullptr": true, "constexpr": true, "typeof": true,
	// Common constants treated as keywords for identifier filtering
	"NULL": true, "EOF": true,
}

// cppKeywords contains C++-only additions on top of cKeywords. It deliberately
// does NOT duplicate any entry already in cKeywords — IsCppKeyword merges
// both maps so that "const" and "class" both resolve correctly without the
// definitions drifting out of sync.
var cppKeywords = map[string]bool{
	"class": true, "namespace": true, "template": true, "typename": true,
	"public": true, "private": true, "protected": true, "virtual": true,
	"override": true, "final": true, "new": true, "delete": true,
	"this": true, "throw": true, "try": true, "catch": true,
	"using": true, "operator": true, "friend": true, "mutable": true,
	"explicit": true, "export": true,
	"consteval": true, "constinit": true,
	"co_await": true, "co_return": true, "co_yield": true,
	"concept": true, "requires": true, "decltype": true,
	"noexcept": true, "static_assert": true, "thread_local": true,
	"alignas": true, "alignof": true,
	// Common C++ standard-library types frequently used as bare identifiers.
	// Treating them as keywords prevents def-use chains from including them
	// as plain variable references when no qualifier is present.
	"string": true, "vector": true, "map": true, "set": true,
	"size_t": true, "ptrdiff_t": true,
	"wchar_t": true, "char8_t": true, "char16_t": true, "char32_t": true,
}

// IsCKeyword reports whether name is a C reserved word or one of the common
// C constants (NULL, EOF) that should be filtered out of variable lists.
func IsCKeyword(name string) bool {
	return cKeywords[name]
}

// IsCppKeyword reports whether name is reserved in C++ — either as a C
// keyword that C++ inherits, or as a C++-only addition. The caller should
// use this for C++ source files; C source should use IsCKeyword to avoid
// rejecting identifiers like "class" or "new" that are legal in C.
func IsCppKeyword(name string) bool {
	return cKeywords[name] || cppKeywords[name]
}
