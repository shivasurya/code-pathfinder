package extraction

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// goKeywords contains Go reserved words, predeclared identifiers, builtin functions,
// and predeclared types that should NOT appear in Uses (they are not user variables).
// Equivalent to isKeyword in statements.go:508 for Python.
var goKeywords = map[string]bool{
	// Language keywords (25)
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
	// Predeclared identifiers
	"nil": true, "true": true, "false": true, "iota": true, "_": true,
	// Builtin functions
	"append": true, "cap": true, "close": true, "complex": true, "copy": true,
	"delete": true, "imag": true, "len": true, "make": true, "new": true,
	"panic": true, "print": true, "println": true, "real": true, "recover": true,
	// Predeclared types (not variable names)
	"error": true, "string": true, "int": true, "int8": true, "int16": true,
	"int32": true, "int64": true, "uint": true, "uint8": true, "uint16": true,
	"uint32": true, "uint64": true, "uintptr": true, "float32": true, "float64": true,
	"complex64": true, "complex128": true, "bool": true, "byte": true, "rune": true,
	"any": true,
}

// isGoKeyword returns true for Go reserved words, builtins, and predeclared types.
func isGoKeyword(name string) bool {
	return goKeywords[name]
}

// extractGoIdentifiers recursively extracts all identifier names from an AST subtree,
// filtering out Go keywords and blank identifiers. Skips field_identifier nodes
// (struct field access like .URL, .Path) since those are not variable references.
// Mirrors extractIdentifiers in statements.go:473 for Python.
func extractGoIdentifiers(node *sitter.Node, sourceCode []byte) []string {
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

		// Only collect "identifier" nodes, not "field_identifier" (which are struct field names)
		if n.Type() == "identifier" {
			name := n.Content(sourceCode)
			if !isGoKeyword(name) && !seen[name] {
				seen[name] = true
				identifiers = append(identifiers, name)
			}
			return
		}

		// Skip into selector_expression: only visit the operand (leftmost identifier),
		// not the field_identifier. For "r.URL.Path", we want "r" only.
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
	return identifiers
}

// extractGoCallTarget extracts the bare method name and full dotted chain
// from a call_expression's function child.
// Mirrors extractCallTarget in statements.go:287 for Python.
//
// Examples:
//
//	f()                → ("f", "f", "")
//	obj.Method()       → ("Method", "obj.Method", "")
//	a.b.c.d()          → ("d", "a.b.c.d", "a.b.c")
func extractGoCallTarget(functionNode *sitter.Node, sourceCode []byte) (callTarget, callChain, attrAccess string) {
	if functionNode == nil {
		return "", "", ""
	}

	switch functionNode.Type() {
	case "identifier":
		name := functionNode.Content(sourceCode)
		return name, name, ""

	case "selector_expression":
		// Go's equivalent of Python's attribute access: obj.Method
		fieldNode := functionNode.ChildByFieldName("field")
		if fieldNode != nil {
			target := fieldNode.Content(sourceCode)
			chain := extractGoSelectorChain(functionNode, sourceCode)
			// Extract attribute access prefix (everything before the last selector)
			operandNode := functionNode.ChildByFieldName("operand")
			attr := ""
			if operandNode != nil && operandNode.Type() == "selector_expression" {
				attr = extractGoSelectorChain(operandNode, sourceCode)
			}
			return target, chain, attr
		}
		content := functionNode.Content(sourceCode)
		return content, content, ""

	default:
		content := functionNode.Content(sourceCode)
		return content, content, ""
	}
}

// extractGoSelectorChain recursively builds the full dotted chain from a selector_expression.
// "obj.field.method" → "obj.field.method"
// Mirrors extractFullAttributeChain in statements.go:318 for Python.
func extractGoSelectorChain(node *sitter.Node, sourceCode []byte) string {
	if node == nil {
		return ""
	}
	switch node.Type() {
	case "identifier", "field_identifier":
		return node.Content(sourceCode)
	case "selector_expression":
		operand := node.ChildByFieldName("operand")
		field := node.ChildByFieldName("field")
		if operand != nil && field != nil {
			prefix := extractGoSelectorChain(operand, sourceCode)
			if prefix != "" {
				return prefix + "." + field.Content(sourceCode)
			}
			return field.Content(sourceCode)
		}
	case "call_expression":
		// Handle chained calls: obj.Method1().Method2()
		funcNode := node.ChildByFieldName("function")
		if funcNode != nil {
			return extractGoSelectorChain(funcNode, sourceCode) + "()"
		}
	}
	return node.Content(sourceCode)
}

// extractGoIdentifiersFromArgs extracts all identifier names from function call arguments.
// Mirrors extractIdentifiersFromArgs in statements.go:341 for Python.
func extractGoIdentifiersFromArgs(argumentsNode *sitter.Node, sourceCode []byte) []string {
	if argumentsNode == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var identifiers []string

	for i := 0; i < int(argumentsNode.NamedChildCount()); i++ {
		argNode := argumentsNode.NamedChild(i)
		if argNode == nil {
			continue
		}

		ids := extractGoIdentifiers(argNode, sourceCode)
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				identifiers = append(identifiers, id)
			}
		}
	}

	return identifiers
}

// ParseGoFile parses a Go source file using tree-sitter.
// Mirrors ParsePythonFile in statements.go:525 for Python.
func ParseGoFile(sourceCode []byte) (*sitter.Tree, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go code: %w", err)
	}

	if tree == nil {
		return nil, fmt.Errorf("tree-sitter returned nil tree")
	}

	return tree, nil
}

// ExtractGoStatements extracts top-level statements from a Go function body.
// Mirrors ExtractStatements (statements.go:15) for the Go tree-sitter grammar.
//
// This is a FLAT extractor: control flow constructs (if, for, switch, select)
// are skipped — they are handled by BuildGoCFGFromAST (cfg/builder_go.go).
//
// Signature matches ExtractStatements exactly so GenerateGoTaintSummaries
// can call both extractors with the same pattern.
func ExtractGoStatements(filePath string, sourceCode []byte, functionNode *sitter.Node) ([]*core.Statement, error) {
	if functionNode == nil {
		return nil, fmt.Errorf("function node is nil")
	}

	bodyNode := functionNode.ChildByFieldName("body")
	if bodyNode == nil {
		return []*core.Statement{}, nil
	}

	var statements []*core.Statement

	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		stmtNode := bodyNode.Child(i)
		if stmtNode == nil {
			continue
		}

		var newStmts []*core.Statement

		switch stmtNode.Type() {
		case "short_var_declaration":
			newStmts = extractGoShortVarDecl(stmtNode, sourceCode)

		case "var_declaration":
			newStmts = extractGoVarDecl(stmtNode, sourceCode)

		case "assignment_statement":
			newStmts = extractGoAssignment(stmtNode, sourceCode)

		case "expression_statement":
			// Unwrap expression_statement to get the actual expression
			for ci := 0; ci < int(stmtNode.NamedChildCount()); ci++ {
				child := stmtNode.NamedChild(ci)
				if child != nil && child.Type() == "call_expression" {
					if stmt := extractGoCall(child, sourceCode); stmt != nil {
						stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
						statements = append(statements, stmt)
					}
				}
			}
			continue

		case "return_statement":
			if stmt := extractGoReturn(stmtNode, sourceCode); stmt != nil {
				stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
				statements = append(statements, stmt)
			}
			continue

		case "go_statement":
			if stmt := extractGoGoDefer(stmtNode, sourceCode); stmt != nil {
				stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
				statements = append(statements, stmt)
			}
			continue

		case "defer_statement":
			if stmt := extractGoGoDefer(stmtNode, sourceCode); stmt != nil {
				stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
				statements = append(statements, stmt)
			}
			continue

		case "send_statement":
			if stmt := extractGoSend(stmtNode, sourceCode); stmt != nil {
				stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
				statements = append(statements, stmt)
			}
			continue

		// Skip control flow — handled by CFG builder
		case "if_statement", "for_statement", "switch_statement", "type_switch_statement",
			"select_statement":
			continue

		default:
			continue
		}

		// Set line numbers on all extracted statements
		for _, stmt := range newStmts {
			if stmt != nil {
				stmt.LineNumber = uint32(stmtNode.StartPoint().Row + 1) //nolint:unconvert
				statements = append(statements, stmt)
			}
		}
	}

	return statements, nil
}

// extractGoShortVarDecl handles `:=` assignments.
// node type: "short_var_declaration"
// Returns one Statement per LHS identifier. Blank identifiers (_) are skipped.
//
// Examples:
//
//	x := 10                → [{Def:"x"}]
//	rows, err := db.Query  → [{Def:"rows"}, {Def:"err"}]
//	_, err := f()          → [{Def:"err"}]
func extractGoShortVarDecl(node *sitter.Node, sourceCode []byte) []*core.Statement {
	if node == nil {
		return nil
	}

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return nil
	}

	// Go tree-sitter wraps LHS and RHS in "expression_list" nodes.
	// Unwrap to get the actual RHS expression.
	actualRight := rightNode
	if rightNode.Type() == "expression_list" && rightNode.NamedChildCount() > 0 {
		actualRight = rightNode.NamedChild(0)
	}

	// Extract call target and uses from RHS
	callTarget, callChain, attrAccess := "", "", ""
	var uses []string

	switch actualRight.Type() {
	case "call_expression":
		funcNode := actualRight.ChildByFieldName("function")
		callTarget, callChain, attrAccess = extractGoCallTarget(funcNode, sourceCode)
		argsNode := actualRight.ChildByFieldName("arguments")
		uses = extractGoIdentifiersFromArgs(argsNode, sourceCode)
	case "selector_expression":
		// Pure attribute access: x := r.URL.Path
		attrAccess = extractGoSelectorChain(actualRight, sourceCode)
		uses = extractGoIdentifiers(actualRight, sourceCode)
	case "unary_expression":
		// Handle receive: x := <-ch
		if actualRight.ChildCount() > 0 && actualRight.Child(0).Content(sourceCode) == "<-" {
			// Channel receive — extract the operand identifier
			for ci := 0; ci < int(actualRight.NamedChildCount()); ci++ {
				child := actualRight.NamedChild(ci)
				if child != nil {
					ids := extractGoIdentifiers(child, sourceCode)
					uses = append(uses, ids...)
				}
			}
		} else {
			uses = extractGoIdentifiers(actualRight, sourceCode)
		}
	default:
		uses = extractGoIdentifiers(actualRight, sourceCode)
	}

	// Collect LHS variable names
	var lhsNames []string
	if leftNode.Type() == "expression_list" {
		for i := 0; i < int(leftNode.NamedChildCount()); i++ {
			child := leftNode.NamedChild(i)
			if child != nil && child.Type() == "identifier" {
				name := child.Content(sourceCode)
				if name != "_" && !isGoKeyword(name) {
					lhsNames = append(lhsNames, name)
				}
			}
		}
	} else if leftNode.Type() == "identifier" {
		name := leftNode.Content(sourceCode)
		if name != "_" && !isGoKeyword(name) {
			lhsNames = append(lhsNames, name)
		}
	}

	// Emit one statement per LHS variable
	var stmts []*core.Statement
	for _, name := range lhsNames {
		stmts = append(stmts, &core.Statement{
			Type:            core.StatementTypeAssignment,
			Def:             name,
			Uses:            uses,
			CallTarget:      callTarget,
			CallChain:       callChain,
			AttributeAccess: attrAccess,
		})
	}

	return stmts
}

// ExtractGoVarDeclFromNode extracts statements from a top-level var_declaration node.
// Exported for use by GenerateGoTaintSummaries for package-level variable extraction.
func ExtractGoVarDeclFromNode(node *sitter.Node, sourceCode []byte) []*core.Statement {
	return extractGoVarDecl(node, sourceCode)
}

// extractGoVarDecl handles `var x = expr` declarations.
// node type: "var_declaration" containing "var_spec" children.
func extractGoVarDecl(node *sitter.Node, sourceCode []byte) []*core.Statement {
	if node == nil {
		return nil
	}

	var stmts []*core.Statement

	for i := 0; i < int(node.NamedChildCount()); i++ {
		spec := node.NamedChild(i)
		if spec == nil || spec.Type() != "var_spec" {
			continue
		}

		// var_spec: identifier (name field) then optionally type then expression_list (value field)
		nameNode := spec.ChildByFieldName("name")
		valueNode := spec.ChildByFieldName("value")

		if nameNode == nil {
			continue
		}

		name := nameNode.Content(sourceCode)
		if isGoKeyword(name) || name == "_" {
			continue
		}

		stmt := &core.Statement{
			Type: core.StatementTypeAssignment,
			Def:  name,
			Uses: []string{},
		}

		if valueNode != nil {
			// var_spec wraps value in "expression_list". Unwrap first.
			actualValue := valueNode
			if valueNode.Type() == "expression_list" && valueNode.NamedChildCount() > 0 {
				actualValue = valueNode.NamedChild(0)
			}

			switch actualValue.Type() {
			case "call_expression":
				funcN := actualValue.ChildByFieldName("function")
				stmt.CallTarget, stmt.CallChain, stmt.AttributeAccess = extractGoCallTarget(funcN, sourceCode)
				argsNode := actualValue.ChildByFieldName("arguments")
				stmt.Uses = extractGoIdentifiersFromArgs(argsNode, sourceCode)
			case "selector_expression":
				stmt.AttributeAccess = extractGoSelectorChain(actualValue, sourceCode)
				stmt.Uses = extractGoIdentifiers(actualValue, sourceCode)
			default:
				stmt.Uses = extractGoIdentifiers(actualValue, sourceCode)
			}
		}

		stmts = append(stmts, stmt)
	}

	return stmts
}

// extractGoAssignment handles `=` and compound assignment operators (+=, -=, etc.).
// node type: "assignment_statement"
// Returns one Statement per LHS identifier.
func extractGoAssignment(node *sitter.Node, sourceCode []byte) []*core.Statement {
	if node == nil {
		return nil
	}

	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	if leftNode == nil || rightNode == nil {
		return nil
	}

	// Check for augmented assignment (+=, -=, etc.)
	isAugmented := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			op := child.Content(sourceCode)
			if op != "=" && strings.HasSuffix(op, "=") && len(op) >= 2 {
				isAugmented = true
				break
			}
		}
	}

	// Unwrap expression_list on RHS
	actualRight := rightNode
	if rightNode.Type() == "expression_list" && rightNode.NamedChildCount() > 0 {
		actualRight = rightNode.NamedChild(0)
	}

	// Extract RHS info
	callTarget, callChain, attrAccess := "", "", ""
	var uses []string

	switch actualRight.Type() {
	case "call_expression":
		funcN := actualRight.ChildByFieldName("function")
		callTarget, callChain, attrAccess = extractGoCallTarget(funcN, sourceCode)
		argsNode := actualRight.ChildByFieldName("arguments")
		uses = extractGoIdentifiersFromArgs(argsNode, sourceCode)
	case "selector_expression":
		attrAccess = extractGoSelectorChain(actualRight, sourceCode)
		uses = extractGoIdentifiers(actualRight, sourceCode)
	default:
		uses = extractGoIdentifiers(actualRight, sourceCode)
	}

	// Collect LHS variable names
	var stmts []*core.Statement

	extractLHS := func(lhs *sitter.Node) {
		if lhs == nil {
			return
		}
		if lhs.Type() != "identifier" {
			// index_expression, selector_expression on LHS — skip (mutates existing, no new Def)
			return
		}
		name := lhs.Content(sourceCode)
		if isGoKeyword(name) || name == "_" {
			return
		}
		stmtUses := uses
		if isAugmented {
			// Augmented: LHS is both def and use (x += y means x = x + y)
			stmtUses = append([]string{name}, uses...)
		}
		stmts = append(stmts, &core.Statement{
			Type:            core.StatementTypeAssignment,
			Def:             name,
			Uses:            stmtUses,
			CallTarget:      callTarget,
			CallChain:       callChain,
			AttributeAccess: attrAccess,
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

// extractGoCall handles standalone call expressions (no assignment).
// node type: "call_expression"
// Mirrors extractCall in statements.go:244 for Python.
func extractGoCall(callNode *sitter.Node, sourceCode []byte) *core.Statement {
	if callNode == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeCall,
		Uses: []string{},
	}

	funcNode := callNode.ChildByFieldName("function")
	if funcNode != nil {
		stmt.CallTarget, stmt.CallChain, _ = extractGoCallTarget(funcNode, sourceCode)
	}

	argsNode := callNode.ChildByFieldName("arguments")
	if argsNode != nil {
		stmt.Uses = extractGoIdentifiersFromArgs(argsNode, sourceCode)
	}

	return stmt
}

// extractGoReturn handles return statements.
// node type: "return_statement"
// Mirrors extractReturn in statements.go:438 for Python.
func extractGoReturn(node *sitter.Node, sourceCode []byte) *core.Statement {
	if node == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeReturn,
		Uses: []string{},
	}

	// Return values are named children after the "return" keyword.
	// May be wrapped in expression_list.
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}

		// Unwrap expression_list
		actual := child
		if child.Type() == "expression_list" && child.NamedChildCount() > 0 {
			actual = child.NamedChild(0)
		}

		switch actual.Type() {
		case "call_expression":
			funcN := actual.ChildByFieldName("function")
			stmt.CallTarget, stmt.CallChain, _ = extractGoCallTarget(funcN, sourceCode)
			argsNode := actual.ChildByFieldName("arguments")
			argIds := extractGoIdentifiersFromArgs(argsNode, sourceCode)
			stmt.Uses = append(stmt.Uses, argIds...)
		default:
			ids := extractGoIdentifiers(actual, sourceCode)
			stmt.Uses = append(stmt.Uses, ids...)
		}
	}

	return stmt
}

// extractGoGoDefer handles go/defer statements by extracting the inner call expression.
// node types: "go_statement", "defer_statement"
// Both contain a call_expression child.
func extractGoGoDefer(node *sitter.Node, sourceCode []byte) *core.Statement {
	if node == nil {
		return nil
	}

	// Find the call_expression child
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type() == "call_expression" {
			return extractGoCall(child, sourceCode)
		}
	}

	return nil
}

// extractGoSend handles channel send statements: ch <- data
// node type: "send_statement"
// Emits Uses for both the channel variable and the sent data.
func extractGoSend(node *sitter.Node, sourceCode []byte) *core.Statement {
	if node == nil {
		return nil
	}

	stmt := &core.Statement{
		Type: core.StatementTypeExpression,
		Uses: []string{},
	}

	// send_statement children: channel_identifier, "<-", value_identifier
	// No field names — iterate named children
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		ids := extractGoIdentifiers(child, sourceCode)
		stmt.Uses = append(stmt.Uses, ids...)
	}

	return stmt
}
