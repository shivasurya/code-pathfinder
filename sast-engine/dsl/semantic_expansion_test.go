package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandSource_HTTPInput_AllFrameworks(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_input", "")
	assert.NotEmpty(t, expanded, "http_input should expand to matchers")

	// Should contain Django (HttpRequest), Flask (Request), and generic patterns
	hasHttpRequest := false
	hasRequest := false
	hasGeneric := false
	for _, m := range expanded {
		if m["receiverType"] == "HttpRequest" {
			hasHttpRequest = true
		}
		if m["receiverType"] == "Request" {
			hasRequest = true
		}
		if m["type"] == "call_matcher" {
			pats, ok := m["patterns"].([]any)
			if ok {
				for _, p := range pats {
					if p == "request.get" {
						hasGeneric = true
					}
				}
			}
		}
	}
	assert.True(t, hasHttpRequest, "should contain Django HttpRequest patterns")
	assert.True(t, hasRequest, "should contain Flask Request patterns")
	assert.True(t, hasGeneric, "should contain generic request.get pattern")
}

func TestExpandSource_HTTPInput_DjangoOnly(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_input", "django")

	// Should contain Django but not Flask-specific patterns
	for _, m := range expanded {
		rt, _ := m["receiverType"].(string)
		if rt == "Request" {
			t.Errorf("Django-only expansion should not contain Flask Request patterns, got %v", m)
		}
	}
}

func TestExpandSource_HTTPInput_FlaskOnly(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_input", "flask")

	for _, m := range expanded {
		rt, _ := m["receiverType"].(string)
		if rt == "HttpRequest" || rt == "QueryDict" {
			t.Errorf("Flask-only expansion should not contain Django patterns, got %v", m)
		}
	}
}

func TestExpandSource_HTTPParams(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_params", "")
	assert.Len(t, expanded, 8, "http_params should expand to 8 matchers")
}

func TestExpandSource_HTTPBody(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_body", "")
	assert.Len(t, expanded, 4, "http_body should expand to 4 matchers")
}

func TestExpandSource_HTTPHeaders(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_headers", "")
	assert.Len(t, expanded, 3, "http_headers should expand to 3 matchers")
}

func TestExpandSource_HTTPCookies(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("http_cookies", "")
	assert.Len(t, expanded, 2, "http_cookies should expand to 2 matchers")
}

func TestExpandSource_FileRead(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("file_read", "")
	assert.Len(t, expanded, 6, "file_read should expand to 6 matchers")
}

func TestExpandSource_FilePath(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("file_path", "")
	assert.Len(t, expanded, 4, "file_path should expand to 4 matchers")
}

func TestExpandSource_EnvVars(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("env_vars", "")
	assert.Len(t, expanded, 3, "env_vars should expand to 3 matchers")
}

func TestExpandSource_CliArgs(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("cli_args", "")
	assert.Len(t, expanded, 3, "cli_args should expand to 3 matchers")
}

func TestExpandSource_DatabaseResult(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("database_result", "")
	assert.Len(t, expanded, 6, "database_result should expand to 6 matchers")
}

func TestExpandSource_UnknownCategory(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSource("nonexistent_category", "")
	assert.Nil(t, expanded, "unknown category should return nil")
}

func TestExpandSink_SQLExecution(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("sql_execution", "")
	assert.Len(t, expanded, 8, "sql_execution should expand to 8 matchers")

	for _, m := range expanded {
		if m["type"] == "type_constrained_call" {
			assert.Equal(t, "none", m["fallbackMode"],
				"SQL sinks must use fallback=none for %v.%v", m["receiverType"], m["methodName"])
		}
	}
}

func TestExpandSink_CommandExecution(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("command_execution", "")
	assert.Len(t, expanded, 7, "command_execution should expand to 7 matchers")

	for _, m := range expanded {
		assert.Equal(t, "call_matcher", m["type"])
	}
}

func TestExpandSink_CodeExecution(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("code_execution", "")
	assert.Len(t, expanded, 4, "code_execution should expand to 4 matchers")
}

func TestExpandSink_TemplateRender(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("template_render", "")
	assert.Len(t, expanded, 4, "template_render should expand to 4 matchers")
}

func TestExpandSink_XPathQuery(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("xpath_query", "")
	assert.Len(t, expanded, 5, "xpath_query should expand to 5 matchers")
}

func TestExpandSink_LDAPQuery(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("ldap_query", "")
	assert.Len(t, expanded, 3, "ldap_query should expand to 3 matchers")
}

func TestExpandSink_FileWrite(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("file_write", "")
	assert.Len(t, expanded, 4, "file_write should expand to 4 matchers")
}

func TestExpandSink_FileOpen(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("file_open", "")
	assert.Len(t, expanded, 2, "file_open should expand to 2 matchers")
}

func TestExpandSink_PathOperation(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("path_operation", "")
	assert.Len(t, expanded, 9, "path_operation should expand to 9 matchers")
}

func TestExpandSink_HTTPRequest(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("http_request", "")
	assert.Len(t, expanded, 9, "http_request should expand to 9 matchers")
}

func TestExpandSink_SocketConnect(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("socket_connect", "")
	assert.Len(t, expanded, 2, "socket_connect should expand to 2 matchers")
}

func TestExpandSink_Deserialize(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("deserialize", "")
	assert.Len(t, expanded, 7, "deserialize should expand to 7 matchers")
}

func TestExpandSink_UnknownCategory(t *testing.T) {
	expander := NewSemanticExpander()
	expanded := expander.ExpandSink("nonexistent_sink", "")
	assert.Nil(t, expanded, "unknown sink category should return nil")
}

// buildSemanticTestCallGraph creates a CallGraph with call sites
// that should match expanded semantic matchers.
func buildSemanticTestCallGraph() *core.CallGraph {
	cg := core.NewCallGraph()

	cg.CallSites["app.views.index"] = []core.CallSite{
		{
			Target:   "request.get",
			Location: core.Location{File: "views.py", Line: 5},
		},
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "views.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.95,
		},
		{
			Target:   "os.system",
			Location: core.Location{File: "views.py", Line: 15},
		},
		{
			Target:   "eval",
			Location: core.Location{File: "views.py", Line: 20},
		},
	}

	cg.CallSites["app.utils.helper"] = []core.CallSite{
		{
			Target:   "os.getenv",
			Location: core.Location{File: "utils.py", Line: 3},
		},
	}

	cg.Edges = map[string][]string{
		"app.views.index":  {"app.utils.helper"},
		"app.utils.helper": {},
	}

	return cg
}

func TestSemanticSource_InDataflow(t *testing.T) {
	cg := buildSemanticTestCallGraph()

	ir := &DataflowIR{
		Type: "dataflow",
		Sources: []any{
			map[string]any{
				"type":     "semantic_source",
				"category": "http_params",
			},
		},
		Sinks: []any{
			map[string]any{
				"type":      "call_matcher",
				"patterns":  []any{"eval"},
				"wildcard":  false,
				"matchMode": "any",
			},
		},
		Sanitizers:  []any{},
		Propagation: []PropagationIR{},
		Scope:       "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.NotEmpty(t, detections, "should detect flow from semantic source to sink")
}

func TestSemanticSink_InDataflow(t *testing.T) {
	cg := buildSemanticTestCallGraph()

	ir := &DataflowIR{
		Type: "dataflow",
		Sources: []any{
			map[string]any{
				"type":      "call_matcher",
				"patterns":  []any{"request.get"},
				"wildcard":  false,
				"matchMode": "any",
			},
		},
		Sinks: []any{
			map[string]any{
				"type":     "semantic_sink",
				"category": "sql_execution",
			},
		},
		Sanitizers:  []any{},
		Propagation: []PropagationIR{},
		Scope:       "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.NotEmpty(t, detections, "should detect flow from source to semantic sink")
}

func TestSemanticSource_MatchesEquivalentExpanded(t *testing.T) {
	cg := buildSemanticTestCallGraph()

	// Approach 1: Semantic compact IR
	semanticMatchers := []any{
		map[string]any{
			"type":     "semantic_source",
			"category": "env_vars",
		},
	}

	// Approach 2: Pre-expanded (like Python sources.py would emit)
	expandedMatchers := []any{
		map[string]any{
			"type": "logic_or",
			"matchers": []any{
				map[string]any{"type": "call_matcher", "patterns": []any{"os.getenv"}, "wildcard": false, "matchMode": "any"},
				map[string]any{"type": "call_matcher", "patterns": []any{"os.environ.get"}, "wildcard": false, "matchMode": "any"},
				map[string]any{"type": "call_matcher", "patterns": []any{"os.environ.*"}, "wildcard": true, "matchMode": "any"},
			},
		},
	}

	executor := NewDataflowExecutor(&DataflowIR{}, cg)

	semanticResults := executor.findMatchingCallsPolymorphic(semanticMatchers)
	expandedResults := executor.findMatchingCallsPolymorphic(expandedMatchers)

	assert.Len(t, semanticResults, 1, "semantic should find 1 match")
	assert.Len(t, expandedResults, 1, "expanded should find 1 match")
	if len(semanticResults) > 0 && len(expandedResults) > 0 {
		assert.Equal(t, semanticResults[0].FunctionFQN, expandedResults[0].FunctionFQN)
		assert.Equal(t, semanticResults[0].Line, expandedResults[0].Line)
	}
}

func TestSemanticMatcher_ExecuteRule(t *testing.T) {
	cg := buildSemanticTestCallGraph()
	loader := NewRuleLoader("")

	t.Run("semantic_source via ExecuteRule", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type":     "semantic_source",
				"category": "env_vars",
			},
		}
		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.NotEmpty(t, detections, "should find os.getenv via semantic_source")
	})

	t.Run("semantic_sink via ExecuteRule", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type":     "semantic_sink",
				"category": "command_execution",
			},
		}
		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.NotEmpty(t, detections, "should find os.system via semantic_sink")
	})

	t.Run("unknown category returns empty", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type":     "semantic_source",
				"category": "nonexistent",
			},
		}
		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.Empty(t, detections, "unknown category should return empty, not error")
	})
}

func TestHelpers_ContainsWildcard(t *testing.T) {
	assert.True(t, containsWildcard([]string{"os.environ.*"}))
	assert.True(t, containsWildcard([]string{"request.args.*", "normal"}))
	assert.False(t, containsWildcard([]string{"os.system"}))
	assert.False(t, containsWildcard([]string{}))
}

func TestHelpers_TypeConstrainedMap(t *testing.T) {
	m := typeConstrainedMap("Cursor", "execute", "none")
	assert.Equal(t, "type_constrained_call", m["type"])
	assert.Equal(t, "Cursor", m["receiverType"])
	assert.Equal(t, "execute", m["methodName"])
	assert.Equal(t, 0.5, m["minConfidence"])
	assert.Equal(t, "none", m["fallbackMode"])
}

func TestHelpers_CallMatcherMap(t *testing.T) {
	m := callMatcherMap("os.system")
	assert.Equal(t, "call_matcher", m["type"])
	pats := m["patterns"].([]any)
	assert.Len(t, pats, 1)
	assert.Equal(t, "os.system", pats[0])
	assert.Equal(t, false, m["wildcard"])

	m2 := callMatcherMap("request.args.*")
	assert.Equal(t, true, m2["wildcard"])
}
