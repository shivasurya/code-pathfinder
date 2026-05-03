package dsl

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- validateTypeConstrainedCallIR tests ---

func TestValidateTypeConstrainedCallIR_Valid(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverType:  "sqlite3.Cursor",
		MethodName:    "execute",
		MinConfidence: 0.5,
		FallbackMode:  "name",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, 0, dc.Count())
}

func TestValidateTypeConstrainedCallIR_Nil(t *testing.T) {
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(nil, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestValidateTypeConstrainedCallIR_EmptyReceiver(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		MethodName: "execute",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateTypeConstrainedCallIR_EmptyMethod(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverType: "sqlite3.Cursor",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err) // Warning, not error.
	assert.True(t, dc.HasWarnings())
	warnings := dc.FilterByLevel("warning")
	assert.Contains(t, warnings[0].Message, "no methodName")
}

func TestValidateTypeConstrainedCallIR_ConfidenceClamping(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverType:  "test.Type",
		MethodName:    "method",
		MinConfidence: 1.5,
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, ir.MinConfidence)
	assert.True(t, dc.HasWarnings())
}

func TestValidateTypeConstrainedCallIR_InvalidFallback(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverType: "test.Type",
		MethodName:   "method",
		FallbackMode: "invalid_mode",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, "name", ir.FallbackMode) // Reset to default.
	assert.True(t, dc.HasWarnings())
}

func TestValidateTypeConstrainedCallIR_ReceiverPatterns(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverPatterns: []string{"flask.*"},
		MethodName:       "get",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err)
}

func TestValidateTypeConstrainedCallIR_ValidFallbackNone(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverType: "test.Type",
		MethodName:   "method",
		FallbackMode: "none",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, "none", ir.FallbackMode)
	assert.Equal(t, 0, dc.Count())
}

func TestValidateTypeConstrainedCallIR_EmptyFallbackNoWarning(t *testing.T) {
	ir := &TypeConstrainedCallIR{
		ReceiverType: "test.Type",
		MethodName:   "method",
		FallbackMode: "",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedCallIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, 0, dc.Count()) // Empty is acceptable (uses default).
}

// --- validateTypeConstrainedAttributeIR tests ---

func TestValidateTypeConstrainedAttributeIR_Valid(t *testing.T) {
	ir := &TypeConstrainedAttributeIR{
		ReceiverType:  "django.http.HttpRequest",
		AttributeName: "GET",
		MinConfidence: 0.5,
		FallbackMode:  "name",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedAttributeIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, 0, dc.Count())
}

func TestValidateTypeConstrainedAttributeIR_Nil(t *testing.T) {
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedAttributeIR(nil, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestValidateTypeConstrainedAttributeIR_EmptyReceiverType(t *testing.T) {
	ir := &TypeConstrainedAttributeIR{
		AttributeName: "GET",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedAttributeIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "receiverType is empty")
}

func TestValidateTypeConstrainedAttributeIR_EmptyAttributeName(t *testing.T) {
	ir := &TypeConstrainedAttributeIR{
		ReceiverType: "django.http.HttpRequest",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedAttributeIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attributeName is empty")
}

func TestValidateTypeConstrainedAttributeIR_ConfidenceClamping(t *testing.T) {
	ir := &TypeConstrainedAttributeIR{
		ReceiverType:  "test.Type",
		AttributeName: "attr",
		MinConfidence: -0.5,
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedAttributeIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, ir.MinConfidence)
	assert.True(t, dc.HasWarnings())
}

func TestValidateTypeConstrainedAttributeIR_InvalidFallback(t *testing.T) {
	ir := &TypeConstrainedAttributeIR{
		ReceiverType:  "test.Type",
		AttributeName: "attr",
		FallbackMode:  "bogus",
	}
	dc := NewDiagnosticCollector()
	err := validateTypeConstrainedAttributeIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, "name", ir.FallbackMode)
	assert.True(t, dc.HasWarnings())
}

// --- validateDataflowIR tests ---

func TestValidateDataflowIR_Valid(t *testing.T) {
	ir := &DataflowIR{
		Sources: []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Sinks:   []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Scope:   "local",
	}
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, 0, dc.Count())
}

func TestValidateDataflowIR_Nil(t *testing.T) {
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(nil, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestValidateDataflowIR_EmptySources(t *testing.T) {
	ir := &DataflowIR{
		Sinks: []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Scope: "local",
	}
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sources are empty")
}

func TestValidateDataflowIR_EmptySinks(t *testing.T) {
	ir := &DataflowIR{
		Sources: []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Scope:   "local",
	}
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sinks are empty")
}

func TestValidateDataflowIR_InvalidScope(t *testing.T) {
	ir := &DataflowIR{
		Sources: []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Sinks:   []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Scope:   "cosmic",
	}
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, "local", ir.Scope) // Defaulted.
	assert.True(t, dc.HasWarnings())
}

func TestValidateDataflowIR_EmptyScope(t *testing.T) {
	ir := &DataflowIR{
		Sources: []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Sinks:   []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
	}
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, "local", ir.Scope)
	assert.Equal(t, 0, dc.Count()) // Empty scope is silently defaulted, not a warning.
}

func TestValidateDataflowIR_GlobalScope(t *testing.T) {
	ir := &DataflowIR{
		Sources: []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Sinks:   []json.RawMessage{json.RawMessage(`{"type":"call_matcher"}`)},
		Scope:   "global",
	}
	dc := NewDiagnosticCollector()
	err := validateDataflowIR(ir, dc)
	assert.NoError(t, err)
	assert.Equal(t, "global", ir.Scope)
}

// --- precompileArgRegexes tests ---

func TestPrecompileArgRegexes_Valid(t *testing.T) {
	args := map[string]ArgumentConstraint{
		"0": {Value: "^user_.*", Comparator: "regex"},
		"1": {Value: "exact", Comparator: ""},
	}
	dc := NewDiagnosticCollector()
	compiled := precompileArgRegexes(args, dc)
	assert.Len(t, compiled, 1)
	assert.NotNil(t, compiled["0"])
	assert.Equal(t, 0, dc.Count())
}

func TestPrecompileArgRegexes_InvalidRegex(t *testing.T) {
	args := map[string]ArgumentConstraint{
		"0": {Value: "[invalid", Comparator: "regex"},
	}
	dc := NewDiagnosticCollector()
	compiled := precompileArgRegexes(args, dc)
	assert.Empty(t, compiled)
	assert.True(t, dc.HasWarnings())
	warnings := dc.FilterByLevel("warning")
	assert.Contains(t, warnings[0].Message, "invalid regex")
}

func TestPrecompileArgRegexes_NonStringRegex(t *testing.T) {
	args := map[string]ArgumentConstraint{
		"0": {Value: 123, Comparator: "regex"},
	}
	dc := NewDiagnosticCollector()
	compiled := precompileArgRegexes(args, dc)
	assert.Empty(t, compiled)
	assert.True(t, dc.HasWarnings())
	warnings := dc.FilterByLevel("warning")
	assert.Contains(t, warnings[0].Message, "not a string")
}

func TestPrecompileArgRegexes_Empty(t *testing.T) {
	dc := NewDiagnosticCollector()
	compiled := precompileArgRegexes(nil, dc)
	assert.Empty(t, compiled)
	assert.Equal(t, 0, dc.Count())
}

func TestPrecompileArgRegexes_MultipleRegexes(t *testing.T) {
	args := map[string]ArgumentConstraint{
		"0": {Value: "^start", Comparator: "regex"},
		"1": {Value: "end$", Comparator: "regex"},
		"2": {Value: "literal", Comparator: ""},
	}
	dc := NewDiagnosticCollector()
	compiled := precompileArgRegexes(args, dc)
	assert.Len(t, compiled, 2)
	assert.NotNil(t, compiled["0"])
	assert.NotNil(t, compiled["1"])
}

// --- safeExecute tests ---

func TestSafeExecute_Normal(t *testing.T) {
	dc := NewDiagnosticCollector()
	results := safeExecute(func() []DataflowDetection {
		return []DataflowDetection{
			{FunctionFQN: "test.func", SourceLine: 1, SinkLine: 2},
		}
	}, dc)

	require.Len(t, results, 1)
	assert.Equal(t, "test.func", results[0].FunctionFQN)
	assert.Equal(t, 0, dc.Count())
}

func TestSafeExecute_PanicRecovery(t *testing.T) {
	dc := NewDiagnosticCollector()
	results := safeExecute(func() []DataflowDetection {
		panic("something went wrong")
	}, dc)

	assert.Nil(t, results)
	assert.True(t, dc.HasErrors())
	errors := dc.FilterByLevel("error")
	require.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "panic recovered")
	assert.Contains(t, errors[0].Message, "something went wrong")
}

func TestSafeExecute_NilPanic(t *testing.T) {
	dc := NewDiagnosticCollector()
	results := safeExecute(func() []DataflowDetection {
		panic(nil)
	}, dc)

	assert.Nil(t, results)
	assert.True(t, dc.HasErrors())
}

func TestSafeExecute_ReturnsNil(t *testing.T) {
	dc := NewDiagnosticCollector()
	results := safeExecute(func() []DataflowDetection {
		return nil
	}, dc)

	assert.Nil(t, results)
	assert.Equal(t, 0, dc.Count())
}

// --- clampConfidence tests ---

func TestClampConfidence_InRange(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 0.5, clampConfidence(0.5, dc, "test"))
	assert.Equal(t, 0, dc.Count())
}

func TestClampConfidence_Negative(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 0.0, clampConfidence(-0.1, dc, "test"))
	assert.True(t, dc.HasWarnings())
}

func TestClampConfidence_AboveOne(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 1.0, clampConfidence(1.5, dc, "test"))
	assert.True(t, dc.HasWarnings())
}

func TestClampConfidence_NaN(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 0.0, clampConfidence(math.NaN(), dc, "test"))
	assert.True(t, dc.HasWarnings())
}

func TestClampConfidence_Inf(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 0.0, clampConfidence(math.Inf(1), dc, "test"))
	assert.True(t, dc.HasWarnings())
}

func TestClampConfidence_Zero(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 0.0, clampConfidence(0.0, dc, "test"))
	assert.Equal(t, 0, dc.Count())
}

func TestClampConfidence_One(t *testing.T) {
	dc := NewDiagnosticCollector()
	assert.Equal(t, 1.0, clampConfidence(1.0, dc, "test"))
	assert.Equal(t, 0, dc.Count())
}

// --- ClampConfidence (exported) tests ---

func TestClampConfidenceExported(t *testing.T) {
	assert.Equal(t, 0.5, ClampConfidence(0.5))
	assert.Equal(t, 0.0, ClampConfidence(-1.0))
	assert.Equal(t, 1.0, ClampConfidence(2.0))
	assert.Equal(t, 0.0, ClampConfidence(math.NaN()))
	assert.Equal(t, 0.0, ClampConfidence(math.Inf(-1)))
}

// --- AttributeMatcherIR validation tests ---

func TestValidateAttributeMatcherIR_Valid(t *testing.T) {
	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.url"},
	}
	dc := NewDiagnosticCollector()
	err := validateAttributeMatcherIR(ir, dc)
	assert.NoError(t, err)
}

func TestValidateAttributeMatcherIR_EmptyPatterns(t *testing.T) {
	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{},
	}
	dc := NewDiagnosticCollector()
	err := validateAttributeMatcherIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "patterns list is empty")
}

func TestValidateAttributeMatcherIR_EmptyStringPattern(t *testing.T) {
	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.url", ""},
	}
	dc := NewDiagnosticCollector()
	err := validateAttributeMatcherIR(ir, dc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty pattern")
}
