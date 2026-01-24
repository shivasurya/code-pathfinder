package mcp

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// MOCK REGISTRY (for query_resolver tests)
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

// =============================================================================
// QUERY RESOLVER TESTS
// =============================================================================

func TestQueryResolver_DetectPatterns(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	tests := []struct {
		query   string
		pattern QueryPattern
	}{
		{"self.process()", PatternSelfCall},
		{"UserService().get_user()", PatternInlineInstantiation},
		{"UserService.create()", PatternStaticMethod},
		{"user.get_name()", PatternInstanceCall},
		{"app.service.run()", PatternChainedCall},
		{"myapp.models.User.get_name", PatternDirectFQN},
		{"unknown pattern", PatternUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := resolver.StandardizeQuery(tt.query, nil, "")
			assert.Equal(t, tt.pattern, result.Pattern)
		})
	}
}

func TestQueryResolver_StandardizeInstanceCall(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	knownVars := map[string]string{
		"service": "myapp.UserService",
	}

	result := resolver.StandardizeQuery("service.get_user()", knownVars, "")

	assert.Equal(t, PatternInstanceCall, result.Pattern)
	assert.Equal(t, "myapp.UserService", result.ClassName)
	assert.Equal(t, "get_user", result.MethodName)
	assert.Equal(t, "myapp.UserService.get_user", result.CanonicalFQN)
	assert.Equal(t, 0.85, result.Confidence)
}

func TestQueryResolver_StandardizeSelfCall(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("self.process()", nil, "myapp.Handler")

	assert.Equal(t, PatternSelfCall, result.Pattern)
	assert.Equal(t, "myapp.Handler", result.ClassName)
	assert.Equal(t, "process", result.MethodName)
	assert.Equal(t, "myapp.Handler.process", result.CanonicalFQN)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestQueryResolver_StandardizeInlineInstantiation(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("UserService().get_user()", nil, "")

	assert.Equal(t, PatternInlineInstantiation, result.Pattern)
	assert.Equal(t, "UserService", result.ClassName)
	assert.Equal(t, "get_user", result.MethodName)
	assert.Equal(t, "UserService.get_user", result.CanonicalFQN)
	assert.Equal(t, 0.90, result.Confidence)
}

func TestQueryResolver_UnknownVariable(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("unknown.method()", nil, "")

	assert.Equal(t, PatternInstanceCall, result.Pattern)
	assert.Equal(t, 0.0, result.Confidence)
	assert.False(t, result.RequiresIndex)
}

func TestQueryResolver_StandardizeDirectFQN(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("myapp.models.User.get_name", nil, "")

	assert.Equal(t, PatternDirectFQN, result.Pattern)
	assert.Equal(t, "myapp.models.User.get_name", result.CanonicalFQN)
	assert.Equal(t, "myapp.models.User", result.ClassName)
	assert.Equal(t, "get_name", result.MethodName)
	assert.Equal(t, 1.0, result.Confidence)
	assert.True(t, result.RequiresIndex)
}

func TestQueryResolver_StandardizeStaticMethod(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("ClassName.method()", nil, "")

	assert.Equal(t, PatternStaticMethod, result.Pattern)
	assert.Equal(t, "ClassName", result.ClassName)
	assert.Equal(t, "method", result.MethodName)
	assert.Equal(t, "ClassName.method", result.CanonicalFQN)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestQueryResolver_StandardizeChainedCall(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("app.service.run()", nil, "")

	assert.Equal(t, PatternChainedCall, result.Pattern)
	assert.Equal(t, 0.0, result.Confidence) // Will be filled by chain resolution
	assert.True(t, result.RequiresIndex)
}

func TestQueryResolver_SelfCallWithoutSelfType(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("self.process()", nil, "")

	assert.Equal(t, PatternSelfCall, result.Pattern)
	assert.Equal(t, 0.0, result.Confidence)
	assert.False(t, result.RequiresIndex)
}

func TestQueryResolver_InstanceCallWithoutVariable(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	knownVars := map[string]string{
		"other": "myapp.Other",
	}

	result := resolver.StandardizeQuery("service.method()", knownVars, "")

	assert.Equal(t, PatternInstanceCall, result.Pattern)
	assert.Equal(t, 0.0, result.Confidence)
	assert.False(t, result.RequiresIndex)
}

func TestQueryResolver_UnknownPattern(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("not a valid pattern", nil, "")

	assert.Equal(t, PatternUnknown, result.Pattern)
	assert.Equal(t, 0.0, result.Confidence)
	assert.False(t, result.RequiresIndex)
}

func TestQueryResolver_WithTrailingParens(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	// Test that patterns work with or without trailing ()
	tests := []struct {
		query   string
		pattern QueryPattern
	}{
		{"self.process", PatternSelfCall},
		{"self.process()", PatternSelfCall},
		{"User.create", PatternStaticMethod},
		{"User.create()", PatternStaticMethod},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := resolver.StandardizeQuery(tt.query, nil, "myapp.Handler")
			assert.Equal(t, tt.pattern, result.Pattern)
		})
	}
}

func TestQueryResolver_ExtractClassMethod(t *testing.T) {
	sq := &StandardizedQuery{}

	sq.extractClassMethod("myapp.models.User.get_name")

	assert.Equal(t, "myapp.models.User", sq.ClassName)
	assert.Equal(t, "get_name", sq.MethodName)
}

func TestQueryResolver_ExtractClassMethod_Short(t *testing.T) {
	sq := &StandardizedQuery{}

	sq.extractClassMethod("Class.method")

	assert.Equal(t, "Class", sq.ClassName)
	assert.Equal(t, "method", sq.MethodName)
}

func TestQueryResolver_ExtractClassMethod_Single(t *testing.T) {
	sq := &StandardizedQuery{}

	sq.extractClassMethod("single")

	// Should not extract anything for single segment
	assert.Equal(t, "", sq.ClassName)
	assert.Equal(t, "", sq.MethodName)
}

func TestQueryResolver_ResolveChainedQuery(t *testing.T) {
	attrReg := &mockAttrRegistry{classes: make(map[string]*core.ClassAttributes)}
	resolver := NewQueryResolver(nil, attrReg)

	knownVars := map[string]string{
		"app": "myapp.Application",
	}

	result := resolver.ResolveChainedQuery("app.service.run()", "main.py", knownVars, "")

	assert.Equal(t, PatternChainedCall, result.Pattern)
	assert.Equal(t, 0.7, result.Confidence)
	assert.True(t, result.RequiresIndex)
}

func TestQueryResolver_ResolveChainedQuery_WithSelf(t *testing.T) {
	attrReg := &mockAttrRegistry{classes: make(map[string]*core.ClassAttributes)}
	resolver := NewQueryResolver(nil, attrReg)

	result := resolver.ResolveChainedQuery("self.service.run()", "main.py", nil, "myapp.Manager")

	assert.Equal(t, PatternChainedCall, result.Pattern)
	assert.Equal(t, 0.7, result.Confidence)
}

func TestQueryResolver_DetectPattern_EdgeCases(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	tests := []struct {
		name    string
		query   string
		pattern QueryPattern
	}{
		{"empty string", "", PatternUnknown},
		{"just identifier", "foo", PatternUnknown},
		{"two segments", "foo.bar()", PatternInstanceCall}, // Parens make it clear it's a call
		{"capitalized two segments", "Foo.bar()", PatternStaticMethod}, // Parens make it clear it's a call
		{"many segments lowercase", "a.b.c.d.e.f()", PatternChainedCall},
		{"FQN short", "a.b()", PatternInstanceCall}, // Too short for FQN, but valid instance call
		{"FQN exactly 3", "a.b.c", PatternDirectFQN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.StandardizeQuery(tt.query, nil, "")
			assert.Equal(t, tt.pattern, result.Pattern, "query: %s", tt.query)
		})
	}
}

func TestQueryResolver_TrimWhitespace(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("  self.process()  ", nil, "myapp.Handler")

	assert.Equal(t, PatternSelfCall, result.Pattern)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestNewQueryResolver(t *testing.T) {
	attrReg := &mockAttrRegistry{classes: make(map[string]*core.ClassAttributes)}

	resolver := NewQueryResolver(nil, attrReg)

	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.attrRegistry)
}

func TestQueryResolver_AllPatternConstants(t *testing.T) {
	// Verify all pattern constants are distinct
	patterns := []QueryPattern{
		PatternUnknown,
		PatternDirectFQN,
		PatternInstanceCall,
		PatternSelfCall,
		PatternChainedCall,
		PatternInlineInstantiation,
		PatternStaticMethod,
		PatternClassMethod,
	}

	seen := make(map[QueryPattern]bool)
	for _, p := range patterns {
		assert.False(t, seen[p], "Duplicate pattern value: %d", p)
		seen[p] = true
	}
}

func TestStandardizedQuery_IndexQuery(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	knownVars := map[string]string{
		"service": "myapp.UserService",
	}

	result := resolver.StandardizeQuery("service.get_user()", knownVars, "")

	assert.True(t, result.RequiresIndex)
	assert.Equal(t, "myapp.UserService.get_user", result.IndexQuery)
}

func TestStandardizedQuery_NoIndexQueryWhenUnknown(t *testing.T) {
	resolver := NewQueryResolver(nil, nil)

	result := resolver.StandardizeQuery("unknown.method()", nil, "")

	assert.False(t, result.RequiresIndex)
	assert.Equal(t, "", result.IndexQuery)
}
