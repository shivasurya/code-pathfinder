package callgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//
// ========== BASIC CONSTRUCTION TESTS ==========
//

func TestBuildDefUseChains_Empty(t *testing.T) {
	statements := []*Statement{}

	chain := BuildDefUseChains(statements)

	assert.NotNil(t, chain)
	assert.NotNil(t, chain.Defs)
	assert.NotNil(t, chain.Uses)
	assert.Equal(t, 0, len(chain.Defs))
	assert.Equal(t, 0, len(chain.Uses))
}

func TestBuildDefUseChains_SingleDefinition(t *testing.T) {
	// x = 10
	stmt1 := &Statement{
		LineNumber: 1,
		Type:       StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{},
	}

	chain := BuildDefUseChains([]*Statement{stmt1})

	assert.Equal(t, 1, len(chain.Defs))
	assert.Equal(t, 1, len(chain.Defs["x"]))
	assert.Equal(t, stmt1, chain.Defs["x"][0])

	// x is defined but not used
	assert.Equal(t, 0, len(chain.Uses["x"]))
}

func TestBuildDefUseChains_SingleUse(t *testing.T) {
	// print(x)  - x is used but not defined (parameter)
	stmt1 := &Statement{
		LineNumber: 1,
		Type:       StatementTypeCall,
		Def:        "",
		Uses:       []string{"x"},
	}

	chain := BuildDefUseChains([]*Statement{stmt1})

	assert.Equal(t, 1, len(chain.Uses))
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, stmt1, chain.Uses["x"][0])

	// x is used but not defined
	assert.Equal(t, 0, len(chain.Defs["x"]))
}

func TestBuildDefUseChains_DefThenUse(t *testing.T) {
	// x = 10
	// y = x
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "y", Uses: []string{"x"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2})

	// x: defined at stmt1, used at stmt2
	assert.Equal(t, 1, len(chain.Defs["x"]))
	assert.Equal(t, stmt1, chain.Defs["x"][0])
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, stmt2, chain.Uses["x"][0])

	// y: defined at stmt2, not used
	assert.Equal(t, 1, len(chain.Defs["y"]))
	assert.Equal(t, stmt2, chain.Defs["y"][0])
	assert.Equal(t, 0, len(chain.Uses["y"]))
}

//
// ========== MULTIPLE DEFINITIONS TESTS ==========
//

func TestBuildDefUseChains_MultipleDefinitions(t *testing.T) {
	// x = source()
	// y = x
	// x = safe()    ← Redefinition!
	// z = x
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "y", Uses: []string{"x"}}
	stmt3 := &Statement{LineNumber: 3, Def: "x", Uses: []string{}}
	stmt4 := &Statement{LineNumber: 4, Def: "z", Uses: []string{"x"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2, stmt3, stmt4})

	// x: TWO definitions (stmt1 and stmt3)
	assert.Equal(t, 2, len(chain.Defs["x"]), "x should have 2 definitions")
	assert.Contains(t, chain.Defs["x"], stmt1)
	assert.Contains(t, chain.Defs["x"], stmt3)

	// x: TWO uses (stmt2 and stmt4)
	assert.Equal(t, 2, len(chain.Uses["x"]), "x should have 2 uses")
	assert.Contains(t, chain.Uses["x"], stmt2)
	assert.Contains(t, chain.Uses["x"], stmt4)
}

func TestBuildDefUseChains_AugmentedAssignment(t *testing.T) {
	// x = 10
	// x += 5     ← Both Def and Use!
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "x", Uses: []string{"x"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2})

	// x: TWO definitions (stmt1 and stmt2)
	assert.Equal(t, 2, len(chain.Defs["x"]))

	// x: ONE use (stmt2)
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, stmt2, chain.Uses["x"][0])
}

//
// ========== MULTIPLE USES TESTS ==========
//

func TestBuildDefUseChains_MultipleUses(t *testing.T) {
	// x = 10
	// y = x
	// z = x
	// w = x
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "y", Uses: []string{"x"}}
	stmt3 := &Statement{LineNumber: 3, Def: "z", Uses: []string{"x"}}
	stmt4 := &Statement{LineNumber: 4, Def: "w", Uses: []string{"x"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2, stmt3, stmt4})

	// x: one definition
	assert.Equal(t, 1, len(chain.Defs["x"]))

	// x: THREE uses
	assert.Equal(t, 3, len(chain.Uses["x"]))
	assert.Contains(t, chain.Uses["x"], stmt2)
	assert.Contains(t, chain.Uses["x"], stmt3)
	assert.Contains(t, chain.Uses["x"], stmt4)
}

//
// ========== COMPLEX FLOW TESTS ==========
//

func TestBuildDefUseChains_LinearChain(t *testing.T) {
	// x = source()
	// y = x
	// z = y
	// sink(z)
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "y", Uses: []string{"x"}}
	stmt3 := &Statement{LineNumber: 3, Def: "z", Uses: []string{"y"}}
	stmt4 := &Statement{LineNumber: 4, Def: "", Uses: []string{"z"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2, stmt3, stmt4})

	// x: defined once, used once
	assert.Equal(t, 1, len(chain.Defs["x"]))
	assert.Equal(t, 1, len(chain.Uses["x"]))

	// y: defined once, used once
	assert.Equal(t, 1, len(chain.Defs["y"]))
	assert.Equal(t, 1, len(chain.Uses["y"]))

	// z: defined once, used once
	assert.Equal(t, 1, len(chain.Defs["z"]))
	assert.Equal(t, 1, len(chain.Uses["z"]))
}

func TestBuildDefUseChains_MultipleVariablesPerStatement(t *testing.T) {
	// result = func(x, y, z)  ← Uses 3 variables, defines 1
	stmt := &Statement{
		LineNumber: 1,
		Def:        "result",
		Uses:       []string{"func", "x", "y", "z"},
	}

	chain := BuildDefUseChains([]*Statement{stmt})

	// result: defined
	assert.Equal(t, 1, len(chain.Defs["result"]))

	// func, x, y, z: used
	assert.Equal(t, 1, len(chain.Uses["func"]))
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, 1, len(chain.Uses["y"]))
	assert.Equal(t, 1, len(chain.Uses["z"]))
}

//
// ========== HELPER METHOD TESTS ==========
//

func TestDefUseChain_GetDefs(t *testing.T) {
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	chain := BuildDefUseChains([]*Statement{stmt1})

	// Defined variable
	defs := chain.GetDefs("x")
	assert.Equal(t, 1, len(defs))
	assert.Equal(t, stmt1, defs[0])

	// Undefined variable (should return empty slice, not nil)
	defs = chain.GetDefs("undefined")
	assert.NotNil(t, defs, "Should return empty slice, not nil")
	assert.Equal(t, 0, len(defs))
}

func TestDefUseChain_GetUses(t *testing.T) {
	stmt1 := &Statement{LineNumber: 1, Def: "", Uses: []string{"x"}}
	chain := BuildDefUseChains([]*Statement{stmt1})

	// Used variable
	uses := chain.GetUses("x")
	assert.Equal(t, 1, len(uses))
	assert.Equal(t, stmt1, uses[0])

	// Unused variable (should return empty slice, not nil)
	uses = chain.GetUses("unused")
	assert.NotNil(t, uses, "Should return empty slice, not nil")
	assert.Equal(t, 0, len(uses))
}

func TestDefUseChain_IsDefined(t *testing.T) {
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	chain := BuildDefUseChains([]*Statement{stmt1})

	assert.True(t, chain.IsDefined("x"))
	assert.False(t, chain.IsDefined("y"))
}

func TestDefUseChain_IsUsed(t *testing.T) {
	stmt1 := &Statement{LineNumber: 1, Def: "", Uses: []string{"x"}}
	chain := BuildDefUseChains([]*Statement{stmt1})

	assert.True(t, chain.IsUsed("x"))
	assert.False(t, chain.IsUsed("y"))
}

func TestDefUseChain_AllVariables(t *testing.T) {
	// x = 10     (x defined)
	// y = x      (x used, y defined)
	// print(z)   (z used, not defined - parameter)
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "y", Uses: []string{"x"}}
	stmt3 := &Statement{LineNumber: 3, Def: "", Uses: []string{"z"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2, stmt3})

	vars := chain.AllVariables()

	// Should include x, y, z (all mentioned variables)
	assert.Equal(t, 3, len(vars))
	assert.Contains(t, vars, "x")
	assert.Contains(t, vars, "y")
	assert.Contains(t, vars, "z")
}

func TestDefUseChain_ComputeStats(t *testing.T) {
	// x = 10        (x defined)
	// x = 20        (x defined again)
	// y = x         (x used, y defined)
	// z = x + y     (x used, y used, z defined)
	// unused = 5    (unused defined, never used - dead variable)
	// print(param)  (param used, never defined - parameter)
	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{}}
	stmt2 := &Statement{LineNumber: 2, Def: "x", Uses: []string{}}
	stmt3 := &Statement{LineNumber: 3, Def: "y", Uses: []string{"x"}}
	stmt4 := &Statement{LineNumber: 4, Def: "z", Uses: []string{"x", "y"}}
	stmt5 := &Statement{LineNumber: 5, Def: "unused", Uses: []string{}}
	stmt6 := &Statement{LineNumber: 6, Def: "", Uses: []string{"param"}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2, stmt3, stmt4, stmt5, stmt6})

	stats := chain.ComputeStats()

	assert.Equal(t, 5, stats.NumVariables, "x, y, z, unused, param")
	assert.Equal(t, 5, stats.NumDefs, "x twice, y once, z once, unused once")
	assert.Equal(t, 4, stats.NumUses, "x twice, y once, param once")
	assert.Equal(t, 2, stats.MaxDefsPerVariable, "x has 2 definitions")
	assert.Equal(t, 2, stats.MaxUsesPerVariable, "x has 2 uses")
	assert.Equal(t, 1, stats.UndefinedVariables, "param is used but not defined")
	assert.Equal(t, 2, stats.DeadVariables, "z and unused are defined but not used")
}

//
// ========== INTEGRATION TESTS ==========
//

func TestBuildDefUseChains_RealisticFunction(t *testing.T) {
	// def vulnerable():
	//     x = request.GET['input']  # Line 1: source
	//     y = x.upper()              # Line 2
	//     z = y                      # Line 3
	//     eval(z)                    # Line 4: sink
	//     return None                # Line 5

	stmt1 := &Statement{LineNumber: 1, Def: "x", Uses: []string{"request"}}
	stmt2 := &Statement{LineNumber: 2, Def: "y", Uses: []string{"x"}}
	stmt3 := &Statement{LineNumber: 3, Def: "z", Uses: []string{"y"}}
	stmt4 := &Statement{LineNumber: 4, Def: "", Uses: []string{"eval", "z"}}
	stmt5 := &Statement{LineNumber: 5, Def: "", Uses: []string{}}

	chain := BuildDefUseChains([]*Statement{stmt1, stmt2, stmt3, stmt4, stmt5})

	// x: defined at line 1, used at line 2
	assert.Equal(t, 1, len(chain.Defs["x"]))
	assert.Equal(t, uint32(1), chain.Defs["x"][0].LineNumber)
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, uint32(2), chain.Uses["x"][0].LineNumber)

	// y: defined at line 2, used at line 3
	assert.Equal(t, 1, len(chain.Defs["y"]))
	assert.Equal(t, uint32(2), chain.Defs["y"][0].LineNumber)
	assert.Equal(t, 1, len(chain.Uses["y"]))
	assert.Equal(t, uint32(3), chain.Uses["y"][0].LineNumber)

	// z: defined at line 3, used at line 4
	assert.Equal(t, 1, len(chain.Defs["z"]))
	assert.Equal(t, uint32(3), chain.Defs["z"][0].LineNumber)
	assert.Equal(t, 1, len(chain.Uses["z"]))
	assert.Equal(t, uint32(4), chain.Uses["z"][0].LineNumber)

	// request: used but not defined (parameter or imported module)
	assert.Equal(t, 0, len(chain.Defs["request"]))
	assert.Equal(t, 1, len(chain.Uses["request"]))

	// eval: used but not defined (builtin function)
	assert.Equal(t, 0, len(chain.Defs["eval"]))
	assert.Equal(t, 1, len(chain.Uses["eval"]))
}

//
// ========== EDGE CASE TESTS ==========
//

func TestBuildDefUseChains_SameVariableDefAndUse(t *testing.T) {
	// x = x + 1  ← Both Def and Use in same statement
	stmt := &Statement{
		LineNumber: 1,
		Def:        "x",
		Uses:       []string{"x"},
	}

	chain := BuildDefUseChains([]*Statement{stmt})

	// x appears in both Defs and Uses
	assert.Equal(t, 1, len(chain.Defs["x"]))
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, stmt, chain.Defs["x"][0])
	assert.Equal(t, stmt, chain.Uses["x"][0])
}

func TestBuildDefUseChains_NoVariables(t *testing.T) {
	// pass  or  return  (no defs, no uses)
	stmt := &Statement{
		LineNumber: 1,
		Def:        "",
		Uses:       []string{},
	}

	chain := BuildDefUseChains([]*Statement{stmt})

	assert.Equal(t, 0, len(chain.Defs))
	assert.Equal(t, 0, len(chain.Uses))
}

func TestBuildDefUseChains_NilStatements(t *testing.T) {
	// Defensive test for nil slice
	chain := BuildDefUseChains(nil)

	assert.NotNil(t, chain)
	assert.NotNil(t, chain.Defs)
	assert.NotNil(t, chain.Uses)
	assert.Equal(t, 0, len(chain.Defs))
	assert.Equal(t, 0, len(chain.Uses))
}

func TestBuildDefUseChains_EmptyDef(t *testing.T) {
	// Statement with empty Def (call or return)
	stmt := &Statement{
		LineNumber: 1,
		Def:        "",
		Uses:       []string{"x", "y"},
	}

	chain := BuildDefUseChains([]*Statement{stmt})

	// No defs should be tracked
	assert.Equal(t, 0, len(chain.Defs))

	// Uses should be tracked
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, 1, len(chain.Uses["y"]))
}

func TestBuildDefUseChains_EmptyUses(t *testing.T) {
	// Statement with empty Uses (assignment from literal)
	stmt := &Statement{
		LineNumber: 1,
		Def:        "x",
		Uses:       []string{},
	}

	chain := BuildDefUseChains([]*Statement{stmt})

	// Def should be tracked
	assert.Equal(t, 1, len(chain.Defs["x"]))

	// No uses
	assert.Equal(t, 0, len(chain.Uses))
}
