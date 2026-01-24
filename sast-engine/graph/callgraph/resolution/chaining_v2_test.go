package resolution

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures for chaining_v2.

type testAttrRegistry struct {
	classes map[string]*core.ClassAttributes
}

func (r *testAttrRegistry) GetClassAttributes(fqn string) *core.ClassAttributes {
	return r.classes[fqn]
}

func (r *testAttrRegistry) GetAttribute(fqn, attr string) *core.ClassAttribute {
	if ca := r.classes[fqn]; ca != nil {
		return ca.Attributes[attr]
	}
	return nil
}

func (r *testAttrRegistry) HasClass(fqn string) bool {
	_, found := r.classes[fqn]
	return found
}

func parseTestCode(t *testing.T, code string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	require.NoError(t, err)
	return tree.RootNode()
}

func findTestCallNode(t *testing.T, root *sitter.Node) *sitter.Node {
	t.Helper()
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

func TestResolveDeepAttributeChain(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.App": {
				Attributes: map[string]*core.ClassAttribute{
					"service": {Type: &core.TypeInfo{TypeFQN: "myapp.Service", Confidence: 0.9}},
				},
			},
			"myapp.Service": {
				Attributes: map[string]*core.ClassAttribute{
					"controller": {Type: &core.TypeInfo{TypeFQN: "myapp.Controller", Confidence: 0.9}},
				},
			},
		},
	}

	startType := core.NewConcreteType("myapp.App", 0.95)
	chain := []string{"service", "controller"}

	resultType, conf := ResolveDeepAttributeChain(chain, startType, attrReg)

	assert.Equal(t, "myapp.Controller", resultType.FQN())
	assert.Greater(t, conf, 0.7)
}

func TestResolveDeepAttributeChain_UnknownAttribute(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.App": {
				Attributes: make(map[string]*core.ClassAttribute),
			},
		},
	}

	startType := core.NewConcreteType("myapp.App", 0.95)
	chain := []string{"unknown"}

	resultType, conf := ResolveDeepAttributeChain(chain, startType, attrReg)

	assert.True(t, core.IsAnyType(resultType))
	assert.Equal(t, 0.0, conf)
}

func TestResolveDeepAttributeChain_EmptyChain(t *testing.T) {
	startType := core.NewConcreteType("myapp.App", 0.95)

	resultType, conf := ResolveDeepAttributeChain([]string{}, startType, nil)

	assert.Equal(t, "myapp.App", resultType.FQN())
	assert.Equal(t, 1.0, conf)
}

func TestResolveDeepAttributeChain_NonConcreteType(t *testing.T) {
	startType := &core.AnyType{Reason: "test"}
	chain := []string{"attr"}

	resultType, conf := ResolveDeepAttributeChain(chain, startType, nil)

	assert.True(t, core.IsAnyType(resultType))
	assert.Equal(t, 0.0, conf)
}

func TestResolveInlineInstantiation(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"UserService": {
				ClassFQN: "UserService",
				Methods:  []string{"UserService.get_user"},
			},
		},
	}

	code := `UserService().get_user()`
	root := parseTestCode(t, code)
	callNode := findTestCallNode(t, root)
	require.NotNil(t, callNode)

	typ, conf := ResolveInlineInstantiation(callNode, []byte(code), attrReg, nil, "test.py")

	assert.Equal(t, "UserService", typ.FQN())
	assert.Greater(t, conf, 0.5)
}

func TestResolveInlineInstantiation_NotAChain(t *testing.T) {
	code := `func()`
	root := parseTestCode(t, code)
	callNode := findTestCallNode(t, root)
	require.NotNil(t, callNode)

	typ, conf := ResolveInlineInstantiation(callNode, []byte(code), nil, nil, "test.py")

	assert.True(t, core.IsAnyType(typ))
	assert.Equal(t, 0.0, conf)
}

func TestChainResolver_FluentAPI(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Methods:  []string{"myapp.Service.process"},
			},
		},
	}

	code := `service.process()`
	resolver := NewChainResolver(attrReg, nil, nil).
		WithContext("test.py", []byte(code)).
		WithVariable("service", core.NewConcreteType("myapp.Service", 0.95))

	root := parseTestCode(t, code)
	callNode := findTestCallNode(t, root)
	require.NotNil(t, callNode)

	typ, conf := resolver.Resolve(callNode)

	assert.Equal(t, "myapp.Service", typ.FQN())
	assert.Greater(t, conf, 0.5)
}

func TestChainResolver_WithSelf(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"myapp.Handler": {
				ClassFQN: "myapp.Handler",
				Attributes: map[string]*core.ClassAttribute{
					"service": {Type: &core.TypeInfo{TypeFQN: "myapp.Service", Confidence: 0.9}},
				},
			},
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Methods:  []string{"myapp.Service.run"},
			},
		},
	}

	code := `self.service.run()`
	resolver := NewChainResolver(attrReg, nil, nil).
		WithSelf(core.NewConcreteType("myapp.Handler", 0.95), "myapp.Handler").
		WithContext("test.py", []byte(code))

	root := parseTestCode(t, code)
	callNode := findTestCallNode(t, root)
	require.NotNil(t, callNode)

	typ, conf := resolver.Resolve(callNode)

	assert.Equal(t, "myapp.Service", typ.FQN())
	assert.Greater(t, conf, 0.3)
}

func TestChainResolver_MultipleVariables(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"ServiceA": {
				ClassFQN: "ServiceA",
				Methods:  []string{"ServiceA.doA"},
			},
			"ServiceB": {
				ClassFQN: "ServiceB",
				Methods:  []string{"ServiceB.doB"},
			},
		},
	}

	code := `svc_a.doA()`
	resolver := NewChainResolver(attrReg, nil, nil).
		WithContext("test.py", []byte(code)).
		WithVariable("svc_a", core.NewConcreteType("ServiceA", 0.9)).
		WithVariable("svc_b", core.NewConcreteType("ServiceB", 0.9))

	root := parseTestCode(t, code)
	callNode := findTestCallNode(t, root)
	require.NotNil(t, callNode)

	typ, conf := resolver.Resolve(callNode)

	assert.Equal(t, "ServiceA", typ.FQN())
	assert.Greater(t, conf, 0.0)
}

func TestChainResolver_ChainMethodCalls(t *testing.T) {
	attrReg := &testAttrRegistry{
		classes: map[string]*core.ClassAttributes{
			"Builder": {
				ClassFQN: "Builder",
				Methods:  []string{"Builder.set_a", "Builder.set_b", "Builder.build"},
			},
		},
	}

	code := `Builder().set_a().set_b().build()`
	resolver := NewChainResolver(attrReg, nil, nil).
		WithContext("test.py", []byte(code))

	root := parseTestCode(t, code)
	callNode := findTestCallNode(t, root)
	require.NotNil(t, callNode)

	typ, conf := resolver.Resolve(callNode)

	// Fluent heuristic - each method returns Builder
	assert.Equal(t, "Builder", typ.FQN())
	assert.Greater(t, conf, 0.0)
}
