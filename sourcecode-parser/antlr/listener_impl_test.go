package parser

import (
	"reflect"
	"testing"
)

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
			input: "FIND class_declaration AS cd WHERE cd.GetName() == \"test\"",
			expectedQuery: Query{
				SelectList: []SelectList{{Entity: "class_declaration", Alias: "cd"}},
				Expression: "cd.GetName()==\"test\"",
				Condition:  []string{"cd.GetName()==\"test\""},
			},
		},
		{
			name:  "Select with multiple entities and aliases",
			input: "FIND entity1 AS e1, entity2 AS e2 WHERE e1.GetName() == \"test\"",
			expectedQuery: Query{
				SelectList: []SelectList{
					{Entity: "entity1", Alias: "e1"},
					{Entity: "entity2", Alias: "e2"},
				},
				Expression: "e1.GetName()==\"test\"",
				Condition:  []string{"e1.GetName()==\"test\""},
			},
		},
		{
			name:  "Select with multiple entities and aliases",
			input: "FIND entity1 AS e1, entity2 AS e2 WHERE e1.GetName() == \"test\" || e2.GetName() == \"test\"",
			expectedQuery: Query{
				SelectList: []SelectList{
					{Entity: "entity1", Alias: "e1"},
					{Entity: "entity2", Alias: "e2"},
				},
				Expression: "e1.GetName()==\"test\" || e2.GetName()==\"test\"",
				Condition:  []string{"e1.GetName()==\"test\"", "e2.GetName()==\"test\""},
			},
		},
		{
			name:  "Select with multiple entities and aliases",
			input: "FIND entity1 AS e1, entity2 AS e2 WHERE e1.GetName() == \"test\" && e2.GetName() == \"test\"",
			expectedQuery: Query{
				SelectList: []SelectList{
					{Entity: "entity1", Alias: "e1"},
					{Entity: "entity2", Alias: "e2"},
				},
				Expression: "e1.GetName()==\"test\" && e2.GetName()==\"test\"",
				Condition:  []string{"e1.GetName()==\"test\"", "e2.GetName()==\"test\""},
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
			if !reflect.DeepEqual(result, tt.expectedQuery) {
				t.Errorf("ParseQuery() = %v, want %v", result, tt.expectedQuery)
			}
		})
	}
}
