package taint

import (
	"slices"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// variableTaintInfo tracks taint status for a variable (internal type).
type variableTaintInfo struct {
	Source     string  // Source function that introduced taint
	Confidence float64 // Confidence level (1.0 = direct, <1.0 = propagated)
	SourceLine uint32  // Line where taint was introduced
}

// TaintState tracks taint information for all variables in a function.
type TaintState struct {
	Variables map[string]*variableTaintInfo
}

// NewTaintState creates an empty taint state.
func NewTaintState() *TaintState {
	return &TaintState{
		Variables: make(map[string]*variableTaintInfo),
	}
}

// SetTainted marks a variable as tainted.
func (ts *TaintState) SetTainted(varName, source string, confidence float64, sourceLine uint32) {
	ts.Variables[varName] = &variableTaintInfo{
		Source:     source,
		Confidence: confidence,
		SourceLine: sourceLine,
	}
}

// SetUntainted marks a variable as untainted (sanitized).
func (ts *TaintState) SetUntainted(varName string) {
	delete(ts.Variables, varName)
}

// GetTaintInfo returns taint information for a variable.
// Returns nil if variable has no taint information.
func (ts *TaintState) GetTaintInfo(varName string) *variableTaintInfo {
	return ts.Variables[varName]
}

// IsTainted returns true if the variable is tainted.
func (ts *TaintState) IsTainted(varName string) bool {
	return ts.Variables[varName] != nil
}

// AnalyzeIntraProceduralTaint performs forward taint analysis on a function.
// Returns a TaintSummary with detections of taint flows.
func AnalyzeIntraProceduralTaint(
	functionFQN string,
	statements []*core.Statement,
	defUseChain *core.DefUseChain,
	sources []string,
	sinks []string,
	sanitizers []string,
) *core.TaintSummary {
	taintState := NewTaintState()
	summary := core.NewTaintSummary(functionFQN)

	// Forward data flow analysis
	for _, stmt := range statements {
		// Check if this is a SOURCE
		if isSource(stmt, sources) {
			// Mark LHS as tainted
			if stmt.Def != "" {
				taintState.SetTainted(stmt.Def, stmt.CallTarget, 1.0, stmt.LineNumber)

				// Add to TaintedVars
				summary.AddTaintedVar(stmt.Def, &core.TaintInfo{
					SourceLine: stmt.LineNumber,
					SourceVar:  stmt.Def,
					Confidence: 1.0,
				})
			}
			continue
		}

		// Check if this is a SANITIZER
		if isSanitizer(stmt, sanitizers) {
			handleSanitizer(stmt, taintState)
			continue
		}

		// Handle ASSIGNMENT propagation
		if stmt.Type == core.StatementTypeAssignment {
			propagateAssignment(stmt, taintState, summary)
		}

		// Handle CALL propagation
		if stmt.Type == core.StatementTypeCall || stmt.CallTarget != "" {
			propagateCall(stmt, taintState, summary)
		}

		// Check if this is a SINK
		if isSink(stmt, sinks) {
			// Check if any argument is tainted
			for _, usedVar := range stmt.Uses {
				if taintInfo := taintState.GetTaintInfo(usedVar); taintInfo != nil {
					// Create detection
					detection := &core.TaintInfo{
						SourceLine: taintInfo.SourceLine,
						SourceVar:  usedVar,
						SinkLine:   stmt.LineNumber,
						SinkCall:   stmt.CallTarget,
						Confidence: taintInfo.Confidence,
					}
					summary.AddDetection(detection)
				}
			}
		}
	}

	return summary
}

// propagateAssignment propagates taint through assignments: y = x.
func propagateAssignment(stmt *core.Statement, taintState *TaintState, summary *core.TaintSummary) {
	if stmt.Def == "" {
		return
	}

	// Check if any variable in RHS (Uses) is tainted
	for _, usedVar := range stmt.Uses {
		if taintInfo := taintState.GetTaintInfo(usedVar); taintInfo != nil {
			// Propagate taint from RHS to LHS (no decay for simple assignment)
			taintState.SetTainted(stmt.Def, taintInfo.Source, taintInfo.Confidence, taintInfo.SourceLine)

			// Add to summary
			summary.AddTaintedVar(stmt.Def, &core.TaintInfo{
				SourceLine: taintInfo.SourceLine,
				SourceVar:  stmt.Def,
				Confidence: taintInfo.Confidence,
			})
			return
		}
	}
}

// propagateCall propagates taint through function calls: y = func(x).
func propagateCall(stmt *core.Statement, taintState *TaintState, summary *core.TaintSummary) {
	if stmt.Def == "" {
		return
	}

	// Check if call is a non-propagator (len, type, etc.)
	if isNonPropagator(stmt.CallTarget) {
		return
	}

	// Check if any argument is tainted
	var taintedArg *variableTaintInfo
	for _, usedVar := range stmt.Uses {
		if info := taintState.GetTaintInfo(usedVar); info != nil {
			taintedArg = info
			break
		}
	}

	if taintedArg == nil {
		return
	}

	// Determine confidence decay based on call type
	decay := 0.7 // Default: conservative propagation for stdlib/third-party

	// Propagate with decay
	newConfidence := taintedArg.Confidence * decay
	taintState.SetTainted(stmt.Def, taintedArg.Source, newConfidence, taintedArg.SourceLine)

	// Add to summary
	summary.AddTaintedVar(stmt.Def, &core.TaintInfo{
		SourceLine: taintedArg.SourceLine,
		SourceVar:  stmt.Def,
		Confidence: newConfidence,
	})
}

// handleSanitizer handles sanitizer calls (removes taint).
func handleSanitizer(stmt *core.Statement, taintState *TaintState) {
	if stmt.Def != "" {
		taintState.SetUntainted(stmt.Def)
	}
}

// isSource checks if statement is a taint source.
func isSource(stmt *core.Statement, sources []string) bool {
	if stmt.CallTarget == "" {
		return false
	}

	for _, source := range sources {
		if matchesFunctionName(stmt.CallTarget, source) {
			return true
		}
	}

	// Check hardcoded stdlib sources
	return isStdlibSource(stmt.CallTarget)
}

// isSink checks if statement is a taint sink.
func isSink(stmt *core.Statement, sinks []string) bool {
	if stmt.CallTarget == "" {
		return false
	}

	for _, sink := range sinks {
		if matchesFunctionName(stmt.CallTarget, sink) {
			return true
		}
	}

	return false
}

// isSanitizer checks if statement is a sanitizer.
func isSanitizer(stmt *core.Statement, sanitizers []string) bool {
	if stmt.CallTarget == "" {
		return false
	}

	for _, sanitizer := range sanitizers {
		if matchesFunctionName(stmt.CallTarget, sanitizer) {
			return true
		}
	}

	// Check hardcoded stdlib sanitizers
	return isStdlibSanitizer(stmt.CallTarget)
}

// Hardcoded stdlib sources (Tier 2).
var stdlibSources = map[string][]string{
	"os":     {"getenv", "environ"},
	"sys":    {"argv"},
	"socket": {"recv", "recvfrom", "recvmsg"},
}

// Hardcoded stdlib sanitizers (Tier 2).
var stdlibSanitizers = map[string][]string{
	"html":         {"escape"},
	"urllib.parse": {"quote", "quote_plus"},
	"shlex":        {"quote"},
}

// Hardcoded non-propagators (Tier 2).
var stdlibNonPropagators = map[string][]string{
	"builtins": {"len", "type", "isinstance", "hasattr", "id", "bool", "int", "str", "float"},
	"os.path":  {"exists", "isfile", "isdir", "getsize", "isabs"},
}

// isStdlibSource checks if call is a known stdlib source.
func isStdlibSource(callTarget string) bool {
	module, funcName := splitModuleFunction(callTarget)
	if sources, ok := stdlibSources[module]; ok {
		if slices.Contains(sources, funcName) {
			return true
		}
	}
	return false
}

// isStdlibSanitizer checks if call is a known stdlib sanitizer.
func isStdlibSanitizer(callTarget string) bool {
	module, funcName := splitModuleFunction(callTarget)
	if sanitizers, ok := stdlibSanitizers[module]; ok {
		if slices.Contains(sanitizers, funcName) {
			return true
		}
	}
	return false
}

// isNonPropagator checks if function doesn't propagate taint.
func isNonPropagator(callTarget string) bool {
	module, funcName := splitModuleFunction(callTarget)

	// Check exact module.function match
	if funcs, ok := stdlibNonPropagators[module]; ok {
		if slices.Contains(funcs, funcName) {
			return true
		}
	}

	// Check builtins (no module prefix)
	if module == "" {
		if funcs, ok := stdlibNonPropagators["builtins"]; ok {
			if slices.Contains(funcs, callTarget) {
				return true
			}
		}
	}

	return false
}

// splitModuleFunction splits "os.path.join" into ("os.path", "join").
func splitModuleFunction(callTarget string) (module, function string) {
	lastDot := strings.LastIndex(callTarget, ".")
	if lastDot == -1 {
		return "", callTarget // No module (builtin)
	}
	return callTarget[:lastDot], callTarget[lastDot+1:]
}

// matchesFunctionName checks if a call target matches a function name pattern.
// Supports exact matches, suffix matches (e.g., "builtins.eval" matches "eval"),
// and handles parentheses (e.g., "input()" matches "input").
func matchesFunctionName(callTarget, pattern string) bool {
	// Strip parentheses from call target if present
	cleanTarget := callTarget
	if before, _, ok := strings.Cut(callTarget, "("); ok {
		cleanTarget = before
	}

	// Exact match: "eval" == "eval"
	if cleanTarget == pattern {
		return true
	}

	// Suffix match: "builtins.eval" ends with ".eval"
	if strings.HasSuffix(cleanTarget, "."+pattern) {
		return true
	}

	// Prefix match: "request.GET.get" starts with "request.GET."
	if strings.HasPrefix(cleanTarget, pattern+".") {
		return true
	}

	// Extract last component and compare
	lastDot := strings.LastIndex(cleanTarget, ".")
	if lastDot >= 0 && lastDot < len(cleanTarget)-1 {
		lastComponent := cleanTarget[lastDot+1:]
		if lastComponent == pattern {
			return true
		}
	}

	return false
}
