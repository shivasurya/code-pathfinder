package cfg

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// BlockStatements maps block IDs to their contained statements.
type BlockStatements map[string][]*core.Statement

// BuildCFGFromAST constructs a CFG from a tree-sitter function definition node.
// It walks the function body, splitting at control flow boundaries (if/for/while/try/with)
// and extracting statements into basic blocks.
//
// Returns the CFG and a map of block ID -> statements for that block.
func BuildCFGFromAST(
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

	b := &cfgBuilder{
		funcFQN:    funcFQN,
		sourceCode: sourceCode,
		cfGraph:    cfGraph,
		blockStmts: blockStmts,
		blockSeq:   0,
	}

	// Create initial block after entry
	firstBlockID := b.newBlockID("body")
	b.addBlock(firstBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(b.cfGraph.EntryBlockID, firstBlockID)

	// Process function body; returns the ID of the last block in the sequence
	lastBlockID := b.processBody(bodyNode, firstBlockID)

	// Connect last block to exit
	if lastBlockID != "" {
		b.cfGraph.AddEdge(lastBlockID, b.cfGraph.ExitBlockID)
	}

	return cfGraph, blockStmts, nil
}

type cfgBuilder struct {
	funcFQN    string
	sourceCode []byte
	cfGraph    *ControlFlowGraph
	blockStmts BlockStatements
	blockSeq   int
}

func (b *cfgBuilder) newBlockID(label string) string {
	b.blockSeq++
	return fmt.Sprintf("%s:block_%s_%d", b.funcFQN, label, b.blockSeq)
}

func (b *cfgBuilder) addBlock(id string, blockType BlockType) {
	block := &BasicBlock{
		ID:           id,
		Type:         blockType,
		Successors:   []string{},
		Predecessors: []string{},
		Instructions: []core.CallSite{},
	}
	b.cfGraph.AddBlock(block)
}

// processBody walks children of a body/block node and returns the last block ID.
// currentBlockID is the block that sequential statements should be appended to.
func (b *cfgBuilder) processBody(bodyNode *sitter.Node, currentBlockID string) string {
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		stmtNode := bodyNode.Child(i)
		if stmtNode == nil {
			continue
		}

		actualNode := stmtNode
		if stmtNode.Type() == "expression_statement" {
			if firstChild := stmtNode.Child(0); firstChild != nil {
				actualNode = firstChild
			}
		}

		switch actualNode.Type() {
		case "if_statement":
			currentBlockID = b.processIf(actualNode, stmtNode, currentBlockID)

		case "for_statement":
			currentBlockID = b.processFor(actualNode, stmtNode, currentBlockID)

		case "while_statement":
			currentBlockID = b.processWhile(actualNode, stmtNode, currentBlockID)

		case "try_statement":
			currentBlockID = b.processTry(actualNode, stmtNode, currentBlockID)

		case "with_statement":
			currentBlockID = b.processWith(actualNode, stmtNode, currentBlockID)

		case "switch_statement":
			currentBlockID = b.processSwitch(actualNode, stmtNode, currentBlockID)

		case "do_statement":
			currentBlockID = b.processDoWhile(actualNode, stmtNode, currentBlockID)

		case "return_statement":
			stmt := b.extractStatement(actualNode, stmtNode)
			if stmt != nil {
				b.appendStmt(currentBlockID, stmt)
			}
			// Return goes directly to exit
			b.cfGraph.AddEdge(currentBlockID, b.cfGraph.ExitBlockID)
			// Create a new unreachable block for any code after return
			currentBlockID = b.newBlockID("after_return")
			b.addBlock(currentBlockID, BlockTypeNormal)

		default:
			stmt := b.extractStatement(actualNode, stmtNode)
			if stmt != nil {
				b.appendStmt(currentBlockID, stmt)
			}
		}
	}

	return currentBlockID
}

// processIf handles if/elif/else statements.
// Creates: condition block -> true branch, false branch -> merge block.
func (b *cfgBuilder) processIf(ifNode, stmtNode *sitter.Node, predBlockID string) string {
	// Create condition block
	condBlockID := b.newBlockID("if_cond")
	b.addBlock(condBlockID, BlockTypeConditional)
	b.cfGraph.AddEdge(predBlockID, condBlockID)

	// Extract condition uses as a statement
	condNode := ifNode.ChildByFieldName("condition")
	if condNode != nil {
		condStmt := &core.Statement{
			Type:       core.StatementTypeIf,
			LineNumber: stmtNode.StartPoint().Row + 1,
			Uses:       extractIdentifiers(condNode, b.sourceCode),
		}
		b.appendStmt(condBlockID, condStmt)
	}

	// Process consequence (true branch)
	consequenceNode := ifNode.ChildByFieldName("consequence")
	trueBlockID := b.newBlockID("if_true")
	b.addBlock(trueBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(condBlockID, trueBlockID)

	var trueEndID string
	if consequenceNode != nil {
		trueEndID = b.processBody(consequenceNode, trueBlockID)
	} else {
		trueEndID = trueBlockID
	}

	// Process alternative (else/elif branch)
	alternativeNode := ifNode.ChildByFieldName("alternative")
	var falseEndID string

	if alternativeNode != nil {
		// Check if it's an elif (elif_clause) or else (else_clause)
		falseBlockID := b.newBlockID("if_false")
		b.addBlock(falseBlockID, BlockTypeNormal)
		b.cfGraph.AddEdge(condBlockID, falseBlockID)

		// The alternative node might be an elif_clause or else_clause
		// Both have a body child
		altBodyNode := alternativeNode.ChildByFieldName("body")
		if altBodyNode == nil {
			// Try direct children for else_clause
			altBodyNode = alternativeNode.ChildByFieldName("consequence")
		}
		if altBodyNode != nil {
			falseEndID = b.processBody(altBodyNode, falseBlockID)
		} else {
			// Walk children directly for else blocks
			falseEndID = b.processBody(alternativeNode, falseBlockID)
		}
	} else {
		// No else branch — false path goes directly to merge
		falseEndID = condBlockID
	}

	// Create merge block
	mergeBlockID := b.newBlockID("if_merge")
	b.addBlock(mergeBlockID, BlockTypeNormal)

	if trueEndID != "" {
		b.cfGraph.AddEdge(trueEndID, mergeBlockID)
	}
	if falseEndID != "" && alternativeNode != nil {
		b.cfGraph.AddEdge(falseEndID, mergeBlockID)
	} else if alternativeNode == nil {
		// No else: false edge from condition goes to merge
		b.cfGraph.AddEdge(condBlockID, mergeBlockID)
	}

	return mergeBlockID
}

// processFor handles for-loop statements.
// Creates: loop header -> loop body -> (back edge to header), after-loop block.
func (b *cfgBuilder) processFor(forNode, stmtNode *sitter.Node, predBlockID string) string {
	headerBlockID := b.newBlockID("for_header")
	b.addBlock(headerBlockID, BlockTypeLoop)
	b.cfGraph.AddEdge(predBlockID, headerBlockID)

	// Extract loop variable as a def
	leftNode := forNode.ChildByFieldName("left")
	rightNode := forNode.ChildByFieldName("right")
	if leftNode != nil {
		headerStmt := &core.Statement{
			Type:       core.StatementTypeFor,
			LineNumber: stmtNode.StartPoint().Row + 1,
			Uses:       []string{},
		}
		if leftNode.Type() == "identifier" {
			headerStmt.Def = leftNode.Content(b.sourceCode)
		}
		if rightNode != nil {
			headerStmt.Uses = extractIdentifiers(rightNode, b.sourceCode)
		}
		b.appendStmt(headerBlockID, headerStmt)
	}

	// Process loop body
	bodyNode := forNode.ChildByFieldName("body")
	bodyBlockID := b.newBlockID("for_body")
	b.addBlock(bodyBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(headerBlockID, bodyBlockID)

	var bodyEndID string
	if bodyNode != nil {
		bodyEndID = b.processBody(bodyNode, bodyBlockID)
	} else {
		bodyEndID = bodyBlockID
	}

	// Back edge from end of body to header
	if bodyEndID != "" {
		b.cfGraph.AddEdge(bodyEndID, headerBlockID)
	}

	// After-loop block
	afterBlockID := b.newBlockID("for_after")
	b.addBlock(afterBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(headerBlockID, afterBlockID)

	return afterBlockID
}

// processWhile handles while-loop statements.
func (b *cfgBuilder) processWhile(whileNode, stmtNode *sitter.Node, predBlockID string) string {
	headerBlockID := b.newBlockID("while_header")
	b.addBlock(headerBlockID, BlockTypeLoop)
	b.cfGraph.AddEdge(predBlockID, headerBlockID)

	// Extract condition
	condNode := whileNode.ChildByFieldName("condition")
	if condNode != nil {
		condStmt := &core.Statement{
			Type:       core.StatementTypeWhile,
			LineNumber: stmtNode.StartPoint().Row + 1,
			Uses:       extractIdentifiers(condNode, b.sourceCode),
		}
		b.appendStmt(headerBlockID, condStmt)
	}

	// Process body
	bodyNode := whileNode.ChildByFieldName("body")
	bodyBlockID := b.newBlockID("while_body")
	b.addBlock(bodyBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(headerBlockID, bodyBlockID)

	var bodyEndID string
	if bodyNode != nil {
		bodyEndID = b.processBody(bodyNode, bodyBlockID)
	} else {
		bodyEndID = bodyBlockID
	}

	// Back edge
	if bodyEndID != "" {
		b.cfGraph.AddEdge(bodyEndID, headerBlockID)
	}

	// After-while block
	afterBlockID := b.newBlockID("while_after")
	b.addBlock(afterBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(headerBlockID, afterBlockID)

	return afterBlockID
}

// processTry handles try/except/finally statements.
func (b *cfgBuilder) processTry(tryNode, _ *sitter.Node, predBlockID string) string {
	// Try block
	tryBlockID := b.newBlockID("try")
	b.addBlock(tryBlockID, BlockTypeTry)
	b.cfGraph.AddEdge(predBlockID, tryBlockID)

	bodyNode := tryNode.ChildByFieldName("body")
	var tryEndID string
	if bodyNode != nil {
		tryEndID = b.processBody(bodyNode, tryBlockID)
	} else {
		tryEndID = tryBlockID
	}

	// After-try merge block
	mergeBlockID := b.newBlockID("try_merge")
	b.addBlock(mergeBlockID, BlockTypeNormal)

	if tryEndID != "" {
		b.cfGraph.AddEdge(tryEndID, mergeBlockID)
	}

	// Process except_clause children
	for i := 0; i < int(tryNode.ChildCount()); i++ {
		child := tryNode.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "except_clause":
			catchBlockID := b.newBlockID("catch")
			b.addBlock(catchBlockID, BlockTypeCatch)
			b.cfGraph.AddEdge(tryBlockID, catchBlockID)

			// Extract exception variable binding (as e)
			// The except_clause has an optional name for the exception
			for j := 0; j < int(child.NamedChildCount()); j++ {
				namedChild := child.NamedChild(j)
				if namedChild != nil && namedChild.Type() == "as_pattern" {
					// Extract alias
					aliasNode := namedChild.ChildByFieldName("alias")
					if aliasNode != nil {
						excStmt := &core.Statement{
							Type:       core.StatementTypeAssignment,
							LineNumber: child.StartPoint().Row + 1,
							Def:        aliasNode.Content(b.sourceCode),
							Uses:       []string{},
						}
						b.appendStmt(catchBlockID, excStmt)
					}
				}
			}

			// Process except body
			exceptBody := child.ChildByFieldName("body")
			if exceptBody == nil {
				// Some tree-sitter versions use direct children
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c != nil && c.Type() == "block" {
						exceptBody = c
						break
					}
				}
			}

			var catchEndID string
			if exceptBody != nil {
				catchEndID = b.processBody(exceptBody, catchBlockID)
			} else {
				catchEndID = catchBlockID
			}

			if catchEndID != "" {
				b.cfGraph.AddEdge(catchEndID, mergeBlockID)
			}

		case "finally_clause":
			finallyBlockID := b.newBlockID("finally")
			b.addBlock(finallyBlockID, BlockTypeFinally)

			// Finally connects from both try-end and merge (always runs)
			b.cfGraph.AddEdge(mergeBlockID, finallyBlockID)

			finallyBody := child.ChildByFieldName("body")
			if finallyBody == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c != nil && c.Type() == "block" {
						finallyBody = c
						break
					}
				}
			}

			var finallyEndID string
			if finallyBody != nil {
				finallyEndID = b.processBody(finallyBody, finallyBlockID)
			} else {
				finallyEndID = finallyBlockID
			}

			// Finally becomes the new merge point
			newMerge := b.newBlockID("after_finally")
			b.addBlock(newMerge, BlockTypeNormal)
			if finallyEndID != "" {
				b.cfGraph.AddEdge(finallyEndID, newMerge)
			}
			mergeBlockID = newMerge

		case "else_clause":
			// try/except/else — else runs if no exception
			elseBlockID := b.newBlockID("try_else")
			b.addBlock(elseBlockID, BlockTypeNormal)
			if tryEndID != "" {
				b.cfGraph.AddEdge(tryEndID, elseBlockID)
			}

			elseBody := child.ChildByFieldName("body")
			if elseBody == nil {
				for j := 0; j < int(child.ChildCount()); j++ {
					c := child.Child(j)
					if c != nil && c.Type() == "block" {
						elseBody = c
						break
					}
				}
			}

			var elseEndID string
			if elseBody != nil {
				elseEndID = b.processBody(elseBody, elseBlockID)
			} else {
				elseEndID = elseBlockID
			}

			if elseEndID != "" {
				b.cfGraph.AddEdge(elseEndID, mergeBlockID)
			}
		}
	}

	return mergeBlockID
}

// processSwitch handles C/C++ switch statements.
//
// A switch produces a fan-out/fan-in CFG: a header block evaluates the
// condition and branches to one block per case (and optionally a
// default block); each case body falls through to the next case unless
// it ends with a break, in which case the edge goes to the merge block.
//
// CFG shape for `switch (x) { case 1: ...; case 2: ...; default: ... }`:
//
//	[header: cond=x]
//	   ├──> [case_1] ──> [body_1] ──fallthrough/break──> [body_2] / [merge]
//	   ├──> [case_2] ──> [body_2] ──fallthrough/break──> [default_body] / [merge]
//	   └──> [default] ─> [body_default] ────────────────> [merge]
//
// Phase 1 keeps the model simple: every case head connects to its
// predecessor case body so fallthrough is the default; explicit `break`
// statements are emitted as ordinary statements inside each body and
// will be lifted into edges by a later refactor. The header's
// successors-only fan-out captures the most important property —
// reachability of every case — for current dataflow consumers.
func (b *cfgBuilder) processSwitch(switchNode, stmtNode *sitter.Node, predBlockID string) string {
	headerBlockID := b.newBlockID("switch_header")
	b.addBlock(headerBlockID, BlockTypeSwitch)
	b.cfGraph.AddEdge(predBlockID, headerBlockID)

	if condNode := switchNode.ChildByFieldName("condition"); condNode != nil {
		b.appendStmt(headerBlockID, &core.Statement{
			Type:       core.StatementTypeIf,
			LineNumber: stmtNode.StartPoint().Row + 1,
			Uses:       extractIdentifiers(condNode, b.sourceCode),
		})
	}

	mergeBlockID := b.newBlockID("switch_after")
	b.addBlock(mergeBlockID, BlockTypeNormal)

	bodyNode := switchNode.ChildByFieldName("body")
	if bodyNode == nil {
		// Empty switch — header falls straight through to merge.
		b.cfGraph.AddEdge(headerBlockID, mergeBlockID)
		return mergeBlockID
	}

	var prevCaseEnd string
	for i := 0; i < int(bodyNode.NamedChildCount()); i++ {
		child := bodyNode.NamedChild(i)
		if child == nil || child.Type() != "case_statement" {
			continue
		}
		caseBlockID := b.newBlockID("case")
		b.addBlock(caseBlockID, BlockTypeNormal)
		b.cfGraph.AddEdge(headerBlockID, caseBlockID)
		if prevCaseEnd != "" {
			// Fallthrough from the previous case.
			b.cfGraph.AddEdge(prevCaseEnd, caseBlockID)
		}
		prevCaseEnd = b.processBody(child, caseBlockID)
	}

	// Last case (or empty body) flows into the merge block. Without a
	// trailing default, the header itself can also bypass every case
	// (e.g. condition matches none) — represent that with a header→merge
	// edge so reachability stays accurate.
	if prevCaseEnd != "" {
		b.cfGraph.AddEdge(prevCaseEnd, mergeBlockID)
	}
	b.cfGraph.AddEdge(headerBlockID, mergeBlockID)

	return mergeBlockID
}

// processDoWhile handles C/C++ do-while loops.
//
// Unlike `while`, the body always executes at least once because the
// condition is evaluated at the end. The CFG mirrors that: the
// predecessor flows directly into the body (no header gate), the body
// flows into the condition, and the condition either loops back to the
// body or falls through to the after block.
//
//	[pred] ──> [body] ──> [cond] ──true──> [body]
//	                      [cond] ──false─> [after]
func (b *cfgBuilder) processDoWhile(doNode, stmtNode *sitter.Node, predBlockID string) string {
	bodyBlockID := b.newBlockID("do_body")
	b.addBlock(bodyBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(predBlockID, bodyBlockID)

	bodyNode := doNode.ChildByFieldName("body")
	bodyEndID := bodyBlockID
	if bodyNode != nil {
		bodyEndID = b.processBody(bodyNode, bodyBlockID)
	}

	condBlockID := b.newBlockID("do_cond")
	b.addBlock(condBlockID, BlockTypeLoop)
	if bodyEndID != "" {
		b.cfGraph.AddEdge(bodyEndID, condBlockID)
	}
	if condNode := doNode.ChildByFieldName("condition"); condNode != nil {
		b.appendStmt(condBlockID, &core.Statement{
			Type:       core.StatementTypeWhile,
			LineNumber: stmtNode.StartPoint().Row + 1,
			Uses:       extractIdentifiers(condNode, b.sourceCode),
		})
	}

	// True edge loops back to the body; false edge falls through.
	b.cfGraph.AddEdge(condBlockID, bodyBlockID)

	afterBlockID := b.newBlockID("do_after")
	b.addBlock(afterBlockID, BlockTypeNormal)
	b.cfGraph.AddEdge(condBlockID, afterBlockID)
	return afterBlockID
}

// processWith handles with-statements.
// Creates a block with the context variable def, then processes the body.
func (b *cfgBuilder) processWith(withNode, stmtNode *sitter.Node, predBlockID string) string {
	// Extract "with expr as var" — var is a def
	for i := 0; i < int(withNode.NamedChildCount()); i++ {
		child := withNode.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Type() == "with_clause" || child.Type() == "with_item" {
			// Look for as_pattern within
			for j := 0; j < int(child.NamedChildCount()); j++ {
				item := child.NamedChild(j)
				if item != nil && item.Type() == "as_pattern" {
					aliasNode := item.ChildByFieldName("alias")
					valueNode := item.ChildByFieldName("value")
					if aliasNode != nil {
						withStmt := &core.Statement{
							Type:       core.StatementTypeWith,
							LineNumber: stmtNode.StartPoint().Row + 1,
							Def:        aliasNode.Content(b.sourceCode),
							Uses:       []string{},
						}
						if valueNode != nil {
							withStmt.Uses = extractIdentifiers(valueNode, b.sourceCode)
							withStmt.CallTarget = valueNode.Content(b.sourceCode)
						}
						b.appendStmt(predBlockID, withStmt)
					}
				}
			}
		}
	}

	// Process with body
	bodyNode := withNode.ChildByFieldName("body")
	if bodyNode != nil {
		return b.processBody(bodyNode, predBlockID)
	}

	return predBlockID
}

func (b *cfgBuilder) appendStmt(blockID string, stmt *core.Statement) {
	b.blockStmts[blockID] = append(b.blockStmts[blockID], stmt)
}

// extractStatement converts an AST node to a Statement.
// Handles assignments, calls, augmented assignments, and returns.
func (b *cfgBuilder) extractStatement(actualNode, stmtNode *sitter.Node) *core.Statement {
	var stmt *core.Statement

	switch actualNode.Type() {
	case "assignment":
		stmt = extractAssignment(actualNode, b.sourceCode)
	case "augmented_assignment":
		stmt = extractAugmentedAssignment(actualNode, b.sourceCode)
	case "call":
		stmt = extractCall(actualNode, b.sourceCode)
	case "return_statement":
		stmt = extractReturn(actualNode, b.sourceCode)
	default:
		return nil
	}

	if stmt != nil {
		stmt.LineNumber = stmtNode.StartPoint().Row + 1
	}
	return stmt
}

// extractAssignment processes "x = expr" nodes.
func extractAssignment(node *sitter.Node, sourceCode []byte) *core.Statement {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeAssignment,
		Uses: []string{},
	}

	switch leftNode.Type() {
	case "identifier":
		name := leftNode.Content(sourceCode)
		if !isKeyword(name) {
			stmt.Def = name
		}
	default:
		// Skip tuple unpacking, attribute, subscript for now
		return nil
	}

	stmt.CallTarget = rightNode.Content(sourceCode)

	switch rightNode.Type() {
	case "call":
		callStmt := extractCall(rightNode, sourceCode)
		if callStmt != nil {
			stmt.Uses = callStmt.Uses
			stmt.CallChain = callStmt.CallChain
		}

	case "subscript":
		// Unwrap nested subscripts to find the innermost non-subscript value.
		innermostValue := rightNode.ChildByFieldName("value")
		for innermostValue != nil && innermostValue.Type() == "subscript" {
			innermostValue = innermostValue.ChildByFieldName("value")
		}
		if innermostValue != nil {
			switch innermostValue.Type() {
			case "attribute":
				stmt.AttributeAccess = extractFullAttributeChain(innermostValue, sourceCode)
				stmt.Uses = extractIdentifiers(rightNode, sourceCode)
			case "call":
				callStmt := extractCall(innermostValue, sourceCode)
				if callStmt != nil {
					stmt.CallTarget = callStmt.CallTarget
					stmt.CallChain = callStmt.CallChain
					stmt.Uses = callStmt.Uses
				}
			default:
				stmt.Uses = extractIdentifiers(rightNode, sourceCode)
			}
		} else {
			stmt.Uses = extractIdentifiers(rightNode, sourceCode)
		}

	case "attribute":
		stmt.AttributeAccess = extractFullAttributeChain(rightNode, sourceCode)
		stmt.Uses = extractIdentifiers(rightNode, sourceCode)

	default:
		stmt.Uses = extractIdentifiers(rightNode, sourceCode)
	}

	return stmt
}

// extractFullAttributeChain recursively builds the full dotted attribute chain
// from a tree-sitter attribute node (e.g., request.GET → "request.GET").
func extractFullAttributeChain(node *sitter.Node, sourceCode []byte) string {
	if node == nil {
		return ""
	}
	switch node.Type() {
	case "identifier":
		return node.Content(sourceCode)
	case "attribute":
		obj := node.ChildByFieldName("object")
		attr := node.ChildByFieldName("attribute")
		if obj != nil && attr != nil {
			prefix := extractFullAttributeChain(obj, sourceCode)
			if prefix != "" {
				return prefix + "." + attr.Content(sourceCode)
			}
			return attr.Content(sourceCode)
		}
	}
	return ""
}

// extractAugmentedAssignment processes "x += expr" nodes.
func extractAugmentedAssignment(node *sitter.Node, sourceCode []byte) *core.Statement {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeAssignment,
		Uses: []string{},
	}

	if leftNode.Type() == "identifier" {
		name := leftNode.Content(sourceCode)
		if !isKeyword(name) {
			stmt.Def = name
			stmt.Uses = append(stmt.Uses, name)
		}
	} else {
		return nil
	}

	rightIds := extractIdentifiers(rightNode, sourceCode)
	stmt.Uses = append(stmt.Uses, rightIds...)
	return stmt
}

// extractCall processes function call nodes.
func extractCall(callNode *sitter.Node, sourceCode []byte) *core.Statement {
	if callNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeCall,
		Uses: []string{},
	}

	functionNode := callNode.ChildByFieldName("function")
	if functionNode != nil {
		stmt.CallTarget, stmt.CallChain = extractCallTarget(functionNode, sourceCode)
		targetIds := extractIdentifiers(functionNode, sourceCode)
		stmt.Uses = append(stmt.Uses, targetIds...)
	}

	argumentsNode := callNode.ChildByFieldName("arguments")
	if argumentsNode != nil {
		argIds := extractIdentifiersFromArgs(argumentsNode, sourceCode)
		stmt.Uses = append(stmt.Uses, argIds...)
	}

	return stmt
}

// extractReturn processes return statements.
func extractReturn(node *sitter.Node, sourceCode []byte) *core.Statement {
	stmt := &core.Statement{
		Type: core.StatementTypeReturn,
		Uses: []string{},
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil || child.Type() == "return" {
			continue
		}
		stmt.CallTarget = child.Content(sourceCode)
		stmt.Uses = append(stmt.Uses, extractIdentifiers(child, sourceCode)...)
	}

	return stmt
}

// extractCallTarget extracts the function name and full dotted chain from a call expression.
func extractCallTarget(functionNode *sitter.Node, sourceCode []byte) (string, string) {
	if functionNode == nil {
		return "", ""
	}

	switch functionNode.Type() {
	case "identifier":
		name := functionNode.Content(sourceCode)
		return name, name
	case "attribute":
		attrNode := functionNode.ChildByFieldName("attribute")
		if attrNode != nil {
			target := attrNode.Content(sourceCode)
			chain := extractFullAttributeChain(functionNode, sourceCode)
			return target, chain
		}
		content := functionNode.Content(sourceCode)
		return content, content
	default:
		content := functionNode.Content(sourceCode)
		return content, content
	}
}

// extractIdentifiers recursively finds all identifier nodes in an AST subtree.
func extractIdentifiers(node *sitter.Node, sourceCode []byte) []string {
	if node == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var identifiers []string

	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "identifier" {
			name := n.Content(sourceCode)
			if !isKeyword(name) && !seen[name] {
				seen[name] = true
				identifiers = append(identifiers, name)
			}
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			visit(n.Child(i))
		}
	}

	visit(node)
	return identifiers
}

// extractIdentifiersFromArgs extracts identifiers from call argument nodes.
func extractIdentifiersFromArgs(argumentsNode *sitter.Node, sourceCode []byte) []string {
	if argumentsNode == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var identifiers []string

	for i := 0; i < int(argumentsNode.ChildCount()); i++ {
		argNode := argumentsNode.Child(i)
		if argNode == nil {
			continue
		}
		if argNode.Type() == "," || argNode.Type() == "(" || argNode.Type() == ")" {
			continue
		}

		if argNode.Type() == "keyword_argument" {
			valueNode := argNode.ChildByFieldName("value")
			if valueNode != nil {
				for _, id := range extractIdentifiers(valueNode, sourceCode) {
					if !seen[id] {
						seen[id] = true
						identifiers = append(identifiers, id)
					}
				}
			}
			continue
		}

		for _, id := range extractIdentifiers(argNode, sourceCode) {
			if !seen[id] {
				seen[id] = true
				identifiers = append(identifiers, id)
			}
		}
	}

	return identifiers
}

// isKeyword checks if a name is a Python keyword.
func isKeyword(name string) bool {
	keywords := map[string]bool{
		"False": true, "None": true, "True": true,
		"and": true, "as": true, "assert": true, "async": true, "await": true,
		"break": true, "class": true, "continue": true, "def": true, "del": true,
		"elif": true, "else": true, "except": true, "finally": true, "for": true,
		"from": true, "global": true, "if": true, "import": true, "in": true,
		"is": true, "lambda": true, "nonlocal": true, "not": true, "or": true,
		"pass": true, "raise": true, "return": true, "try": true, "while": true,
		"with": true, "yield": true,
		"self": true,
	}
	return keywords[name]
}
