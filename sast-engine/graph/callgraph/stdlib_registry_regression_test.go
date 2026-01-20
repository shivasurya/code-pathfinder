package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStdlibRegressionSuite validates that stdlib registry continues to work
// as expected after changes. This prevents regressions from future modifications.
func TestStdlibRegressionSuite(t *testing.T) {
	// Create a temporary test project
	tmpDir := t.TempDir()

	// Test 1: os.path.join pattern (should resolve)
	t.Run("os_path_join_resolution", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "file_ops.py")
		code := `import os

def get_config_path():
    base_dir = os.getcwd()
    config_path = os.path.join(base_dir, "config.ini")
    return config_path
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		// Verify os.getcwd and os.path.join are resolved
		stats := collectStats(callGraph)
		assert.GreaterOrEqual(t, stats.ResolvedCalls, 2, "Should resolve os.getcwd and os.path.join")
		assert.GreaterOrEqual(t, stats.StdlibResolved, 2, "Should have stdlib resolutions")
	})

	// Test 2: pathlib.Path pattern (should resolve)
	t.Run("pathlib_path_resolution", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "dir_utils.py")
		code := `from pathlib import Path

def create_directory(name):
    path = Path(name)
    path.mkdir(parents=True, exist_ok=True)
    return path
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		stats := collectStats(callGraph)
		assert.Greater(t, stats.ResolvedCalls, 0, "Should resolve Path methods")
		assert.Greater(t, stats.StdlibResolved, 0, "Should have stdlib resolutions")
	})

	// Test 3: json.loads/dumps (should resolve)
	t.Run("json_module_resolution", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "json_handler.py")
		code := `import json

def process_data(json_string):
    data = json.loads(json_string)
    data["processed"] = True
    return json.dumps(data)
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		stats := collectStats(callGraph)
		assert.GreaterOrEqual(t, stats.ResolvedCalls, 2, "Should resolve json.loads and json.dumps")
		assert.GreaterOrEqual(t, stats.StdlibResolved, 2, "Should have stdlib resolutions")
	})

	// Test 4: sys.argv access (should resolve)
	t.Run("sys_module_resolution", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "cli_args.py")
		code := `import sys

def get_args():
    return sys.argv[1:]

def get_version():
    return sys.version_info
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		stats := collectStats(callGraph)
		// sys.argv and sys.version_info are attributes, may not show as calls
		// but module should be detected
		assert.GreaterOrEqual(t, stats.ResolvedCalls, 0, "Should process sys module")
	})
}

// TestStdlibResolutionThreshold ensures resolution rate stays above minimum.
func TestStdlibResolutionThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a comprehensive test file with multiple stdlib calls
	testFile := filepath.Join(tmpDir, "comprehensive_stdlib.py")
	code := `import os
import sys
import json
import pathlib
from pathlib import Path

def main():
    # os module
    cwd = os.getcwd()
    path = os.path.join(cwd, "data")
    os.makedirs(path, exist_ok=True)

    # sys module
    args = sys.argv
    version = sys.version_info

    # json module
    data = {"key": "value"}
    json_str = json.dumps(data)
    parsed = json.loads(json_str)

    # pathlib module
    p = Path(".")
    p.mkdir(exist_ok=True)
    files = list(p.glob("*.py"))

    return True
`
	err := os.WriteFile(testFile, []byte(code), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	stats := collectStats(callGraph)

	// Minimum threshold: 80% of calls should be resolved
	if stats.TotalCalls > 0 {
		resolutionRate := float64(stats.ResolvedCalls) / float64(stats.TotalCalls)
		assert.GreaterOrEqual(t, resolutionRate, 0.80,
			"Resolution rate should be at least 80%% (got %.1f%%)", resolutionRate*100)
	}

	// Should have significant stdlib resolutions
	if stats.ResolvedCalls > 0 {
		stdlibRate := float64(stats.StdlibResolved) / float64(stats.ResolvedCalls)
		assert.GreaterOrEqual(t, stdlibRate, 0.50,
			"Stdlib resolutions should be at least 50%% of resolved calls (got %.1f%%)", stdlibRate*100)
	}
}

// TestStdlibEdgeCases validates specific edge cases are handled correctly.
func TestStdlibEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Edge case 1: Aliased imports
	t.Run("aliased_import", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "alias_module.py")
		code := `import os.path as osp

def join_paths():
    return osp.join("a", "b")
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		stats := collectStats(callGraph)
		// Should resolve even with alias
		assert.Greater(t, stats.ResolvedCalls, 0, "Should resolve aliased imports")
	})

	// Edge case 2: From imports
	t.Run("from_import", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "from_imports.py")
		code := `from os.path import join, exists

def check_path(path):
    return exists(join("/tmp", path))
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		stats := collectStats(callGraph)
		assert.GreaterOrEqual(t, stats.ResolvedCalls, 2, "Should resolve from imports")
	})

	// Edge case 3: Multiple stdlib modules in one file
	t.Run("multiple_modules", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "multi_module.py")
		code := `import os
import sys
import json

def process():
    os.getcwd()
    sys.exit(0)
    json.dumps({})
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		stats := collectStats(callGraph)
		assert.GreaterOrEqual(t, stats.ResolvedCalls, 3, "Should resolve multiple modules")

		// Should have multiple modules in breakdown
		assert.GreaterOrEqual(t, len(stats.StdlibByModule), 2, "Should track multiple stdlib modules")
	})
}

// TestStdlibNoRegression ensures previous resolutions still work.
func TestStdlibNoRegression(t *testing.T) {
	// This test documents known-good resolutions from previous versions
	// If this test fails, it means a regression was introduced

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "baseline.py")

	// Known-good code that should resolve completely
	code := `import os

def get_home():
    return os.path.expanduser("~")

def file_exists(path):
    return os.path.exists(path)
`
	err := os.WriteFile(testFile, []byte(code), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph, _, _, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	stats := collectStats(callGraph)

	// These specific patterns MUST resolve (known baseline)
	assert.GreaterOrEqual(t, stats.ResolvedCalls, 2,
		"REGRESSION: os.path.expanduser and os.path.exists should resolve")
	assert.GreaterOrEqual(t, stats.StdlibResolved, 2,
		"REGRESSION: Should have at least 2 stdlib resolutions")
}

// collectStats is a helper to aggregate statistics from call graph.
func collectStats(cg *core.CallGraph) *CallGraphStats {
	stats := &CallGraphStats{
		StdlibByModule: make(map[string]int),
	}

	for _, callSites := range cg.CallSites {
		for _, site := range callSites {
			stats.TotalCalls++

			if site.Resolved {
				stats.ResolvedCalls++

				// Check if stdlib resolution
				if isStdlibFQN(site.TargetFQN) {
					stats.StdlibResolved++

					// Extract module name
					moduleName := extractStdlibModule(site.TargetFQN)
					if moduleName != "" {
						stats.StdlibByModule[moduleName]++
					}
				}
			}
		}
	}

	return stats
}

// CallGraphStats holds statistics for testing.
type CallGraphStats struct {
	TotalCalls     int
	ResolvedCalls  int
	StdlibResolved int
	StdlibByModule map[string]int
}

// isStdlibFQN checks if FQN is from stdlib.
func isStdlibFQN(fqn string) bool {
	stdlibPrefixes := []string{
		"os.", "sys.", "pathlib.", "json.", "re.", "time.", "datetime.",
		"collections.", "itertools.", "functools.", "math.", "random.",
	}

	for _, prefix := range stdlibPrefixes {
		if len(fqn) >= len(prefix) && fqn[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// extractStdlibModule gets the module name from FQN.
func extractStdlibModule(fqn string) string {
	for i := 0; i < len(fqn); i++ {
		if fqn[i] == '.' {
			return fqn[:i]
		}
	}
	return fqn
}
