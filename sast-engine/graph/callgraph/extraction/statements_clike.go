package extraction

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
)

// AST node-type constants emitted by the tree-sitter C and C++ grammars.
// Centralised here so the extractors do not pepper string literals
// across files; renaming a grammar node only touches this list.
const (
	clikeNodeIdentifier         = "identifier"
	clikeNodeFieldIdentifier    = "field_identifier"
	clikeNodeTypeIdentifier     = "type_identifier"
	clikeNodeFieldExpression    = "field_expression"
	clikeNodeQualifiedIdentifier = "qualified_identifier"
	clikeNodeCallExpression     = "call_expression"
	clikeNodeAssignmentExpr     = "assignment_expression"
	clikeNodeInitDeclarator     = "init_declarator"
	clikeNodePointerDeclarator  = "pointer_declarator"
	clikeNodeArrayDeclarator    = "array_declarator"
	clikeNodeReferenceDeclarator = "reference_declarator"
	clikeNodeParenthesised      = "parenthesized_expression"
	clikeNodeSubscriptExpr      = "subscript_expression"
	clikeNodeNumberLiteral      = "number_literal"
	clikeNodeStringLiteral      = "string_literal"
	clikeNodeCharLiteral        = "char_literal"
	clikeNodeTrueFalse          = "true"
	clikeNodeFalse              = "false"
	clikeNodeNullLiteral        = "null"
)

// keywordPredicate decides whether name should be filtered out of Uses.
// The C extractor passes `clike.IsCKeyword`; the C++ extractor passes
// `clike.IsCppKeyword`. A small adapter type keeps the dispatcher
// independent of the language-specific keyword maps.
type keywordPredicate func(string) bool

// clikeExtractor is the shared core of the C / C++ statement
// extractors. The dispatcher routes function-body children to typed
// handlers; each handler builds a *core.Statement and appends it.
//
// Language differences (extra node types, different keyword filter)
// live behind the `isKeyword` predicate and the `extraNodeHandler`
// hook so the C++ extractor can extend the dispatch table without
// duplicating the C body.
type clikeExtractor struct {
	filePath  string
	src       []byte
	isKeyword keywordPredicate
	// extraNodeHandler is consulted before the default `nil` return
	// when a node type is not in the shared dispatch table. It returns
	// (stmts, true) when it handled the node; otherwise (nil, false).
	extraNodeHandler func(node *sitter.Node) ([]*core.Statement, bool)
}

// extractFunctionBody runs the dispatcher over every named child of a
// function's body field. Forward declarations (no body) yield nil.
// Callers are guaranteed non-nil by the public Extract* entry points.
func (e *clikeExtractor) extractFunctionBody(functionNode *sitter.Node) []*core.Statement {
	body := functionNode.ChildByFieldName("body")
	if body == nil {
		return nil
	}
	return e.extractBlock(body)
}

// extractBlock walks every named child of a compound block and routes
// each to the dispatch table. block is guaranteed non-nil by callers
// (entry points and dispatcher both null-check).
func (e *clikeExtractor) extractBlock(block *sitter.Node) []*core.Statement {
	var stmts []*core.Statement
	for i := 0; i < int(block.NamedChildCount()); i++ {
		stmts = append(stmts, e.extractStatement(block.NamedChild(i))...)
	}
	return stmts
}

// extractStatement dispatches on node.Type(). Unknown types fall
// through to the language-specific extra handler so C++ can register
// throw/try/range-for without forking the function. The single nil
// guard here is the only one needed because every internal recursion
// passes through this function.
func (e *clikeExtractor) extractStatement(node *sitter.Node) []*core.Statement {
	if node == nil {
		return nil
	}
	switch node.Type() {
	case "declaration":
		return e.declarationStmt(node)
	case "expression_statement":
		return e.expressionStmt(node)
	case "return_statement":
		return e.returnStmt(node)
	case "if_statement":
		return []*core.Statement{e.ifStmt(node)}
	case "for_statement":
		return []*core.Statement{e.forStmt(node)}
	case "while_statement":
		return []*core.Statement{e.whileStmt(node)}
	case "do_statement":
		return []*core.Statement{e.doStmt(node)}
	case "switch_statement":
		return []*core.Statement{e.switchStmt(node)}
	case "compound_statement":
		return e.extractBlock(node)
	case "else_clause":
		// `else` wraps a single statement (compound or otherwise);
		// route through to extractStatement so the body's children
		// surface as flat NestedStatements / ElseBranch entries.
		var stmts []*core.Statement
		for i := 0; i < int(node.NamedChildCount()); i++ {
			stmts = append(stmts, e.extractStatement(node.NamedChild(i))...)
		}
		return stmts
	case "case_statement":
		// `switch` body children include case_statement nodes that
		// wrap their bodies; flatten so the switch's NestedStatements
		// reads as a list of underlying statements.
		var stmts []*core.Statement
		for i := 0; i < int(node.NamedChildCount()); i++ {
			stmts = append(stmts, e.extractStatement(node.NamedChild(i))...)
		}
		return stmts
	}
	if e.extraNodeHandler != nil {
		if stmts, handled := e.extraNodeHandler(node); handled {
			return stmts
		}
	}
	return nil
}

// =============================================================================
// Declaration handler
// =============================================================================

// declarationStmt emits one assignment per init_declarator. A bare
// declaration (`int x;` with no initialiser) still produces an
// assignment statement with empty Uses so downstream analysis can see
// the def site.
func (e *clikeExtractor) declarationStmt(node *sitter.Node) []*core.Statement {
	var stmts []*core.Statement
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Type() != clikeNodeInitDeclarator {
			continue
		}
		stmt := e.initDeclaratorStmt(node, child)
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}

// initDeclaratorStmt builds one assignment Statement for an
// init_declarator. The init_declarator's `declarator` field is the
// defined name (after stripping pointer/array/reference wrappers); the
// `value` field carries the right-hand side expression.
func (e *clikeExtractor) initDeclaratorStmt(declarationNode, init *sitter.Node) *core.Statement {
	declarator := init.ChildByFieldName("declarator")
	defName := bareDeclaratorName(declarator, e.src)
	if defName == "" {
		return nil
	}
	stmt := &core.Statement{
		Type:       core.StatementTypeAssignment,
		LineNumber: declarationNode.StartPoint().Row + 1,
		Def:        defName,
	}
	if value := init.ChildByFieldName("value"); value != nil {
		e.populateRHS(stmt, value)
	}
	return stmt
}

// =============================================================================
// Expression-statement handler
// =============================================================================

// expressionStmt extracts a Statement from an `expression_statement`
// wrapper. The interesting cases are assignment_expression and
// call_expression; everything else falls back to a generic expression
// statement with all identifiers in Uses.
func (e *clikeExtractor) expressionStmt(node *sitter.Node) []*core.Statement {
	inner := firstNamedChild(node)
	if inner == nil {
		return nil
	}
	switch inner.Type() {
	case clikeNodeAssignmentExpr:
		return []*core.Statement{e.assignmentStmt(node, inner)}
	case clikeNodeCallExpression:
		return []*core.Statement{e.callStmt(node, inner)}
	}
	return []*core.Statement{{
		Type:       core.StatementTypeExpression,
		LineNumber: node.StartPoint().Row + 1,
		Uses:       e.collectIdentifiers(inner),
	}}
}

// assignmentStmt builds a Statement for `lhs = rhs;`. The LHS is
// reduced to a single name via leftHandSideName: subscript and field
// accesses collapse to the base variable, matching the def-use
// convention used elsewhere in the codebase.
func (e *clikeExtractor) assignmentStmt(stmtNode, expr *sitter.Node) *core.Statement {
	lhs := expr.ChildByFieldName("left")
	rhs := expr.ChildByFieldName("right")

	stmt := &core.Statement{
		Type:       core.StatementTypeAssignment,
		LineNumber: stmtNode.StartPoint().Row + 1,
		Def:        leftHandSideName(lhs, e.src),
	}
	if extras := lhsIndexUses(lhs, e); len(extras) > 0 {
		stmt.Uses = mergeUnique(stmt.Uses, extras)
	}
	if rhs != nil {
		e.populateRHS(stmt, rhs)
	}
	return stmt
}

// callStmt builds a Statement for a bare `func(args);` expression.
// Both the receiver (for `obj.method()`) and the arguments contribute
// to Uses.
func (e *clikeExtractor) callStmt(stmtNode, call *sitter.Node) *core.Statement {
	target, callChain := e.callTarget(call)
	stmt := &core.Statement{
		Type:       core.StatementTypeCall,
		LineNumber: stmtNode.StartPoint().Row + 1,
		CallTarget: target,
		CallChain:  callChain,
	}
	stmt.Uses = e.collectCallUses(call)
	stmt.CallArgs = e.collectCallArgs(call)
	return stmt
}

// =============================================================================
// Right-hand-side population
// =============================================================================

// populateRHS fills Uses / CallTarget / CallArgs on stmt from a
// right-hand-side expression. When the RHS is a call, the call target
// is recorded so downstream analysis can follow the edge; the
// receiver of a method call also contributes to Uses.
func (e *clikeExtractor) populateRHS(stmt *core.Statement, rhs *sitter.Node) {
	if rhs.Type() == clikeNodeCallExpression {
		target, chain := e.callTarget(rhs)
		stmt.CallTarget = target
		stmt.CallChain = chain
		stmt.CallArgs = e.collectCallArgs(rhs)
		stmt.Uses = mergeUnique(stmt.Uses, e.collectCallUses(rhs))
		return
	}
	stmt.Uses = mergeUnique(stmt.Uses, e.collectIdentifiers(rhs))
}

// =============================================================================
// Control flow handlers
// =============================================================================

// ifStmt emits `if (cond) { then } else { else }` as a single
// Statement carrying the condition's identifiers in Uses and both
// branches' statements in NestedStatements / ElseBranch.
func (e *clikeExtractor) ifStmt(node *sitter.Node) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeIf,
		LineNumber: node.StartPoint().Row + 1,
	}
	if cond := node.ChildByFieldName("condition"); cond != nil {
		stmt.Uses = e.collectIdentifiers(cond)
	}
	if cons := node.ChildByFieldName("consequence"); cons != nil {
		stmt.NestedStatements = e.extractStatement(cons)
	}
	if alt := node.ChildByFieldName("alternative"); alt != nil {
		stmt.ElseBranch = e.extractStatement(alt)
	}
	return stmt
}

// forStmt handles the C-style `for (init; cond; update) { body }`.
// The init clause's defined variable becomes Def; identifiers from
// cond and update collapse into Uses.
func (e *clikeExtractor) forStmt(node *sitter.Node) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeFor,
		LineNumber: node.StartPoint().Row + 1,
	}
	if init := node.ChildByFieldName("initializer"); init != nil {
		stmt.Def = forInitDef(init, e.src)
		stmt.Uses = mergeUnique(stmt.Uses, e.collectIdentifiers(init))
	}
	if cond := node.ChildByFieldName("condition"); cond != nil {
		stmt.Uses = mergeUnique(stmt.Uses, e.collectIdentifiers(cond))
	}
	if update := node.ChildByFieldName("update"); update != nil {
		stmt.Uses = mergeUnique(stmt.Uses, e.collectIdentifiers(update))
	}
	if body := node.ChildByFieldName("body"); body != nil {
		stmt.NestedStatements = e.extractStatement(body)
	}
	// The defined loop variable participates in every clause; drop it
	// from Uses last so the per-clause merges above remain simple.
	if stmt.Def != "" {
		stmt.Uses = removeName(stmt.Uses, stmt.Def)
	}
	return stmt
}

// whileStmt handles `while (cond) { body }`.
func (e *clikeExtractor) whileStmt(node *sitter.Node) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeWhile,
		LineNumber: node.StartPoint().Row + 1,
	}
	if cond := node.ChildByFieldName("condition"); cond != nil {
		stmt.Uses = e.collectIdentifiers(cond)
	}
	if body := node.ChildByFieldName("body"); body != nil {
		stmt.NestedStatements = e.extractStatement(body)
	}
	return stmt
}

// doStmt handles `do { body } while (cond);` — same shape as
// whileStmt with the condition placed after the body.
func (e *clikeExtractor) doStmt(node *sitter.Node) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeWhile,
		LineNumber: node.StartPoint().Row + 1,
	}
	if cond := node.ChildByFieldName("condition"); cond != nil {
		stmt.Uses = e.collectIdentifiers(cond)
	}
	if body := node.ChildByFieldName("body"); body != nil {
		stmt.NestedStatements = e.extractStatement(body)
	}
	return stmt
}

// switchStmt handles `switch (cond) { case ... }`. The body is a
// compound block whose children are case labels and their statements;
// we inline both into NestedStatements so flow analysis sees a flat
// list per branch.
func (e *clikeExtractor) switchStmt(node *sitter.Node) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeIf,
		LineNumber: node.StartPoint().Row + 1,
	}
	if cond := node.ChildByFieldName("condition"); cond != nil {
		stmt.Uses = e.collectIdentifiers(cond)
	}
	if body := node.ChildByFieldName("body"); body != nil {
		stmt.NestedStatements = e.extractStatement(body)
	}
	return stmt
}

// returnStmt handles `return [expr];`.
func (e *clikeExtractor) returnStmt(node *sitter.Node) []*core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeReturn,
		LineNumber: node.StartPoint().Row + 1,
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		stmt.Uses = mergeUnique(stmt.Uses, e.collectIdentifiers(node.NamedChild(i)))
	}
	return []*core.Statement{stmt}
}

// =============================================================================
// Identifier collection
// =============================================================================

// collectIdentifiers returns every variable-like identifier reachable
// from node, deduplicated and filtered through e.isKeyword. Field
// names (`obj.field`) and the right-hand component of qualified
// identifiers (`ns::name`) are skipped; the LHS of those expressions
// participates as a use because it is the receiver / namespace value.
func (e *clikeExtractor) collectIdentifiers(node *sitter.Node) []string {
	if node == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []string

	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		switch n.Type() {
		case clikeNodeFieldIdentifier, clikeNodeTypeIdentifier:
			return
		case clikeNodeIdentifier:
			name := n.Content(e.src)
			if !e.isKeyword(name) && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
			return
		case clikeNodeFieldExpression:
			// Receiver only; field name is not a use.
			if recv := n.ChildByFieldName("argument"); recv != nil {
				visit(recv)
			}
			return
		case clikeNodeQualifiedIdentifier:
			// `ns::name` references — if used as a value it shouldn't
			// register either side as a variable use.
			return
		case clikeNodeNumberLiteral, clikeNodeStringLiteral, clikeNodeCharLiteral,
			clikeNodeTrueFalse, clikeNodeFalse, clikeNodeNullLiteral:
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			visit(n.Child(i))
		}
	}
	visit(node)
	return out
}

// collectCallUses returns the receiver and argument identifiers of a
// call_expression. The function name itself is intentionally skipped
// so it appears in CallTarget but not Uses.
func (e *clikeExtractor) collectCallUses(call *sitter.Node) []string {
	var uses []string
	if fn := call.ChildByFieldName("function"); fn != nil {
		if fn.Type() == clikeNodeFieldExpression {
			if recv := fn.ChildByFieldName("argument"); recv != nil {
				uses = mergeUnique(uses, e.collectIdentifiers(recv))
			}
		}
	}
	if argList := call.ChildByFieldName("arguments"); argList != nil {
		for i := 0; i < int(argList.NamedChildCount()); i++ {
			uses = mergeUnique(uses, e.collectIdentifiers(argList.NamedChild(i)))
		}
	}
	return uses
}

// collectCallArgs returns the raw text of every argument to a
// call_expression, in source order. Stored separately from Uses so
// downstream consumers can see literals (`"hello"`, `42`) too.
func (e *clikeExtractor) collectCallArgs(call *sitter.Node) []string {
	argList := call.ChildByFieldName("arguments")
	if argList == nil {
		return nil
	}
	args := make([]string, 0, argList.NamedChildCount())
	for i := 0; i < int(argList.NamedChildCount()); i++ {
		if arg := argList.NamedChild(i); arg != nil {
			args = append(args, arg.Content(e.src))
		}
	}
	return args
}

// callTarget returns (callee, callChain) for a call_expression. The
// callee is the bare function name for free / qualified calls and the
// method name for `obj.method()`. The chain is the full dotted form
// (`obj.method`, `ns::func`) so later analysis can match patterns
// without re-parsing.
func (e *clikeExtractor) callTarget(call *sitter.Node) (string, string) {
	fn := call.ChildByFieldName("function")
	if fn == nil {
		return "", ""
	}
	switch fn.Type() {
	case clikeNodeIdentifier:
		name := fn.Content(e.src)
		return name, name
	case clikeNodeFieldExpression:
		method := ""
		if field := fn.ChildByFieldName("field"); field != nil {
			method = field.Content(e.src)
		}
		chain := strings.TrimSpace(fn.Content(e.src))
		return method, chain
	case clikeNodeQualifiedIdentifier:
		qualified := strings.TrimSpace(fn.Content(e.src))
		return qualified, qualified
	}
	return strings.TrimSpace(fn.Content(e.src)), strings.TrimSpace(fn.Content(e.src))
}

// =============================================================================
// Small AST helpers
// =============================================================================

// firstNamedChild returns the first named child of node, or nil when
// node has none.
func firstNamedChild(node *sitter.Node) *sitter.Node {
	if node == nil || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(0)
}

// bareDeclaratorName unwraps pointer / array / reference / function /
// parenthesised declarators down to the underlying identifier and
// returns its source text. Returns "" when no identifier is reachable.
func bareDeclaratorName(node *sitter.Node, src []byte) string {
	for node != nil {
		switch node.Type() {
		case clikeNodeIdentifier, clikeNodeFieldIdentifier:
			return node.Content(src)
		case clikeNodePointerDeclarator, clikeNodeArrayDeclarator,
			clikeNodeReferenceDeclarator, clikeNodeParenthesised,
			"function_declarator":
			next := node.ChildByFieldName("declarator")
			if next == nil {
				next = firstNamedChild(node)
			}
			node = next
		default:
			return strings.TrimSpace(node.Content(src))
		}
	}
	return ""
}

// leftHandSideName returns the variable being assigned to in an
// assignment expression. For `buf[i] = ...`, returns "buf"; for
// `p->name = ...`, returns "p"; for `obj.field = ...`, returns "obj".
// The caller uses `lhsIndexUses` to capture the index/field components
// as Uses.
func leftHandSideName(node *sitter.Node, src []byte) string {
	for node != nil {
		switch node.Type() {
		case clikeNodeIdentifier:
			return node.Content(src)
		case clikeNodeSubscriptExpr:
			if base := node.ChildByFieldName("argument"); base != nil {
				node = base
				continue
			}
		case clikeNodeFieldExpression:
			if recv := node.ChildByFieldName("argument"); recv != nil {
				node = recv
				continue
			}
		case clikeNodeParenthesised:
			node = firstNamedChild(node)
			continue
		}
		return strings.TrimSpace(node.Content(src))
	}
	return ""
}

// lhsIndexUses returns identifier uses that appear in the indexing
// path of a subscript or pointer-arrow LHS. For `buf[i] = ...`, it
// returns ["i"]; for `p->name = ...`, nothing extra; for plain
// identifier LHS, nothing extra.
func lhsIndexUses(node *sitter.Node, e *clikeExtractor) []string {
	for node != nil {
		switch node.Type() {
		case clikeNodeSubscriptExpr:
			if idx := node.ChildByFieldName("index"); idx != nil {
				return e.collectIdentifiers(idx)
			}
			return nil
		case clikeNodeFieldExpression:
			if recv := node.ChildByFieldName("argument"); recv != nil {
				node = recv
				continue
			}
		case clikeNodeParenthesised:
			node = firstNamedChild(node)
			continue
		}
		return nil
	}
	return nil
}

// forInitDef returns the variable defined by a C `for` initializer
// clause. Handles both forms:
//
//	for (int i = 0; ...) — declaration with init_declarator.
//	for (i = 0; ...)     — assignment_expression.
func forInitDef(node *sitter.Node, src []byte) string {
	if node == nil {
		return ""
	}
	switch node.Type() {
	case "declaration":
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child != nil && child.Type() == clikeNodeInitDeclarator {
				if d := child.ChildByFieldName("declarator"); d != nil {
					return bareDeclaratorName(d, src)
				}
			}
		}
	case clikeNodeAssignmentExpr:
		if lhs := node.ChildByFieldName("left"); lhs != nil {
			return leftHandSideName(lhs, src)
		}
	case "expression":
		// Some grammars wrap the assignment in an `expression` node.
		return forInitDef(firstNamedChild(node), src)
	}
	return ""
}

// =============================================================================
// Generic slice helpers
// =============================================================================

// mergeUnique appends every element of extra that is not already in
// dst. Order from dst is preserved; new entries arrive in extra's
// order.
func mergeUnique(dst, extra []string) []string {
	if len(extra) == 0 {
		return dst
	}
	seen := make(map[string]bool, len(dst))
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range extra {
		if seen[v] {
			continue
		}
		seen[v] = true
		dst = append(dst, v)
	}
	return dst
}

// removeName returns names with name removed (first match only).
func removeName(names []string, name string) []string {
	for i, v := range names {
		if v == name {
			return append(names[:i], names[i+1:]...)
		}
	}
	return names
}

// _ enforces that clike.IsCKeyword satisfies keywordPredicate at compile
// time so future changes to clike's API surface are caught here rather
// than in a test.
var _ keywordPredicate = clike.IsCKeyword
