package dsl

import (
	"fmt"
	"math"
	"regexp"
)

// validateTypeConstrainedCallIR validates a TypeConstrainedCallIR before execution.
// Returns an error if the IR is fundamentally invalid (no receiver or method).
// Non-fatal issues are emitted as diagnostics.
func validateTypeConstrainedCallIR(ir *TypeConstrainedCallIR, dc *DiagnosticCollector) error {
	if ir == nil {
		return fmt.Errorf("TypeConstrainedCallIR is nil")
	}

	receivers := ir.GetEffectiveReceiverTypes()
	if len(receivers) == 0 && len(ir.ReceiverPatterns) == 0 {
		return fmt.Errorf("type_constrained_call: receiverType/receiverTypes/receiverPatterns are all empty")
	}

	methods := ir.GetEffectiveMethodNames()
	if len(methods) == 0 {
		dc.Addf("warning", "ir_validation", "type_constrained_call: no methodName/methodNames specified, will match all methods")
	}

	ir.MinConfidence = clampConfidence(ir.MinConfidence, dc, "type_constrained_call")

	if ir.FallbackMode != "" && ir.FallbackMode != "name" && ir.FallbackMode != "none" {
		dc.Addf("warning", "ir_validation",
			"type_constrained_call: unrecognized fallbackMode %q, defaulting to \"name\"", ir.FallbackMode)
		ir.FallbackMode = "name"
	}

	return nil
}

// validateTypeConstrainedAttributeIR validates a TypeConstrainedAttributeIR before execution.
func validateTypeConstrainedAttributeIR(ir *TypeConstrainedAttributeIR, dc *DiagnosticCollector) error {
	if ir == nil {
		return fmt.Errorf("TypeConstrainedAttributeIR is nil")
	}

	if ir.ReceiverType == "" {
		return fmt.Errorf("type_constrained_attribute: receiverType is empty")
	}

	if ir.AttributeName == "" {
		return fmt.Errorf("type_constrained_attribute: attributeName is empty")
	}

	ir.MinConfidence = clampConfidence(ir.MinConfidence, dc, "type_constrained_attribute")

	if ir.FallbackMode != "" && ir.FallbackMode != "name" && ir.FallbackMode != "none" {
		dc.Addf("warning", "ir_validation",
			"type_constrained_attribute: unrecognized fallbackMode %q, defaulting to \"name\"", ir.FallbackMode)
		ir.FallbackMode = "name"
	}

	return nil
}

// validateDataflowIR validates a DataflowIR before execution.
func validateDataflowIR(ir *DataflowIR, dc *DiagnosticCollector) error {
	if ir == nil {
		return fmt.Errorf("DataflowIR is nil")
	}

	if len(ir.Sources) == 0 {
		return fmt.Errorf("dataflow: sources are empty")
	}

	if len(ir.Sinks) == 0 {
		return fmt.Errorf("dataflow: sinks are empty")
	}

	if ir.Scope != "" && ir.Scope != "local" && ir.Scope != "global" {
		dc.Addf("warning", "ir_validation",
			"dataflow: unrecognized scope %q, defaulting to \"local\"", ir.Scope)
		ir.Scope = "local"
	}

	if ir.Scope == "" {
		ir.Scope = "local"
	}

	return nil
}

// precompileArgRegexes pre-compiles regex patterns from argument constraints.
// Invalid patterns are skipped with a diagnostic warning.
func precompileArgRegexes(args map[string]ArgumentConstraint, dc *DiagnosticCollector) map[string]*regexp.Regexp {
	compiled := make(map[string]*regexp.Regexp)
	for key, constraint := range args {
		if constraint.Comparator != "regex" {
			continue
		}
		pattern, ok := constraint.Value.(string)
		if !ok {
			dc.Addf("warning", "ir_validation",
				"argument %s: regex value is not a string, skipping", key)
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			dc.Addf("warning", "ir_validation",
				"argument %s: invalid regex %q: %v", key, pattern, err)
			continue
		}
		compiled[key] = re
	}
	return compiled
}

// safeExecute wraps an executor function with panic recovery.
// If the function panics, the panic is recovered and a diagnostic is emitted.
func safeExecute(fn func() []DataflowDetection, dc *DiagnosticCollector) (results []DataflowDetection) {
	defer func() {
		if r := recover(); r != nil {
			dc.Addf("error", "executor", "panic recovered: %v", r)
			results = nil
		}
	}()
	return fn()
}

// clampConfidence clamps a confidence value to [0.0, 1.0].
// Emits a diagnostic if the value was out of range.
func clampConfidence(val float64, dc *DiagnosticCollector, component string) float64 {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		dc.Addf("warning", "ir_validation",
			"%s: confidence is NaN/Inf, clamping to 0.0", component)
		return 0.0
	}
	if val < 0.0 {
		dc.Addf("warning", "ir_validation",
			"%s: confidence %f is negative, clamping to 0.0", component, val)
		return 0.0
	}
	if val > 1.0 {
		dc.Addf("warning", "ir_validation",
			"%s: confidence %f exceeds 1.0, clamping to 1.0", component, val)
		return 1.0
	}
	return val
}

// ClampConfidence is the exported version for use in ir_types.go and other packages.
func ClampConfidence(val float64) float64 {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0.0
	}
	if val < 0.0 {
		return 0.0
	}
	if val > 1.0 {
		return 1.0
	}
	return val
}
