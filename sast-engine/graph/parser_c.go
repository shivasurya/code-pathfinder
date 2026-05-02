package graph

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	sitter "github.com/smacker/go-tree-sitter"
)

// parser_c.go converts tree-sitter C AST nodes into graph.Node objects.
// The dispatcher in buildGraphFromAST (parser.go) selects the parse
// functions declared here for files whose extension routes them to the
// tree-sitter C grammar — every entry point sets Language="c" on the
// produced node.
//
// C++ has its own dispatcher (parser_cpp.go) — separate file, separate
// constructs (classes, namespaces, templates, throw/try). The two parsers
// share the AST extraction primitives in graph/clike (function metadata,
// type strings, parameter lists, struct fields, call info), and a small
// set of language-neutral helpers (childrenByFieldName, bareIdentifierName,
// extractTaggedName, lineRange, newSourceLocation, scopeFromContext,
// languageOfFile, normaliseIncludePath) that live at the bottom of this
// file. Two parse functions — parseCLikeDeclaration and parseCLikeInclude —
// are deliberately language-neutral and accept an isCpp flag because the
// AST shape for variable declarations and #include directives is identical
// between the two grammars.

// Language tags used as Node.Language values for the C/C++ parsers.
const (
	languageC   = "c"
	languageCpp = "cpp"
)

// Node.Type values produced by the C parser. C++-only types
// (method_declaration, class_declaration, ThrowStmt, TryStmt, …) live in
// parser_cpp.go.
const (
	nodeTypeFunctionDefinition = "function_definition"
	nodeTypeStructDeclaration  = "struct_declaration"
	nodeTypeEnumDeclaration    = "enum_declaration"
	nodeTypeTypeDefinition     = "type_definition"
	nodeTypeVariableDecl       = "variable_declaration"
	nodeTypeCallExpression     = "call_expression"
	nodeTypeIncludeStatement   = "include_statement"
)

// Metadata keys produced by the C parser. The keys covering call-shape
// detection (is_method / is_arrow / is_qualified / receiver) are shared
// with the C++ parser because clike.CallInfo classifies all four call
// shapes regardless of language — C can also produce arrow-method calls
// through function pointers.
const (
	metaIsDeclaration  = "is_declaration"
	metaSystemInclude  = "system_include"
	metaStorageClasses = "storage_classes"
	metaEnumerators    = "enumerators"
	metaUnderlyingType = "underlying_type"
	metaIsAnonymous    = "is_anonymous"
	metaIsMethod       = "is_method"
	metaIsArrow        = "is_arrow"
	metaIsQualified    = "is_qualified"
	metaReceiver       = "receiver"
)

// =============================================================================
// Function definitions
// =============================================================================

// parseCFunctionDefinition converts a tree-sitter `function_definition` (or a
// top-level forward declaration of the same shape) into a graph.Node. The
// returned node becomes the currentContext for any nested AST traversal so
// call expressions inside the body can be linked back to their enclosing
// function.
//
// IsDeclaration semantics: a function with no compound_statement body
// (typical in headers, e.g. `int add(int a, int b);`) gets
// Metadata["is_declaration"] = true. Resolved definitions in .c files do
// not set this key.
//
// Storage-class qualifiers (static, inline, extern, _Noreturn, _Thread_local)
// appear as storage_class_specifier siblings in the AST. They are collected
// into Metadata["storage_classes"] for downstream rule writers and into
// Modifier (joined by space) for ergonomic single-string access.
func parseCFunctionDefinition(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) *Node {
	info := clike.ExtractFunctionInfo(node, sourceCode)
	if info == nil {
		return nil
	}

	storageClasses := collectStorageClassSpecifiers(node, sourceCode)
	metadata := map[string]any{}
	if info.IsDeclaration {
		metadata[metaIsDeclaration] = true
	}
	if len(storageClasses) > 0 {
		metadata[metaStorageClasses] = storageClasses
	}

	functionNode := &Node{
		ID:                   GenerateMethodID("function:"+info.Name, info.ParamTypes, file, info.LineNumber),
		Type:                 nodeTypeFunctionDefinition,
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.ParamTypes,
		MethodArgumentsValue: info.ParamNames,
		Modifier:             strings.Join(storageClasses, " "),
		File:                 file,
		Language:             languageC,
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	}
	graph.AddNode(functionNode)
	return functionNode
}

// =============================================================================
// Struct / Enum / Typedef
// =============================================================================

// parseCStructSpecifier records a `struct_specifier` declaration. Anonymous
// structs (no name child, common when used inline as a typedef target or
// declaration type) are still recorded so downstream rules can scope them
// to their declaring location; Metadata["is_anonymous"] = true marks them.
//
// Fields are stored as MethodArgumentsType (parallel slice of "name: type"
// strings) — reusing the existing field on graph.Node avoids a Metadata
// allocation for what is the most common access pattern.
func parseCStructSpecifier(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	body := node.ChildByFieldName("body")
	if body == nil {
		// `struct S` used as a type reference (e.g. `struct S* p`) — not a
		// declaration. Skip; the variable_declaration / parameter that
		// references it carries the type information.
		return
	}

	name, isAnonymous := extractTaggedName(node, sourceCode)
	fields := clike.ExtractStructFields(body, sourceCode)
	fieldStrings := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Name == "" {
			fieldStrings = append(fieldStrings, f.TypeStr)
		} else {
			fieldStrings = append(fieldStrings, f.Name+": "+f.TypeStr)
		}
	}

	metadata := map[string]any{}
	if isAnonymous {
		metadata[metaIsAnonymous] = true
	}

	graph.AddNode(&Node{
		ID:                  GenerateSha256("struct:" + name + "@" + file + "#" + lineRange(node)),
		Type:                nodeTypeStructDeclaration,
		Name:                name,
		LineNumber:          node.StartPoint().Row + 1,
		MethodArgumentsType: fieldStrings,
		File:                file,
		Language:            languageC,
		SourceLocation:      newSourceLocation(file, node),
		Metadata:            metadata,
	})
}

// parseCEnumSpecifier records an `enum_specifier`. Enumerators are stored in
// Metadata["enumerators"] as a []string of "NAME" or "NAME=VALUE" entries —
// keeping the original source form so rule writers see what authors wrote.
func parseCEnumSpecifier(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	body := node.ChildByFieldName("body")
	if body == nil {
		// `enum E` used as a type reference. Skip — same reasoning as
		// parseCStructSpecifier.
		return
	}

	name, isAnonymous := extractTaggedName(node, sourceCode)
	enumerators := extractEnumerators(body, sourceCode)

	metadata := map[string]any{
		metaEnumerators: enumerators,
	}
	if isAnonymous {
		metadata[metaIsAnonymous] = true
	}

	graph.AddNode(&Node{
		ID:             GenerateSha256("enum:" + name + "@" + file + "#" + lineRange(node)),
		Type:           nodeTypeEnumDeclaration,
		Name:           name,
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageC,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       metadata,
	})
}

// parseCTypeDefinition records a `type_definition` (typedef). The aliased
// type goes into DataType (e.g. "unsigned long", "struct { int x; int y; }")
// and into Metadata["underlying_type"] so consumers needing the structured
// form versus the alias can distinguish them.
//
// Multiple declarators in one typedef (`typedef int a, b, c;`) emit one
// graph node per alias name.
func parseCTypeDefinition(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string) {
	typeNode := node.ChildByFieldName("type")
	underlying := ""
	if typeNode != nil {
		underlying = strings.TrimSpace(typeNode.Content(sourceCode))
	}

	for _, declarator := range childDeclarators(node) {
		aliasName := bareIdentifierName(declarator, sourceCode)
		if aliasName == "" {
			aliasName = strings.TrimSpace(declarator.Content(sourceCode))
		}
		graph.AddNode(&Node{
			ID:             GenerateSha256("typedef:" + aliasName + "@" + file + "#" + lineRange(node)),
			Type:           nodeTypeTypeDefinition,
			Name:           aliasName,
			DataType:       underlying,
			LineNumber:     node.StartPoint().Row + 1,
			File:           file,
			Language:       languageC,
			SourceLocation: newSourceLocation(file, node),
			Metadata:       map[string]any{metaUnderlyingType: underlying},
		})
	}
}

// =============================================================================
// Variable declarations (shared with C++ via isCpp flag)
// =============================================================================

// parseCLikeDeclaration records every variable introduced by a `declaration`
// node. The same code handles C and C++ because tree-sitter exposes the
// node identically in both grammars; the isCpp flag only changes the
// Language tag on the produced graph nodes.
//
// Multi-declarator forms (`int a = 1, b = 2, c;`) emit one graph node per
// variable. Each declarator is unwrapped with bareIdentifierName so pointer
// and array wrappers contribute to DataType (via clike.ExtractTypeString)
// rather than to Name.
//
// Initialisers (the `=` value) are captured as VariableValue when present.
// The Scope is "global" at translation-unit scope or the enclosing
// function's name when the declaration sits inside a function body —
// currentContext (set during AST descent in buildGraphFromAST) carries
// the latter.
func parseCLikeDeclaration(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node, isCpp bool) {
	// A `declaration` node whose declarator chain reaches a function_declarator
	// is a function prototype (forward declaration). Emit a
	// function_definition node so callers and call-graph builders find it
	// alongside actual definitions; Metadata["is_declaration"] = true
	// distinguishes the prototype from a body-bearing definition.
	//
	// In C++, the same shape is used by tree-sitter for destructors and
	// inline method declarations that don't go through field_declaration
	// (e.g. `~ClassName();` inside a class body). When we are in class
	// context we route to the C++ helper which emits a method_declaration
	// node instead — keeping rule writers free of dispatch concerns.
	if isFunctionPrototype(node) {
		if isCpp {
			if classNode := classFromContext(currentContext); classNode != nil {
				emitCppMethodDeclarationFromDeclaration(node, sourceCode, graph, file, classNode)
				return
			}
		}
		emitFunctionDeclaration(node, sourceCode, graph, file, isCpp)
		return
	}

	typeNode := node.ChildByFieldName("type")
	scope := scopeFromContext(currentContext)
	language := languageOfFile(isCpp)
	lineNumber := node.StartPoint().Row + 1

	for _, declarator := range childDeclarators(node) {
		name, valueText := bareIdentifierAndInitialiser(declarator, sourceCode)
		if name == "" {
			continue
		}
		dataType := clike.ExtractTypeString(typeNode, declarator, sourceCode)

		graph.AddNode(&Node{
			ID:             GenerateSha256("var:" + scope + "::" + name + "@" + file + "#" + lineRange(node)),
			Type:           nodeTypeVariableDecl,
			Name:           name,
			DataType:       dataType,
			VariableValue:  valueText,
			Scope:          scope,
			LineNumber:     lineNumber,
			File:           file,
			Language:       language,
			SourceLocation: newSourceLocation(file, node),
		})
	}
}

// isFunctionPrototype reports whether a `declaration` node carries a
// function_declarator (a forward declaration like `int add(int, int);`).
// Multi-declarator declarations (`int x; int f();`) are unusual but legal —
// any declarator being a function_declarator is enough to treat the whole
// declaration as a prototype, which matches what real C codebases do.
func isFunctionPrototype(node *sitter.Node) bool {
	for _, declarator := range childDeclarators(node) {
		for cur := declarator; cur != nil; cur = cur.ChildByFieldName("declarator") {
			if cur.Type() == "function_declarator" {
				return true
			}
		}
	}
	return false
}

// emitFunctionDeclaration produces a function_definition graph node from a
// body-less `declaration` (a function prototype). The shape mirrors
// parseCFunctionDefinition so consumers do not need to special-case
// declarations vs definitions — the only difference is the
// Metadata["is_declaration"] = true flag.
func emitFunctionDeclaration(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isCpp bool) {
	info := clike.ExtractFunctionInfo(node, sourceCode)
	if info == nil || info.Name == "" {
		return
	}
	storageClasses := collectStorageClassSpecifiers(node, sourceCode)
	metadata := map[string]any{metaIsDeclaration: true}
	if len(storageClasses) > 0 {
		metadata[metaStorageClasses] = storageClasses
	}

	graph.AddNode(&Node{
		ID:                   GenerateMethodID("function:"+info.Name, info.ParamTypes, file, info.LineNumber),
		Type:                 nodeTypeFunctionDefinition,
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.ParamTypes,
		MethodArgumentsValue: info.ParamNames,
		Modifier:             strings.Join(storageClasses, " "),
		File:                 file,
		Language:             languageOfFile(isCpp),
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	})
}

// =============================================================================
// Call expressions
// =============================================================================

// parseCCallExpression records a `call_expression`. The shape (free
// function vs method-dot vs method-arrow vs qualified) is determined by
// clike.ExtractCallInfo and stored alongside the target so downstream
// rule writers can match either on call shape or on the target name.
//
// currentContext links the call to its enclosing function via an edge so
// the call-graph builder (PR-07) can follow callers→callees without a
// second AST pass.
func parseCCallExpression(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, currentContext *Node) {
	info := clike.ExtractCallInfo(node, sourceCode)
	if info == nil {
		return
	}

	metadata := map[string]any{}
	if info.IsMethod {
		metadata[metaIsMethod] = true
	}
	if info.IsArrow {
		metadata[metaIsArrow] = true
	}
	if info.IsQualified {
		metadata[metaIsQualified] = true
	}
	if info.Receiver != "" {
		metadata[metaReceiver] = info.Receiver
	}

	callNode := &Node{
		ID:                   GenerateSha256("call:" + info.Target + "@" + file + "#" + lineRange(node)),
		Type:                 nodeTypeCallExpression,
		Name:                 info.Target,
		MethodArgumentsValue: info.Args,
		LineNumber:           node.StartPoint().Row + 1,
		File:                 file,
		Language:             languageC,
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	}
	graph.AddNode(callNode)
	if currentContext != nil {
		graph.AddEdge(currentContext, callNode)
	}
}

// =============================================================================
// Preprocessor includes (shared with C++ via isCpp flag)
// =============================================================================

// parseCLikeInclude records a `preproc_include` directive. Angle-bracket
// includes (`<stdio.h>`) are flagged as system includes via
// Metadata["system_include"] = true; quoted includes (`"myheader.h"`)
// are project-local. The header path is stored in Name with surrounding
// quotes/brackets stripped so resolvers can match on the bare path.
func parseCLikeInclude(node *sitter.Node, sourceCode []byte, graph *CodeGraph, file string, isCpp bool) {
	pathNode := node.ChildByFieldName("path")
	if pathNode == nil {
		return
	}

	rawPath := pathNode.Content(sourceCode)
	headerPath, isSystem := normaliseIncludePath(pathNode.Type(), rawPath)
	if headerPath == "" {
		return
	}

	graph.AddNode(&Node{
		ID:             GenerateSha256("include:" + headerPath + "@" + file + "#" + lineRange(node)),
		Type:           nodeTypeIncludeStatement,
		Name:           headerPath,
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageOfFile(isCpp),
		SourceLocation: newSourceLocation(file, node),
		Metadata:       map[string]any{metaSystemInclude: isSystem},
	})
}

// =============================================================================
// Internal helpers
// =============================================================================

// collectStorageClassSpecifiers returns the storage_class_specifier siblings
// of node (typically "static", "inline", "extern", "_Noreturn",
// "_Thread_local"). Order is preserved from source.
func collectStorageClassSpecifiers(node *sitter.Node, sourceCode []byte) []string {
	var classes []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type() == "storage_class_specifier" {
			classes = append(classes, strings.TrimSpace(child.Content(sourceCode)))
		}
	}
	return classes
}

// childDeclarators returns every direct child of node whose field name is
// "declarator". tree-sitter's stdlib ChildByFieldName returns the *first*
// match only, but several C/C++ constructs (declaration with multiple
// init_declarators, type_definition with multiple alias names) repeat the
// same field — this helper iterates the full child list and yields all
// of them in order.
func childDeclarators(node *sitter.Node) []*sitter.Node {
	var matches []*sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		if node.FieldNameForChild(i) == "declarator" {
			if c := node.Child(i); c != nil {
				matches = append(matches, c)
			}
		}
	}
	return matches
}

// bareIdentifierName unwraps a declarator chain (init_declarator,
// pointer_declarator, reference_declarator, array_declarator) to return the
// inner identifier or field_identifier name. Returns "" when no identifier
// is present (e.g. abstract declarators).
func bareIdentifierName(declarator *sitter.Node, sourceCode []byte) string {
	cur := declarator
	for cur != nil {
		switch cur.Type() {
		case "identifier", "field_identifier", "type_identifier", "primitive_type":
			return cur.Content(sourceCode)
		case "init_declarator":
			cur = cur.ChildByFieldName("declarator")
			continue
		case "pointer_declarator", "reference_declarator", "array_declarator":
			cur = cur.ChildByFieldName("declarator")
			continue
		}
		// Unrecognised wrapper — try the field-named "declarator" child if
		// present, otherwise stop walking.
		if next := cur.ChildByFieldName("declarator"); next != nil && !next.Equal(cur) {
			cur = next
			continue
		}
		return ""
	}
	return ""
}

// bareIdentifierAndInitialiser pulls the variable name and the initialiser
// expression text (when present) out of an init_declarator / declarator
// chain. The initialiser is the source text of the node held in the
// init_declarator's "value" field.
func bareIdentifierAndInitialiser(declarator *sitter.Node, sourceCode []byte) (string, string) {
	if declarator == nil {
		return "", ""
	}
	if declarator.Type() == "init_declarator" {
		nameNode := declarator.ChildByFieldName("declarator")
		valueNode := declarator.ChildByFieldName("value")
		name := bareIdentifierName(nameNode, sourceCode)
		value := ""
		if valueNode != nil {
			value = strings.TrimSpace(valueNode.Content(sourceCode))
		}
		return name, value
	}
	return bareIdentifierName(declarator, sourceCode), ""
}

// extractTaggedName returns the tag name on a struct_specifier or
// enum_specifier, plus a flag indicating whether the construct is anonymous
// (no name child).
func extractTaggedName(node *sitter.Node, sourceCode []byte) (string, bool) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return "", true
	}
	return nameNode.Content(sourceCode), false
}

// extractEnumerators reads an enumerator_list and returns one entry per
// enumerator. Entries with explicit values are formatted as "NAME=VALUE";
// entries without a value are just "NAME". Returns nil when body is nil.
func extractEnumerators(body *sitter.Node, sourceCode []byte) []string {
	if body == nil {
		return nil
	}
	var values []string
	for i := 0; i < int(body.NamedChildCount()); i++ {
		child := body.NamedChild(i)
		if child == nil || child.Type() != "enumerator" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		entry := nameNode.Content(sourceCode)
		if valueNode := child.ChildByFieldName("value"); valueNode != nil {
			entry = entry + "=" + strings.TrimSpace(valueNode.Content(sourceCode))
		}
		values = append(values, entry)
	}
	return values
}

// normaliseIncludePath strips the surrounding `<>` or `""` from an include
// path and returns the bare path plus a flag indicating whether the
// directive used angle brackets (system include).
func normaliseIncludePath(pathNodeType, rawPath string) (string, bool) {
	switch pathNodeType {
	case "system_lib_string":
		return strings.Trim(rawPath, "<>"), true
	case "string_literal":
		return strings.Trim(rawPath, `"`), false
	}
	// Defensive fallback — strip both shapes.
	return strings.Trim(rawPath, `<>"`), strings.HasPrefix(rawPath, "<")
}

// scopeFromContext returns the enclosing function's name when currentContext
// is a C/C++ function definition, or "global" at translation-unit scope.
func scopeFromContext(currentContext *Node) string {
	if currentContext != nil &&
		currentContext.Type == nodeTypeFunctionDefinition &&
		(currentContext.Language == languageC || currentContext.Language == languageCpp) {
		return currentContext.Name
	}
	return "global"
}

// languageOfFile returns "cpp" when isCpp is true, otherwise "c".
func languageOfFile(isCpp bool) string {
	if isCpp {
		return languageCpp
	}
	return languageC
}

// lineRange returns a "start-end" string used to disambiguate IDs for nodes
// that share a name within a translation unit (e.g. anonymous enums in
// different scopes, multiple typedefs of the same alias).
func lineRange(node *sitter.Node) string {
	start := node.StartPoint().Row + 1
	end := node.EndPoint().Row + 1
	return strings.TrimSuffix(joinUint(start)+"-"+joinUint(end), "-")
}

// joinUint formats a tree-sitter row number for use in ID strings.
func joinUint(v uint32) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	buf := [11]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = digits[v%10]
		v /= 10
	}
	return string(buf[i:])
}

// newSourceLocation builds a SourceLocation that lazy-loads the original
// source from file using the byte range of node.
func newSourceLocation(file string, node *sitter.Node) *SourceLocation {
	return &SourceLocation{
		File:      file,
		StartByte: node.StartByte(),
		EndByte:   node.EndByte(),
	}
}
