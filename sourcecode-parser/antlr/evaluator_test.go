package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateExpressionTree(t *testing.T) {
	tests := []struct {
		name     string
		tree     *ExpressionNode
		data     []map[string]interface{}
		expected []map[string]interface{}
		wantErr  bool
	}{
		{
			name: "simple comparison",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: ">",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "age",
				},
				Right: &ExpressionNode{
					Type:  "literal",
					Value: "25",
				},
			},
			data: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 20},
				{"name": "Charlie", "age": 40},
			},
			expected: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Charlie", "age": 40},
			},
			wantErr: false,
		},
		{
			name: "complex AND condition",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: "&&",
				Left: &ExpressionNode{
					Type:     "binary",
					Operator: ">",
					Left: &ExpressionNode{
						Type:  "variable",
						Value: "age",
					},
					Right: &ExpressionNode{
						Type:  "literal",
						Value: "25",
					},
				},
				Right: &ExpressionNode{
					Type:     "binary",
					Operator: "<",
					Left: &ExpressionNode{
						Type:  "variable",
						Value: "age",
					},
					Right: &ExpressionNode{
						Type:  "literal",
						Value: "35",
					},
				},
			},
			data: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 20},
				{"name": "Charlie", "age": 40},
			},
			expected: []map[string]interface{}{
				{"name": "Alice", "age": 30},
			},
			wantErr: false,
		},
		{
			name: "nil tree",
			tree: nil,
			data: []map[string]interface{}{
				{"name": "Alice", "age": 30},
			},
			expected: []map[string]interface{}{
				{"name": "Alice", "age": 30},
			},
			wantErr: false,
		},
		{
			name: "complex OR condition",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: "||",
				Left: &ExpressionNode{
					Type:     "binary",
					Operator: "==",
					Left: &ExpressionNode{
						Type:  "variable",
						Value: "name",
					},
					Right: &ExpressionNode{
						Type:  "literal",
						Value: "\"Alice\"",
					},
				},
				Right: &ExpressionNode{
					Type:     "binary",
					Operator: ">",
					Left: &ExpressionNode{
						Type:  "variable",
						Value: "age",
					},
					Right: &ExpressionNode{
						Type:  "literal",
						Value: "35",
					},
				},
			},
			data: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 40},
				{"name": "Charlie", "age": 25},
			},
			expected: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 40},
			},
			wantErr: false,
		},
		{
			name: "unary NOT condition",
			tree: &ExpressionNode{
				Type:     "unary",
				Operator: "!",
				Right: &ExpressionNode{
					Type:     "binary",
					Operator: "<",
					Left: &ExpressionNode{
						Type:  "variable",
						Value: "age",
					},
					Right: &ExpressionNode{
						Type:  "literal",
						Value: "30",
					},
				},
			},
			data: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
			},
			expected: []map[string]interface{}{
				{"name": "Alice", "age": 30},
				{"name": "Charlie", "age": 35},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpressionTree(tt.tree, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEvaluateNode(t *testing.T) {
	// Mock data with method and predicate functions
	testData := map[string]interface{}{
		"age": 30,
		"name": "Alice",
		"complexity": func() int { return 10 },
		"hasAnnotation": func(annotation string) bool {
			return annotation == "@Test"
		},
	}
	tests := []struct {
		name     string
		node     *ExpressionNode
		data     map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "simple variable",
			node: &ExpressionNode{
				Type:  "variable",
				Value: "age",
			},
			data:     map[string]interface{}{"age": 30},
			expected: 30,
			wantErr:  false,
		},
		{
			name: "simple literal",
			node: &ExpressionNode{
				Type:  "literal",
				Value: "25",
			},
			data:     map[string]interface{}{},
			expected: 25,
			wantErr:  false,
		},
		{
			name: "method call",
			node: &ExpressionNode{
				Type:  "method_call",
				Value: "complexity",
			},
			data:     testData,
			expected: 10,
			wantErr:  false,
		},
		{
			name: "method call with args",
			node: &ExpressionNode{
				Type:  "method_call",
				Value: "hasAnnotation",
				Args: []ExpressionNode{
					{
						Type:  "literal",
						Value: "\"@Test\"",
					},
				},
			},
			data:     testData,
			expected: true,
			wantErr:  false,
		},
		{
			name: "complex expression with method call",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: "&&",
				Left: &ExpressionNode{
					Type:     "binary",
					Operator: ">",
					Left: &ExpressionNode{
						Type:  "method_call",
						Value: "complexity",
					},
					Right: &ExpressionNode{
						Type:  "literal",
						Value: "5",
					},
				},
				Right: &ExpressionNode{
					Type:  "method_call",
					Value: "hasAnnotation",
					Args: []ExpressionNode{
						{
							Type:  "literal",
							Value: "\"@Test\"",
						},
					},
				},
			},
			data:     testData,
			expected: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateNode(tt.node, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
