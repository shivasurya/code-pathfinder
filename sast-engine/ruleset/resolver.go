package ruleset

import (
	"fmt"
	"strings"
)

// ParseSpec parses "docker/security" into RulesetSpec.
func ParseSpec(spec string) (*RulesetSpec, error) {
	parts := strings.Split(spec, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ruleset spec: %s (expected format: category/bundle)", spec)
	}

	return &RulesetSpec{
		Category: parts[0],
		Bundle:   parts[1],
	}, nil
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

// String returns the spec as "category/bundle".
func (s *RulesetSpec) String() string {
	return fmt.Sprintf("%s/%s", s.Category, s.Bundle)
}
