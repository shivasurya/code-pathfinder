package diagnostic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateConsoleReport tests human-readable console output.
func TestGenerateConsoleReport(t *testing.T) {
	metrics := &OverallMetrics{
		TotalFunctions:      100,
		TruePositives:       60,
		FalsePositives:      10,
		FalseNegatives:      20,
		TrueNegatives:       10,
		Precision:           0.857,
		Recall:              0.75,
		F1Score:             0.8,
		Agreement:           0.7,
		TotalProcessingTime: "5m30s",
		FunctionsPerSecond:  0.303,
		FailuresByCategory: map[string]int{
			"control_flow_branch": 15,
			"sanitizer_missed":    10,
			"field_sensitivity":   5,
		},
		TopFailures: []FailureExample{
			{
				Type:         "FALSE_POSITIVE",
				FunctionFQN:  "test.example",
				FunctionFile: "test.py",
				FunctionLine: 42,
				Category:     "sanitizer_missed",
				Reason:       "Data is sanitized by escape function",
			},
		},
	}

	err := GenerateConsoleReport(metrics, "")
	assert.NoError(t, err)
}

// TestGenerateConsoleReport_WithOutputDir tests console report with output directory.
func TestGenerateConsoleReport_WithOutputDir(t *testing.T) {
	metrics := &OverallMetrics{
		TotalFunctions:      50,
		TruePositives:       30,
		FalsePositives:      5,
		FalseNegatives:      10,
		TrueNegatives:       5,
		Precision:           0.857,
		Recall:              0.75,
		F1Score:             0.8,
		Agreement:           0.7,
		TotalProcessingTime: "2m15s",
		FunctionsPerSecond:  0.370,
		FailuresByCategory:  map[string]int{},
		TopFailures:         []FailureExample{},
	}

	err := GenerateConsoleReport(metrics, "/tmp/diagnostic_output")
	assert.NoError(t, err)
}

// TestGenerateConsoleReport_NoFailures tests report with no failures.
func TestGenerateConsoleReport_NoFailures(t *testing.T) {
	metrics := &OverallMetrics{
		TotalFunctions:      10,
		TruePositives:       5,
		FalsePositives:      0,
		FalseNegatives:      0,
		TrueNegatives:       5,
		Precision:           1.0,
		Recall:              1.0,
		F1Score:             1.0,
		Agreement:           1.0,
		TotalProcessingTime: "30s",
		FunctionsPerSecond:  0.333,
		FailuresByCategory:  map[string]int{},
		TopFailures:         []FailureExample{},
	}

	err := GenerateConsoleReport(metrics, "")
	assert.NoError(t, err)
}

// TestGenerateJSONReport tests machine-readable JSON output.
func TestGenerateJSONReport(t *testing.T) {
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "report.json")

	metrics := &OverallMetrics{
		TotalFunctions:      100,
		TruePositives:       60,
		FalsePositives:      10,
		FalseNegatives:      20,
		TrueNegatives:       10,
		Precision:           0.857,
		Recall:              0.75,
		F1Score:             0.8,
		Agreement:           0.7,
		TotalProcessingTime: "5m30s",
		FunctionsPerSecond:  0.303,
		FailuresByCategory: map[string]int{
			"control_flow_branch": 15,
		},
		TopFailures: []FailureExample{
			{
				Type:         "FALSE_POSITIVE",
				FunctionFQN:  "test.example",
				FunctionFile: "test.py",
				FunctionLine: 42,
				Category:     "sanitizer_missed",
				Reason:       "Data is sanitized",
			},
		},
	}

	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.func1",
			BinaryToolResult: true,
			BinaryLLMResult:  true,
			BinaryAgreement:  true,
		},
		{
			FunctionFQN:      "test.func2",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Sanitized",
		},
	}

	err := GenerateJSONReport(metrics, comparisons, jsonPath)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(jsonPath)
	assert.NoError(t, err)

	// Verify JSON structure
	jsonBytes, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(jsonBytes, &report)
	require.NoError(t, err)

	assert.Contains(t, report, "metrics")
	assert.Contains(t, report, "comparisons")
	assert.Contains(t, report, "timestamp")

	// Verify metrics structure
	metricsData := report["metrics"].(map[string]any)
	assert.Equal(t, float64(100), metricsData["TotalFunctions"])
	assert.Equal(t, float64(60), metricsData["TruePositives"])

	// Verify comparisons array
	comparisonsData := report["comparisons"].([]any)
	assert.Equal(t, 2, len(comparisonsData))
}

// TestGenerateJSONReport_EmptyData tests JSON report with empty data.
func TestGenerateJSONReport_EmptyData(t *testing.T) {
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "empty_report.json")

	metrics := &OverallMetrics{
		TotalFunctions:     0,
		FailuresByCategory: map[string]int{},
		TopFailures:        []FailureExample{},
	}

	comparisons := []*DualLevelComparison{}

	err := GenerateJSONReport(metrics, comparisons, jsonPath)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(jsonPath)
	assert.NoError(t, err)

	// Verify JSON is valid
	jsonBytes, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(jsonBytes, &report)
	require.NoError(t, err)
}

// TestGenerateJSONReport_InvalidPath tests error handling for invalid path.
func TestGenerateJSONReport_InvalidPath(t *testing.T) {
	metrics := &OverallMetrics{
		TotalFunctions: 10,
	}
	comparisons := []*DualLevelComparison{}

	// Use invalid path (directory doesn't exist and is deeply nested)
	invalidPath := "/nonexistent/deeply/nested/path/report.json"

	err := GenerateJSONReport(metrics, comparisons, invalidPath)
	assert.Error(t, err)
}

// TestGenerateJSONReport_TimestampFormat tests timestamp is in RFC3339 format.
func TestGenerateJSONReport_TimestampFormat(t *testing.T) {
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "timestamp_report.json")

	metrics := &OverallMetrics{
		TotalFunctions: 10,
	}
	comparisons := []*DualLevelComparison{}

	err := GenerateJSONReport(metrics, comparisons, jsonPath)
	require.NoError(t, err)

	jsonBytes, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var report map[string]any
	err = json.Unmarshal(jsonBytes, &report)
	require.NoError(t, err)

	timestampStr := report["timestamp"].(string)
	assert.NotEmpty(t, timestampStr)

	// Verify RFC3339 format
	_, err = time.Parse(time.RFC3339, timestampStr)
	assert.NoError(t, err)
}

// TestWrapText tests text wrapping helper.
func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		prefix   string
		expected string
	}{
		{
			name:     "short text",
			text:     "Short text",
			width:    20,
			prefix:   "",
			expected: "Short text",
		},
		{
			name:     "long text with wrapping",
			text:     "This is a very long text that should be wrapped at the specified width",
			width:    20,
			prefix:   "  ",
			expected: "This is a very long\n  text that should be\n  wrapped at the\n  specified width",
		},
		{
			name:     "empty text",
			text:     "",
			width:    20,
			prefix:   "",
			expected: "",
		},
		{
			name:     "single word",
			text:     "Word",
			width:    10,
			prefix:   "",
			expected: "Word",
		},
		{
			name:     "exact width",
			text:     "Exactly twenty chars",
			width:    20,
			prefix:   "",
			expected: "Exactly twenty chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGenerateConsoleReport_MultipleFailureCategories tests report with many categories.
func TestGenerateConsoleReport_MultipleFailureCategories(t *testing.T) {
	metrics := &OverallMetrics{
		TotalFunctions: 100,
		TruePositives:  50,
		FalsePositives: 25,
		FalseNegatives: 25,
		TrueNegatives:  0,
		Precision:      0.667,
		Recall:         0.667,
		F1Score:        0.667,
		Agreement:      0.5,
		FailuresByCategory: map[string]int{
			"control_flow_branch": 20,
			"sanitizer_missed":    15,
			"field_sensitivity":   10,
			"container_operation": 3,
			"string_formatting":   2,
		},
		TopFailures:         []FailureExample{},
		TotalProcessingTime: "10m",
		FunctionsPerSecond:  0.167,
	}

	err := GenerateConsoleReport(metrics, "")
	assert.NoError(t, err)
}

// TestGenerateConsoleReport_LongReasonText tests wrapping of long failure reasons.
func TestGenerateConsoleReport_LongReasonText(t *testing.T) {
	metrics := &OverallMetrics{
		TotalFunctions: 10,
		TruePositives:  5,
		FalsePositives: 5,
		Precision:      0.5,
		Recall:         1.0,
		F1Score:        0.667,
		Agreement:      0.5,
		TopFailures: []FailureExample{
			{
				Type:         "FALSE_POSITIVE",
				FunctionFQN:  "test.long_reason",
				FunctionFile: "test.py",
				FunctionLine: 100,
				Category:     "sanitizer_missed",
				Reason:       "This is a very long reason that explains in great detail why the tool incorrectly flagged this function as vulnerable when in reality it is perfectly safe because the data is sanitized by multiple layers of validation and escaping before being used in the SQL query",
			},
		},
		FailuresByCategory:  map[string]int{},
		TotalProcessingTime: "1m",
		FunctionsPerSecond:  0.167,
	}

	err := GenerateConsoleReport(metrics, "")
	assert.NoError(t, err)
}
