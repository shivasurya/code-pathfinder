package ruleset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     *RulesetSpec
		wantErr  bool
		errMsg   string
	}{
		{
			name:  "valid bundle spec",
			input: "docker/security",
			want: &RulesetSpec{
				Category: "docker",
				Bundle:   "security",
			},
			wantErr: false,
		},
		{
			name:  "valid bundle spec with hyphens",
			input: "docker-compose/best-practice",
			want: &RulesetSpec{
				Category: "docker-compose",
				Bundle:   "best-practice",
			},
			wantErr: false,
		},
		{
			name:  "valid category expansion - docker/all",
			input: "docker/all",
			want: &RulesetSpec{
				Category: "docker",
				Bundle:   "*",
			},
			wantErr: false,
		},
		{
			name:  "valid category expansion - python/all",
			input: "python/all",
			want: &RulesetSpec{
				Category: "python",
				Bundle:   "*",
			},
			wantErr: false,
		},
		{
			name:    "invalid - no slash",
			input:   "dockersecurity",
			want:    nil,
			wantErr: true,
			errMsg:  "expected format: category/bundle",
		},
		{
			name:    "invalid - too many parts",
			input:   "docker/security/extra",
			want:    nil,
			wantErr: true,
			errMsg:  "expected format: category/bundle",
		},
		{
			name:    "invalid - empty string",
			input:   "",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSpec(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseRuleSpec(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *RuleSpec
		wantErr bool
		errMsg  string
	}{
		{
			name:  "valid rule spec - DOCKER-BP-007",
			input: "docker/DOCKER-BP-007",
			want: &RuleSpec{
				Language: "docker",
				RuleID:   "DOCKER-BP-007",
			},
			wantErr: false,
		},
		{
			name:  "valid rule spec - PYTHON-SEC-001",
			input: "python/PYTHON-SEC-001",
			want: &RuleSpec{
				Language: "python",
				RuleID:   "PYTHON-SEC-001",
			},
			wantErr: false,
		},
		{
			name:  "valid rule spec - COMPOSE-SEC-008",
			input: "docker-compose/COMPOSE-SEC-008",
			want: &RuleSpec{
				Language: "docker-compose",
				RuleID:   "COMPOSE-SEC-008",
			},
			wantErr: false,
		},
		{
			name:    "invalid - not a rule ID format",
			input:   "docker/security",
			want:    nil,
			wantErr: true,
			errMsg:  "invalid rule ID format",
		},
		{
			name:    "invalid - lowercase rule ID",
			input:   "docker/docker-bp-007",
			want:    nil,
			wantErr: true,
			errMsg:  "invalid rule ID format",
		},
		{
			name:    "invalid - no slash",
			input:   "DOCKER-BP-007",
			want:    nil,
			wantErr: true,
			errMsg:  "expected format: language/RULE-ID",
		},
		{
			name:    "invalid - too many parts",
			input:   "docker/DOCKER-BP-007/extra",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRuleSpec(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsRuleID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid DOCKER-BP-007", input: "DOCKER-BP-007", want: true},
		{name: "valid PYTHON-SEC-001", input: "PYTHON-SEC-001", want: true},
		{name: "valid COMPOSE-SEC-008", input: "COMPOSE-SEC-008", want: true},
		{name: "valid single part prefix", input: "PY-001", want: true},
		{name: "invalid lowercase", input: "docker-bp-007", want: false},
		{name: "invalid mixed case", input: "Docker-BP-007", want: false},
		{name: "invalid no dash", input: "DOCKERBP007", want: false},
		{name: "invalid just text", input: "security", want: false},
		{name: "invalid empty", input: "", want: false},
		{name: "invalid no number", input: "DOCKER-BP-", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRuleID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRulesetSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *RulesetSpec
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid",
			spec:    &RulesetSpec{Category: "docker", Bundle: "security"},
			wantErr: false,
		},
		{
			name:    "empty category",
			spec:    &RulesetSpec{Category: "", Bundle: "security"},
			wantErr: true,
			errMsg:  "category cannot be empty",
		},
		{
			name:    "empty bundle",
			spec:    &RulesetSpec{Category: "docker", Bundle: ""},
			wantErr: true,
			errMsg:  "bundle cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRuleSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *RuleSpec
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid",
			spec:    &RuleSpec{Language: "docker", RuleID: "DOCKER-BP-007"},
			wantErr: false,
		},
		{
			name:    "empty language",
			spec:    &RuleSpec{Language: "", RuleID: "DOCKER-BP-007"},
			wantErr: true,
			errMsg:  "language cannot be empty",
		},
		{
			name:    "empty rule ID",
			spec:    &RuleSpec{Language: "docker", RuleID: ""},
			wantErr: true,
			errMsg:  "rule ID cannot be empty",
		},
		{
			name:    "invalid rule ID format",
			spec:    &RuleSpec{Language: "docker", RuleID: "invalid"},
			wantErr: true,
			errMsg:  "invalid rule ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRulesetSpecString(t *testing.T) {
	spec := &RulesetSpec{Category: "docker", Bundle: "security"}
	assert.Equal(t, "docker/security", spec.String())
}

func TestRuleSpecString(t *testing.T) {
	spec := &RuleSpec{Language: "docker", RuleID: "DOCKER-BP-007"}
	assert.Equal(t, "docker/DOCKER-BP-007", spec.String())
}
