package dsl

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Boundary Value Tests ---

func TestEdge_LongFQN_50Components(t *testing.T) {
	// FQN with 50 dot-separated components should match via FQN bridge.
	parts := make([]string, 50)
	for i := range parts {
		parts[i] = fmt.Sprintf("pkg%d", i)
	}
	longFQN := strings.Join(parts, ".")
	methodName := "method"
	target := parts[len(parts)-1] + "." + methodName

	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:    target,
			TargetFQN: longFQN + "." + methodName,
			Location:  core.Location{Line: 1, Column: 5},
		},
	}

	// ReceiverType = the parent of the last component.
	receiverType := strings.Join(parts[:len(parts)-1], ".")
	ir := &TypeConstrainedCallIR{
		ReceiverType: receiverType,
		MethodName:   methodName,
		FallbackMode: "none",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	assert.NotEmpty(t, results, "Should match long FQN via fqn_prefix")
}

func TestEdge_ReceiverTypes_100Entries(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "method",
			InferredType:             "type_50.Class",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.9,
			Location:                 core.Location{Line: 1},
		},
	}

	receiverTypes := make([]string, 100)
	for i := range receiverTypes {
		receiverTypes[i] = fmt.Sprintf("type_%d.Class", i)
	}

	ir := &TypeConstrainedCallIR{
		ReceiverTypes: receiverTypes,
		MethodName:    "method",
		FallbackMode:  "none",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	require.Len(t, results, 1)
	assert.Equal(t, "type_inference", results[0].MatchMethod)
}

func TestEdge_LargeArgumentString_NoHang(t *testing.T) {
	longArg := strings.Repeat("SELECT * FROM users WHERE id = ", 333) // ~10k chars
	args := []core.Argument{
		{Value: longArg, Position: 0},
	}
	constraints := map[string]ArgumentConstraint{
		"0": {Value: "SELECT.*FROM", Comparator: "regex"},
	}

	result := MatchesPositionalArguments(args, constraints)
	assert.True(t, result)
}

// --- Semantic Correctness Tests ---

func TestEdge_TypeInference_PriorityOverFQNBridge(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "method",
			TargetFQN:                "module.Class.method",
			InferredType:             "module.Class",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.9,
			Location:                 core.Location{Line: 1},
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	require.Len(t, results, 1)
	assert.Equal(t, "type_inference", results[0].MatchMethod,
		"Type inference should take priority over FQN bridge when both match")
}

func TestEdge_FQNBridge_PriorityOverPrefix(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:    "method",
			TargetFQN: "module.Class.method",
			Location:  core.Location{Line: 1},
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	require.Len(t, results, 1)
	assert.Equal(t, "fqn_bridge", results[0].MatchMethod,
		"FQN bridge should take priority over FQN prefix when both match")
}

func TestEdge_FallbackNone_RejectsNoTypeInfo(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:   "method",
			Location: core.Location{Line: 1},
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method",
		FallbackMode: "none",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	assert.Empty(t, results, "fallbackMode=none should reject when no type/FQN info")
}

func TestEdge_ConfidenceExactlyAtThreshold(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "method",
			InferredType:             "module.Class",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.5,
			Location:                 core.Location{Line: 1},
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType:  "module.Class",
		MethodName:    "method",
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	assert.Len(t, results, 1, "Confidence exactly at threshold should match (inclusive)")
}

func TestEdge_ConfidenceBelowThreshold(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "method",
			InferredType:             "module.Class",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.499,
			Location:                 core.Location{Line: 1},
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType:  "module.Class",
		MethodName:    "method",
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	assert.Empty(t, results, "Confidence below threshold should not match")
}

// --- Deeply Nested Logic Tests ---

func TestEdge_DeeplyNestedLogicOr_NoStackOverflow(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Build 10-level nested logic_or, each wrapping a call_matcher.
	inner := map[string]any{
		"type":     "call_matcher",
		"patterns": []any{"eval"},
	}
	for i := 0; i < 10; i++ {
		inner = map[string]any{
			"type":     "logic_or",
			"matchers": []any{inner},
		}
	}

	ruleIR := &RuleIR{Matcher: inner}
	results, err := loader.ExecuteRule(ruleIR, cg)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Deeply nested logic_or should resolve correctly")
	assert.Equal(t, "eval", results[0].SinkCall)
}

func TestEdge_DeeplyNestedLogicAnd_NoStackOverflow(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Build 10-level nested logic_and, each wrapping a call_matcher.
	inner := map[string]any{
		"type":     "call_matcher",
		"patterns": []any{"eval"},
	}
	for i := 0; i < 10; i++ {
		inner = map[string]any{
			"type":     "logic_and",
			"matchers": []any{inner},
		}
	}

	ruleIR := &RuleIR{Matcher: inner}
	results, err := loader.ExecuteRule(ruleIR, cg)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Deeply nested logic_and should resolve correctly")
}

// --- Malformed IR Tests ---

func TestEdge_InvalidFallbackMode_DefaultsName(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "method", Location: core.Location{Line: 1}},
	}

	dc := NewDiagnosticCollector()
	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method",
		FallbackMode: "invalid_mode",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:          ir,
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	// "invalid_mode" is not "name" → fallback switch default → empty string → no match.
	assert.Empty(t, results, "Invalid fallback mode should not match")
}

func TestEdge_MethodNameWithSpecialChars(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "exec*ute", Location: core.Location{Line: 1}},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "exec*ute",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	assert.Len(t, results, 1, "Special chars in method name should be treated literally")
}

// --- Diagnostic Verification Tests ---

func TestEdge_DiagnosticsVerify_NilCallGraph_Warning(t *testing.T) {
	dc := NewDiagnosticCollector()
	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:          ir,
		CallGraph:   nil,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Empty(t, results)
	assert.True(t, dc.HasErrors(), "Nil CallGraph should emit error diagnostic")
}

func TestEdge_DiagnosticsVerify_EmptyTarget_Skipped(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 2}},
	}

	dc := NewDiagnosticCollector()
	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "eval",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:          ir,
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Len(t, results, 1, "Empty target should be skipped, eval should match")
	debugs := dc.FilterByLevel("debug")
	found := false
	for _, d := range debugs {
		if strings.Contains(d.Message, "empty Target") {
			found = true
		}
	}
	assert.True(t, found, "Empty target should emit debug diagnostic")
}

// --- Dedup Key Verification ---

func TestEdge_DedupKey_Format(t *testing.T) {
	d := DataflowDetection{
		FunctionFQN:  "app.main",
		SourceLine:   10,
		SourceColumn: 5,
		SinkLine:     20,
		SinkColumn:   8,
		SinkCall:     "eval",
		MatchMethod:  "type_inference",
	}

	key := dedupKey(d)
	assert.Equal(t, "app.main:10:5:20:8:eval:type_inference", key)
}

func TestEdge_IntersectKey_Format(t *testing.T) {
	d := DataflowDetection{
		FunctionFQN:  "app.main",
		SourceLine:   10,
		SourceColumn: 5,
		SinkLine:     20,
		SinkColumn:   8,
	}

	key := intersectKey(d)
	assert.Equal(t, "app.main:10:5:20:8", key)
}

// --- Configuration Edge Cases ---

func TestEdge_MinConfidence_NegativeClamped(t *testing.T) {
	dc := NewDiagnosticCollector()
	ir := &TypeConstrainedCallIR{
		ReceiverType:  "module.Class",
		MethodName:    "method",
		MinConfidence: -5.0,
	}
	err := validateTypeConstrainedCallIR(ir, dc)
	require.NoError(t, err)
	assert.Equal(t, 0.0, ir.MinConfidence, "Negative MinConfidence should be clamped to 0.0")
}

func TestEdge_MinConfidence_OverOneClamped(t *testing.T) {
	dc := NewDiagnosticCollector()
	ir := &TypeConstrainedCallIR{
		ReceiverType:  "module.Class",
		MethodName:    "method",
		MinConfidence: 99.0,
	}
	err := validateTypeConstrainedCallIR(ir, dc)
	require.NoError(t, err)
	assert.Equal(t, 1.0, ir.MinConfidence, "MinConfidence > 1.0 should be clamped to 1.0")
}

// --- Cross-Function E2E with logic composition ---

func TestEdge_LogicOrAnd_Composition(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "exec", Location: core.Location{Line: 2}},
		{Target: "print", Location: core.Location{Line: 3}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// logic_and(logic_or(eval, exec), logic_or(exec, print)) → intersection = exec
	matcherMap := map[string]any{
		"type": "logic_and",
		"matchers": []any{
			map[string]any{
				"type": "logic_or",
				"matchers": []any{
					map[string]any{"type": "call_matcher", "patterns": []any{"eval"}},
					map[string]any{"type": "call_matcher", "patterns": []any{"exec"}},
				},
			},
			map[string]any{
				"type": "logic_or",
				"matchers": []any{
					map[string]any{"type": "call_matcher", "patterns": []any{"exec"}},
					map[string]any{"type": "call_matcher", "patterns": []any{"print"}},
				},
			},
		},
	}

	ruleIR := &RuleIR{Matcher: matcherMap}
	results, err := loader.ExecuteRule(ruleIR, cg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "exec", results[0].SinkCall)
}

func TestEdge_LogicNot_ComposedWithAnd(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "exec", Location: core.Location{Line: 2}},
		{Target: "print", Location: core.Location{Line: 3}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// logic_and(logic_not(eval), call_matcher(exec, print))
	// Not(eval) = {exec, print}; call_matcher(exec,print) = {exec, print}
	// Intersection = {exec, print}
	matcherMap := map[string]any{
		"type": "logic_and",
		"matchers": []any{
			map[string]any{
				"type": "logic_not",
				"matchers": []any{
					map[string]any{"type": "call_matcher", "patterns": []any{"eval"}},
				},
			},
			map[string]any{
				"type":     "call_matcher",
				"patterns": []any{"exec", "print"},
			},
		},
	}

	ruleIR := &RuleIR{Matcher: matcherMap}
	results, err := loader.ExecuteRule(ruleIR, cg)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	calls := map[string]bool{}
	for _, r := range results {
		calls[r.SinkCall] = true
	}
	assert.True(t, calls["exec"])
	assert.True(t, calls["print"])
	assert.False(t, calls["eval"])
}

// --- SourceColumn/SinkColumn Population Tests ---

func TestEdge_ColumnPopulated_TypeConstrainedCall(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "method",
			InferredType:             "module.Class",
			ResolvedViaTypeInference: true,
			TypeConfidence:           0.9,
			Location:                 core.Location{Line: 10, Column: 42},
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method",
		FallbackMode: "none",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	results := executor.Execute()
	require.Len(t, results, 1)
	assert.Equal(t, 42, results[0].SourceColumn)
	assert.Equal(t, 42, results[0].SinkColumn)
}

func TestEdge_ColumnPopulated_CallMatcher(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 5, Column: 15}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	matcherMap := map[string]any{
		"type":     "call_matcher",
		"patterns": []any{"eval"},
	}

	ruleIR := &RuleIR{Matcher: matcherMap}
	results, err := loader.ExecuteRule(ruleIR, cg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 15, results[0].SourceColumn)
	assert.Equal(t, 15, results[0].SinkColumn)
}
