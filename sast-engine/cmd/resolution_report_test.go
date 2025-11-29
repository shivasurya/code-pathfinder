package cmd

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// Note: categorizeResolutionFailure is in graph/callgraph/builder.go, not cmd package
// This test file validates the resolution report output formatting

func TestAggregateResolutionStatistics(t *testing.T) {
	// Create a mock call graph with various call sites
	cg := core.NewCallGraph()

	// Add resolved call sites
	cg.AddCallSite("test.func1", core.CallSite{
		Target:    "print",
		Resolved:  true,
		TargetFQN: "builtins.print",
	})

	// Add unresolved call sites with different failure reasons
	cg.AddCallSite("test.func2", core.CallSite{
		Target:        "models.ForeignKey",
		Resolved:      false,
		TargetFQN:     "django.db.models.ForeignKey",
		FailureReason: "external_framework",
	})

	cg.AddCallSite("test.func3", core.CallSite{
		Target:        "Task.objects.filter",
		Resolved:      false,
		TargetFQN:     "tasks.models.Task.objects.filter",
		FailureReason: "orm_pattern",
	})

	cg.AddCallSite("test.func4", core.CallSite{
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

func TestIsStdlibResolution(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected bool
	}{
		{"os module", "os.getcwd", true},
		{"sys module", "sys.argv", true},
		{"pathlib module", "pathlib.Path", true},
		{"json module", "json.dumps", true},
		{"non-stdlib module", "django.db.models", false},
		{"user module", "myproject.utils.helper", false},
		{"empty string", "", false},
		{"partial match not stdlib", "custom_os.func", false},
		{"datetime module", "datetime.datetime.now", true},
		{"collections module", "collections.OrderedDict", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStdlibResolution(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractModuleName(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected string
	}{
		{"os.path.join", "os.path.join", "os"},
		{"sys.argv", "sys.argv", "sys"},
		{"single component", "os", "os"},
		{"deeply nested", "a.b.c.d.e", "a"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractModuleName(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineStdlibType(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		expected string
	}{
		{"function", "os.getcwd", "function"},
		{"class", "pathlib.Path", "class"},
		{"method", "pathlib.Path.exists", "method"},
		{"constant starts with uppercase", "os.O_RDONLY", "class"}, // O_ starts with capital, detected as class
		{"nested function", "os.path.join", "function"},
		{"single name lowercase", "print", "function"},
		{"single name uppercase", "Exception", "class"},
		{"empty string", "", "unknown"},
		{"all caps constant", "sys.VERSION_INFO", "class"}, // VERSION_INFO starts with V, detected as class
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineStdlibType(tt.fqn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAggregateResolutionStatistics_WithStdlib(t *testing.T) {
	// Create a mock call graph with stdlib call sites
	cg := core.NewCallGraph()

	// Add stdlib resolved via builtin registry
	cg.AddCallSite("test.func1", core.CallSite{
		Target:     "getcwd",
		Resolved:   true,
		TargetFQN:  "os.getcwd",
		TypeSource: "builtin",
	})

	// Add stdlib resolved via annotation
	cg.AddCallSite("test.func2", core.CallSite{
		Target:     "dumps",
		Resolved:   true,
		TargetFQN:  "json.dumps",
		TypeSource: "stdlib_annotation",
	})

	// Add stdlib resolved via type inference
	cg.AddCallSite("test.func3", core.CallSite{
		Target:                   "Path",
		Resolved:                 true,
		TargetFQN:                "pathlib.Path",
		ResolvedViaTypeInference: true,
		TypeConfidence:           0.95,
	})

	// Add non-stdlib resolved call
	cg.AddCallSite("test.func4", core.CallSite{
		Target:    "myfunction",
		Resolved:  true,
		TargetFQN: "myproject.utils.myfunction",
	})

	// Aggregate statistics
	stats := aggregateResolutionStatistics(cg)

	// Validate overall counts
	assert.Equal(t, 4, stats.TotalCalls)
	assert.Equal(t, 4, stats.ResolvedCalls)
	assert.Equal(t, 0, stats.UnresolvedCalls)

	// Validate stdlib counts
	assert.Equal(t, 3, stats.StdlibResolved)
	assert.Equal(t, 1, stats.StdlibViaBuiltin)
	assert.Equal(t, 1, stats.StdlibViaAnnotation)
	assert.Equal(t, 1, stats.StdlibViaInference)

	// Validate module breakdown
	assert.Equal(t, 1, stats.StdlibByModule["os"])
	assert.Equal(t, 1, stats.StdlibByModule["json"])
	assert.Equal(t, 1, stats.StdlibByModule["pathlib"])

	// Validate type breakdown
	assert.Equal(t, 2, stats.StdlibByType["function"])
	assert.Equal(t, 1, stats.StdlibByType["class"])
}
