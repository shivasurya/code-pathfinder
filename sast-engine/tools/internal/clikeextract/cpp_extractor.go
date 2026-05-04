package clikeextract

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	cpplang "github.com/smacker/go-tree-sitter/cpp"
	sitter "github.com/smacker/go-tree-sitter"
)

// extractCppHeader parses a single C++ header file and returns its symbol table
// as a *core.CStdlibHeader. Compared to extractCHeader, this walker tracks an
// additional piece of state — the current namespace stack — so methods, classes,
// and free functions are emitted with their fully-qualified name.
//
// Stamping rules:
//
//   - Classes go into h.Classes keyed by FQN ("std::vector").
//   - Class methods go into Class.Methods keyed by bare name ("push_back").
//   - Free functions inside namespaces go into h.FreeFunctions keyed by FQN
//     ("std::move").
//   - Free functions at file scope (rare in headers) go into h.Functions keyed
//     by bare name. This is the same map C uses, so a header that mixes C
//     and C++ shapes still emits a coherent registry.
//   - Top-level namespaces (`std`, `boost`, …) are recorded in h.Namespaces.
func extractCppHeader(file HeaderFile, src HeaderSource) (*core.CStdlibHeader, error) {
	source, err := os.ReadFile(file.Path) //nolint:gosec // path is from filepath.WalkDir
	if err != nil {
		return nil, fmt.Errorf("extractCppHeader: reading %q: %w", file.Path, err)
	}

	// libstdc++ headers borrow the same `__attribute__((...))` /
	// `__nonnull((...))` decorations as glibc — strip them before parsing
	// so trailing attribute chains don't trip tree-sitter into ERROR nodes
	// that swallow class methods or free function declarations.
	source = preprocessGlibcAttributes(source)

	parser := sitter.NewParser()
	parser.SetLanguage(cpplang.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, fmt.Errorf("extractCppHeader: parsing %q: %w", file.Path, err)
	}
	defer tree.Close()

	h := newCppHeader(file, src)
	w := &cppWalker{header: h, source: source}
	w.walk(tree.RootNode())
	finaliseNamespaces(h)
	return h, nil
}

func newCppHeader(file HeaderFile, src HeaderSource) *core.CStdlibHeader {
	h := core.NewCStdlibHeader()
	h.SchemaVersion = SchemaVersion
	h.Header = file.Name
	h.ModuleID = "std::" + SanitizeHeaderName(file.Name)
	h.Language = src.Language
	h.Platform = src.Platform
	h.SystemTag = src.SystemTag
	return h
}

// cppWalker carries the mutable walk state. namespaceStack is updated on entry
// to namespace_definition and reverted on exit. templateStack carries the most
// recent template_parameter_list — when the next class_specifier or function is
// extracted, it consumes (and clears) the entry.
type cppWalker struct {
	header *core.CStdlibHeader
	source []byte

	// namespaceStack is the chain of currently-open namespaces, deepest last.
	namespaceStack []string
	// pendingTemplate is a template_parameter_list node whose enclosing
	// declaration has not been seen yet. The walker sets this on entering a
	// template_declaration and clears it after dispatching the child.
	pendingTemplate *sitter.Node
}

func (w *cppWalker) walk(node *sitter.Node) {
	if node == nil {
		return
	}
	switch node.Type() {
	case "namespace_definition":
		w.walkNamespace(node)
	case "template_declaration":
		w.walkTemplate(node)
	case "class_specifier", "struct_specifier":
		w.extractClass(node, w.pendingTemplate)
		w.pendingTemplate = nil
	case "function_definition":
		w.extractFreeFunctionFromDefinition(node)
	case "declaration":
		w.extractFreeDeclaration(node)
	case "type_definition":
		w.extractCppTypedef(node)
	case "preproc_def":
		extractCPreprocDef(node, w.source, w.header)
	case "preproc_if", "preproc_ifdef", "preproc_else", "preproc_elif",
		"declaration_list", "translation_unit":
		w.walkChildren(node)
	case "linkage_specification":
		body := node.ChildByFieldName("body")
		w.walkChildren(body)
	}
}

func (w *cppWalker) walkChildren(node *sitter.Node) {
	if node == nil {
		return
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		w.walk(node.NamedChild(i))
	}
}

// walkNamespace pushes the namespace name onto the stack, walks the body, then
// pops. Anonymous namespaces (no namespace_identifier child) are skipped — by
// definition they don't contribute symbols visible to other translation units,
// and the registry only describes the public surface.
func (w *cppWalker) walkNamespace(node *sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(w.source)
	if IsPrivateNamespace(name) {
		return
	}
	w.namespaceStack = append(w.namespaceStack, name)
	defer func() { w.namespaceStack = w.namespaceStack[:len(w.namespaceStack)-1] }()

	body := node.ChildByFieldName("body")
	w.walkChildren(body)
}

// walkTemplate captures the template_parameter_list and dispatches to the next
// child (which is the entity being templated — a class or function). After
// extraction the pendingTemplate is cleared.
func (w *cppWalker) walkTemplate(node *sitter.Node) {
	w.pendingTemplate = nil
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == "template_parameter_list" {
			w.pendingTemplate = child
			continue
		}
		w.walk(child)
	}
	w.pendingTemplate = nil
}

// extractClass converts a class_specifier or struct_specifier into a
// CppStdlibClass entry. Methods are pulled from the field_declaration_list;
// constructors are recognised as declarations whose name matches the class.
// Template parameters from the enclosing template_declaration (if any) are
// recorded on the class.
func (w *cppWalker) extractClass(node *sitter.Node, tmpl *sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return // anonymous class — no public surface
	}
	bareName := nameNode.Content(w.source)
	if IsPrivateSymbol(bareName) {
		return
	}
	fqn := w.qualified(bareName)

	cls, ok := w.header.Classes[fqn]
	if !ok {
		cls = core.NewCppStdlibClass(fqn)
		w.header.Classes[fqn] = cls
	}

	if tmpl != nil {
		cls.TypeParams = extractTemplateParamNames(tmpl, w.source)
	}

	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	w.walkClassBody(body, cls, bareName)
}

// walkClassBody iterates a field_declaration_list and routes each child to the
// right extractor: field_declaration → method or data field; declaration →
// constructor (when its name matches the class).
func (w *cppWalker) walkClassBody(body *sitter.Node, cls *core.CppStdlibClass, className string) {
	for i := 0; i < int(body.NamedChildCount()); i++ {
		child := body.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "field_declaration":
			w.extractMaybeMethod(child, cls)
		case "declaration":
			w.extractMaybeConstructor(child, cls, className)
		case "preproc_if", "preproc_ifdef", "preproc_else", "preproc_elif":
			// Conditional inside a class body: walk recursively, treating
			// each matching child node as part of the class body.
			w.walkClassBody(child, cls, className)
		}
	}
}

// extractMaybeMethod inspects a field_declaration. If its declarator is a
// function_declarator (possibly through a pointer / reference wrapper), we
// treat it as a method; otherwise it's a data field and we ignore it.
//
// Implementation note: Phase 1's clike.ExtractFunctionInfo does not see through
// reference_declarator wrappers (it only follows the "declarator" field, which
// reference_declarator does not expose). So for member functions returning T&
// (`T& at(...)`, `vector& operator+=(...)`) we extract the name manually via
// findFunctionDeclarator below, then build the FunctionInfo ourselves rather
// than going through clike.
func (w *cppWalker) extractMaybeMethod(node *sitter.Node, cls *core.CppStdlibClass) {
	declarator := node.ChildByFieldName("declarator")
	funcDecl := findFunctionDeclarator(declarator)
	if funcDecl == nil {
		return
	}
	name := functionDeclaratorName(funcDecl, w.source)
	if name == "" || IsPrivateSymbol(name) {
		return
	}

	typeNode := node.ChildByFieldName("type")
	returnType := clike.ExtractTypeString(typeNode, declarator, w.source)
	paramList := funcDecl.ChildByFieldName("parameters")
	pNames, pTypes := clike.ExtractParameters(paramList, w.source)

	info := &clike.FunctionInfo{
		Name:       name,
		ReturnType: returnType,
		ParamNames: pNames,
		ParamTypes: pTypes,
	}
	cls.Methods[name] = w.makeMethod(cls, info)
}

// extractMaybeConstructor recognises declarations inside a class body whose
// declarator name matches the class name — those are constructors. Anything
// else inside a `declaration` node at class scope is ignored (rare; e.g.
// nested using-declarations).
func (w *cppWalker) extractMaybeConstructor(node *sitter.Node, cls *core.CppStdlibClass, className string) {
	declarator := node.ChildByFieldName("declarator")
	if declarator == nil || declarator.Type() != "function_declarator" {
		return
	}
	innerDecl := declarator.ChildByFieldName("declarator")
	if innerDecl == nil || innerDecl.Content(w.source) != className {
		return
	}
	paramList := declarator.ChildByFieldName("parameters")
	names, types := clike.ExtractParameters(paramList, w.source)

	params := make([]*core.CStdlibParam, 0, len(names))
	for i, n := range names {
		typ := ""
		if i < len(types) {
			typ = types[i]
		}
		params = append(params, &core.CStdlibParam{
			Name:     n,
			Type:     NormalizeType(typ),
			Required: true,
		})
	}
	cls.Constructors = append(cls.Constructors, &core.CppStdlibConstructor{
		Params: params,
		Source: core.SourceHeader,
	})
}

// extractCppTypedef is the C++ analog of extractCTypedef but emits typedef
// entries with the namespace prefix when one is active. The registry does not
// currently key typedefs by FQN — they live in a flat map — so namespaced
// typedefs land under their bare name; future schema work may move them to
// FQN-keyed maps.
func (w *cppWalker) extractCppTypedef(node *sitter.Node) {
	declarator := node.ChildByFieldName("declarator")
	name := unwrapDeclaratorIdentifier(declarator, w.source)
	if name == "" || IsPrivateSymbol(name) {
		return
	}
	typeNode := node.ChildByFieldName("type")
	underlying := clike.ExtractTypeString(typeNode, declarator, w.source)
	w.header.Typedefs[name] = &core.CStdlibTypedef{
		Type:   NormalizeType(underlying),
		Source: core.SourceHeader,
	}
}

// extractFreeDeclaration handles a top-level (or namespace-scoped) `declaration`
// node — typically a free-function prototype. The result is keyed in either
// h.Functions (file-scope) or h.FreeFunctions (namespaced) depending on whether
// the namespace stack is currently non-empty.
func (w *cppWalker) extractFreeDeclaration(node *sitter.Node) {
	declarator := node.ChildByFieldName("declarator")
	funcDecl := findFunctionDeclarator(declarator)
	if funcDecl == nil {
		return
	}
	name := functionDeclaratorName(funcDecl, w.source)
	if name == "" || IsPrivateSymbol(name) {
		return
	}
	typeNode := node.ChildByFieldName("type")
	returnType := clike.ExtractTypeString(typeNode, declarator, w.source)
	paramList := funcDecl.ChildByFieldName("parameters")
	pNames, pTypes := clike.ExtractParameters(paramList, w.source)

	info := &clike.FunctionInfo{
		Name:       name,
		ReturnType: returnType,
		ParamNames: pNames,
		ParamTypes: pTypes,
	}
	fn := &core.CStdlibFunction{
		FQN:        w.qualified(name),
		ReturnType: NormalizeType(info.ReturnType),
		Params:     paramListFromInfo(info),
		Confidence: 1.0,
		Source:     core.SourceHeader,
	}
	if len(w.namespaceStack) == 0 {
		w.header.Functions[name] = fn
		return
	}
	w.header.FreeFunctions[fn.FQN] = fn
}

// extractFreeFunctionFromDefinition handles the inline-definition case (rare
// in stdlib headers but used by libstdc++ for trivial wrappers).
func (w *cppWalker) extractFreeFunctionFromDefinition(node *sitter.Node) {
	info := clike.ExtractFunctionInfo(node, w.source)
	if info == nil || info.Name == "" {
		return
	}
	if IsPrivateSymbol(info.Name) {
		return
	}
	fn := &core.CStdlibFunction{
		FQN:        w.qualified(info.Name),
		ReturnType: NormalizeType(info.ReturnType),
		Params:     paramListFromInfo(info),
		Confidence: 1.0,
		Source:     core.SourceHeader,
	}
	if len(w.namespaceStack) == 0 {
		w.header.Functions[info.Name] = fn
		return
	}
	w.header.FreeFunctions[fn.FQN] = fn
}

// makeMethod converts a clike.FunctionInfo into a CStdlibFunction whose FQN is
// `class::method`. Variadic params follow the same convention as the C path.
func (w *cppWalker) makeMethod(cls *core.CppStdlibClass, info *clike.FunctionInfo) *core.CStdlibFunction {
	return &core.CStdlibFunction{
		FQN:        cls.FQN + "::" + info.Name,
		ReturnType: NormalizeType(info.ReturnType),
		Params:     paramListFromInfo(info),
		Confidence: 1.0,
		Source:     core.SourceHeader,
	}
}

// qualified returns name prefixed by the current namespace stack
// (`std::sub::name`), or just name if the stack is empty.
func (w *cppWalker) qualified(name string) string {
	if len(w.namespaceStack) == 0 {
		return name
	}
	return strings.Join(w.namespaceStack, "::") + "::" + name
}

// extractTemplateParamNames pulls the bare type-parameter names out of a
// template_parameter_list. We only capture names; constraints and default
// arguments are ignored at this layer (the overlay is the right place for
// concrete defaults like `Allocator = std::allocator<T>`).
func extractTemplateParamNames(list *sitter.Node, source []byte) []string {
	if list == nil {
		return nil
	}
	var names []string
	for i := 0; i < int(list.NamedChildCount()); i++ {
		child := list.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "type_parameter_declaration", "optional_type_parameter_declaration",
			"variadic_type_parameter_declaration":
			if id := findFirstIdentifier(child, source); id != "" {
				names = append(names, id)
			}
		case "parameter_declaration":
			// Non-type template parameters (e.g., `int N`). Capture the bare
			// identifier from the declarator field; tree-sitter exposes it as
			// `declarator -> identifier`.
			decl := child.ChildByFieldName("declarator")
			if decl != nil && decl.Type() == "identifier" {
				names = append(names, decl.Content(source))
			}
		}
	}
	return names
}

// findFunctionDeclarator walks past pointer_declarator / reference_declarator /
// abstract_*_declarator wrappers and returns the inner function_declarator,
// or nil if none. Unlike clike's helper, this also follows the first-named-
// child fallback used when a wrapper does not expose its body via the
// "declarator" field — necessary for C++ reference_declarator nodes.
func findFunctionDeclarator(node *sitter.Node) *sitter.Node {
	for cur := node; cur != nil; {
		if cur.Type() == "function_declarator" {
			return cur
		}
		cur = innerDeclaratorChild(cur)
	}
	return nil
}

// functionDeclaratorName extracts the name from a function_declarator. The
// name lives under the "declarator" field; that may itself be wrapped (an
// `operator[]` or destructor `~T()` form), so we walk it recursively until
// we reach an identifier-like leaf or run out of declarator field links.
func functionDeclaratorName(funcDecl *sitter.Node, source []byte) string {
	if funcDecl == nil {
		return ""
	}
	name := funcDecl.ChildByFieldName("declarator")
	if name == nil {
		return ""
	}
	// Common leaf shapes: identifier, field_identifier, operator_name,
	// destructor_name, qualified_identifier, template_function (for
	// `function<T>()` declarations — rare in stdlib but legal).
	switch name.Type() {
	case "identifier", "field_identifier", "operator_name", "destructor_name":
		return name.Content(source)
	case "qualified_identifier":
		return name.Content(source)
	case "template_function":
		// Template function call form — emit the bare name.
		if id := name.ChildByFieldName("name"); id != nil {
			return id.Content(source)
		}
		return name.Content(source)
	default:
		// Fallback: dump the raw text. This may include ref/ptr decoration
		// for unusual shapes; downstream consumers can normalise.
		return strings.TrimSpace(name.Content(source))
	}
}

// innerDeclaratorChild returns the next declarator inside a wrapper, falling
// back to the first non-qualifier named child when the "declarator" field is
// not exposed by the grammar (the case for C++ reference_declarator).
func innerDeclaratorChild(wrapper *sitter.Node) *sitter.Node {
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
		if c.Type() == "type_qualifier" {
			continue
		}
		return c
	}
	return nil
}

// findFirstIdentifier returns the content of the first identifier or
// type_identifier descendant of node. Used to pull a parameter name out of a
// template_parameter_declaration that may have a default argument or
// constraint preceding/following the name.
func findFirstIdentifier(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		c := node.NamedChild(i)
		if c == nil {
			continue
		}
		if c.Type() == "type_identifier" || c.Type() == "identifier" {
			return c.Content(source)
		}
	}
	return ""
}

// paramListFromInfo is the C++ analog of makeFunctionFromInfo's param block —
// extracted so methods, constructors, and free functions all share identical
// param-shape conversion logic.
func paramListFromInfo(info *clike.FunctionInfo) []*core.CStdlibParam {
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
	return params
}

// finaliseNamespaces deduplicates and stamps the top-level namespace list onto
// the header. It is computed by inspecting all FreeFunctions and Classes for
// their FQN prefix.
func finaliseNamespaces(h *core.CStdlibHeader) {
	seen := map[string]struct{}{}
	add := func(fqn string) {
		idx := strings.Index(fqn, "::")
		if idx <= 0 {
			return
		}
		ns := fqn[:idx]
		if _, ok := seen[ns]; ok {
			return
		}
		seen[ns] = struct{}{}
		h.Namespaces = append(h.Namespaces, ns)
	}
	for fqn := range h.Classes {
		add(fqn)
	}
	for fqn := range h.FreeFunctions {
		add(fqn)
	}
}
