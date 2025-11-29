package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// Helper function to create test rules (duplicated from ci_test.go).
func createTestRuleScan(id, name, severity, cwe, owasp, description string) dsl.RuleIR {
	rule := dsl.RuleIR{}
	rule.Rule.ID = id
	rule.Rule.Name = name
	rule.Rule.Severity = severity
	rule.Rule.CWE = cwe
	rule.Rule.OWASP = owasp
	rule.Rule.Description = description
	return rule
}

func TestCountTotalCallSites(t *testing.T) {
	t.Run("counts call sites across all functions", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["func1"] = []core.CallSite{
			{Target: "foo", Location: core.Location{Line: 10}},
			{Target: "bar", Location: core.Location{Line: 20}},
		}
		cg.CallSites["func2"] = []core.CallSite{
			{Target: "baz", Location: core.Location{Line: 30}},
		}

		total := countTotalCallSites(cg)
		assert.Equal(t, 3, total)
	})

	t.Run("returns zero for empty callgraph", func(t *testing.T) {
		cg := core.NewCallGraph()
		total := countTotalCallSites(cg)
		assert.Equal(t, 0, total)
	})

	t.Run("handles function with no call sites", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["func1"] = []core.CallSite{}
		total := countTotalCallSites(cg)
		assert.Equal(t, 0, total)
	})
}

func TestPrintDetections(t *testing.T) {
	t.Run("prints detections with all fields", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rule := createTestRuleScan("test-rule", "Test Rule", "high", "CWE-89", "A03:2021", "Test SQL injection detection")

		detections := []dsl.DataflowDetection{
			{
				FunctionFQN: "test.vulnerable_func",
				SourceLine:  10,
				SinkLine:    20,
				SinkCall:    "execute",
				TaintedVar:  "user_input",
				Confidence:  0.9,
				Scope:       "local",
			},
		}

		printDetections(rule, detections)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify output contains expected information
		assert.Contains(t, output, "[high] test-rule (Test Rule)")
		assert.Contains(t, output, "CWE: CWE-89")
		assert.Contains(t, output, "OWASP: A03:2021")
		assert.Contains(t, output, "Test SQL injection detection")
		assert.Contains(t, output, "test.vulnerable_func:20")
		assert.Contains(t, output, "Source: line 10")
		assert.Contains(t, output, "Sink: execute (line 20)")
		assert.Contains(t, output, "Tainted variable: user_input")
		assert.Contains(t, output, "Confidence: 90%")
		assert.Contains(t, output, "Scope: local")
	})

	t.Run("prints detections without optional fields", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rule := createTestRuleScan("simple-rule", "Simple Rule", "medium", "", "", "Simple detection")

		detections := []dsl.DataflowDetection{
			{
				FunctionFQN: "test.func",
				SinkLine:    15,
				Confidence:  0.5,
				Scope:       "global",
			},
		}

		printDetections(rule, detections)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify output
		assert.Contains(t, output, "[medium] simple-rule (Simple Rule)")
		assert.Contains(t, output, "test.func:15")
		assert.Contains(t, output, "Confidence: 50%")
		assert.Contains(t, output, "Scope: global")
		// Should not contain optional fields
		assert.NotContains(t, output, "Source: line 0")
		assert.NotContains(t, output, "Sink: ")
		assert.NotContains(t, output, "Tainted variable: ")
	})

	t.Run("prints multiple detections", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rule := createTestRuleScan("multi-rule", "Multi Rule", "critical", "CWE-79", "A03:2021", "XSS detection")

		detections := []dsl.DataflowDetection{
			{
				FunctionFQN: "test.func1",
				SinkLine:    10,
				Confidence:  0.8,
				Scope:       "local",
			},
			{
				FunctionFQN: "test.func2",
				SinkLine:    20,
				Confidence:  0.7,
				Scope:       "local",
			},
		}

		printDetections(rule, detections)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify both detections are printed
		assert.Contains(t, output, "test.func1:10")
		assert.Contains(t, output, "test.func2:20")
		assert.Contains(t, output, "Confidence: 80%")
		assert.Contains(t, output, "Confidence: 70%")
	})
}
