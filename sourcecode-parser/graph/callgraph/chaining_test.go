package callgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseChain(t *testing.T) {
	tests := []struct {
		name          string
		target        string
		expectedSteps int
		expectedNames []string
	}{
		{
			name:          "simple two-step chain",
			target:        "create_builder().append()",
			expectedSteps: 2,
			expectedNames: []string{"create_builder", "append"},
		},
		{
			name:          "three-step chain",
			target:        "create_builder().append().upper()",
			expectedSteps: 3,
			expectedNames: []string{"create_builder", "append", "upper"},
		},
		{
			name:          "four-step chain",
			target:        "create_builder().append().upper().build()",
			expectedSteps: 4,
			expectedNames: []string{"create_builder", "append", "upper", "build"},
		},
		{
			name:          "chain with arguments",
			target:        `create_builder().append("hello ").append("world").upper().build()`,
			expectedSteps: 5,
			// Note: method names are extracted without arguments for calls
			expectedNames: []string{"create_builder", "append", "append", "upper", "build"},
		},
		{
			name:          "builtin chain",
			target:        "text.strip().upper().split()",
			expectedSteps: 3,
			expectedNames: []string{"strip", "upper", "split"},
		},
		{
			name:          "not a chain - simple call",
			target:        "function()",
			expectedSteps: 0,
			expectedNames: nil,
		},
		{
			name:          "not a chain - attribute access",
			target:        "obj.method()",
			expectedSteps: 0,
			expectedNames: nil,
		},
		{
			name:          "not a chain - nested attribute",
			target:        "obj.attr.method()",
			expectedSteps: 0,
			expectedNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := ParseChain(tt.target)

			if tt.expectedSteps == 0 {
				assert.Nil(t, steps, "Expected no chain for: %s", tt.target)
				return
			}

			assert.NotNil(t, steps, "Expected chain for: %s", tt.target)
			assert.Equal(t, tt.expectedSteps, len(steps), "Wrong number of steps for: %s", tt.target)

			if tt.expectedNames != nil {
				for i, expectedName := range tt.expectedNames {
					if i < len(steps) {
						assert.Equal(t, expectedName, steps[i].MethodName,
							"Step %d: expected name %s, got %s", i, expectedName, steps[i].MethodName)
					}
				}
			}
		})
	}
}

func TestParseStep(t *testing.T) {
	tests := []struct {
		name           string
		expr           string
		expectedName   string
		expectedIsCall bool
	}{
		{
			name:           "simple call",
			expr:           "function()",
			expectedName:   "function",
			expectedIsCall: true,
		},
		{
			name:           "method call",
			expr:           "obj.method()",
			expectedName:   "method",
			expectedIsCall: true,
		},
		{
			name:           "nested method call",
			expr:           "obj.attr.method()",
			expectedName:   "method",
			expectedIsCall: true,
		},
		{
			name:           "variable access",
			expr:           "variable",
			expectedName:   "variable",
			expectedIsCall: false,
		},
		{
			name:           "attribute access",
			expr:           "obj.attr",
			expectedName:   "obj.attr",
			expectedIsCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := parseStep(tt.expr)

			assert.NotNil(t, step)
			assert.Equal(t, tt.expectedName, step.MethodName)
			assert.Equal(t, tt.expectedIsCall, step.IsCall)
			assert.Equal(t, tt.expr, step.Expression)
		})
	}
}
