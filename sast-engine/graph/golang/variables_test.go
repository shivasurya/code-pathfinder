package golang

import (
	"testing"
)

func TestParseVarDeclaration(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedCount  int
		expectedName   string
		expectedValue  string
		expectedType   string
		expectedVis    string
	}{
		{
			name:          "var with type only",
			code:          "package p\nvar name string",
			expectedCount: 1,
			expectedName:  "name",
			expectedValue: "",
			expectedType:  "string",
			expectedVis:   "private",
		},
		{
			name:          "var with type and value",
			code:          "package p\nvar Name string = \"Alice\"",
			expectedCount: 1,
			expectedName:  "Name",
			expectedValue: "\"Alice\"",
			expectedType:  "string",
			expectedVis:   "public",
		},
		{
			name:          "var with inferred type",
			code:          "package p\nvar x = 1",
			expectedCount: 1,
			expectedName:  "x",
			expectedValue: "1",
			expectedType:  "",
			expectedVis:   "private",
		},
		{
			name:          "var grouped",
			code:          "package p\nvar (\n\tx int\n\ty string\n)",
			expectedCount: 2,
			expectedName:  "x",
			expectedVis:   "private",
		},
		{
			name:          "var multi-name",
			code:          "package p\nvar x, y int",
			expectedCount: 2,
			expectedName:  "x",
			expectedType:  "int",
			expectedVis:   "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			varDecl := findNode(root, "var_declaration")
			if varDecl == nil {
				t.Fatal("var_declaration not found")
			}

			vars := ParseVarDeclaration(varDecl, []byte(tt.code))
			if len(vars) != tt.expectedCount {
				t.Fatalf("Expected %d vars, got %d", tt.expectedCount, len(vars))
			}

			// Check first var
			info := vars[0]
			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if tt.expectedValue != "" && info.Value != tt.expectedValue {
				t.Errorf("Value: expected %q, got %q", tt.expectedValue, info.Value)
			}
			if tt.expectedType != "" && info.TypeName != tt.expectedType {
				t.Errorf("TypeName: expected %q, got %q", tt.expectedType, info.TypeName)
			}
			if info.Visibility != tt.expectedVis {
				t.Errorf("Visibility: expected %q, got %q", tt.expectedVis, info.Visibility)
			}
			if info.LineNumber == 0 {
				t.Error("LineNumber should be > 0")
			}
			if info.StartByte == 0 && info.EndByte == 0 {
				t.Error("StartByte/EndByte should be set")
			}
			if info.IsMulti {
				t.Error("var declarations should have IsMulti = false")
			}
		})
	}
}

func TestParseShortVarDeclaration(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedCount int
		expectedNames []string
		expectedValue string
		expectedMulti bool
	}{
		{
			name:          "single short var",
			code:          `name := "Alice"`,
			expectedCount: 1,
			expectedNames: []string{"name"},
			expectedValue: `"Alice"`,
			expectedMulti: false,
		},
		{
			name:          "multi short var",
			code:          "x, y := foo()",
			expectedCount: 2,
			expectedNames: []string{"x", "y"},
			expectedValue: "foo()",
			expectedMulti: true,
		},
		{
			name:          "short var with blank",
			code:          "_, err := foo()",
			expectedCount: 1,
			expectedNames: []string{"err"},
			expectedValue: "foo()",
			expectedMulti: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, "package p\nfunc f() {\n"+tt.code+"\n}")
			defer tree.Close()

			funcDecl := findNode(root, "function_declaration")
			if funcDecl == nil {
				t.Fatal("function_declaration not found")
			}
			body := funcDecl.ChildByFieldName("body")
			if body == nil {
				t.Fatal("body not found")
			}
			shortVarDecl := findNode(body, "short_var_declaration")
			if shortVarDecl == nil {
				t.Fatal("short_var_declaration not found")
			}

			vars := ParseShortVarDeclaration(shortVarDecl, []byte("package p\nfunc f() {\n"+tt.code+"\n}"))
			if len(vars) != tt.expectedCount {
				t.Fatalf("Expected %d vars, got %d", tt.expectedCount, len(vars))
			}

			for i, expectedName := range tt.expectedNames {
				if vars[i].Name != expectedName {
					t.Errorf("vars[%d].Name: expected %q, got %q", i, expectedName, vars[i].Name)
				}
				if vars[i].Value != tt.expectedValue {
					t.Errorf("vars[%d].Value: expected %q, got %q", i, tt.expectedValue, vars[i].Value)
				}
				if vars[i].IsMulti != tt.expectedMulti {
					t.Errorf("vars[%d].IsMulti: expected %v, got %v", i, tt.expectedMulti, vars[i].IsMulti)
				}
				if vars[i].LineNumber == 0 {
					t.Errorf("vars[%d].LineNumber should be > 0", i)
				}
			}
		})
	}
}

func TestParseConstDeclaration(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedCount int
		expectedNames []string
		expectedVis   string
	}{
		{
			name:          "single const",
			code:          "package p\nconst Pi = 3.14",
			expectedCount: 1,
			expectedNames: []string{"Pi"},
			expectedVis:   "public",
		},
		{
			name:          "const iota group",
			code:          "package p\nconst (\n\tA = iota\n\tB\n\tC\n)",
			expectedCount: 3,
			expectedNames: []string{"A", "B", "C"},
			expectedVis:   "public",
		},
		{
			name:          "unexported const",
			code:          "package p\nconst pi = 3.14",
			expectedCount: 1,
			expectedNames: []string{"pi"},
			expectedVis:   "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			constDecl := findNode(root, "const_declaration")
			if constDecl == nil {
				t.Fatal("const_declaration not found")
			}

			consts := ParseConstDeclaration(constDecl, []byte(tt.code))
			if len(consts) != tt.expectedCount {
				t.Fatalf("Expected %d consts, got %d", tt.expectedCount, len(consts))
			}

			for i, expectedName := range tt.expectedNames {
				if consts[i].Name != expectedName {
					t.Errorf("consts[%d].Name: expected %q, got %q", i, expectedName, consts[i].Name)
				}
				if consts[i].Visibility != tt.expectedVis {
					t.Errorf("consts[%d].Visibility: expected %q, got %q", i, tt.expectedVis, consts[i].Visibility)
				}
				if consts[i].LineNumber == 0 {
					t.Errorf("consts[%d].LineNumber should be > 0", i)
				}
			}

			// Check specific values for iota test
			if tt.name == "const iota group" {
				if consts[0].Value != "iota" {
					t.Errorf("A.Value: expected \"iota\", got %q", consts[0].Value)
				}
				if consts[1].Value != "" {
					t.Errorf("B.Value: expected empty (iota follower), got %q", consts[1].Value)
				}
				if consts[2].Value != "" {
					t.Errorf("C.Value: expected empty (iota follower), got %q", consts[2].Value)
				}
			}
		})
	}
}

func TestParseAssignment(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedCount int
		expectedNames []string
		expectedValue string
		expectedMulti bool
	}{
		{
			name:          "single assignment",
			code:          "x = x + 1",
			expectedCount: 1,
			expectedNames: []string{"x"},
			expectedValue: "x + 1",
			expectedMulti: false,
		},
		{
			name:          "multi assignment",
			code:          "x, y = 1, 2",
			expectedCount: 2,
			expectedNames: []string{"x", "y"},
			expectedValue: "1, 2",
			expectedMulti: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, "package p\nfunc f() {\n"+tt.code+"\n}")
			defer tree.Close()

			funcDecl := findNode(root, "function_declaration")
			if funcDecl == nil {
				t.Fatal("function_declaration not found")
			}
			body := funcDecl.ChildByFieldName("body")
			if body == nil {
				t.Fatal("body not found")
			}
			assignStmt := findNode(body, "assignment_statement")
			if assignStmt == nil {
				t.Fatal("assignment_statement not found")
			}

			vars := ParseAssignment(assignStmt, []byte("package p\nfunc f() {\n"+tt.code+"\n}"))
			if len(vars) != tt.expectedCount {
				t.Fatalf("Expected %d vars, got %d", tt.expectedCount, len(vars))
			}

			for i, expectedName := range tt.expectedNames {
				if vars[i].Name != expectedName {
					t.Errorf("vars[%d].Name: expected %q, got %q", i, expectedName, vars[i].Name)
				}
				if vars[i].Value != tt.expectedValue {
					t.Errorf("vars[%d].Value: expected %q, got %q", i, tt.expectedValue, vars[i].Value)
				}
				if vars[i].IsMulti != tt.expectedMulti {
					t.Errorf("vars[%d].IsMulti: expected %v, got %v", i, tt.expectedMulti, vars[i].IsMulti)
				}
			}
		})
	}
}
