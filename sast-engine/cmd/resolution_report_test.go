package cmd

import (
	"os"
	"strings"
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
	stats := aggregateResolutionStatistics(cg, "/project")

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

	// Validate unresolved details
	assert.Equal(t, 3, len(stats.UnresolvedDetails))
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
	stats := aggregateResolutionStatistics(cg, "/project")

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

func TestRelativePath(t *testing.T) {
	tests := []struct {
		name        string
		absPath     string
		projectRoot string
		expected    string
	}{
		{"normal", "/home/user/project/src/app.py", "/home/user/project", "src/app.py"},
		{"same dir", "/home/user/project/app.py", "/home/user/project", "app.py"},
		{"empty abs", "", "/home/user/project", ""},
		{"empty root", "/home/user/project/app.py", "", "/home/user/project/app.py"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := relativePath(tt.absPath, tt.projectRoot)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnresolvedDetails(t *testing.T) {
	cg := core.NewCallGraph()

	cg.AddCallSite("myapp.views.index", core.CallSite{
		Target:        "extend_schema",
		Resolved:      false,
		TargetFQN:     "extend_schema",
		FailureReason: "not_in_imports",
		Location: core.Location{
			File:   "/project/myapp/views.py",
			Line:   42,
			Column: 5,
		},
	})

	cg.AddCallSite("myapp.views.detail", core.CallSite{
		Target:        "get_env",
		Resolved:      false,
		TargetFQN:     "get_env",
		FailureReason: "not_in_imports",
		Location: core.Location{
			File:   "/project/myapp/utils.py",
			Line:   10,
			Column: 12,
		},
	})

	stats := aggregateResolutionStatistics(cg, "/project")

	assert.Equal(t, 2, len(stats.UnresolvedDetails))
	assert.Equal(t, 2, stats.UnresolvedCalls)

	// Verify details are sorted by file then line
	assert.Equal(t, "myapp/utils.py", stats.UnresolvedDetails[0].File)
	assert.Equal(t, 10, stats.UnresolvedDetails[0].Line)
	assert.Equal(t, "myapp/views.py", stats.UnresolvedDetails[1].File)
	assert.Equal(t, 42, stats.UnresolvedDetails[1].Line)

	// Verify per-file counts
	assert.Equal(t, 1, stats.UnresolvedByFile["myapp/views.py"])
	assert.Equal(t, 1, stats.UnresolvedByFile["myapp/utils.py"])
}

func TestExportUnresolvedCSV(t *testing.T) {
	cg := core.NewCallGraph()

	cg.AddCallSite("myapp.views.index", core.CallSite{
		Target:        "extend_schema",
		Resolved:      false,
		TargetFQN:     "drf_spectacular.extend_schema",
		FailureReason: "not_in_imports",
		Location: core.Location{
			File:   "/project/myapp/views.py",
			Line:   42,
			Column: 5,
		},
	})

	cg.AddCallSite("myapp.views.index", core.CallSite{
		Target:        "Response",
		Resolved:      false,
		TargetFQN:     "rest_framework.response.Response",
		FailureReason: "external_framework",
		Location: core.Location{
			File:   "/project/myapp/views.py",
			Line:   55,
			Column: 12,
		},
	})

	stats := aggregateResolutionStatistics(cg, "/project")

	// Write CSV to temp file
	tmpFile, err := os.CreateTemp("", "unresolved-*.csv")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = exportUnresolvedCSV(stats, tmpFile.Name())
	assert.NoError(t, err)

	// Read and verify CSV content
	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)

	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")

	// Header + 2 data rows
	assert.Equal(t, 3, len(lines))

	// Verify header
	assert.Equal(t, "file,line,column,function,target,target_fqn,reason", lines[0])

	// Verify data rows contain expected fields
	assert.Contains(t, lines[1], "myapp/views.py")
	assert.Contains(t, lines[1], "42")
	assert.Contains(t, lines[1], "extend_schema")
	assert.Contains(t, lines[1], "not_in_imports")

	assert.Contains(t, lines[2], "myapp/views.py")
	assert.Contains(t, lines[2], "55")
	assert.Contains(t, lines[2], "Response")
	assert.Contains(t, lines[2], "external_framework")
}

func TestContainsString(t *testing.T) {
	assert.True(t, containsString("builtins.str", "builtins."))
	assert.True(t, containsString("hello world", "world"))
	assert.False(t, containsString("hello", "world"))
	assert.False(t, containsString("", "test"))
}

func TestPrintPerFileBreakdown(t *testing.T) {
	stats := &resolutionStatistics{
		UnresolvedByFile: map[string]int{
			"app/views.py":  10,
			"app/models.py": 5,
			"app/utils.py":  3,
		},
	}

	// Should not panic with valid data.
	printPerFileBreakdown(stats, 2)

	// Should not panic with empty data.
	emptyStats := &resolutionStatistics{
		UnresolvedByFile: map[string]int{},
	}
	printPerFileBreakdown(emptyStats, 5)
}

func TestExportUnresolvedCSV_InvalidPath(t *testing.T) {
	stats := &resolutionStatistics{
		UnresolvedDetails: []unresolvedDetail{
			{File: "app.py", Line: 1, Target: "foo"},
		},
	}

	err := exportUnresolvedCSV(stats, "/nonexistent/dir/file.csv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create CSV file")
}

func TestAggregateResolutionStatistics_EmptyCallGraph(t *testing.T) {
	cg := core.NewCallGraph()
	stats := aggregateResolutionStatistics(cg, "/project")

	assert.Equal(t, 0, stats.TotalCalls)
	assert.Equal(t, 0, stats.ResolvedCalls)
	assert.Equal(t, 0, stats.UnresolvedCalls)
	assert.Equal(t, 0, len(stats.UnresolvedDetails))
}

func TestAggregateResolutionStatistics_FailureReasonBreakdown(t *testing.T) {
	cg := core.NewCallGraph()

	cg.AddCallSite("test.func1", core.CallSite{
		Target:        "super().save",
		Resolved:      false,
		TargetFQN:     "super().save",
		FailureReason: "super_call",
	})

	cg.AddCallSite("test.func2", core.CallSite{
		Target:        "self.method",
		Resolved:      false,
		TargetFQN:     "self.method",
		FailureReason: "variable_method",
	})

	cg.AddCallSite("test.func3", core.CallSite{
		Target:        "foo.bar",
		Resolved:      false,
		TargetFQN:     "foo.bar",
		FailureReason: "attribute_chain",
	})

	stats := aggregateResolutionStatistics(cg, "/project")

	assert.Equal(t, 3, stats.TotalCalls)
	assert.Equal(t, 0, stats.ResolvedCalls)
	assert.Equal(t, 3, stats.UnresolvedCalls)
	assert.Equal(t, 1, stats.FailuresByReason["super_call"])
	assert.Equal(t, 1, stats.FailuresByReason["variable_method"])
	assert.Equal(t, 1, stats.FailuresByReason["attribute_chain"])
}
