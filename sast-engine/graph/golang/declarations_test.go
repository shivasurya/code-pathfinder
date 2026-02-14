package golang

import (
	"testing"
)

func TestParseFunctionDeclaration(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedName   string
		expectedVis    string
		expectedReturn string
		expectedNames  []string
		expectedTypes  []string
		expectedInit   bool
	}{
		{
			name:           "Exported function with params and return",
			code:           `package p; func Foo(a, b int) string { return "" }`,
			expectedName:   "Foo",
			expectedVis:    "public",
			expectedReturn: "string",
			expectedNames:  []string{"a", "b"},
			expectedTypes:  []string{"a: int", "b: int"},
			expectedInit:   false,
		},
		{
			name:         "Private function no params no return",
			code:         `package p; func foo() {}`,
			expectedName: "foo",
			expectedVis:  "private",
			expectedInit: false,
		},
		{
			name:         "init function",
			code:         `package p; func init() {}`,
			expectedName: "init",
			expectedVis:  "private",
			expectedInit: true,
		},
		{
			name:           "Variadic function",
			code:           `package p; func Printf(format string, args ...interface{}) {}`,
			expectedName:   "Printf",
			expectedVis:    "public",
			expectedNames:  []string{"format", "args"},
			expectedTypes:  []string{"format: string", "args: ...interface{}"},
			expectedInit:   false,
		},
		{
			name:           "Multiple return types",
			code:           `package p; func Read(p []byte) (int, error) { return 0, nil }`,
			expectedName:   "Read",
			expectedVis:    "public",
			expectedReturn: "(int, error)",
			expectedNames:  []string{"p"},
			expectedTypes:  []string{"p: []byte"},
			expectedInit:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			funcDecl := findNode(root, "function_declaration")
			if funcDecl == nil {
				t.Fatal("function_declaration not found")
			}

			info := ParseFunctionDeclaration(funcDecl, []byte(tt.code))

			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if info.Visibility != tt.expectedVis {
				t.Errorf("Visibility: expected %q, got %q", tt.expectedVis, info.Visibility)
			}
			if info.ReturnType != tt.expectedReturn {
				t.Errorf("ReturnType: expected %q, got %q", tt.expectedReturn, info.ReturnType)
			}
			if info.IsInit != tt.expectedInit {
				t.Errorf("IsInit: expected %v, got %v", tt.expectedInit, info.IsInit)
			}
			if info.LineNumber == 0 {
				t.Error("LineNumber should be > 0")
			}

			if tt.expectedNames != nil {
				if len(info.Params.Names) != len(tt.expectedNames) {
					t.Fatalf("Params.Names: expected %d, got %d: %v", len(tt.expectedNames), len(info.Params.Names), info.Params.Names)
				}
				for i, name := range tt.expectedNames {
					if info.Params.Names[i] != name {
						t.Errorf("Params.Names[%d]: expected %q, got %q", i, name, info.Params.Names[i])
					}
				}
			}

			if tt.expectedTypes != nil {
				if len(info.Params.Types) != len(tt.expectedTypes) {
					t.Fatalf("Params.Types: expected %d, got %d: %v", len(tt.expectedTypes), len(info.Params.Types), info.Params.Types)
				}
				for i, typ := range tt.expectedTypes {
					if info.Params.Types[i] != typ {
						t.Errorf("Params.Types[%d]: expected %q, got %q", i, typ, info.Params.Types[i])
					}
				}
			}
		})
	}
}

func TestParseMethodDeclaration(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		expectedName     string
		expectedVis      string
		expectedReturn   string
		expectedReceiver string
		expectedNames    []string
		expectedTypes    []string
	}{
		{
			name:             "Pointer receiver method",
			code:             `package p; type S struct{}; func (s *S) Start() error { return nil }`,
			expectedName:     "Start",
			expectedVis:      "public",
			expectedReturn:   "error",
			expectedReceiver: "S",
		},
		{
			name:             "Value receiver method",
			code:             `package p; type S struct{}; func (s S) String() string { return "" }`,
			expectedName:     "String",
			expectedVis:      "public",
			expectedReturn:   "string",
			expectedReceiver: "S",
		},
		{
			name:             "Method with params",
			code:             `package p; type S struct{}; func (s *S) Handle(w Writer, r *Request) {}`,
			expectedName:     "Handle",
			expectedVis:      "public",
			expectedReceiver: "S",
			expectedNames:    []string{"w", "r"},
			expectedTypes:    []string{"w: Writer", "r: *Request"},
		},
		{
			name:             "Private method no params",
			code:             `package p; type S struct{}; func (s *S) stop() {}`,
			expectedName:     "stop",
			expectedVis:      "private",
			expectedReceiver: "S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, root := parseGoSnippet(t, tt.code)
			defer tree.Close()

			methodDecl := findNode(root, "method_declaration")
			if methodDecl == nil {
				t.Fatal("method_declaration not found")
			}

			info := ParseMethodDeclaration(methodDecl, []byte(tt.code))

			if info.Name != tt.expectedName {
				t.Errorf("Name: expected %q, got %q", tt.expectedName, info.Name)
			}
			if info.Visibility != tt.expectedVis {
				t.Errorf("Visibility: expected %q, got %q", tt.expectedVis, info.Visibility)
			}
			if info.ReturnType != tt.expectedReturn {
				t.Errorf("ReturnType: expected %q, got %q", tt.expectedReturn, info.ReturnType)
			}
			if info.ReceiverType != tt.expectedReceiver {
				t.Errorf("ReceiverType: expected %q, got %q", tt.expectedReceiver, info.ReceiverType)
			}
			if info.IsInit {
				t.Error("Methods should never be init functions")
			}
			if info.LineNumber == 0 {
				t.Error("LineNumber should be > 0")
			}

			if tt.expectedNames != nil {
				if len(info.Params.Names) != len(tt.expectedNames) {
					t.Fatalf("Params.Names: expected %d, got %d: %v", len(tt.expectedNames), len(info.Params.Names), info.Params.Names)
				}
				for i, name := range tt.expectedNames {
					if info.Params.Names[i] != name {
						t.Errorf("Params.Names[%d]: expected %q, got %q", i, name, info.Params.Names[i])
					}
				}
			}

			if tt.expectedTypes != nil {
				if len(info.Params.Types) != len(tt.expectedTypes) {
					t.Fatalf("Params.Types: expected %d, got %d: %v", len(tt.expectedTypes), len(info.Params.Types), info.Params.Types)
				}
				for i, typ := range tt.expectedTypes {
					if info.Params.Types[i] != typ {
						t.Errorf("Params.Types[%d]: expected %q, got %q", i, typ, info.Params.Types[i])
					}
				}
			}
		})
	}
}
