package strategies

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CHAIN PARSING TESTS
// =============================================================================

func TestChainStrategy_ParseSimpleChain(t *testing.T) {
	s := NewChainStrategy()

	code := `obj.attr`
	root := parseCode(t, code)
	exprNode := root.Child(0).Child(0)

	steps := s.parseChain(exprNode, []byte(code))

	require.Len(t, steps, 2)
	assert.Equal(t, "obj", steps[0].Name)
	assert.Equal(t, StepIdentifier, steps[0].Kind)
	assert.Equal(t, "attr", steps[1].Name)
	assert.Equal(t, StepAttribute, steps[1].Kind)
}

func TestChainStrategy_ParseMethodChain(t *testing.T) {
	s := NewChainStrategy()

	code := `obj.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	steps := s.parseChain(callNode, []byte(code))

	require.Len(t, steps, 2)
	assert.Equal(t, "obj", steps[0].Name)
	assert.Equal(t, "method", steps[1].Name)
	assert.Equal(t, StepMethodCall, steps[1].Kind)
}

func TestChainStrategy_ParseDeepChain(t *testing.T) {
	s := NewChainStrategy()

	code := `a.b.c.d.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	steps := s.parseChain(callNode, []byte(code))

	require.Len(t, steps, 5)
	assert.Equal(t, "a", steps[0].Name)
	assert.Equal(t, "b", steps[1].Name)
	assert.Equal(t, "c", steps[2].Name)
	assert.Equal(t, "d", steps[3].Name)
	assert.Equal(t, "method", steps[4].Name)
}

func TestChainStrategy_ParseInstantiation(t *testing.T) {
	s := NewChainStrategy()

	code := `UserService().get_user()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	steps := s.parseChain(callNode, []byte(code))

	require.Len(t, steps, 2)
	assert.Equal(t, "UserService", steps[0].Name)
	assert.Equal(t, StepInstantiation, steps[0].Kind)
	assert.Equal(t, "get_user", steps[1].Name)
	assert.Equal(t, StepMethodCall, steps[1].Kind)
}

func TestChainStrategy_ParseConstructorChaining(t *testing.T) {
	s := NewChainStrategy()

	code := `Builder().set_name("x").set_value(42).build()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	steps := s.parseChain(callNode, []byte(code))

	require.Len(t, steps, 4)
	assert.Equal(t, "Builder", steps[0].Name)
	assert.Equal(t, StepInstantiation, steps[0].Kind)
	assert.Equal(t, "set_name", steps[1].Name)
	assert.Equal(t, "set_value", steps[2].Name)
	assert.Equal(t, "build", steps[3].Name)
}

// =============================================================================
// CHAIN RESOLUTION TESTS
// =============================================================================

func TestChainStrategy_CanHandle(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"two levels", "obj.attr", true},
		{"method call", "obj.method()", true},
		{"deep chain", "a.b.c.d()", true},
		{"instantiation chain", "Builder().build()", true},
		{"single identifier", "obj", false},
		{"simple call", "func()", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseCode(t, tt.code)
			node := root.Child(0).Child(0)

			ctx := createContext(store, []byte(tt.code))
			assert.Equal(t, tt.expected, s.CanHandle(node, ctx))
		})
	}
}

func TestChainStrategy_SynthesizeDeepAttributeChain(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	// Setup: app.service.controller where:
	// - app is App
	// - App.service is Service
	// - Service.controller is Controller
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.App": {
				ClassFQN: "myapp.App",
				Attributes: map[string]*core.ClassAttribute{
					"service": {Name: "service", Type: &core.TypeInfo{TypeFQN: "myapp.Service", Confidence: 0.9}},
				},
			},
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Attributes: map[string]*core.ClassAttribute{
					"controller": {Name: "controller", Type: &core.TypeInfo{TypeFQN: "myapp.Controller", Confidence: 0.9}},
				},
			},
			"myapp.Controller": {
				ClassFQN: "myapp.Controller",
				Methods:  []string{"myapp.Controller.run"},
			},
		},
	}

	store.Set("app", core.NewConcreteType("myapp.App", 0.95))

	code := `app.service.controller.run()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	assert.Equal(t, "myapp.Controller", typ.FQN())
	assert.Greater(t, conf, 0.3)
}

func TestChainStrategy_SynthesizeInlineInstantiation(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"UserService": {
				ClassFQN: "UserService",
				Methods:  []string{"UserService.get_user"},
			},
		},
	}

	code := `UserService().get_user()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	assert.Equal(t, "UserService", typ.FQN())
	assert.Greater(t, conf, 0.5)
}

func TestChainStrategy_SynthesizeConstructorChaining(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"Builder": {
				ClassFQN: "Builder",
				Methods:  []string{"Builder.set_name", "Builder.set_value", "Builder.build"},
			},
		},
	}

	code := `Builder().set_name("x").set_value(42).build()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	// Each fluent method returns Builder (heuristic)
	assert.Equal(t, "Builder", typ.FQN())
	assert.Greater(t, conf, 0.2) // Multiple fluent heuristics compound
}

func TestChainStrategy_SynthesizeSelfDeepChain(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	selfType := core.NewConcreteType("myapp.Manager", 0.95)

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Manager": {
				ClassFQN: "myapp.Manager",
				Attributes: map[string]*core.ClassAttribute{
					"service": {Name: "service", Type: &core.TypeInfo{TypeFQN: "myapp.Service", Confidence: 0.9}},
				},
			},
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Attributes: map[string]*core.ClassAttribute{
					"controller": {Name: "controller", Type: &core.TypeInfo{TypeFQN: "myapp.Controller", Confidence: 0.9}},
				},
			},
			"myapp.Controller": {
				ClassFQN: "myapp.Controller",
				Methods:  []string{"myapp.Controller.run"},
			},
		},
	}

	code := `self.service.controller.run()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		SelfType:     selfType,
		ClassFQN:     "myapp.Manager",
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	assert.Equal(t, "myapp.Controller", typ.FQN())
	assert.Greater(t, conf, 0.3)
}

func TestChainStrategy_UnknownFirstStep(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	code := `unknown.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(callNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestChainStrategy_UnknownAttributeInChain(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN:   "myapp.Service",
				Attributes: make(map[string]*core.ClassAttribute), // Empty - no attributes
			},
		},
	}

	store.Set("service", core.NewConcreteType("myapp.Service", 0.95))

	code := `service.unknown_attr.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestChainStrategy_BuiltinMethodInChain(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	store.Set("name", core.NewConcreteType("builtins.str", 0.95))

	builtinReg := &mockBuiltinReg{}

	code := `name.upper().split()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:      []byte(code),
		Store:           store,
		BuiltinRegistry: builtinReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	// upper() -> str, split() -> list
	assert.Equal(t, "builtins.list", typ.FQN())
	assert.Greater(t, conf, 0.5)
}

func TestChainStrategy_Priority(t *testing.T) {
	s := NewChainStrategy()
	assert.Equal(t, 85, s.Priority())
}

func TestChainStrategy_Name(t *testing.T) {
	s := NewChainStrategy()
	assert.Equal(t, "chain", s.Name())
}

func TestChainStrategy_Check(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"Builder": {
				ClassFQN: "Builder",
				Methods:  []string{"Builder.build"},
			},
		},
	}

	code := `Builder().build()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	// Check with correct type
	expectedType := core.NewConcreteType("Builder", 0.8)
	assert.True(t, s.Check(callNode, expectedType, ctx))

	// Check with wrong type
	wrongType := core.NewConcreteType("OtherClass", 0.8)
	assert.False(t, s.Check(callNode, wrongType, ctx))
}

func TestChainStrategy_EmptyChain(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	code := `"string"`
	root := parseCode(t, code)
	node := root.Child(0).Child(0)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(node, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestChainStrategy_TooDeepChain(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	// Create a chain deeper than MaxChainDepth
	code := `a.b.c.d.e.f.g.h.i.j.k.l.m()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(callNode, ctx)

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestChainStrategy_FunctionCallFirstStep(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	code := `get_service().process()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := createContext(store, []byte(code))

	typ, conf := s.Synthesize(callNode, ctx)

	// Function return type unknown - should return Any
	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestChainStrategy_ParseWithNilNode(t *testing.T) {
	s := NewChainStrategy()

	steps := s.parseChain(nil, []byte(""))

	assert.Empty(t, steps)
}

func TestChainStrategy_GetCallArgs(t *testing.T) {
	s := NewChainStrategy()

	code := `func(1, "hello", 3.14)`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	args := s.getCallArgs(callNode)

	assert.Len(t, args, 3)
}

func TestChainStrategy_GetCallArgsNoArgumentList(t *testing.T) {
	s := NewChainStrategy()

	code := `obj.attr`
	root := parseCode(t, code)
	attrNode := root.Child(0).Child(0)

	args := s.getCallArgs(attrNode)

	assert.Nil(t, args)
}

func TestChainStrategy_InstantiationWithModuleRegistry(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"UserService": {
				ClassFQN: "UserService",
				Methods:  []string{"UserService.get_user"},
			},
		},
	}

	code := `UserService().get_user()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		FilePath:     "test.py",
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	// Should find UserService
	assert.Equal(t, "UserService", typ.FQN())
	assert.Greater(t, conf, 0.0)
}

func TestChainStrategy_InstantiationNotFound(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	attrReg := &mockAttrRegistry{
		classes: make(map[string]*core.ClassAttributes),
	}

	code := `UnknownClass().method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	// Should still create type with fluent heuristic
	assert.Equal(t, "UnknownClass", typ.FQN())
	assert.Greater(t, conf, 0.0)
}

func TestChainStrategy_LowConfidenceEarlyExit(t *testing.T) {
	s := NewChainStrategy()
	store := newTestTypeStore()

	// Set up chain where confidence drops below threshold
	attrReg := &mockAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"LowConf": {
				ClassFQN: "LowConf",
				Attributes: map[string]*core.ClassAttribute{
					"next": {Name: "next", Type: &core.TypeInfo{TypeFQN: "VeryLowConf", Confidence: 0.1}},
				},
			},
		},
	}

	store.Set("obj", core.NewConcreteType("LowConf", 0.2))

	code := `obj.next.method()`
	root := parseCode(t, code)
	callNode := findCallNode(root)

	ctx := &InferenceContext{
		SourceCode:   []byte(code),
		Store:        store,
		AttrRegistry: attrReg,
	}

	typ, conf := s.Synthesize(callNode, ctx)

	// Should stop due to low confidence
	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}
