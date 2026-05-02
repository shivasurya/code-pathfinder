package graph

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	sitter "github.com/smacker/go-tree-sitter"
)

// parser_cpp.go converts tree-sitter C++ AST nodes into graph.Node objects.
// Every entry point sets Language="cpp" on the produced node.
//
// C and C++ live in separate files (parser_c.go vs parser_cpp.go) so that
// language-specific concerns — class hierarchies, namespaces, templates,
// access specifiers, exception flow — stay isolated. The two parsers share:
//
//   - graph/clike  — language-neutral AST extraction (function info, type
//     strings, parameter lists, struct fields, call info)
//   - language-neutral helpers in parser_c.go (childrenByFieldName,
//     bareIdentifierName, extractTaggedName, lineRange, newSourceLocation,
//     scopeFromContext, languageOfFile, normaliseIncludePath,
//     collectStorageClassSpecifiers)
//   - parseCLikeDeclaration / parseCLikeInclude in parser_c.go for the two
//     constructs whose AST shape is identical between the two grammars
//     (variable declarations and #include directives)
//
// Within this file, function-definition / class / field / call dispatchers
// rely on currentContext to decide whether a node represents a method
// inside a class, a function inside a namespace, or a free function — the
// dispatcher in parser.go threads currentContext through buildGraphFromAST
// recursion so each handler has the surrounding scope available.

// Node.Type values produced by the C++ parser. Keep these next to the
// parser that emits them so adding new construct support touches one file
// only.
const (
	nodeTypeMethodDeclaration = "method_declaration"
	nodeTypeClassDeclaration  = "class_declaration"
	nodeTypeFieldDecl         = "field_declaration"
	nodeTypeThrowStmt         = "ThrowStmt"
	nodeTypeTryStmt           = "TryStmt"
)

// Metadata keys produced by the C++ parser. The shared keys (is_method,
// is_arrow, is_qualified, receiver) live in parser_c.go because clike
// classifies all four call shapes regardless of language.
const (
	metaTemplateParams = "template_params"
	metaCurrentAccess  = "current_access"
	metaNamespace      = "namespace"
	metaThrowExpr      = "throw_expression"
	metaCatchClauses   = "catch_clauses"
	metaIsDestructor   = "is_destructor"
	metaIsVirtual      = "is_virtual"
	metaIsPureVirtual  = "is_pure_virtual"
	metaIsOverride     = "is_override"
	metaInheritance    = "inheritance" // []string{"public Animal", "private Logger"}
)

// =============================================================================
// Class declarations
// =============================================================================

// parseCppClassSpecifier records a class_specifier and returns the class
// node so the dispatcher in parser.go can use it as currentContext for the
// recursion into the class body. Subsequent access_specifier nodes seen
// during recursion update Metadata[metaCurrentAccess] on the same map; the
// field/method handlers read that value to populate Modifier.
//
// Inheritance is captured from base_class_clause: SuperClass holds the
// first base class's bare name (matching the existing graph.Node convention),
// and Metadata["inheritance"] holds the full list with access specifiers
// (e.g. ["public Animal", "private Logger"]) so multi-inheritance rules can
// see every base.
//
// Anonymous structs/classes used as inline type expressions (e.g. inside
// `typedef struct { ... } X;`) carry empty Name and
// Metadata["is_anonymous"] = true.
func parseCppClassSpecifier(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) *Node {
	body := node.ChildByFieldName("body")
	if body == nil {
		// `class Foo` used as a forward declaration or type reference. The
		// referencing site (declaration / parameter) carries the type
		// information; we do not record an empty class node.
		return nil
	}

	name, isAnonymous := extractTaggedName(node, sourceCode)
	superClass, inheritance := extractBaseClasses(node, sourceCode)

	metadata := map[string]any{}
	if isAnonymous {
		metadata[metaIsAnonymous] = true
	}
	if len(inheritance) > 0 {
		metadata[metaInheritance] = inheritance
	}

	classNode := &Node{
		ID:             GenerateSha256("class:" + name + "@" + file + "#" + lineRange(node)),
		Type:           nodeTypeClassDeclaration,
		Name:           name,
		LineNumber:     node.StartPoint().Row + 1,
		SuperClass:     superClass,
		File:           file,
		Language:       languageCpp,
		PackageName:    packageNameFromContext(currentContext),
		SourceLocation: newSourceLocation(file, node),
		Metadata:       metadata,
	}
	g.AddNode(classNode)
	return classNode
}

// extractBaseClasses parses a class_specifier's base_class_clause and
// returns (firstBareName, inheritanceEntries). The first bare name is the
// most common access pattern (single-inheritance), exposed via
// graph.Node.SuperClass; inheritanceEntries preserves access specifiers
// and ordering for the rare but important multi-inheritance case.
func extractBaseClasses(class *sitter.Node, sourceCode []byte) (string, []string) {
	var clause *sitter.Node
	for i := 0; i < int(class.NamedChildCount()); i++ {
		c := class.NamedChild(i)
		if c != nil && c.Type() == "base_class_clause" {
			clause = c
			break
		}
	}
	if clause == nil {
		return "", nil
	}

	var firstBare string
	var entries []string
	currentAccess := ""
	for i := 0; i < int(clause.NamedChildCount()); i++ {
		child := clause.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "access_specifier":
			currentAccess = strings.TrimSpace(child.Content(sourceCode))
		case "type_identifier", "qualified_identifier", "template_type":
			bare := strings.TrimSpace(child.Content(sourceCode))
			if firstBare == "" {
				firstBare = bare
			}
			if currentAccess != "" {
				entries = append(entries, currentAccess+" "+bare)
				currentAccess = ""
			} else {
				entries = append(entries, bare)
			}
		}
	}
	return firstBare, entries
}

// =============================================================================
// Namespaces
// =============================================================================

// parseCppNamespaceDefinition records a namespace as a context node. The
// returned node is consumed by buildGraphFromAST as currentContext so every
// declaration inside the namespace body inherits PackageName. Anonymous
// namespaces (no name child) emit a node with Name="" and PackageName="" —
// rule writers can still inspect Type==namespace_definition while leaving
// FQNs unqualified.
//
// Inline namespaces (`inline namespace x { ... }`) parse as namespace_definition
// in tree-sitter; for Phase 1 we treat them identically to regular namespaces
// since their visibility-promotion semantics do not change FQN construction.
func parseCppNamespaceDefinition(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) *Node {
	name := ""
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		name = nameNode.Content(sourceCode)
	}

	// Compose nested namespace name for the contextual PackageName: an
	// inner namespace inherits the outer namespace's prefix.
	prefix := packageNameFromContext(currentContext)
	combined := name
	if prefix != "" {
		if name == "" {
			combined = prefix
		} else {
			combined = prefix + "::" + name
		}
	}

	nsNode := &Node{
		ID:             GenerateSha256("namespace:" + combined + "@" + file + "#" + lineRange(node)),
		Type:           "namespace_definition",
		Name:           name,
		LineNumber:     node.StartPoint().Row + 1,
		PackageName:    combined,
		File:           file,
		Language:       languageCpp,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       map[string]any{metaNamespace: combined},
	}
	g.AddNode(nsNode)
	return nsNode
}

// =============================================================================
// Function definitions and method declarations
// =============================================================================

// parseCppFunctionDefinition records a function_definition. When
// currentContext is a class, the produced node is a method_declaration;
// otherwise it is a function_definition. Pure virtual methods (Animal::speak
// = 0) carry Metadata[is_pure_virtual]=true; virtual methods carry
// is_virtual; override-marked methods carry is_override.
//
// The function is also reached for out-of-line method definitions
// (`void Foo::bar() { ... }` at translation-unit scope). In that case the
// declarator chain produces a qualified name like "Foo::bar" via
// clike.ExtractFunctionInfo; we keep the qualified form in Name and emit
// function_definition (not method_declaration) because the surrounding
// context is not a class node — the call-graph builder in a later PR can
// link these to their class declarations by FQN.
func parseCppFunctionDefinition(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) *Node {
	info := clike.ExtractFunctionInfo(node, sourceCode)
	if info == nil {
		return nil
	}

	storageClasses := collectStorageClassSpecifiers(node, sourceCode)
	specifiers := collectVirtualSpecifiers(node, sourceCode)
	hasPureVirtual := node.ChildByFieldName("default_value") != nil || hasChildOfType(node, "pure_virtual_clause")

	insideClass := classFromContext(currentContext)
	nodeType := nodeTypeFunctionDefinition
	modifier := strings.Join(storageClasses, " ")
	packageName := packageNameFromContext(currentContext)

	if insideClass != nil {
		nodeType = nodeTypeMethodDeclaration
		access := classAccessFromContext(insideClass)
		modifier = combineModifiers(access, storageClasses)
	}

	metadata := map[string]any{}
	if info.IsDeclaration {
		metadata[metaIsDeclaration] = true
	}
	if len(storageClasses) > 0 {
		metadata[metaStorageClasses] = storageClasses
	}
	if specifiers["virtual"] {
		metadata[metaIsVirtual] = true
	}
	if specifiers["override"] {
		metadata[metaIsOverride] = true
	}
	if hasPureVirtual {
		metadata[metaIsPureVirtual] = true
		// A pure virtual still has no body — flag it as a declaration too
		// so callers can treat the two forms uniformly.
		metadata[metaIsDeclaration] = true
	}
	if isDestructorName(info.Name) {
		metadata[metaIsDestructor] = true
	}

	functionNode := &Node{
		ID:                   GenerateMethodID("function:"+info.Name, info.ParamTypes, file, info.LineNumber),
		Type:                 nodeType,
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.ParamTypes,
		MethodArgumentsValue: info.ParamNames,
		Modifier:             modifier,
		PackageName:          packageName,
		File:                 file,
		Language:             languageCpp,
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	}
	g.AddNode(functionNode)
	return functionNode
}

// =============================================================================
// Field declarations (class data members and inline method declarations)
// =============================================================================

// parseCppFieldDeclaration handles `field_declaration` nodes inside a class
// body. tree-sitter overloads field_declaration for two distinct C++
// constructs:
//
//   - Data members:    `int x;`           → declarator is field_identifier
//   - Method decls:    `void bar();`      → declarator is function_declarator
//
// We dispatch on declarator type so each construct produces the right
// graph.Node (field_declaration vs method_declaration).
//
// Outside a class body, field_declaration is not expected for C/C++; the
// dispatcher in parser.go guards against it.
func parseCppFieldDeclaration(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) {
	insideClass := classFromContext(currentContext)
	declarator := node.ChildByFieldName("declarator")
	typeNode := node.ChildByFieldName("type")

	if isFunctionDeclarator(declarator) {
		emitMethodDeclarationFromField(node, sourceCode, g, file, insideClass)
		return
	}

	name := bareIdentifierName(declarator, sourceCode)
	if name == "" {
		return
	}

	access := classAccessFromContext(insideClass)
	dataType := clike.ExtractTypeString(typeNode, declarator, sourceCode)

	metadata := map[string]any{}
	if access != "" {
		metadata[metaCurrentAccess] = access
	}

	g.AddNode(&Node{
		ID:             GenerateSha256("field:" + scopedName(insideClass, name) + "@" + file + "#" + lineRange(node)),
		Type:           nodeTypeFieldDecl,
		Name:           name,
		DataType:       dataType,
		Modifier:       access,
		PackageName:    packageNameFromContext(insideClass),
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageCpp,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       metadata,
	})
}

// emitMethodDeclarationFromField records an inline method declaration that
// tree-sitter parses as field_declaration (e.g. `void speak() override;`
// inside a class body, no body of its own).
func emitMethodDeclarationFromField(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, insideClass *Node) {
	info := clike.ExtractFunctionInfo(node, sourceCode)
	if info == nil || info.Name == "" {
		return
	}

	access := classAccessFromContext(insideClass)
	specifiers := collectVirtualSpecifiers(node, sourceCode)
	hasPureVirtual := hasChildOfType(node, "pure_virtual_clause")

	metadata := map[string]any{
		metaIsDeclaration: true,
	}
	if specifiers["virtual"] {
		metadata[metaIsVirtual] = true
	}
	if specifiers["override"] {
		metadata[metaIsOverride] = true
	}
	if hasPureVirtual {
		metadata[metaIsPureVirtual] = true
	}
	if isDestructorName(info.Name) {
		metadata[metaIsDestructor] = true
	}

	g.AddNode(&Node{
		ID:                   GenerateMethodID("function:"+info.Name, info.ParamTypes, file, info.LineNumber),
		Type:                 nodeTypeMethodDeclaration,
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.ParamTypes,
		MethodArgumentsValue: info.ParamNames,
		Modifier:             access,
		PackageName:          packageNameFromContext(insideClass),
		File:                 file,
		Language:             languageCpp,
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	})
}

// emitCppMethodDeclarationFromDeclaration records a method declaration that
// reaches us as a top-level `declaration` node inside a class body — this
// is how tree-sitter parses destructors (`~ClassName();`). Called from
// parseCLikeDeclaration in parser_c.go when isCpp && currentContext is a
// class.
func emitCppMethodDeclarationFromDeclaration(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, insideClass *Node) {
	info := clike.ExtractFunctionInfo(node, sourceCode)
	if info == nil || info.Name == "" {
		return
	}

	access := classAccessFromContext(insideClass)
	metadata := map[string]any{metaIsDeclaration: true}
	if isDestructorName(info.Name) {
		metadata[metaIsDestructor] = true
	}

	g.AddNode(&Node{
		ID:                   GenerateMethodID("function:"+info.Name, info.ParamTypes, file, info.LineNumber),
		Type:                 nodeTypeMethodDeclaration,
		Name:                 info.Name,
		LineNumber:           info.LineNumber,
		ReturnType:           info.ReturnType,
		MethodArgumentsType:  info.ParamTypes,
		MethodArgumentsValue: info.ParamNames,
		Modifier:             access,
		PackageName:          packageNameFromContext(insideClass),
		File:                 file,
		Language:             languageCpp,
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	})
}

// =============================================================================
// Templates
// =============================================================================

// parseCppTemplateDeclaration records the template parameter list. The
// inner construct (function_definition, class_specifier, etc.) is processed
// during normal recursion; we just attach the template parameters to the
// inner node by walking forward at insertion time.
//
// The metadata is stored on the outer template_declaration node so rule
// writers can match on "templated function" without looking at the inner
// shape — that match is then refined by joining on the inner node via
// SourceLocation.
func parseCppTemplateDeclaration(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string) {
	params := extractTemplateParameters(node.ChildByFieldName("parameters"), sourceCode)

	g.AddNode(&Node{
		ID:             GenerateSha256("template:" + strings.Join(params, ",") + "@" + file + "#" + lineRange(node)),
		Type:           "template_declaration",
		Name:           strings.Join(params, ","),
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageCpp,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       map[string]any{metaTemplateParams: params},
	})
}

// extractTemplateParameters reads a template_parameter_list and returns
// each declared parameter in source order.
//
// Examples:
//
//	<typename T>             → ["T"]
//	<typename T, int N>      → ["T", "N"]
//	<class K, class V = int> → ["K", "V"]
func extractTemplateParameters(list *sitter.Node, sourceCode []byte) []string {
	if list == nil {
		return nil
	}
	var params []string
	for i := 0; i < int(list.NamedChildCount()); i++ {
		child := list.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "type_parameter_declaration", "parameter_declaration", "optional_type_parameter_declaration":
			// The parameter name is the last identifier / type_identifier child.
			name := lastNamedIdentifier(child, sourceCode)
			if name != "" {
				params = append(params, name)
			}
		}
	}
	return params
}

// =============================================================================
// Exception flow: throw and try/catch
// =============================================================================

// parseCppThrowStatement records a `throw expr;` statement. The expression
// text is captured in Metadata["throw_expression"] so flow-analysis rules
// can match on what was thrown without re-parsing the AST.
//
// `throw;` (re-throw) parses the same node type with no expression child;
// Metadata["throw_expression"] is empty in that case.
func parseCppThrowStatement(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) {
	expr := ""
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil {
			expr = strings.TrimSpace(child.Content(sourceCode))
			break
		}
	}

	throwNode := &Node{
		ID:             GenerateSha256("throw:" + expr + "@" + file + "#" + lineRange(node)),
		Type:           nodeTypeThrowStmt,
		Name:           expr,
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageCpp,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       map[string]any{metaThrowExpr: expr},
	}
	g.AddNode(throwNode)
	if currentContext != nil {
		g.AddEdge(currentContext, throwNode)
	}
}

// parseCppTryStatement records a `try { ... } catch (...) { ... }` block.
// Each catch clause's parameter type goes into Metadata["catch_clauses"]
// as a []string so rule writers can match handlers by exception type.
func parseCppTryStatement(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) {
	catches := extractCatchExceptionTypes(node, sourceCode)

	tryNode := &Node{
		ID:             GenerateSha256("try@" + file + "#" + lineRange(node)),
		Type:           nodeTypeTryStmt,
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageCpp,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       map[string]any{metaCatchClauses: catches},
	}
	g.AddNode(tryNode)
	if currentContext != nil {
		g.AddEdge(currentContext, tryNode)
	}
}

// extractCatchExceptionTypes walks every catch_clause child of a
// try_statement and returns the type string of the caught exception in
// each handler. `catch (...)` (catch-all) emits "..." as the entry.
func extractCatchExceptionTypes(try *sitter.Node, sourceCode []byte) []string {
	var types []string
	for i := 0; i < int(try.NamedChildCount()); i++ {
		child := try.NamedChild(i)
		if child == nil || child.Type() != "catch_clause" {
			continue
		}
		params := child.ChildByFieldName("parameters")
		if params == nil {
			types = append(types, "...")
			continue
		}
		// catch (...) — single anonymous param.
		paramText := strings.TrimSpace(params.Content(sourceCode))
		if paramText == "(...)" {
			types = append(types, "...")
			continue
		}
		// Single parameter — extract its type via clike.
		var first *sitter.Node
		for j := 0; j < int(params.NamedChildCount()); j++ {
			c := params.NamedChild(j)
			if c != nil && (c.Type() == "parameter_declaration" || c.Type() == "optional_parameter_declaration") {
				first = c
				break
			}
		}
		if first == nil {
			types = append(types, "...")
			continue
		}
		typeNode := first.ChildByFieldName("type")
		declarator := first.ChildByFieldName("declarator")
		types = append(types, clike.ExtractTypeString(typeNode, declarator, sourceCode))
	}
	return types
}

// =============================================================================
// Calls, structs, enums, typedefs (C++ flavour)
// =============================================================================

// parseCppCallExpression records a call_expression with Language="cpp" and
// the call-shape metadata produced by clike.ExtractCallInfo. Every call
// shape (free function, dot method, arrow method, qualified) is handled by
// the same code path because the classification is already done by clike.
func parseCppCallExpression(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string, currentContext *Node) {
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
		Language:             languageCpp,
		SourceLocation:       newSourceLocation(file, node),
		Metadata:             metadata,
	}
	g.AddNode(callNode)
	if currentContext != nil {
		g.AddEdge(currentContext, callNode)
	}
}

// parseCppStructSpecifier records a C++ struct (semantically a class with
// public default visibility). Inheritance via base_class_clause is captured
// the same way as parseCppClassSpecifier; the node Type stays
// "struct_declaration" so rules can target structs vs classes when desired.
func parseCppStructSpecifier(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string) {
	body := node.ChildByFieldName("body")
	if body == nil {
		// `struct S` used as a type reference. Skip.
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

	superClass, inheritance := extractBaseClasses(node, sourceCode)

	metadata := map[string]any{}
	if isAnonymous {
		metadata[metaIsAnonymous] = true
	}
	if len(inheritance) > 0 {
		metadata[metaInheritance] = inheritance
	}

	g.AddNode(&Node{
		ID:                  GenerateSha256("struct:" + name + "@" + file + "#" + lineRange(node)),
		Type:                nodeTypeStructDeclaration,
		Name:                name,
		LineNumber:          node.StartPoint().Row + 1,
		MethodArgumentsType: fieldStrings,
		SuperClass:          superClass,
		File:                file,
		Language:            languageCpp,
		SourceLocation:      newSourceLocation(file, node),
		Metadata:            metadata,
	})
}

// parseCppEnumSpecifier records a C++ enum. C++ adds `enum class` (scoped
// enums); when present, Metadata["is_scoped"] = true. Otherwise the
// behaviour matches parseCEnumSpecifier — we don't share the function
// because the AST shape differs (C++ has an additional `enum class` keyword
// node).
func parseCppEnumSpecifier(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}

	name, isAnonymous := extractTaggedName(node, sourceCode)
	enumerators := extractEnumerators(body, sourceCode)
	isScoped := hasChildOfType(node, "class") || hasChildOfType(node, "struct")

	metadata := map[string]any{
		metaEnumerators: enumerators,
	}
	if isAnonymous {
		metadata[metaIsAnonymous] = true
	}
	if isScoped {
		metadata["is_scoped"] = true
	}

	g.AddNode(&Node{
		ID:             GenerateSha256("enum:" + name + "@" + file + "#" + lineRange(node)),
		Type:           nodeTypeEnumDeclaration,
		Name:           name,
		LineNumber:     node.StartPoint().Row + 1,
		File:           file,
		Language:       languageCpp,
		SourceLocation: newSourceLocation(file, node),
		Metadata:       metadata,
	})
}

// parseCppTypeDefinition records a C++ typedef. C++ also has `using` alias
// declarations that parse as `alias_declaration` (handled in a future PR);
// `typedef` itself parses identically to C, so we simply re-tag the
// produced node with Language="cpp".
func parseCppTypeDefinition(node *sitter.Node, sourceCode []byte, g *CodeGraph, file string) {
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
		g.AddNode(&Node{
			ID:             GenerateSha256("typedef:" + aliasName + "@" + file + "#" + lineRange(node)),
			Type:           nodeTypeTypeDefinition,
			Name:           aliasName,
			DataType:       underlying,
			LineNumber:     node.StartPoint().Row + 1,
			File:           file,
			Language:       languageCpp,
			SourceLocation: newSourceLocation(file, node),
			Metadata:       map[string]any{metaUnderlyingType: underlying},
		})
	}
}

// =============================================================================
// Access specifier propagation
// =============================================================================

// recordAccessSpecifier updates the access-tracking state on the enclosing
// class node. tree-sitter emits access_specifier as a sibling preceding the
// fields/methods it governs; the dispatcher in parser.go calls this when
// it sees one, so subsequent parseCppFieldDeclaration / parseCppFunctionDefinition
// calls (which run on the same currentContext map) read the updated value.
func recordAccessSpecifier(node *sitter.Node, sourceCode []byte, currentContext *Node) {
	classNode := classFromContext(currentContext)
	if classNode == nil {
		return
	}
	access := strings.TrimSpace(node.Content(sourceCode))
	if classNode.Metadata == nil {
		classNode.Metadata = map[string]any{}
	}
	classNode.Metadata[metaCurrentAccess] = access
}

// =============================================================================
// Internal helpers
// =============================================================================

// classFromContext returns currentContext when it is a class_declaration
// node, otherwise nil. Used by method and field handlers to detect class
// membership.
func classFromContext(currentContext *Node) *Node {
	if currentContext != nil && currentContext.Type == nodeTypeClassDeclaration {
		return currentContext
	}
	return nil
}

// classAccessFromContext returns the access specifier currently in effect
// for the class node (the value most recently set by recordAccessSpecifier).
// Returns "" when no access specifier has been seen yet — tree-sitter
// preserves source order, so the first members of a class declared with no
// preceding access_specifier inherit the class's default ("private" for
// `class`, "public" for `struct`).
func classAccessFromContext(classNode *Node) string {
	if classNode == nil || classNode.Metadata == nil {
		return ""
	}
	if v, ok := classNode.Metadata[metaCurrentAccess].(string); ok {
		return v
	}
	return ""
}

// packageNameFromContext walks currentContext to find the enclosing
// namespace's PackageName. Both namespace and class context nodes carry
// PackageName; when neither is present, returns "".
func packageNameFromContext(currentContext *Node) string {
	if currentContext == nil {
		return ""
	}
	return currentContext.PackageName
}

// scopedName joins a class name with a member name using "::" separator.
// Used to produce stable IDs that distinguish identical member names
// across classes in the same translation unit.
func scopedName(classNode *Node, memberName string) string {
	if classNode == nil || classNode.Name == "" {
		return memberName
	}
	return classNode.Name + "::" + memberName
}

// combineModifiers joins a single access modifier ("public"/"private"/...)
// with a list of storage class specifiers ("static", "inline", ...) into
// the single Modifier string used by graph.Node. Empty inputs are skipped.
func combineModifiers(access string, storageClasses []string) string {
	parts := []string{}
	if access != "" {
		parts = append(parts, access)
	}
	parts = append(parts, storageClasses...)
	return strings.Join(parts, " ")
}

// collectVirtualSpecifiers walks the children of a function_definition or
// field_declaration looking for the specifier keywords that affect method
// dispatch: "virtual" appears as a keyword child, "override"/"final" as
// virtual_specifier nodes. Returns a presence-set keyed by the keyword.
func collectVirtualSpecifiers(node *sitter.Node, sourceCode []byte) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < int(node.ChildCount()); i++ {
		c := node.Child(i)
		if c == nil {
			continue
		}
		if c.Type() == "virtual" {
			out["virtual"] = true
			continue
		}
		if c.Type() == "virtual_specifier" {
			text := strings.TrimSpace(c.Content(sourceCode))
			out[text] = true
		}
	}
	// function_declarator inside the field/function may also carry the
	// virtual_specifier (e.g. override appears after the parameter list).
	if decl := node.ChildByFieldName("declarator"); decl != nil {
		for i := 0; i < int(decl.ChildCount()); i++ {
			c := decl.Child(i)
			if c != nil && c.Type() == "virtual_specifier" {
				text := strings.TrimSpace(c.Content(sourceCode))
				out[text] = true
			}
		}
	}
	return out
}

// hasChildOfType reports whether any direct child of node has the given type.
func hasChildOfType(node *sitter.Node, nodeType string) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		c := node.Child(i)
		if c != nil && c.Type() == nodeType {
			return true
		}
	}
	return false
}

// isFunctionDeclarator walks past pointer/reference/array wrappers to
// determine whether a declarator chain bottoms out at a function_declarator.
func isFunctionDeclarator(node *sitter.Node) bool {
	for cur := node; cur != nil; {
		switch cur.Type() {
		case "function_declarator":
			return true
		case "pointer_declarator", "reference_declarator", "array_declarator":
			cur = cur.ChildByFieldName("declarator")
			continue
		}
		return false
	}
	return false
}

// isDestructorName reports whether name is a destructor name (`~ClassName`).
// Destructors are stored as Name="~ClassName" by clike so detection is a
// single-character prefix check.
func isDestructorName(name string) bool {
	return strings.HasPrefix(name, "~")
}

// lastNamedIdentifier returns the content of the last identifier or
// type_identifier among node's named children. Used by template parameter
// extraction where the parameter name is the last token after `typename`,
// `class`, or a type expression.
func lastNamedIdentifier(node *sitter.Node, sourceCode []byte) string {
	last := ""
	for i := 0; i < int(node.NamedChildCount()); i++ {
		c := node.NamedChild(i)
		if c == nil {
			continue
		}
		if c.Type() == "identifier" || c.Type() == "type_identifier" {
			last = c.Content(sourceCode)
		}
	}
	return last
}
