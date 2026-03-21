package dsl

import (
	"encoding/json"
	"fmt"
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

// ============================================================================
// Part 2: Edge Case Tests — Known VDG Limitations
// ============================================================================

// TestEdgeCase_DictConstructionOverApprox verifies that dict construction flows are
// detected at variable level. VDG tracks d@2 depends on tainted@1.
func TestEdgeCase_DictConstructionOverApprox(t *testing.T) {
	funcFQN := "test.edge.dict_construct"
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "tainted", "input", []string{}),
			makeTestAssignStmt(2, "d", "", []string{"tainted"}),
			makeTestCallStmt(3, "sink", []string{"d"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "sink", Location: core.Location{Line: 3}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// VDG sees d depends on tainted, sink uses d → detection.
	// Field-level sensitivity is future work.
	assert.Len(t, detections, 1, "VDG over-approximates dict construction (acceptable)")
}

// TestEdgeCase_TupleUnpacking verifies VDG behavior when tuple unpacking is modeled
// as a single assignment. VDG sees a single def, not individual elements.
func TestEdgeCase_TupleUnpacking(t *testing.T) {
	funcFQN := "test.edge.tuple_unpack"
	// a, b = func_returning_tuple() — modeled as a single assignment to "a".
	// VDG can't distinguish individual tuple elements.
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "a", "get_tuple", []string{}),
			makeTestCallStmt(2, "sink", []string{"a"}),
		},
		[]core.CallSite{
			{Target: "get_tuple", Location: core.Location{Line: 1}},
			{Target: "sink", Location: core.Location{Line: 2}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get_tuple"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// If tree-sitter models tuple unpacking as a single def, VDG detects flow.
	assert.Len(t, detections, 1, "tuple unpacking modeled as single def → over-approximation")
}

// TestEdgeCase_AttributeAccess verifies VDG behavior with dotted attribute access.
// obj.attr = tainted; sink(obj.attr) — depends on whether VDG tracks dotted names.
func TestEdgeCase_AttributeAccess(t *testing.T) {
	funcFQN := "test.edge.attr_access"
	// obj.attr = tainted modeled as separate statements.
	// VDG doesn't link "obj.attr" def to "obj.attr" use since it's not a simple variable.
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "tainted", "input", []string{}),
			makeTestAssignStmt(2, "obj.attr", "", []string{"tainted"}),
			makeTestCallStmt(3, "sink", []string{"obj.attr"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "sink", Location: core.Location{Line: 3}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// VDG may or may not track dotted attribute names as variables.
	if len(detections) > 0 {
		t.Log("VDG tracks obj.attr as variable — over-approximation (acceptable)")
	} else {
		t.Log("VDG does NOT track obj.attr — attribute access not modeled")
	}
}

// TestEdgeCase_StarArgs verifies VDG behavior with *args parameter indexing.
// args[0] is not linked to caller arguments in inter-procedural analysis.
func TestEdgeCase_StarArgs(t *testing.T) {
	funcCaller := "test.edge.star_caller"
	funcCallee := "test.edge.star_callee"

	cg := core.NewCallGraph()
	cg.CallSites[funcCaller] = []core.CallSite{
		{Target: "input", Location: core.Location{Line: 1}},
	}
	cg.CallSites[funcCallee] = []core.CallSite{
		{Target: "sink", Location: core.Location{Line: 5}},
	}
	cg.Edges = map[string][]string{funcCaller: {funcCallee}}
	cg.Statements[funcCaller] = []*core.Statement{
		makeTestAssignStmt(1, "x", "input", []string{}),
	}
	cg.Statements[funcCallee] = []*core.Statement{
		// def f(*args): sink(args[0]) — args[0] not linked to param.
		makeTestCallStmt(5, "sink", []string{"args"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// *args indexing is a known limitation.
	// Global analysis may still detect via optimistic summary (no ParamToSink blocks it).
	globalUnsanitized := 0
	for _, d := range detections {
		if d.Scope == "global" && !d.Sanitized {
			globalUnsanitized++
		}
	}
	t.Logf("*args inter-procedural detections=%d (may detect via optimistic summary)", globalUnsanitized)
}

// TestEdgeCase_TernaryExpression verifies that VDG treats both ternary branches
// as reaching the same variable (over-approximation, no path sensitivity).
func TestEdgeCase_TernaryExpression(t *testing.T) {
	funcFQN := "test.edge.ternary"
	// x = tainted if cond else safe; sink(x)
	// VDG: x@2 depends on tainted@1 (both branches collapsed).
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "tainted", "input", []string{}),
			makeTestAssignStmt(2, "x", "", []string{"tainted"}),
			makeTestCallStmt(3, "sink", []string{"x"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "sink", Location: core.Location{Line: 3}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Over-approximation — VDG cannot distinguish ternary branches.
	assert.Len(t, detections, 1, "ternary over-approximation — both branches reach x")
}

// TestEdgeCase_DelStatement verifies that reassignment kills taint via LatestDef
// even when del statement is not explicitly modeled.
func TestEdgeCase_DelStatement(t *testing.T) {
	funcFQN := "test.edge.del_stmt"
	// x = input(); del x; x = "safe"; sink(x)
	// VDG: x@3="safe" kills x@1=input via LatestDef. sink uses x@3.
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "x", "input", []string{}),
			// del x not modeled as a statement in VDG.
			makeTestAssignStmt(3, "x", "\"safe\"", []string{}),
			makeTestCallStmt(4, "sink", []string{"x"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "sink", Location: core.Location{Line: 4}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// del is not modeled, but reassignment x="safe" kills x@1 via LatestDef.
	assert.Empty(t, detections, "reassignment kills taint even without del modeling")
}

// TestEdgeCase_ClassHierarchy verifies that local analysis works without class hierarchy
// resolution. CHA affects inter-procedural resolution of inherited methods.
func TestEdgeCase_ClassHierarchy(t *testing.T) {
	funcFQN := "test.edge.class_hier"
	// Direct call within same function — no CHA needed for local analysis.
	// The limitation affects inter-procedural resolution.
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "data", "input", []string{}),
			makeTestCallStmt(2, "sink", []string{"data"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "sink", Location: core.Location{Line: 2}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// For local scope, CHA is not needed — direct calls work.
	assert.Len(t, detections, 1, "local analysis works without CHA")
	t.Log("CHA limitation only affects inter-procedural resolution of inherited methods")
}

// TestEdgeCase_Lambda verifies that lambda bodies are not analyzed as separate
// functions, so f = lambda x: sink(x); f(tainted) has no flow.
func TestEdgeCase_Lambda(t *testing.T) {
	funcCaller := "test.edge.lambda_caller"
	funcLambda := "test.edge.<lambda>"

	cg := core.NewCallGraph()
	// Lambda not in call graph — it's an anonymous function.
	cg.CallSites[funcCaller] = []core.CallSite{
		{Target: "input", Location: core.Location{Line: 1}},
	}
	// No call sites or statements for lambda body.
	cg.Statements[funcCaller] = []*core.Statement{
		makeTestAssignStmt(1, "tainted", "input", []string{}),
		makeTestAssignStmt(2, "f", "", []string{}), // f = lambda ...
	}
	// No edge to lambda since it's not resolved.
	_ = funcLambda

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Lambda bodies not analyzed — no sink found anywhere.
	assert.Empty(t, detections, "lambda bodies not analyzed — no detection")
}

// ============================================================================
// Part 3: V4 Regression Tests — ANY FAILURE IS A REGRESSION
// ============================================================================

// TestRegression_V4_OneFile_DirectFlow: x = input(); eval(x) → 1 detection.
func TestRegression_V4_OneFile_DirectFlow(t *testing.T) {
	funcFQN := "test.regression.direct"
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "x", "input", []string{}),
			makeTestCallStmt(2, "eval", []string{"x"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "eval", Location: core.Location{Line: 2}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "R1: 1-file direct flow MUST detect")
	assert.Equal(t, "flat_vdg", detections[0].MatchMethod)
	assert.GreaterOrEqual(t, detections[0].Confidence, 0.85)
}

// TestRegression_V4_OneFile_Sanitized: input() → sanitize() → eval() → 0 detections.
func TestRegression_V4_OneFile_Sanitized(t *testing.T) {
	funcFQN := "test.regression.sanitized"
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "x", "input", []string{}),
			makeTestAssignStmt(2, "x", "sanitize", []string{"x"}),
			makeTestCallStmt(3, "eval", []string{"x"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "sanitize", Location: core.Location{Line: 2}},
			{Target: "eval", Location: core.Location{Line: 3}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sanitize"}}),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	unsanitized := 0
	for _, d := range detections {
		if !d.Sanitized {
			unsanitized++
		}
	}
	assert.Equal(t, 0, unsanitized, "R2: sanitized flow MUST NOT produce unsanitized detections")
}

// TestRegression_V4_OneFile_NoFlow: x = input(); y = "safe"; eval(y) → 0 detections.
func TestRegression_V4_OneFile_NoFlow(t *testing.T) {
	funcFQN := "test.regression.noflow"
	cg := setupTestCallGraph(funcFQN,
		[]*core.Statement{
			makeTestAssignStmt(1, "x", "input", []string{}),
			makeTestAssignStmt(2, "y", "\"safe\"", []string{}),
			makeTestCallStmt(3, "eval", []string{"y"}),
		},
		[]core.CallSite{
			{Target: "input", Location: core.Location{Line: 1}},
			{Target: "eval", Location: core.Location{Line: 3}},
		},
	)

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Empty(t, detections, "R3: no-flow MUST NOT detect (y is unrelated to x)")
}

// TestRegression_V4_TwoFile_CrossModule: source in func A, sink in func B, A→B edge.
func TestRegression_V4_TwoFile_CrossModule(t *testing.T) {
	funcA := "module_a.get_input"
	funcB := "module_b.run_query"

	cg := core.NewCallGraph()
	cg.CallSites[funcA] = []core.CallSite{
		{Target: "input", Location: core.Location{Line: 2}},
	}
	cg.CallSites[funcB] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 5}},
	}
	cg.Edges = map[string][]string{funcA: {funcB}}
	cg.Statements[funcA] = []*core.Statement{
		makeTestAssignStmt(2, "x", "input", []string{}),
	}
	cg.Statements[funcB] = []*core.Statement{
		makeTestCallStmt(5, "eval", []string{"data"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	globalCount := 0
	for _, d := range detections {
		if d.Scope == "global" {
			globalCount++
		}
	}
	assert.GreaterOrEqual(t, globalCount, 1, "R4: 2-file cross-module MUST detect globally")
}

// TestRegression_V4_TwoFile_Sanitized: source→sanitizer→sink across 2 files → 0 detections.
func TestRegression_V4_TwoFile_Sanitized(t *testing.T) {
	funcA := "module_a.get_input_safe"
	funcB := "module_b.safe_run"

	cg := core.NewCallGraph()
	cg.CallSites[funcA] = []core.CallSite{
		{Target: "input", Location: core.Location{Line: 2}},
	}
	cg.CallSites[funcB] = []core.CallSite{
		{Target: "sanitize", Location: core.Location{Line: 5}},
		{Target: "eval", Location: core.Location{Line: 7}},
	}
	cg.Edges = map[string][]string{funcA: {funcB}}
	cg.Statements[funcA] = []*core.Statement{
		makeTestAssignStmt(2, "x", "input", []string{}),
	}
	cg.Statements[funcB] = []*core.Statement{
		makeTestAssignStmt(5, "clean", "sanitize", []string{"data"}),
		makeTestCallStmt(7, "eval", []string{"clean"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sanitize"}}),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	unsanitizedGlobal := 0
	for _, d := range detections {
		if d.Scope == "global" && !d.Sanitized {
			unsanitizedGlobal++
		}
	}
	assert.Equal(t, 0, unsanitizedGlobal, "R5: 2-file sanitized MUST NOT detect globally")
}

// TestRegression_V4_ThreeFile_MultiHop: A→B→C with taint flowing through all 3.
func TestRegression_V4_ThreeFile_MultiHop(t *testing.T) {
	funcA := "module_a.source_func"
	funcB := "module_b.relay_func"
	funcC := "module_c.sink_func"

	cg := core.NewCallGraph()
	cg.CallSites[funcA] = []core.CallSite{
		{Target: "input", Location: core.Location{Line: 2}},
	}
	cg.CallSites[funcB] = []core.CallSite{}
	cg.CallSites[funcC] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 8}},
	}
	cg.Edges = map[string][]string{
		funcA: {funcB},
		funcB: {funcC},
	}
	cg.Statements[funcA] = []*core.Statement{
		makeTestAssignStmt(2, "x", "input", []string{}),
	}
	cg.Statements[funcB] = []*core.Statement{
		// relay: receives data, passes it along.
		makeTestAssignStmt(5, "y", "", []string{"data"}),
	}
	cg.Statements[funcC] = []*core.Statement{
		makeTestCallStmt(8, "eval", []string{"data"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	globalCount := 0
	for _, d := range detections {
		if d.Scope == "global" {
			globalCount++
		}
	}
	assert.GreaterOrEqual(t, globalCount, 1, "R6: 3-file multi-hop MUST detect globally")
}

// buildTenFunctionChain creates a 10-function call chain for regression testing.
// func_01 (source) → func_02 → ... → func_10 (sink).
// If sanitizerAt > 0, adds a sanitizer call site at that function index.
func buildTenFunctionChain(sanitizerAt int) *core.CallGraph {
	cg := core.NewCallGraph()
	funcs := make([]string, 10)
	for i := 0; i < 10; i++ {
		funcs[i] = fmt.Sprintf("chain.func_%02d", i+1)
	}

	// Source at func_01.
	cg.CallSites[funcs[0]] = []core.CallSite{
		{Target: "input", Location: core.Location{Line: 2}},
	}
	cg.Statements[funcs[0]] = []*core.Statement{
		makeTestAssignStmt(2, "x", "input", []string{}),
	}

	// Intermediaries func_02 through func_09.
	for i := 1; i < 9; i++ {
		callSites := []core.CallSite{}
		stmts := []*core.Statement{
			makeTestAssignStmt(uint32(2), "y", "", []string{"data"}),
		}
		if sanitizerAt == i+1 {
			callSites = append(callSites, core.CallSite{
				Target: "sanitize_sql", Location: core.Location{Line: 3},
			})
			stmts = append(stmts, &core.Statement{
				Type:       core.StatementTypeAssignment,
				LineNumber: 3,
				Def:        "y",
				CallTarget: "sanitize_sql",
				Uses:       []string{"y"},
			})
		}
		cg.CallSites[funcs[i]] = callSites
		cg.Statements[funcs[i]] = stmts
	}

	// Sink at func_10.
	cg.CallSites[funcs[9]] = []core.CallSite{
		{Target: "cursor.execute", Location: core.Location{Line: 5}},
	}
	cg.Statements[funcs[9]] = []*core.Statement{
		makeTestCallStmt(5, "cursor.execute", []string{"data"}),
	}

	// Build chain edges: func_01 → func_02 → ... → func_10.
	cg.Edges = make(map[string][]string)
	for i := 0; i < 9; i++ {
		cg.Edges[funcs[i]] = []string{funcs[i+1]}
	}

	return cg
}

// TestRegression_V4_TenFile_Chain: 10-function chain, source at 1, sink at 10.
func TestRegression_V4_TenFile_Chain(t *testing.T) {
	cg := buildTenFunctionChain(0) // No sanitizer.

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"cursor.execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	globalCount := 0
	for _, d := range detections {
		if d.Scope == "global" {
			globalCount++
		}
	}
	assert.GreaterOrEqual(t, globalCount, 1, "R7: 10-file chain MUST detect globally")
}

// TestRegression_V4_TenFile_Sanitized: 10-function chain with sanitizer at func_02.
func TestRegression_V4_TenFile_Sanitized(t *testing.T) {
	cg := buildTenFunctionChain(2) // Sanitizer at func_02.

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"input"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"cursor.execute"}}),
		Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sanitize_sql"}}),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	unsanitizedGlobal := 0
	for _, d := range detections {
		if d.Scope == "global" && !d.Sanitized {
			unsanitizedGlobal++
		}
	}
	assert.Equal(t, 0, unsanitizedGlobal, "R8: 10-file chain with sanitizer at func_02 MUST NOT detect")
}

// ============================================================================
// Part 4: QueryType 10-File Chain Tests
// ============================================================================

// TestQueryType_TenFile_TypeConstrainedSource: 10-file chain with type_constrained_call source.
func TestQueryType_TenFile_TypeConstrainedSource(t *testing.T) {
	cg := buildTenFunctionChain(0)

	// Override func_01 to have a type_constrained_call source instead of plain input().
	cg.CallSites["chain.func_01"] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "rest_framework.request.Request.query_params.get",
			Location:                 core.Location{Line: 2},
			InferredType:             "rest_framework.request.Request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.85,
		},
	}
	cg.Statements["chain.func_01"] = []*core.Statement{
		makeTestAssignStmt(2, "x", "get", []string{}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "rest_framework.request.Request",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"cursor.execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	globalCount := 0
	for _, d := range detections {
		if d.Scope == "global" {
			globalCount++
		}
	}
	assert.GreaterOrEqual(t, globalCount, 1, "Q1: 10-file chain with type_constrained_call source MUST detect")
}

// TestQueryType_TenFile_TypeConstrainedSource_Sanitized: same chain with sanitizer at func_02.
func TestQueryType_TenFile_TypeConstrainedSource_Sanitized(t *testing.T) {
	cg := buildTenFunctionChain(2) // Sanitizer at func_02.

	// Override func_01 for type_constrained_call source.
	cg.CallSites["chain.func_01"] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "rest_framework.request.Request.query_params.get",
			Location:                 core.Location{Line: 2},
			InferredType:             "rest_framework.request.Request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.85,
		},
	}
	cg.Statements["chain.func_01"] = []*core.Statement{
		makeTestAssignStmt(2, "x", "get", []string{}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{
			mustMarshal(TypeConstrainedCallIR{
				Type:          "type_constrained_call",
				ReceiverType:  "rest_framework.request.Request",
				MethodName:    "get",
				MinConfidence: 0.5,
				FallbackMode:  "name",
			}),
		},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"cursor.execute"}}),
		Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sanitize_sql"}}),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	unsanitizedGlobal := 0
	for _, d := range detections {
		if d.Scope == "global" && !d.Sanitized {
			unsanitizedGlobal++
		}
	}
	assert.Equal(t, 0, unsanitizedGlobal, "Q2: 10-file chain with sanitizer MUST NOT detect")
}

// --- Tracked Parameter Tests ---

// intPtr is a helper to create *int for TrackedParam.Index.
func intPtr(i int) *int { return &i }

func TestTrackedParam_LocalVDG_PositionalMatch(t *testing.T) {
	// Taint flows to param 0 of execute, tracks(0) → should detect
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "query", IsVariable: true, Position: 0},
				{Value: "params", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query", "params"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Index: intPtr(0)}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "taint reaches tracked param 0")
	assert.NotNil(t, detections[0].SinkParamIndex)
	assert.Equal(t, 0, *detections[0].SinkParamIndex)
}

func TestTrackedParam_LocalVDG_PositionalReject(t *testing.T) {
	// Taint flows to param 1 of execute, tracks(0) → should NOT detect
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "safe_sql", IsVariable: true, Position: 0},
				{Value: "tainted", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "tainted", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"safe_sql", "tainted"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Index: intPtr(0)}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 0, "taint reaches param 1 but only param 0 is tracked")
}

func TestTrackedParam_LocalVDG_NoTracksAllMatch(t *testing.T) {
	// Taint flows to param 1, no tracks() → should detect (backward compat)
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "execute",
			Location: core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "safe_sql", IsVariable: true, Position: 0},
				{Value: "tainted", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "tainted", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"safe_sql", "tainted"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 1, "no tracks() = all params sensitive (backward compat)")
}

func TestTrackedParam_MultipleTracked(t *testing.T) {
	// tracks(0, 1) — taint reaches param 1 → should detect
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "execvp",
			Location: core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "safe", IsVariable: true, Position: 0},
				{Value: "tainted", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "tainted", "get", []string{}),
		makeTestCallStmt(8, "execvp", []string{"safe", "tainted"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execvp"}, TrackedParams: []TrackedParam{{Index: intPtr(0)}, {Index: intPtr(1)}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 1, "taint reaches tracked param 1")
}

func TestTrackedParam_VarRenamedBeforeSink(t *testing.T) {
	// q = input(); sql = q; execute(sql) — tracks(0) should detect via SinkVar
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "execute",
			Location: core.Location{Line: 10},
			Arguments: []core.Argument{
				{Value: "sql", IsVariable: true, Position: 0},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "q", "get", []string{}),
		{Type: core.StatementTypeAssignment, LineNumber: 7, Def: "sql", Uses: []string{"q"}},
		makeTestCallStmt(10, "execute", []string{"sql"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Index: intPtr(0)}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 1, "SinkVar=sql matches arg at position 0")
}

func TestTrackedParam_Sanitized(t *testing.T) {
	// Taint reaches tracked param 0 but is sanitized → should NOT detect
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{Target: "escape", Location: core.Location{Line: 7}},
		{
			Target:   "execute",
			Location: core.Location{Line: 10},
			Arguments: []core.Argument{
				{Value: "safe_query", IsVariable: true, Position: 0},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "raw", "get", []string{}),
		makeTestAssignStmt(7, "safe_query", "escape", []string{"raw"}),
		makeTestCallStmt(10, "execute", []string{"safe_query"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Index: intPtr(0)}}})},
		Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"escape"}}),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 0, "sanitized before reaching tracked param")
}

func TestTrackedParam_NonexistentName(t *testing.T) {
	// tracks("nonexistent") — name doesn't resolve → should NOT detect
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:    "execute",
			TargetFQN: "db.execute",
			Location:  core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "query", IsVariable: true, Position: 0},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Name: "nonexistent"}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 0, "nonexistent param name resolves to no indices")
}

func TestTrackedParam_NilCallSite(t *testing.T) {
	// CallSite exists for pattern matching but has no Arguments.
	// matchesTrackedParams finds the CallSite but no args match → conservative accept
	// because findCallSiteAtLine returns a CallSite with empty Arguments.
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		// CallSite for execute exists but at line 9 (different from VDG sink line 8)
		// so findCallSiteAtLine returns nil → conservative acceptance
		{Target: "execute", Location: core.Location{Line: 9}},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"query"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Without TrackedParams, detection should pass through (backward compat)
	assert.Len(t, detections, 1, "no TrackedParams: accept detection even with mismatched CallSite line")
}

func TestTrackedParam_TypeConstrainedCall(t *testing.T) {
	// TrackedParams on TypeConstrainedCallIR — positional match
	funcFQN := "app.views.sql_handler"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "flask.Request.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "flask.Request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
		},
		{
			Target:                   "execute",
			TargetFQN:                "sqlite3.Cursor.execute",
			Location:                 core.Location{Line: 10},
			InferredType:             "sqlite3.Cursor",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
			Arguments: []core.Argument{
				{Value: "query", IsVariable: true, Position: 0},
				{Value: "safe_params", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "query", "get", []string{}),
		makeTestCallStmt(10, "execute", []string{"query", "safe_params"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{mustMarshal(TypeConstrainedCallIR{
			Type: "type_constrained_call", ReceiverTypes: []string{"flask.Request"},
			MethodNames: []string{"get"}, MinConfidence: 0.5, FallbackMode: "name",
		})},
		Sinks: []json.RawMessage{mustMarshal(TypeConstrainedCallIR{
			Type: "type_constrained_call", ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames: []string{"execute"}, MinConfidence: 0.5, FallbackMode: "name",
			TrackedParams: []TrackedParam{{Index: intPtr(0)}},
		})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "type-constrained sink with tracks(0)")
	assert.NotNil(t, detections[0].SinkParamIndex)
	assert.Equal(t, 0, *detections[0].SinkParamIndex)
}

func TestTrackedParam_TypeConstrainedCall_Reject(t *testing.T) {
	// TypeConstrainedCallIR — taint at param 1, tracks(0) → no detection
	funcFQN := "app.views.sql_handler"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:                   "get",
			TargetFQN:                "flask.Request.get",
			Location:                 core.Location{Line: 5},
			InferredType:             "flask.Request",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
		},
		{
			Target:                   "execute",
			TargetFQN:                "sqlite3.Cursor.execute",
			Location:                 core.Location{Line: 10},
			InferredType:             "sqlite3.Cursor",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.8,
			Arguments: []core.Argument{
				{Value: "safe_sql", IsVariable: true, Position: 0},
				{Value: "tainted", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "tainted", "get", []string{}),
		makeTestCallStmt(10, "execute", []string{"safe_sql", "tainted"}),
	}

	ir := &DataflowIR{
		Sources: []json.RawMessage{mustMarshal(TypeConstrainedCallIR{
			Type: "type_constrained_call", ReceiverTypes: []string{"flask.Request"},
			MethodNames: []string{"get"}, MinConfidence: 0.5, FallbackMode: "name",
		})},
		Sinks: []json.RawMessage{mustMarshal(TypeConstrainedCallIR{
			Type: "type_constrained_call", ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames: []string{"execute"}, MinConfidence: 0.5, FallbackMode: "name",
			TrackedParams: []TrackedParam{{Index: intPtr(0)}},
		})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 0, "taint at param 1, tracked param 0 only")
}

func TestTrackedParam_LocalVDG_ByName(t *testing.T) {
	// tracks("query") — name resolves to param index 0 → should detect
	funcFQN := "app.views.handle"
	calleeFQN := "db.module.execute"
	cg := core.NewCallGraph()

	cg.Parameters[calleeFQN+".query"] = &core.ParameterSymbol{
		Name: "query", ParentFQN: calleeFQN, Line: 1,
	}
	cg.Parameters[calleeFQN+".params"] = &core.ParameterSymbol{
		Name: "params", ParentFQN: calleeFQN, Line: 1,
	}

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:    "execute",
			TargetFQN: calleeFQN,
			Location:  core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "q", IsVariable: true, Position: 0},
				{Value: "safe", IsVariable: true, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "q", "get", []string{}),
		makeTestCallStmt(8, "execute", []string{"q", "safe"}),
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Name: "query"}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	require.Len(t, detections, 1, "name 'query' resolves to index 0, taint at index 0")
}

func TestTrackedParam_ReturnSource(t *testing.T) {
	// Source with tracks("return") — no-op in v1, flow still detected
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "eval",
			Location: core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "data", IsVariable: true, Position: 0},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "data", "get", []string{}),
		makeTestCallStmt(8, "eval", []string{"data"}),
	}

	ir := &DataflowIR{
		Sources:    []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}, TrackedParams: []TrackedParam{{Return: true}}})},
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 1, "tracks('return') on source is no-op in v1")
}

func TestTrackedParam_CombinedPositionalArgAndTracks(t *testing.T) {
	// PositionalArgs constraint (arg 1 = "True") + tracks(0)
	// Both the arg constraint and tracked param filtering must work together.
	funcFQN := "app.views.handle"
	cg := core.NewCallGraph()

	cg.CallSites[funcFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{
			Target:   "call",
			Location: core.Location{Line: 8},
			Arguments: []core.Argument{
				{Value: "cmd", IsVariable: true, Position: 0},
				{Value: "True", IsVariable: false, Position: 1},
			},
		},
	}
	cg.Statements[funcFQN] = []*core.Statement{
		makeTestAssignStmt(5, "cmd", "get", []string{}),
		makeTestCallStmt(8, "call", []string{"cmd"}),
	}

	ir := &DataflowIR{
		Sources: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks: []json.RawMessage{mustMarshal(CallMatcherIR{
			Type: "call_matcher", Patterns: []string{"call"},
			PositionalArgs: map[string]ArgumentConstraint{"1": {Value: "True"}},
			TrackedParams:  []TrackedParam{{Index: intPtr(0)}},
		})},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.Len(t, detections, 1, "positional arg constraint + tracks(0): taint at tracked param 0")
}

func TestTrackedParam_Global_SummaryFiltered(t *testing.T) {
	// Inter-procedural: taint to param 0, tracks(0) → should detect
	callerFQN := "app.views.handler"
	calleeFQN := "app.db.run_query"
	cg := core.NewCallGraph()

	cg.CallSites[callerFQN] = []core.CallSite{
		{Target: "get", TargetFQN: "request.get", Location: core.Location{Line: 5}},
		{Target: "run_query", TargetFQN: calleeFQN, Location: core.Location{Line: 8}},
	}
	cg.Statements[callerFQN] = []*core.Statement{
		makeTestAssignStmt(5, "data", "get", []string{}),
		makeTestCallStmt(8, "run_query", []string{"data"}),
	}
	cg.CallSites[calleeFQN] = []core.CallSite{
		{
			Target:   "execute",
			Location: core.Location{Line: 20},
			Arguments: []core.Argument{
				{Value: "q", IsVariable: true, Position: 0},
			},
		},
	}
	cg.Statements[calleeFQN] = []*core.Statement{
		makeTestCallStmt(20, "execute", []string{"q"}),
	}
	cg.Edges[callerFQN] = []string{calleeFQN}
	cg.ReverseEdges[calleeFQN] = []string{callerFQN}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"get"}}),
		Sinks:      []json.RawMessage{mustMarshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}, TrackedParams: []TrackedParam{{Index: intPtr(0)}}})},
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.GreaterOrEqual(t, len(detections), 1, "inter-proc: taint reaches tracked param 0")
}

func TestTrackedParam_SummaryConfirmsFlow_Filtered(t *testing.T) {
	// Unit test for summaryConfirmsFlow with TrackedParams.
	// Summary has ParamToSink: {0: false, 1: true} — only param 1 reaches sink.
	// tracks(0) should reject because param 0 doesn't flow to sink.
	cg := core.NewCallGraph()
	cg.Edges["source_func"] = []string{"sink_func"}

	executor := &DataflowExecutor{
		IR:          &DataflowIR{},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	source := CallSiteMatch{FunctionFQN: "source_func", Line: 5}
	sink := CallSiteMatch{
		FunctionFQN:   "sink_func",
		Line:          10,
		TrackedParams: []TrackedParam{{Index: intPtr(0)}},
	}

	summaries := map[string]*taint.TaintTransferSummary{
		"source_func": {
			FunctionFQN:            "source_func",
			IsSource:               true,
			ReturnTaintedBySource:  true,
			ParamNames:             []string{"input"},
			ParamToReturn:          map[int]bool{0: true},
			ParamToSink:            map[int]bool{},
		},
		"sink_func": {
			FunctionFQN: "sink_func",
			ParamNames:  []string{"safe_param", "tainted_param"},
			ParamToSink: map[int]bool{0: false, 1: true},
		},
	}

	// tracks(0) but only param 1 flows to sink → should reject
	result := executor.summaryConfirmsFlow(source, sink, summaries)
	assert.False(t, result, "tracks(0) rejects when only param 1 flows to sink")

	// tracks(1) should accept since param 1 flows to sink
	sink.TrackedParams = []TrackedParam{{Index: intPtr(1)}}
	result = executor.summaryConfirmsFlow(source, sink, summaries)
	assert.True(t, result, "tracks(1) accepts when param 1 flows to sink")

	// No TrackedParams (backward compat) → should accept any param flowing
	sink.TrackedParams = nil
	result = executor.summaryConfirmsFlow(source, sink, summaries)
	assert.True(t, result, "no TrackedParams: any param to sink counts")
}
