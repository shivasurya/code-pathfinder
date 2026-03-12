package dsl

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustMarshal marshals any value to json.RawMessage, panicking on error.
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(b)
}

// --- Test 1: type_constrained_call source + call_matcher sink (local VDG) ---

// TestTypeConstrainedCall_LocalVDG_DirectFlow verifies that a type_constrained_call source
// matched via type inference flows through VDG to a call_matcher sink.
func TestTypeConstrainedCall_LocalVDG_DirectFlow(t *testing.T) {
	funcFQN := "myapp.views.show"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "django.http.HttpRequest.GET.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "django.http.HttpRequest",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
		},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
		},
	}

	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "django.http.HttpRequest",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "expected 1 detection for direct VDG flow")
	assert.Equal(t, "local", detections[0].Scope)
	assert.Equal(t, funcFQN, detections[0].FunctionFQN)
	assert.Equal(t, "flat_vdg", detections[0].MatchMethod)
	assert.False(t, detections[0].Sanitized)
}

// --- Test 2: type_constrained_call source, no data flow (local VDG) ---

// TestTypeConstrainedCall_LocalVDG_NoFlow verifies that when the sink uses a different
// variable from the source, VDG correctly reports no flow.
func TestTypeConstrainedCall_LocalVDG_NoFlow(t *testing.T) {
	funcFQN := "myapp.views.safe_show"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "django.http.HttpRequest.GET.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "django.http.HttpRequest",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
		},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
		},
	}

	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestAssignStmt(6, "safe", "\"SELECT 1\"", []string{}),
		makeTestCallStmt(8, "execute", []string{"safe"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "django.http.HttpRequest",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Empty(t, detections, "VDG should find no flow: 'safe' is not connected to 'query'")
}

// --- Test 3: type_constrained_call with fallbackMode "none", no type data ---

// TestTypeConstrainedCall_NoFallback_NoTypeData verifies that when fallbackMode is "none"
// and the call site has no type inference data, resolveMatchers returns 0 source matches.
func TestTypeConstrainedCall_NoFallback_NoTypeData(t *testing.T) {
	funcFQN := "myapp.views.handler"
	cg := core.NewCallGraph()

	// Call site with NO type inference data.
	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:   "get",
			Location: core.Location{Line: 5},
			// No InferredType, ResolvedViaTypeInference=false, TypeConfidence=0.
		},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
		},
	}

	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "django.http.HttpRequest",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "none", // No fallback — requires type match.
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Empty(t, detections, "no type data + fallbackMode=none → 0 source matches → 0 detections")
}

// --- Test 4: Mixed matcher sources — both call_matcher and type_constrained_call ---

// TestMixedMatchers_LocalVDG verifies that both call_matcher and type_constrained_call
// sources can independently produce detections through VDG.
func TestMixedMatchers_LocalVDG(t *testing.T) {
	funcFQN := "myapp.views.mixed"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:   "input",
			Location: core.Location{Line: 3},
		},
		{
			Target:                   "get",
			TargetFQN:                "flask.request.args.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "flask.request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.9,
		},
		{
			Target:   "eval",
			Location: core.Location{Line: 7},
		},
		{
			Target:   "os.system",
			Location: core.Location{Line: 9},
		},
	}

	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(3, "x", "input", []string{}),
		makeTestAssignStmt(5, "y", "get", []string{}),
		makeTestCallStmt(7, "eval", []string{"x"}),
		makeTestCallStmt(9, "os.system", []string{"y"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "flask.request",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval", "os.system"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.GreaterOrEqual(t, len(detections), 2, "expected at least 2 detections (input→eval, get→os.system)")
}

// --- Test 5: Mixed sources — only one has data flow ---

// TestMixedMatchers_PartialFlow verifies that when two sources exist but only one
// has data flow to a sink, only one detection is produced.
func TestMixedMatchers_PartialFlow(t *testing.T) {
	funcFQN := "myapp.views.partial"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:   "input",
			Location: core.Location{Line: 3},
		},
		{
			Target:                   "get",
			TargetFQN:                "flask.request.args.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "flask.request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.9,
		},
		{
			Target:   "eval",
			Location: core.Location{Line: 8},
		},
	}

	// x = input() flows to eval(x), but y = request.get() does NOT flow to eval.
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(3, "x", "input", []string{}),
		makeTestAssignStmt(5, "y", "get", []string{}),
		makeTestCallStmt(8, "eval", []string{"x"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "flask.request",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Only x→eval should be detected; y has no flow to eval.
	assert.Len(t, detections, 1, "only input→eval should be detected, not request.get→eval")
}

// --- Test 6: type_constrained_call source + global inter-procedural ---

// TestTypeConstrainedCall_GlobalVDG verifies global inter-procedural detection where a
// type_constrained_call source in one function flows to a call_matcher sink in another.
func TestTypeConstrainedCall_GlobalVDG(t *testing.T) {
	funcView := "myapp.views.view"
	funcRunQuery := "myapp.db.run_query"

	cg := core.NewCallGraph()

	// view() has type_constrained_call source.
	cg.CallSites[funcView] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "django.http.HttpRequest.GET.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "django.http.HttpRequest",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
		},
	}

	// run_query() has call_matcher sink.
	cg.CallSites[funcRunQuery] = []core.CallSite{
		{
			Target:   "execute",
			Location: core.Location{Line: 10},
		},
	}

	// Call graph edge: view → run_query.
	cg.Edges = map[string][]string{
		funcView: {funcRunQuery},
	}

	// Statements for transfer summary computation.
	cg.Statements[funcView] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
	}
	cg.Statements[funcRunQuery] = []*core.Statement{
		makeTestCallStmt(10, "execute", []string{"sql"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "django.http.HttpRequest",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	globalCount := 0
	for _, d := range detections {
		if d.Scope == "global" {
			globalCount++
			assert.Equal(t, "interprocedural_vdg", d.MatchMethod)
		}
	}
	assert.GreaterOrEqual(t, globalCount, 1, "expected at least 1 global detection via summaryConfirmsFlow")
}

// --- Test 7: type_constrained_call global with sanitizer ---

// TestTypeConstrainedCall_GlobalVDG_Sanitized verifies that a sanitizer on the call path
// prevents global detection even with type_constrained_call sources.
func TestTypeConstrainedCall_GlobalVDG_Sanitized(t *testing.T) {
	funcView := "myapp.views.safe_view"
	funcSafeRun := "myapp.db.safe_run"

	cg := core.NewCallGraph()

	// safe_view() has type_constrained_call source.
	cg.CallSites[funcView] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "django.http.HttpRequest.GET.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "django.http.HttpRequest",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
		},
	}

	// safe_run() has sink but also a sanitizer.
	cg.CallSites[funcSafeRun] = []core.CallSite{
		{
			Target:   "sanitize",
			Location: core.Location{Line: 9},
		},
		{
			Target:   "execute",
			Location: core.Location{Line: 12},
		},
	}

	cg.Edges = map[string][]string{
		funcView: {funcSafeRun},
	}

	cg.Statements[funcView] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
	}
	cg.Statements[funcSafeRun] = []*core.Statement{
		makeTestAssignStmt(9, "clean", "sanitize", []string{"data"}),
		makeTestCallStmt(12, "execute", []string{"clean"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "django.http.HttpRequest",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sanitize"}}),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Count unsanitized global detections.
	unsanitizedGlobal := 0
	for _, d := range detections {
		if d.Scope == "global" && !d.Sanitized {
			unsanitizedGlobal++
		}
	}
	assert.Equal(t, 0, unsanitizedGlobal, "sanitizer on path should prevent global detection")
}

// --- Test 8: fallbackMode "name" matches without type inference ---

// TestTypeConstrainedCall_NameFallback verifies that fallbackMode "name" matches
// by method name alone when no type inference data is available.
func TestTypeConstrainedCall_NameFallback(t *testing.T) {
	funcFQN := "myapp.views.untyped"
	cg := core.NewCallGraph()

	// Call site with NO type inference data — only name match possible.
	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:   "get",
			Location: core.Location{Line: 5},
			// No InferredType, no TargetFQN.
		},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
		},
	}

	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "django.http.HttpRequest",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name", // Falls back to name matching.
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "name fallback should match 'get' by method name")
	assert.Equal(t, "local", detections[0].Scope)
	assert.Equal(t, funcFQN, detections[0].FunctionFQN)
}

// --- Test 9: Multiple receiver types ---

// TestTypeConstrainedCall_MultipleReceivers verifies that when ReceiverTypes contains
// multiple types, a call site matching any one of them is detected.
func TestTypeConstrainedCall_MultipleReceivers(t *testing.T) {
	funcFQN := "myapp.views.flask_handler"
	cg := core.NewCallGraph()

	// Call site with InferredType matching the second receiver type.
	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:                   "get",
			Location:                 core.Location{Line: 5},
			InferredType:             "flask.request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.85,
		},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
		},
	}

	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverTypes: []string{"django.http.HttpRequest", "flask.request"},
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "none",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "should match via second receiver type 'flask.request'")
	assert.Equal(t, funcFQN, detections[0].FunctionFQN)
	assert.Equal(t, "flat_vdg", detections[0].MatchMethod)
}

// --- Verify summaryConfirmsFlow works with type_constrained_call matches ---

// TestTypeConstrainedCall_SummaryConfirmsFlow_Integration verifies the summaryConfirmsFlow
// function correctly validates data propagation when sources come from type_constrained_call.
func TestTypeConstrainedCall_SummaryConfirmsFlow_Integration(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges = map[string][]string{
		"view_func": {"db_func"},
	}

	executor := NewDataflowExecutor(&DataflowIR{}, cg)

	source := CallSiteMatch{
		CallSite:    core.CallSite{Target: "get", InferredType: "django.http.HttpRequest"},
		FunctionFQN: "view_func",
		Line:        5,
	}
	sink := CallSiteMatch{
		CallSite:    core.CallSite{Target: "execute"},
		FunctionFQN: "db_func",
		Line:        10,
	}

	summaries := map[string]*taint.TaintTransferSummary{
		"view_func": {
			FunctionFQN:           "view_func",
			IsSource:              true,
			ReturnTaintedBySource: true,
			ParamToReturn:         map[int]bool{},
			ParamToSink:           map[int]bool{},
		},
		"db_func": {
			FunctionFQN:   "db_func",
			ParamToReturn: map[int]bool{},
			ParamToSink:   map[int]bool{0: true},
		},
	}

	assert.True(t, executor.summaryConfirmsFlow(source, sink, summaries),
		"flow should be confirmed: view_func IsSource → db_func ParamToSink[0]")
}
