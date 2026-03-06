package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// makeTestAssignStmt creates an assignment statement for testing.
func makeTestAssignStmt(line uint32, def string, callTarget string, uses []string) *core.Statement {
	return &core.Statement{
		Type:       core.StatementTypeAssignment,
		LineNumber: line,
		Def:        def,
		CallTarget: callTarget,
		Uses:       uses,
	}
}

// makeTestCallStmt creates a standalone call statement for testing.
func makeTestCallStmt(line uint32, callTarget string, uses []string) *core.Statement {
	return &core.Statement{
		Type:       core.StatementTypeCall,
		LineNumber: line,
		Def:        "",
		CallTarget: callTarget,
		Uses:       uses,
	}
}

// setupTestCallGraph creates a CallGraph with the given function's statements and call sites.
func setupTestCallGraph(funcFQN string, stmts []*core.Statement, callSites []core.CallSite) *core.CallGraph {
	cg := core.NewCallGraph()
	cg.Statements[funcFQN] = stmts
	cg.CallSites[funcFQN] = callSites
	return cg
}

// TestVDGIntegration_Case1_DirectFlow tests: x = source(); sink(x) -> DETECT
func TestVDGIntegration_Case1_DirectFlow(t *testing.T) {
	funcFQN := "test.module.case_direct"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestCallStmt(2, "eval", []string{"x"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 2}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	if len(detections) != 1 {
		t.Fatalf("Case 1 (direct flow): expected 1 detection, got %d", len(detections))
	}
	if detections[0].Sanitized {
		t.Error("Case 1: should not be sanitized")
	}
}

// TestVDGIntegration_Case2_TransitiveFlow tests: x = source(); y = x; sink(y) -> DETECT
func TestVDGIntegration_Case2_TransitiveFlow(t *testing.T) {
	funcFQN := "test.module.case_transitive"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestAssignStmt(2, "y", "", []string{"x"}),
		makeTestCallStmt(3, "eval", []string{"y"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 3}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	if len(detections) != 1 {
		t.Fatalf("Case 2 (transitive flow): expected 1 detection, got %d", len(detections))
	}
}

// TestVDGIntegration_Case3_FlowThroughCall tests: x = source(); y = transform(x); sink(y) -> DETECT
func TestVDGIntegration_Case3_FlowThroughCall(t *testing.T) {
	funcFQN := "test.module.case_call"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestAssignStmt(2, "y", "str", []string{"x"}),
		makeTestCallStmt(3, "eval", []string{"y"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 3}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	if len(detections) != 1 {
		t.Fatalf("Case 3 (flow through call): expected 1 detection, got %d", len(detections))
	}
}

// TestVDGIntegration_Case4_SanitizerKills tests: x = source(); x = sanitize(x); sink(x) -> NO DETECT
func TestVDGIntegration_Case4_SanitizerKills(t *testing.T) {
	funcFQN := "test.module.case_sanitizer"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestAssignStmt(2, "x", "html.escape", []string{"x"}),
		makeTestCallStmt(3, "eval", []string{"x"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "html.escape", Location: core.Location{Line: 2}},
		{Target: "eval", Location: core.Location{Line: 3}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{{Patterns: []string{"html.escape"}}},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	unsanitizedCount := 0
	for _, d := range detections {
		if !d.Sanitized {
			unsanitizedCount++
		}
	}
	if unsanitizedCount != 0 {
		t.Fatalf("Case 4 (sanitizer): expected 0 unsanitized detections, got %d", unsanitizedCount)
	}
}

// TestVDGIntegration_Case5_UnrelatedVars tests: x = source(); sink(y) -> NO DETECT
func TestVDGIntegration_Case5_UnrelatedVars(t *testing.T) {
	funcFQN := "test.module.case_unrelated"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestCallStmt(2, "eval", []string{"y"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 2}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	if len(detections) != 0 {
		t.Fatalf("Case 5 (unrelated vars): expected 0 detections, got %d", len(detections))
	}
}

// TestVDGIntegration_Case6_ReassignmentKills tests: x = source(); x = "safe"; sink(x) -> NO DETECT
func TestVDGIntegration_Case6_ReassignmentKills(t *testing.T) {
	funcFQN := "test.module.case_reassign"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestAssignStmt(2, "x", "\"safe\"", []string{}),
		makeTestCallStmt(3, "eval", []string{"x"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 3}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	if len(detections) != 0 {
		t.Fatalf("Case 6 (reassignment kills): expected 0 detections, got %d", len(detections))
	}
}

// TestVDGIntegration_Case7_MultiHop tests: x = source(); y = x; z = y; sink(z) -> DETECT
func TestVDGIntegration_Case7_MultiHop(t *testing.T) {
	funcFQN := "test.module.case_multihop"
	stmts := []*core.Statement{
		makeTestAssignStmt(1, "x", "os.getenv", []string{}),
		makeTestAssignStmt(2, "y", "", []string{"x"}),
		makeTestAssignStmt(3, "z", "", []string{"y"}),
		makeTestCallStmt(4, "eval", []string{"z"}),
	}
	callSites := []core.CallSite{
		{Target: "os.getenv", Location: core.Location{Line: 1}},
		{Target: "eval", Location: core.Location{Line: 4}},
	}
	cg := setupTestCallGraph(funcFQN, stmts, callSites)

	ir := &DataflowIR{
		Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
		Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
		Sanitizers: []CallMatcherIR{},
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	if len(detections) != 1 {
		t.Fatalf("Case 7 (multi-hop): expected 1 detection, got %d", len(detections))
	}
}

// TestVDGIntegration_Scorecard runs all 7 cases and prints a scorecard summary.
func TestVDGIntegration_Scorecard(t *testing.T) {
	type testCase struct {
		name           string
		stmts          []*core.Statement
		callSites      []core.CallSite
		sanitizers     []CallMatcherIR
		expectDetected bool
	}

	cases := []testCase{
		{
			name: "1. Direct flow (source -> sink)",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestCallStmt(2, "eval", []string{"x"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "eval", Location: core.Location{Line: 2}},
			},
			expectDetected: true,
		},
		{
			name: "2. Transitive flow (source -> x -> sink)",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestAssignStmt(2, "y", "", []string{"x"}),
				makeTestCallStmt(3, "eval", []string{"y"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "eval", Location: core.Location{Line: 3}},
			},
			expectDetected: true,
		},
		{
			name: "3. Flow through function call",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestAssignStmt(2, "y", "str", []string{"x"}),
				makeTestCallStmt(3, "eval", []string{"y"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "eval", Location: core.Location{Line: 3}},
			},
			expectDetected: true,
		},
		{
			name: "4. Sanitizer kills taint",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestAssignStmt(2, "x", "html.escape", []string{"x"}),
				makeTestCallStmt(3, "eval", []string{"x"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "html.escape", Location: core.Location{Line: 2}},
				{Target: "eval", Location: core.Location{Line: 3}},
			},
			sanitizers:     []CallMatcherIR{{Patterns: []string{"html.escape"}}},
			expectDetected: false,
		},
		{
			name: "5. Unrelated variables",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestCallStmt(2, "eval", []string{"y"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "eval", Location: core.Location{Line: 2}},
			},
			expectDetected: false,
		},
		{
			name: "6. Reassignment kills taint",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestAssignStmt(2, "x", "\"safe\"", []string{}),
				makeTestCallStmt(3, "eval", []string{"x"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "eval", Location: core.Location{Line: 3}},
			},
			expectDetected: false,
		},
		{
			name: "7. Multi-hop transitive flow",
			stmts: []*core.Statement{
				makeTestAssignStmt(1, "x", "os.getenv", []string{}),
				makeTestAssignStmt(2, "y", "", []string{"x"}),
				makeTestAssignStmt(3, "z", "", []string{"y"}),
				makeTestCallStmt(4, "eval", []string{"z"}),
			},
			callSites: []core.CallSite{
				{Target: "os.getenv", Location: core.Location{Line: 1}},
				{Target: "eval", Location: core.Location{Line: 4}},
			},
			expectDetected: true,
		},
	}

	passed := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			funcFQN := "test.scorecard." + tc.name
			cg := setupTestCallGraph(funcFQN, tc.stmts, tc.callSites)

			ir := &DataflowIR{
				Sources:    []CallMatcherIR{{Patterns: []string{"os.getenv"}}},
				Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
				Sanitizers: tc.sanitizers,
				Scope:      "local",
			}

			executor := NewDataflowExecutor(ir, cg)
			detections := executor.Execute()

			// Count unsanitized detections
			unsanitized := 0
			for _, d := range detections {
				if !d.Sanitized {
					unsanitized++
				}
			}

			detected := unsanitized > 0

			if detected != tc.expectDetected {
				t.Errorf("FAIL: expected detected=%v, got detected=%v (unsanitized=%d, total=%d)",
					tc.expectDetected, detected, unsanitized, len(detections))
			} else {
				passed++
			}
		})
	}

	t.Logf("\n=== VDG PoC SCORECARD: %d/7 ===", passed)
}
