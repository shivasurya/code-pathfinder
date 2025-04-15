package eval

import (
	"testing"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateExpressionTree(t *testing.T) {
	// Create test data
	ctx := &EvaluationContext{
		RelationshipMap: buildTestRelationshipMap(),
		ProxyEnv:        buildTestEntityData(),
		EntityModel:     buildTestEntityModel(),
	}

	// Test cases
	testCases := []struct {
		name          string
		expr          *parser.ExpressionNode
		expectedData  []interface{}
		expectedError bool
	}{
		{
			name: "simple single entity comparison",
			expr: &parser.ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getName()",
					Alias:  "md",
					Entity: "method_declaration",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "\"onClick\"",
				},
			},
			expectedData: []interface{}{
				model.Method{
					ID:            "1",
					QualifiedName: "onClick",
					Name:          "onClick",
					Visibility:    "public",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateExpressionTree(tc.expr, ctx)
			if tc.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.ElementsMatch(t, tc.expectedData, result.Data)
		})
	}
}

func buildTestRelationshipMap() *RelationshipMap {
	rm := NewRelationshipMap()
	rm.AddRelationship("class", "methods", []string{"method"})
	rm.AddRelationship("method", "class", []string{"class"})
	return rm
}

func buildTestEntityModel() map[string][]interface{} {
	return map[string][]interface{}{
		"class_declaration": {
			model.Class{
				ClassId: "1",
				ClassOrInterface: model.ClassOrInterface{
					RefType: model.RefType{
						QualifiedName: "MyClass",
						Package:       "com.example",
					},
				},
			},
			model.Class{
				ClassId: "2",
				ClassOrInterface: model.ClassOrInterface{
					RefType: model.RefType{
						QualifiedName: "OtherClass",
						Package:       "com.example",
					},
				},
			},
		},
		"method_declaration": {
			model.Method{
				ID:            "1",
				QualifiedName: "onClick",
				Name:          "onClick",
				Visibility:    "public",
			},
			model.Method{
				ID:            "2",
				QualifiedName: "doOther",
				Name:          "doOther",
				Visibility:    "public",
			},
			model.Method{
				ID:            "3",
				QualifiedName: "doThird",
				Name:          "doThird",
				Visibility:    "public",
			},
			model.Method{
				ID:            "4",
				QualifiedName: "OtherClass",
				Name:          "OtherClass",
				Visibility:    "public",
			},
		},
	}
}

func buildTestEntityData() map[string][]map[string]interface{} {
	class1 := model.Class{
		ClassId: "1",
		ClassOrInterface: model.ClassOrInterface{
			RefType: model.RefType{
				QualifiedName: "MyClass",
				Package:       "com.example",
			},
		},
	}
	class2 := model.Class{
		ClassId: "2",
		ClassOrInterface: model.ClassOrInterface{
			RefType: model.RefType{
				QualifiedName: "OtherClass",
				Package:       "com.example",
			},
		},
	}
	method1 := model.Method{
		ID:            "1",
		QualifiedName: "onClick",
		Name:          "onClick",
		Visibility:    "public",
	}
	method2 := model.Method{
		ID:            "2",
		QualifiedName: "doOther",
		Name:          "doOther",
		Visibility:    "public",
	}
	method3 := model.Method{
		ID:            "3",
		QualifiedName: "doThird",
		Name:          "doThird",
		Visibility:    "public",
	}
	method4 := model.Method{
		ID:            "4",
		QualifiedName: "OtherClass",
		Name:          "OtherClass",
		Visibility:    "public",
	}
	return map[string][]map[string]interface{}{
		"class_declaration": {
			class1.GetProxyEnv(),
			class2.GetProxyEnv(),
		},
		"method_declaration": {
			method1.GetProxyEnv(),
			method2.GetProxyEnv(),
			method3.GetProxyEnv(),
			method4.GetProxyEnv(),
		},
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

func TestDetectComparisonType(t *testing.T) {
	tests := []struct {
		name     string
		node     *parser.ExpressionNode
		expected ComparisonType
		wantErr  bool
	}{
		{
			name: "single entity with literal",
			node: &parser.ExpressionNode{
				Type:     "binary",
				Operator: ">",
				Left: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getName()",
					Alias:  "md",
					Entity: "method_declaration",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "onClick",
				},
			},
			expected: SINGLE_ENTITY,
			wantErr:  false,
		},
		{
			name: "dual entity comparison",
			node: &parser.ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getName()",
					Alias:  "md",
					Entity: "method_declaration",
				},
				Right: &parser.ExpressionNode{
					Type:   "method_call",
					Value:  "getName()",
					Alias:  "cd",
					Entity: "class_declaration",
				},
			},
			expected: DUAL_ENTITY,
			wantErr:  false,
		},
		{
			name: "non-binary node",
			node: &parser.ExpressionNode{
				Type:  "literal",
				Value: "25",
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "nil node",
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
		"age":        30,
		"name":       "Alice",
		"complexity": func() int { return 10 },
		"hasAnnotation": func(annotation string) bool {
			return annotation == "@Test"
		},
	}
	tests := []struct {
		name     string
		node     *parser.ExpressionNode
		data     map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "simple variable",
			node: &parser.ExpressionNode{
				Type:     "binary",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:  "variable",
					Value: "age",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "30",
				},
			},
			data:     map[string]interface{}{"age": 30},
			expected: true,
			wantErr:  false,
		},
		{
			name: "method call",
			node: &parser.ExpressionNode{
				Type:     "binary",
				Value:    "",
				Operator: "==",
				Left: &parser.ExpressionNode{
					Type:  "method_call",
					Value: "complexity()",
				},
				Right: &parser.ExpressionNode{
					Type:  "literal",
					Value: "10",
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
