package cmd

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/stretchr/testify/assert"
)

// Note: categorizeResolutionFailure is in graph/callgraph/builder.go, not cmd package
// This test file validates the resolution report output formatting

func TestAggregateResolutionStatistics(t *testing.T) {
	// Create a mock call graph with various call sites
	cg := callgraph.NewCallGraph()

	// Add resolved call sites
	cg.AddCallSite("test.func1", callgraph.CallSite{
		Target:    "print",
		Resolved:  true,
		TargetFQN: "builtins.print",
	})

	// Add unresolved call sites with different failure reasons
	cg.AddCallSite("test.func2", callgraph.CallSite{
		Target:        "models.ForeignKey",
		Resolved:      false,
		TargetFQN:     "django.db.models.ForeignKey",
		FailureReason: "external_framework",
	})

	cg.AddCallSite("test.func3", callgraph.CallSite{
		Target:        "Task.objects.filter",
		Resolved:      false,
		TargetFQN:     "tasks.models.Task.objects.filter",
		FailureReason: "orm_pattern",
	})

	cg.AddCallSite("test.func4", callgraph.CallSite{
		Target:        "response.json",
		Resolved:      false,
		TargetFQN:     "response.json",
		FailureReason: "variable_method",
	})

	// Aggregate statistics
	stats := aggregateResolutionStatistics(cg)

	// Validate overall counts
	assert.Equal(t, 4, stats.TotalCalls)
	assert.Equal(t, 1, stats.ResolvedCalls)
	assert.Equal(t, 3, stats.UnresolvedCalls)

	// Validate failure breakdown
	assert.Equal(t, 1, stats.FailuresByReason["external_framework"])
	assert.Equal(t, 1, stats.FailuresByReason["orm_pattern"])
	assert.Equal(t, 1, stats.FailuresByReason["variable_method"])

	// Validate pattern counts
	assert.Equal(t, 1, stats.PatternCounts["models.ForeignKey"])
	assert.Equal(t, 1, stats.PatternCounts["Task.objects.filter"])
	assert.Equal(t, 1, stats.PatternCounts["response.json"])

	// Validate framework counts
	assert.Equal(t, 1, stats.FrameworkCounts["django"])
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		name     string
		part     int
		total    int
		expected float64
	}{
		{"50 percent", 50, 100, 50.0},
		{"zero percent", 0, 100, 0.0},
		{"hundred percent", 100, 100, 100.0},
		{"zero total", 10, 0, 0.0},
		{"decimal result", 1, 3, 33.333333333333336},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percentage(tt.part, tt.total)
			assert.Equal(t, tt.expected, result)
		})
	}
}
