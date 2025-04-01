package evaluator

import "testing"

func TestProcessAST(t *testing.T) {
	data := map[string][]map[string]interface{}{
		"m": {
			{"id": 1, "name": "foo", "class_id": 10, "modifiers": "public"},
			{"id": 2, "name": "bar", "class_id": 20, "modifiers": "private"},
		},
		"c": {
			{"id": 10, "name": "ClassA", "package": "public"},
			{"id": 20, "name": "ClassB", "package": "private"},
		},
	}

	tests := []struct {
		condition string
		wantLen   int
		wantErr   bool
	}{
		{"m.name = 'foo'", 1, false},
		{"m.getDeclaringType() = c", 2, false},
		{"m.modifiers = c.package", 2, false},
		{"m.name = 'foo' && m.getDeclaringType() = c", 1, false},
		{"m.getDeclaringType() = c || m.modifiers = c.package", 2, false},
		{"not (m.name = 'foo')", 3, false},
		{"", 0, true},
		{"m.name = 'foo' && (", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			node, err := ParseCondition(tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCondition() -- error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			result, err := ProcessAST(node, data)
			if err != nil {
				t.Errorf("ProcessAST() error = %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("ProcessAST() len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
