package mcp

import (
	"encoding/json"
	"fmt"
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

	// Verify stats structure (enhanced with Phase 3).
	stats, ok := parsed["stats"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, stats, "total_symbols")
	assert.Contains(t, stats, "call_edges")
	assert.Contains(t, stats, "modules")
	assert.Contains(t, stats, "files")
	assert.Contains(t, stats, "class_fields")

	// Verify new enhanced fields.
	assert.Contains(t, parsed, "symbols_by_type")
	assert.Contains(t, parsed, "symbols_by_lsp_kind")
	assert.Contains(t, parsed, "top_modules")
	assert.Contains(t, parsed, "health")

	// Verify symbols_by_type has data.
	symbolsByType, ok := parsed["symbols_by_type"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, symbolsByType)

	// Verify symbols_by_lsp_kind has data.
	symbolsByLSPKind, ok := parsed["symbols_by_lsp_kind"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, symbolsByLSPKind)

	t.Logf("Enhanced Index Info:\n%s", result)
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
	assert.Contains(t, result, "No symbols found")
	assert.Contains(t, result, "suggestion")
}

func TestToolFindSymbol_EmptyName(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": ""})

	assert.True(t, isError)
	assert.Contains(t, result, "At least one filter required")
}

// TestToolFindSymbol_AttributeFound tests finding a class attribute by exact name.
func TestToolFindSymbol_AttributeFound(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "email"})

	assert.False(t, isError)
	assert.Contains(t, result, "email")
	assert.Contains(t, result, "class_field")
	assert.Contains(t, result, "myapp.models.User.email")
	assert.Contains(t, result, "builtins.str")
}

// TestToolFindSymbol_AttributePartialMatch tests finding attributes via partial name matching.
func TestToolFindSymbol_AttributePartialMatch(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "name"})

	// Should find username attribute via partial match.
	assert.False(t, isError)
	assert.Contains(t, result, "username")
	assert.Contains(t, result, "myapp.models.User.username")
}

// TestToolFindSymbol_AttributeWithConfidence tests that confidence scores are included.
func TestToolFindSymbol_AttributeWithConfidence(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "email"})

	assert.False(t, isError)
	// Verify confidence is included in output.
	assert.Contains(t, result, "confidence")
	assert.Contains(t, result, "0.9")
}

// TestToolFindSymbol_AttributeAndFunction tests finding both attributes and functions.
func TestToolFindSymbol_AttributeAndFunction(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "User"})

	assert.False(t, isError)
	// Should find both class User and possibly attributes containing "User".
	assert.Contains(t, result, "User")
}

// TestToolFindSymbol_NoAttributeRegistry tests behavior when no attributes are indexed.
func TestToolFindSymbol_NoAttributeRegistry(t *testing.T) {
	server := createTestServer() // No attributes registry.

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate_user"})

	// Should still find functions.
	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
}

// TestToolFindSymbol_NilAttributes tests handling when Attributes field is nil.
func TestToolFindSymbol_NilAttributes(t *testing.T) {
	server := createTestServer()
	server.callGraph.Attributes = nil // Explicitly set to nil.

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate_user"})

	// Should still work for functions.
	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
}

// TestToolFindSymbol_WrongTypeAttributes tests handling when Attributes is wrong type.
func TestToolFindSymbol_WrongTypeAttributes(t *testing.T) {
	server := createTestServer()
	server.callGraph.Attributes = "not a registry" // Wrong type.

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate_user"})

	// Should still work for functions without crashing.
	assert.False(t, isError)
	assert.Contains(t, result, "validate_user")
}

// TestToolFindSymbol_AttributeNoType tests attributes without type information.
func TestToolFindSymbol_AttributeNoType(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "id"})

	// Should find attribute even without type info.
	assert.False(t, isError)
	assert.Contains(t, result, "id")
	assert.Contains(t, result, "myapp.models.User.id")
	// Should not crash when Type is nil.
}

// TestToolFindSymbol_AttributeEmptyType tests attributes with empty TypeFQN.
func TestToolFindSymbol_AttributeEmptyType(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "created_at"})

	// Should find attribute even with empty type.
	assert.False(t, isError)
	assert.Contains(t, result, "created_at")
	assert.Contains(t, result, "myapp.models.User.created_at")
}

// TestToolFindSymbol_AttributeNoLocation tests attributes without location information.
func TestToolFindSymbol_AttributeNoLocation(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "id"})

	// Should find attribute even without location.
	assert.False(t, isError)
	assert.Contains(t, result, "id")
	// Location field should not cause crash when nil.
}

// TestToolFindSymbol_AttributeNoAssignedIn tests attributes without AssignedIn information.
func TestToolFindSymbol_AttributeNoAssignedIn(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "created_at"})

	// Should find attribute even without AssignedIn.
	assert.False(t, isError)
	assert.Contains(t, result, "created_at")
	// Should not include assigned_in field when empty.
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

	assert.Len(t, tools, 8) // Updated for Phase 3B: added find_module and list_modules

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
// ============================================================================
// Phase 3B Tests: LSP Symbol Kind Mapping + Module Search
// ============================================================================

// TestGetSymbolKind tests mapping of all 12 Python symbol types to LSP kinds.
func TestGetSymbolKind(t *testing.T) {
	tests := []struct {
		symbolType       string
		expectedKind     int
		expectedKindName string
	}{
		// Function types
		{"function_definition", SymbolKindFunction, "Function"},
		{"method", SymbolKindMethod, "Method"},
		{"constructor", SymbolKindConstructor, "Constructor"},
		{"property", SymbolKindProperty, "Property"},
		{"special_method", SymbolKindOperator, "Operator"},

		// Class types
		{"class_definition", SymbolKindClass, "Class"},
		{"interface", SymbolKindInterface, "Interface"},
		{"enum", SymbolKindEnum, "Enum"},
		{"dataclass", SymbolKindStruct, "Struct"},

		// Variable types
		{"module_variable", SymbolKindVariable, "Variable"},
		{"constant", SymbolKindConstant, "Constant"},
		{"class_field", SymbolKindField, "Field"},

		// Java types (for compatibility)
		{"method_declaration", SymbolKindMethod, "Method"},
		{"class_declaration", SymbolKindClass, "Class"},
		{"variable_declaration", SymbolKindVariable, "Variable"},

		// Unknown type
		{"unknown_type", SymbolKindVariable, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.symbolType, func(t *testing.T) {
			kind, kindName := getSymbolKind(tt.symbolType)
			assert.Equal(t, tt.expectedKind, kind,
				"Symbol type '%s' should map to LSP kind %d", tt.symbolType, tt.expectedKind)
			assert.Equal(t, tt.expectedKindName, kindName,
				"Symbol type '%s' should have kind name '%s'", tt.symbolType, tt.expectedKindName)
		})
	}
}

// TestToolFindSymbol_SymbolKindFields verifies that symbol_kind and symbol_kind_name
// are ALWAYS present in find_symbol results.
func TestToolFindSymbol_SymbolKindFields(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "validate_user"})

	assert.False(t, isError)

	// Parse JSON response.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	// Get matches array.
	matches, ok := parsed["matches"].([]interface{})
	assert.True(t, ok, "Should have matches array")
	assert.Greater(t, len(matches), 0, "Should have at least one match")

	// Verify each match has symbol_kind and symbol_kind_name.
	for i, matchInterface := range matches {
		match, ok := matchInterface.(map[string]interface{})
		assert.True(t, ok, "Match %d should be an object", i)

		// Verify symbol_kind (integer).
		symbolKind, hasKind := match["symbol_kind"]
		assert.True(t, hasKind, "Match %d should have symbol_kind field", i)
		assert.IsType(t, float64(0), symbolKind, "Match %d symbol_kind should be a number", i)

		// Verify symbol_kind_name (string).
		symbolKindName, hasKindName := match["symbol_kind_name"]
		assert.True(t, hasKindName, "Match %d should have symbol_kind_name field", i)
		assert.IsType(t, "", symbolKindName, "Match %d symbol_kind_name should be a string", i)

		// Verify symbol_kind is a valid LSP kind (1-26).
		kindValue := symbolKind.(float64)
		assert.GreaterOrEqual(t, kindValue, float64(1), "Match %d symbol_kind should be >= 1", i)
		assert.LessOrEqual(t, kindValue, float64(26), "Match %d symbol_kind should be <= 26", i)

		t.Logf("Match %d: type=%s, symbol_kind=%v, symbol_kind_name=%s",
			i, match["type"], symbolKind, symbolKindName)
	}
}

// TestToolFindSymbol_ClassFieldSymbolKind verifies class fields have correct symbol kind.
func TestToolFindSymbol_ClassFieldSymbolKind(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "email"})

	assert.False(t, isError)

	// Parse JSON response.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	// Get matches array.
	matches, ok := parsed["matches"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(matches), 0)

	// Find the class_field match.
	var fieldMatch map[string]interface{}
	for _, matchInterface := range matches {
		match := matchInterface.(map[string]interface{})
		if match["type"] == "class_field" {
			fieldMatch = match
			break
		}
	}

	assert.NotNil(t, fieldMatch, "Should find class_field match")

	// Verify symbol kind for class_field.
	assert.Equal(t, float64(SymbolKindField), fieldMatch["symbol_kind"],
		"class_field should have symbol_kind = SymbolKindField (8)")
	assert.Equal(t, "Field", fieldMatch["symbol_kind_name"],
		"class_field should have symbol_kind_name = Field")

	// Verify it includes standard fields.
	assert.Equal(t, "class_field", fieldMatch["type"])
	assert.Contains(t, fieldMatch["fqn"], "email")
}

// TestToolFindModule_ExactMatch tests finding a module by exact FQN.
func TestToolFindModule_ExactMatch(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindModule("myapp.auth")

	assert.False(t, isError)
	assert.Contains(t, result, "myapp.auth")
	assert.Contains(t, result, "match_type")
	assert.Contains(t, result, "exact")
	assert.Contains(t, result, "functions_count")

	// Parse JSON to verify structure.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	assert.Equal(t, "myapp.auth", parsed["module_fqn"])
	assert.Equal(t, "exact", parsed["match_type"])
	assert.Contains(t, parsed, "file_path")
	assert.Contains(t, parsed, "functions_count")
}

// TestToolFindModule_PartialMatch tests finding modules by partial name.
func TestToolFindModule_PartialMatch(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindModule("auth")

	assert.False(t, isError)
	assert.Contains(t, result, "auth")

	// Parse JSON to verify structure.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	// Should have matches array for partial matches.
	if matches, ok := parsed["matches"].([]interface{}); ok {
		assert.Greater(t, len(matches), 0, "Should have at least one match")
		firstMatch := matches[0].(map[string]interface{})
		assert.Contains(t, firstMatch["module_fqn"], "auth")
		assert.Equal(t, "partial", firstMatch["match_type"])
	} else {
		// Or single exact/short match.
		assert.Contains(t, parsed, "module_fqn")
		assert.Contains(t, parsed["module_fqn"], "auth")
	}
}

// TestToolFindModule_NotFound tests module not found error.
func TestToolFindModule_NotFound(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindModule("nonexistent_module_xyz")

	assert.True(t, isError)
	assert.Contains(t, result, "not found")
	assert.Contains(t, result, "suggestion")
}

// TestToolFindModule_EmptyName tests empty module name error.
func TestToolFindModule_EmptyName(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolFindModule("")

	assert.True(t, isError)
	assert.Contains(t, result, "required")
}

// TestToolListModules tests listing all modules.
func TestToolListModules(t *testing.T) {
	server := createTestServer()

	result, isError := server.toolListModules()

	assert.False(t, isError)
	assert.Contains(t, result, "modules")
	assert.Contains(t, result, "total_modules")

	// Parse JSON to verify structure.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	// Verify modules array.
	modules, ok := parsed["modules"].([]interface{})
	assert.True(t, ok, "Should have modules array")
	assert.Greater(t, len(modules), 0, "Should have at least one module")

	// Verify total_modules count.
	totalModules, ok := parsed["total_modules"].(float64)
	assert.True(t, ok, "Should have total_modules count")
	assert.Equal(t, float64(len(modules)), totalModules, "total_modules should match array length")

	// Verify each module has required fields.
	for i, moduleInterface := range modules {
		module, ok := moduleInterface.(map[string]interface{})
		assert.True(t, ok, "Module %d should be an object", i)

		assert.Contains(t, module, "module_fqn", "Module %d should have module_fqn", i)
		assert.Contains(t, module, "file_path", "Module %d should have file_path", i)
		assert.Contains(t, module, "functions_count", "Module %d should have functions_count", i)

		// Verify functions_count is a number.
		assert.IsType(t, float64(0), module["functions_count"],
			"Module %d functions_count should be a number", i)

		t.Logf("Module %d: fqn=%s, functions=%v",
			i, module["module_fqn"], module["functions_count"])
	}
}

// TestToolFindSymbol_InterfaceWithSymbolKind tests interface symbol kind.
func TestToolFindSymbol_InterfaceWithSymbolKind(t *testing.T) {
	// Create a test server with an interface class.
	callGraph := core.NewCallGraph()

	interfaceNode := &graph.Node{
		Name:       "IDrawable",
		Type:       "interface",
		File:       "/test/interfaces.py",
		LineNumber: 10,
		Interface:  []string{"Protocol"},
	}
	callGraph.Functions["myapp.interfaces.IDrawable"] = interfaceNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.interfaces"] = "/test/interfaces.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "IDrawable"})

	assert.False(t, isError)

	// Parse JSON.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	matches := parsed["matches"].([]interface{})
	assert.Greater(t, len(matches), 0)

	match := matches[0].(map[string]interface{})

	// Verify interface has correct symbol kind.
	assert.Equal(t, float64(SymbolKindInterface), match["symbol_kind"],
		"Interface should have symbol_kind = SymbolKindInterface (11)")
	assert.Equal(t, "Interface", match["symbol_kind_name"])

	// Verify interfaces field is present.
	assert.Contains(t, match, "interfaces")
	interfaces := match["interfaces"].([]interface{})
	assert.Equal(t, "Protocol", interfaces[0])
}

// TestToolFindSymbol_EnumWithSymbolKind tests enum symbol kind.
func TestToolFindSymbol_EnumWithSymbolKind(t *testing.T) {
	callGraph := core.NewCallGraph()

	enumNode := &graph.Node{
		Name:       "Color",
		Type:       "enum",
		File:       "/test/enums.py",
		LineNumber: 5,
		Interface:  []string{"Enum"},
	}
	callGraph.Functions["myapp.enums.Color"] = enumNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.enums"] = "/test/enums.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "Color"})

	assert.False(t, isError)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(result), &parsed)

	match := parsed["matches"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, float64(SymbolKindEnum), match["symbol_kind"],
		"Enum should have symbol_kind = SymbolKindEnum (10)")
	assert.Equal(t, "Enum", match["symbol_kind_name"])
}

// TestToolFindSymbol_DataclassWithSymbolKind tests dataclass symbol kind.
func TestToolFindSymbol_DataclassWithSymbolKind(t *testing.T) {
	callGraph := core.NewCallGraph()

	dataclassNode := &graph.Node{
		Name:       "Point",
		Type:       "dataclass",
		File:       "/test/models.py",
		LineNumber: 8,
		Annotation: []string{"dataclass"},
	}
	callGraph.Functions["myapp.models.Point"] = dataclassNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.models"] = "/test/models.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "Point"})

	assert.False(t, isError)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(result), &parsed)

	match := parsed["matches"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, float64(SymbolKindStruct), match["symbol_kind"],
		"Dataclass should have symbol_kind = SymbolKindStruct (23)")
	assert.Equal(t, "Struct", match["symbol_kind_name"])
	assert.Contains(t, match, "decorators")
}

// TestToolFindSymbol_ConstructorWithSymbolKind tests constructor symbol kind.
func TestToolFindSymbol_ConstructorWithSymbolKind(t *testing.T) {
	callGraph := core.NewCallGraph()

	constructorNode := &graph.Node{
		Name:       "__init__",
		Type:       "constructor",
		File:       "/test/user.py",
		LineNumber: 15,
	}
	callGraph.Functions["myapp.models.User.__init__"] = constructorNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.models"] = "/test/user.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "__init__"})

	assert.False(t, isError)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(result), &parsed)

	match := parsed["matches"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, float64(SymbolKindConstructor), match["symbol_kind"],
		"Constructor should have symbol_kind = SymbolKindConstructor (9)")
	assert.Equal(t, "Constructor", match["symbol_kind_name"])
}

// TestToolFindSymbol_PropertyWithSymbolKind tests property symbol kind.
func TestToolFindSymbol_PropertyWithSymbolKind(t *testing.T) {
	callGraph := core.NewCallGraph()

	propertyNode := &graph.Node{
		Name:       "name",
		Type:       "property",
		File:       "/test/user.py",
		LineNumber: 20,
		Annotation: []string{"property"},
	}
	callGraph.Functions["myapp.models.User.name"] = propertyNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.models"] = "/test/user.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "name"})

	assert.False(t, isError)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(result), &parsed)

	matches := parsed["matches"].([]interface{})
	// Find the property match (there might be other "name" matches).
	var propertyMatch map[string]interface{}
	for _, m := range matches {
		match := m.(map[string]interface{})
		if match["type"] == "property" {
			propertyMatch = match
			break
		}
	}

	assert.NotNil(t, propertyMatch, "Should find property match")
	assert.Equal(t, float64(SymbolKindProperty), propertyMatch["symbol_kind"],
		"Property should have symbol_kind = SymbolKindProperty (7)")
	assert.Equal(t, "Property", propertyMatch["symbol_kind_name"])
}

// TestToolFindSymbol_SpecialMethodWithSymbolKind tests special method symbol kind.
func TestToolFindSymbol_SpecialMethodWithSymbolKind(t *testing.T) {
	callGraph := core.NewCallGraph()

	specialMethodNode := &graph.Node{
		Name:       "__str__",
		Type:       "special_method",
		File:       "/test/user.py",
		LineNumber: 25,
	}
	callGraph.Functions["myapp.models.User.__str__"] = specialMethodNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.models"] = "/test/user.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "__str__"})

	assert.False(t, isError)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(result), &parsed)

	match := parsed["matches"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, float64(SymbolKindOperator), match["symbol_kind"],
		"Special method should have symbol_kind = SymbolKindOperator (25)")
	assert.Equal(t, "Operator", match["symbol_kind_name"])
}

// TestToolFindSymbol_MethodWithSymbolKind tests method symbol kind.
func TestToolFindSymbol_MethodWithSymbolKind(t *testing.T) {
	callGraph := core.NewCallGraph()

	methodNode := &graph.Node{
		Name:       "get_profile",
		Type:       "method",
		File:       "/test/user.py",
		LineNumber: 30,
	}
	callGraph.Functions["myapp.models.User.get_profile"] = methodNode

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.models"] = "/test/user.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolFindSymbol(map[string]interface{}{"name": "get_profile"})

	assert.False(t, isError)

	var parsed map[string]interface{}
	json.Unmarshal([]byte(result), &parsed)

	match := parsed["matches"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, float64(SymbolKindMethod), match["symbol_kind"],
		"Method should have symbol_kind = SymbolKindMethod (6)")
	assert.Equal(t, "Method", match["symbol_kind_name"])
	assert.Equal(t, "method", match["type"])
}
// TestToolGetIndexInfo_Enhanced demonstrates the enhanced index info with all symbol types.
func TestToolGetIndexInfo_Enhanced(t *testing.T) {
	// Create a comprehensive test server with all 12 symbol types.
	callGraph := core.NewCallGraph()

	// Add various symbol types.
	symbolTypes := []struct {
		name string
		typ  string
	}{
		{"login", "function_definition"},
		{"logout", "function_definition"},
		{"get_profile", "method"},
		{"validate_email", "method"},
		{"process_payment", "method"},
		{"__init__", "constructor"},
		{"name", "property"},
		{"email", "property"},
		{"__str__", "special_method"},
		{"__add__", "special_method"},
		{"User", "class_definition"},
		{"Product", "class_definition"},
		{"IDrawable", "interface"},
		{"IStorage", "interface"},
		{"Color", "enum"},
		{"Priority", "enum"},
		{"Point", "dataclass"},
		{"Rectangle", "dataclass"},
	}

	for i, s := range symbolTypes {
		node := &graph.Node{
			Name:       s.name,
			Type:       s.typ,
			File:       fmt.Sprintf("/test/file%d.py", i%3),
			LineNumber: uint32(10 + i),
		}
		callGraph.Functions[fmt.Sprintf("myapp.module%d.%s", i%5, s.name)] = node
	}

	// Add module variables and constants (these would be in separate index).
	// For this test, we'll just show the function-based symbols.

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.module0"] = "/test/module0.py"
	moduleRegistry.Modules["myapp.module1"] = "/test/module1.py"
	moduleRegistry.Modules["myapp.module2"] = "/test/module2.py"
	moduleRegistry.Modules["myapp.module3"] = "/test/module3.py"
	moduleRegistry.Modules["myapp.module4"] = "/test/module4.py"

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)

	result, isError := server.toolGetIndexInfo()

	assert.False(t, isError)

	// Parse and display the comprehensive result.
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err)

	// Print the result for demonstration.
	prettyJSON, _ := json.MarshalIndent(parsed, "", "  ")
	t.Logf("\n\n=== ENHANCED INDEX INFO EXAMPLE ===\n%s\n", string(prettyJSON))

	// Verify all sections are present.
	assert.Contains(t, parsed, "stats")
	assert.Contains(t, parsed, "symbols_by_type")
	assert.Contains(t, parsed, "symbols_by_lsp_kind")
	assert.Contains(t, parsed, "top_modules")
	assert.Contains(t, parsed, "health")

	// Verify symbols_by_type has all the types we added.
	symbolsByType := parsed["symbols_by_type"].(map[string]interface{})
	assert.Contains(t, symbolsByType, "function_definition")
	assert.Contains(t, symbolsByType, "method")
	assert.Contains(t, symbolsByType, "constructor")
	assert.Contains(t, symbolsByType, "property")
	assert.Contains(t, symbolsByType, "special_method")
	assert.Contains(t, symbolsByType, "class_definition")
	assert.Contains(t, symbolsByType, "interface")
	assert.Contains(t, symbolsByType, "enum")
	assert.Contains(t, symbolsByType, "dataclass")

	// Verify LSP kind breakdown.
	symbolsByLSPKind := parsed["symbols_by_lsp_kind"].(map[string]interface{})
	assert.Contains(t, symbolsByLSPKind, "Function")
	assert.Contains(t, symbolsByLSPKind, "Method")
	assert.Contains(t, symbolsByLSPKind, "Constructor")
	assert.Contains(t, symbolsByLSPKind, "Property")
	assert.Contains(t, symbolsByLSPKind, "Operator")
	assert.Contains(t, symbolsByLSPKind, "Class")
	assert.Contains(t, symbolsByLSPKind, "Interface")
	assert.Contains(t, symbolsByLSPKind, "Enum")
	assert.Contains(t, symbolsByLSPKind, "Struct")

	t.Logf("\n=== Symbol Type Breakdown ===")
	for typ, count := range symbolsByType {
		t.Logf("  %s: %v", typ, count)
	}

	t.Logf("\n=== LSP Symbol Kind Breakdown ===")
	for kind, count := range symbolsByLSPKind {
		t.Logf("  %s: %v", kind, count)
	}
}

// ========== Type Filtering Tests ==========

// createMultiTypeTestServer creates a server with multiple symbol types for type filtering tests.
func createMultiTypeTestServer() *Server {
	callGraph := core.NewCallGraph()

	// Add function_definition.
	callGraph.Functions["myapp.utils.login"] = &graph.Node{
		ID:         "1",
		Type:       "function_definition",
		Name:       "login",
		File:       "/path/to/utils.py",
		LineNumber: 10,
	}

	callGraph.Functions["myapp.utils.logout"] = &graph.Node{
		ID:         "2",
		Type:       "function_definition",
		Name:       "logout",
		File:       "/path/to/utils.py",
		LineNumber: 20,
	}

	// Add method.
	callGraph.Functions["myapp.models.User.get_profile"] = &graph.Node{
		ID:         "3",
		Type:       "method",
		Name:       "get_profile",
		File:       "/path/to/models.py",
		LineNumber: 30,
	}

	callGraph.Functions["myapp.models.User.save"] = &graph.Node{
		ID:         "4",
		Type:       "method",
		Name:       "save",
		File:       "/path/to/models.py",
		LineNumber: 40,
	}

	// Add constructor.
	callGraph.Functions["myapp.models.User.__init__"] = &graph.Node{
		ID:         "5",
		Type:       "constructor",
		Name:       "__init__",
		File:       "/path/to/models.py",
		LineNumber: 5,
	}

	// Add property.
	callGraph.Functions["myapp.models.User.email"] = &graph.Node{
		ID:         "6",
		Type:       "property",
		Name:       "email",
		File:       "/path/to/models.py",
		LineNumber: 50,
	}

	// Add special_method.
	callGraph.Functions["myapp.models.User.__str__"] = &graph.Node{
		ID:         "7",
		Type:       "special_method",
		Name:       "__str__",
		File:       "/path/to/models.py",
		LineNumber: 60,
	}

	// Add class_definition.
	callGraph.Functions["myapp.models.User"] = &graph.Node{
		ID:         "8",
		Type:       "class_definition",
		Name:       "User",
		File:       "/path/to/models.py",
		LineNumber: 1,
	}

	callGraph.Functions["myapp.models.Product"] = &graph.Node{
		ID:         "9",
		Type:       "class_definition",
		Name:       "Product",
		File:       "/path/to/models.py",
		LineNumber: 100,
	}

	// Add interface.
	callGraph.Functions["myapp.interfaces.IDrawable"] = &graph.Node{
		ID:         "10",
		Type:       "interface",
		Name:       "IDrawable",
		File:       "/path/to/interfaces.py",
		LineNumber: 10,
	}

	// Add enum.
	callGraph.Functions["myapp.enums.Color"] = &graph.Node{
		ID:         "11",
		Type:       "enum",
		Name:       "Color",
		File:       "/path/to/enums.py",
		LineNumber: 5,
	}

	// Add dataclass.
	callGraph.Functions["myapp.models.Point"] = &graph.Node{
		ID:         "12",
		Type:       "dataclass",
		Name:       "Point",
		File:       "/path/to/models.py",
		LineNumber: 200,
	}

	moduleRegistry := core.NewModuleRegistry()
	moduleRegistry.Modules["myapp.utils"] = "/path/to/utils.py"
	moduleRegistry.Modules["myapp.models"] = "/path/to/models.py"

	return NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)
}

// Test: No filters provided (should error).
func TestToolFindSymbol_NoFilters(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{})

	assert.True(t, isError)
	assert.Contains(t, result, "At least one filter required")
}

// Test: Filter by single type.
func TestToolFindSymbol_FilterBySingleType(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{"type": "method"})

	assert.False(t, isError)
	assert.Contains(t, result, "get_profile")
	assert.Contains(t, result, "save")
	assert.NotContains(t, result, "login") // Should not include function_definition

	// Verify only methods are returned.
	var parsedResult map[string]interface{}
	json.Unmarshal([]byte(result), &parsedResult)
	matches := parsedResult["matches"].([]interface{})
	for _, match := range matches {
		m := match.(map[string]interface{})
		assert.Equal(t, "method", m["type"])
	}

	// Verify filters_applied.
	assert.Contains(t, result, "filters_applied")
	filtersApplied := parsedResult["filters_applied"].(map[string]interface{})
	assert.Equal(t, "method", filtersApplied["type"])
}

// Test: Filter by multiple types.
func TestToolFindSymbol_FilterByMultipleTypes(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"types": []interface{}{"interface", "enum"},
	})

	assert.False(t, isError)
	assert.Contains(t, result, "IDrawable")
	assert.Contains(t, result, "Color")
	assert.NotContains(t, result, "login")
	assert.NotContains(t, result, "User")

	// Verify filters_applied.
	assert.Contains(t, result, "filters_applied")
	assert.Contains(t, result, "types")
}

// Test: Combine name + type filters.
func TestToolFindSymbol_CombineNameAndType(t *testing.T) {
	server := createMultiTypeTestServer()

	// Search for anything named "User" but only class_definition type.
	result, isError := server.toolFindSymbol(map[string]interface{}{
		"name": "User",
		"type": "class_definition",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "myapp.models.User")
	assert.Contains(t, result, `"type": "class_definition"`)
	// Should NOT include User.__init__ (constructor) or User.get_profile (method).
	assert.NotContains(t, result, "__init__")
	assert.NotContains(t, result, "get_profile")
}

// Test: Combine name + types filters.
func TestToolFindSymbol_CombineNameAndTypes(t *testing.T) {
	server := createMultiTypeTestServer()

	// Search for anything with "User" in name, but only methods or constructors.
	result, isError := server.toolFindSymbol(map[string]interface{}{
		"name":  "User",
		"types": []interface{}{"method", "constructor"},
	})

	assert.False(t, isError)
	// Should include User.__init__ and User.get_profile, User.save.
	assert.Contains(t, result, "__init__")
	assert.Contains(t, result, "get_profile")
	assert.Contains(t, result, "save")
	// Should NOT include class User itself.
	parsedResult := map[string]interface{}{}
	json.Unmarshal([]byte(result), &parsedResult)
	matches := parsedResult["matches"].([]interface{})
	for _, match := range matches {
		m := match.(map[string]interface{})
		typ := m["type"].(string)
		assert.NotEqual(t, "class_definition", typ)
	}
}

// Test: Both type and types provided (should error).
func TestToolFindSymbol_BothTypeAndTypes(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type":  "method",
		"types": []interface{}{"function_definition"},
	})

	assert.True(t, isError)
	assert.Contains(t, result, "Cannot specify both")
}

// Test: Invalid type name (should error).
func TestToolFindSymbol_InvalidType(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "invalid_type_xyz",
	})

	assert.True(t, isError)
	assert.Contains(t, result, "Invalid symbol type")
	assert.Contains(t, result, "invalid_type_xyz")
	assert.Contains(t, result, "valid_types")
}

// Test: Invalid type in types array (should error).
func TestToolFindSymbol_InvalidTypeInArray(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"types": []interface{}{"method", "bad_type"},
	})

	assert.True(t, isError)
	assert.Contains(t, result, "Invalid symbol type")
	assert.Contains(t, result, "bad_type")
}

// Test: No results with type filter.
func TestToolFindSymbol_NoResultsWithTypeFilter(t *testing.T) {
	server := createMultiTypeTestServer()

	// Search for module_variable (not in test data).
	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "module_variable",
	})

	assert.True(t, isError)
	assert.Contains(t, result, "No symbols found")
	assert.Contains(t, result, "module_variable")
}

// Test: No results with combined filters.
func TestToolFindSymbol_NoResultsCombinedFilters(t *testing.T) {
	server := createMultiTypeTestServer()

	// Search for "login" but with type "method" (login is a function_definition).
	result, isError := server.toolFindSymbol(map[string]interface{}{
		"name": "login",
		"type": "method",
	})

	assert.True(t, isError)
	assert.Contains(t, result, "No symbols found")
}

// Test: Filter by constructor type.
func TestToolFindSymbol_FilterByConstructor(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "constructor",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "__init__")
	assert.Contains(t, result, `"type": "constructor"`)
	assert.NotContains(t, result, "get_profile")
}

// Test: Filter by property type.
func TestToolFindSymbol_FilterByProperty(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "property",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "email")
	assert.Contains(t, result, `"type": "property"`)
}

// Test: Filter by special_method type.
func TestToolFindSymbol_FilterBySpecialMethod(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "special_method",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "__str__")
	assert.Contains(t, result, `"type": "special_method"`)
}

// Test: Filter by class_definition type.
func TestToolFindSymbol_FilterByClassDefinition(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "class_definition",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "User")
	assert.Contains(t, result, "Product")
	assert.Contains(t, result, `"type": "class_definition"`)
	assert.NotContains(t, result, "get_profile") // Should not include methods
}

// Test: Filter by interface type.
func TestToolFindSymbol_FilterByInterface(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "interface",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "IDrawable")
	assert.Contains(t, result, `"type": "interface"`)
}

// Test: Filter by enum type.
func TestToolFindSymbol_FilterByEnum(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "enum",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "Color")
	assert.Contains(t, result, `"type": "enum"`)
}

// Test: Filter by dataclass type.
func TestToolFindSymbol_FilterByDataclass(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "dataclass",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "Point")
	assert.Contains(t, result, `"type": "dataclass"`)
}

// Test: Filter by function_definition type.
func TestToolFindSymbol_FilterByFunctionDefinition(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "function_definition",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "login")
	assert.Contains(t, result, "logout")
	assert.Contains(t, result, `"type": "function_definition"`)
	assert.NotContains(t, result, "get_profile") // Should not include methods
}

// Test: Filter class_field with type filter.
func TestToolFindSymbol_FilterClassFieldWithType(t *testing.T) {
	server := createTestServerWithAttributes()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "class_field",
	})

	assert.False(t, isError)
	assert.Contains(t, result, `"type": "class_field"`)
}

// Test: Filter excludes class_field when filtering for other types.
func TestToolFindSymbol_ExcludeClassFieldWhenFiltering(t *testing.T) {
	server := createTestServerWithAttributes()

	// Filter by method type only - should not include class fields.
	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "method",
	})

	// Should not error (may or may not find methods in this server).
	// The key is that class_field should not be included.
	if !isError {
		assert.NotContains(t, result, `"type": "class_field"`)
	}
}

// Test: Verify filters_applied in response for name only.
func TestToolFindSymbol_FiltersAppliedNameOnly(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"name": "User",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "filters_applied")

	// Parse JSON to verify filters_applied structure.
	var parsedResult map[string]interface{}
	json.Unmarshal([]byte(result), &parsedResult)
	filtersApplied := parsedResult["filters_applied"].(map[string]interface{})

	// Should have name but not type or types.
	assert.Equal(t, "User", filtersApplied["name"])
	_, hasType := filtersApplied["type"]
	assert.False(t, hasType, "filters_applied should not contain 'type' when not provided")
	_, hasTypes := filtersApplied["types"]
	assert.False(t, hasTypes, "filters_applied should not contain 'types' when not provided")
}

// Test: Verify filters_applied in response for type only.
func TestToolFindSymbol_FiltersAppliedTypeOnly(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type": "method",
	})

	assert.False(t, isError)
	assert.Contains(t, result, "filters_applied")
	assert.Contains(t, result, `"type": "method"`)

	// Parse JSON to verify name is not in filters_applied.
	var parsedResult map[string]interface{}
	json.Unmarshal([]byte(result), &parsedResult)
	filtersApplied := parsedResult["filters_applied"].(map[string]interface{})
	_, hasName := filtersApplied["name"]
	assert.False(t, hasName, "filters_applied should not contain 'name' when not provided")
}

// Test: Verify filters_applied in response for multiple types.
func TestToolFindSymbol_FiltersAppliedMultipleTypes(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"types": []interface{}{"method", "function_definition"},
	})

	assert.False(t, isError)
	assert.Contains(t, result, "filters_applied")
	assert.Contains(t, result, `"types"`)

	// Verify 'types' is an array in filters_applied (not 'type' as string).
	var parsedResult map[string]interface{}
	json.Unmarshal([]byte(result), &parsedResult)
	filtersApplied := parsedResult["filters_applied"].(map[string]interface{})
	types, hasTypes := filtersApplied["types"]
	assert.True(t, hasTypes, "filters_applied should contain 'types'")
	typesArray, ok := types.([]interface{})
	assert.True(t, ok, "types should be an array")
	assert.Equal(t, 2, len(typesArray))
}

// Test: Verify filters_applied in response for combined name + type.
func TestToolFindSymbol_FiltersAppliedCombined(t *testing.T) {
	server := createMultiTypeTestServer()

	result, _ := server.toolFindSymbol(map[string]interface{}{
		"name": "User",
		"type": "method",
	})

	// May or may not find results, but should have filters_applied.
	assert.Contains(t, result, "filters_applied")
	assert.Contains(t, result, `"name": "User"`)
	assert.Contains(t, result, `"type": "method"`)
}

// Test: Pagination works with type filters.
func TestToolFindSymbol_PaginationWithTypeFilter(t *testing.T) {
	server := createMultiTypeTestServer()

	result, isError := server.toolFindSymbol(map[string]interface{}{
		"type":  "function_definition",
		"limit": 1,
	})

	assert.False(t, isError)
	assert.Contains(t, result, "pagination")

	// Parse to verify pagination info.
	var parsedResult map[string]interface{}
	json.Unmarshal([]byte(result), &parsedResult)
	matches := parsedResult["matches"].([]interface{})
	assert.LessOrEqual(t, len(matches), 1, "Should respect limit")
	pagination := parsedResult["pagination"].(map[string]interface{})
	assert.NotNil(t, pagination)
}

// Test: maxInt helper function.
func TestMaxInt(t *testing.T) {
	assert.Equal(t, 5, maxInt(5, 3))
	assert.Equal(t, 10, maxInt(7, 10))
	assert.Equal(t, 0, maxInt(0, 0))
	assert.Equal(t, 1, maxInt(-5, 1))
}
