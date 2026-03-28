package cfg

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parsePythonFunction(t *testing.T, source string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	require.NoError(t, err)
	// Find the function_definition node
	root := tree.RootNode()
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() == "function_definition" {
			return child
		}
	}
	t.Fatal("no function_definition found in source")
	return nil
}

func TestBuildCFG_LinearFunction(t *testing.T) {
	source := `def foo():
    x = source()
    y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have: entry, body block, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 3)

	// Count total statements across all blocks
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.Equal(t, 3, totalStmts, "should have 3 statements: assignment, assignment, call")

	// Verify paths: should be exactly 1 path from entry to exit
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 1)
}

func TestBuildCFG_IfElse(t *testing.T) {
	source := `def foo():
    x = source()
    if x:
        y = x
    else:
        y = "safe"
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have blocks for: entry, body, if_cond, if_true, if_false, if_merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 6)

	// Verify statements are extracted from BOTH branches
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// x=source(), condition, y=x (true), y="safe" (false), sink(y) (after merge)
	assert.GreaterOrEqual(t, totalStmts, 4, "should have statements from both branches")

	// Verify at least 2 paths (true and false branches)
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 2, "should have at least 2 paths through if/else")
}

func TestBuildCFG_IfNoBranch(t *testing.T) {
	source := `def foo():
    x = source()
    if x:
        y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should extract y=x from inside the if
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.GreaterOrEqual(t, totalStmts, 3, "should have: x=source, cond, y=x, sink")
	_ = blockStmts
}

func TestBuildCFG_ForLoop(t *testing.T) {
	source := `def foo():
    items = source()
    for item in items:
        sink(item)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have loop header and body blocks
	hasLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			hasLoop = true
			break
		}
	}
	assert.True(t, hasLoop, "should have a loop block")

	// Statement inside for body should be extracted
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// items=source(), for header (item in items), sink(item)
	assert.GreaterOrEqual(t, totalStmts, 3)
}

func TestBuildCFG_WhileLoop(t *testing.T) {
	source := `def foo():
    x = source()
    while x:
        sink(x)
        x = transform(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	hasLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			hasLoop = true
			break
		}
	}
	assert.True(t, hasLoop, "should have a while loop block")

	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// x=source(), while cond, sink(x), x=transform(x)
	assert.GreaterOrEqual(t, totalStmts, 4)
}

func TestBuildCFG_TryExcept(t *testing.T) {
	source := `def foo():
    try:
        x = source()
        sink(x)
    except ValueError:
        y = "safe"
        sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have try and catch blocks
	hasTry := false
	hasCatch := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeTry {
			hasTry = true
		}
		if block.Type == BlockTypeCatch {
			hasCatch = true
		}
	}
	assert.True(t, hasTry, "should have a try block")
	assert.True(t, hasCatch, "should have a catch block")

	// Statements from both try and except bodies should be extracted
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// x=source(), sink(x), y="safe", sink(y)
	assert.GreaterOrEqual(t, totalStmts, 4)
}

func TestBuildCFG_WithStatement(t *testing.T) {
	source := `def foo():
    with open(filename) as f:
        data = f.read()
        sink(data)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Statements inside with body should be extracted
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// with-var def (f), data=f.read(), sink(data)
	assert.GreaterOrEqual(t, totalStmts, 2, "should extract statements from with body")
}

func TestBuildCFG_NestedIfInFor(t *testing.T) {
	source := `def foo():
    items = source()
    for item in items:
        if item:
            sink(item)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have both loop and conditional blocks
	hasLoop := false
	hasCond := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			hasLoop = true
		}
		if block.Type == BlockTypeConditional {
			hasCond = true
		}
	}
	assert.True(t, hasLoop, "should have loop block")
	assert.True(t, hasCond, "should have conditional block inside loop")

	// sink(item) inside if inside for should be extracted
	totalStmts := 0
	foundSink := false
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
		for _, s := range stmts {
			if s.CallTarget == "sink" {
				foundSink = true
			}
		}
	}
	assert.True(t, foundSink, "should extract sink() call from inside nested control flow")
}

func TestBuildCFG_ReturnInMiddle(t *testing.T) {
	source := `def foo():
    x = source()
    if x:
        return x
    sink(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, _, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The return inside if should connect to exit
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 1, "should have paths through the function")
}

func TestBuildCFG_StatementsHaveLineNumbers(t *testing.T) {
	source := `def foo():
    x = source()
    y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	// All statements should have non-zero line numbers
	for _, stmts := range blockStmts {
		for _, s := range stmts {
			assert.Greater(t, s.LineNumber, uint32(0), "statement should have line number set")
		}
	}
}

func TestBuildCFG_BlockStatementsPreserveDefUse(t *testing.T) {
	source := `def foo():
    x = source()
    y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	// Collect all statements
	var allStmts []*struct {
		def  string
		uses []string
	}
	for _, stmts := range blockStmts {
		for _, s := range stmts {
			allStmts = append(allStmts, &struct {
				def  string
				uses []string
			}{s.Def, s.Uses})
		}
	}

	// Should have: x = source() (def=x), y = x (def=y, uses=[x]), sink(y) (uses=[y])
	foundXDef := false
	foundYDef := false
	foundSinkUse := false
	for _, s := range allStmts {
		if s.def == "x" {
			foundXDef = true
		}
		if s.def == "y" {
			foundYDef = true
			assert.Contains(t, s.uses, "x", "y=x should have x in uses")
		}
		if s.def == "" && len(s.uses) > 0 {
			for _, u := range s.uses {
				if u == "y" {
					foundSinkUse = true
				}
			}
		}
	}
	assert.True(t, foundXDef, "should have x definition")
	assert.True(t, foundYDef, "should have y definition")
	assert.True(t, foundSinkUse, "should have sink using y")
}

// ========== GAP-012: Subscript handling in CFG block statements ==========

func TestBuildCFG_SubscriptOnAttribute_SetsAttributeAccess(t *testing.T) {
	source := `def vuln(request):
    cmd = request.GET["cmd"]
    subprocess.run(cmd)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfGraph, blockStmts, err := BuildCFGFromAST("test.vuln", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfGraph)

	// Find the assignment statement and verify AttributeAccess is set
	foundAttrAccess := false
	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "cmd" {
				assert.Equal(t, "request.GET", stmt.AttributeAccess,
					"CFG builder must set AttributeAccess for subscript on attribute")
				assert.Contains(t, stmt.Uses, "request")
				foundAttrAccess = true
			}
		}
	}
	assert.True(t, foundAttrAccess, "should find cmd assignment with AttributeAccess")
}

func TestBuildCFG_SubscriptOnCall_UnmasksCallTarget(t *testing.T) {
	source := `def fetch(url):
    data = requests.get(url).json()["results"]
    process(data)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfGraph, blockStmts, err := BuildCFGFromAST("test.fetch", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfGraph)

	foundCall := false
	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "data" {
				assert.Equal(t, "json", stmt.CallTarget,
					"CFG builder must unmask call target through subscript")
				assert.Contains(t, stmt.Uses, "url")
				foundCall = true
			}
		}
	}
	assert.True(t, foundCall, "should find data assignment with unmasked CallTarget")
}

func TestBuildCFG_PureAttributeAccess_SetsAttributeAccess(t *testing.T) {
	source := `def handler(request):
    url = request.url
    fetch(url)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfGraph, blockStmts, err := BuildCFGFromAST("test.handler", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfGraph)

	foundAttr := false
	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "url" {
				assert.Equal(t, "request.url", stmt.AttributeAccess,
					"CFG builder must set AttributeAccess for pure attribute access")
				foundAttr = true
			}
		}
	}
	assert.True(t, foundAttr, "should find url assignment with AttributeAccess")
}

func TestBuildCFG_NestedSubscriptOnAttribute(t *testing.T) {
	source := `def handler(request):
    val = request.GET["a"]["b"]
    sink(val)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.handler", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "val" {
				assert.Equal(t, "request.GET", stmt.AttributeAccess,
					"Nested subscript should unwrap to innermost attribute chain")
				assert.Contains(t, stmt.Uses, "request")
			}
		}
	}
}

func TestBuildCFG_SubscriptOnPlainIdentifier(t *testing.T) {
	source := `def handler(data):
    val = data["key"]
    sink(val)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.handler", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "val" {
				assert.Equal(t, "", stmt.AttributeAccess,
					"Plain subscript should not set AttributeAccess")
				assert.Contains(t, stmt.Uses, "data")
			}
		}
	}
}

func TestBuildCFG_DeepAttributeChain(t *testing.T) {
	source := `def handler(app):
    val = app.config.SECRET_KEY
    sink(val)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.handler", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "val" {
				assert.Equal(t, "app.config.SECRET_KEY", stmt.AttributeAccess)
			}
		}
	}
}

func TestExtractFullAttributeChain_NilNode(t *testing.T) {
	assert.Equal(t, "", extractFullAttributeChain(nil, []byte("")))
}

func TestExtractCallTarget_NilNode_CFG(t *testing.T) {
	target, chain := extractCallTarget(nil, []byte(""))
	assert.Equal(t, "", target)
	assert.Equal(t, "", chain)
}

// ========== CALL CHAIN TESTS (GAP-004) ==========

func TestBuildCFG_CallChain_MethodCall(t *testing.T) {
	source := `def handler(request):
    query = request.args.get("q")
    process(query)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.handler", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "query" {
				assert.Equal(t, "request.args.get", stmt.CallChain,
					"CFG builder must extract full call chain")
			}
		}
	}
}

func TestBuildCFG_CallChain_ThreeLevel(t *testing.T) {
	source := `def reconnect(self):
    script = self.pyload.config.get("script")
    run(script)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.reconnect", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "script" {
				assert.Equal(t, "self.pyload.config.get", stmt.CallChain)
			}
		}
	}
}

func TestBuildCFG_CallChain_SimpleCall(t *testing.T) {
	source := `def foo():
    x = bar()
    sink(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "x" {
				assert.Equal(t, "bar", stmt.CallChain, "Simple call: chain equals target")
			}
		}
	}
}

func TestBuildCFG_AugmentedAssignment(t *testing.T) {
	source := `def foo():
    x = 10
    x += 5
    sink(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	totalStmts := 0
	foundAugmented := false
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
		for _, stmt := range stmts {
			if stmt.Def == "x" && len(stmt.Uses) > 0 {
				for _, u := range stmt.Uses {
					if u == "x" {
						foundAugmented = true
					}
				}
			}
		}
	}
	assert.True(t, foundAugmented, "should find augmented assignment x += 5")
	assert.GreaterOrEqual(t, totalStmts, 3, "should have at least 3 statements")
}

func TestBuildCFG_BareCallStatement(t *testing.T) {
	source := `def foo():
    print("hello")
    subprocess.run(cmd)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	foundBareCall := false
	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.CallTarget == "run" && stmt.CallChain == "subprocess.run" {
				foundBareCall = true
			}
		}
	}
	assert.True(t, foundBareCall, "should find bare call subprocess.run with chain")
}

func TestBuildCFG_LambdaCall(t *testing.T) {
	// Lambda call has a complex expression as function node (not identifier or attribute)
	source := `def foo():
    x = (lambda y: y)(10)
    sink(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			if stmt.Def == "x" {
				assert.NotEmpty(t, stmt.CallChain, "Lambda call should have non-empty chain")
			}
		}
	}
}
