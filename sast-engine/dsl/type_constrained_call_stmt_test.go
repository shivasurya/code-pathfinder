package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for TypeConstrainedCallExecutor statement scanning (PR-08 fix).
// When CallSites have no InferredType (e.g., function parameter calls),
// the executor falls back to matching enriched stmt.CallChain.

func TestTypeConstrainedCall_StatementMatch_BasicMatch(t *testing.T) {
	cg := core.NewCallGraph()

	// Enriched statement: r.FormValue → "net/http.Request.FormValue"
	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			Def:        "query",
			CallTarget: "FormValue",
			CallChain:  "net/http.Request.FormValue",
			LineNumber: 5,
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"net/http.Request"},
			MethodNames:   []string{"FormValue"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	require.Len(t, detections, 1)
	assert.Equal(t, "testapp.handler", detections[0].FunctionFQN)
	assert.Equal(t, 5, detections[0].SourceLine)
	assert.Equal(t, "statement_callchain", detections[0].MatchMethod)
}

func TestTypeConstrainedCall_StatementMatch_NoMatch(t *testing.T) {
	cg := core.NewCallGraph()

	// Statement with wrong receiver type
	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			Def:        "data",
			CallTarget: "ReadAll",
			CallChain:  "io.ReadAll",
			LineNumber: 5,
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"net/http.Request"},
			MethodNames:   []string{"FormValue"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 0)
}

func TestTypeConstrainedCall_StatementMatch_DeduplicateWithCallSite(t *testing.T) {
	cg := core.NewCallGraph()

	// CallSite with type inference — already matches via CallSite path
	cg.CallSites["testapp.handler"] = []core.CallSite{
		{
			Target:                   "FormValue",
			TargetFQN:                "net/http.Request.FormValue",
			Resolved:                 true,
			ResolvedViaTypeInference: true,
			InferredType:             "net/http.Request",
			TypeConfidence:           0.95,
			Location:                 core.Location{Line: 5},
		},
	}

	// Same match in statements — should NOT duplicate
	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			Def:        "query",
			CallTarget: "FormValue",
			CallChain:  "net/http.Request.FormValue",
			LineNumber: 5,
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"net/http.Request"},
			MethodNames:   []string{"FormValue"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 1, "Should deduplicate CallSite + Statement match at same line")
}

func TestTypeConstrainedCall_StatementMatch_MultipleReceiverTypes(t *testing.T) {
	cg := core.NewCallGraph()

	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:       core.StatementTypeCall,
			CallTarget: "Exec",
			CallChain:  "database/sql.DB.Exec",
			LineNumber: 10,
		},
		{
			Type:       core.StatementTypeAssignment,
			Def:        "rows",
			CallTarget: "Query",
			CallChain:  "database/sql.Tx.Query",
			LineNumber: 12,
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"database/sql.DB", "database/sql.Tx", "database/sql.Stmt"},
			MethodNames:   []string{"Query", "Exec"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 2, "Should match both DB.Exec and Tx.Query")
}

func TestTypeConstrainedCall_StatementMatch_EmptyCallChain(t *testing.T) {
	cg := core.NewCallGraph()

	// Statement with empty CallChain — should be skipped
	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			Def:        "x",
			CallTarget: "",
			CallChain:  "",
			LineNumber: 5,
		},
	}

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"net/http.Request"},
			MethodNames:   []string{"FormValue"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 0)
}

func TestTypeConstrainedCall_StatementMatch_NilStatements(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Statements = nil // explicitly nil

	executor := &TypeConstrainedCallExecutor{
		IR: &TypeConstrainedCallIR{
			ReceiverTypes: []string{"net/http.Request"},
			MethodNames:   []string{"FormValue"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 0, "Should not panic with nil Statements")
}
