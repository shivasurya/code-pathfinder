package callgraph

import (
	"strings"
)

// PatternType categorizes security patterns for analysis.
type PatternType string

const (
	// PatternTypeSourceSink detects tainted data flow from source to sink.
	PatternTypeSourceSink PatternType = "source-sink"

	// PatternTypeMissingSanitizer detects missing sanitization between source and sink.
	PatternTypeMissingSanitizer PatternType = "missing-sanitizer"

	// PatternTypeDangerousFunction detects calls to dangerous functions.
	PatternTypeDangerousFunction PatternType = "dangerous-function"
)

// Severity indicates the risk level of a security pattern match.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Pattern represents a security pattern to detect in the call graph.
type Pattern struct {
	ID          string      // Unique identifier (e.g., "SQL-INJECTION-001")
	Name        string      // Human-readable name
	Description string      // What this pattern detects
	Type        PatternType // Pattern category
	Severity    Severity    // Risk level

	// Sources are function names that introduce tainted data
	Sources []string

	// Sinks are function names that consume tainted data dangerously
	Sinks []string

	// Sanitizers are function names that clean tainted data
	Sanitizers []string

	// DangerousFunctions for PatternTypeDangerousFunction
	DangerousFunctions []string

	CWE   string // Common Weakness Enumeration
	OWASP string // OWASP Top 10 category
}

// PatternRegistry manages security patterns.
type PatternRegistry struct {
	Patterns       map[string]*Pattern        // Pattern ID -> Pattern
	PatternsByType map[PatternType][]*Pattern // Type -> Patterns
}

// NewPatternRegistry creates a new pattern registry.
func NewPatternRegistry() *PatternRegistry {
	return &PatternRegistry{
		Patterns:       make(map[string]*Pattern),
		PatternsByType: make(map[PatternType][]*Pattern),
	}
}

// AddPattern registers a pattern in the registry.
func (pr *PatternRegistry) AddPattern(pattern *Pattern) {
	pr.Patterns[pattern.ID] = pattern
	pr.PatternsByType[pattern.Type] = append(pr.PatternsByType[pattern.Type], pattern)
}

// GetPattern retrieves a pattern by ID.
func (pr *PatternRegistry) GetPattern(id string) (*Pattern, bool) {
	pattern, exists := pr.Patterns[id]
	return pattern, exists
}

// GetPatternsByType retrieves all patterns of a specific type.
func (pr *PatternRegistry) GetPatternsByType(patternType PatternType) []*Pattern {
	return pr.PatternsByType[patternType]
}

// LoadDefaultPatterns loads the hardcoded example pattern.
// Additional patterns will be loaded from queries in future PRs.
func (pr *PatternRegistry) LoadDefaultPatterns() {
	// Example hardcoded pattern: Code injection via eval()
	pr.AddPattern(&Pattern{
		ID:          "CODE-INJECTION-001",
		Name:        "Code injection via eval with user input",
		Description: "Detects code injection when user input flows to eval() without sanitization",
		Type:        PatternTypeMissingSanitizer,
		Severity:    SeverityCritical,
		Sources:     []string{"request.GET", "request.POST", "input", "raw_input", "request.query_params.get"},
		Sinks:       []string{"eval", "exec"},
		Sanitizers:  []string{"sanitize", "escape", "validate"},
		CWE:         "CWE-94",
		OWASP:       "A03:2021-Injection",
	})
}

// MatchPattern checks if a call graph matches a pattern.
// Returns detailed match information if a vulnerability is found.
func (pr *PatternRegistry) MatchPattern(pattern *Pattern, callGraph *CallGraph) *PatternMatchDetails {
	switch pattern.Type {
	case PatternTypeDangerousFunction:
		return pr.matchDangerousFunction(pattern, callGraph)
	case PatternTypeSourceSink:
		return pr.matchSourceSink(pattern, callGraph)
	case PatternTypeMissingSanitizer:
		return pr.matchMissingSanitizer(pattern, callGraph)
	default:
		return nil
	}
}

// PatternMatchDetails contains detailed information about a pattern match.
type PatternMatchDetails struct {
	Matched      bool
	SourceFQN    string   // Fully qualified name of function containing the source call
	SourceCall   string   // The actual dangerous call (e.g., "input", "request.GET")
	SinkFQN      string   // Fully qualified name of function containing the sink call
	SinkCall     string   // The actual dangerous call (e.g., "eval", "exec")
	DataFlowPath []string // Complete path from source to sink
}

// matchDangerousFunction checks if any dangerous function is called.
func (pr *PatternRegistry) matchDangerousFunction(pattern *Pattern, callGraph *CallGraph) *PatternMatchDetails {
	for caller, callSites := range callGraph.CallSites {
		for _, callSite := range callSites {
			for _, dangerousFunc := range pattern.DangerousFunctions {
				if matchesFunctionName(callSite.TargetFQN, dangerousFunc) ||
					matchesFunctionName(callSite.Target, dangerousFunc) {
					return &PatternMatchDetails{
						Matched:      true,
						SourceFQN:    caller,
						SinkFQN:      callSite.TargetFQN,
						DataFlowPath: []string{caller, callSite.TargetFQN},
					}
				}
			}
		}
	}
	return &PatternMatchDetails{Matched: false}
}

// matchSourceSink checks if there's a path from source to sink.
func (pr *PatternRegistry) matchSourceSink(pattern *Pattern, callGraph *CallGraph) *PatternMatchDetails {
	sourceCalls := pr.findCallsByFunctions(pattern.Sources, callGraph)
	if len(sourceCalls) == 0 {
		return &PatternMatchDetails{Matched: false}
	}

	sinkCalls := pr.findCallsByFunctions(pattern.Sinks, callGraph)
	if len(sinkCalls) == 0 {
		return &PatternMatchDetails{Matched: false}
	}

	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			path := pr.findPath(source.caller, sink.caller, callGraph)
			if len(path) > 0 {
				return &PatternMatchDetails{
					Matched:      true,
					SourceFQN:    source.caller,
					SinkFQN:      sink.caller,
					DataFlowPath: path,
				}
			}
		}
	}

	return &PatternMatchDetails{Matched: false}
}

// matchMissingSanitizer checks if there's a path from source to sink without sanitization.
func (pr *PatternRegistry) matchMissingSanitizer(pattern *Pattern, callGraph *CallGraph) *PatternMatchDetails {
	sourceCalls := pr.findCallsByFunctions(pattern.Sources, callGraph)
	if len(sourceCalls) == 0 {
		return &PatternMatchDetails{Matched: false}
	}

	sinkCalls := pr.findCallsByFunctions(pattern.Sinks, callGraph)
	if len(sinkCalls) == 0 {
		return &PatternMatchDetails{Matched: false}
	}

	sanitizerCalls := pr.findCallsByFunctions(pattern.Sanitizers, callGraph)

	// Sort for deterministic results
	sortCallInfo(sourceCalls)
	sortCallInfo(sinkCalls)

	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			// Skip false positives where source and sink are in the same function
			if source.caller == sink.caller {
				continue
			}

			path := pr.findPath(source.caller, sink.caller, callGraph)
			if len(path) > 1 { // Require at least 2 functions in path
				// Check if any sanitizer is on the path
				hasSanitizer := false
				for _, sanitizer := range sanitizerCalls {
					// Check if sanitizer is in the path
					for _, pathFunc := range path {
						if pathFunc == sanitizer.caller {
							hasSanitizer = true
							break
						}
					}
					if hasSanitizer {
						break
					}
				}
				if !hasSanitizer {
					return &PatternMatchDetails{
						Matched:      true,
						SourceFQN:    source.caller,
						SourceCall:   source.target,
						SinkFQN:      sink.caller,
						SinkCall:     sink.target,
						DataFlowPath: path,
					}
				}
			}
		}
	}

	return &PatternMatchDetails{Matched: false}
}

// callInfo stores information about a function call location.
type callInfo struct {
	caller string
	target string
}

// findCallsByFunctions finds all calls to specific functions.
func (pr *PatternRegistry) findCallsByFunctions(functionNames []string, callGraph *CallGraph) []callInfo {
	var calls []callInfo
	for caller, callSites := range callGraph.CallSites {
		for _, callSite := range callSites {
			for _, funcName := range functionNames {
				if matchesFunctionName(callSite.TargetFQN, funcName) ||
					matchesFunctionName(callSite.Target, funcName) {
					calls = append(calls, callInfo{caller: caller, target: callSite.TargetFQN})
				}
			}
		}
	}
	return calls
}

// hasPath checks if there's a path from caller to callee in the call graph.
func (pr *PatternRegistry) hasPath(from, to string, callGraph *CallGraph) bool {
	if from == to {
		return true
	}

	visited := make(map[string]bool)
	return pr.dfsPath(from, to, callGraph, visited)
}

// dfsPath performs depth-first search to find a path.
func (pr *PatternRegistry) dfsPath(current, target string, callGraph *CallGraph, visited map[string]bool) bool {
	if current == target {
		return true
	}

	if visited[current] {
		return false
	}

	visited[current] = true

	callees := callGraph.GetCallees(current)
	for _, callee := range callees {
		if pr.dfsPath(callee, target, callGraph, visited) {
			return true
		}
	}

	return false
}

// findPath finds the complete path from source to sink in the call graph.
// Returns the path as a slice of function FQNs, or empty slice if no path exists.
func (pr *PatternRegistry) findPath(from, to string, callGraph *CallGraph) []string {
	if from == to {
		return []string{from}
	}

	visited := make(map[string]bool)
	path := make([]string, 0)

	if pr.dfsPathWithTrace(from, to, callGraph, visited, &path) {
		return path
	}

	return []string{}
}

// dfsPathWithTrace performs depth-first search and captures the path.
func (pr *PatternRegistry) dfsPathWithTrace(current, target string, callGraph *CallGraph, visited map[string]bool, path *[]string) bool {
	*path = append(*path, current)

	if current == target {
		return true
	}

	if visited[current] {
		*path = (*path)[:len(*path)-1] // backtrack
		return false
	}

	visited[current] = true

	callees := callGraph.GetCallees(current)
	for _, callee := range callees {
		if pr.dfsPathWithTrace(callee, target, callGraph, visited, path) {
			return true
		}
	}

	// Backtrack if no path found
	*path = (*path)[:len(*path)-1]
	return false
}

// sortCallInfo sorts callInfo slices by caller FQN for deterministic results.
func sortCallInfo(calls []callInfo) {
	// Simple bubble sort - good enough for small slices
	for i := 0; i < len(calls); i++ {
		for j := i + 1; j < len(calls); j++ {
			if calls[i].caller > calls[j].caller {
				calls[i], calls[j] = calls[j], calls[i]
			}
		}
	}
}

// matchesFunctionName checks if a function name matches a pattern.
// Supports exact matches, suffix matches, and prefix matches.
// Examples:
//   - "builtins.eval" matches pattern "eval" (suffix match)
//   - "request.GET.get" matches pattern "request.GET" (prefix match for sources)
//   - "vulnerable_app.eval" matches pattern "eval" (last component match)
func matchesFunctionName(fqn, pattern string) bool {
	// Exact match: "eval" == "eval"
	if fqn == pattern {
		return true
	}

	// Suffix match: "builtins.eval" ends with ".eval"
	if strings.HasSuffix(fqn, "."+pattern) {
		return true
	}

	// Prefix match: "request.GET.get" starts with "request.GET."
	// This handles attribute access chains for sources
	if strings.HasPrefix(fqn, pattern+".") {
		return true
	}

	// Extract last component after last dot and compare
	// This handles cases like "vulnerable_app.eval" → "eval"
	// but avoids matching "executor" against "exec"
	lastDot := strings.LastIndex(fqn, ".")
	if lastDot >= 0 && lastDot < len(fqn)-1 {
		lastComponent := fqn[lastDot+1:]
		if lastComponent == pattern {
			return true
		}
	}

	return false
}
