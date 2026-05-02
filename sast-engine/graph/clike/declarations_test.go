package clike

import (
	"testing"
)

// TestExtractFunctionInfo covers the full range of C and C++ function shapes:
// definitions vs forward declarations, void/typed returns, pointer returns,
// variadic functions, and member functions in C++ classes.
func TestExtractFunctionInfo(t *testing.T) {
	tests := []struct {
		name             string
		language         string // "c" or "cpp"
		code             string
		wantName         string
		wantReturn       string
		wantParamNames   []string
		wantParamTypes   []string
		wantIsDeclaration bool
	}{
		{
			name:             "C function with body",
			language:         "c",
			code:             "int add(int a, int b) { return a + b; }",
			wantName:         "add",
			wantReturn:       "int",
			wantParamNames:   []string{"a", "b"},
			wantParamTypes:   []string{"int", "int"},
			wantIsDeclaration: false,
		},
		{
			name:             "C void function",
			language:         "c",
			code:             "void log_msg(const char* fmt) { (void)fmt; }",
			wantName:         "log_msg",
			wantReturn:       "void",
			wantParamNames:   []string{"fmt"},
			wantParamTypes:   []string{"const char*"},
			wantIsDeclaration: false,
		},
		{
			name:             "C function returning pointer",
			language:         "c",
			code:             "char* allocate(size_t n) { return 0; }",
			wantName:         "allocate",
			wantReturn:       "char*",
			wantParamNames:   []string{"n"},
			wantParamTypes:   []string{"size_t"},
			wantIsDeclaration: false,
		},
		{
			name:             "C variadic function",
			language:         "c",
			code:             "int printf(const char* fmt, ...) { return 0; }",
			wantName:         "printf",
			wantReturn:       "int",
			wantParamNames:   []string{"fmt", "..."},
			wantParamTypes:   []string{"const char*", "..."},
			wantIsDeclaration: false,
		},
		{
			name:             "C++ method definition",
			language:         "cpp",
			code:             "int Socket::send(const std::string& msg) { return 0; }",
			wantName:         "Socket::send",
			wantReturn:       "int",
			wantParamNames:   []string{"msg"},
			wantParamTypes:   []string{"const std::string&"},
			wantIsDeclaration: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, rt := snippet(t, tt.language, tt.code)
			defer tr.Close()

			fnDef := findNode(rt, "function_definition")
			if fnDef == nil {
				t.Fatal("function_definition not found")
			}

			info := ExtractFunctionInfo(fnDef, []byte(tt.code))
			if info == nil {
				t.Fatal("ExtractFunctionInfo returned nil")
			}

			if info.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", info.Name, tt.wantName)
			}
			if info.ReturnType != tt.wantReturn {
				t.Errorf("ReturnType = %q, want %q", info.ReturnType, tt.wantReturn)
			}
			if !equalStringSlices(info.ParamNames, tt.wantParamNames) {
				t.Errorf("ParamNames = %v, want %v", info.ParamNames, tt.wantParamNames)
			}
			if !equalStringSlices(info.ParamTypes, tt.wantParamTypes) {
				t.Errorf("ParamTypes = %v, want %v", info.ParamTypes, tt.wantParamTypes)
			}
			if info.IsDeclaration != tt.wantIsDeclaration {
				t.Errorf("IsDeclaration = %v, want %v", info.IsDeclaration, tt.wantIsDeclaration)
			}
			if info.LineNumber == 0 {
				t.Error("LineNumber should be > 0")
			}
		})
	}
}

// TestExtractFunctionInfo_NilNode verifies the nil guard.
func TestExtractFunctionInfo_NilNode(t *testing.T) {
	if got := ExtractFunctionInfo(nil, nil); got != nil {
		t.Errorf("ExtractFunctionInfo(nil) = %+v, want nil", got)
	}
}

// TestExtractStructFields covers field extraction for C structs and C++
// classes, including pointer fields and primitive fields.
func TestExtractStructFields(t *testing.T) {
	tests := []struct {
		name     string
		language string
		code     string
		want     []FieldInfo
	}{
		{
			name:     "C struct with primitive and pointer fields",
			language: "c",
			code:     "struct Buffer { char* data; size_t len; int capacity; };",
			want: []FieldInfo{
				{Name: "data", TypeStr: "char*"},
				{Name: "len", TypeStr: "size_t"},
				{Name: "capacity", TypeStr: "int"},
			},
		},
		{
			name:     "Empty C struct",
			language: "c",
			code:     "struct Empty { };",
			want:     []FieldInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, rt := snippet(t, tt.language, tt.code)
			defer tr.Close()

			list := findNode(rt, "field_declaration_list")
			if list == nil {
				t.Fatal("field_declaration_list not found")
			}

			got := ExtractStructFields(list, []byte(tt.code))
			if len(got) != len(tt.want) {
				t.Fatalf("got %d fields, want %d (%v)", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("field[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestExtractStructFields_NilList verifies the nil guard.
func TestExtractStructFields_NilList(t *testing.T) {
	if got := ExtractStructFields(nil, nil); got != nil {
		t.Errorf("ExtractStructFields(nil) = %v, want nil", got)
	}
}

// TestExtractStructFields_BitfieldAndArray covers the array_declarator
// path inside fieldDeclaratorName and the bitfield case where the
// declarator is a plain field_identifier with a sibling bitfield_clause.
func TestExtractStructFields_BitfieldAndArray(t *testing.T) {
	code := "struct S { int x : 3; int arr[10]; };"
	tr, rt := parseCSnippet(t, code)
	defer tr.Close()

	list := findNode(rt, "field_declaration_list")
	if list == nil {
		t.Fatal("field_declaration_list not found")
	}

	got := ExtractStructFields(list, []byte(code))
	want := []FieldInfo{
		{Name: "x", TypeStr: "int"},
		{Name: "arr", TypeStr: "int"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestExtractStructFields_SkipNonFieldChildren covers the filter that
// rejects non-field_declaration children inside a class body — C++ class
// bodies routinely interleave access_specifier nodes with the actual
// fields, and those must not show up in the FieldInfo slice.
func TestExtractStructFields_SkipNonFieldChildren(t *testing.T) {
	code := "class C { public: int x; private: int y; };"
	tr, rt := parseCppSnippet(t, code)
	defer tr.Close()

	list := findNode(rt, "field_declaration_list")
	if list == nil {
		t.Fatal("field_declaration_list not found")
	}

	got := ExtractStructFields(list, []byte(code))
	want := []FieldInfo{
		{Name: "x", TypeStr: "int"},
		{Name: "y", TypeStr: "int"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestExtractFunctionInfo_NoFunctionDeclarator covers the defensive
// fallback that triggers when the declarator chain never reaches a
// function_declarator (malformed AST, or a bare declaration like
// "int x;" passed by mistake). The returned info is still non-nil so
// callers can record a partial entry.
func TestExtractFunctionInfo_NoFunctionDeclarator(t *testing.T) {
	code := "int x;"
	tr, rt := parseCSnippet(t, code)
	defer tr.Close()

	decl := findNode(rt, "declaration")
	if decl == nil {
		t.Fatal("declaration not found")
	}
	info := ExtractFunctionInfo(decl, []byte(code))
	if info == nil {
		t.Fatal("expected non-nil partial info")
	}
	if info.Name != "" {
		t.Errorf("Name = %q, want empty", info.Name)
	}
	if !info.IsDeclaration {
		t.Error("expected IsDeclaration=true for body-less node")
	}
}
