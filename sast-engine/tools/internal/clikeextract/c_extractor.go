package clikeextract

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	clang "github.com/smacker/go-tree-sitter/c"
	sitter "github.com/smacker/go-tree-sitter"
)

// extractCHeader parses a single C header file and returns its symbol table
// as a *core.CStdlibHeader. Every entry is stamped with Source="header" —
// the overlay merge step (overlay.go) is responsible for promoting entries to
// "merged" or "overlay" status afterwards.
//
// Parse errors are returned as errors, not silently swallowed: callers (the
// orchestrator in extractor.go) decide whether to log-and-continue or abort
// the run. tree-sitter is forgiving and returns a (partial) parse tree even
// on syntax errors, so the most common return path is "(*CStdlibHeader, nil)"
// even for headers with macro magic the grammar can't fully digest.
func extractCHeader(file HeaderFile, src HeaderSource) (*core.CStdlibHeader, error) {
	source, err := os.ReadFile(file.Path) //nolint:gosec // file path is from filepath.WalkDir, not user input
	if err != nil {
		return nil, fmt.Errorf("extractCHeader: reading %q: %w", file.Path, err)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(clang.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, fmt.Errorf("extractCHeader: parsing %q: %w", file.Path, err)
	}
	defer tree.Close()

	h := newCHeader(file, src)
	walkCRoot(tree.RootNode(), source, h)
	return h, nil
}

// newCHeader allocates a fresh CStdlibHeader with the metadata fields filled
// in — Header, ModuleID, Language, Platform, SystemTag, GeneratedAt are all
// independent of which symbols the file contains, so they are pre-populated
// before AST walking starts.
func newCHeader(file HeaderFile, src HeaderSource) *core.CStdlibHeader {
	h := core.NewCStdlibHeader()
	h.SchemaVersion = SchemaVersion
	h.Header = file.Name
	h.ModuleID = "c::" + SanitizeHeaderName(file.Name)
	h.Language = src.Language
	h.Platform = src.Platform
	h.SystemTag = src.SystemTag
	return h
}

// walkCRoot visits every direct child of a translation_unit node and dispatches
// to the appropriate extractor. The C grammar nests most declarations directly
// under the root, so a one-level walk is sufficient — only `linkage_specification`
// (for `extern "C"`) and `preproc_if` (conditional sections) need recursive
// descent, and we handle them inline.
func walkCRoot(root *sitter.Node, source []byte, h *core.CStdlibHeader) {
	if root == nil {
		return
	}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child == nil {
			continue
		}
		walkCNode(child, source, h)
	}
}

// walkCNode dispatches one AST node to the matching extractor. Shape types
// not relevant to the public registry (pragmas, line directives, comments)
// are silently ignored.
func walkCNode(node *sitter.Node, source []byte, h *core.CStdlibHeader) {
	switch node.Type() {
	case "function_definition":
		extractCFunction(node, source, h)
	case "declaration":
		extractCDeclaration(node, source, h)
	case "type_definition":
		extractCTypedef(node, source, h)
	case "preproc_def":
		extractCPreprocDef(node, source, h)
	case "preproc_if", "preproc_ifdef", "preproc_else", "preproc_elif":
		// Conditional blocks: walk their bodies as if they were the outer
		// scope. Tree-sitter's grammar nests the body as named children
		// directly under the conditional node.
		for i := 0; i < int(node.NamedChildCount()); i++ {
			if c := node.NamedChild(i); c != nil {
				walkCNode(c, source, h)
			}
		}
	case "linkage_specification":
		// `extern "C" { ... }` blocks are unusual in pure C headers but
		// occasionally appear when a header is shared with C++ users.
		// Walk the inner body the same way.
		body := node.ChildByFieldName("body")
		if body == nil {
			return
		}
		for i := 0; i < int(body.NamedChildCount()); i++ {
			if c := body.NamedChild(i); c != nil {
				walkCNode(c, source, h)
			}
		}
	}
}

// extractCFunction handles a function_definition node — uncommon in headers
// (most just declare prototypes) but legal for inline definitions in glibc
// and libc++. Reuses clike.ExtractFunctionInfo which returns a normalised
// FunctionInfo regardless of whether the body is present.
func extractCFunction(node *sitter.Node, source []byte, h *core.CStdlibHeader) {
	info := clike.ExtractFunctionInfo(node, source)
	if info == nil || info.Name == "" {
		return
	}
	if IsPrivateSymbol(info.Name) {
		return
	}
	h.Functions[info.Name] = makeFunctionFromInfo(h, info)
}

// extractCDeclaration handles a `declaration` node, which is the typical shape
// for header function prototypes (`int printf(const char* fmt, ...);`). The
// declaration's declarator chain may be:
//
//   - function_declarator                    → bare prototype
//   - pointer_declarator(function_declarator)→ function returning a pointer
//   - identifier                             → variable declaration (skipped here)
//
// Only the function-prototype form is registered as a CStdlibFunction.
// Plain variable declarations at the top of headers (rare; e.g. `extern int
// errno;`) are skipped — they belong to a future "globals" extension.
func extractCDeclaration(node *sitter.Node, source []byte, h *core.CStdlibHeader) {
	declarator := node.ChildByFieldName("declarator")
	funcDecl := unwrapToFunctionDeclaratorPublic(declarator)
	if funcDecl == nil {
		return
	}
	info := clike.ExtractFunctionInfo(node, source)
	if info == nil || info.Name == "" {
		return
	}
	if IsPrivateSymbol(info.Name) {
		return
	}
	h.Functions[info.Name] = makeFunctionFromInfo(h, info)
}

// extractCTypedef converts a `type_definition` node into a CStdlibTypedef
// entry. The C grammar exposes the underlying type as the `type` field and
// the new name as the `declarator` field's identifier, with optional
// pointer/array wrappers we must walk through.
func extractCTypedef(node *sitter.Node, source []byte, h *core.CStdlibHeader) {
	declarator := node.ChildByFieldName("declarator")
	name := unwrapDeclaratorIdentifier(declarator, source)
	if name == "" || IsPrivateSymbol(name) {
		return
	}
	typeNode := node.ChildByFieldName("type")
	underlying := clike.ExtractTypeString(typeNode, declarator, source)
	h.Typedefs[name] = &core.CStdlibTypedef{
		Type:   NormalizeType(underlying),
		Source: core.SourceHeader,
	}
}

// extractCPreprocDef handles `#define NAME value` lines. Only literals (integer,
// negative integer, string, character, hex/octal) are emitted; expression bodies
// and function-like macros are skipped. The conservative cut keeps the output
// trustworthy: registry consumers can rely on `Value` being a literal that round-
// trips through `strconv` or pure string comparison.
func extractCPreprocDef(node *sitter.Node, source []byte, h *core.CStdlibHeader) {
	nameNode := node.ChildByFieldName("name")
	valueNode := node.ChildByFieldName("value")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(source)
	if IsPrivateSymbol(name) {
		return
	}

	value := ""
	typ := ""
	if valueNode != nil {
		value = strings.TrimSpace(valueNode.Content(source))
		// Drop comments and trailing semicolons that sometimes sneak into
		// the right-hand side via tree-sitter's lossy macro tokenization.
		value = stripTrailingComment(value)
		typ = inferConstantType(value)
		if typ == "" {
			// Non-literal body: skip rather than emit something we can't
			// give a type to. Future overlay extension can backfill.
			return
		}
	}

	h.Constants[name] = &core.CStdlibConstant{
		Type:   typ,
		Value:  value,
		Source: core.SourceHeader,
	}
}

// makeFunctionFromInfo converts a clike.FunctionInfo into a core.CStdlibFunction
// with the canonical FQN and Source="header". Variadic positions in the
// FunctionInfo (`ParamNames[i] == "..."`) are translated into the registry's
// variadic convention (CStdlibParam{Name: "...", Type: "variadic", Required:
// false}).
func makeFunctionFromInfo(h *core.CStdlibHeader, info *clike.FunctionInfo) *core.CStdlibFunction {
	params := make([]*core.CStdlibParam, 0, len(info.ParamNames))
	for i, name := range info.ParamNames {
		typ := ""
		if i < len(info.ParamTypes) {
			typ = info.ParamTypes[i]
		}
		if name == "..." || typ == "..." {
			params = append(params, &core.CStdlibParam{Name: "...", Type: "variadic", Required: false})
			continue
		}
		params = append(params, &core.CStdlibParam{
			Name:     name,
			Type:     NormalizeType(typ),
			Required: true,
		})
	}

	return &core.CStdlibFunction{
		FQN:        h.ModuleID + "::" + info.Name,
		ReturnType: NormalizeType(info.ReturnType),
		Params:     params,
		Confidence: 1.0,
		Source:     core.SourceHeader,
	}
}

// unwrapToFunctionDeclaratorPublic walks past pointer wrappers and returns the
// inner function_declarator, or nil if there isn't one. Mirrors the unexported
// helper in graph/clike/declarations.go but reachable here.
func unwrapToFunctionDeclaratorPublic(declarator *sitter.Node) *sitter.Node {
	for cur := declarator; cur != nil; {
		if cur.Type() == "function_declarator" {
			return cur
		}
		next := cur.ChildByFieldName("declarator")
		if next == nil {
			return nil
		}
		cur = next
	}
	return nil
}

// unwrapDeclaratorIdentifier walks down a declarator chain and returns the
// bare identifier name, stripping pointer / array / reference wrappers.
// Returns "" if no identifier is reachable.
//
// `primitive_type` is included in the leaf set because tree-sitter's C grammar
// reclassifies recognised typedef names (size_t, ssize_t, pid_t, …) as
// primitive types via a built-in lookup table. The token spelling is still
// a valid identifier; we just can't rely on the AST shape alone to tell us
// "this is the new name".
func unwrapDeclaratorIdentifier(declarator *sitter.Node, source []byte) string {
	for cur := declarator; cur != nil; {
		switch cur.Type() {
		case "type_identifier", "identifier", "primitive_type":
			return cur.Content(source)
		case "pointer_declarator", "reference_declarator", "array_declarator":
			cur = cur.ChildByFieldName("declarator")
			if cur == nil {
				return ""
			}
		case "function_declarator":
			cur = cur.ChildByFieldName("declarator")
			if cur == nil {
				return ""
			}
		default:
			return ""
		}
	}
	return ""
}

// stripTrailingComment removes a `// ...` or `/* ... */` tail that some
// preprocessor definitions carry alongside the value. Non-greedy: stops at
// the first comment-introducing token and trims trailing whitespace.
func stripTrailingComment(s string) string {
	if i := strings.Index(s, "/*"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "//"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, " \t")
}

// inferConstantType returns a best-guess type for a #define right-hand side.
// The caller drops the entry if this returns "" — that signals "expression
// too complex; refusing to emit unsound type info".
func inferConstantType(value string) string {
	if value == "" {
		return ""
	}
	// Strip redundant parens so common forms like `EOF (-1)` and `(0x10)`
	// classify as the integer literal they represent.
	stripped := stripBalancedOuterParens(value)

	// Integer / negative integer / hex / octal — `int` is conservative; the
	// overlay can refine to `long`/`unsigned int`/etc. for specific constants
	// (BUFSIZ, EOF, etc.) when needed.
	if isIntegerLiteral(stripped) {
		return "int"
	}
	// String literal.
	if strings.HasPrefix(stripped, `"`) && strings.HasSuffix(stripped, `"`) {
		return "const char*"
	}
	// Char literal.
	if strings.HasPrefix(stripped, "'") && strings.HasSuffix(stripped, "'") {
		return "char"
	}
	return ""
}

// stripBalancedOuterParens removes a single layer of `(…)` wrapping iff the
// opening `(` matches the closing `)` at the very ends of the string. Repeats
// until no more outer wrap. Conservative: only strips when balanced — does
// NOT touch `((void*)0)` (the outer ) doesn't balance the leading () shape).
func stripBalancedOuterParens(s string) string {
	for {
		s = strings.TrimSpace(s)
		if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
			return s
		}
		// Only strip when paren depth never returns to 0 before the very last char.
		depth := 0
		ok := true
		for i, r := range s {
			switch r {
			case '(':
				depth++
			case ')':
				depth--
			}
			if depth == 0 && i < len(s)-1 {
				ok = false
				break
			}
		}
		if !ok {
			return s
		}
		s = s[1 : len(s)-1]
	}
}

func isIntegerLiteral(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '-' || s[0] == '+' {
		i = 1
	}
	if i >= len(s) {
		return false
	}
	// Hex literal `0x..`
	if i+1 < len(s) && s[i] == '0' && (s[i+1] == 'x' || s[i+1] == 'X') {
		hex := s[i+2:]
		if hex == "" {
			return false
		}
		for _, r := range hex {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
		return true
	}
	// Plain decimal / octal.
	for _, r := range s[i:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
