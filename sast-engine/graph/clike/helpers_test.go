package clike

import (
	"testing"
)

// TestExtractParameters covers the parameter shapes that statement
// extraction (PR-05) needs to consume cleanly: typed parameters, pointer
// and reference parameters, variadics, and unnamed (abstract) parameters.
func TestExtractParameters(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		code      string
		wantNames []string
		wantTypes []string
	}{
		{
			name:      "two named C parameters",
			language:  "c",
			code:      "void f(int a, int b);",
			wantNames: []string{"a", "b"},
			wantTypes: []string{"int", "int"},
		},
		{
			name:      "C variadic",
			language:  "c",
			code:      "int printf(const char* fmt, ...);",
			wantNames: []string{"fmt", "..."},
			wantTypes: []string{"const char*", "..."},
		},
		{
			name:      "C unnamed parameter",
			language:  "c",
			code:      "void f(int);",
			wantNames: []string{""},
			wantTypes: []string{"int"},
		},
		{
			name:      "C void parameter list",
			language:  "c",
			code:      "void f(void);",
			wantNames: []string{""},
			wantTypes: []string{"void"},
		},
		{
			name:      "C++ const reference",
			language:  "cpp",
			code:      "void f(const std::string& s, int n);",
			wantNames: []string{"s", "n"},
			wantTypes: []string{"const std::string&", "int"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, rt := snippet(t, tt.language, tt.code)
			defer tr.Close()

			pl := findNode(rt, "parameter_list")
			if pl == nil {
				t.Fatal("parameter_list not found")
			}

			names, types := ExtractParameters(pl, []byte(tt.code))
			if !equalStringSlices(names, tt.wantNames) {
				t.Errorf("names = %v, want %v", names, tt.wantNames)
			}
			if !equalStringSlices(types, tt.wantTypes) {
				t.Errorf("types = %v, want %v", types, tt.wantTypes)
			}
		})
	}
}

// TestExtractParameters_Nil verifies the nil-safe behaviour required by
// callers who pass through unchecked AST traversals.
func TestExtractParameters_Nil(t *testing.T) {
	names, types := ExtractParameters(nil, nil)
	if len(names) != 0 || len(types) != 0 {
		t.Errorf("ExtractParameters(nil) = (%v, %v), want empty", names, types)
	}
}

// TestExtractParameters_AbstractPointer covers the abstract_pointer_declarator
// path: forward declarations and function-pointer typedefs commonly omit
// parameter names (`void f(int*)`). The * suffix must still appear in the
// type and the name must come back empty.
func TestExtractParameters_AbstractPointer(t *testing.T) {
	code := "void f(int*);"
	tr, rt := parseCSnippet(t, code)
	defer tr.Close()

	pl := findNode(rt, "parameter_list")
	if pl == nil {
		t.Fatal("parameter_list not found")
	}
	names, types := ExtractParameters(pl, []byte(code))
	if !equalStringSlices(names, []string{""}) {
		t.Errorf("names = %v, want [\"\"]", names)
	}
	if !equalStringSlices(types, []string{"int*"}) {
		t.Errorf("types = %v, want [int*]", types)
	}
}

// TestExtractCallInfo covers the four call shapes emitted by tree-sitter
// for C and C++: free function, method (.), arrow method (->), and
// qualified (namespace) calls. The receiver and method-flag combinations
// drive the call-resolution logic in PR-03/PR-04.
func TestExtractCallInfo(t *testing.T) {
	tests := []struct {
		name     string
		language string
		code     string
		want     CallInfo
	}{
		{
			name:     "simple free function call",
			language: "c",
			code:     "void f() { malloc(128); }",
			want: CallInfo{
				Target: "malloc",
				Args:   []string{"128"},
			},
		},
		{
			name:     "free function with two args",
			language: "c",
			code:     "void f() { strcpy(dst, src); }",
			want: CallInfo{
				Target: "strcpy",
				Args:   []string{"dst", "src"},
			},
		},
		{
			name:     "method call via dot",
			language: "cpp",
			code:     "void f(Buffer b) { b.free(); }",
			want: CallInfo{
				Target:   "free",
				Args:     []string{},
				IsMethod: true,
				Receiver: "b",
			},
		},
		{
			name:     "method call via arrow",
			language: "cpp",
			code:     "void f(Buffer* b) { b->free(); }",
			want: CallInfo{
				Target:   "free",
				Args:     []string{},
				IsMethod: true,
				IsArrow:  true,
				Receiver: "b",
			},
		},
		{
			name:     "qualified namespace call",
			language: "cpp",
			code:     "void f() { std::move(x); }",
			want: CallInfo{
				Target:      "std::move",
				Args:        []string{"x"},
				IsQualified: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, rt := snippet(t, tt.language, tt.code)
			defer tr.Close()

			call := findNode(rt, "call_expression")
			if call == nil {
				t.Fatal("call_expression not found")
			}

			got := ExtractCallInfo(call, []byte(tt.code))
			if got == nil {
				t.Fatal("ExtractCallInfo returned nil")
			}
			if got.Target != tt.want.Target {
				t.Errorf("Target = %q, want %q", got.Target, tt.want.Target)
			}
			if got.IsMethod != tt.want.IsMethod {
				t.Errorf("IsMethod = %v, want %v", got.IsMethod, tt.want.IsMethod)
			}
			if got.IsArrow != tt.want.IsArrow {
				t.Errorf("IsArrow = %v, want %v", got.IsArrow, tt.want.IsArrow)
			}
			if got.IsQualified != tt.want.IsQualified {
				t.Errorf("IsQualified = %v, want %v", got.IsQualified, tt.want.IsQualified)
			}
			if got.Receiver != tt.want.Receiver {
				t.Errorf("Receiver = %q, want %q", got.Receiver, tt.want.Receiver)
			}
			if !equalStringSlices(got.Args, tt.want.Args) {
				t.Errorf("Args = %v, want %v", got.Args, tt.want.Args)
			}
		})
	}
}

// TestExtractCallInfo_FunctionPointerCall covers the populateCallTarget
// default branch — calls through a function pointer or any other
// non-classifiable function expression should preserve the raw source so
// downstream code can still match on it.
func TestExtractCallInfo_FunctionPointerCall(t *testing.T) {
	code := "void f() { (*fp)(x); }"
	tr, rt := parseCSnippet(t, code)
	defer tr.Close()

	call := findNode(rt, "call_expression")
	if call == nil {
		t.Fatal("call_expression not found")
	}
	got := ExtractCallInfo(call, []byte(code))
	if got == nil {
		t.Fatal("ExtractCallInfo returned nil")
	}
	if got.Target != "(*fp)" {
		t.Errorf("Target = %q, want %q", got.Target, "(*fp)")
	}
	if got.IsMethod || got.IsQualified || got.IsArrow {
		t.Errorf("expected unclassified call, got method=%v qualified=%v arrow=%v",
			got.IsMethod, got.IsQualified, got.IsArrow)
	}
	if !equalStringSlices(got.Args, []string{"x"}) {
		t.Errorf("Args = %v, want [x]", got.Args)
	}
}

// TestExtractCallInfo_NilOrWrongNode verifies the guards that protect
// callers from passing arbitrary nodes through this helper.
func TestExtractCallInfo_NilOrWrongNode(t *testing.T) {
	if got := ExtractCallInfo(nil, nil); got != nil {
		t.Errorf("ExtractCallInfo(nil) = %+v, want nil", got)
	}

	tr, rt := parseCSnippet(t, "int x;")
	defer tr.Close()
	if got := ExtractCallInfo(rt, nil); got != nil {
		t.Errorf("ExtractCallInfo(non-call) = %+v, want nil", got)
	}
}

// TestIsCKeyword verifies the C reserved-word set covers the spans the
// grammar emits as identifiers in tree-sitter so statement extraction in
// PR-05 can filter them safely.
func TestIsCKeyword(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// C89/C90 core
		{"int", true}, {"void", true}, {"return", true}, {"struct", true},
		{"const", true}, {"static", true}, {"sizeof", true},
		// C99
		{"inline", true}, {"restrict", true},
		// C11
		{"_Atomic", true}, {"_Generic", true},
		// C23
		{"true", true}, {"false", true}, {"nullptr", true},
		// Common constants
		{"NULL", true}, {"EOF", true},
		// Non-keywords
		{"foo", false}, {"bar", false}, {"my_func", false},
		// C++-only — must NOT be a C keyword
		{"class", false}, {"namespace", false}, {"new", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCKeyword(tt.name); got != tt.want {
				t.Errorf("IsCKeyword(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestIsCppKeyword verifies that C++ keyword recognition includes both the
// inherited C keywords and the C++-only additions.
func TestIsCppKeyword(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// Inherited from C
		{"int", true}, {"const", true}, {"static", true}, {"return", true},
		// C++-only additions
		{"class", true}, {"namespace", true}, {"template", true},
		{"new", true}, {"delete", true}, {"this", true},
		{"throw", true}, {"try", true}, {"catch", true},
		{"public", true}, {"private", true}, {"protected", true},
		// Common stdlib types treated as keywords for filtering
		{"string", true}, {"vector", true}, {"size_t", true},
		// Non-keywords
		{"foo", false}, {"my_class", false}, {"compute", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCppKeyword(tt.name); got != tt.want {
				t.Errorf("IsCppKeyword(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
