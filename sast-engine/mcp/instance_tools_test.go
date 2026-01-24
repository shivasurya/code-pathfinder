package mcp

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MOCK REGISTRIES
// =============================================================================

type mockAttrReg struct {
	classes map[string]*core.ClassAttributes
}

func (m *mockAttrReg) GetClassAttributes(fqn string) *core.ClassAttributes {
	return m.classes[fqn]
}

func (m *mockAttrReg) GetAttribute(fqn, attr string) *core.ClassAttribute {
	if ca := m.classes[fqn]; ca != nil {
		return ca.Attributes[attr]
	}
	return nil
}

func (m *mockAttrReg) HasClass(fqn string) bool {
	_, found := m.classes[fqn]
	return found
}

// =============================================================================
// RESOLVE INSTANCE CALL TESTS
// =============================================================================

func TestHandleResolveInstanceCall_SimpleCase(t *testing.T) {
	attrReg := &mockAttrReg{
		classes: map[string]*core.ClassAttributes{
			"myapp.UserService": {
				ClassFQN: "myapp.UserService",
				Methods:  []string{"myapp.UserService.get_user"},
				FilePath: "myapp/services.py",
			},
		},
	}

	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "service.get_user()",
		FilePath:   "main.py",
		Line:       10,
		Column:     0,
		Context: &InstanceCallContext{
			Variables: map[string]string{
				"service": "myapp.UserService",
			},
		},
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "myapp.UserService", resp.ResolvedType)
	assert.Equal(t, "get_user", resp.Method)
	assert.Equal(t, "myapp.UserService.get_user", resp.CanonicalFQN)
	assert.Greater(t, resp.Confidence, 0.5)
}

func TestHandleResolveInstanceCall_SelfReference(t *testing.T) {
	attrReg := &mockAttrReg{
		classes: map[string]*core.ClassAttributes{
			"myapp.Handler": {
				ClassFQN: "myapp.Handler",
				Methods:  []string{"myapp.Handler.process"},
				FilePath: "myapp/handlers.py",
			},
		},
	}

	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "self.process()",
		FilePath:   "myapp/handlers.py",
		Line:       25,
		Context: &InstanceCallContext{
			SelfType: "myapp.Handler",
		},
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "myapp.Handler", resp.ResolvedType)
	assert.Equal(t, "process", resp.Method)
}

func TestHandleResolveInstanceCall_UnknownVariable(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "unknown.method()",
		FilePath:   "main.py",
		Context:    &InstanceCallContext{},
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "could not resolve")
}

func TestHandleResolveInstanceCall_InvalidJSON(t *testing.T) {
	handler := NewInstanceToolHandler(nil, nil, nil)

	resp, err := handler.HandleResolveInstanceCall([]byte("invalid json"))

	require.NoError(t, err) // Handler returns error in response, not as error
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "invalid request")
}

func TestHandleResolveInstanceCall_EmptyExpression(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "",
		FilePath:   "main.py",
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "empty expression")
}

func TestHandleResolveInstanceCall_ParseError(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "if for while", // Invalid Python
		FilePath:   "main.py",
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	// Parser might succeed but result won't resolve
	assert.False(t, resp.Success)
}

func TestHandleResolveInstanceCall_WithDefinitionLocation(t *testing.T) {
	attrReg := &mockAttrReg{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Methods:  []string{"myapp.Service.process"},
				FilePath: "myapp/service.py",
			},
		},
	}

	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "svc.process()",
		FilePath:   "main.py",
		Context: &InstanceCallContext{
			Variables: map[string]string{
				"svc": "myapp.Service",
			},
		},
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Definition)
	assert.Equal(t, "myapp/service.py", resp.Definition.FilePath)
}

func TestHandleResolveInstanceCall_NoDefinition(t *testing.T) {
	attrReg := &mockAttrReg{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Methods:  []string{},  // No methods
				FilePath: "myapp/service.py",
			},
		},
	}

	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := ResolveInstanceCallRequest{
		Expression: "svc.process()",
		FilePath:   "main.py",
		Context: &InstanceCallContext{
			Variables: map[string]string{
				"svc": "myapp.Service",
			},
		},
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleResolveInstanceCall(args)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Nil(t, resp.Definition) // No definition found
}

// =============================================================================
// GET INSTANCE TYPE TESTS
// =============================================================================

func TestHandleGetInstanceType_SimpleVariable(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := GetInstanceTypeRequest{
		Variable: "\"hello\"",
		FilePath: "main.py",
		Line:     10,
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleGetInstanceType(args)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "builtins.str", resp.TypeFQN)
	assert.Greater(t, resp.Confidence, 0.9)
}

func TestHandleGetInstanceType_InvalidJSON(t *testing.T) {
	handler := NewInstanceToolHandler(nil, nil, nil)

	resp, err := handler.HandleGetInstanceType([]byte("invalid"))

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "invalid request")
}

func TestHandleGetInstanceType_ParseError(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := GetInstanceTypeRequest{
		Variable: "if for", // Invalid
		FilePath: "main.py",
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleGetInstanceType(args)

	require.NoError(t, err)
	// Will parse but won't resolve to a concrete type
	assert.False(t, resp.Success)
}

func TestHandleGetInstanceType_UnknownVariable(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	handler := NewInstanceToolHandler(inferencer, attrReg, nil)

	req := GetInstanceTypeRequest{
		Variable: "unknown_var",
		FilePath: "main.py",
	}

	args, _ := json.Marshal(req)
	resp, err := handler.HandleGetInstanceType(args)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "unknown")
}

// =============================================================================
// HELPER METHOD TESTS
// =============================================================================

func TestExtractMethodName_CallNode(t *testing.T) {
	handler := NewInstanceToolHandler(nil, nil, nil)

	// This would need a real tree-sitter node, so we test the nil path
	name := handler.extractMethodName(nil, []byte("test"))
	assert.Equal(t, "", name)
}

func TestLookupDefinition_NilRegistry(t *testing.T) {
	handler := NewInstanceToolHandler(nil, nil, nil)

	def := handler.lookupDefinition("myapp.Class", "method")
	assert.Nil(t, def)
}

func TestLookupDefinition_ClassNotFound(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	handler := NewInstanceToolHandler(nil, attrReg, nil)

	def := handler.lookupDefinition("unknown.Class", "method")
	assert.Nil(t, def)
}

func TestLookupDefinition_MethodNotFound(t *testing.T) {
	attrReg := &mockAttrReg{
		classes: map[string]*core.ClassAttributes{
			"myapp.Service": {
				ClassFQN: "myapp.Service",
				Methods:  []string{"myapp.Service.other"},
				FilePath: "myapp/service.py",
			},
		},
	}
	handler := NewInstanceToolHandler(nil, attrReg, nil)

	def := handler.lookupDefinition("myapp.Service", "notfound")
	assert.Nil(t, def)
}

func TestNewInstanceToolHandler(t *testing.T) {
	attrReg := &mockAttrReg{classes: make(map[string]*core.ClassAttributes)}
	inferencer := resolution.NewBidirectionalInferencer(attrReg, nil, nil, 1000)
	cg := &core.CallGraph{}

	handler := NewInstanceToolHandler(inferencer, attrReg, cg)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.inferencer)
	assert.NotNil(t, handler.attrRegistry)
	assert.NotNil(t, handler.callGraph)
}
