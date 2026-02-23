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
	t.Run("logic operators return empty for now", func(t *testing.T) {
		cg := core.NewCallGraph()
		loader := NewRuleLoader("")

		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "logic_and",
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

// Helper: Create temporary Python file for testing.
func createTempPythonFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_rules.py")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	return tmpFile
}
