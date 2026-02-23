package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// VariableMatcherExecutor executes variable_matcher IR.
type VariableMatcherExecutor struct {
	IR        *VariableMatcherIR
	CallGraph *core.CallGraph
}

// NewVariableMatcherExecutor creates a new executor.
func NewVariableMatcherExecutor(ir *VariableMatcherIR, cg *core.CallGraph) *VariableMatcherExecutor {
	return &VariableMatcherExecutor{
		IR:        ir,
		CallGraph: cg,
	}
}

// Execute finds all variable references matching the pattern.
//
// Algorithm:
//  1. Iterate over callGraph.CallSites
//  2. For each call site, check arguments for variable references
//  3. Match argument values against pattern (with wildcard support)
//  4. Return list of matching call sites with argument positions
func (e *VariableMatcherExecutor) Execute() []VariableMatchResult {
	matches := []VariableMatchResult{}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, callSite := range callSites {
			// Check each argument
			for _, arg := range callSite.Arguments {
				if arg.IsVariable && e.matchesPattern(arg.Value) {
					matches = append(matches, VariableMatchResult{
						CallSite:     callSite,
						VariableName: arg.Value,
						ArgumentPos:  arg.Position,
						FunctionFQN:  functionFQN,
						SourceFile:   callSite.Location.File,
						Line:         callSite.Location.Line,
					})
				}
			}
		}
	}

	return matches
}

// VariableMatchResult contains match information.
type VariableMatchResult struct {
	CallSite     core.CallSite
	VariableName string // The matched variable name
	ArgumentPos  int    // Position in argument list
	FunctionFQN  string
	SourceFile   string
	Line         int
}

// matchesPattern checks if variable name matches pattern.
func (e *VariableMatcherExecutor) matchesPattern(varName string) bool {
	pattern := e.IR.Pattern

	if !e.IR.Wildcard {
		return varName == pattern
	}

	// Wildcard matching (same as CallMatcher)
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := strings.Trim(pattern, "*")
		return strings.Contains(varName, substr)
	}

	if after, ok := strings.CutPrefix(pattern, "*"); ok {
		suffix := after
		return strings.HasSuffix(varName, suffix)
	}

	if before, ok := strings.CutSuffix(pattern, "*"); ok {
		prefix := before
		return strings.HasPrefix(varName, prefix)
	}

	return varName == pattern
}
