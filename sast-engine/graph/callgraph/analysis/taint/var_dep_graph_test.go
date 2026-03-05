package taint

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

func makeAssignStmt(line uint32, def string, callTarget string, uses []string) *core.Statement {
	return &core.Statement{
		Type:       core.StatementTypeAssignment,
		LineNumber: line,
		Def:        def,
		CallTarget: callTarget,
		Uses:       uses,
	}
}

func makeCallStmt(line uint32, callTarget string, uses []string) *core.Statement {
	return &core.Statement{
		Type:       core.StatementTypeCall,
		LineNumber: line,
		Def:        "",
		CallTarget: callTarget,
		Uses:       uses,
	}
}

// TestVDGBuild_DirectFlow verifies that x = source() creates a taint source node.
// Scenario: x = source(); sink(x)
func TestVDGBuild_DirectFlow(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeCallStmt(2, "sink", []string{"x"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)

	// Verify x@1 exists and is marked as taint source
	key := nodeKey("x", 1)
	node, ok := g.Nodes[key]
	if !ok {
		t.Fatalf("expected node %s to exist", key)
	}
	if !node.IsTaintSrc {
		t.Errorf("expected node %s to be marked as taint source", key)
	}
	if node.VarName != "x" {
		t.Errorf("expected VarName 'x', got %q", node.VarName)
	}
	if node.Line != 1 {
		t.Errorf("expected Line 1, got %d", node.Line)
	}

	// sink(x) has no Def, so it should not create a node
	if _, exists := g.Nodes[nodeKey("", 2)]; exists {
		t.Error("sink call with no Def should not create a node")
	}
}

// TestVDGBuild_TransitiveFlow verifies edges through transitive assignments.
// Scenario: x = source(); y = x; sink(y)
func TestVDGBuild_TransitiveFlow(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "y", "", []string{"x"}),
		makeCallStmt(3, "sink", []string{"y"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)

	// Verify edge from x@1 -> y@2
	xKey := nodeKey("x", 1)
	yKey := nodeKey("y", 2)

	edges, ok := g.Edges[xKey]
	if !ok {
		t.Fatalf("expected edges from %s", xKey)
	}

	found := false
	for _, e := range edges {
		if e == yKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected edge from %s to %s, got edges: %v", xKey, yKey, edges)
	}

	// Verify both nodes exist
	if _, ok := g.Nodes[xKey]; !ok {
		t.Errorf("expected node %s to exist", xKey)
	}
	if _, ok := g.Nodes[yKey]; !ok {
		t.Errorf("expected node %s to exist", yKey)
	}
}

// TestVDGBuild_ReassignmentKills verifies that reassignment updates LatestDef.
// Scenario: x = source(); x = "safe"; sink(x)
func TestVDGBuild_ReassignmentKills(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "x", "", nil),
		makeCallStmt(3, "sink", []string{"x"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)

	// LatestDef[x] should point to x@2, not x@1
	expectedKey := nodeKey("x", 2)
	if g.LatestDef["x"] != expectedKey {
		t.Errorf("expected LatestDef[x] = %s, got %s", expectedKey, g.LatestDef["x"])
	}

	// x@2 should NOT be a taint source (no call target)
	node := g.Nodes[expectedKey]
	if node == nil {
		t.Fatalf("expected node %s to exist", expectedKey)
	}
	if node.IsTaintSrc {
		t.Errorf("expected node %s to NOT be a taint source", expectedKey)
	}

	// x@1 should still be a taint source
	srcNode := g.Nodes[nodeKey("x", 1)]
	if srcNode == nil {
		t.Fatal("expected node x@1 to exist")
	}
	if !srcNode.IsTaintSrc {
		t.Error("expected node x@1 to be a taint source")
	}
}

// TestVDGBuild_SanitizerMarks verifies that sanitizer calls mark nodes.
// Scenario: x = source(); x = sanitize(x); sink(x)
func TestVDGBuild_SanitizerMarks(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "x", "sanitize", []string{"x"}),
		makeCallStmt(3, "sink", []string{"x"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, []string{"sanitize"})

	// x@2 should be marked as sanitized
	sanitizedKey := nodeKey("x", 2)
	node := g.Nodes[sanitizedKey]
	if node == nil {
		t.Fatalf("expected node %s to exist", sanitizedKey)
	}
	if !node.IsSanitized {
		t.Errorf("expected node %s to be marked as sanitized", sanitizedKey)
	}

	// x@1 should be taint source but NOT sanitized
	srcNode := g.Nodes[nodeKey("x", 1)]
	if srcNode == nil {
		t.Fatal("expected node x@1 to exist")
	}
	if !srcNode.IsTaintSrc {
		t.Error("expected node x@1 to be a taint source")
	}
	if srcNode.IsSanitized {
		t.Error("expected node x@1 to NOT be sanitized")
	}

	// There should be an edge from x@1 -> x@2 (since x@2 uses x)
	edges := g.Edges[nodeKey("x", 1)]
	found := false
	for _, e := range edges {
		if e == sanitizedKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected edge from x@1 to %s, got edges: %v", sanitizedKey, edges)
	}
}

// --- Reachability Tests ---

// TestVDGReachability_DirectFlow: x = source(); sink(x) -> 1 detection
func TestVDGReachability_DirectFlow(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeCallStmt(2, "sink", []string{"x"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	d := detections[0]
	if d.SourceLine != 1 {
		t.Errorf("expected SourceLine=1, got %d", d.SourceLine)
	}
	if d.SinkLine != 2 {
		t.Errorf("expected SinkLine=2, got %d", d.SinkLine)
	}
	if d.SourceVar != "x" {
		t.Errorf("expected SourceVar='x', got %q", d.SourceVar)
	}
	if d.SinkCall != "sink" {
		t.Errorf("expected SinkCall='sink', got %q", d.SinkCall)
	}
}

// TestVDGReachability_TransitiveFlow: x = source(); y = x; sink(y) -> 1 detection
func TestVDGReachability_TransitiveFlow(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "y", "", []string{"x"}),
		makeCallStmt(3, "sink", []string{"y"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	d := detections[0]
	if d.SourceLine != 1 {
		t.Errorf("expected SourceLine=1, got %d", d.SourceLine)
	}
	if d.SinkLine != 3 {
		t.Errorf("expected SinkLine=3, got %d", d.SinkLine)
	}
}

// TestVDGReachability_FlowThroughCall: x = source(); y = transform(x); sink(y) -> 1 detection
func TestVDGReachability_FlowThroughCall(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "y", "transform", []string{"x"}),
		makeCallStmt(3, "sink", []string{"y"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	d := detections[0]
	if d.SourceLine != 1 {
		t.Errorf("expected SourceLine=1, got %d", d.SourceLine)
	}
	if d.SinkLine != 3 {
		t.Errorf("expected SinkLine=3, got %d", d.SinkLine)
	}
}

// TestVDGReachability_SanitizerKills: x = source(); x = sanitize(x); sink(x) -> 0 detections
func TestVDGReachability_SanitizerKills(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "x", "sanitize", []string{"x"}),
		makeCallStmt(3, "sink", []string{"x"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, []string{"sanitize"})
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 0 {
		t.Fatalf("expected 0 detections, got %d: %+v", len(detections), detections)
	}
}

// TestVDGReachability_UnrelatedVariables: x = source(); sink(y) -> 0 detections
func TestVDGReachability_UnrelatedVariables(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeCallStmt(2, "sink", []string{"y"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 0 {
		t.Fatalf("expected 0 detections, got %d: %+v", len(detections), detections)
	}
}

// TestVDGReachability_ReassignmentKills: x = source(); x = "safe"; sink(x) -> 0 detections
func TestVDGReachability_ReassignmentKills(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "x", "", nil),
		makeCallStmt(3, "sink", []string{"x"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 0 {
		t.Fatalf("expected 0 detections, got %d: %+v", len(detections), detections)
	}
}

// TestVDGReachability_MultiHopTransitive: x = source(); y = x; z = y; sink(z) -> 1 detection
func TestVDGReachability_MultiHopTransitive(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", nil),
		makeAssignStmt(2, "y", "", []string{"x"}),
		makeAssignStmt(3, "z", "", []string{"y"}),
		makeCallStmt(4, "sink", []string{"z"}),
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"source"}, []string{"sink"}, nil)
	detections := g.FindTaintFlows(stmts, []string{"sink"})

	if len(detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(detections))
	}
	d := detections[0]
	if d.SourceLine != 1 {
		t.Errorf("expected SourceLine=1, got %d", d.SourceLine)
	}
	if d.SinkLine != 4 {
		t.Errorf("expected SinkLine=4, got %d", d.SinkLine)
	}
	// Verify propagation path includes all hops
	if len(d.PropagationPath) < 3 {
		t.Errorf("expected propagation path with at least 3 entries, got %v", d.PropagationPath)
	}
}
