package taint

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// VarDefSite represents a single definition of a variable at a specific line.
type VarDefSite struct {
	VarName     string
	Line        uint32
	IsTaintSrc  bool
	IsSanitized bool
	CallTarget  string
}

// VarDepGraph is a directed graph of variable data dependencies within a function.
type VarDepGraph struct {
	Nodes     map[string]*VarDefSite // key: "varname@line"
	Edges     map[string][]string    // forward edges
	LatestDef map[string]string      // variable name -> current live def-site key
}

// NewVarDepGraph creates an empty variable dependency graph.
func NewVarDepGraph() *VarDepGraph {
	return &VarDepGraph{
		Nodes:     make(map[string]*VarDefSite),
		Edges:     make(map[string][]string),
		LatestDef: make(map[string]string),
	}
}

func nodeKey(varName string, line uint32) string {
	return fmt.Sprintf("%s@%d", varName, line)
}

// Build constructs the VDG from statements.
// sources/sinks/sanitizers are function name patterns.
func (g *VarDepGraph) Build(
	statements []*core.Statement,
	sources []string,
	sinks []string,
	sanitizers []string,
) {
	for _, stmt := range statements {
		if stmt.Def == "" {
			continue
		}

		key := nodeKey(stmt.Def, stmt.LineNumber)
		node := &VarDefSite{
			VarName:    stmt.Def,
			Line:       stmt.LineNumber,
			CallTarget: stmt.CallTarget,
		}

		if stmt.CallTarget != "" && matchesAnyPattern(stmt.CallTarget, sources) {
			node.IsTaintSrc = true
		}

		if stmt.CallTarget != "" && matchesAnyPattern(stmt.CallTarget, sanitizers) {
			node.IsSanitized = true
		}

		g.Nodes[key] = node

		for _, usedVar := range stmt.Uses {
			if srcKey, ok := g.LatestDef[usedVar]; ok {
				g.Edges[srcKey] = append(g.Edges[srcKey], key)
			}
		}

		g.LatestDef[stmt.Def] = key
	}
}

func matchesAnyPattern(callTarget string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesFunctionName(callTarget, pattern) {
			return true
		}
	}
	return false
}
