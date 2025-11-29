package output

import (
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// ExitCode represents the exit code for the CLI.
type ExitCode int

const (
	// ExitCodeSuccess indicates successful execution with no findings or no --fail-on match.
	ExitCodeSuccess ExitCode = 0

	// ExitCodeFindings indicates findings match --fail-on severities.
	ExitCodeFindings ExitCode = 1

	// ExitCodeError indicates configuration or execution error.
	ExitCodeError ExitCode = 2
)

// InvalidSeverityError is returned when an invalid severity is provided.
type InvalidSeverityError struct {
	Severity string
	Valid    []string
}

func (e *InvalidSeverityError) Error() string {
	return fmt.Sprintf("invalid severity '%s', must be one of: %s",
		e.Severity, strings.Join(e.Valid, ", "))
}

var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
	"info":     true,
}

// DetermineExitCode calculates the appropriate exit code based on detections,
// fail-on severities, and whether errors occurred during execution.
//
// Exit code precedence:
// 1. ExitCodeError (2) - if hadErrors is true.
// 2. ExitCodeFindings (1) - if any detections match fail-on severities.
// 3. ExitCodeSuccess (0) - otherwise (no findings or no --fail-on match).
func DetermineExitCode(detections []*dsl.EnrichedDetection, failOn []string, hadErrors bool) ExitCode {
	// Errors take precedence over findings
	if hadErrors {
		return ExitCodeError
	}

	// If no --fail-on specified, always return success
	if len(failOn) == 0 {
		return ExitCodeSuccess
	}

	// Check if any detections match fail-on severities
	failOnMap := make(map[string]bool)
	for _, severity := range failOn {
		failOnMap[strings.ToLower(severity)] = true
	}

	for _, det := range detections {
		if failOnMap[strings.ToLower(det.Rule.Severity)] {
			return ExitCodeFindings
		}
	}

	return ExitCodeSuccess
}

// ParseFailOn parses the comma-separated --fail-on flag value into a slice of severities.
// Empty strings and whitespace are trimmed. Returns empty slice for empty input.
func ParseFailOn(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ValidateSeverities checks that all provided severities are valid.
// Valid severities are: critical, high, medium, low, info (case-insensitive).
// Returns InvalidSeverityError for the first invalid severity encountered.
func ValidateSeverities(severities []string) error {
	validList := []string{"critical", "high", "medium", "low", "info"}

	for _, severity := range severities {
		normalized := strings.ToLower(severity)
		if !validSeverities[normalized] {
			return &InvalidSeverityError{
				Severity: severity,
				Valid:    validList,
			}
		}
	}
	return nil
}
