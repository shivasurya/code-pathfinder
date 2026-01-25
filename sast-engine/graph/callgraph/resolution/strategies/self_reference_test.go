package strategies

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func findAttributeNode(root *sitter.Node) *sitter.Node {
	var find func(*sitter.Node) *sitter.Node
	find = func(n *sitter.Node) *sitter.Node {
		if n.Type() == "attribute" {
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

func TestSelfReferenceStrategy_CanHandle(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

	tests := []struct {
		name     string
		code     string
		selfType core.Type
		expected bool
	}{
		{"self.attr", "self.name", selfType, true},
		{"self.method", "self.process", selfType, true},
		{"other.attr", "other.name", selfType, false},
		{"self without context", "self.name", nil, false},
		{"not attribute", `"hello"`, selfType, false},
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

func TestSelfReferenceStrategy_SynthesizeAttribute(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

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

	code := `self.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		SelfType:     selfType,
		ClassFQN:     "myapp.Service",
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.Equal(t, "builtins.str", typ.FQN())
	assert.InDelta(t, 0.9, conf, 0.01)
}

func TestSelfReferenceStrategy_SynthesizeMethod(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Handler", 0.95)

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Handler": {
				ClassFQN:   "myapp.Handler",
				Methods:    []string{"myapp.Handler.process"},
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	code := `self.process`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		SelfType:     selfType,
		ClassFQN:     "myapp.Handler",
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	// Should return FunctionType for methods
	_, ok := typ.(*core.FunctionType)
	assert.True(t, ok, "Expected FunctionType for method reference")
	assert.Greater(t, conf, 0.0)
}

func TestSelfReferenceStrategy_UnknownAttribute(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN:   "myapp.Service",
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	code := `self.unknown_attr`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		SelfType:     selfType,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestSelfReferenceStrategy_NoSelfType(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()

	code := `self.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode: []byte(code),
		Store:      store,
		SelfType:   nil, // No self type
	}

	typ, conf := s.Synthesize(attrNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestSelfReferenceStrategy_Priority(t *testing.T) {
	s := NewSelfReferenceStrategy()
	// Should be higher than instance_call (80)
	assert.Equal(t, 90, s.Priority())
}

func TestSelfReferenceStrategy_Name(t *testing.T) {
	s := NewSelfReferenceStrategy()
	assert.Equal(t, "self_reference", s.Name())
}

func TestSelfReferenceStrategy_Check(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

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

	code := `self.name`
	root := parseCode(t, code)
	attrNode := findAttributeNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		SelfType:     selfType,
		ClassFQN:     "myapp.Service",
		AttrRegistry: attrReg,
	}

	// Check with correct type
	expectedType := core.NewConcreteType("builtins.str", 0.9)
	assert.True(t, s.Check(attrNode, expectedType, ctx))

	// Check with wrong type
	wrongType := core.NewConcreteType("builtins.int", 0.9)
	assert.False(t, s.Check(attrNode, wrongType, ctx))
}

func TestSelfReferenceStrategy_CanHandleNilNode(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

	ctx := &InferenceContext{
		SourceCode: []byte(""),
		Store:      store,
		SelfType:   selfType,
	}

	assert.False(t, s.CanHandle(nil, ctx))
}

func TestSelfReferenceStrategy_NoAttributeNode(t *testing.T) {
	s := NewSelfReferenceStrategy()
	store := newTestTypeStore()
	selfType := core.NewConcreteType("myapp.Service", 0.95)

	// Create a simple identifier node (not attribute)
	code := `self`
	root := parseCode(t, code)
	identNode := root.Child(0).Child(0) // This is identifier, not attribute

	ctx := &InferenceContext{
		SourceCode: []byte(code),
		Store:      store,
		SelfType:   selfType,
	}

	assert.False(t, s.CanHandle(identNode, ctx))
}
