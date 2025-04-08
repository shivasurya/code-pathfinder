package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateExpressionTree(t *testing.T) {
	// Create a relationship map
	rm := NewRelationshipMap()
	rm.AddRelationship("class", "methods", []string{"method"})

	// Create test data
	testData := map[string][]map[string]interface{}{
		"class": {
			{"id": 1, "name": "Class1", "age": 30},
			{"id": 2, "name": "Class2", "age": 20},
			{"id": 3, "name": "Class3", "age": 40},
		},
		"method": {
			{"id": 1, "class_id": 1, "name": "Method1"},
			{"id": 2, "class_id": 2, "name": "Method2"},
		},
	}

	// Create evaluation context
	ctx := &EvaluationContext{
		RelationshipMap: rm,
		EntityData:      testData,
	}
	tests := []struct {
		name     string
		tree     *ExpressionNode
		expected *EvaluationResult
		wantErr  bool
	}{
		{
			name: "single entity comparison",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: ">",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "class.age",
				},
				Right: &ExpressionNode{
					Type:  "literal",
					Value: "25",
				},
			},
			expected: &EvaluationResult{
				Data: []map[string]interface{}{
					{"id": 1, "name": "Class1", "age": 30},
					{"id": 3, "name": "Class3", "age": 40},
				},
				Entities: []string{"class"},
			},
			wantErr: false,
		},
		{
			name: "related entities comparison",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "class.name",
				},
				Right: &ExpressionNode{
					Type:  "variable",
					Value: "method.name",
				},
			},
			expected: &EvaluationResult{
				Data: []map[string]interface{}{
					{"id": 1, "name": "Class1", "age": 30, "method_name": "Method1", "class_id": 1},
				},
				Entities: []string{"class", "method"},
			},
			wantErr: false,
		},
		{
			name: "nil tree",
			tree: nil,
			expected: &EvaluationResult{},
			wantErr:  false,
		},
		{
			name: "complex OR condition",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: "||",
				Left: &ExpressionNode{
					Type:     "binary",
					Operator: "<",
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
			expected: &EvaluationResult{
				Data: []map[string]interface{}{
					{"id": 2, "name": "Class2", "age": 20},
					{"id": 3, "name": "Class3", "age": 40},
				},
				Entities: []string{"class"},
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
			expected: &EvaluationResult{
				Data: []map[string]interface{}{
					{"id": 1, "name": "Class1", "age": 30},
					{"id": 3, "name": "Class3", "age": 40},
				},
				Entities: []string{"class"},
			},
			wantErr: false,
		},
		{
			name: "string comparison",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "class.name",
				},
				Right: &ExpressionNode{
					Type:  "literal",
					Value: "\"Class1\"",
				},
			},
			expected: &EvaluationResult{
				Data: []map[string]interface{}{
					{"id": 1, "name": "Class1", "age": 30},
				},
				Entities: []string{"class"},
			},
			wantErr: false,
		},
		{
			name: "invalid operator",
			tree: &ExpressionNode{
				Type:     "binary",
				Operator: "invalid",
			},
			expected: &EvaluationResult{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpressionTree(tt.tree, ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Data, got.Data)
			assert.Equal(t, tt.expected.Entities, got.Entities)
		})
	}
}

func TestRelationshipMap(t *testing.T) {
	// Create a relationship map
	rm := NewRelationshipMap()

	// Add some relationships
	rm.AddRelationship("class", "methods", []string{"method", "function"})
	rm.AddRelationship("method", "parameters", []string{"parameter", "variable"})
	rm.AddRelationship("function", "returns", []string{"type", "class"})

	tests := []struct {
		name     string
		entity1  string
		entity2  string
		expected bool
	}{
		{
			name:     "direct relationship exists",
			entity1:  "class",
			entity2:  "method",
			expected: true,
		},
		{
			name:     "reverse relationship exists",
			entity1:  "function",
			entity2:  "class",
			expected: true,
		},
		{
			name:     "no relationship exists",
			entity1:  "class",
			entity2:  "parameter",
			expected: false,
		},
		{
			name:     "unknown entity",
			entity1:  "unknown",
			entity2:  "class",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.HasRelationship(tt.entity1, tt.entity2)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCheckExpressionRelationship(t *testing.T) {
	// Create a relationship map
	rm := NewRelationshipMap()
	rm.AddRelationship("class", "methods", []string{"method"})

	tests := []struct {
		name     string
		node     *ExpressionNode
		expected bool
		wantErr  bool
	}{
		{
			name: "related entities comparison",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "class",
				},
				Right: &ExpressionNode{
					Type:  "variable",
					Value: "method",
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "unrelated entities comparison",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "class",
				},
				Right: &ExpressionNode{
					Type:  "variable",
					Value: "unrelated",
				},
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "single entity comparison",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: ">",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "class",
				},
				Right: &ExpressionNode{
					Type:  "literal",
					Value: "10",
				},
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "non-binary node",
			node: &ExpressionNode{
				Type:  "literal",
				Value: "25",
			},
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckExpressionRelationship(tt.node, rm)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetectComparisonType(t *testing.T) {
	tests := []struct {
		name     string
		node     *ExpressionNode
		expected ComparisonType
		wantErr  bool
	}{
		{
			name: "single entity with literal",
			node: &ExpressionNode{
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
			expected: SINGLE_ENTITY,
			wantErr:  false,
		},
		{
			name: "dual entity comparison",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &ExpressionNode{
					Type:  "variable",
					Value: "age",
				},
				Right: &ExpressionNode{
					Type:  "variable",
					Value: "count",
				},
			},
			expected: DUAL_ENTITY,
			wantErr:  false,
		},
		{
			name: "single entity method call",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: ">",
				Left: &ExpressionNode{
					Type:  "method_call",
					Value: "method.complexity",
				},
				Right: &ExpressionNode{
					Type:  "literal",
					Value: "10",
				},
			},
			expected: SINGLE_ENTITY,
			wantErr:  false,
		},
		{
			name: "dual entity method calls",
			node: &ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &ExpressionNode{
					Type:  "method_call",
					Value: "method1.complexity",
				},
				Right: &ExpressionNode{
					Type:  "method_call",
					Value: "method2.complexity",
				},
			},
			expected: DUAL_ENTITY,
			wantErr:  false,
		},
		{
			name: "non-binary node",
			node: &ExpressionNode{
				Type:  "literal",
				Value: "25",
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "nil node",
			node:     nil,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectComparisonType(tt.node)
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
