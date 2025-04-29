package eval

import (
	"testing"
)

func TestRelationshipMapOperations(t *testing.T) {
	t.Run("test add and check relationships", func(t *testing.T) {
		rm := NewRelationshipMap()

		// Test adding relationships
		rm.AddRelationship("class1", "extends", []string{"class2"})
		rm.AddRelationship("class3", "implements", []string{"interface1", "interface2"})

		// Test direct relationships
		testCases := []struct {
			entity1   string
			entity2   string
			expected  bool
			testName  string
		}{
			{"class1", "class2", true, "direct relationship exists"},
			{"class2", "class1", true, "bidirectional relationship exists"},
			{"class3", "interface1", true, "multiple relationships exist"},
			{"class3", "interface2", true, "multiple relationships exist"},
			{"interface1", "interface2", false, "unrelated entities"},
			{"class1", "class3", false, "unrelated classes"},
			{"nonexistent", "class1", false, "nonexistent entity"},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				result := rm.HasRelationship(tc.entity1, tc.entity2)
				if result != tc.expected {
					t.Errorf("HasRelationship(%s, %s) = %v; want %v", 
						tc.entity1, tc.entity2, result, tc.expected)
				}
			})
		}
	})

	t.Run("test relationship attributes", func(t *testing.T) {
		rm := NewRelationshipMap()

		// Add relationships with different attributes
		rm.AddRelationship("method1", "calls", []string{"method2", "method3"})
		rm.AddRelationship("method1", "uses", []string{"variable1"})

		// Check if relationships are stored correctly
		if relations := rm.Relationships["method1"]["calls"]; len(relations) != 2 {
			t.Errorf("Expected 2 'calls' relationships for method1, got %d", len(relations))
		}

		if relations := rm.Relationships["method1"]["uses"]; len(relations) != 1 {
			t.Errorf("Expected 1 'uses' relationship for method1, got %d", len(relations))
		}

		// Verify direct relationships are created for all attributes
		if !rm.HasRelationship("method1", "method2") {
			t.Error("Expected direct relationship between method1 and method2")
		}

		if !rm.HasRelationship("method1", "variable1") {
			t.Error("Expected direct relationship between method1 and variable1")
		}
	})
}
