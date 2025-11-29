package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

func TestNewTextFormatter(t *testing.T) {
	tf := NewTextFormatter(nil, nil)
	if tf == nil {
		t.Fatal("expected non-nil formatter")
	}
	if tf.options == nil {
		t.Error("expected default options")
	}
}

func TestTextFormatterNoFindings(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	err := tf.Format(nil, &Summary{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No security issues found") {
		t.Errorf("expected 'No security issues found', got: %s", output)
	}
}

func TestTextFormatterWithFindings(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SinkLine:   10,
				TaintedVar: "user_input",
				SinkCall:   "eval",
				Confidence: 0.9,
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Location: dsl.LocationInfo{
				RelPath: "auth/login.py",
				Line:    10,
			},
			Rule: dsl.RuleMetadata{
				ID:       "command-injection",
				Name:     "Command Injection",
				Severity: "critical",
				CWE:      []string{"CWE-78"},
			},
		},
	}

	summary := BuildSummary(detections, 5)
	err := tf.Format(detections, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "Code Pathfinder Security Scan") {
		t.Error("missing header")
	}

	// Check severity section
	if !strings.Contains(output, "Critical Issues (1)") {
		t.Error("missing critical issues section")
	}

	// Check detection badge
	if !strings.Contains(output, "[Taint-Local]") {
		t.Error("missing detection badge")
	}

	// Check rule info
	if !strings.Contains(output, "command-injection") {
		t.Error("missing rule ID")
	}

	// Check CWE
	if !strings.Contains(output, "CWE-78") {
		t.Error("missing CWE")
	}

	// Check location
	if !strings.Contains(output, "auth/login.py:10") {
		t.Error("missing location")
	}

	// Check summary
	if !strings.Contains(output, "1 findings across 5 rules") {
		t.Error("missing summary")
	}
}

func TestTextFormatterSeverityGrouping(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule:     dsl.RuleMetadata{ID: "low1", Severity: "low"},
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1},
		},
		{
			Rule:     dsl.RuleMetadata{ID: "crit1", Severity: "critical"},
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 2},
		},
		{
			Rule:     dsl.RuleMetadata{ID: "high1", Severity: "high"},
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 3},
		},
	}

	summary := BuildSummary(detections, 3)
	tf.Format(detections, summary)

	output := buf.String()

	// Critical should come before high, high before low
	critIdx := strings.Index(output, "Critical Issues")
	highIdx := strings.Index(output, "High Issues")
	lowIdx := strings.Index(output, "Low Issues")

	if critIdx == -1 || highIdx == -1 || lowIdx == -1 {
		t.Fatal("missing severity sections")
	}

	if critIdx > highIdx {
		t.Error("critical should come before high")
	}
	if highIdx > lowIdx {
		t.Error("high should come before low")
	}
}

func TestTextFormatterDetailedVsAbbreviated(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule:          dsl.RuleMetadata{ID: "crit1", Severity: "critical", Name: "Critical Bug"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 10},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
		},
		{
			Rule:          dsl.RuleMetadata{ID: "low1", Severity: "low", Name: "Low Bug"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 20},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.5},
		},
	}

	summary := BuildSummary(detections, 2)
	tf.Format(detections, summary)

	output := buf.String()

	// Critical should have detailed output (Confidence line)
	if !strings.Contains(output, "Confidence:") {
		t.Error("critical finding should have confidence line")
	}

	// Low should be abbreviated (single line)
	lines := strings.Split(output, "\n")
	lowLineCount := 0
	inLowSection := false
	for _, line := range lines {
		if strings.Contains(line, "Low Issues") {
			inLowSection = true
			continue
		}
		if inLowSection && strings.HasPrefix(strings.TrimSpace(line), "[low]") {
			lowLineCount++
		}
	}
	if lowLineCount != 1 {
		t.Errorf("expected 1 abbreviated low finding line, got %d", lowLineCount)
	}
}

func TestTextFormatterCodeSnippet(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "critical"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 5},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
			Snippet: dsl.CodeSnippet{
				StartLine:     3,
				HighlightLine: 5,
				Lines: []dsl.SnippetLine{
					{Number: 3, Content: "def foo():", IsHighlight: false},
					{Number: 4, Content: "    x = input()", IsHighlight: false},
					{Number: 5, Content: "    eval(x)", IsHighlight: true},
					{Number: 6, Content: "    return", IsHighlight: false},
				},
			},
		},
	}

	summary := BuildSummary(detections, 1)
	tf.Format(detections, summary)

	output := buf.String()

	// Check snippet lines present
	if !strings.Contains(output, "def foo():") {
		t.Error("missing snippet line 3")
	}
	if !strings.Contains(output, "eval(x)") {
		t.Error("missing snippet line 5")
	}

	// Check highlight marker
	if !strings.Contains(output, "> ") {
		t.Error("missing highlight marker")
	}
}

func TestTextFormatterTaintFlow(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "user_input",
				SinkCall:   "os.system",
				Confidence: 0.9,
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Rule:          dsl.RuleMetadata{ID: "cmd-inj", Severity: "critical"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 20},
		},
	}

	summary := BuildSummary(detections, 1)
	tf.Format(detections, summary)

	output := buf.String()

	// Check flow line
	if !strings.Contains(output, "Flow: user_input (line 10) -> os.system (line 20)") {
		t.Error("missing taint flow line")
	}

	// Check tainted variable message
	if !strings.Contains(output, "Tainted variable 'user_input'") {
		t.Error("missing tainted variable message")
	}
}

func TestFormatLocation(t *testing.T) {
	tf := NewTextFormatter(nil, nil)

	tests := []struct {
		name     string
		loc      dsl.LocationInfo
		expected string
	}{
		{
			"relative path with line",
			dsl.LocationInfo{RelPath: "auth/login.py", Line: 42},
			"auth/login.py:42",
		},
		{
			"absolute path fallback",
			dsl.LocationInfo{FilePath: "/full/path/file.py", Line: 10},
			"/full/path/file.py:10",
		},
		{
			"function only",
			dsl.LocationInfo{Function: "my_function"},
			"my_function",
		},
		{
			"no line number",
			dsl.LocationInfo{RelPath: "test.py"},
			"test.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tf.formatLocation(tt.loc)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildSummary(t *testing.T) {
	detections := []*dsl.EnrichedDetection{
		{Rule: dsl.RuleMetadata{Severity: "critical"}, DetectionType: dsl.DetectionTypeTaintLocal},
		{Rule: dsl.RuleMetadata{Severity: "critical"}, DetectionType: dsl.DetectionTypeTaintLocal},
		{Rule: dsl.RuleMetadata{Severity: "high"}, DetectionType: dsl.DetectionTypePattern},
		{Rule: dsl.RuleMetadata{Severity: "low"}, DetectionType: dsl.DetectionTypePattern},
	}

	summary := BuildSummary(detections, 10)

	if summary.TotalFindings != 4 {
		t.Errorf("TotalFindings: got %d, want 4", summary.TotalFindings)
	}
	if summary.RulesExecuted != 10 {
		t.Errorf("RulesExecuted: got %d, want 10", summary.RulesExecuted)
	}
	if summary.BySeverity["critical"] != 2 {
		t.Errorf("critical count: got %d, want 2", summary.BySeverity["critical"])
	}
	if summary.BySeverity["high"] != 1 {
		t.Errorf("high count: got %d, want 1", summary.BySeverity["high"])
	}
	if summary.ByDetectionType["taint-local"] != 2 {
		t.Errorf("taint-local count: got %d, want 2", summary.ByDetectionType["taint-local"])
	}
}

func TestTextFormatterStatistics(t *testing.T) {
	var buf bytes.Buffer
	opts := &OutputOptions{Verbosity: VerbosityVerbose}
	tf := NewTextFormatterWithWriter(&buf, opts, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "high"},
			DetectionType: dsl.DetectionTypePattern,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 1},
		},
	}

	summary := BuildSummary(detections, 5)
	tf.Format(detections, summary)

	output := buf.String()

	// Verbose should show detection methods
	if !strings.Contains(output, "Detection Methods:") {
		t.Error("verbose mode should show detection methods")
	}
}

func TestTextFormatterDetectionBadges(t *testing.T) {
	tests := []struct {
		detType  dsl.DetectionType
		expected string
	}{
		{dsl.DetectionTypePattern, "[Pattern]"},
		{dsl.DetectionTypeTaintLocal, "[Taint-Local]"},
		{dsl.DetectionTypeTaintGlobal, "[Taint-Global]"},
	}

	for _, tt := range tests {
		det := &dsl.EnrichedDetection{DetectionType: tt.detType}
		got := det.DetectionBadge()
		if got != tt.expected {
			t.Errorf("type %v: got %q, want %q", tt.detType, got, tt.expected)
		}
	}
}

func TestFormatDetectionMethod(t *testing.T) {
	tf := NewTextFormatter(nil, nil)

	tests := []struct {
		detType  dsl.DetectionType
		expected string
	}{
		{dsl.DetectionTypePattern, "Pattern matching"},
		{dsl.DetectionTypeTaintLocal, "Intra-procedural taint analysis"},
		{dsl.DetectionTypeTaintGlobal, "Inter-procedural taint analysis"},
		{"unknown", "Unknown"},
	}

	for _, tt := range tests {
		got := tf.formatDetectionMethod(tt.detType)
		if got != tt.expected {
			t.Errorf("type %v: got %q, want %q", tt.detType, got, tt.expected)
		}
	}
}

func TestTextFormatterMultipleCWEAndOWASP(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule: dsl.RuleMetadata{
				ID:       "test",
				Severity: "critical",
				CWE:      []string{"CWE-78", "CWE-77"},
				OWASP:    []string{"A03:2021", "A01:2021"},
			},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 10},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
		},
	}

	summary := BuildSummary(detections, 1)
	tf.Format(detections, summary)

	output := buf.String()

	// Should display first CWE and OWASP
	if !strings.Contains(output, "CWE-78") {
		t.Error("missing CWE-78")
	}
	if !strings.Contains(output, "A03:2021") {
		t.Error("missing OWASP A03:2021")
	}
}

func TestTextFormatterEmptySnippet(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "critical", Name: "Test"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 10},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
			Snippet:       dsl.CodeSnippet{Lines: []dsl.SnippetLine{}}, // Empty snippet
		},
	}

	summary := BuildSummary(detections, 1)
	err := tf.Format(detections, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not crash with empty snippet
	output := buf.String()
	if !strings.Contains(output, "test.py:10") {
		t.Error("missing location despite empty snippet")
	}
}

func TestTextFormatterTaintFlowWithoutTaintedVar(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "", // Empty tainted var
				SinkCall:   "os.system",
				Confidence: 0.9,
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "critical"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 20},
		},
	}

	summary := BuildSummary(detections, 1)
	tf.Format(detections, summary)

	output := buf.String()

	// Should not display flow line when TaintedVar is empty
	if strings.Contains(output, "Flow:") {
		t.Error("should not display flow when TaintedVar is empty")
	}
}

func TestGroupBySeverity(t *testing.T) {
	tf := NewTextFormatter(nil, nil)

	detections := []*dsl.EnrichedDetection{
		{Rule: dsl.RuleMetadata{Severity: "critical"}},
		{Rule: dsl.RuleMetadata{Severity: "critical"}},
		{Rule: dsl.RuleMetadata{Severity: "high"}},
		{Rule: dsl.RuleMetadata{Severity: "low"}},
		{Rule: dsl.RuleMetadata{Severity: "low"}},
		{Rule: dsl.RuleMetadata{Severity: "low"}},
	}

	grouped := tf.groupBySeverity(detections)

	if len(grouped["critical"]) != 2 {
		t.Errorf("critical: got %d, want 2", len(grouped["critical"]))
	}
	if len(grouped["high"]) != 1 {
		t.Errorf("high: got %d, want 1", len(grouped["high"]))
	}
	if len(grouped["low"]) != 3 {
		t.Errorf("low: got %d, want 3", len(grouped["low"]))
	}
}

func TestWriteSnippetAlignment(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	// Test with varying line number widths (9, 10, 99, 100)
	detections := []*dsl.EnrichedDetection{
		{
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "critical"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 100},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
			Snippet: dsl.CodeSnippet{
				StartLine:     98,
				HighlightLine: 100,
				Lines: []dsl.SnippetLine{
					{Number: 98, Content: "line 98", IsHighlight: false},
					{Number: 99, Content: "line 99", IsHighlight: false},
					{Number: 100, Content: "line 100", IsHighlight: true},
				},
			},
		},
	}

	summary := BuildSummary(detections, 1)
	tf.Format(detections, summary)

	output := buf.String()

	// Check alignment - line numbers should be right-aligned
	if !strings.Contains(output, " 98 |") {
		t.Error("missing aligned line 98")
	}
	if !strings.Contains(output, " 99 |") {
		t.Error("missing aligned line 99")
	}
	if !strings.Contains(output, "100 |") {
		t.Error("missing aligned line 100")
	}
}

func TestTextFormatterEmptySummary(t *testing.T) {
	var buf bytes.Buffer
	tf := NewTextFormatterWithWriter(&buf, nil, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "high"},
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 1},
			DetectionType: dsl.DetectionTypePattern,
		},
	}

	summary := &Summary{
		TotalFindings:   1,
		RulesExecuted:   1,
		BySeverity:      map[string]int{},
		ByDetectionType: map[string]int{},
	}

	err := tf.Format(detections, summary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1 findings across 1 rules") {
		t.Error("missing summary line")
	}
}
