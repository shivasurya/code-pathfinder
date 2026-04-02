package cfg

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// BuildGoCFGFromAST constructs a ControlFlowGraph from a Go function AST node.
// Mirrors BuildCFGFromAST (builder.go:18) for Go tree-sitter grammar.
//
// Output types (*ControlFlowGraph, BlockStatements) are identical to the Python
// version, so taint.AnalyzeWithCFG (var_dep_graph.go:285) works unchanged.
func BuildGoCFGFromAST(
	funcFQN string,
	functionNode *sitter.Node,
	sourceCode []byte,
) (*ControlFlowGraph, BlockStatements, error) {
	if functionNode == nil {
		return nil, nil, fmt.Errorf("function node is nil")
	}

	bodyNode := functionNode.ChildByFieldName("body")
	if bodyNode == nil {
		return nil, nil, fmt.Errorf("function has no body")
	}

	cfGraph := NewControlFlowGraph(funcFQN)
	blockStmts := make(BlockStatements)

	b := &goCFGBuilder{
		funcFQN:    funcFQN,
		sourceCode: sourceCode,
		cfGraph:    cfGraph,
		blockStmts: blockStmts,
		blockSeq:   0,
	}

	firstBlockID := b.newBlockID("body")
	b.addBlock(firstBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(b.cfGraph.EntryBlockID, firstBlockID)

	lastBlockID := b.processGoBody(bodyNode, firstBlockID)

	if lastBlockID != "" {
		b.cfGraph.AddEdge(lastBlockID, b.cfGraph.ExitBlockID)
	}

	return cfGraph, blockStmts, nil
}

type goCFGBuilder struct {
	funcFQN    string
	sourceCode []byte
	cfGraph    *ControlFlowGraph
	blockStmts BlockStatements
	blockSeq   int
}

func (b *goCFGBuilder) newBlockID(label string) string {
	b.blockSeq++
	return fmt.Sprintf("%s:block_%s_%d", b.funcFQN, label, b.blockSeq)
}

func (b *goCFGBuilder) addBlock(id string, blockType BlockType) {
	block := &BasicBlock{
		ID:           id,
		Type:         blockType,
		Successors:   []string{},
		Predecessors: []string{},
		Instructions: []core.CallSite{},
	}
	b.cfGraph.AddBlock(block)
}

func (b *goCFGBuilder) appendGoStmt(blockID string, stmt *core.Statement) {
	b.blockStmts[blockID] = append(b.blockStmts[blockID], stmt)
}

// processGoBody walks children of a Go block node and returns the last block ID.
func (b *goCFGBuilder) processGoBody(blockNode *sitter.Node, currentBlockID string) string {
	for i := 0; i < int(blockNode.ChildCount()); i++ {
		stmtNode := blockNode.Child(i)
		if stmtNode == nil {
			continue
		}

		switch stmtNode.Type() {
		case "if_statement":
			currentBlockID = b.processGoIf(stmtNode, currentBlockID)

		case "for_statement":
			currentBlockID = b.processGoFor(stmtNode, currentBlockID)

		case "expression_switch_statement", "type_switch_statement":
			currentBlockID = b.processGoSwitch(stmtNode, currentBlockID)

		case "select_statement":
			currentBlockID = b.processGoSelect(stmtNode, currentBlockID)

		case "return_statement":
			stmts := b.extractGoStmts(stmtNode)
			for _, stmt := range stmts {
				b.appendGoStmt(currentBlockID, stmt)
			}
			b.cfGraph.AddEdge(currentBlockID, b.cfGraph.ExitBlockID)
			currentBlockID = b.newBlockID("after_return")
			b.addBlock(currentBlockID, BlockTypeNormal)

		default:
			stmts := b.extractGoStmts(stmtNode)
			for _, stmt := range stmts {
				b.appendGoStmt(currentBlockID, stmt)
			}
		}
	}

	return currentBlockID
}

// extractGoStmts extracts core.Statement(s) from a Go AST node.
// This is an inline extraction (not importing extraction package to avoid import cycles).
// Mirrors the logic in extraction/statements_go.go but operates within the cfg package.
func (b *goCFGBuilder) extractGoStmts(node *sitter.Node) []*core.Statement {
	if node == nil {
		return nil
	}

	line := uint32(node.StartPoint().Row + 1) //nolint:unconvert

	switch node.Type() {
	case "short_var_declaration":
		return b.extractShortVarDecl(node, line)
	case "var_declaration":
		return b.extractVarDecl(node, line)
	case "assignment_statement":
		return b.extractAssignment(node, line)
	case "expression_statement":
		for ci := 0; ci < int(node.NamedChildCount()); ci++ {
			child := node.NamedChild(ci)
			if child != nil && child.Type() == "call_expression" {
				if stmt := b.extractCall(child, line); stmt != nil {
					return []*core.Statement{stmt}
				}
			}
		}
	case "return_statement":
		return []*core.Statement{b.extractReturn(node, line)}
	case "go_statement", "defer_statement":
		for ci := 0; ci < int(node.NamedChildCount()); ci++ {
			child := node.NamedChild(ci)
			if child != nil && child.Type() == "call_expression" {
				if stmt := b.extractCall(child, line); stmt != nil {
					return []*core.Statement{stmt}
				}
			}
		}
	case "send_statement":
		stmt := &core.Statement{
			Type:       core.StatementTypeExpression,
			LineNumber: line,
			Uses:       b.collectIdentifiers(node),
		}
		return []*core.Statement{stmt}
	}
	return nil
}

func (b *goCFGBuilder) extractShortVarDecl(node *sitter.Node, line uint32) []*core.Statement {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return nil
	}

	actualRight := rightNode
	if rightNode.Type() == "expression_list" && rightNode.NamedChildCount() > 0 {
		actualRight = rightNode.NamedChild(0)
	}

	callTarget, callChain, attrAccess := "", "", ""
	var uses []string

	switch actualRight.Type() {
	case "call_expression":
		callTarget, callChain, attrAccess = b.extractCallTarget(actualRight.ChildByFieldName("function"))
		uses = b.collectArgIdentifiers(actualRight.ChildByFieldName("arguments"))
	case "selector_expression":
		attrAccess = b.selectorChain(actualRight)
		uses = b.collectIdentifiers(actualRight)
	default:
		uses = b.collectIdentifiers(actualRight)
	}

	var lhsNames []string
	if leftNode.Type() == "expression_list" {
		for i := 0; i < int(leftNode.NamedChildCount()); i++ {
			child := leftNode.NamedChild(i)
			if child != nil && child.Type() == "identifier" {
				name := child.Content(b.sourceCode)
				if name != "_" {
					lhsNames = append(lhsNames, name)
				}
			}
		}
	} else if leftNode.Type() == "identifier" {
		name := leftNode.Content(b.sourceCode)
		if name != "_" {
			lhsNames = append(lhsNames, name)
		}
	}

	var stmts []*core.Statement
	for _, name := range lhsNames {
		stmts = append(stmts, &core.Statement{
			Type:            core.StatementTypeAssignment,
			Def:             name,
			Uses:            uses,
			CallTarget:      callTarget,
			CallChain:       callChain,
			AttributeAccess: attrAccess,
			LineNumber:      line,
		})
	}
	return stmts
}

func (b *goCFGBuilder) extractVarDecl(node *sitter.Node, line uint32) []*core.Statement {
	var stmts []*core.Statement
	for i := 0; i < int(node.NamedChildCount()); i++ {
		spec := node.NamedChild(i)
		if spec == nil || spec.Type() != "var_spec" {
			continue
		}
		nameNode := spec.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		name := nameNode.Content(b.sourceCode)
		if name == "_" {
			continue
		}
		stmt := &core.Statement{Type: core.StatementTypeAssignment, Def: name, Uses: []string{}, LineNumber: line}
		valueNode := spec.ChildByFieldName("value")
		if valueNode != nil {
			actual := valueNode
			if valueNode.Type() == "expression_list" && valueNode.NamedChildCount() > 0 {
				actual = valueNode.NamedChild(0)
			}
			switch actual.Type() {
			case "call_expression":
				stmt.CallTarget, stmt.CallChain, stmt.AttributeAccess = b.extractCallTarget(actual.ChildByFieldName("function"))
				stmt.Uses = b.collectArgIdentifiers(actual.ChildByFieldName("arguments"))
			case "selector_expression":
				stmt.AttributeAccess = b.selectorChain(actual)
				stmt.Uses = b.collectIdentifiers(actual)
			default:
				stmt.Uses = b.collectIdentifiers(actual)
			}
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

func (b *goCFGBuilder) extractAssignment(node *sitter.Node, line uint32) []*core.Statement {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return nil
	}

	isAugmented := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			op := child.Content(b.sourceCode)
			if op != "=" && strings.HasSuffix(op, "=") && len(op) >= 2 {
				isAugmented = true
				break
			}
		}
	}

	actualRight := rightNode
	if rightNode.Type() == "expression_list" && rightNode.NamedChildCount() > 0 {
		actualRight = rightNode.NamedChild(0)
	}

	callTarget, callChain, attrAccess := "", "", ""
	var uses []string
	switch actualRight.Type() {
	case "call_expression":
		callTarget, callChain, attrAccess = b.extractCallTarget(actualRight.ChildByFieldName("function"))
		uses = b.collectArgIdentifiers(actualRight.ChildByFieldName("arguments"))
	case "selector_expression":
		attrAccess = b.selectorChain(actualRight)
		uses = b.collectIdentifiers(actualRight)
	default:
		uses = b.collectIdentifiers(actualRight)
	}

	var stmts []*core.Statement
	extractLHS := func(lhs *sitter.Node) {
		if lhs == nil || lhs.Type() != "identifier" {
			return
		}
		name := lhs.Content(b.sourceCode)
		if name == "_" {
			return
		}
		u := uses
		if isAugmented {
			u = append([]string{name}, uses...)
		}
		stmts = append(stmts, &core.Statement{
			Type: core.StatementTypeAssignment, Def: name, Uses: u,
			CallTarget: callTarget, CallChain: callChain, AttributeAccess: attrAccess, LineNumber: line,
		})
	}

	if leftNode.Type() == "expression_list" {
		for i := 0; i < int(leftNode.NamedChildCount()); i++ {
			extractLHS(leftNode.NamedChild(i))
		}
	} else {
		extractLHS(leftNode)
	}
	return stmts
}

func (b *goCFGBuilder) extractCall(callNode *sitter.Node, line uint32) *core.Statement {
	if callNode == nil {
		return nil
	}
	stmt := &core.Statement{Type: core.StatementTypeCall, LineNumber: line, Uses: []string{}}
	funcNode := callNode.ChildByFieldName("function")
	if funcNode != nil {
		stmt.CallTarget, stmt.CallChain, _ = b.extractCallTarget(funcNode)
	}
	argsNode := callNode.ChildByFieldName("arguments")
	if argsNode != nil {
		stmt.Uses = b.collectArgIdentifiers(argsNode)
	}
	return stmt
}

func (b *goCFGBuilder) extractReturn(node *sitter.Node, line uint32) *core.Statement {
	stmt := &core.Statement{Type: core.StatementTypeReturn, LineNumber: line, Uses: []string{}}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		actual := child
		if child.Type() == "expression_list" && child.NamedChildCount() > 0 {
			actual = child.NamedChild(0)
		}
		if actual.Type() == "call_expression" {
			f := actual.ChildByFieldName("function")
			stmt.CallTarget, stmt.CallChain, _ = b.extractCallTarget(f)
			a := actual.ChildByFieldName("arguments")
			stmt.Uses = append(stmt.Uses, b.collectArgIdentifiers(a)...)
		} else {
			stmt.Uses = append(stmt.Uses, b.collectIdentifiers(actual)...)
		}
	}
	return stmt
}

// extractCallTarget gets (bare name, dotted chain, attr prefix) from a call's function node.
func (b *goCFGBuilder) extractCallTarget(funcNode *sitter.Node) (string, string, string) {
	if funcNode == nil {
		return "", "", ""
	}
	switch funcNode.Type() {
	case "identifier":
		name := funcNode.Content(b.sourceCode)
		return name, name, ""
	case "selector_expression":
		fieldNode := funcNode.ChildByFieldName("field")
		if fieldNode != nil {
			target := fieldNode.Content(b.sourceCode)
			chain := b.selectorChain(funcNode)
			operand := funcNode.ChildByFieldName("operand")
			attr := ""
			if operand != nil && operand.Type() == "selector_expression" {
				attr = b.selectorChain(operand)
			}
			return target, chain, attr
		}
	}
	content := funcNode.Content(b.sourceCode)
	return content, content, ""
}

// selectorChain recursively builds "a.b.c" from selector_expression nodes.
func (b *goCFGBuilder) selectorChain(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	switch node.Type() {
	case "identifier", "field_identifier":
		return node.Content(b.sourceCode)
	case "selector_expression":
		operand := node.ChildByFieldName("operand")
		field := node.ChildByFieldName("field")
		if operand != nil && field != nil {
			prefix := b.selectorChain(operand)
			if prefix != "" {
				return prefix + "." + field.Content(b.sourceCode)
			}
			return field.Content(b.sourceCode)
		}
	}
	return node.Content(b.sourceCode)
}

// collectIdentifiers extracts all variable identifiers from a subtree, skipping field_identifiers.
func (b *goCFGBuilder) collectIdentifiers(node *sitter.Node) []string {
	if node == nil {
		return []string{}
	}
	seen := make(map[string]bool)
	var ids []string
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "identifier" {
			name := n.Content(b.sourceCode)
			if !seen[name] {
				seen[name] = true
				ids = append(ids, name)
			}
			return
		}
		if n.Type() == "selector_expression" {
			operand := n.ChildByFieldName("operand")
			if operand != nil {
				visit(operand)
			}
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			visit(n.Child(i))
		}
	}
	visit(node)
	return ids
}

// collectArgIdentifiers extracts identifiers from function call arguments.
func (b *goCFGBuilder) collectArgIdentifiers(argsNode *sitter.Node) []string {
	if argsNode == nil {
		return []string{}
	}
	seen := make(map[string]bool)
	var ids []string
	for i := 0; i < int(argsNode.NamedChildCount()); i++ {
		for _, id := range b.collectIdentifiers(argsNode.NamedChild(i)) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// processGoIf handles if/else statements with optional initializer.
func (b *goCFGBuilder) processGoIf(ifNode *sitter.Node, predBlockID string) string {
	condBlockID := b.newBlockID("if_cond")
	b.addBlock(condBlockID, BlockTypeConditional)
	b.cfGraph.AddEdge(predBlockID, condBlockID)

	initNode := ifNode.ChildByFieldName("initializer")
	if initNode != nil {
		for _, s := range b.extractGoStmts(initNode) {
			b.appendGoStmt(condBlockID, s)
		}
	}

	condNode := ifNode.ChildByFieldName("condition")
	if condNode != nil {
		b.appendGoStmt(condBlockID, &core.Statement{
			Type:       core.StatementTypeIf,
			LineNumber: uint32(ifNode.StartPoint().Row + 1), //nolint:unconvert
			Uses:       b.collectIdentifiers(condNode),
		})
	}

	consequenceNode := ifNode.ChildByFieldName("consequence")
	trueBlockID := b.newBlockID("if_true")
	b.addBlock(trueBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(condBlockID, trueBlockID)

	var trueEndID string
	if consequenceNode != nil {
		trueEndID = b.processGoBody(consequenceNode, trueBlockID)
	} else {
		trueEndID = trueBlockID
	}

	alternativeNode := ifNode.ChildByFieldName("alternative")
	var falseEndID string
	if alternativeNode != nil {
		falseBlockID := b.newBlockID("if_false")
		b.addBlock(falseBlockID, BlockTypeNormal)
		b.cfGraph.AddEdge(condBlockID, falseBlockID)
		falseEndID = b.processGoBody(alternativeNode, falseBlockID)
	}

	mergeBlockID := b.newBlockID("if_merge")
	b.addBlock(mergeBlockID, BlockTypeNormal)
	if trueEndID != "" {
		b.cfGraph.AddEdge(trueEndID, mergeBlockID)
	}
	if falseEndID != "" {
		b.cfGraph.AddEdge(falseEndID, mergeBlockID)
	}
	if alternativeNode == nil {
		b.cfGraph.AddEdge(condBlockID, mergeBlockID)
	}

	return mergeBlockID
}

// processGoFor handles for statements (range, C-style, bare).
func (b *goCFGBuilder) processGoFor(forNode *sitter.Node, predBlockID string) string {
	headerBlockID := b.newBlockID("for_header")
	b.addBlock(headerBlockID, BlockTypeLoop)
	b.cfGraph.AddEdge(predBlockID, headerBlockID)

	var rangeClause, forClause *sitter.Node
	for i := 0; i < int(forNode.ChildCount()); i++ {
		child := forNode.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "range_clause":
			rangeClause = child
		case "for_clause":
			forClause = child
		}
	}

	if rangeClause != nil {
		leftNode := rangeClause.ChildByFieldName("left")
		rightNode := rangeClause.ChildByFieldName("right")
		headerStmt := &core.Statement{
			Type:       core.StatementTypeFor,
			LineNumber: uint32(forNode.StartPoint().Row + 1), //nolint:unconvert
			Uses:       []string{},
		}
		if leftNode != nil {
			for j := 0; j < int(leftNode.NamedChildCount()); j++ {
				child := leftNode.NamedChild(j)
				if child != nil && child.Type() == "identifier" {
					name := child.Content(b.sourceCode)
					if name != "_" && headerStmt.Def == "" {
						headerStmt.Def = name
					}
				}
			}
		}
		if rightNode != nil {
			headerStmt.Uses = b.collectIdentifiers(rightNode)
		}
		b.appendGoStmt(headerBlockID, headerStmt)
	} else if forClause != nil {
		initNode := forClause.ChildByFieldName("initializer")
		condNode := forClause.ChildByFieldName("condition")
		if initNode != nil {
			for _, s := range b.extractGoStmts(initNode) {
				b.appendGoStmt(headerBlockID, s)
			}
		}
		if condNode != nil {
			b.appendGoStmt(headerBlockID, &core.Statement{
				Type:       core.StatementTypeFor,
				LineNumber: uint32(forNode.StartPoint().Row + 1), //nolint:unconvert
				Uses:       b.collectIdentifiers(condNode),
			})
		}
	}

	bodyNode := forNode.ChildByFieldName("body")
	bodyBlockID := b.newBlockID("for_body")
	b.addBlock(bodyBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(headerBlockID, bodyBlockID)

	var bodyEndID string
	if bodyNode != nil {
		bodyEndID = b.processGoBody(bodyNode, bodyBlockID)
	} else {
		bodyEndID = bodyBlockID
	}
	if bodyEndID != "" {
		b.cfGraph.AddEdge(bodyEndID, headerBlockID)
	}

	afterBlockID := b.newBlockID("for_after")
	b.addBlock(afterBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(headerBlockID, afterBlockID)
	return afterBlockID
}

// processGoSwitch handles expression_switch_statement and type_switch_statement.
func (b *goCFGBuilder) processGoSwitch(switchNode *sitter.Node, predBlockID string) string {
	switchBlockID := b.newBlockID("switch")
	b.addBlock(switchBlockID, BlockTypeSwitch)
	b.cfGraph.AddEdge(predBlockID, switchBlockID)

	valueNode := switchNode.ChildByFieldName("value")
	if valueNode != nil {
		b.appendGoStmt(switchBlockID, &core.Statement{
			Type:       core.StatementTypeExpression,
			LineNumber: uint32(switchNode.StartPoint().Row + 1), //nolint:unconvert
			Uses:       b.collectIdentifiers(valueNode),
		})
	}

	mergeBlockID := b.newBlockID("switch_merge")
	b.addBlock(mergeBlockID, BlockTypeNormal)

	hasDefault := false
	for i := 0; i < int(switchNode.ChildCount()); i++ {
		child := switchNode.Child(i)
		if child == nil {
			continue
		}
		ct := child.Type()
		if ct != "expression_case" && ct != "default_case" && ct != "type_case" {
			continue
		}
		if ct == "default_case" {
			hasDefault = true
		}
		caseBlockID := b.newBlockID("case")
		b.addBlock(caseBlockID, BlockTypeNormal)
		b.cfGraph.AddEdge(switchBlockID, caseBlockID)
		caseEndID := b.processGoBody(child, caseBlockID)
		if caseEndID != "" {
			b.cfGraph.AddEdge(caseEndID, mergeBlockID)
		}
	}
	if !hasDefault {
		b.cfGraph.AddEdge(switchBlockID, mergeBlockID)
	}
	return mergeBlockID
}

// processGoSelect handles select statements.
func (b *goCFGBuilder) processGoSelect(selectNode *sitter.Node, predBlockID string) string {
	selectBlockID := b.newBlockID("select")
	b.addBlock(selectBlockID, BlockTypeSwitch)
	b.cfGraph.AddEdge(predBlockID, selectBlockID)

	mergeBlockID := b.newBlockID("select_merge")
	b.addBlock(mergeBlockID, BlockTypeNormal)

	for i := 0; i < int(selectNode.ChildCount()); i++ {
		child := selectNode.Child(i)
		if child == nil {
			continue
		}
		if child.Type() != "communication_case" && child.Type() != "default_case" {
			continue
		}
		caseBlockID := b.newBlockID("comm_case")
		b.addBlock(caseBlockID, BlockTypeNormal)
		b.cfGraph.AddEdge(selectBlockID, caseBlockID)
		caseEndID := b.processGoBody(child, caseBlockID)
		if caseEndID != "" {
			b.cfGraph.AddEdge(caseEndID, mergeBlockID)
		}
	}
	return mergeBlockID
}
