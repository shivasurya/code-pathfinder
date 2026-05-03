package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeConstrainedAttribute_Statements_BasicMatch(t *testing.T) {
	cg := core.NewCallGraph()

	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			Def:             "path",
			AttributeAccess: "net/http.Request.URL.Path",
			LineNumber:      5,
		},
		{
			Type:            core.StatementTypeAssignment,
			Def:             "host",
			AttributeAccess: "net/http.Request.Host",
			LineNumber:      6,
		},
		{
			Type:       core.StatementTypeCall,
			CallTarget: "Query",
			LineNumber: 10,
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverTypes:  []string{"net/http.Request"},
			AttributeNames: []string{"URL.Path", "Host"},
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()

	require.Len(t, detections, 2, "Should match both URL.Path and Host")
	assert.Equal(t, "testapp.handler", detections[0].FunctionFQN)
	assert.Equal(t, 5, detections[0].SourceLine)
	assert.Equal(t, "testapp.handler", detections[1].FunctionFQN)
	assert.Equal(t, 6, detections[1].SourceLine)
}

func TestTypeConstrainedAttribute_Statements_NoMatch(t *testing.T) {
	cg := core.NewCallGraph()

	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			Def:             "val",
			AttributeAccess: "os.File.Name",
			LineNumber:      5,
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverTypes:  []string{"net/http.Request"},
			AttributeNames: []string{"URL.Path"},
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 0, "Should not match — wrong receiver type")
}

func TestTypeConstrainedAttribute_Statements_SingularIR(t *testing.T) {
	cg := core.NewCallGraph()

	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			Def:             "path",
			AttributeAccess: "net/http.Request.URL.Path",
			LineNumber:      5,
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverType:  "net/http.Request",
			AttributeName: "URL.Path",
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	require.Len(t, detections, 1, "Should match with singular IR fields")
}

func TestTypeConstrainedAttribute_Statements_EmptyStatements(t *testing.T) {
	cg := core.NewCallGraph()

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverTypes:  []string{"net/http.Request"},
			AttributeNames: []string{"URL.Path"},
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Len(t, detections, 0)
}

func TestTypeConstrainedAttribute_Statements_PrefixMatch(t *testing.T) {
	cg := core.NewCallGraph()

	// "net/http.Request.URL.Path" should match attr "URL" with prefix match
	cg.Statements["testapp.handler"] = []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			Def:             "url",
			AttributeAccess: "net/http.Request.URL.Path",
			LineNumber:      5,
		},
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverTypes:  []string{"net/http.Request"},
			AttributeNames: []string{"URL"},
		},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	require.Len(t, detections, 1, "Should match URL as prefix of URL.Path")
}

func TestTypeConstrainedAttribute_NilCallGraph(t *testing.T) {
	executor := &TypeConstrainedAttributeExecutor{
		IR: &TypeConstrainedAttributeIR{
			ReceiverTypes:  []string{"net/http.Request"},
			AttributeNames: []string{"URL.Path"},
		},
		CallGraph:   nil,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	detections := executor.Execute()
	assert.Nil(t, detections)
}
