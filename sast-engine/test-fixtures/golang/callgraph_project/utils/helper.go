package utils

import "strings"

// Helper performs string validation.
func Helper(input string) bool {
	trimmed := strings.TrimSpace(input)
	return len(trimmed) > 0
}

// ValidateLength checks if string meets minimum length.
func ValidateLength(s string, minLen int) bool {
	return len(s) >= minLen
}
