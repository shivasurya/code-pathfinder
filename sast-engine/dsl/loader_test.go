package dsl

import (
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
		rulesContent := `from codepathfinder import rule, calls

@rule(id="test-eval", severity="high", cwe="CWE-94")
def detect_eval():
    """Test rule."""
    return calls("eval")

if __name__ == "__main__":
    import json
    print(json.dumps([detect_eval.execute()], indent=2))
`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		rules, err := loader.LoadRules()

		require.NoError(t, err)
		assert.Len(t, rules, 1)
		assert.Equal(t, "test-eval", rules[0].Rule.ID)
		assert.Equal(t, "high", rules[0].Rule.Severity)
		assert.Equal(t, "CWE-94", rules[0].Rule.CWE)
	})

	t.Run("handles invalid Python syntax", func(t *testing.T) {
		rulesContent := `this is not valid python`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		_, err := loader.LoadRules()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute Python rules")
	})

	t.Run("handles invalid JSON output", func(t *testing.T) {
		rulesContent := `print("not json")`
		tmpFile := createTempPythonFile(t, rulesContent)
		defer os.Remove(tmpFile)

		loader := NewRuleLoader(tmpFile)
		_, err := loader.LoadRules()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse rule JSON IR")
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		loader := NewRuleLoader("/nonexistent/file.py")
		_, err := loader.LoadRules()

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
			Matcher: map[string]interface{}{
				"type":     "call_matcher",
				"patterns": []interface{}{"eval"},
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
			Matcher: map[string]interface{}{
				"type": "dataflow",
				"sources": []interface{}{
					map[string]interface{}{
						"patterns": []interface{}{"request.GET"},
						"wildcard": false,
					},
				},
				"sinks": []interface{}{
					map[string]interface{}{
						"patterns": []interface{}{"eval"},
						"wildcard": false,
					},
				},
				"sanitizers": []interface{}{},
				"propagation": []interface{}{},
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
			Matcher: map[string]interface{}{
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
			Matcher: map[string]interface{}{
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
			Matcher: map[string]interface{}{
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
			Matcher: map[string]interface{}{
				"type": "logic_and",
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)

		require.NoError(t, err)
		assert.Empty(t, detections)
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
