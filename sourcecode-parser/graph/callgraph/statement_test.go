package callgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatementGetDef(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *Statement
		expected string
	}{
		{
			name: "assignment with def",
			stmt: &Statement{
				Type: StatementTypeAssignment,
				Def:  "x",
			},
			expected: "x",
		},
		{
			name: "call without def",
			stmt: &Statement{
				Type: StatementTypeCall,
				Def:  "",
			},
			expected: "",
		},
		{
			name: "for loop with def",
			stmt: &Statement{
				Type: StatementTypeFor,
				Def:  "item",
			},
			expected: "item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.GetDef()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatementGetUses(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *Statement
		expected []string
	}{
		{
			name: "assignment with uses",
			stmt: &Statement{
				Type: StatementTypeAssignment,
				Uses: []string{"a", "b"},
			},
			expected: []string{"a", "b"},
		},
		{
			name: "call with no uses",
			stmt: &Statement{
				Type: StatementTypeCall,
				Uses: []string{},
			},
			expected: []string{},
		},
		{
			name: "if statement with condition uses",
			stmt: &Statement{
				Type: StatementTypeIf,
				Uses: []string{"flag", "count"},
			},
			expected: []string{"flag", "count"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.GetUses()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatementIsCall(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *Statement
		expected bool
	}{
		{
			name:     "call statement",
			stmt:     &Statement{Type: StatementTypeCall},
			expected: true,
		},
		{
			name:     "expression statement",
			stmt:     &Statement{Type: StatementTypeExpression},
			expected: true,
		},
		{
			name:     "assignment statement",
			stmt:     &Statement{Type: StatementTypeAssignment},
			expected: false,
		},
		{
			name:     "return statement",
			stmt:     &Statement{Type: StatementTypeReturn},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.IsCall()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatementIsAssignment(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *Statement
		expected bool
	}{
		{
			name:     "assignment statement",
			stmt:     &Statement{Type: StatementTypeAssignment},
			expected: true,
		},
		{
			name:     "call statement",
			stmt:     &Statement{Type: StatementTypeCall},
			expected: false,
		},
		{
			name:     "for statement",
			stmt:     &Statement{Type: StatementTypeFor},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.IsAssignment()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatementIsControlFlow(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *Statement
		expected bool
	}{
		{
			name:     "if statement",
			stmt:     &Statement{Type: StatementTypeIf},
			expected: true,
		},
		{
			name:     "for statement",
			stmt:     &Statement{Type: StatementTypeFor},
			expected: true,
		},
		{
			name:     "while statement",
			stmt:     &Statement{Type: StatementTypeWhile},
			expected: true,
		},
		{
			name:     "with statement",
			stmt:     &Statement{Type: StatementTypeWith},
			expected: true,
		},
		{
			name:     "try statement",
			stmt:     &Statement{Type: StatementTypeTry},
			expected: true,
		},
		{
			name:     "assignment statement",
			stmt:     &Statement{Type: StatementTypeAssignment},
			expected: false,
		},
		{
			name:     "call statement",
			stmt:     &Statement{Type: StatementTypeCall},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.IsControlFlow()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatementHasNestedStatements(t *testing.T) {
	tests := []struct {
		name     string
		stmt     *Statement
		expected bool
	}{
		{
			name: "if with nested statements",
			stmt: &Statement{
				Type: StatementTypeIf,
				NestedStatements: []*Statement{
					{Type: StatementTypeAssignment},
				},
			},
			expected: true,
		},
		{
			name: "if with else branch",
			stmt: &Statement{
				Type: StatementTypeIf,
				ElseBranch: []*Statement{
					{Type: StatementTypeReturn},
				},
			},
			expected: true,
		},
		{
			name: "simple assignment",
			stmt: &Statement{
				Type: StatementTypeAssignment,
			},
			expected: false,
		},
		{
			name: "if with both nested and else",
			stmt: &Statement{
				Type: StatementTypeIf,
				NestedStatements: []*Statement{
					{Type: StatementTypeAssignment},
				},
				ElseBranch: []*Statement{
					{Type: StatementTypeReturn},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.HasNestedStatements()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatementAllStatements(t *testing.T) {
	tests := []struct {
		name          string
		stmt          *Statement
		expectedCount int
	}{
		{
			name: "simple statement",
			stmt: &Statement{
				Type:       StatementTypeAssignment,
				LineNumber: 1,
			},
			expectedCount: 1,
		},
		{
			name: "if with one nested statement",
			stmt: &Statement{
				Type:       StatementTypeIf,
				LineNumber: 1,
				NestedStatements: []*Statement{
					{Type: StatementTypeAssignment, LineNumber: 2},
				},
			},
			expectedCount: 2,
		},
		{
			name: "if with nested and else",
			stmt: &Statement{
				Type:       StatementTypeIf,
				LineNumber: 1,
				NestedStatements: []*Statement{
					{Type: StatementTypeAssignment, LineNumber: 2},
					{Type: StatementTypeCall, LineNumber: 3},
				},
				ElseBranch: []*Statement{
					{Type: StatementTypeReturn, LineNumber: 5},
				},
			},
			expectedCount: 4,
		},
		{
			name: "deeply nested statements",
			stmt: &Statement{
				Type:       StatementTypeIf,
				LineNumber: 1,
				NestedStatements: []*Statement{
					{
						Type:       StatementTypeFor,
						LineNumber: 2,
						NestedStatements: []*Statement{
							{Type: StatementTypeAssignment, LineNumber: 3},
							{Type: StatementTypeCall, LineNumber: 4},
						},
					},
				},
			},
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.stmt.AllStatements()
			assert.Equal(t, tt.expectedCount, len(result))

			// Verify first statement is always the root
			assert.Equal(t, tt.stmt, result[0])
		})
	}
}

func TestNewDefUseChain(t *testing.T) {
	chain := NewDefUseChain()

	assert.NotNil(t, chain)
	assert.NotNil(t, chain.Defs)
	assert.NotNil(t, chain.Uses)
	assert.Equal(t, 0, len(chain.Defs))
	assert.Equal(t, 0, len(chain.Uses))
}

func TestDefUseChainAddDef(t *testing.T) {
	chain := NewDefUseChain()
	stmt1 := &Statement{Type: StatementTypeAssignment, LineNumber: 1, Def: "x"}
	stmt2 := &Statement{Type: StatementTypeAssignment, LineNumber: 2, Def: "x"}

	// Add first definition
	chain.AddDef("x", stmt1)
	assert.Equal(t, 1, len(chain.Defs["x"]))
	assert.Equal(t, stmt1, chain.Defs["x"][0])

	// Add second definition of same variable
	chain.AddDef("x", stmt2)
	assert.Equal(t, 2, len(chain.Defs["x"]))
	assert.Equal(t, stmt2, chain.Defs["x"][1])

	// Add definition for different variable
	stmt3 := &Statement{Type: StatementTypeAssignment, LineNumber: 3, Def: "y"}
	chain.AddDef("y", stmt3)
	assert.Equal(t, 1, len(chain.Defs["y"]))
	assert.Equal(t, stmt3, chain.Defs["y"][0])

	// Test empty variable name (should be ignored)
	chain.AddDef("", stmt1)
	_, exists := chain.Defs[""]
	assert.False(t, exists)
}

func TestDefUseChainAddUse(t *testing.T) {
	chain := NewDefUseChain()
	stmt1 := &Statement{Type: StatementTypeCall, LineNumber: 1, Uses: []string{"x"}}
	stmt2 := &Statement{Type: StatementTypeAssignment, LineNumber: 2, Uses: []string{"x"}}

	// Add first use
	chain.AddUse("x", stmt1)
	assert.Equal(t, 1, len(chain.Uses["x"]))
	assert.Equal(t, stmt1, chain.Uses["x"][0])

	// Add second use of same variable
	chain.AddUse("x", stmt2)
	assert.Equal(t, 2, len(chain.Uses["x"]))
	assert.Equal(t, stmt2, chain.Uses["x"][1])

	// Add use for different variable
	stmt3 := &Statement{Type: StatementTypeReturn, LineNumber: 3, Uses: []string{"y"}}
	chain.AddUse("y", stmt3)
	assert.Equal(t, 1, len(chain.Uses["y"]))
	assert.Equal(t, stmt3, chain.Uses["y"][0])

	// Test empty variable name (should be ignored)
	chain.AddUse("", stmt1)
	_, exists := chain.Uses[""]
	assert.False(t, exists)
}

func TestDefUseChainGetDefs(t *testing.T) {
	chain := NewDefUseChain()
	stmt1 := &Statement{Type: StatementTypeAssignment, LineNumber: 1, Def: "x"}
	stmt2 := &Statement{Type: StatementTypeAssignment, LineNumber: 2, Def: "x"}

	chain.AddDef("x", stmt1)
	chain.AddDef("x", stmt2)

	defs := chain.GetDefs("x")
	assert.Equal(t, 2, len(defs))
	assert.Equal(t, stmt1, defs[0])
	assert.Equal(t, stmt2, defs[1])

	// Test non-existent variable
	nonExistent := chain.GetDefs("nonexistent")
	assert.Nil(t, nonExistent)
}

func TestDefUseChainGetUses(t *testing.T) {
	chain := NewDefUseChain()
	stmt1 := &Statement{Type: StatementTypeCall, LineNumber: 1, Uses: []string{"x"}}
	stmt2 := &Statement{Type: StatementTypeAssignment, LineNumber: 2, Uses: []string{"x"}}

	chain.AddUse("x", stmt1)
	chain.AddUse("x", stmt2)

	uses := chain.GetUses("x")
	assert.Equal(t, 2, len(uses))
	assert.Equal(t, stmt1, uses[0])
	assert.Equal(t, stmt2, uses[1])

	// Test non-existent variable
	nonExistent := chain.GetUses("nonexistent")
	assert.Nil(t, nonExistent)
}

func TestDefUseChainIsDefined(t *testing.T) {
	chain := NewDefUseChain()
	stmt := &Statement{Type: StatementTypeAssignment, LineNumber: 1, Def: "x"}

	assert.False(t, chain.IsDefined("x"))

	chain.AddDef("x", stmt)
	assert.True(t, chain.IsDefined("x"))
	assert.False(t, chain.IsDefined("y"))
}

func TestDefUseChainIsUsed(t *testing.T) {
	chain := NewDefUseChain()
	stmt := &Statement{Type: StatementTypeCall, LineNumber: 1, Uses: []string{"x"}}

	assert.False(t, chain.IsUsed("x"))

	chain.AddUse("x", stmt)
	assert.True(t, chain.IsUsed("x"))
	assert.False(t, chain.IsUsed("y"))
}

func TestDefUseChainAllVariables(t *testing.T) {
	chain := NewDefUseChain()

	stmt1 := &Statement{Type: StatementTypeAssignment, LineNumber: 1, Def: "x"}
	stmt2 := &Statement{Type: StatementTypeCall, LineNumber: 2, Uses: []string{"y"}}
	stmt3 := &Statement{Type: StatementTypeAssignment, LineNumber: 3, Def: "z", Uses: []string{"x"}}

	chain.AddDef("x", stmt1)
	chain.AddUse("y", stmt2)
	chain.AddDef("z", stmt3)
	chain.AddUse("x", stmt3)

	vars := chain.AllVariables()
	assert.Equal(t, 3, len(vars))

	// Create a set to check presence
	varSet := make(map[string]bool)
	for _, v := range vars {
		varSet[v] = true
	}

	assert.True(t, varSet["x"])
	assert.True(t, varSet["y"])
	assert.True(t, varSet["z"])
}

func TestDefUseChainComplexScenario(t *testing.T) {
	// Simulate a real code scenario:
	// 1: x = 5
	// 2: y = x + 10
	// 3: if y > 15:
	// 4:     z = x * 2
	// 5:     print(z)

	chain := NewDefUseChain()

	stmt1 := &Statement{Type: StatementTypeAssignment, LineNumber: 1, Def: "x"}
	stmt2 := &Statement{Type: StatementTypeAssignment, LineNumber: 2, Def: "y", Uses: []string{"x"}}
	stmt3 := &Statement{Type: StatementTypeIf, LineNumber: 3, Uses: []string{"y"}}
	stmt4 := &Statement{Type: StatementTypeAssignment, LineNumber: 4, Def: "z", Uses: []string{"x"}}
	stmt5 := &Statement{Type: StatementTypeCall, LineNumber: 5, Uses: []string{"z"}}

	chain.AddDef("x", stmt1)

	chain.AddDef("y", stmt2)
	chain.AddUse("x", stmt2)

	chain.AddUse("y", stmt3)

	chain.AddDef("z", stmt4)
	chain.AddUse("x", stmt4)

	chain.AddUse("z", stmt5)

	// Verify x: 1 def, 2 uses
	assert.Equal(t, 1, len(chain.GetDefs("x")))
	assert.Equal(t, 2, len(chain.GetUses("x")))

	// Verify y: 1 def, 1 use
	assert.Equal(t, 1, len(chain.GetDefs("y")))
	assert.Equal(t, 1, len(chain.GetUses("y")))

	// Verify z: 1 def, 1 use
	assert.Equal(t, 1, len(chain.GetDefs("z")))
	assert.Equal(t, 1, len(chain.GetUses("z")))

	// All variables
	vars := chain.AllVariables()
	assert.Equal(t, 3, len(vars))
}
