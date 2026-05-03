package extraction

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
)

// AST node-type constants emitted by the C++ tree-sitter grammar that
// are not present in the C grammar. Centralised here so the dispatcher
// remains the only place that touches the literal strings.
const (
	cppNodeThrowStatement = "throw_statement"
	cppNodeTryStatement   = "try_statement"
	cppNodeCatchClause    = "catch_clause"
	cppNodeForRangeLoop   = "for_range_loop"
)

// ExtractCppStatements walks a C++ function body and produces one
// *core.Statement per recognised construct.
//
// The C and C++ extractors share every dispatcher; the C++ wrapper
// adds three extra node types via the `extraNodeHandler` hook:
//
//   - throw_statement  → StatementTypeRaise (with optional CallTarget
//     for `throw std::runtime_error("...")`)
//   - try_statement    → StatementTypeTry with the body in
//     NestedStatements and each catch clause flattened into
//     ElseBranch.
//   - for_range_loop   → StatementTypeFor capturing the loop
//     variable as Def and the iterable expression as Uses.
//
// The keyword filter is `clike.IsCppKeyword`, which inherits all C
// keywords and adds `class`, `new`, `this`, `static_cast`, etc. so
// they never appear in Uses.
func ExtractCppStatements(filePath string, sourceCode []byte, functionNode *sitter.Node) ([]*core.Statement, error) {
	if functionNode == nil {
		return nil, nil
	}
	var e *clikeExtractor
	e = &clikeExtractor{
		filePath:  filePath,
		src:       sourceCode,
		isKeyword: clike.IsCppKeyword,
	}
	e.extraNodeHandler = func(node *sitter.Node) ([]*core.Statement, bool) {
		switch node.Type() {
		case cppNodeThrowStatement:
			return []*core.Statement{cppThrowStmt(node, e)}, true
		case cppNodeTryStatement:
			return []*core.Statement{cppTryStmt(node, e)}, true
		case cppNodeForRangeLoop:
			return []*core.Statement{cppForRangeStmt(node, e)}, true
		}
		return nil, false
	}
	return e.extractFunctionBody(functionNode), nil
}

// cppThrowStmt handles `throw expr;`. When the thrown expression is a
// constructor call (`throw std::runtime_error("msg")`), the call's
// target is recorded so taint analysis can follow the edge.
func cppThrowStmt(node *sitter.Node, e *clikeExtractor) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeRaise,
		LineNumber: node.StartPoint().Row + 1,
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == clikeNodeCallExpression {
			target, chain := e.callTarget(child)
			stmt.CallTarget = target
			stmt.CallChain = chain
			stmt.CallArgs = e.collectCallArgs(child)
			stmt.Uses = mergeUnique(stmt.Uses, e.collectCallUses(child))
			continue
		}
		stmt.Uses = mergeUnique(stmt.Uses, e.collectIdentifiers(child))
	}
	return stmt
}

// cppTryStmt handles `try { body } catch (T x) { handler } ...`. Each
// catch clause's body contributes its statements to ElseBranch (in
// source order), with the caught variable filtered out of Uses by the
// keyword/identifier walker since it is a definition rather than a
// use.
func cppTryStmt(node *sitter.Node, e *clikeExtractor) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeTry,
		LineNumber: node.StartPoint().Row + 1,
	}
	if body := node.ChildByFieldName("body"); body != nil {
		stmt.NestedStatements = e.extractStatement(body)
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Type() != cppNodeCatchClause {
			continue
		}
		stmt.ElseBranch = append(stmt.ElseBranch, cppCatchStatements(child, e)...)
	}
	return stmt
}

// cppCatchStatements flattens one catch clause's body into a slice of
// statements. The clause's exception parameter (if any) is recorded
// as the Def of an empty assignment statement so def-use analysis
// sees the binding site.
func cppCatchStatements(clause *sitter.Node, e *clikeExtractor) []*core.Statement {
	var stmts []*core.Statement
	if param := clause.ChildByFieldName("parameters"); param != nil {
		if name := exceptionParamName(param, e.src); name != "" {
			stmts = append(stmts, &core.Statement{
				Type:       core.StatementTypeAssignment,
				LineNumber: clause.StartPoint().Row + 1,
				Def:        name,
			})
		}
	}
	if body := clause.ChildByFieldName("body"); body != nil {
		stmts = append(stmts, e.extractStatement(body)...)
	}
	return stmts
}

// exceptionParamName returns the bound variable in `catch (T name)`.
// The clause's parameter list contains a single parameter_declaration
// whose declarator is the variable.
func exceptionParamName(paramList *sitter.Node, src []byte) string {
	for i := 0; i < int(paramList.NamedChildCount()); i++ {
		param := paramList.NamedChild(i)
		if param == nil {
			continue
		}
		if d := param.ChildByFieldName("declarator"); d != nil {
			return bareDeclaratorName(d, src)
		}
	}
	return ""
}

// cppForRangeStmt handles range-based for loops `for (auto x : c) { body }`.
// The loop variable is captured as Def; the iterable expression
// contributes its identifiers to Uses.
func cppForRangeStmt(node *sitter.Node, e *clikeExtractor) *core.Statement {
	stmt := &core.Statement{
		Type:       core.StatementTypeFor,
		LineNumber: node.StartPoint().Row + 1,
	}
	if d := node.ChildByFieldName("declarator"); d != nil {
		stmt.Def = bareDeclaratorName(d, e.src)
	}
	if right := node.ChildByFieldName("right"); right != nil {
		stmt.Uses = e.collectIdentifiers(right)
	}
	if body := node.ChildByFieldName("body"); body != nil {
		stmt.NestedStatements = e.extractStatement(body)
	}
	if stmt.Def != "" {
		stmt.Uses = removeName(stmt.Uses, stmt.Def)
	}
	return stmt
}
