package callgraph

import (
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/patterns"
)

// Deprecated: Use patterns.PatternType instead.
type PatternType = patterns.PatternType

// Deprecated: Use patterns constants instead.
const (
	PatternTypeSourceSink        = patterns.PatternTypeSourceSink
	PatternTypeMissingSanitizer  = patterns.PatternTypeMissingSanitizer
	PatternTypeDangerousFunction = patterns.PatternTypeDangerousFunction
)

// Deprecated: Use patterns.Severity instead.
type Severity = patterns.Severity

// Deprecated: Use patterns severity constants instead.
const (
	SeverityCritical = patterns.SeverityCritical
	SeverityHigh     = patterns.SeverityHigh
	SeverityMedium   = patterns.SeverityMedium
	SeverityLow      = patterns.SeverityLow
)

// Deprecated: Use patterns.Pattern instead.
type Pattern = patterns.Pattern

// Deprecated: Use patterns.PatternRegistry instead.
type PatternRegistry = patterns.PatternRegistry

// Deprecated: Use patterns.NewPatternRegistry instead.
func NewPatternRegistry() *PatternRegistry {
	return patterns.NewPatternRegistry()
}

// Deprecated: Use patterns.PatternMatchDetails instead.
type PatternMatchDetails = patterns.PatternMatchDetails

// MatchPattern checks if a call graph matches a pattern.
// Deprecated: Use PatternRegistry.MatchPattern from patterns package instead.
func MatchPattern(pattern *Pattern, callGraph *CallGraph) *PatternMatchDetails {
	registry := patterns.NewPatternRegistry()
	return registry.MatchPattern(pattern, callGraph)
}
