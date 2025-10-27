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
	Patterns       map[string]*Pattern            // Pattern ID -> Pattern
	PatternsByType map[PatternType][]*Pattern     // Type -> Patterns
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
		Sources:     []string{"request.GET", "request.POST", "input", "raw_input"},
		Sinks:       []string{"eval", "exec"},
		Sanitizers:  []string{"sanitize", "escape", "validate"},
		CWE:         "CWE-94",
		OWASP:       "A03:2021-Injection",
	})
}

// MatchPattern checks if a call graph matches a pattern.
func (pr *PatternRegistry) MatchPattern(pattern *Pattern, callGraph *CallGraph) bool {
	switch pattern.Type {
	case PatternTypeDangerousFunction:
		return pr.matchDangerousFunction(pattern, callGraph)
	case PatternTypeSourceSink:
		return pr.matchSourceSink(pattern, callGraph)
	case PatternTypeMissingSanitizer:
		return pr.matchMissingSanitizer(pattern, callGraph)
	default:
		return false
	}
}

// matchDangerousFunction checks if any dangerous function is called.
func (pr *PatternRegistry) matchDangerousFunction(pattern *Pattern, callGraph *CallGraph) bool {
	for _, callSites := range callGraph.CallSites {
		for _, callSite := range callSites {
			for _, dangerousFunc := range pattern.DangerousFunctions {
				if matchesFunctionName(callSite.TargetFQN, dangerousFunc) ||
					matchesFunctionName(callSite.Target, dangerousFunc) {
					return true
				}
			}
		}
	}
	return false
}

// matchSourceSink checks if there's a path from source to sink.
func (pr *PatternRegistry) matchSourceSink(pattern *Pattern, callGraph *CallGraph) bool {
	sourceCalls := pr.findCallsByFunctions(pattern.Sources, callGraph)
	if len(sourceCalls) == 0 {
		return false
	}

	sinkCalls := pr.findCallsByFunctions(pattern.Sinks, callGraph)
	if len(sinkCalls) == 0 {
		return false
	}

	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			if pr.hasPath(source.caller, sink.caller, callGraph) {
				return true
			}
		}
	}

	return false
}

// matchMissingSanitizer checks if there's a path from source to sink without sanitization.
func (pr *PatternRegistry) matchMissingSanitizer(pattern *Pattern, callGraph *CallGraph) bool {
	sourceCalls := pr.findCallsByFunctions(pattern.Sources, callGraph)
	if len(sourceCalls) == 0 {
		return false
	}

	sinkCalls := pr.findCallsByFunctions(pattern.Sinks, callGraph)
	if len(sinkCalls) == 0 {
		return false
	}

	sanitizerCalls := pr.findCallsByFunctions(pattern.Sanitizers, callGraph)

	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			if pr.hasPath(source.caller, sink.caller, callGraph) {
				hasSanitizer := false
				for _, sanitizer := range sanitizerCalls {
					if pr.hasPath(source.caller, sanitizer.caller, callGraph) &&
						pr.hasPath(sanitizer.caller, sink.caller, callGraph) {
						hasSanitizer = true
						break
					}
				}
				if !hasSanitizer {
					return true
				}
			}
		}
	}

	return false
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

// matchesFunctionName checks if a function name matches a pattern.
// Supports exact matches and suffix matches.
func matchesFunctionName(fqn, pattern string) bool {
	if fqn == pattern {
		return true
	}

	if strings.HasSuffix(fqn, "."+pattern) {
		return true
	}

	if strings.Contains(fqn, pattern) {
		return true
	}

	return false
}
