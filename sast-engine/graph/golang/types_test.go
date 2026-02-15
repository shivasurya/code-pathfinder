package golang

import (
	"testing"
)

func TestParseTypeDeclarationStruct(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedName   string
		expectedKind   string
		expectedVis    string
		expectedFields []string
	}{
		{
			name:           "Struct with fields",
			code:           "package p\ntype Server struct {\n\tHost string\n\tPort int\n}",
			expectedName:   "Server",
			expectedKind:   "struct",
			expectedVis:    "public",
			expectedFields: []string{"Host: string", "Port: int"},
		},
		{
			name:           "Struct with embedded type",
			code:           "package p\ntype Admin struct {\n\tUser\n\tName string\n}",
			expectedName:   "Admin",
			expectedKind:   "struct",
			expectedVis:    "public",
			expectedFields: []string{"User", "Name: string"},
		},
		{
			name:           "Empty struct",
			code:           "package p\ntype Empty struct{}",
			expectedName:   "Empty",
			expectedKind:   "struct",
			expectedVis:    "public",
			expectedFields: nil,
		},
		{
			name:           "Unexported struct",
			code:           "package p\ntype server struct {\n\taddr string\n}",
			expectedName:   "server",
			expectedKind:   "struct",
			expectedVis:    "private",
			expectedFields: []string{"addr: string"},
		},
		{
			name:           "Struct with tags",
			code:           "package p\ntype S struct {\n\tName string `json:\"name\"`\n}",
			expectedName:   "S",
			expectedKind:   "struct",
			expectedFields: []string{"Name: string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			typeDecl := findNode(root, "type_declaration")
			if typeDecl == nil {
				t.Fatal("type_declaration not found")
			}

			types := ParseTypeDeclaration(typeDecl, []byte(tt.code))
			if len(types) != 1 {
				t.Fatalf("Expected 1 type, got %d", len(types))
			}
			info := types[0]

			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if info.Kind != tt.expectedKind {
				t.Errorf("Kind: expected %q, got %q", tt.expectedKind, info.Kind)
			}
			if tt.expectedVis != "" && info.Visibility != tt.expectedVis {
				t.Errorf("Visibility: expected %q, got %q", tt.expectedVis, info.Visibility)
			}
			if info.LineNumber == 0 {
				t.Error("LineNumber should be > 0")
			}
			if info.StartByte == 0 && info.EndByte == 0 {
				t.Error("StartByte/EndByte should be set")
			}

			if tt.expectedFields != nil {
				if len(info.Fields) != len(tt.expectedFields) {
					t.Fatalf("Fields: expected %d, got %d: %v", len(tt.expectedFields), len(info.Fields), info.Fields)
				}
				for i, f := range tt.expectedFields {
					if info.Fields[i] != f {
						t.Errorf("Fields[%d]: expected %q, got %q", i, f, info.Fields[i])
					}
				}
			} else if len(info.Fields) != 0 {
				t.Errorf("Expected empty fields, got %v", info.Fields)
			}
		})
	}
}

func TestParseTypeDeclarationInterface(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectedName    string
		expectedVis     string
		minMethodCount  int
		expectedMethods []string
	}{
		{
			name:           "Interface with methods",
			code:           "package p\ntype Handler interface {\n\tHandle() error\n\tClose()\n}",
			expectedName:   "Handler",
			expectedVis:    "public",
			minMethodCount: 2,
		},
		{
			name:           "Interface with embedded",
			code:           "package p\ntype RWC interface {\n\tio.Reader\n\tio.Writer\n\tClose() error\n}",
			expectedName:   "RWC",
			expectedVis:    "public",
			minMethodCount: 3,
		},
		{
			name:           "Unexported interface",
			code:           "package p\ntype handler interface {\n\tHandle()\n}",
			expectedName:   "handler",
			expectedVis:    "private",
			minMethodCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			typeDecl := findNode(root, "type_declaration")
			if typeDecl == nil {
				t.Fatal("type_declaration not found")
			}

			types := ParseTypeDeclaration(typeDecl, []byte(tt.code))
			if len(types) != 1 {
				t.Fatalf("Expected 1 type, got %d", len(types))
			}
			info := types[0]

			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if info.Kind != "interface" {
				t.Errorf("Kind: expected %q, got %q", "interface", info.Kind)
			}
			if info.Visibility != tt.expectedVis {
				t.Errorf("Visibility: expected %q, got %q", tt.expectedVis, info.Visibility)
			}
			if len(info.Methods) < tt.minMethodCount {
				t.Errorf("Methods: expected at least %d, got %d: %v", tt.minMethodCount, len(info.Methods), info.Methods)
			}
		})
	}
}

func TestParseTypeDeclarationAlias(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedName string
		expectedVis  string
	}{
		{
			name:         "Named type",
			code:         `package p; type UserID int64`,
			expectedName: "UserID",
			expectedVis:  "public",
		},
		{
			name:         "Function type",
			code:         `package p; type Handler func(string) error`,
			expectedName: "Handler",
			expectedVis:  "public",
		},
		{
			name:         "Slice type",
			code:         `package p; type StringSlice []string`,
			expectedName: "StringSlice",
			expectedVis:  "public",
		},
		{
			name:         "True alias with equals",
			code:         `package p; type MyInt = int`,
			expectedName: "MyInt",
			expectedVis:  "public",
		},
		{
			name:         "Unexported alias",
			code:         `package p; type userID int64`,
			expectedName: "userID",
			expectedVis:  "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			typeDecl := findNode(root, "type_declaration")
			if typeDecl == nil {
				t.Fatal("type_declaration not found")
			}

			types := ParseTypeDeclaration(typeDecl, []byte(tt.code))
			if len(types) != 1 {
				t.Fatalf("Expected 1 type, got %d", len(types))
			}
			info := types[0]

			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if info.Kind != "alias" {
				t.Errorf("Kind: expected %q, got %q", "alias", info.Kind)
			}
			if info.Visibility != tt.expectedVis {
				t.Errorf("Visibility: expected %q, got %q", tt.expectedVis, info.Visibility)
			}
		})
	}
}

func TestParseTypeDeclarationGrouped(t *testing.T) {
	code := `package p
type (
	A int
	B string
	C []byte
)`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	typeDecl := findNode(root, "type_declaration")
	if typeDecl == nil {
		t.Fatal("type_declaration not found")
	}

	types := ParseTypeDeclaration(typeDecl, []byte(code))
	if len(types) != 3 {
		t.Fatalf("Expected 3 types from grouped declaration, got %d", len(types))
	}

	expectedNames := []string{"A", "B", "C"}
	for i, name := range expectedNames {
		if types[i].Name != name {
			t.Errorf("types[%d].Name: expected %q, got %q", i, name, types[i].Name)
		}
		if types[i].Kind != "alias" {
			t.Errorf("types[%d].Kind: expected %q, got %q", i, "alias", types[i].Kind)
		}
	}
}

func TestParseTypeDeclarationGeneric(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedName   string
		expectedKind   string
		expectedVis    string
		minFieldCount  int
		minMethodCount int
	}{
		{
			name:          "Generic struct with single type param",
			code:          "package p\ntype Set[T comparable] struct {\n\titems []T\n}",
			expectedName:  "Set",
			expectedKind:  "struct",
			expectedVis:   "public",
			minFieldCount: 1,
		},
		{
			name:           "Generic interface with single type param",
			code:           "package p\ntype Container[T any] interface {\n\tGet() T\n}",
			expectedName:   "Container",
			expectedKind:   "interface",
			expectedVis:    "public",
			minMethodCount: 1,
		},
		{
			name:          "Generic struct with multiple type params",
			code:          "package p\ntype Pair[K comparable, V any] struct {\n\tKey K\n\tValue V\n}",
			expectedName:  "Pair",
			expectedKind:  "struct",
			expectedVis:   "public",
			minFieldCount: 2,
		},
		{
			name:         "Generic named type (slice)",
			code:         `package p; type Vector[T any] []T`,
			expectedName: "Vector",
			expectedKind: "alias",
			expectedVis:  "public",
		},
		{
			name:         "Unexported generic struct",
			code:         "package p\ntype stack[T any] struct {\n\titems []T\n}",
			expectedName: "stack",
			expectedKind: "struct",
			expectedVis:  "private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			typeDecl := findNode(root, "type_declaration")
			if typeDecl == nil {
				t.Fatal("type_declaration not found")
			}

			types := ParseTypeDeclaration(typeDecl, []byte(tt.code))
			if len(types) != 1 {
				t.Fatalf("Expected 1 type, got %d", len(types))
			}
			info := types[0]

			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if info.Kind != tt.expectedKind {
				t.Errorf("Kind: expected %q, got %q", tt.expectedKind, info.Kind)
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
			if tt.minFieldCount > 0 && len(info.Fields) < tt.minFieldCount {
				t.Errorf("Fields: expected at least %d, got %d: %v", tt.minFieldCount, len(info.Fields), info.Fields)
			}
			if tt.minMethodCount > 0 && len(info.Methods) < tt.minMethodCount {
				t.Errorf("Methods: expected at least %d, got %d: %v", tt.minMethodCount, len(info.Methods), info.Methods)
			}
		})
	}
}

func TestParseTypeDeclarationGroupedMixed(t *testing.T) {
	code := `package p
type (
	Server struct {
		Host string
	}
	Handler interface {
		Handle()
	}
	ID int64
)`
	tree, root := parseGoSnippet(t, code)
	defer tree.Close()

	typeDecl := findNode(root, "type_declaration")
	if typeDecl == nil {
		t.Fatal("type_declaration not found")
	}

	types := ParseTypeDeclaration(typeDecl, []byte(code))
	if len(types) != 3 {
		t.Fatalf("Expected 3 types, got %d", len(types))
	}

	if types[0].Kind != "struct" || types[0].Name != "Server" {
		t.Errorf("types[0]: expected struct Server, got %s %s", types[0].Kind, types[0].Name)
	}
	if len(types[0].Fields) != 1 || types[0].Fields[0] != "Host: string" {
		t.Errorf("types[0].Fields: expected [\"Host: string\"], got %v", types[0].Fields)
	}

	if types[1].Kind != "interface" || types[1].Name != "Handler" {
		t.Errorf("types[1]: expected interface Handler, got %s %s", types[1].Kind, types[1].Name)
	}
	if len(types[1].Methods) < 1 {
		t.Errorf("types[1].Methods: expected at least 1, got %d", len(types[1].Methods))
	}

	if types[2].Kind != "alias" || types[2].Name != "ID" {
		t.Errorf("types[2]: expected alias ID, got %s %s", types[2].Kind, types[2].Name)
	}
}
