package builder

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeCallGraphs(t *testing.T) {
	// Create Python call graph (dst)
	pythonCG := core.NewCallGraph()
	pythonCG.Functions["myapp.handlers.handle_request"] = &graph.Node{
		ID:       "py1",
		Type:     "function_definition",
		Name:     "handle_request",
		Language: "python",
	}
	pythonCG.Edges["myapp.handlers.handle_request"] = []string{"myapp.utils.helper"}
	pythonCG.ReverseEdges["myapp.utils.helper"] = []string{"myapp.handlers.handle_request"}
	pythonCG.CallSites["myapp.handlers.handle_request"] = []core.CallSite{
		{Target: "helper", Resolved: true, TargetFQN: "myapp.utils.helper"},
	}

	// Create Go call graph (src)
	goCG := core.NewCallGraph()
	goCG.Functions["github.com/myapp/handlers.HandleRequest"] = &graph.Node{
		ID:       "go1",
		Type:     "function_definition",
		Name:     "HandleRequest",
		Language: "go",
	}
	goCG.Edges["github.com/myapp/handlers.HandleRequest"] = []string{"fmt.Println"}
	goCG.ReverseEdges["fmt.Println"] = []string{"github.com/myapp/handlers.HandleRequest"}
	goCG.CallSites["github.com/myapp/handlers.HandleRequest"] = []core.CallSite{
		{Target: "Println", Resolved: true, TargetFQN: "fmt.Println"},
	}

	// Merge
	MergeCallGraphs(pythonCG, goCG)

	// Verify Functions merged
	assert.Len(t, pythonCG.Functions, 2, "Should have 2 functions after merge")
	assert.Contains(t, pythonCG.Functions, "myapp.handlers.handle_request")
	assert.Contains(t, pythonCG.Functions, "github.com/myapp/handlers.HandleRequest")

	// Verify Edges merged
	assert.Len(t, pythonCG.Edges, 2, "Should have 2 edge entries")
	assert.Contains(t, pythonCG.Edges["myapp.handlers.handle_request"], "myapp.utils.helper")
	assert.Contains(t, pythonCG.Edges["github.com/myapp/handlers.HandleRequest"], "fmt.Println")

	// Verify ReverseEdges merged
	assert.Contains(t, pythonCG.ReverseEdges["myapp.utils.helper"], "myapp.handlers.handle_request")
	assert.Contains(t, pythonCG.ReverseEdges["fmt.Println"], "github.com/myapp/handlers.HandleRequest")

	// Verify CallSites merged
	assert.Len(t, pythonCG.CallSites, 2)
	assert.Len(t, pythonCG.CallSites["myapp.handlers.handle_request"], 1)
	assert.Len(t, pythonCG.CallSites["github.com/myapp/handlers.HandleRequest"], 1)
}

func TestMergeCallGraphs_EmptySource(t *testing.T) {
	dst := core.NewCallGraph()
	dst.Functions["test"] = &graph.Node{ID: "1"}

	src := core.NewCallGraph() // Empty

	MergeCallGraphs(dst, src)

	assert.Len(t, dst.Functions, 1, "Should preserve dst when src is empty")
}

func TestMergeCallGraphs_EmptyDestination(t *testing.T) {
	dst := core.NewCallGraph() // Empty

	src := core.NewCallGraph()
	src.Functions["test"] = &graph.Node{ID: "1"}
	src.Edges["test"] = []string{"target"}

	MergeCallGraphs(dst, src)

	assert.Len(t, dst.Functions, 1)
	assert.Len(t, dst.Edges, 1)
}

func TestMergeCallGraphs_DataflowMaps(t *testing.T) {
	// Create Python call graph (dst) with statements
	pythonCG := core.NewCallGraph()
	pythonCG.Functions["myapp.handler"] = &graph.Node{ID: "py1", Language: "python"}
	pythonCG.Statements["myapp.handler"] = []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "x", LineNumber: 1},
	}
	pythonCG.Summaries["myapp.handler"] = core.NewTaintSummary("myapp.handler")

	// Create Go call graph (src) with statements
	goCG := core.NewCallGraph()
	goCG.Functions["github.com/app/handlers.Handle"] = &graph.Node{ID: "go1", Language: "go"}
	goCG.Statements["github.com/app/handlers.Handle"] = []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "query", LineNumber: 5},
		{Type: core.StatementTypeCall, CallTarget: "Query", LineNumber: 6},
	}
	goCG.Summaries["github.com/app/handlers.Handle"] = core.NewTaintSummary("github.com/app/handlers.Handle")

	// Merge
	MergeCallGraphs(pythonCG, goCG)

	// Verify Statements merged
	require.Len(t, pythonCG.Statements, 2, "Should have statements for both Python and Go functions")
	assert.Contains(t, pythonCG.Statements, "myapp.handler")
	assert.Contains(t, pythonCG.Statements, "github.com/app/handlers.Handle")
	assert.Len(t, pythonCG.Statements["github.com/app/handlers.Handle"], 2)

	// Verify Summaries merged
	assert.Len(t, pythonCG.Summaries, 2)
	assert.Contains(t, pythonCG.Summaries, "github.com/app/handlers.Handle")
}

func TestMergeCallGraphs_DataflowMaps_CFGs(t *testing.T) {
	dst := core.NewCallGraph()
	src := core.NewCallGraph()

	// Simulate CFG data (stored as any)
	src.CFGs["github.com/app.Func"] = "mock_cfg"
	src.CFGBlockStatements["github.com/app.Func"] = "mock_blockstmts"

	MergeCallGraphs(dst, src)

	assert.Equal(t, "mock_cfg", dst.CFGs["github.com/app.Func"])
	assert.Equal(t, "mock_blockstmts", dst.CFGBlockStatements["github.com/app.Func"])
}
