package taint

import (
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// TaintDetection represents a detected taint flow from source to sink.
type TaintDetection struct {
	SourceLine      uint32
	SourceVar       string
	SinkLine        uint32
	SinkCall        string
	PropagationPath []string
	Confidence      float64
}

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

// FindTaintFlows discovers taint flows from sources to sinks using BFS reachability.
func (g *VarDepGraph) FindTaintFlows(statements []*core.Statement, sinks []string) []TaintDetection {
	// Collect all taint source node keys
	var sourceKeys []string
	for key, node := range g.Nodes {
		if node.IsTaintSrc {
			sourceKeys = append(sourceKeys, key)
		}
	}

	var detections []TaintDetection

	// For each sink statement (Type==Call, matches sink patterns, Def=="")
	for _, stmt := range statements {
		if stmt.Type != core.StatementTypeCall || stmt.Def != "" {
			continue
		}
		if !matchesAnyPattern(stmt.CallTarget, sinks) {
			continue
		}

		for _, usedVar := range stmt.Uses {
			defKey, found := g.LatestDefAt(usedVar, stmt.LineNumber)
			if !found {
				continue
			}

			for _, srcKey := range sourceKeys {
				path := g.findPath(srcKey, defKey)
				if path == nil {
					continue
				}
				if g.pathContainsSanitizer(path) {
					continue
				}

				srcNode := g.Nodes[srcKey]
				detections = append(detections, TaintDetection{
					SourceLine:      srcNode.Line,
					SourceVar:       srcNode.VarName,
					SinkLine:        stmt.LineNumber,
					SinkCall:        stmt.CallTarget,
					PropagationPath: g.pathToVarNames(path),
					Confidence:      1.0,
				})
			}
		}
	}

	return detections
}

// LatestDefAt finds the node with matching VarName and Line <= beforeLine,
// with the highest Line value. Returns the node key and true, or ("", false).
func (g *VarDepGraph) LatestDefAt(varName string, beforeLine uint32) (string, bool) {
	var bestKey string
	var bestLine uint32
	found := false

	for key, node := range g.Nodes {
		if node.VarName == varName && node.Line <= beforeLine {
			if !found || node.Line > bestLine {
				bestKey = key
				bestLine = node.Line
				found = true
			}
		}
	}

	return bestKey, found
}

// findPath performs BFS from src to dst and returns the path as node keys, or nil if unreachable.
func (g *VarDepGraph) findPath(src, dst string) []string {
	if src == dst {
		return []string{src}
	}

	visited := map[string]bool{src: true}
	parent := map[string]string{}
	queue := []string{src}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range g.Edges[current] {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			parent[neighbor] = current

			if neighbor == dst {
				// Reconstruct path
				var path []string
				for n := dst; n != src; n = parent[n] {
					path = append(path, n)
				}
				path = append(path, src)
				// Reverse
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path
			}

			queue = append(queue, neighbor)
		}
	}

	return nil
}

// pathContainsSanitizer checks if any node on the path has IsSanitized == true.
func (g *VarDepGraph) pathContainsSanitizer(path []string) bool {
	for _, key := range path {
		if node, ok := g.Nodes[key]; ok && node.IsSanitized {
			return true
		}
	}
	return false
}

// AnalyzeWithVDG performs intra-procedural taint analysis using the Variable Dependency Graph.
// Returns a TaintSummary compatible with the existing reporting pipeline.
func AnalyzeWithVDG(
	functionFQN string,
	statements []*core.Statement,
	sources []string,
	sinks []string,
	sanitizers []string,
) *core.TaintSummary {
	summary := core.NewTaintSummary(functionFQN)

	vdg := NewVarDepGraph()
	vdg.Build(statements, sources, sinks, sanitizers)

	detections := vdg.FindTaintFlows(statements, sinks)

	for _, det := range detections {
		taintInfo := &core.TaintInfo{
			SourceLine:      det.SourceLine,
			SourceVar:       det.SourceVar,
			SinkLine:        det.SinkLine,
			SinkCall:        det.SinkCall,
			PropagationPath: det.PropagationPath,
			Confidence:      det.Confidence,
		}
		summary.AddDetection(taintInfo)
		summary.AddTaintedVar(det.SourceVar, &core.TaintInfo{
			SourceLine: det.SourceLine,
			SourceVar:  det.SourceVar,
			Confidence: det.Confidence,
		})
	}

	return summary
}

// pathToVarNames extracts VarName from each node key.
func (g *VarDepGraph) pathToVarNames(path []string) []string {
	names := make([]string, len(path))
	for i, key := range path {
		// Node key format is "varname@line"
		if node, ok := g.Nodes[key]; ok {
			names[i] = node.VarName
		} else {
			// Fallback: parse from key
			parts := strings.SplitN(key, "@", 2)
			names[i] = parts[0]
		}
	}
	return names
}
