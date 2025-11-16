package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test rules.
func createTestRule(id, name, severity, cwe, owasp, description string) dsl.RuleIR {
	rule := dsl.RuleIR{}
	rule.Rule.ID = id
	rule.Rule.Name = name
	rule.Rule.Severity = severity
	rule.Rule.CWE = cwe
	rule.Rule.OWASP = owasp
	rule.Rule.Description = description
	return rule
}

func TestGenerateSARIFOutput(t *testing.T) {
	t.Run("generates valid SARIF output with detections", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rules := []dsl.RuleIR{
			createTestRule("sql-injection", "SQL Injection", "critical", "CWE-89", "A03:2021", "Detects SQL injection vulnerabilities"),
		}

		allDetections := map[string][]dsl.DataflowDetection{
			"sql-injection": {
				{
					FunctionFQN: "test.vulnerable",
					SourceLine:  10,
					SinkLine:    20,
					SinkCall:    "execute",
					Confidence:  0.9,
					Scope:       "local",
				},
			},
		}

		err := generateSARIFOutput(rules, allDetections)
		require.NoError(t, err)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Parse JSON to verify structure
		var sarifReport map[string]interface{}
		err = json.Unmarshal([]byte(output), &sarifReport)
		require.NoError(t, err)

		// Verify SARIF structure
		assert.Equal(t, "2.1.0", sarifReport["version"])
		runs := sarifReport["runs"].([]interface{})
		assert.Len(t, runs, 1)

		run := runs[0].(map[string]interface{})
		tool := run["tool"].(map[string]interface{})
		driver := tool["driver"].(map[string]interface{})
		assert.Equal(t, "Code Pathfinder", driver["name"])

		// Verify rule is included
		rules_array := driver["rules"].([]interface{})
		assert.Len(t, rules_array, 1)
		rule := rules_array[0].(map[string]interface{})
		assert.Equal(t, "sql-injection", rule["id"])
		assert.Equal(t, "SQL Injection", rule["name"])

		// Check description field (could be "fullDescription" or "shortDescription")
		if fullDesc, ok := rule["fullDescription"].(map[string]interface{}); ok {
			assert.Contains(t, fullDesc["text"], "Detects SQL injection vulnerabilities")
		} else if shortDesc, ok := rule["shortDescription"].(map[string]interface{}); ok {
			assert.Contains(t, shortDesc["text"], "Detects SQL injection vulnerabilities")
		}

		// Verify result is included
		results := run["results"].([]interface{})
		assert.Len(t, results, 1)
		result := results[0].(map[string]interface{})
		assert.Equal(t, "sql-injection", result["ruleId"])
		message := result["message"].(map[string]interface{})
		assert.Contains(t, message["text"], "test.vulnerable")
		assert.Contains(t, message["text"], "execute")
		assert.Contains(t, message["text"], "90%")
	})

	t.Run("generates SARIF with multiple rules and detections", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rules := []dsl.RuleIR{
			createTestRule("rule1", "Rule 1", "high", "CWE-1", "", "Rule 1 description"),
			createTestRule("rule2", "Rule 2", "medium", "", "A01:2021", "Rule 2 description"),
		}

		allDetections := map[string][]dsl.DataflowDetection{
			"rule1": {
				{FunctionFQN: "test.func1", SinkLine: 10, Confidence: 0.8, Scope: "local"},
			},
			"rule2": {
				{FunctionFQN: "test.func2", SinkLine: 20, Confidence: 0.7, Scope: "global"},
				{FunctionFQN: "test.func3", SinkLine: 30, Confidence: 0.6, Scope: "local"},
			},
		}

		err := generateSARIFOutput(rules, allDetections)
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		var sarifReport map[string]interface{}
		err = json.Unmarshal([]byte(output), &sarifReport)
		require.NoError(t, err)

		runs := sarifReport["runs"].([]interface{})
		run := runs[0].(map[string]interface{})

		// Verify 2 rules
		rules_array := run["tool"].(map[string]interface{})["driver"].(map[string]interface{})["rules"].([]interface{})
		assert.Len(t, rules_array, 2)

		// Verify 3 results total
		results := run["results"].([]interface{})
		assert.Len(t, results, 3)
	})

	t.Run("generates SARIF with no detections", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rules := []dsl.RuleIR{
			createTestRule("clean-rule", "Clean Rule", "low", "", "", "No issues found"),
		}

		allDetections := map[string][]dsl.DataflowDetection{}

		err := generateSARIFOutput(rules, allDetections)
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		var sarifReport map[string]interface{}
		err = json.Unmarshal([]byte(output), &sarifReport)
		require.NoError(t, err)

		runs := sarifReport["runs"].([]interface{})
		run := runs[0].(map[string]interface{})

		// Verify rule is included
		rules_array := run["tool"].(map[string]interface{})["driver"].(map[string]interface{})["rules"].([]interface{})
		assert.Len(t, rules_array, 1)

		// Verify no results
		results := run["results"].([]interface{})
		assert.Len(t, results, 0)
	})

	t.Run("maps severity levels correctly", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rules := []dsl.RuleIR{
			createTestRule("r1", "R1", "critical", "", "", "D1"),
			createTestRule("r2", "R2", "high", "", "", "D2"),
			createTestRule("r3", "R3", "medium", "", "", "D3"),
			createTestRule("r4", "R4", "low", "", "", "D4"),
		}

		err := generateSARIFOutput(rules, map[string][]dsl.DataflowDetection{})
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		var sarifReport map[string]interface{}
		err = json.Unmarshal([]byte(output), &sarifReport)
		require.NoError(t, err)

		runs := sarifReport["runs"].([]interface{})
		run := runs[0].(map[string]interface{})
		rules_array := run["tool"].(map[string]interface{})["driver"].(map[string]interface{})["rules"].([]interface{})

		// Verify severity mappings
		for _, r := range rules_array {
			rule := r.(map[string]interface{})
			config := rule["defaultConfiguration"].(map[string]interface{})
			level := config["level"].(string)

			switch rule["id"].(string) {
			case "r1", "r2":
				assert.Equal(t, "error", level, "critical/high should map to error")
			case "r3":
				assert.Equal(t, "warning", level, "medium should map to warning")
			case "r4":
				assert.Equal(t, "note", level, "low should map to note")
			}
		}
	})
}

func TestGenerateJSONOutput(t *testing.T) {
	t.Run("generates valid JSON output with detections", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Prevent os.Exit(1) in test
		oldOSExit := osExit
		exitCode := 0
		osExit = func(code int) {
			exitCode = code
		}
		defer func() { osExit = oldOSExit }()

		rules := []dsl.RuleIR{
			createTestRule("xss-vuln", "XSS Vulnerability", "high", "CWE-79", "A03:2021", "Cross-site scripting vulnerability"),
		}

		allDetections := map[string][]dsl.DataflowDetection{
			"xss-vuln": {
				{
					FunctionFQN: "web.render",
					SourceLine:  5,
					SinkLine:    15,
					SinkCall:    "innerHTML",
					TaintedVar:  "user_input",
					Confidence:  0.85,
					Scope:       "local",
				},
			},
		}

		err := generateJSONOutput(rules, allDetections)
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Parse JSON
		var jsonOutput map[string]interface{}
		err = json.Unmarshal([]byte(output), &jsonOutput)
		require.NoError(t, err)

		// Verify structure
		assert.Equal(t, "Code Pathfinder", jsonOutput["tool"])
		assert.Equal(t, Version, jsonOutput["version"])

		results := jsonOutput["results"].([]interface{})
		assert.Len(t, results, 1)

		result := results[0].(map[string]interface{})
		assert.Equal(t, "xss-vuln", result["ruleId"])
		assert.Equal(t, "XSS Vulnerability", result["ruleName"])
		assert.Equal(t, "high", result["severity"])
		assert.Equal(t, "CWE-79", result["cwe"])
		assert.Equal(t, "A03:2021", result["owasp"])
		assert.Equal(t, "Cross-site scripting vulnerability", result["description"])
		assert.Equal(t, "web.render", result["functionFQN"])
		assert.Equal(t, float64(5), result["sourceLine"])
		assert.Equal(t, float64(15), result["sinkLine"])
		assert.Equal(t, "innerHTML", result["sinkCall"])
		assert.Equal(t, "user_input", result["taintedVar"])
		assert.Equal(t, "local", result["scope"])
		assert.Equal(t, 0.85, result["confidence"])

		summary := jsonOutput["summary"].(map[string]interface{})
		assert.Equal(t, float64(1), summary["totalVulnerabilities"])
		assert.Equal(t, float64(1), summary["rulesExecuted"])

		// Verify os.Exit(1) was called
		assert.Equal(t, 1, exitCode)
	})

	t.Run("generates JSON with multiple detections", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		oldOSExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldOSExit }()

		rules := []dsl.RuleIR{
			createTestRule("r1", "R1", "high", "CWE-1", "A01", "D1"),
			createTestRule("r2", "R2", "medium", "CWE-2", "A02", "D2"),
		}

		allDetections := map[string][]dsl.DataflowDetection{
			"r1": {
				{FunctionFQN: "f1", SinkLine: 10, Confidence: 0.9, Scope: "local"},
			},
			"r2": {
				{FunctionFQN: "f2", SinkLine: 20, Confidence: 0.8, Scope: "global"},
				{FunctionFQN: "f3", SinkLine: 30, Confidence: 0.7, Scope: "local"},
			},
		}

		err := generateJSONOutput(rules, allDetections)
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		var jsonOutput map[string]interface{}
		err = json.Unmarshal([]byte(output), &jsonOutput)
		require.NoError(t, err)

		results := jsonOutput["results"].([]interface{})
		assert.Len(t, results, 3)

		summary := jsonOutput["summary"].(map[string]interface{})
		assert.Equal(t, float64(3), summary["totalVulnerabilities"])
		assert.Equal(t, float64(2), summary["rulesExecuted"])
	})

	t.Run("generates JSON with no detections", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		oldOSExit := osExit
		exitCode := 0
		osExit = func(code int) {
			exitCode = code
		}
		defer func() { osExit = oldOSExit }()

		rules := []dsl.RuleIR{
			createTestRule("clean", "Clean", "low", "", "", "No issues"),
		}

		allDetections := map[string][]dsl.DataflowDetection{}

		err := generateJSONOutput(rules, allDetections)
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		var jsonOutput map[string]interface{}
		err = json.Unmarshal([]byte(output), &jsonOutput)
		require.NoError(t, err)

		results := jsonOutput["results"].([]interface{})
		assert.Len(t, results, 0)

		summary := jsonOutput["summary"].(map[string]interface{})
		assert.Equal(t, float64(0), summary["totalVulnerabilities"])

		// Verify os.Exit(1) was NOT called
		assert.Equal(t, 0, exitCode)
	})

	t.Run("handles optional fields correctly", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		oldOSExit := osExit
		osExit = func(code int) {}
		defer func() { osExit = oldOSExit }()

		rules := []dsl.RuleIR{
			createTestRule("minimal", "Minimal", "low", "", "", "Minimal detection"),
		}

		allDetections := map[string][]dsl.DataflowDetection{
			"minimal": {
				{
					FunctionFQN: "func",
					SinkLine:    10,
					Confidence:  0.5,
					Scope:       "local",
					// No SourceLine, SinkCall, or TaintedVar
				},
			},
		}

		err := generateJSONOutput(rules, allDetections)
		require.NoError(t, err)

		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		var jsonOutput map[string]interface{}
		err = json.Unmarshal([]byte(output), &jsonOutput)
		require.NoError(t, err)

		results := jsonOutput["results"].([]interface{})
		result := results[0].(map[string]interface{})

		// Verify optional fields are not present or empty
		_, hasSourceLine := result["sourceLine"]
		assert.False(t, hasSourceLine, "sourceLine should not be present when 0")

		_, hasTaintedVar := result["taintedVar"]
		if hasTaintedVar {
			assert.Equal(t, "", result["taintedVar"])
		}
	})
}
