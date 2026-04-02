package extraction

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== TEST HELPERS ==========

// parseGoFunction parses Go source code and returns the tree, function node, and source bytes.
// Caller must close the tree with defer tree.Close().
func parseGoFunction(t *testing.T, source string, funcName string) (*sitter.Tree, *sitter.Node, []byte) {
	t.Helper()
	sourceBytes := []byte(source)

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceBytes)
	require.NoError(t, err)

	root := tree.RootNode()
	funcNode := findGoFunctionByName(root, funcName, sourceBytes)
	require.NotNil(t, funcNode, "Function %s not found in Go source", funcName)

	return tree, funcNode, sourceBytes
}

// findGoFunctionByName finds a function_declaration or method_declaration by name.
func findGoFunctionByName(node *sitter.Node, name string, source []byte) *sitter.Node {
	if node == nil {
		return nil
	}

	if node.Type() == "function_declaration" || node.Type() == "method_declaration" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil && nameNode.Content(source) == name {
			return node
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		if result := findGoFunctionByName(node.Child(i), name, source); result != nil {
			return result
		}
	}

	return nil
}

// ========== KEYWORD FILTER TESTS ==========

func TestIsGoKeyword(t *testing.T) {
	// Go language keywords must be filtered
	keywords := []string{
		"break", "case", "chan", "const", "continue", "default", "defer",
		"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
		"interface", "map", "package", "range", "return", "select", "struct",
		"switch", "type", "var",
	}
	for _, kw := range keywords {
		assert.True(t, isGoKeyword(kw), "%s should be a keyword", kw)
	}

	// Predeclared identifiers
	predeclared := []string{"nil", "true", "false", "iota", "_"}
	for _, p := range predeclared {
		assert.True(t, isGoKeyword(p), "%s should be filtered", p)
	}

	// Builtin functions
	builtins := []string{"append", "cap", "close", "copy", "delete", "len", "make", "new", "panic", "recover"}
	for _, b := range builtins {
		assert.True(t, isGoKeyword(b), "%s should be filtered", b)
	}

	// Predeclared types
	types := []string{"error", "string", "int", "bool", "byte", "rune", "float64", "any"}
	for _, tp := range types {
		assert.True(t, isGoKeyword(tp), "%s should be filtered", tp)
	}

	// Real variable names must NOT be filtered
	variables := []string{"db", "query", "r", "w", "input", "result", "user", "config", "ctx"}
	for _, v := range variables {
		assert.False(t, isGoKeyword(v), "%s should NOT be a keyword", v)
	}
}

// ========== IDENTIFIER EXTRACTION TESTS ==========

func TestExtractGoIdentifiers_SimpleExpression(t *testing.T) {
	source := `package main

func foo() {
	x := a + b
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	// Get the function body
	bodyNode := funcNode.ChildByFieldName("body")
	require.NotNil(t, bodyNode)

	// Get the first statement (short_var_declaration)
	var shortVarDecl *sitter.Node
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() == "short_var_declaration" {
			shortVarDecl = child
			break
		}
	}
	require.NotNil(t, shortVarDecl)

	// Extract identifiers from RHS (expression_list wrapping binary_expression)
	rightNode := shortVarDecl.ChildByFieldName("right")
	require.NotNil(t, rightNode)

	ids := extractGoIdentifiers(rightNode, sourceBytes)
	assert.Contains(t, ids, "a")
	assert.Contains(t, ids, "b")
	assert.Equal(t, 2, len(ids))
}

func TestExtractGoIdentifiers_SelectorExpression(t *testing.T) {
	source := `package main

func foo() {
	x := r.URL.Path
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	bodyNode := funcNode.ChildByFieldName("body")
	require.NotNil(t, bodyNode)

	var shortVarDecl *sitter.Node
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() == "short_var_declaration" {
			shortVarDecl = child
			break
		}
	}
	require.NotNil(t, shortVarDecl)

	rightNode := shortVarDecl.ChildByFieldName("right")
	require.NotNil(t, rightNode)

	// Should extract only "r" (the root variable), not "URL" or "Path" (field names)
	ids := extractGoIdentifiers(rightNode, sourceBytes)
	assert.Contains(t, ids, "r")
	assert.Equal(t, 1, len(ids), "should only extract 'r', not field names URL/Path")
}

// ========== SHORT VAR DECLARATION TESTS ==========

func TestExtractGoStatements_ShortVarDecl_Simple(t *testing.T) {
	source := `package main

func foo() {
	x := 10
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, "x", stmt.Def)
	assert.Equal(t, uint32(4), stmt.LineNumber)
	assert.Empty(t, stmt.Uses)
}

func TestExtractGoStatements_ShortVarDecl_FromVariable(t *testing.T) {
	source := `package main

func foo() {
	y := x
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	assert.Equal(t, "y", stmts[0].Def)
	assert.Contains(t, stmts[0].Uses, "x")
}

func TestExtractGoStatements_ShortVarDecl_FromCall(t *testing.T) {
	source := `package main

func foo() {
	result := db.Query(sql)
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, "result", stmt.Def)
	assert.Equal(t, "Query", stmt.CallTarget)
	assert.Equal(t, "db.Query", stmt.CallChain)
	assert.Contains(t, stmt.Uses, "sql")
}

func TestExtractGoStatements_ShortVarDecl_MultiAssign(t *testing.T) {
	source := `package main

func foo() {
	rows, err := db.Query(sql)
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 2, len(stmts), "multi-assign emits one statement per LHS var")

	assert.Equal(t, "rows", stmts[0].Def)
	assert.Equal(t, "Query", stmts[0].CallTarget)
	assert.Equal(t, "db.Query", stmts[0].CallChain)

	assert.Equal(t, "err", stmts[1].Def)
	assert.Equal(t, "Query", stmts[1].CallTarget)
	assert.Equal(t, "db.Query", stmts[1].CallChain)
}

func TestExtractGoStatements_ShortVarDecl_BlankIdentifier(t *testing.T) {
	source := `package main

func foo() {
	_, err := db.Query(sql)
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts), "blank identifier _ should not emit a statement")

	assert.Equal(t, "err", stmts[0].Def)
}

func TestExtractGoStatements_ShortVarDecl_AttributeAccess(t *testing.T) {
	source := `package main

func foo() {
	path := r.URL.Path
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, "path", stmt.Def)
	assert.Equal(t, "r.URL.Path", stmt.AttributeAccess)
	assert.Contains(t, stmt.Uses, "r")
}

// ========== VAR DECLARATION TESTS ==========

func TestExtractGoStatements_VarDecl(t *testing.T) {
	source := `package main

func foo() {
	var x = getValue()
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, "x", stmt.Def)
	assert.Equal(t, "getValue", stmt.CallTarget)
}

// ========== ASSIGNMENT STATEMENT TESTS ==========

func TestExtractGoStatements_Assignment(t *testing.T) {
	source := `package main

func foo() {
	x = y + z
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, "x", stmt.Def)
	assert.Contains(t, stmt.Uses, "y")
	assert.Contains(t, stmt.Uses, "z")
}

func TestExtractGoStatements_AugmentedAssignment(t *testing.T) {
	source := `package main

func foo() {
	x += y
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, "x", stmt.Def)
	assert.Contains(t, stmt.Uses, "x") // augmented: LHS is both def and use
	assert.Contains(t, stmt.Uses, "y")
}

// ========== CALL EXPRESSION TESTS ==========

func TestExtractGoStatements_StandaloneCall(t *testing.T) {
	source := `package main

func foo() {
	fmt.Println(x)
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	assert.Equal(t, "", stmt.Def)
	assert.Equal(t, "Println", stmt.CallTarget)
	assert.Equal(t, "fmt.Println", stmt.CallChain)
	assert.Contains(t, stmt.Uses, "x")
}

// ========== RETURN TESTS ==========

func TestExtractGoStatements_Return(t *testing.T) {
	source := `package main

func foo() string {
	return input
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeReturn, stmt.Type)
	assert.Contains(t, stmt.Uses, "input")
}

func TestExtractGoStatements_ReturnCall(t *testing.T) {
	source := `package main

func foo() string {
	return r.FormValue("q")
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeReturn, stmt.Type)
	assert.Equal(t, "FormValue", stmt.CallTarget)
	assert.Equal(t, "r.FormValue", stmt.CallChain)
}

// ========== GO/DEFER TESTS ==========

func TestExtractGoStatements_GoStatement(t *testing.T) {
	source := `package main

func foo() {
	go handler(data)
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	assert.Equal(t, "handler", stmt.CallTarget)
	assert.Contains(t, stmt.Uses, "data")
}

func TestExtractGoStatements_DeferStatement(t *testing.T) {
	source := `package main

func foo() {
	defer rows.Close()
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeCall, stmt.Type)
	assert.Equal(t, "Close", stmt.CallTarget)
	assert.Equal(t, "rows.Close", stmt.CallChain)
}

// ========== CHANNEL SEND TESTS ==========

func TestExtractGoStatements_ChannelSend(t *testing.T) {
	source := `package main

func foo() {
	ch <- data
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeExpression, stmt.Type)
	assert.Equal(t, "", stmt.Def)
	assert.Contains(t, stmt.Uses, "data")
	assert.Contains(t, stmt.Uses, "ch")
}

// ========== CHANNEL RECEIVE IN SHORT VAR TESTS ==========

func TestExtractGoStatements_ChannelReceive(t *testing.T) {
	source := `package main

func foo() {
	val := <-ch
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 1, len(stmts))

	stmt := stmts[0]
	assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
	assert.Equal(t, "val", stmt.Def)
	assert.Contains(t, stmt.Uses, "ch")
}

// ========== CONTROL FLOW SKIPPING TESTS ==========

func TestExtractGoStatements_SkipsControlFlow(t *testing.T) {
	source := `package main

func foo() {
	x := 1
	if true {
		y := 2
	}
	z := 3
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	require.Equal(t, 2, len(stmts), "if_statement should be skipped, only x and z extracted")

	assert.Equal(t, "x", stmts[0].Def)
	assert.Equal(t, "z", stmts[1].Def)
}

// ========== NIL/EMPTY TESTS ==========

func TestExtractGoStatements_NilNode(t *testing.T) {
	stmts, err := ExtractGoStatements("test.go", nil, nil)
	assert.Error(t, err)
	assert.Nil(t, stmts)
}

func TestExtractGoStatements_EmptyFunction(t *testing.T) {
	source := `package main

func foo() {}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "foo")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)
	assert.Equal(t, 0, len(stmts))
}

// ========== INTEGRATION: SQL INJECTION PATTERN ==========

func TestExtractGoStatements_SQLInjectionPattern(t *testing.T) {
	source := `package main

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	sql := "SELECT * FROM items WHERE name = '" + query + "'"
	rows, err := db.Query(sql)
	_ = err
	defer rows.Close()
}
`
	tree, funcNode, sourceBytes := parseGoFunction(t, source, "handleSearch")
	defer tree.Close()

	stmts, err := ExtractGoStatements("test.go", sourceBytes, funcNode)
	require.NoError(t, err)

	// Expected statements:
	// 1. query := r.FormValue("q") → Def:"query", CallTarget:"FormValue", CallChain:"r.FormValue"
	// 2. sql := "..." + query + "..." → Def:"sql", Uses:["query"]
	// 3. rows := db.Query(sql) → Def:"rows", CallTarget:"Query", CallChain:"db.Query", Uses:["sql"]
	// 4. err := db.Query(sql) → Def:"err", CallTarget:"Query", CallChain:"db.Query", Uses:["sql"]
	// 5. defer rows.Close() → Type:call, CallTarget:"Close", CallChain:"rows.Close"

	// Filter to meaningful statements (with Def or call type)
	var meaningful []*core.Statement
	for _, s := range stmts {
		if s.Def != "" || s.Type == core.StatementTypeCall {
			meaningful = append(meaningful, s)
		}
	}

	require.GreaterOrEqual(t, len(meaningful), 4, "should have at least: query, sql, rows+err (2 from multi-assign), defer")

	// Verify source statement
	assert.Equal(t, "query", meaningful[0].Def)
	assert.Equal(t, "FormValue", meaningful[0].CallTarget)
	assert.Equal(t, "r.FormValue", meaningful[0].CallChain)

	// Verify concat statement
	assert.Equal(t, "sql", meaningful[1].Def)
	assert.Contains(t, meaningful[1].Uses, "query")

	// Verify sink statements (multi-assign: rows, err)
	sinkDefs := []string{}
	for _, s := range meaningful[2:] {
		if s.CallTarget == "Query" {
			sinkDefs = append(sinkDefs, s.Def)
			assert.Equal(t, "db.Query", s.CallChain)
			assert.Contains(t, s.Uses, "sql")
		}
	}
	assert.Contains(t, sinkDefs, "rows")
	assert.Contains(t, sinkDefs, "err")
}
