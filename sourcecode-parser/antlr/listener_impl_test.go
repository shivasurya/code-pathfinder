package parser

import (
	"reflect"
	"testing"
)

// compareQueryIgnoringExpressionTree compares two Query structs but ignores the ExpressionTree field
func compareQueryIgnoringExpressionTree(a, b Query) bool {
	// Compare all fields except ExpressionTree
	return reflect.DeepEqual(a.Classes, b.Classes) &&
		reflect.DeepEqual(a.SelectList, b.SelectList) &&
		a.Expression == b.Expression &&
		reflect.DeepEqual(a.Condition, b.Condition) &&
		reflect.DeepEqual(a.Predicate, b.Predicate) &&
		reflect.DeepEqual(a.PredicateInvocation, b.PredicateInvocation) &&
		reflect.DeepEqual(a.SelectOutput, b.SelectOutput)
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedQuery  Query
		expectedSelect []SelectList
		expectedExpr   string
	}{
		{
			name:  "Simple select with single entity",
			input: "FROM class_declaration AS cd WHERE cd.GetName() == \"test\" SELECT cd",
			expectedQuery: Query{
				SelectList: []SelectList{{Entity: "class_declaration", Alias: "cd"}},
				Expression: "cd.GetName()==\"test\"",
				Condition:  []string{"cd.GetName()==\"test\""},
				SelectOutput: []SelectOutput{
					{
						SelectEntity: "cd",
						Type:         "variable",
					},
				},
			},
		},
		{
			name:  "Select with multiple entities and aliases",
			input: "FROM entity1 AS e1, entity2 AS e2 WHERE e1.GetName() == \"test\" SELECT e1.GetName()",
			expectedQuery: Query{
				SelectList: []SelectList{
					{Entity: "entity1", Alias: "e1"},
					{Entity: "entity2", Alias: "e2"},
				},
				Expression: "e1.GetName()==\"test\"",
				Condition:  []string{"e1.GetName()==\"test\""},
				SelectOutput: []SelectOutput{
					{
						SelectEntity: "e1.GetName()",
						Type:         "method_chain",
					},
				},
			},
		},
		{
			name:  "Select with multiple entities and aliases",
			input: "FROM entity1 AS e1, entity2 AS e2 WHERE e1.GetName() == \"test\" || e2.GetName() == \"test\" SELECT e1.GetName()",
			expectedQuery: Query{
				SelectList: []SelectList{
					{Entity: "entity1", Alias: "e1"},
					{Entity: "entity2", Alias: "e2"},
				},
				Expression: "e1.GetName()==\"test\" || e2.GetName()==\"test\"",
				Condition:  []string{"e1.GetName()==\"test\"", "e2.GetName()==\"test\""},
				SelectOutput: []SelectOutput{
					{
						SelectEntity: "e1.GetName()",
						Type:         "method_chain",
					},
				},
			},
		},
		{
			name:  "Select with multiple entities and aliases",
			input: "FROM entity1 AS e1, entity2 AS e2 WHERE e1.GetName() == \"test\" && e2.GetName() == \"test\" SELECT e1.GetName()",
			expectedQuery: Query{
				SelectList: []SelectList{
					{Entity: "entity1", Alias: "e1"},
					{Entity: "entity2", Alias: "e2"},
				},
				Expression: "e1.GetName()==\"test\" && e2.GetName()==\"test\"",
				Condition:  []string{"e1.GetName()==\"test\"", "e2.GetName()==\"test\""},
				SelectOutput: []SelectOutput{
					{
						SelectEntity: "e1.GetName()",
						Type:         "method_chain",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQuery(tt.input)
			if err != nil {
				t.Errorf("ParseQuery() error = %v", err)
				return
			}

			// Use custom comparison function that ignores ExpressionTree
			if !compareQueryIgnoringExpressionTree(result, tt.expectedQuery) {
				t.Errorf("ParseQuery() = %v, want %v", result, tt.expectedQuery)
			}
		})
	}
}
