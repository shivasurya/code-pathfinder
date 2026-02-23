package output

import (
	"bytes"
	"encoding/csv"
	"slices"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

func TestNewCSVFormatter(t *testing.T) {
	cf := NewCSVFormatter(nil)
	if cf == nil {
		t.Fatal("expected non-nil formatter")
	}
	if cf.options == nil {
		t.Error("expected default options")
	}
}

func TestCSVHeaders(t *testing.T) {
	headers := CSVHeaders()
	if len(headers) != 17 {
		t.Errorf("expected 17 headers, got %d", len(headers))
	}

	// Verify key headers exist
	expectedHeaders := []string{"severity", "rule_id", "file", "line", "detection_type"}
	for _, expected := range expectedHeaders {
		found := slices.Contains(headers, expected)
		if !found {
			t.Errorf("missing expected header: %s", expected)
		}
	}

	// Verify exact header names and order
	if headers[0] != "severity" {
		t.Errorf("headers[0]: got %q, want 'severity'", headers[0])
	}
	if headers[1] != "confidence" {
		t.Errorf("headers[1]: got %q, want 'confidence'", headers[1])
	}
	if headers[2] != "rule_id" {
		t.Errorf("headers[2]: got %q, want 'rule_id'", headers[2])
	}
}

func TestCSVFormatterOutput(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

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
				Column:   8,
				Function: "login",
			},
			Rule: dsl.RuleMetadata{
				ID:          "command-injection",
				Name:        "Command Injection",
				Severity:    "critical",
				Description: "User input flows to eval",
				CWE:         []string{"CWE-78"},
				OWASP:       []string{"A1:2017"},
			},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse CSV
	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	// Header + 1 row
	if len(records) != 2 {
		t.Fatalf("expected 2 rows (header + data), got %d", len(records))
	}

	// Verify header
	if records[0][0] != "severity" {
		t.Errorf("first header should be 'severity', got %q", records[0][0])
	}

	// Verify data row
	row := records[1]
	if row[0] != "critical" {
		t.Errorf("severity: got %q", row[0])
	}
	if row[1] != "high" {
		t.Errorf("confidence: got %q", row[1])
	}
	if row[2] != "command-injection" {
		t.Errorf("rule_id: got %q", row[2])
	}
	if row[3] != "Command Injection" {
		t.Errorf("rule_name: got %q", row[3])
	}
	if row[4] != "CWE-78" {
		t.Errorf("cwe: got %q", row[4])
	}
	if row[5] != "A1:2017" {
		t.Errorf("owasp: got %q", row[5])
	}
	if row[6] != "auth/login.py" {
		t.Errorf("file: got %q", row[6])
	}
	if row[7] != "20" {
		t.Errorf("line: got %q", row[7])
	}
	if row[8] != "8" {
		t.Errorf("column: got %q", row[8])
	}
	if row[9] != "login" {
		t.Errorf("function: got %q", row[9])
	}
	if row[10] != "User input flows to eval" {
		t.Errorf("message: got %q", row[10])
	}
	if row[11] != "taint-local" {
		t.Errorf("detection_type: got %q", row[11])
	}
	if row[12] != "local" {
		t.Errorf("detection_scope: got %q", row[12])
	}
	if row[13] != "10" {
		t.Errorf("source_line: got %q", row[13])
	}
	if row[14] != "20" {
		t.Errorf("sink_line: got %q", row[14])
	}
	if row[15] != "user_input" {
		t.Errorf("tainted_var: got %q", row[15])
	}
	if row[16] != "eval" {
		t.Errorf("sink_call: got %q", row[16])
	}
}

func TestCSVFormatterEscaping(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1},
			Rule: dsl.RuleMetadata{
				ID:          "test",
				Name:        `Rule with "quotes"`,
				Severity:    "high",
				Description: `Description with "quotes" and, commas`,
			},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.7},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse should succeed despite special characters
	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV parsing failed: %v", err)
	}

	// Verify description was properly escaped/quoted
	row := records[1]
	if row[10] != `Description with "quotes" and, commas` {
		t.Errorf("description not properly escaped: %q", row[10])
	}
	if row[3] != `Rule with "quotes"` {
		t.Errorf("rule_name not properly escaped: %q", row[3])
	}
}

func TestCSVFormatterEmptyResults(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	err := cf.Format(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	// Should have header only
	if len(records) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(records))
	}
}

func TestCSVFormatterMultipleRows(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location:      dsl.LocationInfo{RelPath: "file1.py", Line: 10},
			Rule:          dsl.RuleMetadata{ID: "rule1", Severity: "high"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
		},
		{
			Location:      dsl.LocationInfo{RelPath: "file2.py", Line: 20},
			Rule:          dsl.RuleMetadata{ID: "rule2", Severity: "medium"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.7},
		},
		{
			Location:      dsl.LocationInfo{RelPath: "file3.py", Line: 30},
			Rule:          dsl.RuleMetadata{ID: "rule3", Severity: "low"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.5},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	// Header + 3 rows
	if len(records) != 4 {
		t.Errorf("expected 4 rows, got %d", len(records))
	}

	// Verify each data row has 17 columns
	for i := 1; i < len(records); i++ {
		if len(records[i]) != 17 {
			t.Errorf("row %d: expected 17 columns, got %d", i, len(records[i]))
		}
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, ""},
		{1, "1"},
		{42, "42"},
		{-1, "-1"},
		{999, "999"},
	}

	for _, tt := range tests {
		got := intToString(tt.input)
		if got != tt.expected {
			t.Errorf("intToString(%d): got %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCSVFormatterZeroValues(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{
				RelPath: "test.py",
				Line:    10,
				// Column: 0 (should be empty string)
			},
			Rule: dsl.RuleMetadata{
				ID:       "test",
				Severity: "low",
				// CWE, OWASP empty
			},
			Detection: dsl.DataflowDetection{
				// SourceLine, SinkLine: 0 (should be empty)
				Confidence: 0.6,
			},
			DetectionType: dsl.DetectionTypePattern,
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	row := records[1]
	if row[8] != "" { // column
		t.Errorf("column should be empty for 0, got %q", row[8])
	}
	if row[13] != "" { // source_line
		t.Errorf("source_line should be empty for 0, got %q", row[13])
	}
	if row[14] != "" { // sink_line
		t.Errorf("sink_line should be empty for 0, got %q", row[14])
	}
}

func TestCSVFormatterFallbackFilePath(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{
				FilePath: "/absolute/path/test.py",
				RelPath:  "", // Empty relpath
				Line:     10,
			},
			Rule:          dsl.RuleMetadata{ID: "test", Severity: "low"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.5},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	row := records[1]
	if row[6] != "/absolute/path/test.py" {
		t.Errorf("file: got %q", row[6])
	}
}

func TestCSVFormatterTaintDetection(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 5,
				SinkLine:   15,
				TaintedVar: "tainted_data",
				SinkCall:   "dangerous_sink",
				Confidence: 0.95,
				Scope:      "global",
			},
			DetectionType: dsl.DetectionTypeTaintGlobal,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 15},
			Rule:          dsl.RuleMetadata{ID: "taint-rule", Severity: "critical"},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	row := records[1]
	if row[11] != "taint-global" {
		t.Errorf("detection_type: got %q", row[11])
	}
	if row[12] != "global" {
		t.Errorf("detection_scope: got %q", row[12])
	}
	if row[13] != "5" {
		t.Errorf("source_line: got %q", row[13])
	}
	if row[14] != "15" {
		t.Errorf("sink_line: got %q", row[14])
	}
	if row[15] != "tainted_data" {
		t.Errorf("tainted_var: got %q", row[15])
	}
	if row[16] != "dangerous_sink" {
		t.Errorf("sink_call: got %q", row[16])
	}
}

func TestCSVFormatterMultipleCWEOWASP(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1},
			Rule: dsl.RuleMetadata{
				ID:       "test",
				Severity: "high",
				CWE:      []string{"CWE-78", "CWE-79", "CWE-89"},
				OWASP:    []string{"A1:2017", "A3:2021"},
			},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.8},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	row := records[1]
	// Should only include first CWE and OWASP
	if row[4] != "CWE-78" {
		t.Errorf("cwe: got %q, want CWE-78", row[4])
	}
	if row[5] != "A1:2017" {
		t.Errorf("owasp: got %q, want A1:2017", row[5])
	}
}

func TestCSVFormatterConfidenceLevels(t *testing.T) {
	var buf bytes.Buffer
	cf := NewCSVFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location:      dsl.LocationInfo{RelPath: "test1.py", Line: 1},
			Rule:          dsl.RuleMetadata{ID: "test1", Severity: "high"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.9},
		},
		{
			Location:      dsl.LocationInfo{RelPath: "test2.py", Line: 2},
			Rule:          dsl.RuleMetadata{ID: "test2", Severity: "medium"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.6},
		},
		{
			Location:      dsl.LocationInfo{RelPath: "test3.py", Line: 3},
			Rule:          dsl.RuleMetadata{ID: "test3", Severity: "low"},
			DetectionType: dsl.DetectionTypePattern,
			Detection:     dsl.DataflowDetection{Confidence: 0.3},
		},
	}

	err := cf.Format(detections)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV: %v", err)
	}

	if records[1][1] != "high" {
		t.Errorf("confidence for 0.9: got %q, want 'high'", records[1][1])
	}
	if records[2][1] != "medium" {
		t.Errorf("confidence for 0.6: got %q, want 'medium'", records[2][1])
	}
	if records[3][1] != "low" {
		t.Errorf("confidence for 0.3: got %q, want 'low'", records[3][1])
	}
}
