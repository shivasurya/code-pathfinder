package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSARIFFormatter(t *testing.T) {
	sf := NewSARIFFormatter(nil)
	assert.NotNil(t, sf)
	assert.NotNil(t, sf.writer)
	assert.NotNil(t, sf.options)
}

func TestSARIFFormatterVersion(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1, Column: 1},
			Rule:     dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test rule"},
		},
	}

	err := sf.Format(detections, ScanInfo{Target: "/project"})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	assert.Equal(t, "2.1.0", report["version"])
}

func TestSARIFFormatterTool(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1, Column: 1},
			Rule:     dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test rule"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	require.Len(t, runs, 1)

	run := runs[0].(map[string]any)
	tool := run["tool"].(map[string]any)
	driver := tool["driver"].(map[string]any)
	assert.Equal(t, "Code Pathfinder", driver["name"])
}

func TestSARIFFormatterRules(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1, Column: 1},
			Rule: dsl.RuleMetadata{
				ID:          "command-injection",
				Name:        "Command Injection",
				Severity:    "critical",
				Description: "User input flows to shell command",
				CWE:         []string{"CWE-78"},
			},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	tool := run["tool"].(map[string]any)
	driver := tool["driver"].(map[string]any)
	rules := driver["rules"].([]any)
	require.Len(t, rules, 1)

	rule := rules[0].(map[string]any)
	assert.Equal(t, "command-injection", rule["id"])
	assert.Equal(t, "Command Injection", rule["name"])

	// Check description (could be in fullDescription or shortDescription)
	if fullDesc, ok := rule["fullDescription"].(map[string]any); ok {
		assert.Contains(t, fullDesc["text"], "User input flows to shell command")
		assert.Contains(t, fullDesc["text"], "CWE-78")
	} else if shortDesc, ok := rule["shortDescription"].(map[string]any); ok {
		assert.Contains(t, shortDesc["text"], "User input flows to shell command")
		assert.Contains(t, shortDesc["text"], "CWE-78")
	} else {
		t.Fatal("No description found in rule")
	}
}

func TestSARIFFormatterRuleProperties(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test.py", Line: 1, Column: 1},
			Rule: dsl.RuleMetadata{
				ID:          "test",
				Name:        "Test",
				Severity:    "critical",
				Description: "Test rule",
			},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	tool := run["tool"].(map[string]any)
	driver := tool["driver"].(map[string]any)
	rules := driver["rules"].([]any)
	rule := rules[0].(map[string]any)

	props := rule["properties"].(map[string]any)
	assert.Equal(t, "9.0", props["security-severity"])
	assert.Equal(t, "high", props["precision"])
	assert.Contains(t, props["tags"], "security")
}

func TestSARIFFormatterResults(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "user_input",
				SinkCall:   "os.system",
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Location: dsl.LocationInfo{
				RelPath: "auth/login.py",
				Line:    20,
				Column:  8,
			},
			Rule: dsl.RuleMetadata{
				ID:          "cmd-inj",
				Name:        "Command Injection",
				Severity:    "critical",
				Description: "Command injection vulnerability",
			},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	require.Len(t, results, 1)

	result := results[0].(map[string]any)
	assert.Equal(t, "cmd-inj", result["ruleId"])
	// Level may be optional in result, it's defined in rule configuration
	if level, ok := result["level"]; ok {
		assert.Equal(t, "error", level)
	}

	// Check location
	locations := result["locations"].([]any)
	require.Len(t, locations, 1)
	loc := locations[0].(map[string]any)
	physLoc := loc["physicalLocation"].(map[string]any)
	artifact := physLoc["artifactLocation"].(map[string]any)
	assert.Equal(t, "auth/login.py", artifact["uri"])

	region := physLoc["region"].(map[string]any)
	assert.Equal(t, float64(20), region["startLine"])
	assert.Equal(t, float64(8), region["startColumn"])
}

func TestSARIFFormatterCodeFlows(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "user_input",
				SinkCall:   "eval",
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 20, Column: 1},
			Rule:          dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	result := results[0].(map[string]any)

	// Check code flows exist for taint detection
	codeFlows := result["codeFlows"].([]any)
	require.Len(t, codeFlows, 1)

	codeFlow := codeFlows[0].(map[string]any)
	threadFlows := codeFlow["threadFlows"].([]any)
	require.Len(t, threadFlows, 1)

	threadFlow := threadFlows[0].(map[string]any)
	tfLocations := threadFlow["locations"].([]any)
	require.Len(t, tfLocations, 2)

	// Source should be line 10
	sourceLoc := tfLocations[0].(map[string]any)
	sourcePhys := sourceLoc["location"].(map[string]any)["physicalLocation"].(map[string]any)
	sourceRegion := sourcePhys["region"].(map[string]any)
	assert.Equal(t, float64(10), sourceRegion["startLine"])

	// Sink should be line 20
	sinkLoc := tfLocations[1].(map[string]any)
	sinkPhys := sinkLoc["location"].(map[string]any)["physicalLocation"].(map[string]any)
	sinkRegion := sinkPhys["region"].(map[string]any)
	assert.Equal(t, float64(20), sinkRegion["startLine"])
}

func TestSARIFFormatterRelatedLocations(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "user_input",
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 20, Column: 1},
			Rule:          dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	result := results[0].(map[string]any)

	// Check related locations
	relatedLocs := result["relatedLocations"].([]any)
	require.Len(t, relatedLocs, 1)

	relatedLoc := relatedLocs[0].(map[string]any)
	physLoc := relatedLoc["physicalLocation"].(map[string]any)
	region := physLoc["region"].(map[string]any)
	assert.Equal(t, float64(10), region["startLine"])
}

func TestSARIFFormatterNoCodeFlowForPattern(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			DetectionType: dsl.DetectionTypePattern,
			Location:      dsl.LocationInfo{RelPath: "test.py", Line: 10, Column: 1},
			Rule:          dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	result := results[0].(map[string]any)

	// Pattern matches should NOT have code flows
	_, hasCodeFlows := result["codeFlows"]
	assert.False(t, hasCodeFlows)

	_, hasRelatedLocs := result["relatedLocations"]
	assert.False(t, hasRelatedLocs)
}

func TestSARIFFormatterSeverityLevels(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "error"},
		{"high", "error"},
		{"medium", "warning"},
		{"low", "note"},
		{"info", "note"},
		{"unknown", "warning"},
	}

	sf := NewSARIFFormatter(nil)
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := sf.severityToLevelString(tt.severity)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSARIFFormatterSecuritySeverity(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "9.0"},
		{"high", "7.0"},
		{"medium", "5.0"},
		{"low", "3.0"},
		{"unknown", "5.0"},
	}

	sf := NewSARIFFormatter(nil)
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := sf.severityToScore(tt.severity)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractCWENumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CWE-78", "78"},
		{"CWE-79", "79"},
		{"CWE-123", "123"},
		{"78", "78"},
		{"", ""},
		{"CWE", "CWE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractCWENumber(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSARIFFormatterMultipleRules(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "test1.py", Line: 1, Column: 1},
			Rule:     dsl.RuleMetadata{ID: "rule1", Name: "Rule 1", Severity: "high", Description: "Test 1"},
		},
		{
			Location: dsl.LocationInfo{RelPath: "test2.py", Line: 2, Column: 1},
			Rule:     dsl.RuleMetadata{ID: "rule2", Name: "Rule 2", Severity: "medium", Description: "Test 2"},
		},
		{
			Location: dsl.LocationInfo{RelPath: "test3.py", Line: 3, Column: 1},
			Rule:     dsl.RuleMetadata{ID: "rule1", Name: "Rule 1", Severity: "high", Description: "Test 1"}, // Duplicate
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	tool := run["tool"].(map[string]any)
	driver := tool["driver"].(map[string]any)

	// Should have 2 unique rules
	rules := driver["rules"].([]any)
	assert.Len(t, rules, 2)

	// Should have 3 results
	results := run["results"].([]any)
	assert.Len(t, results, 3)
}

func TestSARIFFormatterTaintGlobalCodeFlow(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 5,
				SinkLine:   50,
				TaintedVar: "global_input",
				SinkCall:   "execute_query",
			},
			DetectionType: dsl.DetectionTypeTaintGlobal,
			Location:      dsl.LocationInfo{RelPath: "app.py", Line: 50, Column: 4},
			Rule:          dsl.RuleMetadata{ID: "sql-inj", Name: "SQL Injection", Severity: "critical", Description: "SQL injection"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	result := results[0].(map[string]any)

	// Taint-global should also have code flows
	codeFlows := result["codeFlows"].([]any)
	require.Len(t, codeFlows, 1)

	codeFlow := codeFlows[0].(map[string]any)
	assert.NotNil(t, codeFlow["message"])
}

func TestSARIFFormatterFallbackToFilePath(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{
				FilePath: "/absolute/path/to/file.py",
				RelPath:  "", // Empty RelPath
				Line:     10,
				Column:   1,
			},
			Rule: dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	result := results[0].(map[string]any)

	locations := result["locations"].([]any)
	loc := locations[0].(map[string]any)
	physLoc := loc["physicalLocation"].(map[string]any)
	artifact := physLoc["artifactLocation"].(map[string]any)
	assert.Equal(t, "/absolute/path/to/file.py", artifact["uri"])
}

func TestSARIFFormatterEmptyFilePathSkipsResult(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{
				FilePath: "", // Both empty
				RelPath:  "",
				Line:     10,
			},
			Rule: dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
		{
			Location: dsl.LocationInfo{
				RelPath: "valid.py",
				Line:    5,
			},
			Rule: dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	// Only the detection with a valid path should be included
	assert.Len(t, results, 1, "Should exclude results with empty file path")
}

func TestSARIFFormatterEmptyFilePathSkipsCodeFlow(t *testing.T) {
	var buf bytes.Buffer
	sf := NewSARIFFormatterWithWriter(&buf, nil)

	detections := []*dsl.EnrichedDetection{
		{
			Detection: dsl.DataflowDetection{
				SourceLine: 10,
				SinkLine:   20,
				TaintedVar: "user_input",
				SinkCall:   "eval",
			},
			DetectionType: dsl.DetectionTypeTaintLocal,
			Location: dsl.LocationInfo{
				FilePath: "",
				RelPath:  "",
				Line:     20,
			},
			Rule: dsl.RuleMetadata{ID: "test", Name: "Test", Severity: "high", Description: "Test"},
		},
	}

	err := sf.Format(detections, ScanInfo{})
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(buf.Bytes(), &report)
	require.NoError(t, err)

	runs := report["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)

	// Entire result should be excluded when file path is empty
	assert.Empty(t, results, "Should exclude results with empty file path")
}

func TestBuildHelpMarkdown(t *testing.T) {
	sf := NewSARIFFormatter(nil)

	rule := dsl.RuleMetadata{
		Name:        "SQL Injection",
		Description: "User input flows to SQL query",
		CWE:         []string{"CWE-89", "CWE-564"},
	}

	markdown := sf.buildHelpMarkdown(rule)

	assert.Contains(t, markdown, "## SQL Injection")
	assert.Contains(t, markdown, "User input flows to SQL query")
	assert.Contains(t, markdown, "### References")
	assert.Contains(t, markdown, "CWE-89")
	assert.Contains(t, markdown, "cwe.mitre.org/data/definitions/89.html")
	assert.Contains(t, markdown, "CWE-564")
}
