package dsl

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// AttributeMatcherExecutor finds statements where AttributeAccess matches a pattern.
type AttributeMatcherExecutor struct {
	IR        *AttributeMatcherIR
	CallGraph *core.CallGraph
}

// NewAttributeMatcherExecutor creates a new executor.
func NewAttributeMatcherExecutor(ir *AttributeMatcherIR, cg *core.CallGraph) *AttributeMatcherExecutor {
	return &AttributeMatcherExecutor{
		IR:        ir,
		CallGraph: cg,
	}
}

// AttributeMatch represents a matched attribute access with context.
type AttributeMatch struct {
	CallSite    core.CallSite
	FunctionFQN string
	Line        int
}

// Execute scans all statements in the call graph and returns matches where
// AttributeAccess matches any of the patterns.
func (e *AttributeMatcherExecutor) Execute() []AttributeMatch {
	var matches []AttributeMatch

	if e.CallGraph == nil || e.CallGraph.Statements == nil {
		return matches
	}

	for funcFQN, stmts := range e.CallGraph.Statements {
		for _, stmt := range stmts {
			if stmt.AttributeAccess == "" {
				continue
			}
			if e.matchesAny(stmt.AttributeAccess) {
				matches = append(matches, AttributeMatch{
					CallSite: core.CallSite{
						Target: stmt.AttributeAccess,
						Location: core.Location{
							Line: int(stmt.LineNumber),
						},
					},
					FunctionFQN: funcFQN,
					Line:        int(stmt.LineNumber),
				})
			}
		}
	}

	return matches
}

// matchesAny checks if the attribute access matches any pattern.
func (e *AttributeMatcherExecutor) matchesAny(attrAccess string) bool {
	for _, pattern := range e.IR.Patterns {
		if matchesAttributePattern(attrAccess, pattern) {
			return true
		}
	}
	return false
}

// matchesAttributePattern checks if an attribute access string matches a pattern.
// Replicates the matching logic from taint.matchesFunctionName for use in the DSL layer.
func matchesAttributePattern(attrAccess, pattern string) bool {
	// Exact match
	if attrAccess == pattern {
		return true
	}

	// Suffix match: "request.url" matches pattern "url"
	if len(attrAccess) > len(pattern)+1 && attrAccess[len(attrAccess)-len(pattern)-1] == '.' && attrAccess[len(attrAccess)-len(pattern):] == pattern {
		return true
	}

	return false
}
