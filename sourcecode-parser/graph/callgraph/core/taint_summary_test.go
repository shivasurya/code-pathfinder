package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaintInfoIsTainted(t *testing.T) {
	tests := []struct {
		name     string
		info     *TaintInfo
		expected bool
	}{
		{
			name: "high confidence taint",
			info: &TaintInfo{
				Confidence: 1.0,
				Sanitized:  false,
			},
			expected: true,
		},
		{
			name: "medium confidence taint",
			info: &TaintInfo{
				Confidence: 0.7,
				Sanitized:  false,
			},
			expected: true,
		},
		{
			name: "sanitized taint",
			info: &TaintInfo{
				Confidence: 1.0,
				Sanitized:  true,
			},
			expected: false,
		},
		{
			name: "zero confidence",
			info: &TaintInfo{
				Confidence: 0.0,
				Sanitized:  false,
			},
			expected: false,
		},
		{
			name: "low confidence but not sanitized",
			info: &TaintInfo{
				Confidence: 0.3,
				Sanitized:  false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.IsTainted()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaintInfoIsHighConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   bool
	}{
		{name: "perfect confidence", confidence: 1.0, expected: true},
		{name: "high confidence", confidence: 0.9, expected: true},
		{name: "exactly 0.8", confidence: 0.8, expected: true},
		{name: "just below threshold", confidence: 0.79, expected: false},
		{name: "medium confidence", confidence: 0.6, expected: false},
		{name: "low confidence", confidence: 0.3, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &TaintInfo{Confidence: tt.confidence}
			result := info.IsHighConfidence()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaintInfoIsMediumConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   bool
	}{
		{name: "high confidence", confidence: 1.0, expected: false},
		{name: "just below high", confidence: 0.79, expected: true},
		{name: "mid range", confidence: 0.6, expected: true},
		{name: "exactly 0.5", confidence: 0.5, expected: true},
		{name: "just below medium", confidence: 0.49, expected: false},
		{name: "low confidence", confidence: 0.3, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &TaintInfo{Confidence: tt.confidence}
			result := info.IsMediumConfidence()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaintInfoIsLowConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   bool
	}{
		{name: "medium confidence", confidence: 0.6, expected: false},
		{name: "just below medium", confidence: 0.49, expected: true},
		{name: "low confidence", confidence: 0.3, expected: true},
		{name: "very low", confidence: 0.1, expected: true},
		{name: "zero confidence", confidence: 0.0, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &TaintInfo{Confidence: tt.confidence}
			result := info.IsLowConfidence()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTaintSummary(t *testing.T) {
	summary := NewTaintSummary("module.Class.method")

	assert.Equal(t, "module.Class.method", summary.FunctionFQN)
	assert.NotNil(t, summary.TaintedVars)
	assert.Equal(t, 0, len(summary.TaintedVars))
	assert.NotNil(t, summary.Detections)
	assert.Equal(t, 0, len(summary.Detections))
	assert.NotNil(t, summary.TaintedParams)
	assert.Equal(t, 0, len(summary.TaintedParams))
	assert.False(t, summary.TaintedReturn)
	assert.Nil(t, summary.ReturnTaintInfo)
	assert.False(t, summary.AnalysisError)
	assert.Equal(t, "", summary.ErrorMessage)
}

func TestTaintSummaryAddTaintedVar(t *testing.T) {
	summary := NewTaintSummary("test.function")

	taint1 := &TaintInfo{
		SourceLine: 1,
		SourceVar:  "input",
		Confidence: 1.0,
	}

	taint2 := &TaintInfo{
		SourceLine: 2,
		SourceVar:  "input2",
		Confidence: 0.7,
	}

	// Add first taint
	summary.AddTaintedVar("x", taint1)
	assert.Equal(t, 1, len(summary.TaintedVars["x"]))
	assert.Equal(t, taint1, summary.TaintedVars["x"][0])

	// Add second taint to same variable
	summary.AddTaintedVar("x", taint2)
	assert.Equal(t, 2, len(summary.TaintedVars["x"]))
	assert.Equal(t, taint2, summary.TaintedVars["x"][1])

	// Test empty variable name (should be ignored)
	summary.AddTaintedVar("", taint1)
	_, exists := summary.TaintedVars[""]
	assert.False(t, exists)

	// Test nil taint info (should be ignored)
	summary.AddTaintedVar("y", nil)
	_, exists = summary.TaintedVars["y"]
	assert.False(t, exists)
}

func TestTaintSummaryGetTaintInfo(t *testing.T) {
	summary := NewTaintSummary("test.function")

	taint := &TaintInfo{
		SourceLine: 1,
		SourceVar:  "input",
		Confidence: 1.0,
	}

	summary.AddTaintedVar("x", taint)

	// Get existing taint
	result := summary.GetTaintInfo("x")
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, taint, result[0])

	// Get non-existent variable
	nonExistent := summary.GetTaintInfo("nonexistent")
	assert.Nil(t, nonExistent)
}

func TestTaintSummaryIsTainted(t *testing.T) {
	summary := NewTaintSummary("test.function")

	// Add tainted variable
	taint1 := &TaintInfo{
		Confidence: 1.0,
		Sanitized:  false,
	}
	summary.AddTaintedVar("x", taint1)
	assert.True(t, summary.IsTainted("x"))

	// Add sanitized taint
	taint2 := &TaintInfo{
		Confidence: 1.0,
		Sanitized:  true,
	}
	summary.AddTaintedVar("y", taint2)
	assert.False(t, summary.IsTainted("y"))

	// Add variable with both tainted and sanitized paths
	summary.AddTaintedVar("z", taint1) // tainted
	summary.AddTaintedVar("z", taint2) // sanitized
	assert.True(t, summary.IsTainted("z")) // Should return true if ANY path is tainted

	// Check non-existent variable
	assert.False(t, summary.IsTainted("nonexistent"))
}

func TestTaintSummaryAddDetection(t *testing.T) {
	summary := NewTaintSummary("test.function")

	detection1 := &TaintInfo{
		SourceLine: 1,
		SinkLine:   5,
		SinkCall:   "execute",
		Confidence: 1.0,
	}

	detection2 := &TaintInfo{
		SourceLine: 2,
		SinkLine:   6,
		SinkCall:   "eval",
		Confidence: 0.8,
	}

	summary.AddDetection(detection1)
	assert.Equal(t, 1, len(summary.Detections))
	assert.Equal(t, detection1, summary.Detections[0])

	summary.AddDetection(detection2)
	assert.Equal(t, 2, len(summary.Detections))
	assert.Equal(t, detection2, summary.Detections[1])

	// Test nil detection (should be ignored)
	summary.AddDetection(nil)
	assert.Equal(t, 2, len(summary.Detections))
}

func TestTaintSummaryHasDetections(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.False(t, summary.HasDetections())

	detection := &TaintInfo{
		SourceLine: 1,
		SinkLine:   5,
		Confidence: 1.0,
	}
	summary.AddDetection(detection)

	assert.True(t, summary.HasDetections())
}

func TestTaintSummaryGetHighConfidenceDetections(t *testing.T) {
	summary := NewTaintSummary("test.function")

	high1 := &TaintInfo{Confidence: 1.0}
	high2 := &TaintInfo{Confidence: 0.9}
	medium := &TaintInfo{Confidence: 0.6}
	low := &TaintInfo{Confidence: 0.3}

	summary.AddDetection(high1)
	summary.AddDetection(medium)
	summary.AddDetection(high2)
	summary.AddDetection(low)

	highConf := summary.GetHighConfidenceDetections()
	assert.Equal(t, 2, len(highConf))
	assert.Equal(t, high1, highConf[0])
	assert.Equal(t, high2, highConf[1])
}

func TestTaintSummaryGetMediumConfidenceDetections(t *testing.T) {
	summary := NewTaintSummary("test.function")

	high := &TaintInfo{Confidence: 1.0}
	medium1 := &TaintInfo{Confidence: 0.7}
	medium2 := &TaintInfo{Confidence: 0.5}
	low := &TaintInfo{Confidence: 0.3}

	summary.AddDetection(high)
	summary.AddDetection(medium1)
	summary.AddDetection(low)
	summary.AddDetection(medium2)

	mediumConf := summary.GetMediumConfidenceDetections()
	assert.Equal(t, 2, len(mediumConf))
	assert.Equal(t, medium1, mediumConf[0])
	assert.Equal(t, medium2, mediumConf[1])
}

func TestTaintSummaryGetLowConfidenceDetections(t *testing.T) {
	summary := NewTaintSummary("test.function")

	high := &TaintInfo{Confidence: 1.0}
	medium := &TaintInfo{Confidence: 0.6}
	low1 := &TaintInfo{Confidence: 0.4}
	low2 := &TaintInfo{Confidence: 0.1}

	summary.AddDetection(high)
	summary.AddDetection(low1)
	summary.AddDetection(medium)
	summary.AddDetection(low2)

	lowConf := summary.GetLowConfidenceDetections()
	assert.Equal(t, 2, len(lowConf))
	assert.Equal(t, low1, lowConf[0])
	assert.Equal(t, low2, lowConf[1])
}

func TestTaintSummaryMarkTaintedParam(t *testing.T) {
	summary := NewTaintSummary("test.function")

	// Mark first param
	summary.MarkTaintedParam("param1")
	assert.Equal(t, 1, len(summary.TaintedParams))
	assert.Equal(t, "param1", summary.TaintedParams[0])

	// Mark second param
	summary.MarkTaintedParam("param2")
	assert.Equal(t, 2, len(summary.TaintedParams))
	assert.Equal(t, "param2", summary.TaintedParams[1])

	// Try to mark same param again (should not duplicate)
	summary.MarkTaintedParam("param1")
	assert.Equal(t, 2, len(summary.TaintedParams))

	// Test empty param name (should be ignored)
	summary.MarkTaintedParam("")
	assert.Equal(t, 2, len(summary.TaintedParams))
}

func TestTaintSummaryIsParamTainted(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.False(t, summary.IsParamTainted("param1"))

	summary.MarkTaintedParam("param1")
	assert.True(t, summary.IsParamTainted("param1"))
	assert.False(t, summary.IsParamTainted("param2"))

	summary.MarkTaintedParam("param2")
	assert.True(t, summary.IsParamTainted("param2"))
}

func TestTaintSummaryMarkReturnTainted(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.False(t, summary.TaintedReturn)
	assert.Nil(t, summary.ReturnTaintInfo)

	taint := &TaintInfo{
		SourceLine: 1,
		Confidence: 1.0,
	}

	summary.MarkReturnTainted(taint)
	assert.True(t, summary.TaintedReturn)
	assert.Equal(t, taint, summary.ReturnTaintInfo)
}

func TestTaintSummarySetError(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.False(t, summary.AnalysisError)
	assert.Equal(t, "", summary.ErrorMessage)

	summary.SetError("parse error")
	assert.True(t, summary.AnalysisError)
	assert.Equal(t, "parse error", summary.ErrorMessage)
}

func TestTaintSummaryIsComplete(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.True(t, summary.IsComplete())

	summary.SetError("error")
	assert.False(t, summary.IsComplete())
}

func TestTaintSummaryGetTaintedVarCount(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.Equal(t, 0, summary.GetTaintedVarCount())

	// Add tainted variable
	taint1 := &TaintInfo{Confidence: 1.0, Sanitized: false}
	summary.AddTaintedVar("x", taint1)
	assert.Equal(t, 1, summary.GetTaintedVarCount())

	// Add another taint to same variable (should still count as 1)
	taint2 := &TaintInfo{Confidence: 0.7, Sanitized: false}
	summary.AddTaintedVar("x", taint2)
	assert.Equal(t, 1, summary.GetTaintedVarCount())

	// Add tainted second variable
	summary.AddTaintedVar("y", taint1)
	assert.Equal(t, 2, summary.GetTaintedVarCount())

	// Add sanitized variable (should not count)
	sanitized := &TaintInfo{Confidence: 1.0, Sanitized: true}
	summary.AddTaintedVar("z", sanitized)
	assert.Equal(t, 2, summary.GetTaintedVarCount())
}

func TestTaintSummaryGetDetectionCount(t *testing.T) {
	summary := NewTaintSummary("test.function")

	assert.Equal(t, 0, summary.GetDetectionCount())

	detection1 := &TaintInfo{Confidence: 1.0}
	summary.AddDetection(detection1)
	assert.Equal(t, 1, summary.GetDetectionCount())

	detection2 := &TaintInfo{Confidence: 0.8}
	summary.AddDetection(detection2)
	assert.Equal(t, 2, summary.GetDetectionCount())
}

func TestTaintSummaryComplexScenario(t *testing.T) {
	// Simulate a real security finding scenario
	summary := NewTaintSummary("app.views.process_payment")

	// Taint flows from user input
	userInputTaint := &TaintInfo{
		SourceLine:      10,
		SourceVar:       "request.GET['amount']",
		SinkLine:        25,
		SinkVar:         "query",
		SinkCall:        "cursor.execute",
		PropagationPath: []string{"user_amount", "amount", "query"},
		Confidence:      1.0,
		Sanitized:       false,
	}

	// Track the variable propagation
	summary.AddTaintedVar("user_amount", &TaintInfo{
		SourceLine: 10,
		SourceVar:  "request.GET['amount']",
		Confidence: 1.0,
	})

	summary.AddTaintedVar("amount", &TaintInfo{
		SourceLine: 15,
		SourceVar:  "user_amount",
		Confidence: 1.0,
	})

	summary.AddTaintedVar("query", &TaintInfo{
		SourceLine: 20,
		SourceVar:  "amount",
		Confidence: 1.0,
	})

	// Record the detection
	summary.AddDetection(userInputTaint)

	// Mark the request parameter as tainted
	summary.MarkTaintedParam("request")

	// Verify the summary
	assert.True(t, summary.IsTainted("user_amount"))
	assert.True(t, summary.IsTainted("amount"))
	assert.True(t, summary.IsTainted("query"))
	assert.Equal(t, 3, summary.GetTaintedVarCount())
	assert.True(t, summary.HasDetections())
	assert.Equal(t, 1, summary.GetDetectionCount())
	assert.Equal(t, 1, len(summary.GetHighConfidenceDetections()))
	assert.True(t, summary.IsParamTainted("request"))
	assert.True(t, summary.IsComplete())

	// Verify the detection details
	detection := summary.Detections[0]
	assert.Equal(t, uint32(10), detection.SourceLine)
	assert.Equal(t, uint32(25), detection.SinkLine)
	assert.Equal(t, "cursor.execute", detection.SinkCall)
	assert.Equal(t, 3, len(detection.PropagationPath))
	assert.True(t, detection.IsHighConfidence())
	assert.False(t, detection.Sanitized)
}
