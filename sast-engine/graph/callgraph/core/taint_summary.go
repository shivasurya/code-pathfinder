package core

import "slices"

// TaintInfo represents detailed taint tracking information for a single detection.
type TaintInfo struct {
	// SourceLine is the line number where taint originated (1-indexed)
	SourceLine uint32

	// SourceVar is the variable name at the taint source
	SourceVar string

	// SinkLine is the line number where tainted data reaches a dangerous sink (1-indexed)
	SinkLine uint32

	// SinkVar is the variable name at the sink
	SinkVar string

	// SinkCall is the dangerous function/method call at the sink
	// Examples: "execute", "eval", "os.system"
	SinkCall string

	// PropagationPath is the list of variables through which taint propagated
	// Example: ["user_input", "data", "query"] shows user_input -> data -> query
	PropagationPath []string

	// Confidence is a score from 0.0 to 1.0 indicating detection confidence
	// 1.0 = high confidence (direct flow)
	// 0.7 = medium confidence (through stdlib function)
	// 0.5 = low confidence (through third-party library)
	// 0.0 = no taint detected
	Confidence float64

	// Sanitized indicates if a sanitizer was detected in the propagation path
	// If true, the taint was neutralized and should not trigger a finding
	Sanitized bool

	// SanitizerLine is the line number where sanitization occurred (if Sanitized == true)
	SanitizerLine uint32

	// SanitizerCall is the sanitizer function that was called
	// Examples: "escape_html", "quote_sql", "validate_email"
	SanitizerCall string
}

// IsTainted returns true if this TaintInfo represents actual taint (confidence > 0).
func (ti *TaintInfo) IsTainted() bool {
	return ti.Confidence > 0.0 && !ti.Sanitized
}

// IsHighConfidence returns true if confidence >= 0.8.
func (ti *TaintInfo) IsHighConfidence() bool {
	return ti.Confidence >= 0.8
}

// IsMediumConfidence returns true if 0.5 <= confidence < 0.8.
func (ti *TaintInfo) IsMediumConfidence() bool {
	return ti.Confidence >= 0.5 && ti.Confidence < 0.8
}

// IsLowConfidence returns true if 0.0 < confidence < 0.5.
func (ti *TaintInfo) IsLowConfidence() bool {
	return ti.Confidence > 0.0 && ti.Confidence < 0.5
}

// TaintSummary represents the complete taint analysis results for a function.
type TaintSummary struct {
	// FunctionFQN is the fully qualified name of the analyzed function
	// Format: "module.Class.method" or "module.function"
	FunctionFQN string

	// TaintedVars maps variable names to their taint information
	// If a variable is not in this map, it is considered untainted
	// Multiple TaintInfo entries indicate multiple taint paths to the same variable
	TaintedVars map[string][]*TaintInfo

	// Detections is a list of all taint flows that reached a dangerous sink
	// These represent potential security vulnerabilities
	Detections []*TaintInfo

	// TaintedParams tracks which function parameters are tainted (by parameter name)
	// Used for inter-procedural analysis
	TaintedParams []string

	// TaintedReturn indicates if the function's return value is tainted
	TaintedReturn bool

	// ReturnTaintInfo provides details if TaintedReturn is true
	ReturnTaintInfo *TaintInfo

	// AnalysisError indicates if the analysis failed for this function
	// If true, the summary is incomplete
	AnalysisError bool

	// ErrorMessage contains the error description if AnalysisError is true
	ErrorMessage string
}

// NewTaintSummary creates an empty taint summary for a function.
func NewTaintSummary(functionFQN string) *TaintSummary {
	return &TaintSummary{
		FunctionFQN:   functionFQN,
		TaintedVars:   make(map[string][]*TaintInfo),
		Detections:    make([]*TaintInfo, 0),
		TaintedParams: make([]string, 0),
	}
}

// AddTaintedVar records taint information for a variable.
func (ts *TaintSummary) AddTaintedVar(varName string, taintInfo *TaintInfo) {
	if varName == "" || taintInfo == nil {
		return
	}
	ts.TaintedVars[varName] = append(ts.TaintedVars[varName], taintInfo)
}

// GetTaintInfo retrieves all taint information for a variable.
// Returns nil if variable is not tainted.
func (ts *TaintSummary) GetTaintInfo(varName string) []*TaintInfo {
	return ts.TaintedVars[varName]
}

// IsTainted checks if a variable is tainted (has at least one unsanitized taint path).
func (ts *TaintSummary) IsTainted(varName string) bool {
	taintInfos := ts.TaintedVars[varName]
	for _, info := range taintInfos {
		if info.IsTainted() {
			return true
		}
	}
	return false
}

// AddDetection records a taint flow that reached a dangerous sink.
func (ts *TaintSummary) AddDetection(detection *TaintInfo) {
	if detection == nil {
		return
	}
	ts.Detections = append(ts.Detections, detection)
}

// HasDetections returns true if any taint flows reached dangerous sinks.
func (ts *TaintSummary) HasDetections() bool {
	return len(ts.Detections) > 0
}

// GetHighConfidenceDetections returns detections with confidence >= 0.8.
func (ts *TaintSummary) GetHighConfidenceDetections() []*TaintInfo {
	result := make([]*TaintInfo, 0)
	for _, detection := range ts.Detections {
		if detection.IsHighConfidence() {
			result = append(result, detection)
		}
	}
	return result
}

// GetMediumConfidenceDetections returns detections with 0.5 <= confidence < 0.8.
func (ts *TaintSummary) GetMediumConfidenceDetections() []*TaintInfo {
	result := make([]*TaintInfo, 0)
	for _, detection := range ts.Detections {
		if detection.IsMediumConfidence() {
			result = append(result, detection)
		}
	}
	return result
}

// GetLowConfidenceDetections returns detections with 0.0 < confidence < 0.5.
func (ts *TaintSummary) GetLowConfidenceDetections() []*TaintInfo {
	result := make([]*TaintInfo, 0)
	for _, detection := range ts.Detections {
		if detection.IsLowConfidence() {
			result = append(result, detection)
		}
	}
	return result
}

// MarkTaintedParam marks a function parameter as tainted.
func (ts *TaintSummary) MarkTaintedParam(paramName string) {
	if paramName == "" {
		return
	}

	// Check if already marked
	if slices.Contains(ts.TaintedParams, paramName) {
		return
	}

	ts.TaintedParams = append(ts.TaintedParams, paramName)
}

// IsParamTainted checks if a function parameter is tainted.
func (ts *TaintSummary) IsParamTainted(paramName string) bool {
	return slices.Contains(ts.TaintedParams, paramName)
}

// MarkReturnTainted marks the function's return value as tainted.
func (ts *TaintSummary) MarkReturnTainted(taintInfo *TaintInfo) {
	ts.TaintedReturn = true
	ts.ReturnTaintInfo = taintInfo
}

// SetError marks the analysis as failed with an error message.
func (ts *TaintSummary) SetError(errorMsg string) {
	ts.AnalysisError = true
	ts.ErrorMessage = errorMsg
}

// IsComplete returns true if analysis completed without errors.
func (ts *TaintSummary) IsComplete() bool {
	return !ts.AnalysisError
}

// GetTaintedVarCount returns the number of distinct tainted variables.
func (ts *TaintSummary) GetTaintedVarCount() int {
	count := 0
	for _, taintInfos := range ts.TaintedVars {
		for _, info := range taintInfos {
			if info.IsTainted() {
				count++
				break // Count each variable only once
			}
		}
	}
	return count
}

// GetDetectionCount returns the total number of detections.
func (ts *TaintSummary) GetDetectionCount() int {
	return len(ts.Detections)
}
