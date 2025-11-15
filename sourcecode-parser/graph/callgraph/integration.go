package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/patterns"
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
	// Use builder package for call graph construction
	callGraph, registry, err := builder.BuildCallGraphFromPath(codeGraph, projectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	// Load security patterns
	patternRegistry := patterns.NewPatternRegistry()
	patternRegistry.LoadDefaultPatterns()

	return callGraph, registry, patternRegistry, nil
}

// AnalyzePatterns runs pattern matching against the call graph.
// Returns a list of matched patterns with their details.
func AnalyzePatterns(callGraph *CallGraph, patternRegistry *PatternRegistry) []PatternMatch {
	var matches []PatternMatch

	for _, pattern := range patternRegistry.Patterns {
		details := patternRegistry.MatchPattern(pattern, callGraph)
		if details != nil && details.Matched {
			match := PatternMatch{
				PatternID:    pattern.ID,
				PatternName:  pattern.Name,
				Description:  pattern.Description,
				Severity:     pattern.Severity,
				CWE:          pattern.CWE,
				OWASP:        pattern.OWASP,
				SourceFQN:    details.SourceFQN,
				SourceCall:   details.SourceCall,
				SinkFQN:      details.SinkFQN,
				SinkCall:     details.SinkCall,
				DataFlowPath: details.DataFlowPath,
			}

			// Lookup source function details from call graph
			if sourceNode, ok := callGraph.Functions[details.SourceFQN]; ok {
				match.SourceFile = sourceNode.File
				match.SourceLine = sourceNode.LineNumber
				match.SourceCode = sourceNode.GetCodeSnippet()
			}

			// Lookup sink function details from call graph
			if sinkNode, ok := callGraph.Functions[details.SinkFQN]; ok {
				match.SinkFile = sinkNode.File
				match.SinkLine = sinkNode.LineNumber
				match.SinkCode = sinkNode.GetCodeSnippet()
			}

			matches = append(matches, match)
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

	// Vulnerability location details
	SourceFQN      string // Fully qualified name of the source function
	SourceCall     string // The actual dangerous call (e.g., "input", "request.GET")
	SourceFile     string // File path where source is located
	SourceLine     uint32 // Line number of source function
	SourceCode     string // Code snippet of source function

	SinkFQN        string // Fully qualified name of the sink function
	SinkCall       string // The actual dangerous call (e.g., "eval", "exec")
	SinkFile       string // File path where sink is located
	SinkLine       uint32 // Line number of sink function
	SinkCode       string // Code snippet of sink function

	DataFlowPath   []string // Complete path from source to sink (FQNs)
}
