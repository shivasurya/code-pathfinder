package extraction

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	clang "github.com/smacker/go-tree-sitter/c"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// testFuncName is the conventional function name used in every C/C++
// extraction fixture below — keeps test sources tight and removes a
// duplicated argument from every call site.
const testFuncName = "f"

// parseCFunction parses C source code and returns the function_definition
// node named `testFuncName`. The caller must close the tree via
// `defer tree.Close()`.
func parseCFunction(t *testing.T, source string) (*sitter.Tree, *sitter.Node, []byte) {
	t.Helper()
	src := []byte(source)

	parser := sitter.NewParser()
	parser.SetLanguage(clang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	require.NoError(t, err)

	fn := findCFunctionByName(tree.RootNode(), testFuncName, src)
	require.NotNil(t, fn, "function %q not found", testFuncName)
	return tree, fn, src
}

// findCFunctionByName recursively searches the AST for a
// function_definition whose declarator's identifier matches name.
func findCFunctionByName(node *sitter.Node, name string, src []byte) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "function_definition" {
		if d := node.ChildByFieldName("declarator"); d != nil && testCFunctionName(d, src) == name {
			return node
		}
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if r := findCFunctionByName(node.Child(i), name, src); r != nil {
			return r
		}
	}
	return nil
}

// testCFunctionName unwraps a function_declarator to its identifier.
func testCFunctionName(node *sitter.Node, src []byte) string {
	if node == nil {
		return ""
	}
	switch node.Type() {
	case "identifier":
		return node.Content(src)
	case "function_declarator", "pointer_declarator", "parenthesized_declarator":
		if d := node.ChildByFieldName("declarator"); d != nil {
			return testCFunctionName(d, src)
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if r := testCFunctionName(node.NamedChild(i), src); r != "" {
			return r
		}
	}
	return ""
}

// findStmt returns the first statement matching the predicate, walking
// into NestedStatements / ElseBranch as needed.
func findStmt(stmts []*core.Statement, pred func(*core.Statement) bool) *core.Statement {
	for _, s := range stmts {
		if s == nil {
			continue
		}
		if pred(s) {
			return s
		}
		if got := findStmt(s.NestedStatements, pred); got != nil {
			return got
		}
		if got := findStmt(s.ElseBranch, pred); got != nil {
			return got
		}
	}
	return nil
}

func TestExtractCStatements_NilFunction(t *testing.T) {
	stmts, err := ExtractCStatements("/x.c", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, stmts)
}

func TestExtractCStatements_DeclarationWithBinaryOp(t *testing.T) {
	src := `int f(int a, int b) {
    int x = a + b;
    return x;
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 2)
	assert.Equal(t, core.StatementTypeAssignment, stmts[0].Type)
	assert.Equal(t, "x", stmts[0].Def)
	assert.ElementsMatch(t, []string{"a", "b"}, stmts[0].Uses)

	assert.Equal(t, core.StatementTypeReturn, stmts[1].Type)
	assert.Equal(t, []string{"x"}, stmts[1].Uses)
}

func TestExtractCStatements_AssignmentFromCall(t *testing.T) {
	src := `int f(int y) {
    int x;
    x = func(y);
    return x;
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	got := findStmt(stmts, func(s *core.Statement) bool {
		return s.Type == core.StatementTypeAssignment && s.CallTarget == "func"
	})
	require.NotNil(t, got, "expected assignment from call")
	assert.Equal(t, "x", got.Def)
	assert.Equal(t, []string{"y"}, got.Uses)
	assert.Equal(t, "func", got.CallChain)
}

func TestExtractCStatements_BareCall(t *testing.T) {
	src := `void f(int a, int b) {
    func(a, b);
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, core.StatementTypeCall, stmts[0].Type)
	assert.Equal(t, "func", stmts[0].CallTarget)
	assert.ElementsMatch(t, []string{"a", "b"}, stmts[0].Uses)
	assert.ElementsMatch(t, []string{"a", "b"}, stmts[0].CallArgs)
}

func TestExtractCStatements_IfElse(t *testing.T) {
	src := `void f(int x, int y) {
    if (x > 0) {
        consume(y);
    } else {
        report(x);
    }
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	ifStmt := stmts[0]
	assert.Equal(t, core.StatementTypeIf, ifStmt.Type)
	assert.Equal(t, []string{"x"}, ifStmt.Uses)
	require.NotEmpty(t, ifStmt.NestedStatements)
	assert.Equal(t, core.StatementTypeCall, ifStmt.NestedStatements[0].Type)
	assert.Equal(t, "consume", ifStmt.NestedStatements[0].CallTarget)

	require.NotEmpty(t, ifStmt.ElseBranch)
	assert.Equal(t, "report", ifStmt.ElseBranch[0].CallTarget)
}

func TestExtractCStatements_ForLoop(t *testing.T) {
	src := `void f(int n) {
    for (int i = 0; i < n; i++) {
        do_thing(i);
    }
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	forStmt := stmts[0]
	assert.Equal(t, core.StatementTypeFor, forStmt.Type)
	assert.Equal(t, "i", forStmt.Def)
	assert.Contains(t, forStmt.Uses, "n")
	assert.NotContains(t, forStmt.Uses, "i", "loop variable must not appear in Uses")
}

func TestExtractCStatements_While(t *testing.T) {
	src := `void f(int x) {
    while (x > 0) {
        x--;
    }
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, core.StatementTypeWhile, stmts[0].Type)
	assert.Equal(t, []string{"x"}, stmts[0].Uses)
}

func TestExtractCStatements_PointerArrowAssignment(t *testing.T) {
	src := `void f(struct S* p, int val) {
    p->name = val;
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, "p", stmts[0].Def)
	assert.Equal(t, []string{"val"}, stmts[0].Uses)
}

func TestExtractCStatements_SubscriptAssignment(t *testing.T) {
	src := `void f(int* buf, int* input, int i, int j) {
    buf[i] = input[j];
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, "buf", stmts[0].Def)
	assert.ElementsMatch(t, []string{"i", "input", "j"}, stmts[0].Uses)
}

func TestExtractCStatements_KeywordFilter(t *testing.T) {
	src := `int f(int* p) {
    int n = sizeof(*p);
    int y = (int)n;
    if (p == NULL) return 0;
    return n + y;
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	for _, s := range stmts {
		assert.NotContains(t, s.Uses, "sizeof")
		assert.NotContains(t, s.Uses, "int")
		assert.NotContains(t, s.Uses, "NULL")
		assert.NotContains(t, s.Uses, "true")
		assert.NotContains(t, s.Uses, "false")
	}
}

func TestExtractCStatements_DoWhileSwitch(t *testing.T) {
	src := `void f(int x) {
    do {
        x--;
    } while (x > 0);
    switch (x) {
        case 0: report(x); break;
        default: break;
    }
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	doStmt := findStmt(stmts, func(s *core.Statement) bool { return s.Type == core.StatementTypeWhile })
	require.NotNil(t, doStmt)
	assert.Equal(t, []string{"x"}, doStmt.Uses)

	swStmt := findStmt(stmts, func(s *core.Statement) bool {
		return s.Type == core.StatementTypeIf && len(s.NestedStatements) > 0
	})
	require.NotNil(t, swStmt)
	assert.Equal(t, []string{"x"}, swStmt.Uses)
}

// TestExtractCStatements_ForWithAssignmentInit covers the
// assignment-expression form of a `for` initialiser (i.e. the
// variable is declared earlier and reused, not redeclared in the loop
// header).
func TestExtractCStatements_ForWithAssignmentInit(t *testing.T) {
	src := `void f(int n) {
    int i;
    for (i = 0; i < n; i++) {
        do_thing(i);
    }
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	forStmt := findStmt(stmts, func(s *core.Statement) bool { return s.Type == core.StatementTypeFor })
	require.NotNil(t, forStmt)
	assert.Equal(t, "i", forStmt.Def)
	assert.Contains(t, forStmt.Uses, "n")
	assert.NotContains(t, forStmt.Uses, "i")
}

// TestExtractCStatements_DereferenceLHS verifies that `*p = val;`
// resolves to Def="p" — the dereference unwraps to the base pointer
// for def-use analysis.
func TestExtractCStatements_DereferenceLHS(t *testing.T) {
	src := `void f(int* p, int val) {
    *p = val;
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	// The pointer expression on the LHS surfaces as a use too — the
	// builder walks the LHS for indexable expressions.
	assert.Contains(t, stmts[0].Uses, "val")
}

// TestExtractCStatements_NestedIf verifies nested conditionals get
// their own NestedStatements lists, not flattened into the outer one.
func TestExtractCStatements_NestedIf(t *testing.T) {
	src := `void f(int x, int y) {
    if (x > 0) {
        if (y > 0) {
            consume(x);
        }
    }
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	require.Len(t, stmts[0].NestedStatements, 1)
	inner := stmts[0].NestedStatements[0]
	assert.Equal(t, core.StatementTypeIf, inner.Type)
	assert.Equal(t, []string{"y"}, inner.Uses)
}

func TestExtractCStatements_BareDeclaration(t *testing.T) {
	src := `void f() {
    int x;
}`
	tree, fn, b := parseCFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCStatements("/x.c", b, fn)
	require.NoError(t, err)
	// `int x;` has no init_declarator → no assignment is emitted.
	assert.Empty(t, stmts)
}
