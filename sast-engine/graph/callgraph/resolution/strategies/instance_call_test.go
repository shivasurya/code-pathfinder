package strategies

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TEST TYPE STORE
// =============================================================================

type testTypeStore struct {
	bindings map[string]core.Type
}

func newTestTypeStore() *testTypeStore {
	return &testTypeStore{
		bindings: make(map[string]core.Type),
	}
}

func (s *testTypeStore) Lookup(varName string) core.Type {
	return s.bindings[varName]
}

func (s *testTypeStore) CurrentScopeDepth() int {
	return 1
}

func (s *testTypeStore) Set(varName string, typ core.Type) {
	s.bindings[varName] = typ
}

// =============================================================================
// MOCK REGISTRIES
// =============================================================================

type mockAttrRegistry struct {
	classes map[string]*core.ClassAttributes
}

func (m *mockAttrRegistry) GetClassAttributes(fqn string) *core.ClassAttributes {
	return m.classes[fqn]
}

func (m *mockAttrRegistry) GetAttribute(fqn, attr string) *core.ClassAttribute {
	if ca := m.classes[fqn]; ca != nil {
		return ca.Attributes[attr]
	}
	return nil
}

func (m *mockAttrRegistry) HasClass(fqn string) bool {
	_, found := m.classes[fqn]
	return found
}

type mockBuiltinReg struct{}

func (m *mockBuiltinReg) GetMethodReturnType(typeFQN, method string) (string, bool) {
	if typeFQN == "builtins.str" && method == "upper" {
		return "builtins.str", true
	}
	if typeFQN == "builtins.str" && method == "split" {
		return "builtins.list", true
	}
	return "", false
}

func (m *mockBuiltinReg) IsBuiltinType(fqn string) bool {
	return fqn == "builtins.str" || fqn == "builtins.int" || fqn == "builtins.list"
}

type mockModuleReg struct{}

func (m *mockModuleReg) GetModulePath(fp string) string           { return "test" }
func (m *mockModuleReg) ResolveImport(i, f string) (string, bool) { return i, true }

// =============================================================================
// HELPER
// =============================================================================

func parseCode(t *testing.T, code string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	require.NoError(t, err)
	return tree.RootNode()
}

func findCallNode(root *sitter.Node) *sitter.Node {
	// DFS to find first call node
	var find func(*sitter.Node) *sitter.Node
	find = func(n *sitter.Node) *sitter.Node {
		if n.Type() == "call" {
			return n
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			if result := find(n.Child(i)); result != nil {
				return result
			}
		}
		return nil
	}
	return find(root)
}

func createContext(store TypeStore, code []byte) *InferenceContext {
	return &InferenceContext{
		SourceCode:      code,
		FilePath:        "test.py",
		Store:           store,
		SelfType:        nil,
		ClassFQN:        "",
		AttrRegistry:    &mockAttrRegistry{classes: make(map[string]*core.ClassAttributes)},
		ModuleRegistry:  &mockModuleReg{},
		BuiltinRegistry: &mockBuiltinReg{},
	}
}

// =============================================================================
// TESTS
// =============================================================================

func TestInstanceCallStrategy_CanHandle(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"method call", "obj.method()", true},
		{"nested call", "obj.a.method()", true},
		{"simple call", "func()", false},
		{"literal", `"hello"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseCode(t, tt.code)
			callNode := findCallNode(root)
			ctx := createContext(store, []byte(tt.code))

			if callNode == nil {
				assert.False(t, tt.expected)
			} else {
				assert.Equal(t, tt.expected, s.CanHandle(callNode, ctx))
			}
		})
	}
}

func TestInstanceCallStrategy_SynthesizeWithKnownVariable(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Set up: a = A() where A has method 'something'
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.A": {
				ClassFQN:   "myapp.A",
				Methods:    []string{"myapp.A.something"},
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	store.Set("a", core.NewConcreteType("myapp.A", 0.95))

	code := `a.something()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:      []byte(code),
		FilePath:        "test.py",
		Store:           store,
		AttrRegistry:    attrReg,
		BuiltinRegistry: &mockBuiltinReg{},
	}

	typ, conf := s.Synthesize(callNode, ctx)

	// Method found, uses fluent heuristic
	assert.Equal(t, "myapp.A", typ.FQN())
	assert.Greater(t, conf, 0.5)
}

func TestInstanceCallStrategy_SynthesizeBuiltinMethod(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Set up: name = "hello"
	store.Set("name", core.NewConcreteType("builtins.str", 0.95))

	code := `name.upper()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(callNode, ctx)

	assert.Equal(t, "builtins.str", typ.FQN())
	assert.InDelta(t, 0.855, conf, 0.01) // 0.95 * 0.9
}

func TestInstanceCallStrategy_SynthesizeSelfMethod(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	selfType := core.NewConcreteType("myapp.Handler", 0.95)

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Handler": {
				ClassFQN:   "myapp.Handler",
				Methods:    []string{"myapp.Handler.validate"},
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	code := `self.validate()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:      []byte(code),
		FilePath:        "test.py",
		Store:           store,
		SelfType:        selfType,
		ClassFQN:        "myapp.Handler",
		AttrRegistry:    attrReg,
		BuiltinRegistry: &mockBuiltinReg{},
	}

	typ, conf := s.Synthesize(callNode, ctx)

	assert.Equal(t, "myapp.Handler", typ.FQN())
	assert.Greater(t, conf, 0.5)
}

func TestInstanceCallStrategy_UnknownVariable(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	code := `unknown.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(callNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestInstanceCallStrategy_Priority(t *testing.T) {
	s := NewInstanceCallStrategy()
	assert.Equal(t, 80, s.Priority())
}

func TestInstanceCallStrategy_Name(t *testing.T) {
	s := NewInstanceCallStrategy()
	assert.Equal(t, "instance_call", s.Name())
}

func TestInstanceCallStrategy_CanHandleNilNode(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()
	ctx := createContext(store, []byte(""))

	assert.False(t, s.CanHandle(nil, ctx))
}

func TestInstanceCallStrategy_MethodNotFound(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	// Set up: a = A() but A doesn't have method 'missing'
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.A": {
				ClassFQN:   "myapp.A",
				Methods:    []string{"myapp.A.existing"},
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	store.Set("a", core.NewConcreteType("myapp.A", 0.95))

	code := `a.missing()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:      []byte(code),
		FilePath:        "test.py",
		Store:           store,
		AttrRegistry:    attrReg,
		BuiltinRegistry: &mockBuiltinReg{},
	}

	typ, conf := s.Synthesize(callNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestInstanceCallStrategy_Check(t *testing.T) {
	s := NewInstanceCallStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.A": {
				ClassFQN:   "myapp.A",
				Methods:    []string{"myapp.A.something"},
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	store.Set("a", core.NewConcreteType("myapp.A", 0.95))

	code := `a.something()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:      []byte(code),
		FilePath:        "test.py",
		Store:           store,
		AttrRegistry:    attrReg,
		BuiltinRegistry: &mockBuiltinReg{},
	}

	// Check with correct expected type
	expectedType := core.NewConcreteType("myapp.A", 0.9)
	assert.True(t, s.Check(callNode, expectedType, ctx))

	// Check with wrong expected type
	wrongType := core.NewConcreteType("myapp.B", 0.9)
	assert.False(t, s.Check(callNode, wrongType, ctx))
}
