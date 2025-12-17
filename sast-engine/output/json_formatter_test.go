package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

func TestNewJSONFormatter(t *testing.T) {
	jf := NewJSONFormatter(nil)
	if jf == nil {
		t.Fatal("expected non-nil formatter")
	}
	if jf.options == nil {
		t.Error("expected default options")
	}
}

func TestJSONFormatterStructure(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "user_input",
				SinkCall:   "eval",
				Confidence: 0.9,
				Scope:      "local",
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Location: dsl.LocationInfo{
				RelPath:  "auth/login.py",
				Line:     20,
				Function: "login",
			},
			Rule: dsl.RuleMetadata{
				ID:          "command-injection",
				Name:        "Command Injection",
				Severity:    "critical",
				Description: "User input flows to dangerous function",
				CWE:         []string{"CWE-78"},
				OWASP:       []string{"A1:2017"},
			},
		},
	}

	summary := BuildSummary(detections, 10)
	scanInfo := ScanInfo{
		Target:        "/project/path",
		Version:       "1.2.3-test",
		RulesExecuted: 10,
		Duration:      5 * time.Second,
	}

	err := jf.Format(detections, summary, scanInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse output
	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify tool section
	if output.Tool.Name != "Code Pathfinder" {
		t.Errorf("tool.name: got %q, want %q", output.Tool.Name, "Code Pathfinder")
	}
	if output.Tool.Version != "1.2.3-test" {
		t.Errorf("tool.version: got %q, want %q", output.Tool.Version, "1.2.3-test")
	}
	if output.Tool.URL != "https://github.com/shivasurya/code-pathfinder" {
		t.Errorf("tool.url: got %q", output.Tool.URL)
	}

	// Verify scan section
	if output.Scan.Target != "/project/path" {
		t.Errorf("scan.target: got %q, want %q", output.Scan.Target, "/project/path")
	}
	if output.Scan.RulesExecuted != 10 {
		t.Errorf("scan.rules_executed: got %d, want 10", output.Scan.RulesExecuted)
	}
	if output.Scan.Duration != 5.0 {
		t.Errorf("scan.duration: got %f, want 5.0", output.Scan.Duration)
	}
	if output.Scan.Timestamp == "" {
		t.Error("scan.timestamp should not be empty")
	}

	// Verify results
	if len(output.Results) != 1 {
		t.Fatalf("results: got %d, want 1", len(output.Results))
	}

	result := output.Results[0]
	if result.RuleID != "command-injection" {
		t.Errorf("rule_id: got %q", result.RuleID)
	}
	if result.RuleName != "Command Injection" {
		t.Errorf("rule_name: got %q", result.RuleName)
	}
	if result.Severity != "critical" {
		t.Errorf("severity: got %q", result.Severity)
	}
	if result.Confidence != "high" {
		t.Errorf("confidence: got %q", result.Confidence)
	}
	if result.Message != "User input flows to dangerous function" {
		t.Errorf("message: got %q", result.Message)
	}

	// Verify location
	if result.Location.File != "auth/login.py" {
		t.Errorf("location.file: got %q", result.Location.File)
	}
	if result.Location.Line != 20 {
		t.Errorf("location.line: got %d", result.Location.Line)
	}
	if result.Location.Function != "login" {
		t.Errorf("location.function: got %q", result.Location.Function)
	}

	// Verify detection
	if result.Detection.Type != "taint-local" {
		t.Errorf("detection.type: got %q", result.Detection.Type)
	}
	if result.Detection.Scope != "local" {
		t.Errorf("detection.scope: got %q", result.Detection.Scope)
	}
	if result.Detection.ConfidenceScore != 0.9 {
		t.Errorf("detection.confidence_score: got %f", result.Detection.ConfidenceScore)
	}
	if result.Detection.Source == nil {
		t.Fatal("detection.source should not be nil")
	}
	if result.Detection.Source.Line != 10 {
		t.Errorf("detection.source.line: got %d", result.Detection.Source.Line)
	}
	if result.Detection.Source.Variable != "user_input" {
		t.Errorf("detection.source.variable: got %q", result.Detection.Source.Variable)
	}
	if result.Detection.Sink == nil {
		t.Fatal("detection.sink should not be nil")
	}
	if result.Detection.Sink.Line != 20 {
		t.Errorf("detection.sink.line: got %d", result.Detection.Sink.Line)
	}
	if result.Detection.Sink.Call != "eval" {
		t.Errorf("detection.sink.call: got %q", result.Detection.Sink.Call)
	}

	// Verify metadata
	if len(result.Metadata.CWE) != 1 || result.Metadata.CWE[0] != "CWE-78" {
		t.Errorf("metadata.cwe: got %v", result.Metadata.CWE)
	}
	if len(result.Metadata.OWASP) != 1 || result.Metadata.OWASP[0] != "A1:2017" {
		t.Errorf("metadata.owasp: got %v", result.Metadata.OWASP)
	}

	// Verify summary
	if output.Summary.Total != 1 {
		t.Errorf("summary.total: got %d", output.Summary.Total)
	}
	if output.Summary.BySeverity["critical"] != 1 {
		t.Errorf("summary.by_severity[critical]: got %d", output.Summary.BySeverity["critical"])
	}
	if output.Summary.ByDetectionType["taint-local"] != 1 {
		t.Errorf("summary.by_detection_type[taint-local]: got %d", output.Summary.ByDetectionType["taint-local"])
	}
}

func TestJSONFormatterEmptyResults(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	summary := &Summary{
		BySeverity:      make(map[string]int),
		ByDetectionType: make(map[string]int),
	}
	scanInfo := ScanInfo{Target: "/project"}

	err := jf.Format(nil, summary, scanInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(output.Results) != 0 {
		t.Errorf("expected empty results, got %d", len(output.Results))
	}
	if output.Summary.Total != 0 {
		t.Errorf("summary.total: got %d, want 0", output.Summary.Total)
	}
}

func TestJSONFormatterSnippet(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 5},
			Rule:     dsl.RuleMetadata{ID: "test", Severity: "high"},
			Snippet: dsl.CodeSnippet{
				StartLine: 3,
				Lines: []dsl.SnippetLine{
					{Number: 3, Content: "line 3"},
					{Number: 4, Content: "line 4"},
					{Number: 5, Content: "line 5"},
				},
			},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.8},
		},
	}

	summary := BuildSummary(detections, 1)
	jf.Format(detections, summary, ScanInfo{Target: "/test"})

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	snippet := output.Results[0].Location.Snippet
	if snippet == nil {
		t.Fatal("expected snippet")
	}
	if snippet.StartLine != 3 {
		t.Errorf("snippet.start_line: got %d", snippet.StartLine)
	}
	if snippet.EndLine != 5 {
		t.Errorf("snippet.end_line: got %d, want 5", snippet.EndLine)
	}
	if len(snippet.Lines) != 3 {
		t.Errorf("snippet.lines: got %d", len(snippet.Lines))
	}
	if snippet.Lines[0] != "line 3" {
		t.Errorf("snippet.lines[0]: got %q", snippet.Lines[0])
	}
}

func TestJSONFormatterMetadata(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1},
			Rule: dsl.RuleMetadata{
				ID:         "test",
				Severity:   "high",
				CWE:        []string{"CWE-78", "CWE-79"},
				OWASP:      []string{"A1:2017"},
				References: []string{"https://example.com"},
			},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.7},
		},
	}

	summary := BuildSummary(detections, 1)
	jf.Format(detections, summary, ScanInfo{Target: "/test"})

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	meta := output.Results[0].Metadata
	if len(meta.CWE) != 2 {
		t.Errorf("metadata.cwe: got %d", len(meta.CWE))
	}
	if meta.CWE[0] != "CWE-78" {
		t.Errorf("metadata.cwe[0]: got %q", meta.CWE[0])
	}
	if len(meta.References) != 1 {
		t.Errorf("metadata.references: got %d", len(meta.References))
	}
}

func TestJSONFormatterPatternDetection(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				Confidence: 0.85,
			},
			DetectionType: dsl.DetectionTypePattern,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 10},
			Rule:          dsl.RuleMetadata{ID: "pattern-rule", Severity: "medium"},
		},
	}

	summary := BuildSummary(detections, 1)
	jf.Format(detections, summary, ScanInfo{Target: "/test"})

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	detection := output.Results[0].Detection
	if detection.Type != "pattern" {
		t.Errorf("detection.type: got %q", detection.Type)
	}
	// Pattern detections should not have source/sink
	if detection.Source != nil {
		t.Error("pattern detection should not have source")
	}
	if detection.Sink != nil {
		t.Error("pattern detection should not have sink")
	}
	if detection.Scope != "" {
		t.Error("pattern detection should not have scope")
	}
}

func TestJSONFormatterGlobalTaintDetection(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 5,
				SinkLine:   15,
				TaintedVar: "global_var",
				SinkCall:   "dangerous_func",
				Confidence: 0.95,
				Scope:      "global",
			},
			DetectionType: dsl.DetectionTypeTaintGlobal,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 15},
			Rule:          dsl.RuleMetadata{ID: "global-taint", Severity: "critical"},
		},
	}

	summary := BuildSummary(detections, 1)
	jf.Format(detections, summary, ScanInfo{Target: "/test"})

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	detection := output.Results[0].Detection
	if detection.Type != "taint-global" {
		t.Errorf("detection.type: got %q", detection.Type)
	}
	if detection.Scope != "global" {
		t.Errorf("detection.scope: got %q", detection.Scope)
	}
	if detection.Source == nil || detection.Sink == nil {
		t.Fatal("global taint should have source and sink")
	}
}

func TestJSONFormatterFallbackFilePath(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{
				FilePath: "/absolute/path/test.py",
				RelPath:  "", // Empty relpath
				Line:     10,
			},
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "low"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.6},
		},
	}

	summary := BuildSummary(detections, 1)
	jf.Format(detections, summary, ScanInfo{Target: "/test"})

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	if output.Results[0].Location.File != "/absolute/path/test.py" {
		t.Errorf("location.file: got %q", output.Results[0].Location.File)
	}
}

func TestJSONFormatterWithErrors(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	summary := &Summary{
		BySeverity:      make(map[string]int),
		ByDetectionType: make(map[string]int),
	}
	scanInfo := ScanInfo{
		Target: "/test",
		Errors: []string{"error 1", "error 2"},
	}

	jf.Format(nil, summary, scanInfo)

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	if len(output.Errors) != 2 {
		t.Errorf("errors: got %d, want 2", len(output.Errors))
	}
	if output.Errors[0] != "error 1" {
		t.Errorf("errors[0]: got %q", output.Errors[0])
	}
}

func TestJSONFormatterOmitsOptionalFields(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{
				RelPath: "test.py",
				Line:    10,
				// Column, Function omitted
			},
			Rule: dsl.RuleMetadata{
				ID:       "test",
				Severity: "low",
				// CWE, OWASP, References omitted
			},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.5},
		},
	}

	summary := BuildSummary(detections, 1)
	jf.Format(detections, summary, ScanInfo{Target: "/test"})

	// Check that optional fields are omitted from JSON
	jsonStr := buf.String()
	if bytes.Contains(buf.Bytes(), []byte(`"column"`)) {
		t.Error("column should be omitted when 0")
	}
	if bytes.Contains(buf.Bytes(), []byte(`"cwe"`)) && bytes.Contains(buf.Bytes(), []byte(`[]`)) {
		// CWE should either be omitted or empty array based on omitempty behavior
	}
	_ = jsonStr // Use variable to avoid unused warning
}

func TestJSONFormatterMultipleDetections(t *testing.T) {
	var buf bytes.Buffer
	jf := NewJSONFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location:      dsl.LocationInfo{RelPath: "file1.py", Line: 10},
			Rule:          dsl.RuleMetadata{ID: "rule1", Severity: "critical"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
		},
		{
			Location:      dsl.LocationInfo{RelPath: "file2.py", Line: 20},
			Rule:          dsl.RuleMetadata{ID: "rule2", Severity: "high"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.8},
		},
		{
			Location:      dsl.LocationInfo{RelPath: "file3.py", Line: 30},
			Rule:          dsl.RuleMetadata{ID: "rule3", Severity: "medium"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.7},
		},
	}

	summary := BuildSummary(detections, 3)
	jf.Format(detections, summary, ScanInfo{Target: "/test", RulesExecuted: 3})

	var output JSONOutput
	json.Unmarshal(buf.Bytes(), &output)

	if len(output.Results) != 3 {
		t.Errorf("results: got %d, want 3", len(output.Results))
	}
	if output.Summary.Total != 3 {
		t.Errorf("summary.total: got %d, want 3", output.Summary.Total)
	}
}
