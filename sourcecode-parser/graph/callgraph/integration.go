package callgraph

import (
	"time"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
)

// InitializeCallGraph builds the call graph from a code graph.
// This integrates the 3-pass algorithm into the main initialization pipeline.
//
// Algorithm:
//  1. Build module registry from project directory
//  2. Build call graph from code graph using registry
//  3. Load default security patterns
//  4. Return integrated result
//
// Parameters:
//   - codeGraph: the parsed code graph from Initialize()
//   - projectRoot: absolute path to project root directory
//
// Returns:
//   - CallGraph: complete call graph with edges and call sites
//   - ModuleRegistry: module path mappings
//   - PatternRegistry: loaded security patterns
//   - error: if any step fails
func InitializeCallGraph(codeGraph *graph.CodeGraph, projectRoot string) (*CallGraph, *ModuleRegistry, *PatternRegistry, error) {
	// Pass 1: Build module registry
	startRegistry := time.Now()
	registry, err := BuildModuleRegistry(projectRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	elapsedRegistry := time.Since(startRegistry)

	// Pass 2-3: Build call graph (includes import extraction and call site extraction)
	startCallGraph := time.Now()
	callGraph, err := BuildCallGraph(codeGraph, registry, projectRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	elapsedCallGraph := time.Since(startCallGraph)

	// Load security patterns
	startPatterns := time.Now()
	patternRegistry := NewPatternRegistry()
	patternRegistry.LoadDefaultPatterns()
	elapsedPatterns := time.Since(startPatterns)

	// Log timing information
	graph.Log("Module registry built in:", elapsedRegistry)
	graph.Log("Call graph built in:", elapsedCallGraph)
	graph.Log("Patterns loaded in:", elapsedPatterns)

	return callGraph, registry, patternRegistry, nil
}

// AnalyzePatterns runs pattern matching against the call graph.
// Returns a list of matched patterns with their details.
func AnalyzePatterns(callGraph *CallGraph, patternRegistry *PatternRegistry) []PatternMatch {
	var matches []PatternMatch

	for _, pattern := range patternRegistry.Patterns {
		if patternRegistry.MatchPattern(pattern, callGraph) {
			matches = append(matches, PatternMatch{
				PatternID:   pattern.ID,
				PatternName: pattern.Name,
				Description: pattern.Description,
				Severity:    pattern.Severity,
				CWE:         pattern.CWE,
				OWASP:       pattern.OWASP,
			})
		}
	}

	return matches
}

// PatternMatch represents a detected security pattern in the code.
type PatternMatch struct {
	PatternID   string   // Pattern identifier
	PatternName string   // Human-readable name
	Description string   // What was detected
	Severity    Severity // Risk level
	CWE         string   // CWE identifier
	OWASP       string   // OWASP category
}
