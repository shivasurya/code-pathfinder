package strategies

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// BASE STRATEGY TESTS
// =============================================================================

func TestNewBaseStrategy(t *testing.T) {
	bs := NewBaseStrategy("test", 50)

	assert.Equal(t, "test", bs.Name())
	assert.Equal(t, 50, bs.Priority())
}

// =============================================================================
// STRATEGY REGISTRY TESTS
// =============================================================================

type mockTestStrategy struct {
	BaseStrategy
	handlesNode bool
	returnType  core.Type
	returnConf  float64
}

func (s *mockTestStrategy) CanHandle(node *sitter.Node, ctx *InferenceContext) bool {
	return s.handlesNode
}

func (s *mockTestStrategy) Synthesize(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	return s.returnType, s.returnConf
}

func (s *mockTestStrategy) Check(node *sitter.Node, expected core.Type, ctx *InferenceContext) bool {
	typ, _ := s.Synthesize(node, ctx)
	return typ.Equals(expected)
}

func TestStrategyRegistry_NewStrategyRegistry(t *testing.T) {
	reg := NewStrategyRegistry()

	assert.NotNil(t, reg)
	assert.Empty(t, reg.GetStrategies())
}

func TestStrategyRegistry_Register(t *testing.T) {
	reg := NewStrategyRegistry()

	strategy := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("test", 50),
		handlesNode:  true,
	}

	reg.Register(strategy)

	strategies := reg.GetStrategies()
	assert.Len(t, strategies, 1)
	assert.Equal(t, "test", strategies[0].Name())
}

func TestStrategyRegistry_SortByPriority(t *testing.T) {
	reg := NewStrategyRegistry()

	low := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("low", 10),
	}
	mid := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("mid", 50),
	}
	high := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("high", 100),
	}

	// Register in random order
	reg.Register(mid)
	reg.Register(low)
	reg.Register(high)

	strategies := reg.GetStrategies()
	require.Len(t, strategies, 3)

	// Should be sorted by priority descending
	assert.Equal(t, "high", strategies[0].Name())
	assert.Equal(t, "mid", strategies[1].Name())
	assert.Equal(t, "low", strategies[2].Name())
}

func TestStrategyRegistry_FindStrategy(t *testing.T) {
	reg := NewStrategyRegistry()

	cannotHandle := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("cannot", 50),
		handlesNode:  false,
	}
	canHandle := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("can", 100),
		handlesNode:  true,
	}

	reg.Register(cannotHandle)
	reg.Register(canHandle)

	store := newTestTypeStore()
	ctx := &InferenceContext{
		Store: store,
	}

	found := reg.FindStrategy(nil, ctx)

	assert.NotNil(t, found)
	assert.Equal(t, "can", found.Name())
}

func TestStrategyRegistry_FindStrategy_NotFound(t *testing.T) {
	reg := NewStrategyRegistry()

	cannotHandle := &mockTestStrategy{
		BaseStrategy: NewBaseStrategy("cannot", 50),
		handlesNode:  false,
	}

	reg.Register(cannotHandle)

	store := newTestTypeStore()
	ctx := &InferenceContext{
		Store: store,
	}

	found := reg.FindStrategy(nil, ctx)

	assert.Nil(t, found)
}

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestGetNodeText(t *testing.T) {
	code := `hello`
	root := parseCode(t, code)
	identNode := root.Child(0).Child(0) // expression_statement -> identifier

	text := GetNodeText(identNode, []byte(code))
	assert.Equal(t, "hello", text)
}

func TestGetNodeText_NilNode(t *testing.T) {
	text := GetNodeText(nil, []byte("test"))
	assert.Equal(t, "", text)
}

func TestGetChildByType(t *testing.T) {
	code := `a.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	// Find attribute child
	attrNode := GetChildByType(callNode, "attribute")
	assert.NotNil(t, attrNode)
	assert.Equal(t, "attribute", attrNode.Type())
}

func TestGetChildByType_NotFound(t *testing.T) {
	code := `hello`
	root := parseCode(t, code)

	child := GetChildByType(root, "nonexistent")
	assert.Nil(t, child)
}

func TestGetChildByType_NilNode(t *testing.T) {
	child := GetChildByType(nil, "any")
	assert.Nil(t, child)
}

func TestGetChildrenByType(t *testing.T) {
	code := `
def foo(a, b, c):
    pass
`
	root := parseCode(t, code)

	// Find all identifier nodes in parameters
	funcDef := root.Child(0) // function_definition
	params := funcDef.ChildByFieldName("parameters")

	identifiers := GetChildrenByType(params, "identifier")
	assert.Len(t, identifiers, 3) // a, b, c
}

func TestGetChildrenByType_NoneFound(t *testing.T) {
	code := `hello`
	root := parseCode(t, code)

	children := GetChildrenByType(root, "nonexistent")
	assert.Empty(t, children)
}

func TestGetChildrenByType_NilNode(t *testing.T) {
	children := GetChildrenByType(nil, "any")
	assert.Empty(t, children)
}

// =============================================================================
// ADDITIONAL EDGE CASES FOR STRATEGIES
// =============================================================================

func TestInstanceCallStrategy_SynthesizeNoAttributeNode(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Call without attribute (simple call)
	code := `func()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(callNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestInstanceCallStrategy_inferReceiverType_CallNode(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Nested call like: get_user().process()
	// For now this returns Any with "chain" reason
	code := `get_user().process()`
	root := parseCode(t, code)
	// We need the inner call node
	callNode := findCallNode(root)
	attrNode := GetChildByType(callNode, "attribute")
	receiverNode := attrNode.ChildByFieldName("object") // This is the get_user() call

	ctx := createContext(store, []byte(code))

	typ, conf := s.inferReceiverType(receiverNode, ctx)

	// Since it's a call and we recursively call Synthesize, which has no attribute, it returns Any
	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestInstanceCallStrategy_inferReceiverType_AttributeNode(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Chained like: obj.attr.method()
	code := `obj.attr.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)
	attrNode := GetChildByType(callNode, "attribute")
	receiverNode := attrNode.ChildByFieldName("object") // This is obj.attr (attribute node)

	ctx := createContext(store, []byte(code))

	typ, conf := s.inferReceiverType(receiverNode, ctx)

	// Returns Any with "chain - defer to ChainStrategy"
	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestInstanceCallStrategy_CheckLowConfidence(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Unknown variable has 0 confidence
	code := `unknown.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	// Even with "correct" type, should fail due to low confidence
	result := s.Check(callNode, core.NewConcreteType("any", 0.9), ctx)
	assert.False(t, result)
}

func TestAttributeAccessStrategy_SynthesizeCallObject(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	// get_user().name
	code := `get_user().name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode: []byte(code),
		Store:      store,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	// Call result returns Any with reason
	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestSelfReferenceStrategy_SelfNotConcreteType(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()

	// Set self type to AnyType (not concrete)
	selfType := &core.AnyType{Reason: "test"}

	code := `self.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode: []byte(code),
		Store:      store,
		SelfType:   selfType,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestSelfReferenceStrategy_NoAttrRegistry(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

	code := `self.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		SelfType:     selfType,
		AttrRegistry: nil, // No registry
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}
