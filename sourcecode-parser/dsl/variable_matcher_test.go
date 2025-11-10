package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/stretchr/testify/assert"
)

func TestVariableMatcherExecutor_Execute(t *testing.T) {
	cg := callgraph.NewCallGraph()

	cg.CallSites["test.main"] = []callgraph.CallSite{
		{
			Target: "eval",
			Arguments: []callgraph.Argument{
				{Value: "user_input", IsVariable: true, Position: 0},
			},
			Location: callgraph.Location{File: "test.py", Line: 10},
		},
		{
			Target: "print",
			Arguments: []callgraph.Argument{
				{Value: "\"hello\"", IsVariable: false, Position: 0},
			},
			Location: callgraph.Location{File: "test.py", Line: 15},
		},
	}

	t.Run("exact match", func(t *testing.T) {
		ir := &VariableMatcherIR{
			Pattern:  "user_input",
			Wildcard: false,
		}

		executor := NewVariableMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1)
		assert.Equal(t, "user_input", matches[0].VariableName)
		assert.Equal(t, 0, matches[0].ArgumentPos)
	})

	t.Run("wildcard prefix", func(t *testing.T) {
		cg2 := callgraph.NewCallGraph()
		cg2.CallSites["test.main"] = []callgraph.CallSite{
			{
				Target: "process",
				Arguments: []callgraph.Argument{
					{Value: "user_input", IsVariable: true},
					{Value: "user_id", IsVariable: true},
					{Value: "admin_name", IsVariable: true},
				},
			},
		}

		ir := &VariableMatcherIR{
			Pattern:  "user_*",
			Wildcard: true,
		}

		executor := NewVariableMatcherExecutor(ir, cg2)
		matches := executor.Execute()

		assert.Len(t, matches, 2) // user_input, user_id
	})

	t.Run("no matches - literal argument", func(t *testing.T) {
		ir := &VariableMatcherIR{
			Pattern:  "user_input",
			Wildcard: false,
		}

		cg2 := callgraph.NewCallGraph()
		cg2.CallSites["test.main"] = []callgraph.CallSite{
			{
				Target: "print",
				Arguments: []callgraph.Argument{
					{Value: "\"literal\"", IsVariable: false}, // NOT a variable
				},
			},
		}

		executor := NewVariableMatcherExecutor(ir, cg2)
		matches := executor.Execute()

		assert.Len(t, matches, 0)
	})
}
