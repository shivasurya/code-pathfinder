package clikeextract

import (
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cFixtureDir = "testdata/c"

func cTestSource() HeaderSource {
	return HeaderSource{
		Platform:   core.PlatformLinux,
		Language:   core.LanguageC,
		SearchDirs: []string{cFixtureDir},
		HeaderExts: []string{".h"},
		SystemTag:  "glibc-test",
	}
}

func TestExtractCHeader_Stdio(t *testing.T) {
	src := cTestSource()
	hf := HeaderFile{Name: "stdio.h", Path: filepath.Join(cFixtureDir, "stdio.h")}

	h, err := extractCHeader(hf, src)
	require.NoError(t, err)
	require.NotNil(t, h)

	// Header metadata
	assert.Equal(t, "stdio.h", h.Header)
	assert.Equal(t, "c::stdio", h.ModuleID)
	assert.Equal(t, core.LanguageC, h.Language)
	assert.Equal(t, core.PlatformLinux, h.Platform)
	assert.Equal(t, "glibc-test", h.SystemTag)
	assert.Equal(t, SchemaVersion, h.SchemaVersion)

	// Spot-check a function with variadic params.
	printf := h.Functions["printf"]
	require.NotNil(t, printf, "printf must extract")
	assert.Equal(t, "c::stdio::printf", printf.FQN)
	assert.Equal(t, "int", printf.ReturnType)
	assert.Equal(t, core.SourceHeader, printf.Source)
	require.Len(t, printf.Params, 2)
	assert.Equal(t, "format", printf.Params[0].Name)
	assert.Equal(t, "const char*", printf.Params[0].Type)
	assert.Equal(t, "...", printf.Params[1].Name)
	assert.Equal(t, "variadic", printf.Params[1].Type)
	assert.False(t, printf.Params[1].Required)

	// Function returning a pointer.
	fopen := h.Functions["fopen"]
	require.NotNil(t, fopen)
	assert.Equal(t, "FILE*", fopen.ReturnType)

	// Functions with multiple plain params.
	fread := h.Functions["fread"]
	require.NotNil(t, fread)
	require.Len(t, fread.Params, 4)
	assert.Equal(t, "void*", fread.Params[0].Type)
	assert.Equal(t, "FILE*", fread.Params[3].Type)

	// Private symbols (leading _ / __) must be skipped.
	assert.Nil(t, h.Functions["_internal_buffer_flush"])
	assert.Nil(t, h.Functions["__builtin_printf_check"])

	// Typedefs.
	require.Contains(t, h.Typedefs, "FILE")
	assert.Equal(t, "struct __FILE", h.Typedefs["FILE"].Type)
	require.Contains(t, h.Typedefs, "fpos_t")

	// Constants.
	eof := h.Constants["EOF"]
	require.NotNil(t, eof)
	assert.Equal(t, "int", eof.Type)
	assert.Equal(t, "(-1)", eof.Value)
	assert.Equal(t, core.SourceHeader, eof.Source)

	bufsiz := h.Constants["BUFSIZ"]
	require.NotNil(t, bufsiz)
	assert.Equal(t, "int", bufsiz.Type)
	assert.Equal(t, "8192", bufsiz.Value)

	defaultPath := h.Constants["DEFAULT_PATH"]
	require.NotNil(t, defaultPath)
	assert.Equal(t, "const char*", defaultPath.Type)
}

func TestExtractCHeader_String(t *testing.T) {
	src := cTestSource()
	hf := HeaderFile{Name: "string.h", Path: filepath.Join(cFixtureDir, "string.h")}

	h, err := extractCHeader(hf, src)
	require.NoError(t, err)

	for _, name := range []string{"memcpy", "strcpy", "strlen", "strchr", "memcmp"} {
		assert.NotNil(t, h.Functions[name], "expected function %q", name)
	}
	require.Contains(t, h.Typedefs, "size_t")
	// `((void*)0)` is not a literal — NULL constant gets dropped because
	// inferConstantType returns "" for it. This is the conservative-cut
	// behaviour we documented in the package: we'd rather be silent than
	// emit untyped constants.
	assert.Nil(t, h.Constants["NULL"])
}

func TestExtractCHeader_Unistd_PointerArrayParam(t *testing.T) {
	src := cTestSource()
	hf := HeaderFile{Name: "unistd.h", Path: filepath.Join(cFixtureDir, "unistd.h")}

	h, err := extractCHeader(hf, src)
	require.NoError(t, err)

	// Function with `void` parameter list (fork(void)).
	fork := h.Functions["fork"]
	require.NotNil(t, fork)
	assert.Equal(t, "pid_t", fork.ReturnType)
	// `void` is a single explicit-empty param in tree-sitter; we accept the
	// flat representation and don't try to special-case it.
	assert.LessOrEqual(t, len(fork.Params), 1)

	// Function whose parameter is `char* const argv[]` — type tracking should
	// at least capture the array part. Exact form is grammar-dependent so
	// we just check it's non-empty.
	exec := h.Functions["execvp"]
	require.NotNil(t, exec)
	assert.GreaterOrEqual(t, len(exec.Params), 2)
}

func TestExtractCHeader_Inline(t *testing.T) {
	src := cTestSource()
	hf := HeaderFile{Name: "inline.h", Path: filepath.Join(cFixtureDir, "inline.h")}

	h, err := extractCHeader(hf, src)
	require.NoError(t, err)

	// Inline function (function_definition) — exercises extractCFunction.
	abs := h.Functions["abs_diff"]
	require.NotNil(t, abs, "extractCFunction must capture inline definitions")
	assert.Equal(t, "int", abs.ReturnType)

	// preproc_ifdef + preproc_else branches both walked.
	assert.NotNil(t, h.Functions["foo_a"])
	assert.NotNil(t, h.Functions["foo_b"])

	// Pointer typedef.
	require.Contains(t, h.Typedefs, "int_ptr")
	assert.Equal(t, "int*", h.Typedefs["int_ptr"].Type)
}

func TestExtractCHeader_FileNotFound(t *testing.T) {
	src := cTestSource()
	_, err := extractCHeader(HeaderFile{Name: "absent.h", Path: "/nope/absent.h"}, src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestStripTrailingComment(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"42", "42"},
		{"42 // line comment", "42"},
		{"42 /* block */", "42"},
		{"42 // c1\n", "42"},
		{"42  ", "42"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, stripTrailingComment(tt.in))
		})
	}
}

func TestInferConstantType(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"42", "int"},
		{"-1", "int"},
		{"(-1)", "int"},
		{"((42))", "int"},
		{"0x1F", "int"},
		{"077", "int"},
		{`"hello"`, "const char*"},
		{`'a'`, "char"},
		{`((void*)0)`, ""},
		{`x + 1`, ""},
		{"FOO", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, inferConstantType(tt.in))
		})
	}
}

func TestIsIntegerLiteral(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"42", true},
		{"-1", true},
		{"+5", true},
		{"-", false},
		{"0x1F", true},
		{"0X1f", true},
		{"0x", false},
		{"0xZ", false},
		{"abc", false},
		{"1a", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isIntegerLiteral(tt.in))
		})
	}
}

func TestStripBalancedOuterParens(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"42", "42"},
		{"(42)", "42"},
		{"((42))", "42"},
		{"(-1)", "-1"},
		{"((void*)0)", "(void*)0"},
		{"(a)+(b)", "(a)+(b)"},
		{"  (5)  ", "5"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, stripBalancedOuterParens(tt.in))
		})
	}
}

func TestUnwrapToFunctionDeclaratorPublic_NilInput(t *testing.T) {
	assert.Nil(t, unwrapToFunctionDeclaratorPublic(nil))
}

func TestUnwrapDeclaratorIdentifier_NilInput(t *testing.T) {
	assert.Equal(t, "", unwrapDeclaratorIdentifier(nil, nil))
}

// extractCHeader invariant: returns a fully-allocated header with
// pre-populated maps, even if no symbols are extracted from a body that
// only has comments.
func TestExtractCHeader_EmptyHeader(t *testing.T) {
	dir := t.TempDir()
	emptyPath := filepath.Join(dir, "empty.h")
	mustWriteFile(t, emptyPath, "/* empty */\n")

	src := HeaderSource{
		Platform: core.PlatformLinux, Language: core.LanguageC,
		SystemTag: "glibc-test", SearchDirs: []string{dir}, HeaderExts: []string{".h"},
	}
	h, err := extractCHeader(HeaderFile{Name: "empty.h", Path: emptyPath}, src)
	require.NoError(t, err)
	require.NotNil(t, h)
	assert.Equal(t, "empty.h", h.Header)
	assert.Empty(t, h.Functions)
	assert.Empty(t, h.Typedefs)
	assert.Empty(t, h.Constants)
}
