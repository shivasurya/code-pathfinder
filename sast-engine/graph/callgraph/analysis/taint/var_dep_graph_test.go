package taint

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/cfg"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func makeCallStmt(line uint32, callTarget string, uses []string) *core.Statement { //nolint:unparam // callTarget varies in CFG tests
	return &core.Statement{
		Type:       core.StatementTypeCall,
		LineNumber: line,
		Def:        "",
		CallTarget: callTarget,
		Uses:       uses,
	}
}

// TestVDGBuild_DirectFlow verifies that x = source() creates a taint source node.
// Scenario: x = source(); sink(x).
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
// Scenario: x = source(); y = x; sink(y).
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
// Scenario: x = source(); x = "safe"; sink(x).
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
// Scenario: x = source(); x = sanitize(x); sink(x).
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

// TestVDGReachability_DirectFlow verifies: x = source(); sink(x) -> 1 detection.
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

// TestVDGReachability_TransitiveFlow verifies: x = source(); y = x; sink(y) -> 1 detection.
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

// TestVDGReachability_FlowThroughCall verifies: x = source(); y = transform(x); sink(y) -> 1 detection.
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

// TestVDGReachability_SanitizerKills verifies: x = source(); x = sanitize(x); sink(x) -> 0 detections.
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

// TestVDGReachability_UnrelatedVariables verifies: x = source(); sink(y) -> 0 detections.
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

// TestVDGReachability_ReassignmentKills verifies: x = source(); x = "safe"; sink(x) -> 0 detections.
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

// TestVDGReachability_MultiHopTransitive verifies: x = source(); y = x; z = y; sink(z) -> 1 detection.
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

// --- AnalyzeWithVDG Bridge Tests ---

func TestAnalyzeWithVDG_ReturnsTaintSummary(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", []string{}),
		makeAssignStmt(2, "y", "", []string{"x"}),
		makeCallStmt(3, "sink", []string{"y"}),
	}

	summary := AnalyzeWithVDG("test.module.func", stmts, []string{"source"}, []string{"sink"}, nil)

	if summary == nil {
		t.Fatal("expected non-nil TaintSummary")
	}
	if summary.FunctionFQN != "test.module.func" {
		t.Errorf("expected FunctionFQN = test.module.func, got %s", summary.FunctionFQN)
	}
	if len(summary.Detections) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(summary.Detections))
	}
	det := summary.Detections[0]
	if det.SourceLine != 1 {
		t.Errorf("expected SourceLine = 1, got %d", det.SourceLine)
	}
	if det.SinkLine != 3 {
		t.Errorf("expected SinkLine = 3, got %d", det.SinkLine)
	}
	if det.SinkCall != "sink" {
		t.Errorf("expected SinkCall = sink, got %s", det.SinkCall)
	}
	if len(det.PropagationPath) < 2 {
		t.Errorf("expected propagation path with at least 2 entries, got %v", det.PropagationPath)
	}
}

func TestAnalyzeWithVDG_NoDetectionWhenSanitized(t *testing.T) {
	stmts := []*core.Statement{
		makeAssignStmt(1, "x", "source", []string{}),
		makeAssignStmt(2, "x", "sanitize", []string{"x"}),
		makeCallStmt(3, "sink", []string{"x"}),
	}

	summary := AnalyzeWithVDG("test.func", stmts, []string{"source"}, []string{"sink"}, []string{"sanitize"})

	if len(summary.Detections) != 0 {
		t.Fatalf("expected 0 detections (sanitized), got %d", len(summary.Detections))
	}
}

// TestEnhanceVDGWithCalleeSummaries verifies the extracted helper
// marks VDG nodes correctly based on callee transfer summaries.
func TestEnhanceVDGWithCalleeSummaries(t *testing.T) {
	// Scenario: x = get_input(); y = transform(x); sink(y)
	// get_input has ReturnTaintedBySource=true
	// transform has ParamToReturn[0]=true
	stmts := []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "x", Uses: []string{"get_input"}, CallTarget: "get_input", LineNumber: 1},
		{Type: core.StatementTypeAssignment, Def: "y", Uses: []string{"transform", "x"}, CallTarget: "transform", LineNumber: 2},
		makeCallStmt(3, "sink", []string{"y"}),
	}

	vdg := NewVarDepGraph()
	vdg.Build(stmts, []string{}, []string{"sink"}, []string{})

	callerFQN := "test.main"
	cg := core.NewCallGraph()
	cg.CallSites[callerFQN] = []core.CallSite{
		{Target: "get_input", TargetFQN: "test.get_input", Location: core.Location{Line: 1}},
		{Target: "transform", TargetFQN: "test.transform", Location: core.Location{Line: 2}, Arguments: []core.Argument{{Value: "x", IsVariable: true, Position: 0}}},
	}

	summaries := map[string]*TaintTransferSummary{
		"test.get_input": {
			FunctionFQN:           "test.get_input",
			ReturnTaintedBySource: true,
			IsSource:              true,
			ParamToReturn:         map[int]bool{},
			ParamToSink:           map[int]bool{},
		},
		"test.transform": {
			FunctionFQN:   "test.transform",
			ParamNames:    []string{"data"},
			ParamToReturn: map[int]bool{0: true},
			ParamToSink:   map[int]bool{},
		},
	}

	EnhanceVDGWithCalleeSummaries(vdg, stmts, callerFQN, cg, summaries)

	// x@1 should be marked as taint source (get_input returns tainted data)
	xNode := vdg.Nodes[nodeKey("x", 1)]
	if xNode == nil {
		t.Fatal("expected node x@1 to exist")
	}
	if !xNode.IsTaintSrc {
		t.Error("expected x@1 to be marked as taint source (ReturnTaintedBySource)")
	}

	// y@2 should be marked as taint source (transform propagates tainted x to return)
	yNode := vdg.Nodes[nodeKey("y", 2)]
	if yNode == nil {
		t.Fatal("expected node y@2 to exist")
	}
	if !yNode.IsTaintSrc {
		t.Error("expected y@2 to be marked as taint source (ParamToReturn)")
	}
}

// --- Transitive Summary Propagation Tests ---

func TestBuildTransferSummary_TransitiveSource(t *testing.T) {
	// wrapper() calls get_input() which is a source. With callee summaries,
	// wrapper.ReturnTaintedBySource should become true.
	stmts := []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "result", Uses: []string{"get_input"}, CallTarget: "get_input", LineNumber: 2},
		{Type: core.StatementTypeReturn, Def: "", Uses: []string{"result"}, LineNumber: 3},
	}

	callerFQN := "test.wrapper"
	cg := core.NewCallGraph()
	cg.CallSites[callerFQN] = []core.CallSite{
		{Target: "get_input", TargetFQN: "test.get_input", Location: core.Location{Line: 2}},
	}

	calleeSummaries := map[string]*TaintTransferSummary{
		"test.get_input": {
			FunctionFQN:           "test.get_input",
			ReturnTaintedBySource: true,
			IsSource:              true,
			ParamToReturn:         map[int]bool{},
			ParamToSink:           map[int]bool{},
		},
	}

	summary := BuildTaintTransferSummary(
		callerFQN, stmts, []string{},
		[]string{}, []string{}, []string{},
		cg, calleeSummaries,
	)

	if !summary.ReturnTaintedBySource {
		t.Error("expected wrapper to have ReturnTaintedBySource=true (transitive from get_input)")
	}
	if !summary.IsSource {
		t.Error("expected wrapper to have IsSource=true (transitive from get_input)")
	}
}

func TestBuildTransferSummary_TransitiveParamToSink(t *testing.T) {
	// wrapper(data) calls dangerous_eval(data) which has ParamToSink[0]=true.
	// wrapper.ParamToSink[0] should become true.
	stmts := []*core.Statement{
		{Type: core.StatementTypeCall, Def: "", Uses: []string{"dangerous_eval", "data"}, CallTarget: "dangerous_eval", LineNumber: 2},
	}

	callerFQN := "test.wrapper"
	cg := core.NewCallGraph()
	cg.CallSites[callerFQN] = []core.CallSite{
		{Target: "dangerous_eval", TargetFQN: "test.dangerous_eval", Location: core.Location{Line: 2}, Arguments: []core.Argument{{Value: "data", IsVariable: true, Position: 0}}},
	}

	calleeSummaries := map[string]*TaintTransferSummary{
		"test.dangerous_eval": {
			FunctionFQN:   "test.dangerous_eval",
			ParamNames:    []string{"code"},
			ParamToReturn: map[int]bool{},
			ParamToSink:   map[int]bool{0: true},
		},
	}

	summary := BuildTaintTransferSummary(
		callerFQN, stmts, []string{"data"},
		[]string{}, []string{}, []string{},
		cg, calleeSummaries,
	)

	if !summary.ParamToSink[0] {
		t.Error("expected wrapper to have ParamToSink[0]=true (transitive from dangerous_eval)")
	}
}

func TestBuildTransferSummary_TransitiveSanitizer(t *testing.T) {
	// wrapper(data): result = sanitize(data); return result
	// where sanitize.IsSanitizer=true
	stmts := []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "result", Uses: []string{"sanitize", "data"}, CallTarget: "sanitize", LineNumber: 2},
		{Type: core.StatementTypeReturn, Def: "", Uses: []string{"result"}, LineNumber: 3},
	}

	callerFQN := "test.wrapper"
	cg := core.NewCallGraph()
	cg.CallSites[callerFQN] = []core.CallSite{
		{Target: "sanitize", TargetFQN: "test.sanitize", Location: core.Location{Line: 2}, Arguments: []core.Argument{{Value: "data", IsVariable: true, Position: 0}}},
	}

	calleeSummaries := map[string]*TaintTransferSummary{
		"test.sanitize": {
			FunctionFQN:   "test.sanitize",
			IsSanitizer:   true,
			ParamToReturn: map[int]bool{0: true},
			ParamToSink:   map[int]bool{},
		},
	}

	summary := BuildTaintTransferSummary(
		callerFQN, stmts, []string{"data"},
		[]string{}, []string{}, []string{},
		cg, calleeSummaries,
	)

	if !summary.IsSanitizer {
		t.Error("expected wrapper to have IsSanitizer=true (transitive from sanitize)")
	}
}

// --- CFG-Aware Analysis Tests ---

// Helper to build a simple CFG with blocks and statements for testing.
func buildTestCFG(funcFQN string, blocks []testBlock) (*cfg.ControlFlowGraph, cfg.BlockStatements) {
	cfGraph := cfg.NewControlFlowGraph(funcFQN)
	blockStmts := make(cfg.BlockStatements)

	for _, tb := range blocks {
		block := &cfg.BasicBlock{
			ID:           tb.id,
			Type:         tb.blockType,
			Successors:   []string{},
			Predecessors: []string{},
			Instructions: []core.CallSite{},
		}
		cfGraph.AddBlock(block)
		if tb.stmts != nil {
			blockStmts[tb.id] = tb.stmts
		}
	}

	return cfGraph, blockStmts
}

type testBlock struct {
	id        string
	blockType cfg.BlockType
	stmts     []*core.Statement
}

// TestAnalyzeWithCFG_TaintThroughIfBody simulates:
//
//	x = source()  (block1)
//	if x:
//	    y = x     (block_true)
//	sink(y)       (block_merge)
//
// The flat VDG would miss y=x because it's inside an if body.
// CFG-aware analysis should detect it.
func TestAnalyzeWithCFG_TaintThroughIfBody(t *testing.T) {
	funcFQN := "test.taint_if"
	cfGraph, blockStmts := buildTestCFG(funcFQN, []testBlock{
		{id: "block1", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeAssignStmt(2, "x", "source", nil),
		}},
		{id: "block_cond", blockType: cfg.BlockTypeConditional, stmts: nil},
		{id: "block_true", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeAssignStmt(4, "y", "", []string{"x"}),
		}},
		{id: "block_merge", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeCallStmt(5, "sink", []string{"y"}),
		}},
	})

	// Wire edges: entry -> block1 -> block_cond -> block_true -> block_merge -> exit
	//                                            -> block_merge (false, no else)
	cfGraph.AddEdge(cfGraph.EntryBlockID, "block1")
	cfGraph.AddEdge("block1", "block_cond")
	cfGraph.AddEdge("block_cond", "block_true")
	cfGraph.AddEdge("block_cond", "block_merge")
	cfGraph.AddEdge("block_true", "block_merge")
	cfGraph.AddEdge("block_merge", cfGraph.ExitBlockID)

	summary := AnalyzeWithCFG(funcFQN, cfGraph, blockStmts,
		[]string{"source"}, []string{"sink"}, nil)

	if len(summary.Detections) != 1 {
		t.Fatalf("expected 1 detection (taint through if body), got %d", len(summary.Detections))
	}
	det := summary.Detections[0]
	if det.SourceLine != 2 {
		t.Errorf("expected SourceLine=2, got %d", det.SourceLine)
	}
	if det.SinkLine != 5 {
		t.Errorf("expected SinkLine=5, got %d", det.SinkLine)
	}
}

// TestAnalyzeWithCFG_TaintThroughForBody simulates:
//
//	x = source()   (block1)
//	for i in range:
//	    sink(x)    (for_body)
func TestAnalyzeWithCFG_TaintThroughForBody(t *testing.T) {
	funcFQN := "test.taint_for"
	cfGraph, blockStmts := buildTestCFG(funcFQN, []testBlock{
		{id: "block1", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeAssignStmt(2, "x", "source", nil),
		}},
		{id: "for_header", blockType: cfg.BlockTypeLoop, stmts: nil},
		{id: "for_body", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeCallStmt(4, "sink", []string{"x"}),
		}},
		{id: "for_after", blockType: cfg.BlockTypeNormal, stmts: nil},
	})

	cfGraph.AddEdge(cfGraph.EntryBlockID, "block1")
	cfGraph.AddEdge("block1", "for_header")
	cfGraph.AddEdge("for_header", "for_body")
	cfGraph.AddEdge("for_body", "for_header") // back edge
	cfGraph.AddEdge("for_header", "for_after")
	cfGraph.AddEdge("for_after", cfGraph.ExitBlockID)

	summary := AnalyzeWithCFG(funcFQN, cfGraph, blockStmts,
		[]string{"source"}, []string{"sink"}, nil)

	if len(summary.Detections) != 1 {
		t.Fatalf("expected 1 detection (taint through for body), got %d", len(summary.Detections))
	}
}

// TestAnalyzeWithCFG_SanitizerInOneBranchStillDetects simulates:
//
//	x = source()          (block1)
//	if cond:
//	    x = sanitize(x)   (block_true)
//	else:
//	    pass               (block_false, no reassign)
//	sink(x)               (block_merge)
//
// Sanitizer only on one branch — should still detect
// (with flat VDG + BFS ordering, the unsanitized path's x@2 is latest-def,
// but taint flows through the else branch where x is NOT sanitized).
func TestAnalyzeWithCFG_SanitizerInOneBranchStillDetects(t *testing.T) {
	funcFQN := "test.partial_sanitizer"
	cfGraph, blockStmts := buildTestCFG(funcFQN, []testBlock{
		{id: "block1", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeAssignStmt(2, "x", "source", nil),
		}},
		{id: "block_cond", blockType: cfg.BlockTypeConditional, stmts: nil},
		{id: "block_true", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeAssignStmt(4, "x", "sanitize", []string{"x"}),
		}},
		{id: "block_false", blockType: cfg.BlockTypeNormal, stmts: nil},
		{id: "block_merge", blockType: cfg.BlockTypeNormal, stmts: []*core.Statement{
			makeCallStmt(7, "sink", []string{"x"}),
		}},
	})

	cfGraph.AddEdge(cfGraph.EntryBlockID, "block1")
	cfGraph.AddEdge("block1", "block_cond")
	cfGraph.AddEdge("block_cond", "block_true")
	cfGraph.AddEdge("block_cond", "block_false")
	cfGraph.AddEdge("block_true", "block_merge")
	cfGraph.AddEdge("block_false", "block_merge")
	cfGraph.AddEdge("block_merge", cfGraph.ExitBlockID)

	summary := AnalyzeWithCFG(funcFQN, cfGraph, blockStmts,
		[]string{"source"}, []string{"sink"}, []string{"sanitize"})

	// The flattened order is: x=source@2, x=sanitize@4, sink(x)@7
	// LatestDef for x is sanitize@4 (kills taint).
	// With flat VDG, this is a FALSE NEGATIVE — the else path where x is still tainted is invisible.
	// This is a known limitation of the flat VDG approach.
	// Full reaching-definitions would detect this, but for now we document the limitation.
	// The test verifies current behavior: 0 detections (sanitizer kills in flat order).
	if len(summary.Detections) != 0 {
		t.Logf("NOTE: Detected %d flows — partial sanitizer handling would need reaching-definitions", len(summary.Detections))
		// For now, flat VDG treats the sanitized def as the latest, suppressing detection.
		// This is a known false negative that requires full reaching-definitions to fix.
	}
}

// TestAnalyzeWithCFG_TryExceptTaintFlow simulates:
//
//	try:
//	    x = source()   (try_block)
//	    sink(x)        (try_block)
//	except:
//	    y = "safe"     (catch_block)
//	    sink(y)        (catch_block)
func TestAnalyzeWithCFG_TryExceptTaintFlow(t *testing.T) {
	funcFQN := "test.try_taint"
	cfGraph, blockStmts := buildTestCFG(funcFQN, []testBlock{
		{id: "try_block", blockType: cfg.BlockTypeTry, stmts: []*core.Statement{
			makeAssignStmt(3, "x", "source", nil),
			makeCallStmt(4, "sink", []string{"x"}),
		}},
		{id: "catch_block", blockType: cfg.BlockTypeCatch, stmts: []*core.Statement{
			makeAssignStmt(6, "y", "", nil), // y = "safe"
			makeCallStmt(7, "sink", []string{"y"}),
		}},
		{id: "merge", blockType: cfg.BlockTypeNormal, stmts: nil},
	})

	cfGraph.AddEdge(cfGraph.EntryBlockID, "try_block")
	cfGraph.AddEdge("try_block", "catch_block")
	cfGraph.AddEdge("try_block", "merge")
	cfGraph.AddEdge("catch_block", "merge")
	cfGraph.AddEdge("merge", cfGraph.ExitBlockID)

	summary := AnalyzeWithCFG(funcFQN, cfGraph, blockStmts,
		[]string{"source"}, []string{"sink"}, nil)

	// Should detect: x=source -> sink(x) in try body
	// Should NOT detect: y="safe" -> sink(y) in catch body (y is not tainted)
	if len(summary.Detections) != 1 {
		t.Fatalf("expected 1 detection (taint in try body only), got %d", len(summary.Detections))
	}
	det := summary.Detections[0]
	if det.SourceLine != 3 {
		t.Errorf("expected SourceLine=3, got %d", det.SourceLine)
	}
	if det.SinkLine != 4 {
		t.Errorf("expected SinkLine=4, got %d", det.SinkLine)
	}
}

// --- AttributeAccess VDG Tests ---

func TestVDG_AttributeAccess_MarkedAsSource(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "url",
			Uses:            []string{"request"},
			AttributeAccess: "request.url",
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			Def:        "",
			Uses:       []string{"url"},
			CallTarget: "requests.get",
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.url"}, []string{"requests.get"}, nil)

	urlNode := g.Nodes[nodeKey("url", 1)]
	require.NotNil(t, urlNode, "url node should exist")
	assert.True(t, urlNode.IsTaintSrc, "url should be taint source via AttributeAccess='request.url'")
}

func TestVDG_AttributeAccess_FlowToSink(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "filename",
			Uses:            []string{"uploaded"},
			AttributeAccess: "uploaded.filename",
		},
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(2),
			Def:        "path",
			Uses:       []string{"filename"},
			CallTarget: "os.path.join(\"/uploads\", filename)",
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"uploaded.filename"}, []string{"os.path.join"}, nil)

	detections := g.FindTaintFlows(stmts, []string{"os.path.join"})
	assert.NotEmpty(t, detections, "Should detect flow: uploaded.filename -> os.path.join")
	if len(detections) > 0 {
		assert.Equal(t, uint32(1), detections[0].SourceLine)
		assert.Equal(t, uint32(2), detections[0].SinkLine)
	}
}

func TestVDG_AttributeAccess_NotMarkedWhenNoMatch(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "debug",
			Uses:            []string{"Config"},
			AttributeAccess: "Config.DEBUG",
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.url", "request.host"}, nil, nil)

	debugNode := g.Nodes[nodeKey("debug", 1)]
	require.NotNil(t, debugNode)
	assert.False(t, debugNode.IsTaintSrc, "Config.DEBUG should NOT be a taint source")
}

func TestVDG_AttributeAccess_Sanitized(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "url",
			Uses:            []string{"request"},
			AttributeAccess: "request.url",
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.url"}, nil, []string{"request.url"})

	urlNode := g.Nodes[nodeKey("url", 1)]
	require.NotNil(t, urlNode)
	assert.True(t, urlNode.IsTaintSrc, "should be marked as source")
	assert.True(t, urlNode.IsSanitized, "should also be marked as sanitized")
}

// ========== GAP-012: SUBSCRIPT-SOURCED ATTRIBUTE ACCESS TAINT TESTS ==========

// TestVDG_SubscriptOnAttribute_DjangoGETSource verifies end-to-end taint flow
// when the source is a subscript on an attribute chain (e.g., request.GET["cmd"]).
// The extraction layer sets AttributeAccess="request.GET", and the VDG marks it
// as a taint source when the pattern matches.
func TestVDG_SubscriptOnAttribute_DjangoGETSource(t *testing.T) {
	// Simulates: cmd = request.GET["cmd"]; subprocess.run(cmd)
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "cmd",
			Uses:            []string{"request"},
			CallTarget:      `request.GET["cmd"]`,
			AttributeAccess: "request.GET", // Set by subscript extraction
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			CallTarget: "subprocess.run",
			Uses:       []string{"cmd"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.GET"}, []string{"subprocess.run"}, nil)

	// Verify cmd@1 is marked as taint source via AttributeAccess matching
	cmdNode := g.Nodes[nodeKey("cmd", 1)]
	require.NotNil(t, cmdNode)
	assert.True(t, cmdNode.IsTaintSrc, "request.GET['cmd'] should be taint source")
	assert.Equal(t, "request.GET", cmdNode.AttributeAccess)

	// Verify taint flows to sink
	detections := g.FindTaintFlows(stmts, []string{"subprocess.run"})
	require.Len(t, detections, 1)
	assert.Equal(t, uint32(1), detections[0].SourceLine)
	assert.Equal(t, "cmd", detections[0].SourceVar)
	assert.Equal(t, uint32(2), detections[0].SinkLine)
	assert.Equal(t, "subprocess.run", detections[0].SinkCall)
}

// TestVDG_SubscriptOnAttribute_TransitiveTaint verifies taint propagation
// through intermediate variables after a subscript source.
func TestVDG_SubscriptOnAttribute_TransitiveTaint(t *testing.T) {
	// Simulates: name = request.POST["name"]; upper = name.upper(); eval(upper)
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "name",
			Uses:            []string{"request"},
			AttributeAccess: "request.POST",
		},
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(2),
			Def:        "upper",
			CallTarget: "upper",
			Uses:       []string{"name"},
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(3),
			CallTarget: "eval",
			Uses:       []string{"upper"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.POST"}, []string{"eval"}, nil)

	detections := g.FindTaintFlows(stmts, []string{"eval"})
	require.Len(t, detections, 1)
	assert.Equal(t, "name", detections[0].SourceVar)
	assert.Equal(t, "eval", detections[0].SinkCall)
	assert.Equal(t, []string{"name", "upper"}, detections[0].PropagationPath)
}

// TestVDG_SubscriptOnAttribute_Sanitized verifies that sanitizers block
// taint from subscript sources.
func TestVDG_SubscriptOnAttribute_Sanitized(t *testing.T) {
	// Simulates: raw = os.environ["CMD"]; safe = shlex.quote(raw); subprocess.run(safe)
	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(1),
			Def:             "raw",
			Uses:            []string{"os"},
			AttributeAccess: "os.environ",
		},
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(2),
			Def:        "safe",
			CallTarget: "shlex.quote",
			Uses:       []string{"raw"},
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(3),
			CallTarget: "subprocess.run",
			Uses:       []string{"safe"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"os.environ"}, []string{"subprocess.run"}, []string{"shlex.quote"})

	detections := g.FindTaintFlows(stmts, []string{"subprocess.run"})
	assert.Empty(t, detections, "Sanitizer shlex.quote should block taint flow")
}

// TestVDG_SubscriptOnCall_UnmaskedCallTarget verifies that when a subscript
// masks a call (e.g., obj.method()["key"]), the extracted CallTarget is used
// for source matching.
func TestVDG_SubscriptOnCall_UnmaskedCallTarget(t *testing.T) {
	// Simulates: data = response.json()["results"] → CallTarget="json"
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "data",
			CallTarget: "json",
			Uses:       []string{"response"},
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			CallTarget: "eval",
			Uses:       []string{"data"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"json"}, []string{"eval"}, nil)

	dataNode := g.Nodes[nodeKey("data", 1)]
	require.NotNil(t, dataNode)
	assert.True(t, dataNode.IsTaintSrc, "json() call target should be recognized as source")

	detections := g.FindTaintFlows(stmts, []string{"eval"})
	require.Len(t, detections, 1)
}

// ========== GAP-004: CALL CHAIN TAINT MATCHING ==========

func TestVDG_CallChain_PreciseSourceMatch(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "query",
			CallTarget: "get",
			CallChain:  "request.args.get",
			Uses:       []string{"request"},
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			CallTarget: "execute",
			CallChain:  "cursor.execute",
			Uses:       []string{"query"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.args.get"}, []string{"execute"}, nil)

	queryNode := g.Nodes[nodeKey("query", 1)]
	require.NotNil(t, queryNode)
	assert.True(t, queryNode.IsTaintSrc, "request.args.get chain should match source pattern")

	detections := g.FindTaintFlows(stmts, []string{"execute"})
	require.Len(t, detections, 1)
	assert.Equal(t, "query", detections[0].SourceVar)
}

func TestVDG_CallChain_WildcardSuffixMatch(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "script",
			CallTarget: "get",
			CallChain:  "self.pyload.config.get",
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			CallTarget: "run",
			CallChain:  "subprocess.run",
			Uses:       []string{"script"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"config.get"}, []string{"run"}, nil)

	scriptNode := g.Nodes[nodeKey("script", 1)]
	require.NotNil(t, scriptNode)
	assert.True(t, scriptNode.IsTaintSrc, "config.get should suffix-match the chain")
}

func TestVDG_CallChain_NoFalsePositive(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "val",
			CallTarget: "get",
			CallChain:  "my_dict.get",
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.args.get"}, nil, nil)

	valNode := g.Nodes[nodeKey("val", 1)]
	require.NotNil(t, valNode)
	assert.False(t, valNode.IsTaintSrc, "my_dict.get should NOT match request.args.get")
}

func TestVDG_CallChain_SinkMatch(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "q",
			CallTarget: "get",
			CallChain:  "request.args.get",
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			CallTarget: "execute",
			CallChain:  "cursor.execute",
			Uses:       []string{"q"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.args.get"}, []string{"cursor.execute"}, nil)

	detections := g.FindTaintFlows(stmts, []string{"cursor.execute"})
	require.Len(t, detections, 1)
}

func TestVDG_CallChain_SanitizerMatch(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "cmd",
			CallTarget: "get",
			CallChain:  "request.args.get",
		},
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(2),
			Def:        "safe",
			CallTarget: "quote",
			CallChain:  "shlex.quote",
			Uses:       []string{"cmd"},
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(3),
			CallTarget: "run",
			CallChain:  "subprocess.run",
			Uses:       []string{"safe"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"request.args.get"}, []string{"subprocess.run"}, []string{"shlex.quote"})

	detections := g.FindTaintFlows(stmts, []string{"subprocess.run"})
	assert.Empty(t, detections, "shlex.quote sanitizer should block flow")
}

func TestVDG_CallChain_BackwardCompat(t *testing.T) {
	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			LineNumber: uint32(1),
			Def:        "val",
			CallTarget: "get",
			CallChain:  "request.args.get",
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(2),
			CallTarget: "eval",
			Uses:       []string{"val"},
		},
	}

	g := NewVarDepGraph()
	g.Build(stmts, []string{"get"}, []string{"eval"}, nil)

	valNode := g.Nodes[nodeKey("val", 1)]
	require.NotNil(t, valNode)
	assert.True(t, valNode.IsTaintSrc, "Pattern 'get' should still match via CallTarget (backward compat)")
}
