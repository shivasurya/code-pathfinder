package strategies

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestAttributeAccessStrategy_CanHandle(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	tests := []struct {
		name     string
		code     string
		selfType core.Type
		expected bool
	}{
		{"obj.attr", "obj.name", nil, true},
		{"obj.attr with self context", "obj.name", core.NewConcreteType("A", 0.9), true},
		{"self.attr", "self.name", core.NewConcreteType("A", 0.9), false}, // Handled by SelfReference
		{"not attribute", `"hello"`, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseCode(t, tt.code)
			attrNode := findAttributeNode(root)

			ctx := &InferenceContext{
				SourceCode: []byte(tt.code),
				Store:      store,
				SelfType:   tt.selfType,
			}

			if attrNode == nil {
				assert.False(t, tt.expected)
			} else {
				assert.Equal(t, tt.expected, s.CanHandle(attrNode, ctx))
			}
		})
	}
}

func TestAttributeAccessStrategy_SynthesizeKnownObject(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	// Set up: service = Service() where Service has 'name' attribute
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Attributes: map[string]*core.ClassAttribute{
					"name": {
						Name: "name",
						Type: &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 0.9},
					},
				},
			},
		},
	}

	store.Set("service", core.NewConcreteType("myapp.Service", 0.95))

	code := `service.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.Equal(t, "builtins.str", typ.FQN())
	assert.InDelta(t, 0.855, conf, 0.01) // 0.95 * 0.9
}

func TestAttributeAccessStrategy_UnknownObject(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	code := `unknown.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode: []byte(code),
		Store:      store,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestAttributeAccessStrategy_ChainedAttribute(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	// Set up: obj.a.b where obj.a is known
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Outer": {
				ClassFQN: "myapp.Outer",
				Attributes: map[string]*core.ClassAttribute{
					"inner": {
						Name: "inner",
						Type: &core.TypeInfo{TypeFQN: "myapp.Inner", Confidence: 0.9},
					},
				},
			},
			"myapp.Inner": {
				ClassFQN: "myapp.Inner",
				Attributes: map[string]*core.ClassAttribute{
					"value": {
						Name: "value",
						Type: &core.TypeInfo{TypeFQN: "builtins.int", Confidence: 0.9},
					},
				},
			},
		},
	}

	store.Set("obj", core.NewConcreteType("myapp.Outer", 0.95))

	code := `obj.inner.value`
	root := parseCode(t, code)

	// Find the outermost attribute (obj.inner.value)
	exprNode := root.Child(0).Child(0) // expression_statement -> attribute

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(exprNode, ctx)

	assert.Equal(t, "builtins.int", typ.FQN())
	assert.Greater(t, conf, 0.0)
}

func TestAttributeAccessStrategy_Priority(t *testing.T) {
	s := NewAttributeAccessStrategy()
	// Lower than self_reference (90)
	assert.Equal(t, 70, s.Priority())
}

func TestAttributeAccessStrategy_Name(t *testing.T) {
	s := NewAttributeAccessStrategy()
	assert.Equal(t, "attribute_access", s.Name())
}

func TestAttributeAccessStrategy_Check(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Attributes: map[string]*core.ClassAttribute{
					"name": {
						Name: "name",
						Type: &core.TypeInfo{TypeFQN: "builtins.str", Confidence: 0.9},
					},
				},
			},
		},
	}

	store.Set("service", core.NewConcreteType("myapp.Service", 0.95))

	code := `service.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	// Check with correct type
	expectedType := core.NewConcreteType("builtins.str", 0.9)
	assert.True(t, s.Check(attrNode, expectedType, ctx))

	// Check with wrong type
	wrongType := core.NewConcreteType("builtins.int", 0.9)
	assert.False(t, s.Check(attrNode, wrongType, ctx))
}

func TestAttributeAccessStrategy_CanHandleNilNode(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	ctx := &InferenceContext{
		SourceCode: []byte(""),
		Store:      store,
	}

	assert.False(t, s.CanHandle(nil, ctx))
}

func TestAttributeAccessStrategy_UnknownAttributeType(t *testing.T) {
	s := NewAttributeAccessStrategy()
	store := newTestTypeStore()

	// Registry exists but attribute not found
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN:   "myapp.Service",
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	store.Set("service", core.NewConcreteType("myapp.Service", 0.95))

	code := `service.unknown`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}
