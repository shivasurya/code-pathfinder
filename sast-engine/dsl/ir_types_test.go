package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallMatcherIR_GetType(t *testing.T) {
	t.Run("returns correct IR type", func(t *testing.T) {
		matcher := &CallMatcherIR{
			Type:      "call_matcher",
			Patterns:  []string{"eval", "exec"},
			Wildcard:  false,
			MatchMode: "any",
		}

		assert.Equal(t, IRTypeCallMatcher, matcher.GetType())
	})

	t.Run("works with wildcard patterns", func(t *testing.T) {
		matcher := &CallMatcherIR{
			Type:      "call_matcher",
			Patterns:  []string{"request.*", "*.GET"},
			Wildcard:  true,
			MatchMode: "all",
		}

		assert.Equal(t, IRTypeCallMatcher, matcher.GetType())
	})
}

func TestVariableMatcherIR_GetType(t *testing.T) {
	t.Run("returns correct IR type", func(t *testing.T) {
		matcher := &VariableMatcherIR{
			Type:     "variable_matcher",
			Pattern:  "user_input",
			Wildcard: false,
		}

		assert.Equal(t, IRTypeVariableMatcher, matcher.GetType())
	})

	t.Run("works with wildcard pattern", func(t *testing.T) {
		matcher := &VariableMatcherIR{
			Type:     "variable_matcher",
			Pattern:  "user_*",
			Wildcard: true,
		}

		assert.Equal(t, IRTypeVariableMatcher, matcher.GetType())
	})
}

func TestDataflowIR_GetType(t *testing.T) {
	t.Run("returns correct IR type", func(t *testing.T) {
		dataflow := &DataflowIR{
			Type: "dataflow",
			Sources: []CallMatcherIR{
				{Type: "call_matcher", Patterns: []string{"request.GET"}},
			},
			Sinks: []CallMatcherIR{
				{Type: "call_matcher", Patterns: []string{"eval"}},
			},
			Sanitizers: []CallMatcherIR{
				{Type: "call_matcher", Patterns: []string{"escape"}},
			},
			Scope: "local",
		}

		assert.Equal(t, IRTypeDataflow, dataflow.GetType())
	})

	t.Run("works with global scope", func(t *testing.T) {
		dataflow := &DataflowIR{
			Type: "dataflow",
			Sources: []CallMatcherIR{
				{Type: "call_matcher", Patterns: []string{"input"}},
			},
			Sinks: []CallMatcherIR{
				{Type: "call_matcher", Patterns: []string{"execute"}},
			},
			Sanitizers: []CallMatcherIR{},
			Propagation: []PropagationIR{
				{Type: "assignment", Metadata: map[string]any{"key": "value"}},
			},
			Scope: "global",
		}

		assert.Equal(t, IRTypeDataflow, dataflow.GetType())
	})
}

func TestIRTypeConstants(t *testing.T) {
	t.Run("IR type constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, IRType("call_matcher"), IRTypeCallMatcher)
		assert.Equal(t, IRType("variable_matcher"), IRTypeVariableMatcher)
		assert.Equal(t, IRType("dataflow"), IRTypeDataflow)
		assert.Equal(t, IRType("logic_and"), IRTypeLogicAnd)
		assert.Equal(t, IRType("logic_or"), IRTypeLogicOr)
		assert.Equal(t, IRType("logic_not"), IRTypeLogicNot)
	})
}

func TestMatcherIR_Interface(t *testing.T) {
	t.Run("CallMatcherIR implements MatcherIR interface", func(t *testing.T) {
		var matcher MatcherIR = &CallMatcherIR{
			Type:     "call_matcher",
			Patterns: []string{"test"},
		}
		assert.Equal(t, IRTypeCallMatcher, matcher.GetType())
	})

	t.Run("VariableMatcherIR implements MatcherIR interface", func(t *testing.T) {
		var matcher MatcherIR = &VariableMatcherIR{
			Type:    "variable_matcher",
			Pattern: "test_var",
		}
		assert.Equal(t, IRTypeVariableMatcher, matcher.GetType())
	})

	t.Run("DataflowIR implements MatcherIR interface", func(t *testing.T) {
		var matcher MatcherIR = &DataflowIR{
			Type:  "dataflow",
			Scope: "local",
		}
		assert.Equal(t, IRTypeDataflow, matcher.GetType())
	})
}
