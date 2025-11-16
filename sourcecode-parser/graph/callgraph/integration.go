package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/patterns"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
)

// SecurityMatch represents a detected security vulnerability.
type SecurityMatch struct {
	Severity      string   // "critical", "high", "medium", "low"
	PatternName   string   // Name of the security pattern
	Description   string   // Description of the vulnerability
	CWE           string   // CWE ID (e.g., "CWE-89")
	OWASP         string   // OWASP category (e.g., "A03:2021")
	SourceFQN     string   // Fully qualified name of source function
	SourceCall    string   // The source call name
	SourceFile    string   // Source file path
	SourceLine    uint32   // Source line number
	SourceCode    string   // Source code snippet
	SinkFQN       string   // Fully qualified name of sink function
	SinkCall      string   // The sink call name
	SinkFile      string   // Sink file path
	SinkLine      uint32   // Sink line number
	SinkCode      string   // Sink code snippet
	DataFlowPath  []string // Path from source to sink
}

// InitializeCallGraph builds a complete call graph with all analysis components.
// It returns the call graph, module registry, pattern registry, and any error.
//
// This is a convenience function that orchestrates:
//  1. Module registry building
//  2. Call graph construction
//  3. Pattern registry initialization
//
// Parameters:
//   - codeGraph: the parsed code graph from graph.Initialize()
//   - projectPath: absolute path to project root
//
// Returns:
//   - CallGraph: complete call graph with edges and call sites
//   - ModuleRegistry: module path mappings
//   - PatternRegistry: security patterns for analysis
//   - error: if any step fails
func InitializeCallGraph(codeGraph *graph.CodeGraph, projectPath string) (*core.CallGraph, *core.ModuleRegistry, *patterns.PatternRegistry, error) {
	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(projectPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build call graph
	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// Initialize pattern registry
	patternRegistry := patterns.NewPatternRegistry()
	patternRegistry.LoadDefaultPatterns()

	return callGraph, moduleRegistry, patternRegistry, nil
}

// AnalyzePatterns detects security vulnerabilities using the pattern registry.
// It analyzes the call graph against all loaded security patterns.
//
// Parameters:
//   - callGraph: the call graph to analyze
//   - patternRegistry: security patterns to check
//
// Returns:
//   - []SecurityMatch: list of detected security issues
func AnalyzePatterns(callGraph *core.CallGraph, patternRegistry *patterns.PatternRegistry) []SecurityMatch {
	var matches []SecurityMatch

	// Check each pattern type
	patternTypes := []patterns.PatternType{
		patterns.PatternTypeSourceSink,
		patterns.PatternTypeMissingSanitizer,
		patterns.PatternTypeDangerousFunction,
	}

	for _, patternType := range patternTypes {
		// Get patterns of this type
		patternsOfType := patternRegistry.GetPatternsByType(patternType)

		// Check each pattern against the call graph
		for _, pattern := range patternsOfType {
			match := patternRegistry.MatchPattern(pattern, callGraph)
			if match.Matched {
				// Convert PatternMatchDetails to SecurityMatch
				securityMatch := SecurityMatch{
					Severity:     string(pattern.Severity),
					PatternName:  pattern.Name,
					Description:  pattern.Description,
					CWE:          pattern.CWE,
					OWASP:        pattern.OWASP,
					SourceFQN:    match.SourceFQN,
					SourceCall:   match.SourceCall,
					SinkFQN:      match.SinkFQN,
					SinkCall:     match.SinkCall,
					DataFlowPath: match.DataFlowPath,
				}
				matches = append(matches, securityMatch)
			}
		}
	}

	return matches
}
