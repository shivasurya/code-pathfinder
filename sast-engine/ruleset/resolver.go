package ruleset

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule ID pattern: starts with uppercase letters, followed by dash, uppercase letters/numbers, dash, and numbers.
// Examples: DOCKER-BP-007, PYTHON-SEC-001, COMPOSE-SEC-008.
var ruleIDPattern = regexp.MustCompile(`^[A-Z]+(-[A-Z]+)?-\d+$`)

// ParseSpec parses "docker/security" or "docker/all" into RulesetSpec.
// The "all" keyword is special and expands to all bundles in the category.
func ParseSpec(spec string) (*RulesetSpec, error) {
	parts := strings.Split(spec, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ruleset spec: %s (expected format: category/bundle)", spec)
	}

	category := parts[0]
	bundle := parts[1]

	// Check if bundle is "all" - special keyword for category expansion
	// We use "*" as internal marker for expansion logic
	if bundle == "all" {
		return &RulesetSpec{
			Category: category,
			Bundle:   "*",
		}, nil
	}

	return &RulesetSpec{
		Category: category,
		Bundle:   bundle,
	}, nil
}

// ParseRuleSpec parses "docker/DOCKER-BP-007" into RuleSpec.
func ParseRuleSpec(spec string) (*RuleSpec, error) {
	parts := strings.Split(spec, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid rule spec: %s (expected format: language/RULE-ID)", spec)
	}

	ruleID := parts[1]
	if !ruleIDPattern.MatchString(ruleID) {
		return nil, fmt.Errorf("invalid rule ID format: %s (expected format like DOCKER-BP-007)", ruleID)
	}

	return &RuleSpec{
		Language: parts[0],
		RuleID:   ruleID,
	}, nil
}

// IsRuleID checks if a string looks like a rule ID (e.g., DOCKER-BP-007).
func IsRuleID(s string) bool {
	return ruleIDPattern.MatchString(s)
}

// Validate checks if spec is valid.
func (s *RulesetSpec) Validate() error {
	if s.Category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	if s.Bundle == "" {
		return fmt.Errorf("bundle cannot be empty")
	}
	// Add more validation as needed
	return nil
}

// Validate checks if rule spec is valid.
func (s *RuleSpec) Validate() error {
	if s.Language == "" {
		return fmt.Errorf("language cannot be empty")
	}
	if s.RuleID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}
	if !ruleIDPattern.MatchString(s.RuleID) {
		return fmt.Errorf("invalid rule ID format: %s", s.RuleID)
	}
	return nil
}

// String returns the spec as "category/bundle".
func (s *RulesetSpec) String() string {
	return fmt.Sprintf("%s/%s", s.Category, s.Bundle)
}

// String returns the spec as "language/RULE-ID".
func (s *RuleSpec) String() string {
	return fmt.Sprintf("%s/%s", s.Language, s.RuleID)
}
