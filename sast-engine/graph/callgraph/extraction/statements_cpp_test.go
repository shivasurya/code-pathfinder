package extraction

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	cpplang "github.com/smacker/go-tree-sitter/cpp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// parseCppFunction parses C++ source and returns the function node
// named `testFuncName`. The caller must close the tree.
func parseCppFunction(t *testing.T, source string) (*sitter.Tree, *sitter.Node, []byte) {
	t.Helper()
	src := []byte(source)

	parser := sitter.NewParser()
	parser.SetLanguage(cpplang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	require.NoError(t, err)

	fn := findCppFunction(tree.RootNode(), testFuncName, src)
	require.NotNil(t, fn, "function %q not found", testFuncName)
	return tree, fn, src
}

func findCppFunction(node *sitter.Node, name string, src []byte) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "function_definition" {
		if d := node.ChildByFieldName("declarator"); d != nil && testCFunctionName(d, src) == name {
			return node
		}
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if r := findCppFunction(node.Child(i), name, src); r != nil {
			return r
		}
	}
	return nil
}

func TestExtractCppStatements_NilFunction(t *testing.T) {
	stmts, err := ExtractCppStatements("/x.cpp", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, stmts)
}

func TestExtractCppStatements_MethodCallOnObject(t *testing.T) {
	src := `void f(Obj obj, int x) {
    obj.method(x);
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, core.StatementTypeCall, stmts[0].Type)
	assert.Equal(t, "method", stmts[0].CallTarget)
	assert.Equal(t, "obj.method", stmts[0].CallChain)
	assert.ElementsMatch(t, []string{"obj", "x"}, stmts[0].Uses)
}

func TestExtractCppStatements_QualifiedCall(t *testing.T) {
	src := `void f(int* begin, int* end) {
    std::sort(begin, end);
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, "std::sort", stmts[0].CallTarget)
	assert.Equal(t, "std::sort", stmts[0].CallChain)
	assert.ElementsMatch(t, []string{"begin", "end"}, stmts[0].Uses)
}

func TestExtractCppStatements_AutoFromMethodCall(t *testing.T) {
	src := `void f(Obj obj) {
    auto x = obj.get();
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, core.StatementTypeAssignment, stmts[0].Type)
	assert.Equal(t, "x", stmts[0].Def)
	assert.Equal(t, []string{"obj"}, stmts[0].Uses)
	assert.Equal(t, "get", stmts[0].CallTarget)
}

func TestExtractCppStatements_ThrowConstructor(t *testing.T) {
	src := `void f() {
    throw std::runtime_error("msg");
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, core.StatementTypeRaise, stmts[0].Type)
	assert.Equal(t, "std::runtime_error", stmts[0].CallTarget)
}

func TestExtractCppStatements_TryCatch(t *testing.T) {
	src := `void f() {
    try {
        risky();
    } catch (const std::exception& e) {
        log(e);
    }
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	tryStmt := stmts[0]
	assert.Equal(t, core.StatementTypeTry, tryStmt.Type)
	require.NotEmpty(t, tryStmt.NestedStatements)
	assert.Equal(t, "risky", tryStmt.NestedStatements[0].CallTarget)
	require.NotEmpty(t, tryStmt.ElseBranch)
	// First catch element binds the exception name.
	assert.Equal(t, "e", tryStmt.ElseBranch[0].Def)
	logStmt := findStmt(tryStmt.ElseBranch, func(s *core.Statement) bool {
		return s.Type == core.StatementTypeCall && s.CallTarget == "log"
	})
	require.NotNil(t, logStmt)
	assert.Equal(t, []string{"e"}, logStmt.Uses)
}

func TestExtractCppStatements_RangeBasedFor(t *testing.T) {
	src := `void f(std::vector<int> items) {
    for (auto x : items) {
        consume(x);
    }
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	forStmt := findStmt(stmts, func(s *core.Statement) bool { return s.Type == core.StatementTypeFor })
	require.NotNil(t, forStmt)
	assert.Equal(t, "x", forStmt.Def)
	assert.Equal(t, []string{"items"}, forStmt.Uses)
}

func TestExtractCppStatements_KeywordFilterCpp(t *testing.T) {
	src := `void f(Obj* p) {
    auto* q = static_cast<Obj*>(p);
    if (q == nullptr) return;
    delete q;
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	for _, s := range stmts {
		assert.NotContains(t, s.Uses, "nullptr")
		assert.NotContains(t, s.Uses, "static_cast")
		assert.NotContains(t, s.Uses, "delete")
		assert.NotContains(t, s.Uses, "this")
		assert.NotContains(t, s.Uses, "auto")
	}
}

func TestExtractCppStatements_FallthroughToCBuilder(t *testing.T) {
	src := `int f(int a, int b) {
    int x = a + b;
    return x;
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 2)
	assert.Equal(t, "x", stmts[0].Def)
	assert.ElementsMatch(t, []string{"a", "b"}, stmts[0].Uses)
}

func TestExtractCppStatements_NamespaceAssignment(t *testing.T) {
	src := `void f() {
    auto v = std::make_unique();
}`
	tree, fn, b := parseCppFunction(t, src)
	defer tree.Close()
	stmts, err := ExtractCppStatements("/x.cpp", b, fn)
	require.NoError(t, err)

	require.Len(t, stmts, 1)
	assert.Equal(t, "v", stmts[0].Def)
	assert.Equal(t, "std::make_unique", stmts[0].CallTarget)
}
