package dsl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleLoader_New(t *testing.T) {
	loader := NewRuleLoader("test_rules.py")
	assert.NotNil(t, loader)
	assert.Equal(t, "test_rules.py", loader.RulesPath)
}

func TestRuleLoader_LoadRules(t *testing.T) {
	t.Run("loads valid Python DSL rules", func(t *testing.T) {
		// Create test rules file
		// Note: Rules auto-execute when run as __main__ (no manual __main__ block needed)
		rulesContent := `from codepathfinder import rule, calls

@rule(id="test-eval", severity="high", cwe="CWE-94")
def detect_eval():
    """Test rule."""
    return calls("eval")
`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		rules, err := loader.LoadRules(nil)

		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, "test-eval", rules[0].Rule.ID)
		assert.Equal(t, "high", rules[0].Rule.Severity)
		assert.Equal(t, "CWE-94", rules[0].Rule.CWE)
	})

	t.Run("returns empty list for file without rule decorators", func(t *testing.T) {
		// Normal Python file without @rule decorator
		// After fix: should return empty list instead of error (consistent with directory behavior)
		appContent := `def main():
    print("Hello world")

if __name__ == "__main__":
    main()
`
		tmpFile := createTempPythonFile(t, appContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		rules, err := loader.LoadRules(nil)

		require.NoError(t, err)
		assert.Empty(t, rules, "should return empty list for file without code analysis rules")
	})

	t.Run("returns empty list for container rule file without code analysis rules", func(t *testing.T) {
		// Container rule file with @dockerfile_rule but no @rule decorator
		// This tests the fix: single container rule files should return empty list, not error
		containerRuleContent := `from rules.container_decorators import dockerfile_rule
from rules.container_matchers import instruction

@dockerfile_rule(
    id="TEST-DOCKER-001",
    name="Test Container Rule",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Test container rule"
)
def test_container_rule():
    return instruction(type="USER")
`
		tmpFile := createTempPythonFile(t, containerRuleContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		rules, err := loader.LoadRules(nil)

		require.NoError(t, err)
		assert.Empty(t, rules, "should return empty list for container rule file (handled by LoadContainerRules)")
	})

	t.Run("skips files without rule decorators in directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid rule file
		validRule := `from codepathfinder import rule, calls

@rule(id="test-rule", severity="high", cwe="")
def test():
    return calls("eval")
`
		err := os.WriteFile(filepath.Join(tmpDir, "valid_rule.py"), []byte(validRule), 0644)
		require.NoError(t, err)

		// Create normal app file (should be skipped)
		appFile := `def main():
    print("Hello")
`
		err = os.WriteFile(filepath.Join(tmpDir, "app.py"), []byte(appFile), 0644)
		require.NoError(t, err)

		loader := NewRuleLoader(tmpDir)
		rules, err := loader.LoadRules(nil)

		require.NoError(t, err)
		// Should only load the valid rule, not the app.py
		assert.Len(t, rules, 1)
		assert.Equal(t, "test-rule", rules[0].Rule.ID)
	})

	t.Run("loads code analysis rules and skips container rules in directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create code analysis rule file
		codeRule := `from codepathfinder import rule, calls

@rule(id="code-rule", severity="high", cwe="")
def test_code():
    return calls("eval")
`
		err := os.WriteFile(filepath.Join(tmpDir, "code_rule.py"), []byte(codeRule), 0644)
		require.NoError(t, err)

		// Create container rule file (should be skipped by LoadRules)
		containerRule := `from rules.container_decorators import dockerfile_rule

@dockerfile_rule(id="container-rule", name="Container", severity="HIGH", cwe="", category="security", message="msg")
def test_container():
    return missing(instruction="USER")
`
		err = os.WriteFile(filepath.Join(tmpDir, "container_rule.py"), []byte(containerRule), 0644)
		require.NoError(t, err)

		loader := NewRuleLoader(tmpDir)
		rules, err := loader.LoadRules(nil)

		require.NoError(t, err)
		// Should only load the code analysis rule, skip container rule
		assert.Len(t, rules, 1)
		assert.Equal(t, "code-rule", rules[0].Rule.ID)
	})

	t.Run("handles invalid Python syntax", func(t *testing.T) {
		// Include decorator so it passes early filtering
		rulesContent := `from codepathfinder import rule
this is not valid python`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		_, err := loader.LoadRules(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute Python rules")
	})

	t.Run("handles invalid JSON output", func(t *testing.T) {
		// Include decorator so it passes early filtering
		rulesContent := `from codepathfinder import rule
print("not json")`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		_, err := loader.LoadRules(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse rule JSON IR")
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		loader := NewRuleLoader("/nonexistent/file.py")
		_, err := loader.LoadRules(nil)

		assert.Error(t, err)
	})
}

func TestRuleLoader_ExecuteRule(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["test.main"] = []core.CallSite{
		{
			Target:   "eval",
			Location: core.Location{File: "test.py", Line: 10},
		},
		{
			Target:   "exec",
			Location: core.Location{File: "test.py", Line: 15},
		},
	}

	t.Run("executes call_matcher rule", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type":     "call_matcher",
				"patterns": []any{"eval"},
				"wildcard": false,
			},
		}

		loader := NewRuleLoader("")
		detections, err := loader.ExecuteRule(rule, cg)

		require.NoError(t, err)
		assert.Len(t, detections, 1)
		assert.Equal(t, "eval", detections[0].SinkCall)
		assert.Equal(t, 10, detections[0].SinkLine)
	})

	t.Run("executes dataflow rule", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "dataflow",
				"sources": []any{
					map[string]any{
						"patterns": []any{"request.GET"},
						"wildcard": false,
					},
				},
				"sinks": []any{
					map[string]any{
						"patterns": []any{"eval"},
						"wildcard": false,
					},
				},
				"sanitizers":  []any{},
				"propagation": []any{},
				"scope":       "local",
			},
		}

		loader := NewRuleLoader("")
		detections, err := loader.ExecuteRule(rule, cg)

		require.NoError(t, err)
		assert.NotNil(t, detections)
	})

	t.Run("executes variable_matcher rule", func(t *testing.T) {
		cg2 := core.NewCallGraph()
		cg2.CallSites["test.func"] = []core.CallSite{
			{
				Target: "process",
				Arguments: []core.Argument{
					{Value: "user_input", IsVariable: true, Position: 0},
				},
				Location: core.Location{Line: 20},
			},
		}

		rule := &RuleIR{
			Matcher: map[string]any{
				"type":     "variable_matcher",
				"pattern":  "user_input",
				"wildcard": false,
			},
		}

		loader := NewRuleLoader("")
		detections, err := loader.ExecuteRule(rule, cg2)

		require.NoError(t, err)
		assert.Len(t, detections, 1)
		assert.Equal(t, "user_input", detections[0].TaintedVar)
	})

	t.Run("handles invalid matcher type", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "invalid_type",
			},
		}

		loader := NewRuleLoader("")
		_, err := loader.ExecuteRule(rule, cg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown matcher type")
	})

	t.Run("handles missing matcher type", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"no_type_field": "value",
			},
		}

		loader := NewRuleLoader("")
		_, err := loader.ExecuteRule(rule, cg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "matcher type not found")
	})

	t.Run("handles non-map matcher", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: "not a map",
		}

		loader := NewRuleLoader("")
		_, err := loader.ExecuteRule(rule, cg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "matcher is not a map")
	})
}

func TestRuleLoader_ExecuteLogic(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{
					Target:   "hashlib.md5",
					Location: core.Location{File: "test.py", Line: 5},
				},
				{
					Target:   "hashlib.sha1",
					Location: core.Location{File: "test.py", Line: 10},
				},
				{
					Target:   "hashlib.sha256",
					Location: core.Location{File: "test.py", Line: 15},
				},
			},
		},
	}
	loader := NewRuleLoader("")

	t.Run("logic_or returns union of results", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "logic_or",
				"matchers": []any{
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"hashlib.md5"},
						"wildcard": false,
					},
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"hashlib.sha1"},
						"wildcard": false,
					},
				},
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.Len(t, detections, 2)
	})

	t.Run("logic_and returns intersection", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "logic_and",
				"matchers": []any{
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"hashlib.*"},
						"wildcard": true,
					},
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"hashlib.md5"},
						"wildcard": false,
					},
				},
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.Len(t, detections, 1)
		assert.Equal(t, "hashlib.md5", detections[0].SinkCall)
	})

	t.Run("logic_or deduplicates", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "logic_or",
				"matchers": []any{
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"hashlib.md5"},
						"wildcard": false,
					},
					map[string]any{
						"type":     "call_matcher",
						"patterns": []any{"hashlib.md5"},
						"wildcard": false,
					},
				},
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.Len(t, detections, 1)
	})

	t.Run("logic_not returns universe when no matchers key", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type":    "logic_not",
				"matcher": map[string]any{"type": "call_matcher", "patterns": []any{"x"}, "wildcard": false},
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		// "matcher" (singular) is not recognized; treated as no matchers → entire universe returned.
		assert.NotEmpty(t, detections)
	})

	t.Run("logic_and with no matchers returns nil", func(t *testing.T) {
		rule := &RuleIR{
			Matcher: map[string]any{
				"type":     "logic_and",
				"matchers": []any{},
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		require.NoError(t, err)
		assert.Empty(t, detections)
	})
}

func TestRuleLoader_LoadContainerRules(t *testing.T) {
	t.Run("loads container rules from single file", func(t *testing.T) {
		// Create test container rule file
		// NOTE: Rule files should NOT have if __name__ == "__main__" blocks
		// The loader's wrapper script handles compilation via container_ir.compile_all_rules()
		rulesContent := `from rules.container_decorators import dockerfile_rule
from rules.container_matchers import missing

@dockerfile_rule(
    id="TEST-001",
    name="Test Rule",
    severity="HIGH",
    cwe="CWE-250",
    category="security",
    message="Test message"
)
def test_rule():
    return missing(instruction="USER")
`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		jsonData, err := loader.LoadContainerRules(nil)

		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify it's valid JSON
		var result map[string]any
		err = json.Unmarshal(jsonData, &result)
		require.NoError(t, err)
		assert.Contains(t, result, "dockerfile")
		assert.Contains(t, result, "compose")
	})

	t.Run("loads container rules from directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create multiple rule files
		// NOTE: Rule files should NOT have if __name__ == "__main__" blocks
		rule1 := `from rules.container_decorators import dockerfile_rule
from rules.container_matchers import missing

@dockerfile_rule(id="RULE-1", name="Rule 1", severity="HIGH", cwe="", category="security", message="msg")
def rule1():
    return missing(instruction="USER")
`

		rule2 := `from rules.container_decorators import compose_rule
from rules.container_matchers import service_has

@compose_rule(id="RULE-2", name="Rule 2", severity="CRITICAL", cwe="", category="security", message="msg")
def rule2():
    return service_has(key="privileged", equals=True)
`

		err := os.WriteFile(filepath.Join(tmpDir, "rule1.py"), []byte(rule1), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "rule2.py"), []byte(rule2), 0644)
		require.NoError(t, err)

		loader := NewRuleLoader(tmpDir)
		jsonData, err := loader.LoadContainerRules(nil)

		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)
	})

	t.Run("handles nonexistent path", func(t *testing.T) {
		loader := NewRuleLoader("/nonexistent/path")
		_, err := loader.LoadContainerRules(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to access rules path")
	})

	t.Run("handles invalid Python in container rules", func(t *testing.T) {
		rulesContent := `this is not valid python`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		_, err := loader.LoadContainerRules(nil)

		assert.Error(t, err)
	})

	t.Run("returns error for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		loader := NewRuleLoader(tmpDir)
		_, err := loader.LoadContainerRules(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no container rules detected")
	})

	t.Run("returns error for directory without container rule decorators", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create Python file without container rule decorators (code analysis rule)
		rulesContent := `from codepathfinder import rule, calls

@rule(id="test-eval", severity="high", cwe="CWE-94")
def detect_eval():
    """Test rule."""
    return calls("eval")
`
		err := os.WriteFile(filepath.Join(tmpDir, "code_rule.py"), []byte(rulesContent), 0644)
		require.NoError(t, err)

		loader := NewRuleLoader(tmpDir)
		_, err = loader.LoadContainerRules(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no container rules detected")
		assert.Contains(t, err.Error(), "no @dockerfile_rule or @compose_rule decorators found")
	})

	t.Run("returns error for file without container rule decorators", func(t *testing.T) {
		rulesContent := `from codepathfinder import rule, calls

@rule(id="test-eval", severity="high", cwe="CWE-94")
def detect_eval():
    """Test rule."""
    return calls("eval")
`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		_, err := loader.LoadContainerRules(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no container rules detected")
	})
}

func TestRuleLoader_LoadRulesFromFile_ContainerFormat(t *testing.T) {
	t.Run("returns empty list for container format in LoadRules", func(t *testing.T) {
		// Create a file that outputs container format (not code analysis format)
		// After fix: should return empty list since it doesn't have @rule decorator
		rulesContent := `if __name__ == "__main__":
    import json
    print(json.dumps({"dockerfile": [], "compose": []}))
`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		rules, err := loader.LoadRules(nil)

		// Should return empty list (consistent with directory behavior)
		require.NoError(t, err)
		assert.Empty(t, rules, "container format file should return empty list from LoadRules")
	})
}

// --- executeLogicOr edge cases ---

func TestExecuteLogicOr_NonMapElement(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{Target: "hashlib.md5", Location: core.Location{File: "test.py", Line: 5}},
			},
		},
	}
	loader := NewRuleLoader("")

	// One valid matcher plus a non-map element (string) — non-map should be skipped
	rule := &RuleIR{
		Matcher: map[string]any{
			"type": "logic_or",
			"matchers": []any{
				"not_a_map",
				map[string]any{
					"type":     "call_matcher",
					"patterns": []any{"hashlib.md5"},
					"wildcard": false,
				},
			},
		},
	}

	detections, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)
	assert.Len(t, detections, 1, "Should skip non-map element and still find valid match")
}

func TestExecuteLogicAnd_FirstMatcherNonMap(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{Target: "hashlib.md5", Location: core.Location{File: "test.py", Line: 5}},
			},
		},
	}
	loader := NewRuleLoader("")

	// First matcher is not a map — should error
	rule := &RuleIR{
		Matcher: map[string]any{
			"type": "logic_and",
			"matchers": []any{
				"not_a_map",
				map[string]any{
					"type":     "call_matcher",
					"patterns": []any{"hashlib.md5"},
					"wildcard": false,
				},
			},
		},
	}

	_, err := loader.ExecuteRule(rule, cg)
	assert.Error(t, err, "First matcher as non-map should cause error")
	assert.Contains(t, err.Error(), "matcher is not a map")
}

func TestExecuteLogicAnd_SubsequentMatcherNonMap(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{
			"myapp.test": {
				{Target: "hashlib.md5", Location: core.Location{File: "test.py", Line: 5}},
			},
		},
	}
	loader := NewRuleLoader("")

	// First matcher valid, second is non-map — non-map should be skipped
	rule := &RuleIR{
		Matcher: map[string]any{
			"type": "logic_and",
			"matchers": []any{
				map[string]any{
					"type":     "call_matcher",
					"patterns": []any{"hashlib.md5"},
					"wildcard": false,
				},
				"not_a_map",
			},
		},
	}

	detections, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)
	// The second matcher is skipped so no intersection happens with it;
	// result should still contain the first matcher's detections.
	assert.Len(t, detections, 1, "Should skip non-map subsequent matcher")
}

func TestExecuteLogicOr_ErrorPropagation(t *testing.T) {
	cg := &core.CallGraph{
		CallSites: map[string][]core.CallSite{},
	}
	loader := NewRuleLoader("")

	// Inner matcher has an unknown type → ExecuteRule returns error → logic_or propagates it
	rule := &RuleIR{
		Matcher: map[string]any{
			"type": "logic_or",
			"matchers": []any{
				map[string]any{
					"type": "totally_invalid_type",
				},
			},
		},
	}

	_, err := loader.ExecuteRule(rule, cg)
	assert.Error(t, err, "Error from inner matcher should propagate through logic_or")
	assert.Contains(t, err.Error(), "unknown matcher type")
}

// --- Container matcher types return empty result ---

func TestExecuteRule_ContainerMatcherTypes(t *testing.T) {
	cg := core.NewCallGraph()
	loader := NewRuleLoader("")

	containerTypes := []string{
		"missing_instruction",
		"instruction",
		"service_has",
		"service_missing",
		"any_of",
		"all_of",
		"none_of",
	}

	for _, matcherType := range containerTypes {
		t.Run(matcherType, func(t *testing.T) {
			rule := &RuleIR{
				Matcher: map[string]any{
					"type": matcherType,
				},
			}
			detections, err := loader.ExecuteRule(rule, cg)
			require.NoError(t, err)
			assert.Empty(t, detections, "Container matcher %s should return empty result", matcherType)
		})
	}
}

// --- extractInheritanceChecker edge cases ---

func TestExtractInheritanceChecker_NilCallGraph(t *testing.T) {
	checker := extractInheritanceChecker(nil)
	assert.Nil(t, checker, "nil callgraph should return nil checker")
}

func TestExtractInheritanceChecker_NoRemotes(t *testing.T) {
	cg := core.NewCallGraph()
	// ThirdPartyRemote and StdlibRemote are nil by default
	checker := extractInheritanceChecker(cg)
	assert.Nil(t, checker, "callgraph with no remotes should return nil checker")
}

// --- deduplicateDetections and intersectDetections ---

func TestDeduplicateDetections(t *testing.T) {
	dets := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "eval"},
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "eval"},
		{FunctionFQN: "test.b", SourceLine: 2, SinkLine: 20, SinkCall: "exec"},
	}

	result := deduplicateDetections(dets)
	assert.Len(t, result, 2, "Should remove duplicate detection")
	fqns := map[string]bool{}
	for _, d := range result {
		fqns[d.FunctionFQN] = true
	}
	assert.True(t, fqns["test.a"])
	assert.True(t, fqns["test.b"])
}

func TestDeduplicateDetections_Empty(t *testing.T) {
	result := deduplicateDetections([]DataflowDetection{})
	assert.Empty(t, result)
}

func TestIntersectDetections(t *testing.T) {
	// Intersection now keys on FunctionFQN:SourceLine:SourceColumn:SinkLine:SinkColumn.
	a := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "eval"},
		{FunctionFQN: "test.b", SourceLine: 2, SinkLine: 20, SinkCall: "exec"},
		{FunctionFQN: "test.c", SourceLine: 3, SinkLine: 30, SinkCall: "system"},
	}
	b := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "render"},
		{FunctionFQN: "test.c", SourceLine: 3, SinkLine: 30, SinkCall: "output"},
	}

	result := intersectDetections(a, b)
	assert.Len(t, result, 2, "Intersection should return detections present in both (by full key)")
	fqns := map[string]bool{}
	for _, d := range result {
		fqns[d.FunctionFQN] = true
	}
	assert.True(t, fqns["test.a"])
	assert.True(t, fqns["test.c"])
}

func TestIntersectDetections_NoOverlap(t *testing.T) {
	a := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1},
	}
	b := []DataflowDetection{
		{FunctionFQN: "test.b", SourceLine: 2},
	}

	result := intersectDetections(a, b)
	assert.Empty(t, result, "No overlap should return empty")
}

func TestDedup_DifferentMatchMethod_BothKept(t *testing.T) {
	dets := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SourceColumn: 5, SinkLine: 10, SinkColumn: 8, SinkCall: "eval", MatchMethod: "type_inference", Confidence: 0.9},
		{FunctionFQN: "test.a", SourceLine: 1, SourceColumn: 5, SinkLine: 10, SinkColumn: 8, SinkCall: "eval", MatchMethod: "fqn_bridge", Confidence: 0.7},
	}

	result := deduplicateDetections(dets)
	assert.Len(t, result, 2, "Different MatchMethod → different dedup keys → both kept")
}

func TestDedup_SameMatchMethod_HighestConfidence(t *testing.T) {
	dets := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "eval", MatchMethod: "type_inference", Confidence: 0.5},
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "eval", MatchMethod: "type_inference", Confidence: 0.9},
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10, SinkCall: "eval", MatchMethod: "type_inference", Confidence: 0.3},
	}

	result := deduplicateDetections(dets)
	require.Len(t, result, 1, "Same key → single entry")
	assert.Equal(t, 0.9, result[0].Confidence, "Highest confidence wins")
}

func TestDedup_ColumnDisambiguates(t *testing.T) {
	dets := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SourceColumn: 5, SinkLine: 10, SinkColumn: 8, SinkCall: "eval"},
		{FunctionFQN: "test.a", SourceLine: 1, SourceColumn: 20, SinkLine: 10, SinkColumn: 8, SinkCall: "eval"},
	}

	result := deduplicateDetections(dets)
	assert.Len(t, result, 2, "Different SourceColumn → different dedup keys → both kept")
}

func TestIntersect_IncludesSinkLine(t *testing.T) {
	a := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10},
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 20},
	}
	b := []DataflowDetection{
		{FunctionFQN: "test.a", SourceLine: 1, SinkLine: 10},
	}

	result := intersectDetections(a, b)
	assert.Len(t, result, 1, "Different SinkLine → different intersection keys")
	assert.Equal(t, 10, result[0].SinkLine)
}

// --- type_constrained_call via ExecuteRule ---

func TestExecuteRule_TypeConstrainedCall_ViaLoader(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["myapp.views.index"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			TargetFQN:                "sqlite3.Cursor.execute",
			Location:                 core.Location{File: "views.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	loader := NewRuleLoader("")
	rule := &RuleIR{
		Matcher: map[string]any{
			"type":          "type_constrained_call",
			"receiverTypes": []any{"sqlite3.Cursor"},
			"methodNames":   []any{"execute"},
			"minConfidence": 0.5,
			"fallbackMode":  "none",
		},
	}

	detections, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)
	assert.NotEmpty(t, detections, "Should find type_constrained_call match")
	assert.Equal(t, "myapp.views.index", detections[0].FunctionFQN)
}

// --- type_constrained_attribute via ExecuteRule ---

func TestExecuteRule_TypeConstrainedAttribute_ViaLoader(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["myapp.views.index"] = []core.CallSite{
		{
			Target:                   "request.GET",
			TargetFQN:                "django.http.HttpRequest.GET",
			Location:                 core.Location{File: "views.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "django.http.HttpRequest",
			TypeConfidence:           0.9,
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

	detections, err := loader.ExecuteRule(rule, cg)
	require.NoError(t, err)
	assert.NotEmpty(t, detections, "Should find type_constrained_attribute match")
	assert.Equal(t, "myapp.views.index", detections[0].FunctionFQN)
}

// Helper: Create temporary Python file for testing.
func createTempPythonFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_rules.py")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	return tmpFile
}
