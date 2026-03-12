package dsl

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueryType_SQLInjection_EndToEnd simulates the full SQL injection detection
// pipeline: TypeConstrainedCallIR JSON → executor → detections.
// This mirrors the Python DSL rule:
//
//	flows(source=WebRequest.method("get", "args"), sink=DBCursor.method("execute"))
func TestQueryType_SQLInjection_EndToEnd(t *testing.T) {
	// Build a call graph that simulates a Flask app with SQLite.
	cg := core.NewCallGraph()
	cg.Edges = make(map[string][]string)

	// /search endpoint — vulnerable: source + sink in same function.
	cg.CallSites["app.search"] = []core.CallSite{
		{
			Target:                   "request.args.get",
			TargetFQN:               "flask.request.args.get",
			Location:                 core.Location{File: "app.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "flask.Request",
			TypeConfidence:           0.9,
		},
		{
			Target:    "sqlite3.connect",
			TargetFQN: "sqlite3.connect",
			Location:  core.Location{File: "app.py", Line: 6},
		},
		{
			Target:                   "cursor.execute",
			TargetFQN:               "sqlite3.Cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 8},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	// /safe endpoint — parameterized query (still detected by our matcher,
	// but arg constraint could exclude it in a more refined rule).
	cg.CallSites["app.safe_search"] = []core.CallSite{
		{
			Target:                   "request.args.get",
			TargetFQN:               "flask.request.args.get",
			Location:                 core.Location{File: "app.py", Line: 15},
			ResolvedViaTypeInference: true,
			InferredType:             "flask.Request",
			TypeConfidence:           0.9,
		},
		{
			Target:                   "cursor.execute",
			TargetFQN:               "sqlite3.Cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 18},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	// /no-sql endpoint — no source, only sink.
	cg.CallSites["app.no_sql"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			TargetFQN:               "sqlite3.Cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 25},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	// Build the dataflow IR as it would come from the Python DSL.
	sourceIR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"flask.Request"},
		MethodNames:   []string{"get", "args"},
		MinConfidence: 0.5,
		FallbackMode:  "name",
	}
	sinkIR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"sqlite3.Cursor"},
		MethodNames:   []string{"execute"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	ir := &DataflowIR{
		Sources:    toRawMessagesTyped(sourceIR),
		Sinks:      toRawMessagesTyped(sinkIR),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Should detect flows in search and safe_search (both have source+sink).
	// no_sql has no source, so no detection.
	searchDetections := filterByFQN(detections, "app.search")
	safeDetections := filterByFQN(detections, "app.safe_search")
	noSQLDetections := filterByFQN(detections, "app.no_sql")

	assert.NotEmpty(t, searchDetections, "/search should have detection")
	assert.NotEmpty(t, safeDetections, "/safe should have detection (basic matcher)")
	assert.Empty(t, noSQLDetections, "/no-sql should have no detection (no source)")

	// Verify search detection details.
	assert.Equal(t, 5, searchDetections[0].SourceLine)
	assert.Equal(t, 8, searchDetections[0].SinkLine)
	assert.Equal(t, "cursor.execute", searchDetections[0].SinkCall)
}

// TestQueryType_FQNBridge_EndToEnd tests detection via the FQN-to-receiver bridge
// (covers ~82% of calls resolved via import analysis but not type inference).
func TestQueryType_FQNBridge_EndToEnd(t *testing.T) {
	cg := core.NewCallGraph()

	// Call site resolved via import (TargetFQN set) but NOT via type inference.
	cg.CallSites["myapp.run_cmd"] = []core.CallSite{
		{
			Target:    "user_input",
			Location:  core.Location{File: "app.py", Line: 3},
		},
		{
			Target:    "system",
			TargetFQN: "os.system",
			Location:  core.Location{File: "app.py", Line: 5},
			// No type inference — FQN bridge should derive receiver "os".
		},
	}

	sourceJSON, _ := json.Marshal(CallMatcherIR{
		Type:     "call_matcher",
		Patterns: []string{"user_input"},
	})
	sinkJSON, _ := json.Marshal(TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"os"},
		MethodNames:   []string{"system"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
	})

	ir := &DataflowIR{
		Sources:    []json.RawMessage{sourceJSON},
		Sinks:      []json.RawMessage{sinkJSON},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 1, "FQN bridge should enable detection")
	assert.Equal(t, "myapp.run_cmd", detections[0].FunctionFQN)
	assert.Equal(t, 3, detections[0].SourceLine)
	assert.Equal(t, 5, detections[0].SinkLine)
}

// TestQueryType_OrLogic_WeakHash tests Or() logic operator end-to-end.
func TestQueryType_OrLogic_WeakHash(t *testing.T) {
	cg := core.NewCallGraph()

	// Function using md5.
	cg.CallSites["myapp.hash_md5"] = []core.CallSite{
		{
			Target:                   "hashlib.md5",
			TargetFQN:               "hashlib.md5",
			Location:                 core.Location{File: "crypto.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "hashlib",
			TypeConfidence:           0.9,
		},
	}

	// Function using sha256 (should NOT be detected).
	cg.CallSites["myapp.hash_sha256"] = []core.CallSite{
		{
			Target:                   "hashlib.sha256",
			TargetFQN:               "hashlib.sha256",
			Location:                 core.Location{File: "crypto.py", Line: 20},
			ResolvedViaTypeInference: true,
			InferredType:             "hashlib",
			TypeConfidence:           0.9,
		},
	}

	// Function using sha1.
	cg.CallSites["myapp.hash_sha1"] = []core.CallSite{
		{
			Target:                   "hashlib.sha1",
			TargetFQN:               "hashlib.sha1",
			Location:                 core.Location{File: "crypto.py", Line: 30},
			ResolvedViaTypeInference: true,
			InferredType:             "hashlib",
			TypeConfidence:           0.9,
		},
	}

	// Build Or IR: logic_or wrapping two type_constrained_call matchers.
	md5IR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"hashlib"},
		MethodNames:   []string{"md5"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}
	sha1IR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"hashlib"},
		MethodNames:   []string{"sha1"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	// Build matchers as map[string]any (as they would arrive from JSON unmarshaling).
	logicOrRule := &RuleIR{
		Matcher: map[string]any{
			"type": "logic_or",
			"matchers": []any{
				irToMap(md5IR),
				irToMap(sha1IR),
			},
		},
	}

	loader := NewRuleLoader("")
	detections, err := loader.ExecuteRule(logicOrRule, cg)
	require.NoError(t, err)

	// Should detect md5 and sha1, NOT sha256.
	md5Detections := filterByFQN(detections, "myapp.hash_md5")
	sha1Detections := filterByFQN(detections, "myapp.hash_sha1")
	sha256Detections := filterByFQN(detections, "myapp.hash_sha256")

	assert.NotEmpty(t, md5Detections, "md5 should be detected")
	assert.NotEmpty(t, sha1Detections, "sha1 should be detected")
	assert.Empty(t, sha256Detections, "sha256 should NOT be detected")
}

// TestQueryType_AndLogic tests And() logic operator — intersection of results.
// And intersects by (FunctionFQN, SourceLine), so both matchers must match the same call site.
func TestQueryType_AndLogic(t *testing.T) {
	cg := core.NewCallGraph()

	// A call site that matches both a wildcard pattern and an exact pattern.
	cg.CallSites["myapp.hash"] = []core.CallSite{
		{
			Target:                   "hashlib.md5",
			TargetFQN:               "hashlib.md5",
			Location:                 core.Location{File: "crypto.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "hashlib",
			TypeConfidence:           0.9,
		},
		{
			Target:                   "hashlib.sha256",
			TargetFQN:               "hashlib.sha256",
			Location:                 core.Location{File: "crypto.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "hashlib",
			TypeConfidence:           0.9,
		},
	}

	// And(hashlib.* wildcard, hashlib.md5 exact) should only return md5.
	wildcardIR := TypeConstrainedCallIR{
		Type:             "type_constrained_call",
		ReceiverPatterns: []string{"hashlib*"},
		MinConfidence:    0.5,
		FallbackMode:     "none",
	}
	exactIR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"hashlib"},
		MethodNames:   []string{"md5"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	logicAndRule := &RuleIR{
		Matcher: map[string]any{
			"type": "logic_and",
			"matchers": []any{
				irToMap(wildcardIR),
				irToMap(exactIR),
			},
		},
	}

	loader := NewRuleLoader("")
	detections, err := loader.ExecuteRule(logicAndRule, cg)
	require.NoError(t, err)

	// Only md5 should be in intersection (both matchers match it at line 5).
	assert.Len(t, detections, 1, "And should return only md5 (intersection)")
	assert.Equal(t, "hashlib.md5", detections[0].SinkCall)
}

// TestQueryType_DataflowViaLoader_FullPath tests the complete loader → dataflow path
// with polymorphic matchers, as the scan command would invoke it.
func TestQueryType_DataflowViaLoader_FullPath(t *testing.T) {
	cg := core.NewCallGraph()

	cg.CallSites["views.search"] = []core.CallSite{
		{
			Target:                   "request.args.get",
			TargetFQN:               "flask.request.args.get",
			Location:                 core.Location{File: "views.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "flask.Request",
			TypeConfidence:           0.8,
		},
		{
			Target:    "escape_html",
			TargetFQN: "markupsafe.escape",
			Location:  core.Location{File: "views.py", Line: 7},
		},
		{
			Target:                   "cursor.execute",
			TargetFQN:               "sqlite3.Cursor.execute",
			Location:                 core.Location{File: "views.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	// Build rule IR as the Python DSL would produce it via JSON.
	rule := &RuleIR{
		Matcher: map[string]any{
			"type": "dataflow",
			"sources": []any{
				map[string]any{
					"type":          "type_constrained_call",
					"receiverTypes": []any{"flask.Request"},
					"methodNames":   []any{"get", "args"},
					"minConfidence": 0.5,
					"fallbackMode":  "name",
				},
			},
			"sinks": []any{
				map[string]any{
					"type":          "type_constrained_call",
					"receiverTypes": []any{"sqlite3.Cursor"},
					"methodNames":   []any{"execute"},
					"minConfidence": 0.5,
					"fallbackMode":  "none",
				},
			},
			"sanitizers": []any{
				map[string]any{
					"type":     "call_matcher",
					"patterns": []any{"escape_html"},
					"wildcard": false,
				},
			},
			"propagation": []any{},
			"scope":       "local",
		},
	}

	loader := NewRuleLoader("")
	detections, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)

	assert.Empty(t, detections, "Sanitized flow should be filtered out, not reported")
}

// TestQueryType_ReceiverPatterns_Wildcard tests wildcard receiver patterns.
func TestQueryType_ReceiverPatterns_Wildcard(t *testing.T) {
	cg := core.NewCallGraph()

	cg.CallSites["myapp.use_cursor"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			TargetFQN:               "mysql.connector.cursor.MySQLCursor.execute",
			Location:                 core.Location{File: "db.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "mysql.connector.cursor.MySQLCursor",
			TypeConfidence:           0.9,
		},
	}

	// Use wildcard pattern to match any *Cursor type.
	ir := TypeConstrainedCallIR{
		Type:             "type_constrained_call",
		ReceiverPatterns: []string{"*Cursor"},
		MethodNames:      []string{"execute"},
		MinConfidence:    0.5,
		FallbackMode:     "none",
	}

	executor := &TypeConstrainedCallExecutor{
		IR:        &ir,
		CallGraph: cg,
	}
	detections := executor.Execute()

	assert.Len(t, detections, 1)
	assert.Equal(t, "myapp.use_cursor", detections[0].FunctionFQN)
}

// TestQueryType_ArgumentConstraints_Integration tests argument matching in context.
func TestQueryType_ArgumentConstraints_Integration(t *testing.T) {
	cg := core.NewCallGraph()

	cg.CallSites["myapp.set_perms"] = []core.CallSite{
		{
			Target:                   "os.chmod",
			TargetFQN:               "os.chmod",
			Location:                 core.Location{File: "perms.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "os",
			TypeConfidence:           0.9,
			Arguments: []core.Argument{
				{Position: 0, Value: "/tmp/file"},
				{Position: 1, Value: "0o777"},
			},
		},
	}

	// Match os.chmod with arg[1] matching wildcard "0o7*".
	ir := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"os"},
		MethodNames:   []string{"chmod"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
		PositionalArgs: map[string]ArgumentConstraint{
			"1": {Value: "0o7*", Wildcard: true},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR:        &ir,
		CallGraph: cg,
	}
	detections := executor.Execute()

	assert.Len(t, detections, 1, "Should match chmod with 0o777 via wildcard 0o7*")

	// Now test with non-matching arg.
	ir2 := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverTypes: []string{"os"},
		MethodNames:   []string{"chmod"},
		MinConfidence: 0.5,
		FallbackMode:  "none",
		PositionalArgs: map[string]ArgumentConstraint{
			"1": {Value: "0o644"},
		},
	}

	executor2 := &TypeConstrainedCallExecutor{
		IR:        &ir2,
		CallGraph: cg,
	}
	detections2 := executor2.Execute()

	assert.Empty(t, detections2, "Should NOT match chmod with 0o777 when expecting 0o644")
}

// filterByFQN filters detections by function FQN.
func filterByFQN(detections []DataflowDetection, fqn string) []DataflowDetection {
	var filtered []DataflowDetection
	for _, d := range detections {
		if d.FunctionFQN == fqn {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// TestQueryType_InheritanceViaDataflowExecutor_BUG1 proves BUG-1 fix works
// through the DataflowExecutor path (used by flows() rules).
// Before the fix, ThirdPartyRemote was nil in the executor, making MRO dead code.
func TestQueryType_InheritanceViaDataflowExecutor_BUG1(t *testing.T) {
	checker := newMockChecker()

	cg := core.NewCallGraph()
	// Source: request.args.get (generic call)
	// Sink: self.get_queryset on a ListView (subclass of View)
	cg.CallSites["myapp.views.MyListView.get"] = []core.CallSite{
		{
			Target:                   "request.args.get",
			TargetFQN:               "flask.request.args.get",
			Location:                 core.Location{File: "views.py", Line: 5},
			ResolvedViaTypeInference: false,
		},
		{
			Target:                   "self.get_queryset",
			Location:                 core.Location{File: "views.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "django.views.generic.ListView", // subclass
			TypeConfidence:           0.9,
		},
	}
	cg.ThirdPartyRemote = checker // BUG-1 fix

	// Sink rule: match any View subclass's get_queryset — uses inheritance
	sinkIR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverType:  "django.views.View", // parent class
		MethodName:    "get_queryset",
		MinConfidence: 0.5,
		FallbackMode:  "none", // strict — must match via type, not name fallback
	}

	// Execute via the loader path (same as production)
	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: irToMap(sinkIR),
	}

	results, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Should match ListView.get_queryset via MRO inheritance (BUG-1 fix)")
	if len(results) > 0 {
		assert.Equal(t, "myapp.views.MyListView.get", results[0].FunctionFQN)
	}
}

// irToMap converts any struct to map[string]any via JSON round-trip.
// This simulates how the loader receives matchers after JSON unmarshaling.
func irToMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	json.Unmarshal(b, &m) //nolint:errcheck // test helper
	return m
}

// BUG-4: MatchMethod tracking in real rule — verifies each match path is labeled
// Simulates: QueryType("SQLi", fqns=["sqlite3.Cursor"]).method("execute").
func TestQueryType_MatchMethod_RealRule_BUG4(t *testing.T) {
	cg := core.NewCallGraph()

	// Call site matched via type inference
	cg.CallSites["app.handler_typed"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}
	// Call site matched via FQN bridge
	cg.CallSites["app.handler_fqn"] = []core.CallSite{
		{
			Target:    "cursor.execute",
			TargetFQN: "sqlite3.Cursor.execute",
			Location:  core.Location{File: "app.py", Line: 20},
		},
	}
	// Call site with no type info — should NOT match with fallbackMode=none
	cg.CallSites["app.handler_unresolved"] = []core.CallSite{
		{
			Target:   "something.execute",
			Location: core.Location{File: "app.py", Line: 30},
		},
	}

	sinkIR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverType:  "sqlite3.Cursor",
		MethodName:    "execute",
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{Matcher: irToMap(sinkIR)}
	results, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)

	// Should match typed + FQN but NOT unresolved (fallbackMode=none)
	assert.Len(t, results, 2, "Should match 2 calls (type_inference + fqn_bridge), reject unresolved")

	methods := make(map[string]string) // functionFQN → matchMethod
	for _, r := range results {
		methods[r.FunctionFQN] = r.MatchMethod
	}
	assert.Equal(t, "type_inference", methods["app.handler_typed"], "Typed call should be matched via type_inference")
	assert.Equal(t, "fqn_bridge", methods["app.handler_fqn"], "FQN call should be matched via fqn_bridge")
	assert.Empty(t, methods["app.handler_unresolved"], "Unresolved call should not match with fallbackMode=none")
}

// BUG-6: Attribute executor FQN bridge — real rule test
// Simulates: QueryType("UnsafeGET", fqns=["django.http.HttpRequest"]).attr("GET").
func TestQueryType_AttributeFQNBridge_RealRule_BUG6(t *testing.T) {
	cg := core.NewCallGraph()

	// FQN-resolved attribute access (no type inference)
	cg.CallSites["myapp.views.index"] = []core.CallSite{
		{
			Target:    "request.GET",
			TargetFQN: "django.http.HttpRequest.GET",
			Location:  core.Location{File: "views.py", Line: 3},
		},
	}
	// Type-inference resolved attribute access
	cg.CallSites["myapp.views.detail"] = []core.CallSite{
		{
			Target:                   "request.GET",
			Location:                 core.Location{File: "views.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "django.http.HttpRequest",
			TypeConfidence:           0.95,
		},
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: map[string]any{
			"type":          "type_constrained_attribute",
			"receiverType":  "django.http.HttpRequest",
			"attributeName": "GET",
			"minConfidence": 0.5,
			"fallbackMode":  "none",
		},
	}

	results, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)
	assert.Len(t, results, 2, "Should match both FQN-resolved and type-inferred attribute access (BUG-6 fix)")
}

// BUG-3: Stdlib MRO integration test — real rule targeting io.IOBase.read()
// Simulates: QueryType("UnsafeRead", fqns=["io.IOBase"]).method("read")
// Should match call sites where InferredType is io.FileIO (subclass of IOBase).
func TestQueryType_StdlibMRO_IOBase_RealRule_BUG3(t *testing.T) {
	// Simulate stdlib with MRO data (future CDN state)
	stdlibChecker := &mockInheritanceChecker{
		modules: map[string]bool{"io": true},
		classes: map[string]mockClassInfo{
			"io.FileIO": {
				mro: []string{"io.FileIO", "io.RawIOBase", "io.IOBase", "builtins.object"},
			},
			"io.BufferedReader": {
				mro: []string{"io.BufferedReader", "io.BufferedIOBase", "io.IOBase", "builtins.object"},
			},
		},
	}

	cg := core.NewCallGraph()
	// Function that opens a file and reads from it
	cg.CallSites["app.process_file"] = []core.CallSite{
		{
			Target:                   "open",
			TargetFQN:               "builtins.open",
			Location:                 core.Location{File: "app.py", Line: 3},
			ResolvedViaTypeInference: false,
		},
		{
			Target:                   "f.read",
			Location:                 core.Location{File: "app.py", Line: 4},
			ResolvedViaTypeInference: true,
			InferredType:             "io.FileIO",
			TypeConfidence:           0.85,
		},
	}
	// Function using BufferedReader
	cg.CallSites["app.buffered_read"] = []core.CallSite{
		{
			Target:                   "br.read",
			Location:                 core.Location{File: "app.py", Line: 12},
			ResolvedViaTypeInference: true,
			InferredType:             "io.BufferedReader",
			TypeConfidence:           0.90,
		},
	}
	cg.StdlibRemote = stdlibChecker

	// This mirrors the Python SDK rule:
	//   QueryType("UnsafeRead", fqns=["io.IOBase"]).method("read")
	sinkIR := TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverType:  "io.IOBase",
		MethodName:    "read",
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{Matcher: irToMap(sinkIR)}
	results, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)

	// Should match BOTH FileIO.read and BufferedReader.read via MRO inheritance
	assert.Len(t, results, 2, "Should match both FileIO and BufferedReader as subclasses of IOBase (BUG-3)")

	// Verify both functions are found
	fqns := make(map[string]bool)
	for _, r := range results {
		fqns[r.FunctionFQN] = true
	}
	assert.True(t, fqns["app.process_file"], "Should match FileIO.read in process_file")
	assert.True(t, fqns["app.buffered_read"], "Should match BufferedReader.read in buffered_read")
}
