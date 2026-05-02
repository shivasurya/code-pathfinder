package clike

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// TestExtractTypeString covers the type strings produced for parameter and
// field declarations across both C and C++. Each case parses a real source
// snippet, locates the first parameter_declaration or field_declaration, and
// invokes ExtractTypeString on its (type, declarator) pair.
func TestExtractTypeString(t *testing.T) {
	tests := []struct {
		name string
		// language: "c" or "cpp"
		language string
		code     string
		// node selects which AST node to feed into ExtractTypeString:
		// "param" for the first parameter_declaration, "field" for the
		// first field_declaration.
		nodeKind string
		want     string
	}{
		{
			name:     "plain int parameter",
			language: "c",
			code:     "void f(int x);",
			nodeKind: "param",
			want:     "int",
		},
		{
			name:     "char pointer parameter",
			language: "c",
			code:     "void f(char* buf);",
			nodeKind: "param",
			want:     "char*",
		},
		{
			name:     "double pointer parameter",
			language: "c",
			code:     "void f(int** pp);",
			nodeKind: "param",
			want:     "int**",
		},
		{
			name:     "const char pointer parameter",
			language: "c",
			code:     "void f(const char* fmt);",
			nodeKind: "param",
			want:     "const char*",
		},
		{
			name:     "FILE pointer parameter",
			language: "c",
			code:     "void f(FILE* fp);",
			nodeKind: "param",
			want:     "FILE*",
		},
		{
			name:     "unsigned long long parameter",
			language: "c",
			code:     "void f(unsigned long long n);",
			nodeKind: "param",
			want:     "unsigned long long",
		},
		{
			name:     "void pointer parameter",
			language: "c",
			code:     "void f(void* p);",
			nodeKind: "param",
			want:     "void*",
		},
		{
			name:     "struct field pointer",
			language: "c",
			code:     "struct S { char* name; };",
			nodeKind: "field",
			want:     "char*",
		},
		{
			name:     "C++ string reference",
			language: "cpp",
			code:     "void f(std::string& s);",
			nodeKind: "param",
			want:     "std::string&",
		},
		{
			name:     "C++ const string reference",
			language: "cpp",
			code:     "void f(const std::string& s);",
			nodeKind: "param",
			want:     "const std::string&",
		},
		{
			name:     "C++ vector of int",
			language: "cpp",
			code:     "void f(std::vector<int> v);",
			nodeKind: "param",
			want:     "std::vector<int>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tree *sitter.Tree
			var root *sitter.Node
			if tt.language == "cpp" {
				tree, root = parseCppSnippet(t, tt.code)
			} else {
				tree, root = parseCSnippet(t, tt.code)
			}
			defer tree.Close()

			var typeNode, declarator *sitter.Node
			switch tt.nodeKind {
			case "param":
				p := findNode(root, "parameter_declaration")
				if p == nil {
					t.Fatal("parameter_declaration not found")
				}
				typeNode = p.ChildByFieldName("type")
				declarator = p.ChildByFieldName("declarator")
			case "field":
				f := findNode(root, "field_declaration")
				if f == nil {
					t.Fatal("field_declaration not found")
				}
				typeNode = f.ChildByFieldName("type")
				declarator = f.ChildByFieldName("declarator")
			default:
				t.Fatalf("unknown nodeKind %q", tt.nodeKind)
			}

			got := ExtractTypeString(typeNode, declarator, []byte(tt.code))
			if got != tt.want {
				t.Errorf("ExtractTypeString = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractTypeString_NilType verifies the void fallback for a nil type
// node (used when tree-sitter omits the type slot for void returns).
func TestExtractTypeString_NilType(t *testing.T) {
	got := ExtractTypeString(nil, nil, nil)
	if got != "void" {
		t.Errorf("ExtractTypeString(nil, nil, nil) = %q, want %q", got, "void")
	}
}
