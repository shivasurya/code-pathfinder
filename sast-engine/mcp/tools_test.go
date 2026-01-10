package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestToolGetIndexInfo(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetIndexInfo()

	assert.False(t, isError)
	assert.Contains(t, result, "project_path")
	assert.Contains(t, result, "/test/project")
	assert.Contains(t, result, "python_version")
	assert.Contains(t, result, "3.11")
	assert.Contains(t, result, "stats")

	// Verify JSON is valid.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	// Verify stats structure.
	stats, ok := parsed["stats"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, stats, "functions")
	assert.Contains(t, stats, "call_edges")
	assert.Contains(t, stats, "modules")
}

func TestToolFindSymbol_Found(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate_user"})

	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
	assert.Contains(t, result, "matches")
	assert.Contains(t, result, "myapp.auth.validate_user")
}

func TestToolFindSymbol_PartialMatch(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate"})

	// Should find validate_user via partial match.
	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
}

func TestToolFindSymbol_MultipleMatches(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "log"})

	// Should find both login and logout.
	assert.False(t, isError)
	assert.Contains(t, result, "login")
	assert.Contains(t, result, "logout")
}

func TestToolFindSymbol_NotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "nonexistent_function_xyz"})

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
	assert.Contains(t, result, "suggestion")
}

func TestToolFindSymbol_EmptyName(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": ""})

	assert.True(t, isError)
	assert.Contains(t, result, "required")
}

func TestToolGetCallers_Found(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallers(map[string]interface{}{"function": "validate_user"})

	assert.False(t, isError)
	assert.Contains(t, result, "callers")
	assert.Contains(t, result, "login")
	assert.Contains(t, result, "target")
	assert.Contains(t, result, "pagination")
}

func TestToolGetCallers_NoCallers(t *testing.T) {
	server := createTestServer()

	// login has no callers in our test data.
	result, isError := server.toolGetCallers(map[string]interface{}{"function": "login"})

	assert.False(t, isError)
	assert.Contains(t, result, "pagination")
	assert.Contains(t, result, `"total": 0`)
}

func TestToolGetCallers_NotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallers(map[string]interface{}{"function": "nonexistent_function"})

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
}

func TestToolGetCallers_EmptyName(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallers(map[string]interface{}{"function": ""})

	assert.True(t, isError)
	assert.Contains(t, result, "required")
}

func TestToolGetCallees_Found(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallees(map[string]interface{}{"function": "login"})

	assert.False(t, isError)
	assert.Contains(t, result, "callees")
	assert.Contains(t, result, "validate_user")
	assert.Contains(t, result, "source")
	assert.Contains(t, result, "resolved_count")
}

func TestToolGetCallees_NoCallees(t *testing.T) {
	server := createTestServer()

	// validate_user has no callees in our test data.
	result, isError := server.toolGetCallees(map[string]interface{}{"function": "validate_user"})

	assert.False(t, isError)
	assert.Contains(t, result, "pagination")
}

func TestToolGetCallees_NotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallees(map[string]interface{}{"function": "nonexistent_function"})

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
}

func TestToolGetCallees_EmptyName(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallees(map[string]interface{}{"function": ""})

	assert.True(t, isError)
	assert.Contains(t, result, "required")
}

func TestToolGetCallDetails_Found(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallDetails("login", "validate_user")

	assert.False(t, isError)
	assert.Contains(t, result, "call_site")
	assert.Contains(t, result, "location")
	assert.Contains(t, result, "resolution")
}

func TestToolGetCallDetails_NotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallDetails("login", "nonexistent")

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
}

func TestToolGetCallDetails_CallerNotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallDetails("nonexistent", "validate_user")

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
}

func TestToolGetCallDetails_EmptyParams(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolGetCallDetails("", "validate_user")

	assert.True(t, isError)
	assert.Contains(t, result, "required")
}

func TestToolResolveImport_ExactMatch(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolResolveImport("myapp.auth")

	assert.False(t, isError)
	assert.Contains(t, result, "resolved")
	assert.Contains(t, result, "auth.py")
	assert.Contains(t, result, "exact")
}

func TestToolResolveImport_ShortName(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolResolveImport("auth")

	assert.False(t, isError)
	assert.Contains(t, result, "auth.py")
}

func TestToolResolveImport_NotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolResolveImport("nonexistent.module")

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
}

func TestToolResolveImport_EmptyPath(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolResolveImport("")

	assert.True(t, isError)
	assert.Contains(t, result, "required")
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	server := createTestServer()

	result, isError := server.executeTool("unknown_tool", nil)

	assert.True(t, isError)
	assert.Contains(t, result, "Unknown tool")
}

func TestExecuteTool_AllToolsDispatch(t *testing.T) {
	server := createTestServer()

	tests := []struct {
		name      string
		toolName  string
		args      map[string]interface{}
		wantError bool
	}{
		{"get_index_info", "get_index_info", nil, false},
		{"find_symbol", "find_symbol", map[string]interface{}{"name": "login"}, false},
		{"get_callers", "get_callers", map[string]interface{}{"function": "validate_user"}, false},
		{"get_callees", "get_callees", map[string]interface{}{"function": "login"}, false},
		{"get_call_details", "get_call_details", map[string]interface{}{"caller": "login", "callee": "validate_user"}, false},
		{"resolve_import", "resolve_import", map[string]interface{}{"import": "myapp.auth"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isError := server.executeTool(tt.toolName, tt.args)
			assert.Equal(t, tt.wantError, isError)
			assert.NotEmpty(t, result)
		})
	}
}

func TestFindMatchingFQNs(t *testing.T) {
	server := createTestServer()

	// Note: findMatchingFQNs does exact short name, suffix, or FQN match.
	// It does NOT do substring matching like toolFindSymbol does.
	tests := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{"exact short name", "validate_user", 1},
		{"exact short name login", "login", 1},
		{"exact short name logout", "logout", 1},
		{"no partial match", "validate", 0}, // findMatchingFQNs doesn't do substring matching
		{"no match", "xyz123", 0},
		{"fqn match", "myapp.auth.validate_user", 1},
		{"no substring match", "log", 0}, // doesn't match login/logout without Contains
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fqns := server.findMatchingFQNs(tt.input)
			assert.Equal(t, tt.expectedCount, len(fqns))
		})
	}
}

func TestGetShortName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"myapp.auth.validate_user", "validate_user"},
		{"simple", "simple"},
		{"a.b.c.d.e", "e"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getShortName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToolOutputFormat_ValidJSON(t *testing.T) {
	server := createTestServer()

	// All tools should return valid JSON.
	tools := []struct {
		name string
		args map[string]interface{}
	}{
		{"get_index_info", nil},
		{"find_symbol", map[string]interface{}{"name": "validate_user"}},
		{"get_callers", map[string]interface{}{"function": "validate_user"}},
		{"get_callees", map[string]interface{}{"function": "login"}},
		{"get_call_details", map[string]interface{}{"caller": "login", "callee": "validate_user"}},
		{"resolve_import", map[string]interface{}{"import": "myapp.auth"}},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := server.executeTool(tt.name, tt.args)

			var parsed interface{}
			err := json.Unmarshal([]byte(result), &parsed)
			assert.NoError(t, err, "Tool %s should return valid JSON", tt.name)
		})
	}
}

func TestGetToolDefinitions(t *testing.T) {
	server := createTestServer()

	tools := server.getToolDefinitions()

	assert.Len(t, tools, 6)

	// Verify each tool has required fields.
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.Equal(t, "object", tool.InputSchema.Type)
	}

	// Verify specific tools exist.
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["get_index_info"])
	assert.True(t, toolNames["find_symbol"])
	assert.True(t, toolNames["get_callers"])
	assert.True(t, toolNames["get_callees"])
	assert.True(t, toolNames["get_call_details"])
	assert.True(t, toolNames["resolve_import"])
}

// ============================================================================
// Extended Coverage Tests
// ============================================================================

func TestToolFindSymbol_WithAllFields(t *testing.T) {
	server := createExtendedTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate_user"})

	assert.False(t, isError)

	// Verify all optional fields are included.
	assert.Contains(t, result, "validate_user")
	assert.Contains(t, result, "return_type")
	assert.Contains(t, result, "User")
	assert.Contains(t, result, "parameters")
	assert.Contains(t, result, "username")
	assert.Contains(t, result, "modifier")
	assert.Contains(t, result, "public")
	assert.Contains(t, result, "decorators")
	assert.Contains(t, result, "@login_required")
	assert.Contains(t, result, "superclass")
	assert.Contains(t, result, "BaseValidator")
}

func TestToolGetCallDetails_WithArguments(t *testing.T) {
	server := createExtendedTestServer()

	result, isError := server.toolGetCallDetails("login", "validate_user")

	assert.False(t, isError)
	assert.Contains(t, result, "arguments")
	assert.Contains(t, result, "request.username")
	assert.Contains(t, result, "request.password")
	assert.Contains(t, result, "position")
}

func TestToolGetCallDetails_UnresolvedCall(t *testing.T) {
	server := createExtendedTestServer()

	result, isError := server.toolGetCallDetails("login", "external_call")

	assert.False(t, isError)
	assert.Contains(t, result, "external_call")
	assert.Contains(t, result, "resolved")
	assert.Contains(t, result, "failure_reason")
	assert.Contains(t, result, "Module 'external' not found")
}

func TestToolGetCallDetails_TypeInference(t *testing.T) {
	server := createExtendedTestServer()

	result, isError := server.toolGetCallDetails("login", "inferred_method")

	assert.False(t, isError)
	assert.Contains(t, result, "inferred_method")
	assert.Contains(t, result, "via_type_inference")
	assert.Contains(t, result, "inferred_type")
	assert.Contains(t, result, "MyClass")
	assert.Contains(t, result, "type_confidence")
	assert.Contains(t, result, "type_source")
	assert.Contains(t, result, "assignment")
}

func TestToolGetCallees_WithUnresolvedCalls(t *testing.T) {
	server := createExtendedTestServer()

	result, isError := server.toolGetCallees(map[string]interface{}{"function": "login"})

	assert.False(t, isError)
	assert.Contains(t, result, "callees")
	assert.Contains(t, result, "resolved_count")
	assert.Contains(t, result, "unresolved_count")
}

func TestToolResolveImport_AmbiguousShortName(t *testing.T) {
	server := createExtendedTestServer()

	// "utils" maps to both myapp.utils and other.utils.
	result, isError := server.toolResolveImport("utils")

	// Ambiguous matches return false for isError but with alternatives.
	assert.False(t, isError)
	assert.Contains(t, result, "ambiguous")
	assert.Contains(t, result, "alternatives")
	assert.Contains(t, result, "myapp.utils")
	assert.Contains(t, result, "other.utils")
}

func TestToolResolveImport_PartialMatch(t *testing.T) {
	server := createExtendedTestServer()

	// "models" is a partial match.
	result, isError := server.toolResolveImport("models")

	assert.False(t, isError)
	assert.Contains(t, result, "models")
}

func TestToolResolveImport_PartialFQNMatch(t *testing.T) {
	server := createExtendedTestServer()

	// Try partial match that's not exact and not short name.
	// "app.auth" matches short name "auth" so it uses short_name match.
	result, isError := server.toolResolveImport("app.auth")

	assert.False(t, isError)
	assert.Contains(t, result, "myapp.auth")
	assert.Contains(t, result, "resolved")
}

func TestToolResolveImport_NoMatch(t *testing.T) {
	server := createExtendedTestServer()

	// Try something that doesn't exist.
	result, isError := server.toolResolveImport("completely.nonexistent.module")

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
}

func TestToolResolveImport_PartialContainsMatch(t *testing.T) {
	server := createExtendedTestServer()

	// "myapp" is not exact match, not a short name, but IS contained in FQNs.
	result, isError := server.toolResolveImport("myapp")

	assert.False(t, isError)
	assert.Contains(t, result, "partial")
	assert.Contains(t, result, "alternatives")
	assert.Contains(t, result, "myapp.auth")
}

func TestToolGetCallers_WithMultipleCallSites(t *testing.T) {
	server := createExtendedTestServer()

	result, isError := server.toolGetCallers(map[string]interface{}{"function": "validate_user"})

	assert.False(t, isError)
	assert.Contains(t, result, "callers")
	assert.Contains(t, result, "login")
	assert.Contains(t, result, "pagination")
}

func TestToolGetCallers_MultipleMatches(t *testing.T) {
	// Create a server with multiple functions that match same short name.
	callGraph := core.NewCallGraph()

	callGraph.Functions["pkg1.handler"] = &graph.Node{
		ID: "1", Name: "handler", File: "/pkg1/handler.py", LineNumber: 1,
	}
	callGraph.Functions["pkg2.handler"] = &graph.Node{
		ID: "2", Name: "handler", File: "/pkg2/handler.py", LineNumber: 1,
	}
	callGraph.Functions["main.caller"] = &graph.Node{
		ID: "3", Name: "caller", File: "/main.py", LineNumber: 1,
	}

	callGraph.ReverseEdges["pkg1.handler"] = []string{"main.caller"}
	callGraph.CallSites["main.caller"] = []core.CallSite{
		{Target: "handler", TargetFQN: "pkg1.handler", Location: core.Location{Line: 5, Column: 4}},
	}

	server := NewServer("/test", "3.11", callGraph, &core.ModuleRegistry{
		Modules: map[string]string{}, FileToModule: map[string]string{}, ShortNames: map[string][]string{},
	}, nil, time.Second)

	result, isError := server.toolGetCallers(map[string]interface{}{"function": "handler"})

	assert.False(t, isError)
	// Should have a note about multiple matches.
	assert.Contains(t, result, "note")
	assert.Contains(t, result, "Multiple matches")
}

func TestToolGetCallers_NilCallerNode(t *testing.T) {
	// Create a server with a reverse edge pointing to non-existent function.
	callGraph := core.NewCallGraph()

	callGraph.Functions["target.func"] = &graph.Node{
		ID: "1", Name: "func", File: "/target.py", LineNumber: 1,
	}
	// This caller is in ReverseEdges but not in Functions.
	callGraph.ReverseEdges["target.func"] = []string{"nonexistent.caller"}

	server := NewServer("/test", "3.11", callGraph, &core.ModuleRegistry{
		Modules: map[string]string{}, FileToModule: map[string]string{}, ShortNames: map[string][]string{},
	}, nil, time.Second)

	result, isError := server.toolGetCallers(map[string]interface{}{"function": "func"})

	// Should still succeed but skip the nil caller.
	assert.False(t, isError)
	assert.Contains(t, result, "pagination")
	assert.Contains(t, result, `"total": 0`)
}

func TestToolFindSymbol_SubstringMatch(t *testing.T) {
	server := createExtendedTestServer()

	// "valid" should match "validate_user" via substring.
	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "valid"})

	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
}

func TestToolFindSymbol_FQNSubstringMatch(t *testing.T) {
	server := createExtendedTestServer()

	// "myapp.auth" should match via FQN substring.
	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "myapp.auth"})

	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
}
