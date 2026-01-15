package ruleset

import (
	"testing"
)

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		category    string
		bundle      string
	}{
		{
			name:        "valid spec",
			input:       "docker/security",
			expectError: false,
			category:    "docker",
			bundle:      "security",
		},
		{
			name:        "valid spec with hyphen",
			input:       "docker-compose/networking",
			expectError: false,
			category:    "docker-compose",
			bundle:      "networking",
		},
		{
			name:        "invalid spec - no slash",
			input:       "docker",
			expectError: true,
		},
		{
			name:        "invalid spec - too many parts",
			input:       "docker/security/extra",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := ParseSpec(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if spec.Category != tt.category {
				t.Errorf("expected category %s, got %s", tt.category, spec.Category)
			}

			if spec.Bundle != tt.bundle {
				t.Errorf("expected bundle %s, got %s", tt.bundle, spec.Bundle)
			}
		})
	}
}

func TestRulesetSpecValidate(t *testing.T) {
	tests := []struct {
		name        string
		spec        RulesetSpec
		expectError bool
	}{
		{
			name: "valid spec",
			spec: RulesetSpec{
				Category: "docker",
				Bundle:   "security",
			},
			expectError: false,
		},
		{
			name: "empty category",
			spec: RulesetSpec{
				Category: "",
				Bundle:   "security",
			},
			expectError: true,
		},
		{
			name: "empty bundle",
			spec: RulesetSpec{
				Category: "docker",
				Bundle:   "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRulesetSpecString(t *testing.T) {
	spec := RulesetSpec{
		Category: "docker",
		Bundle:   "security",
	}

	expected := "docker/security"
	result := spec.String()

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
