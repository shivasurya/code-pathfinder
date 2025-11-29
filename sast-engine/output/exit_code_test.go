package output

import (
	"errors"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineExitCode(t *testing.T) {
	tests := []struct {
		name       string
		detections []*dsl.EnrichedDetection
		failOn     []string
		hadErrors  bool
		expected   ExitCode
	}{
		{
			name:       "No detections, no fail-on",
			detections: []*dsl.EnrichedDetection{},
			failOn:     []string{},
			hadErrors:  false,
			expected:   ExitCodeSuccess,
		},
		{
			name: "Detections present, no fail-on",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
			},
			failOn:    []string{},
			hadErrors: false,
			expected:  ExitCodeSuccess,
		},
		{
			name: "Critical finding matches fail-on critical",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
			},
			failOn:    []string{"critical"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "High finding matches fail-on high",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "high"}},
			},
			failOn:    []string{"high"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "Multiple severities, matches critical",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
				{Rule: dsl.RuleMetadata{Severity: "low"}},
			},
			failOn:    []string{"critical", "high"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "Multiple severities, matches high",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "high"}},
				{Rule: dsl.RuleMetadata{Severity: "medium"}},
			},
			failOn:    []string{"critical", "high"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "Finding does not match fail-on",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "low"}},
			},
			failOn:    []string{"critical", "high"},
			hadErrors: false,
			expected:  ExitCodeSuccess,
		},
		{
			name: "Medium finding, fail-on critical/high",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "medium"}},
			},
			failOn:    []string{"critical", "high"},
			hadErrors: false,
			expected:  ExitCodeSuccess,
		},
		{
			name:       "Errors take precedence over no findings",
			detections: []*dsl.EnrichedDetection{},
			failOn:     []string{"critical"},
			hadErrors:  true,
			expected:   ExitCodeError,
		},
		{
			name: "Errors take precedence over findings",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
			},
			failOn:    []string{"critical"},
			hadErrors: true,
			expected:  ExitCodeError,
		},
		{
			name: "Case insensitive matching - uppercase severity",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "CRITICAL"}},
			},
			failOn:    []string{"critical"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "Case insensitive matching - uppercase fail-on",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
			},
			failOn:    []string{"CRITICAL"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "Case insensitive matching - mixed case",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "CrItIcAl"}},
			},
			failOn:    []string{"cRiTiCaL"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "All severities match",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
				{Rule: dsl.RuleMetadata{Severity: "high"}},
				{Rule: dsl.RuleMetadata{Severity: "medium"}},
				{Rule: dsl.RuleMetadata{Severity: "low"}},
				{Rule: dsl.RuleMetadata{Severity: "info"}},
			},
			failOn:    []string{"critical", "high", "medium", "low", "info"},
			hadErrors: false,
			expected:  ExitCodeFindings,
		},
		{
			name: "No findings match any fail-on severity",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "info"}},
			},
			failOn:    []string{"critical", "high"},
			hadErrors: false,
			expected:  ExitCodeSuccess,
		},
		{
			name: "Empty fail-on with errors",
			detections: []*dsl.EnrichedDetection{
				{Rule: dsl.RuleMetadata{Severity: "critical"}},
			},
			failOn:    []string{},
			hadErrors: true,
			expected:  ExitCodeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineExitCode(tt.detections, tt.failOn, tt.hadErrors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFailOn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Whitespace only",
			input:    "   ",
			expected: []string{},
		},
		{
			name:     "Single severity",
			input:    "critical",
			expected: []string{"critical"},
		},
		{
			name:     "Multiple severities",
			input:    "critical,high",
			expected: []string{"critical", "high"},
		},
		{
			name:     "Multiple severities with spaces",
			input:    "critical, high, medium",
			expected: []string{"critical", "high", "medium"},
		},
		{
			name:     "Trimming leading/trailing spaces",
			input:    "  critical  ,  high  ",
			expected: []string{"critical", "high"},
		},
		{
			name:     "All severities",
			input:    "critical,high,medium,low,info",
			expected: []string{"critical", "high", "medium", "low", "info"},
		},
		{
			name:     "Empty segments ignored",
			input:    "critical,,high",
			expected: []string{"critical", "high"},
		},
		{
			name:     "Trailing comma ignored",
			input:    "critical,high,",
			expected: []string{"critical", "high"},
		},
		{
			name:     "Leading comma ignored",
			input:    ",critical,high",
			expected: []string{"critical", "high"},
		},
		{
			name:     "Multiple empty segments",
			input:    "critical,,,high",
			expected: []string{"critical", "high"},
		},
		{
			name:     "Mixed case preserved",
			input:    "CRITICAL,High,MeDiUm",
			expected: []string{"CRITICAL", "High", "MeDiUm"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFailOn(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSeverities(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Empty list",
			input:     []string{},
			wantError: false,
		},
		{
			name:      "Valid single severity - critical",
			input:     []string{"critical"},
			wantError: false,
		},
		{
			name:      "Valid single severity - high",
			input:     []string{"high"},
			wantError: false,
		},
		{
			name:      "Valid single severity - medium",
			input:     []string{"medium"},
			wantError: false,
		},
		{
			name:      "Valid single severity - low",
			input:     []string{"low"},
			wantError: false,
		},
		{
			name:      "Valid single severity - info",
			input:     []string{"info"},
			wantError: false,
		},
		{
			name:      "Valid multiple severities",
			input:     []string{"critical", "high", "medium"},
			wantError: false,
		},
		{
			name:      "Valid all severities",
			input:     []string{"critical", "high", "medium", "low", "info"},
			wantError: false,
		},
		{
			name:      "Invalid severity",
			input:     []string{"invalid"},
			wantError: true,
			errorMsg:  "invalid severity 'invalid', must be one of: critical, high, medium, low, info",
		},
		{
			name:      "Valid then invalid",
			input:     []string{"critical", "invalid"},
			wantError: true,
			errorMsg:  "invalid severity 'invalid', must be one of: critical, high, medium, low, info",
		},
		{
			name:      "Invalid then valid",
			input:     []string{"invalid", "critical"},
			wantError: true,
			errorMsg:  "invalid severity 'invalid', must be one of: critical, high, medium, low, info",
		},
		{
			name:      "Case insensitive - uppercase",
			input:     []string{"CRITICAL", "HIGH"},
			wantError: false,
		},
		{
			name:      "Case insensitive - mixed case",
			input:     []string{"CrItIcAl", "HiGh"},
			wantError: false,
		},
		{
			name:      "Invalid case preserved in error",
			input:     []string{"INVALID"},
			wantError: true,
			errorMsg:  "invalid severity 'INVALID', must be one of: critical, high, medium, low, info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSeverities(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())

				// Verify it's an InvalidSeverityError using errors.As
				var invalidErr *InvalidSeverityError
				assert.True(t, errors.As(err, &invalidErr), "error should be *InvalidSeverityError")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSeverities_ErrorAsCheck(t *testing.T) {
	err := ValidateSeverities([]string{"invalid"})
	require.Error(t, err)

	var invalidErr *InvalidSeverityError
	require.True(t, errors.As(err, &invalidErr), "error should be *InvalidSeverityError")
	require.Equal(t, "invalid", invalidErr.Severity)
}

func TestInvalidSeverityError(t *testing.T) {
	err := &InvalidSeverityError{
		Severity: "unknown",
		Valid:    []string{"critical", "high", "medium", "low", "info"},
	}

	expected := "invalid severity 'unknown', must be one of: critical, high, medium, low, info"
	assert.Equal(t, expected, err.Error())
}

func TestExitCodeConstants(t *testing.T) {
	assert.Equal(t, ExitCode(0), ExitCodeSuccess)
	assert.Equal(t, ExitCode(1), ExitCodeFindings)
	assert.Equal(t, ExitCode(2), ExitCodeError)
}
