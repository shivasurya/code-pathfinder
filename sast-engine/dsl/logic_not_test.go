package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogicNot_BasicSubtraction(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "print", Location: core.Location{Line: 2}},
		{Target: "os.system", Location: core.Location{Line: 3}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Not(call_matcher["eval"]) → should return print and os.system.
	matcherMap := map[string]any{
		"type": "logic_not",
		"matchers": []any{
			map[string]any{
				"type":     "call_matcher",
				"patterns": []any{"eval"},
			},
		},
	}

	results, err := loader.executeLogicNot(matcherMap, cg)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	targets := map[string]bool{}
	for _, r := range results {
		targets[r.SinkCall] = true
		assert.Equal(t, "logic_not", r.MatchMethod)
		assert.Equal(t, 1.0, r.Confidence)
	}
	assert.True(t, targets["print"])
	assert.True(t, targets["os.system"])
	assert.False(t, targets["eval"])
}

func TestLogicNot_EmptyMatchers(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "print", Location: core.Location{Line: 2}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Not() with empty matchers → entire universe.
	matcherMap := map[string]any{
		"type":     "logic_not",
		"matchers": []any{},
	}

	results, err := loader.executeLogicNot(matcherMap, cg)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Entire universe.
}

func TestLogicNot_AllMatched(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Not(call_matcher["eval"]) when eval is the only call site → empty.
	matcherMap := map[string]any{
		"type": "logic_not",
		"matchers": []any{
			map[string]any{
				"type":     "call_matcher",
				"patterns": []any{"eval"},
			},
		},
	}

	results, err := loader.executeLogicNot(matcherMap, cg)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestLogicNot_NestedDoubleNegation(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "print", Location: core.Location{Line: 2}},
		{Target: "os.system", Location: core.Location{Line: 3}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Not(Not(call_matcher["eval"])) ≈ call_matcher["eval"].
	// Inner Not(eval) → {print, os.system}
	// Outer Not({print, os.system}) → {eval}
	matcherMap := map[string]any{
		"type": "logic_not",
		"matchers": []any{
			map[string]any{
				"type": "logic_not",
				"matchers": []any{
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"eval"},
					},
				},
			},
		},
	}

	// Use ExecuteRule for full recursive dispatch.
	ruleIR := &RuleIR{Matcher: matcherMap}
	results, err := loader.ExecuteRule(ruleIR, cg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "eval", results[0].SinkCall)
}

func TestLogicNot_LargeUniverse_Diagnostic(t *testing.T) {
	cg := core.NewCallGraph()
	for i := 0; i < 100; i++ {
		cg.CallSites["app.main"] = append(cg.CallSites["app.main"], core.CallSite{
			Target:   "call_" + string(rune('a'+i%26)),
			Location: core.Location{Line: i + 1},
		})
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	matcherMap := map[string]any{
		"type":     "logic_not",
		"matchers": []any{},
	}

	results, err := loader.executeLogicNot(matcherMap, cg)
	require.NoError(t, err)
	assert.Len(t, results, 100)

	// Check diagnostic with counts.
	debugs := dc.FilterByLevel("debug")
	require.NotEmpty(t, debugs)
	found := false
	for _, d := range debugs {
		if d.Component == "logic_not" {
			found = true
			assert.Contains(t, d.Message, "universe=100")
			assert.Contains(t, d.Message, "matched=0")
			assert.Contains(t, d.Message, "result=100")
		}
	}
	assert.True(t, found, "expected logic_not diagnostic with counts")
}

func TestLogicNot_NilCallGraph(t *testing.T) {
	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	matcherMap := map[string]any{
		"type":     "logic_not",
		"matchers": []any{},
	}

	results, err := loader.executeLogicNot(matcherMap, nil)
	require.NoError(t, err)
	assert.Nil(t, results)
	assert.True(t, dc.HasWarnings())
}

func TestLogicNot_ViaExecuteLogic(t *testing.T) {
	// Verify logic_not is correctly dispatched through executeLogic.
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "print", Location: core.Location{Line: 2}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	matcherMap := map[string]any{
		"type": "logic_not",
		"matchers": []any{
			map[string]any{
				"type":     "call_matcher",
				"patterns": []any{"eval"},
			},
		},
	}

	results, err := loader.executeLogic("logic_not", matcherMap, cg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "print", results[0].SinkCall)
}

func TestLogicNot_InvalidMatcherElement(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
		{Target: "print", Location: core.Location{Line: 2}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Matchers array contains a non-map element → should be skipped.
	matcherMap := map[string]any{
		"type":     "logic_not",
		"matchers": []any{"not_a_map"},
	}

	results, err := loader.executeLogicNot(matcherMap, cg)
	require.NoError(t, err)
	// No valid matchers → entire universe returned.
	assert.Len(t, results, 2)
}

func TestLogicNot_NestedMatcherError(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
	}

	dc := NewDiagnosticCollector()
	loader := &RuleLoader{Config: DefaultConfig(), Diagnostics: dc}

	// Nested matcher with unknown type → ExecuteRule returns error.
	matcherMap := map[string]any{
		"type": "logic_not",
		"matchers": []any{
			map[string]any{
				"type": "unknown_matcher_type",
			},
		},
	}

	_, err := loader.executeLogicNot(matcherMap, cg)
	assert.Error(t, err)
}
