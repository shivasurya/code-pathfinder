package resolution

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK REGISTRIES FOR TESTING
// =============================================================================

type mockAttributeRegistry struct {
	classes map[string]*core.ClassAttributes
}

func (m *mockAttributeRegistry) GetClassAttributes(classFQN string) *core.ClassAttributes {
	return m.classes[classFQN]
}

func (m *mockAttributeRegistry) GetAttribute(classFQN, attrName string) *core.ClassAttribute {
	if ca := m.classes[classFQN]; ca != nil {
		return ca.Attributes[attrName]
	}
	return nil
}

func (m *mockAttributeRegistry) HasClass(classFQN string) bool {
	_, found := m.classes[classFQN]
	return found
}

type mockModuleRegistry struct{}

func (m *mockModuleRegistry) GetModulePath(filePath string) string {
	return "test_module"
}

func (m *mockModuleRegistry) ResolveImport(importPath, fromFile string) (string, bool) {
	return importPath, true
}

type mockBuiltinRegistry struct{}

func (m *mockBuiltinRegistry) GetMethodReturnType(typeFQN, methodName string) (string, bool) {
	if typeFQN == "builtins.str" && methodName == "upper" {
		return "builtins.str", true
	}
	return "", false
}

func (m *mockBuiltinRegistry) IsBuiltinType(typeFQN string) bool {
	return typeFQN == "builtins.str" || typeFQN == "builtins.int"
}

// =============================================================================
// MOCK STRATEGY FOR TESTING
// =============================================================================

type mockStrategy struct {
	strategies.BaseStrategy
	canHandleFunc  func(*sitter.Node) bool
	synthesizeFunc func(*sitter.Node) (core.Type, float64)
}

func (s *mockStrategy) CanHandle(node *sitter.Node, ctx *strategies.InferenceContext) bool {
	return s.canHandleFunc(node)
}

func (s *mockStrategy) Synthesize(node *sitter.Node, ctx *strategies.InferenceContext) (core.Type, float64) {
	return s.synthesizeFunc(node)
}

func (s *mockStrategy) Check(node *sitter.Node, expected core.Type, ctx *strategies.InferenceContext) bool {
	typ, _ := s.Synthesize(node, ctx)
	return typ.Equals(expected)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func parseCode(t *testing.T, code string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	require.NoError(t, err)
	return tree.RootNode()
}

func createTestInferencer() *BidirectionalInferencer {
	return NewBidirectionalInferencer(
		&mockAttributeRegistry{classes: make(map[string]*core.ClassAttributes)},
		&mockModuleRegistry{},
		&mockBuiltinRegistry{},
		1000,
	)
}

// =============================================================================
// TESTS
// =============================================================================

func TestBidirectionalInferencer_Creation(t *testing.T) {
	bi := createTestInferencer()
	assert.NotNil(t, bi)
	assert.NotNil(t, bi.strategies)
	assert.NotNil(t, bi.cache)
}

func TestBidirectionalInferencer_RegisterStrategy(t *testing.T) {
	bi := createTestInferencer()

	strategy := &mockStrategy{
		BaseStrategy:  strategies.NewBaseStrategy("test", 100),
		canHandleFunc: func(n *sitter.Node) bool { return n.Type() == "string" },
		synthesizeFunc: func(n *sitter.Node) (core.Type, float64) {
			return core.NewConcreteType("builtins.str", 0.95), 0.95
		},
	}

	bi.RegisterStrategy(strategy)

	// Verify by inferring a string
	code := `"hello"`
	root := parseCode(t, code)
	stringNode := root.Child(0).Child(0) // expression_statement -> string

	store := NewTypeStore()
	typ, conf := bi.InferType(stringNode, store, []byte(code), "test.py", nil, "", "")

	assert.True(t, core.IsConcreteType(typ))
	assert.Equal(t, 0.95, conf)
}

func TestBidirectionalInferencer_FallbackLiterals(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	tests := []struct {
		code     string
		expected string
	}{
		{`"hello"`, "builtins.str"},
		{`42`, "builtins.int"},
		{`3.14`, "builtins.float"},
		{`True`, "builtins.bool"},
		{`None`, "builtins.NoneType"},
		{`[1, 2, 3]`, "builtins.list"},
		{`{"a": 1}`, "builtins.dict"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			root := parseCode(t, tt.code)
			exprNode := root.Child(0).Child(0) // expression_statement -> literal

			typ, _ := bi.InferType(exprNode, store, []byte(tt.code), "test.py", nil, "", "")

			assert.Equal(t, tt.expected, typ.FQN(), "Code: %s", tt.code)
		})
	}
}

func TestBidirectionalInferencer_VariableLookup(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	// Set up a variable
	store.Set("my_var", core.NewConcreteType("myapp.User", 0.9), core.ConfidenceAssignment, "test.py", 1, 0)

	code := `my_var`
	root := parseCode(t, code)
	identNode := root.Child(0).Child(0) // expression_statement -> identifier

	typ, conf := bi.InferType(identNode, store, []byte(code), "test.py", nil, "", "")

	assert.Equal(t, "myapp.User", typ.FQN())
	assert.Equal(t, 0.9, conf)
}

func TestBidirectionalInferencer_UnboundVariable(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	code := `undefined_var`
	root := parseCode(t, code)
	identNode := root.Child(0).Child(0)

	typ, conf := bi.InferType(identNode, store, []byte(code), "test.py", nil, "", "")

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestBidirectionalInferencer_Caching(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	code := `"hello"`
	root := parseCode(t, code)
	stringNode := root.Child(0).Child(0)

	// First call
	bi.InferType(stringNode, store, []byte(code), "test.py", nil, "", "")

	// Second call (should hit cache)
	bi.InferType(stringNode, store, []byte(code), "test.py", nil, "", "")

	hits, misses, _ := bi.CacheStats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(1), misses)
}

func TestBidirectionalInferencer_InvalidateFile(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	code := `"hello"`
	root := parseCode(t, code)
	stringNode := root.Child(0).Child(0)

	bi.InferType(stringNode, store, []byte(code), "test.py", nil, "", "")

	count := bi.InvalidateFile("test.py")
	assert.Equal(t, 1, count)

	// Next call should be a miss (cache was invalidated)
	bi.InferType(stringNode, store, []byte(code), "test.py", nil, "", "")

	hits, misses, _ := bi.CacheStats()
	assert.Equal(t, int64(0), hits)  // No hits (first was miss, second was miss after invalidation)
	assert.Equal(t, int64(2), misses) // Initial + after invalidation
}

func TestBidirectionalInferencer_NilNode(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	typ, conf := bi.InferType(nil, store, []byte(""), "test.py", nil, "", "")

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestBidirectionalInferencer_CheckType(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	strategy := &mockStrategy{
		BaseStrategy:  strategies.NewBaseStrategy("test", 100),
		canHandleFunc: func(n *sitter.Node) bool { return n.Type() == "string" },
		synthesizeFunc: func(n *sitter.Node) (core.Type, float64) {
			return core.NewConcreteType("builtins.str", 0.95), 0.95
		},
	}
	bi.RegisterStrategy(strategy)

	code := `"hello"`
	root := parseCode(t, code)
	stringNode := root.Child(0).Child(0)

	// Check against correct type
	assert.True(t, bi.CheckType(stringNode, core.NewConcreteType("builtins.str", 0.9), store, []byte(code), "test.py", nil, ""))

	// Check against wrong type
	assert.False(t, bi.CheckType(stringNode, core.NewConcreteType("builtins.int", 0.9), store, []byte(code), "test.py", nil, ""))
}

func TestBidirectionalInferencer_StrategyPriority(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	// Low priority strategy
	lowPriority := &mockStrategy{
		BaseStrategy:  strategies.NewBaseStrategy("low", 10),
		canHandleFunc: func(n *sitter.Node) bool { return n.Type() == "identifier" },
		synthesizeFunc: func(n *sitter.Node) (core.Type, float64) {
			return core.NewConcreteType("low.Type", 0.5), 0.5
		},
	}

	// High priority strategy
	highPriority := &mockStrategy{
		BaseStrategy:  strategies.NewBaseStrategy("high", 100),
		canHandleFunc: func(n *sitter.Node) bool { return n.Type() == "identifier" },
		synthesizeFunc: func(n *sitter.Node) (core.Type, float64) {
			return core.NewConcreteType("high.Type", 0.9), 0.9
		},
	}

	// Register in opposite order
	bi.RegisterStrategy(lowPriority)
	bi.RegisterStrategy(highPriority)

	code := `my_var`
	root := parseCode(t, code)
	identNode := root.Child(0).Child(0)

	typ, _ := bi.InferType(identNode, store, []byte(code), "test.py", nil, "", "")

	// High priority should win
	assert.Equal(t, "high.Type", typ.FQN())
}

// =============================================================================
// INTEGRATION TESTS (with TypeStore)
// =============================================================================

func TestBidirectionalInferencer_WithNestedScopes(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	// Global variable
	store.Set("global_var", core.NewConcreteType("GlobalType", 0.9), core.ConfidenceAssignment, "test.py", 1, 0)

	// Enter function scope
	store.PushScope("function")
	store.Set("local_var", core.NewConcreteType("LocalType", 0.9), core.ConfidenceAssignment, "test.py", 5, 0)

	// Parse code with both variables on different lines to avoid cache collision
	combinedCode := "global_var\nlocal_var"
	root := parseCode(t, combinedCode)

	// First identifier (global_var)
	globalNode := root.Child(0).Child(0)
	// Second identifier (local_var)
	localNode := root.Child(1).Child(0)

	globalTyp, _ := bi.InferType(globalNode, store, []byte(combinedCode), "test.py", nil, "", "")
	localTyp, _ := bi.InferType(localNode, store, []byte(combinedCode), "test.py", nil, "", "")

	assert.Equal(t, "GlobalType", globalTyp.FQN())
	assert.Equal(t, "LocalType", localTyp.FQN())

	// Exit function scope
	store.PopScope()

	// Invalidate cache to ensure fresh lookup after scope change
	bi.InvalidateFile("test.py")

	// Local should no longer resolve (returns Any because not in scope)
	localTyp2, _ := bi.InferType(localNode, store, []byte(combinedCode), "test.py", nil, "", "")
	assert.True(t, core.IsAnyType(localTyp2))
}

func TestBidirectionalInferencer_CheckTypeNilInputs(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	// Nil node
	assert.False(t, bi.CheckType(nil, core.NewConcreteType("str", 0.9), store, []byte(""), "test.py", nil, ""))

	// Nil expected type
	code := `"hello"`
	root := parseCode(t, code)
	stringNode := root.Child(0).Child(0)
	assert.False(t, bi.CheckType(stringNode, nil, store, []byte(code), "test.py", nil, ""))
}

func TestBidirectionalInferencer_CheckTypeNoStrategy(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	code := `"hello"`
	root := parseCode(t, code)
	stringNode := root.Child(0).Child(0)

	// No strategy registered, should return false
	assert.False(t, bi.CheckType(stringNode, core.NewConcreteType("str", 0.9), store, []byte(code), "test.py", nil, ""))
}

func TestBidirectionalInferencer_FallbackEdgeCases(t *testing.T) {
	bi := createTestInferencer()
	store := NewTypeStore()

	tests := []struct {
		code     string
		expected string
		isAny    bool
	}{
		{`{1, 2, 3}`, "builtins.set", false},
		{`(1, 2, 3)`, "builtins.tuple", false},
		{`unknown_node_type`, "", true}, // Will be "Any" for unknown node type
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			root := parseCode(t, tt.code)
			exprNode := root.Child(0).Child(0)

			typ, _ := bi.InferType(exprNode, store, []byte(tt.code), "test.py", nil, "", "")

			if tt.isAny {
				assert.True(t, core.IsAnyType(typ), "Expected Any type for: %s", tt.code)
			} else {
				assert.Equal(t, tt.expected, typ.FQN(), "Code: %s", tt.code)
			}
		})
	}
}
