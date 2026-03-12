package dsl

import (
	"math"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchesCallSite_EmptyTarget_Skipped(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"app.main": {
				{Target: "", Location: core.Location{Line: 1}},
				{Target: "cursor.execute", Location: core.Location{Line: 2}},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType: "sqlite3.Cursor",
			MethodName:   "execute",
			FallbackMode: "name",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	// Empty target should be skipped; only "cursor.execute" should match.
	assert.Len(t, results, 1)
	assert.Equal(t, "cursor.execute", results[0].SinkCall)

	// Diagnostic emitted for skipped empty target.
	debugEntries := dc.FilterByLevel("debug")
	require.NotEmpty(t, debugEntries)
	assert.Contains(t, debugEntries[0].Message, "empty Target")
}

func TestMatchesReceiverType_WhitespaceTrimming(t *testing.T) {
	// Whitespace in actual or pattern should be trimmed.
	assert.True(t, matchesReceiverType(" sqlite3.Cursor ", "sqlite3.Cursor", nil))
	assert.True(t, matchesReceiverType("sqlite3.Cursor", " sqlite3.Cursor ", nil))
	assert.True(t, matchesReceiverType("  sqlite3.Cursor  ", "  sqlite3.Cursor  ", nil))
	assert.False(t, matchesReceiverType("  ", "  ", nil))
}

func TestMatchesCallSite_NaNConfidence_Clamped(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"app.main": {
				{
					Target:                  "cursor.execute",
					Location:                core.Location{Line: 5},
					ResolvedViaTypeInference: true,
					InferredType:             "sqlite3.Cursor",
					TypeConfidence:           float32(math.NaN()),
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.0, // Accept any confidence.
			FallbackMode:  "name",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	require.NotEmpty(t, results)
	// NaN confidence should be clamped to 0.0, then replaced by FQN bridge confidence.
	assert.Equal(t, 0.7, results[0].Confidence)
	assert.True(t, dc.HasWarnings())
}

func TestMatchesCallSite_InfConfidence_Clamped(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"app.main": {
				{
					Target:                  "cursor.execute",
					Location:                core.Location{Line: 5},
					ResolvedViaTypeInference: true,
					InferredType:             "sqlite3.Cursor",
					TypeConfidence:           float32(math.Inf(1)),
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.0,
			FallbackMode:  "name",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	require.NotEmpty(t, results)
	// Inf confidence is clamped to 0.0 (NaN/Inf → 0.0), then replaced by FQN bridge confidence.
	assert.Equal(t, 0.7, results[0].Confidence)
	assert.True(t, dc.HasWarnings())
}

func TestMatchesCallSite_NegativeConfidence_Clamped(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"app.main": {
				{
					Target:                  "cursor.execute",
					Location:                core.Location{Line: 5},
					ResolvedViaTypeInference: true,
					InferredType:             "sqlite3.Cursor",
					TypeConfidence:           -5.0,
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.0,
			FallbackMode:  "name",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	require.NotEmpty(t, results)
	// Negative should be clamped to 0.0, then replaced by FQN bridge confidence.
	assert.Equal(t, 0.7, results[0].Confidence)
	assert.True(t, dc.HasWarnings())
}

func TestDataflowExecutor_NilCallGraph(t *testing.T) {
	dc := NewDiagnosticCollector()
	executor := &DataflowExecutor{
		IR: &DataflowIR{
			Scope: "local",
		},
		CallGraph:   nil,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Nil(t, results)
	assert.True(t, dc.HasErrors())
	errors := dc.FilterByLevel("error")
	require.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "CallGraph is nil")
}

func TestDataflowExecutor_NilCallSitesMap(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := &core.CallGraph{
		CallSites: nil,
	}
	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType: "test.Type",
			MethodName:   "method",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Nil(t, results)
	assert.True(t, dc.HasWarnings())
	warnings := dc.FilterByLevel("warning")
	assert.Contains(t, warnings[0].Message, "CallSites map is nil")
}

func TestArgumentMatcher_TupleIndexOutOfRange(t *testing.T) {
	// Tuple index "0[5]" where the tuple has fewer than 6 elements.
	args := []core.Argument{
		{Value: "(1, 2, 3)", Position: 0},
	}
	constraints := map[string]ArgumentConstraint{
		"0[5]": {Value: "6"},
	}
	// extractTupleElement will return false for out-of-range index.
	assert.False(t, MatchesPositionalArguments(args, constraints))
}

func TestExecutor_DiagnosticsField_NilSafe(t *testing.T) {
	// Executors with nil Diagnostics should not panic.
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"app.main": {
				{Target: "cursor.execute", Location: core.Location{Line: 1}},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType: "sqlite3.Cursor",
			MethodName:   "execute",
			FallbackMode: "name",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: nil, // Explicitly nil.
	}

	// Should not panic.
	results := executor.Execute()
	assert.NotNil(t, results)
}

func TestFallbackMode_Name_EmitsWarning(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"app.main": {
				{
					Target:   "cursor.execute",
					Location: core.Location{Line: 5},
					// No type inference, no FQN — will hit name_fallback.
				},
			},
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverType: "sqlite3.Cursor",
			MethodName:   "execute",
			FallbackMode: "name",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	require.Len(t, results, 1)
	assert.Equal(t, "name_fallback", results[0].MatchMethod)

	// Fallback warning should be emitted.
	warnings := dc.FilterByLevel("warning")
	require.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if w.Component == "type_match" {
			found = true
			assert.Contains(t, w.Message, "name_fallback")
			assert.Contains(t, w.Message, "no type info")
		}
	}
	assert.True(t, found, "expected type_match warning for name_fallback")
}

func TestTypeConstrainedAttributeExecutor_NilCallGraph(t *testing.T) {
	dc := NewDiagnosticCollector()
	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET",
		},
		CallGraph:   nil,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Nil(t, results)
	assert.True(t, dc.HasErrors())
}

func TestTypeConstrainedAttributeExecutor_NilCallSitesMap(t *testing.T) {
	dc := NewDiagnosticCollector()
	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "django.http.HttpRequest",
			AttributeName: "GET",
		},
		CallGraph:   &core.CallGraph{CallSites: nil},
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Nil(t, results)
	assert.True(t, dc.HasWarnings())
}

func TestClampTypeConfidence_NilDiagnostics(t *testing.T) {
	// clampTypeConfidence should not panic with nil diagnostics.
	assert.Equal(t, 0.5, clampTypeConfidence(0.5, nil))
	assert.Equal(t, 0.0, clampTypeConfidence(-1.0, nil))
	assert.Equal(t, 1.0, clampTypeConfidence(2.0, nil))
	assert.Equal(t, 0.0, clampTypeConfidence(math.NaN(), nil))
}
