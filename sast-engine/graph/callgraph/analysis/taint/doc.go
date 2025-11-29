// Package taint provides intra-procedural taint analysis for detecting
// data flow from sources to sinks.
//
// This package implements forward data flow analysis to track taint
// propagation within a single function, identifying potential security
// vulnerabilities where untrusted input reaches sensitive operations.
//
// Example:
//
//	summary := taint.AnalyzeIntraProceduralTaint(
//	    "myapp.views.handler",
//	    statements,
//	    defUseChain,
//	    []string{"request.GET"},      // Sources
//	    []string{"eval", "exec"},      // Sinks
//	    []string{"sanitize"},          // Sanitizers
//	)
//
//	for _, detection := range summary.Detections {
//	    fmt.Printf("Taint flow detected: %s\n", detection.Variable)
//	}
package taint
