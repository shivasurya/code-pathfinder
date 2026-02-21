package goextract

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Unit tests for helper functions ---

func TestShouldSkipComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"internal", "internal", true},
		{"cmd", "cmd", true},
		{"testdata", "testdata", true},
		{"vendor", "vendor", true},
		{"builtin", "builtin", true},
		{"hidden dot", ".git", true},
		{"underscore", "_test", true},
		{"regular", "fmt", false},
		{"net", "net", false},
		{"http", "http", false},
		{"os", "os", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldSkipComponent(tt.input))
		})
	}
}

func TestPackageFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fmt", "fmt_stdlib.json"},
		{"net/http", "net_http_stdlib.json"},
		{"net/http/httputil", "net_http_httputil_stdlib.json"},
		{"encoding/json", "encoding_json_stdlib.json"},
		{"os", "os_stdlib.json"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, packageFileName(tt.input))
		})
	}
}

func TestChecksumBytes(t *testing.T) {
	data := []byte("hello world")
	checksum := checksumBytes(data)
	assert.True(t, strings.HasPrefix(checksum, "sha256:"), "checksum must start with sha256:")
	assert.Len(t, checksum, len("sha256:")+64, "sha256 hex is 64 chars")

	// Same data → same checksum.
	assert.Equal(t, checksum, checksumBytes(data))

	// Different data → different checksum.
	assert.NotEqual(t, checksum, checksumBytes([]byte("different")))
}

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		input string
		major int
		minor int
		patch int
		full  string
	}{
		{"1.21", 1, 21, 0, "1.21.0"},
		{"1.21.0", 1, 21, 0, "1.21.0"},
		{"1.26.0", 1, 26, 0, "1.26.0"},
		{"1.18", 1, 18, 0, "1.18.0"},
		{"1.22.3", 1, 22, 3, "1.22.3"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v := parseGoVersion(tt.input)
			assert.Equal(t, tt.major, v.Major)
			assert.Equal(t, tt.minor, v.Minor)
			assert.Equal(t, tt.patch, v.Patch)
			assert.Equal(t, tt.full, v.Full)
		})
	}
}

func TestExtractDocstring(t *testing.T) {
	t.Run("nil comment", func(t *testing.T) {
		assert.Equal(t, "", extractDocstring(nil))
	})

	t.Run("short comment", func(t *testing.T) {
		src := `package p
// Println formats and writes to stdout.
func Println() {}`
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		funcDecl := f.Decls[0].(*ast.FuncDecl)
		doc := extractDocstring(funcDecl.Doc)
		assert.Contains(t, doc, "Println")
	})

	t.Run("long comment truncated", func(t *testing.T) {
		long := strings.Repeat("a", 600)
		src := "package p\n// " + long + "\nfunc F() {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		funcDecl := f.Decls[0].(*ast.FuncDecl)
		doc := extractDocstring(funcDecl.Doc)
		assert.True(t, len(doc) <= 503, "truncated to 500 chars + '...'")
		assert.True(t, strings.HasSuffix(doc, "..."), "truncated doc ends with ...")
	})
}

func TestIsDeprecated(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		dep, msg := isDeprecated(nil)
		assert.False(t, dep)
		assert.Empty(t, msg)
	})

	t.Run("not deprecated", func(t *testing.T) {
		src := "package p\n// Println writes to stdout.\nfunc Println() {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		dep, msg := isDeprecated(f.Decls[0].(*ast.FuncDecl).Doc)
		assert.False(t, dep)
		assert.Empty(t, msg)
	})

	t.Run("deprecated with message", func(t *testing.T) {
		src := "package p\n// OldFunc is old.\n// Deprecated: Use NewFunc instead.\nfunc OldFunc() {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		dep, msg := isDeprecated(f.Decls[0].(*ast.FuncDecl).Doc)
		assert.True(t, dep)
		assert.Equal(t, "Use NewFunc instead.", msg)
	})
}

func TestExprToString(t *testing.T) {
	t.Run("nil expr", func(t *testing.T) {
		fset := token.NewFileSet()
		assert.Equal(t, "", exprToString(nil, fset))
	})

	t.Run("string ident", func(t *testing.T) {
		src := "package p\nfunc F(x string) {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		field := f.Decls[0].(*ast.FuncDecl).Type.Params.List[0]
		assert.Equal(t, "string", exprToString(field.Type, fset))
	})

	t.Run("pointer star expr", func(t *testing.T) {
		src := "package p\nfunc F(x *os.File) {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		field := f.Decls[0].(*ast.FuncDecl).Type.Params.List[0]
		assert.Equal(t, "*os.File", exprToString(field.Type, fset))
	})
}

func TestExtractParams(t *testing.T) {
	t.Run("nil params", func(t *testing.T) {
		params, variadic := extractParams(nil, token.NewFileSet())
		assert.Nil(t, params)
		assert.False(t, variadic)
	})

	t.Run("empty params", func(t *testing.T) {
		src := "package p\nfunc F() {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		params, variadic := extractParams(f.Decls[0].(*ast.FuncDecl).Type.Params, fset)
		assert.Nil(t, params)
		assert.False(t, variadic)
	})

	t.Run("named params", func(t *testing.T) {
		src := "package p\nfunc F(name string, age int) {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		params, variadic := extractParams(f.Decls[0].(*ast.FuncDecl).Type.Params, fset)
		require.Len(t, params, 2)
		assert.Equal(t, "name", params[0].Name)
		assert.Equal(t, "string", params[0].Type)
		assert.Equal(t, "age", params[1].Name)
		assert.Equal(t, "int", params[1].Type)
		assert.False(t, variadic)
	})

	t.Run("unnamed params", func(t *testing.T) {
		src := "package p\nfunc F(string, int) {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		params, variadic := extractParams(f.Decls[0].(*ast.FuncDecl).Type.Params, fset)
		require.Len(t, params, 2)
		assert.Equal(t, "", params[0].Name)
		assert.False(t, variadic)
	})

	t.Run("variadic param", func(t *testing.T) {
		src := "package p\nfunc F(a ...any) {}"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		params, variadic := extractParams(f.Decls[0].(*ast.FuncDecl).Type.Params, fset)
		require.Len(t, params, 1)
		assert.True(t, variadic)
		assert.True(t, params[0].IsVariadic)
		assert.Equal(t, "any", params[0].Type)
	})
}

func TestExtractReturns(t *testing.T) {
	t.Run("nil returns", func(t *testing.T) {
		returns := extractReturns(nil, token.NewFileSet())
		assert.Nil(t, returns)
	})

	t.Run("single unnamed return", func(t *testing.T) {
		src := "package p\nfunc F() string { return \"\" }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		returns := extractReturns(f.Decls[0].(*ast.FuncDecl).Type.Results, fset)
		require.Len(t, returns, 1)
		assert.Equal(t, "string", returns[0].Type)
		assert.Equal(t, "", returns[0].Name)
	})

	t.Run("named multi-return", func(t *testing.T) {
		src := "package p\nfunc F() (n int, err error) { return }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		returns := extractReturns(f.Decls[0].(*ast.FuncDecl).Type.Results, fset)
		require.Len(t, returns, 2)
		assert.Equal(t, "int", returns[0].Type)
		assert.Equal(t, "n", returns[0].Name)
		assert.Equal(t, "error", returns[1].Type)
		assert.Equal(t, "err", returns[1].Name)
	})
}

func TestExtractTypeParams(t *testing.T) {
	t.Run("nil type params", func(t *testing.T) {
		params := extractTypeParams(nil, token.NewFileSet())
		assert.Nil(t, params)
	})

	t.Run("generic function", func(t *testing.T) {
		src := "package p\nfunc Max[T interface{ ~int | ~float64 }](a, b T) T { return a }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		funcDecl := f.Decls[0].(*ast.FuncDecl)
		params := extractTypeParams(funcDecl.Type.TypeParams, fset)
		require.Len(t, params, 1)
		assert.Equal(t, "T", params[0].Name)
		assert.NotEmpty(t, params[0].Constraint)
	})
}

func TestExtractStructFields(t *testing.T) {
	t.Run("nil struct", func(t *testing.T) {
		fields := extractStructFields(nil, token.NewFileSet())
		assert.Nil(t, fields)
	})

	t.Run("exported and unexported fields", func(t *testing.T) {
		src := `package p
type S struct {
    Name string
    age  int
    Tag  string ` + "`json:\"tag\"`" + `
}`
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		st := typeSpec.Type.(*ast.StructType)
		fields := extractStructFields(st, fset)
		require.Len(t, fields, 2, "only exported fields")
		assert.Equal(t, "Name", fields[0].Name)
		assert.Equal(t, "string", fields[0].Type)
		assert.True(t, fields[0].Exported)
		assert.Equal(t, "Tag", fields[1].Name)
		assert.Equal(t, `json:"tag"`, fields[1].Tag)
	})

	t.Run("embedded exported field", func(t *testing.T) {
		src := "package p\ntype S struct { io.Reader }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		st := typeSpec.Type.(*ast.StructType)
		fields := extractStructFields(st, fset)
		require.Len(t, fields, 1)
		assert.Equal(t, "Reader", fields[0].Name)
	})

	t.Run("embedded unexported field skipped", func(t *testing.T) {
		src := "package p\ntype S struct { someType }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		st := typeSpec.Type.(*ast.StructType)
		fields := extractStructFields(st, fset)
		assert.Empty(t, fields)
	})

	t.Run("pointer embedded field", func(t *testing.T) {
		src := "package p\ntype S struct { *Buffer }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		st := typeSpec.Type.(*ast.StructType)
		fields := extractStructFields(st, fset)
		require.Len(t, fields, 1)
		assert.Equal(t, "Buffer", fields[0].Name)
	})
}

func TestEmbeddedFieldName(t *testing.T) {
	tests := []struct {
		src      string
		expected string
	}{
		{"package p; type S struct { Foo }", "Foo"},
		{"package p; type S struct { *Bar }", "Bar"},
		{"package p; type S struct { io.Reader }", "Reader"},
	}
	for _, tt := range tests {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", tt.src, 0)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		st := typeSpec.Type.(*ast.StructType)
		name := embeddedFieldName(st.Fields.List[0].Type)
		assert.Equal(t, tt.expected, name)
	}

	t.Run("unknown expr", func(t *testing.T) {
		// Pass a non-ident/star/selector expression.
		assert.Equal(t, "", embeddedFieldName(&ast.ArrayType{}))
	})
}

func TestExtractInterfaceMethods(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) {
		methods := extractInterfaceMethods(nil, token.NewFileSet())
		assert.Empty(t, methods)
	})

	t.Run("exported methods only", func(t *testing.T) {
		src := `package p
type R interface {
    Read(p []byte) (n int, err error)
}`
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		iface := typeSpec.Type.(*ast.InterfaceType)
		methods := extractInterfaceMethods(iface, fset)
		require.Contains(t, methods, "Read")
		assert.Equal(t, "Read", methods["Read"].Name)
		assert.Equal(t, 1.0, float64(methods["Read"].Confidence))
	})

	t.Run("embedded interface skipped", func(t *testing.T) {
		src := "package p\ntype R interface { io.Reader }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		typeSpec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		iface := typeSpec.Type.(*ast.InterfaceType)
		methods := extractInterfaceMethods(iface, fset)
		assert.Empty(t, methods)
	})
}

func TestContainsIota(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.False(t, containsIota(nil))
	})

	src := `package p
const (
    A = iota
    B
    C = 1 << iota
)`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)
	decl := f.Decls[0].(*ast.GenDecl)
	// A = iota
	spec0 := decl.Specs[0].(*ast.ValueSpec)
	assert.True(t, containsIota(spec0.Values[0]))
	// C = 1 << iota
	spec2 := decl.Specs[2].(*ast.ValueSpec)
	assert.True(t, containsIota(spec2.Values[0]))
}

func TestExtractConsts(t *testing.T) {
	t.Run("iota constants", func(t *testing.T) {
		src := `package p
const (
    // ReadOnly is the read-only flag.
    ReadOnly = iota
    WriteOnly
    ReadWrite
)`
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		decl := f.Decls[0].(*ast.GenDecl)
		consts := extractConsts(decl, fset)
		require.Len(t, consts, 3)
		assert.True(t, consts[0].IsIota)
		assert.Equal(t, 1.0, float64(consts[0].Confidence))
	})

	t.Run("unexported constants skipped", func(t *testing.T) {
		src := "package p\nconst (\n\tPublic = 1\n\tprivate = 2\n)"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		decl := f.Decls[0].(*ast.GenDecl)
		consts := extractConsts(decl, fset)
		require.Len(t, consts, 1)
		assert.Equal(t, "Public", consts[0].Name)
	})

	t.Run("typed constant", func(t *testing.T) {
		src := "package p\nconst StatusOK int = 200"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		decl := f.Decls[0].(*ast.GenDecl)
		consts := extractConsts(decl, fset)
		require.Len(t, consts, 1)
		assert.Equal(t, "int", consts[0].Type)
		assert.Equal(t, "200", consts[0].Value)
	})
}

func TestExtractVars(t *testing.T) {
	t.Run("exported var with type", func(t *testing.T) {
		src := "package p\nvar Stdin *os.File"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		decl := f.Decls[0].(*ast.GenDecl)
		vars := extractVars(decl, fset)
		require.Len(t, vars, 1)
		assert.Equal(t, "Stdin", vars[0].Name)
		assert.Equal(t, "*os.File", vars[0].Type)
		assert.Equal(t, float32(1.0), vars[0].Confidence)
	})

	t.Run("unexported var skipped", func(t *testing.T) {
		src := "package p\nvar (\n\tPublicVar int\n\tprivateVar int\n)"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		decl := f.Decls[0].(*ast.GenDecl)
		vars := extractVars(decl, fset)
		require.Len(t, vars, 1)
		assert.Equal(t, "PublicVar", vars[0].Name)
	})
}

func TestExtractTypeSpec(t *testing.T) {
	t.Run("struct type", func(t *testing.T) {
		src := `package p
// Buffer is a byte buffer.
type Buffer struct {
    Len int
}`
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		genDecl := f.Decls[0].(*ast.GenDecl)
		spec := genDecl.Specs[0].(*ast.TypeSpec)
		t2 := extractTypeSpec(spec, genDecl.Doc, fset)
		assert.Equal(t, "Buffer", t2.Name)
		assert.Equal(t, "struct", t2.Kind)
		assert.Contains(t, t2.Docstring, "Buffer")
	})

	t.Run("interface type", func(t *testing.T) {
		src := "package p\ntype Reader interface { Read(p []byte) (n int, err error) }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		spec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		t2 := extractTypeSpec(spec, nil, fset)
		assert.Equal(t, "interface", t2.Kind)
		assert.Contains(t, t2.Methods, "Read")
	})

	t.Run("alias type", func(t *testing.T) {
		src := "package p\ntype MyInt int"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		spec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		t2 := extractTypeSpec(spec, nil, fset)
		assert.Equal(t, "alias", t2.Kind)
		assert.Equal(t, "int", t2.Underlying)
	})

	t.Run("generic type", func(t *testing.T) {
		src := "package p\ntype Pair[A, B any] struct { First A; Second B }"
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
		require.NoError(t, err)
		spec := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
		t2 := extractTypeSpec(spec, nil, fset)
		assert.True(t, t2.IsGeneric)
		require.Len(t, t2.TypeParams, 2)
		assert.Equal(t, "A", t2.TypeParams[0].Name)
		assert.Equal(t, "B", t2.TypeParams[1].Name)
	})
}

func TestReceiverTypeName(t *testing.T) {
	tests := []struct {
		src      string
		expected string
	}{
		{"package p; func (b Buffer) String() string { return \"\" }", "Buffer"},
		{"package p; func (b *Buffer) Write(p []byte) (int, error) { return 0, nil }", "Buffer"},
	}
	for _, tt := range tests {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", tt.src, 0)
		require.NoError(t, err)
		funcDecl := f.Decls[0].(*ast.FuncDecl)
		recv := funcDecl.Recv.List[0].Type
		assert.Equal(t, tt.expected, receiverTypeName(recv))
	}

	t.Run("unknown type", func(t *testing.T) {
		assert.Equal(t, "", receiverTypeName(&ast.ArrayType{}))
	})
}

func TestFuncSignatureStr(t *testing.T) {
	src := "package p\n// Println writes output.\nfunc Println(a ...any) (n int, err error) {}"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)
	funcDecl := f.Decls[0].(*ast.FuncDecl)
	sig := funcSignatureStr(funcDecl, fset)
	assert.Contains(t, sig, "Println")
	assert.Contains(t, sig, "any")
	// Body should not be present in the signature.
	assert.NotContains(t, sig, "{}")
	// Body should still be intact after the call.
	assert.NotNil(t, funcDecl.Body)
}

// --- Tests for DirHasGoFiles ---

func TestDirHasGoFiles(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		has, err := dirHasGoFiles(dir)
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("directory with go file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package p"), 0o644))
		has, err := dirHasGoFiles(dir)
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("directory with only test file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package p"), 0o644))
		has, err := dirHasGoFiles(dir)
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := dirHasGoFiles("/nonexistent/path/xyz")
		assert.Error(t, err)
	})

	t.Run("subdirectory not counted", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package p"), 0o644))
		has, err := dirHasGoFiles(dir)
		require.NoError(t, err)
		assert.False(t, has)
	})
}

// --- Integration tests using real GOROOT ---

func TestDiscoverPackages(t *testing.T) {
	goroot := runtime.GOROOT()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: goroot, OutputDir: t.TempDir()})

	pkgs, err := e.discoverPackages()
	require.NoError(t, err)

	// There should be many packages in the stdlib.
	assert.Greater(t, len(pkgs), 50, "stdlib has more than 50 packages")

	// fmt must be present.
	assert.Contains(t, pkgs, "fmt")
	// os must be present.
	assert.Contains(t, pkgs, "os")
	// slices must be present (Go 1.21+).
	assert.Contains(t, pkgs, "slices")

	// internal packages must not be present.
	for _, pkg := range pkgs {
		parts := strings.Split(pkg, "/")
		for _, part := range parts {
			assert.NotEqual(t, "internal", part, "package %q contains 'internal' component", pkg)
			assert.NotEqual(t, "cmd", part, "package %q contains 'cmd' component", pkg)
			assert.NotEqual(t, "testdata", part, "package %q contains 'testdata' component", pkg)
		}
	}
}

func TestDiscoverPackages_InvalidGoroot(t *testing.T) {
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: "/nonexistent/goroot", OutputDir: t.TempDir()})
	_, err := e.discoverPackages()
	assert.Error(t, err)
}

func TestExtractPackage_fmt(t *testing.T) {
	goroot := runtime.GOROOT()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: goroot, OutputDir: t.TempDir()})

	pkg, err := e.extractPackage("fmt")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Equal(t, "fmt", pkg.ImportPath)
	assert.Equal(t, "1.26", pkg.GoVersion)
	assert.NotEmpty(t, pkg.GeneratedAt)

	// fmt must have Println, Sprintf, Fprintf, etc.
	assert.Contains(t, pkg.Functions, "Println")
	assert.Contains(t, pkg.Functions, "Sprintf")
	assert.Contains(t, pkg.Functions, "Fprintf")

	// Println should be variadic.
	printlnFn := pkg.Functions["Println"]
	assert.NotNil(t, printlnFn)
	assert.True(t, printlnFn.IsVariadic)
	assert.NotEmpty(t, printlnFn.Signature)
	assert.Equal(t, float32(1.0), printlnFn.Confidence)

	// Sprintf should have a format param.
	sprintf := pkg.Functions["Sprintf"]
	assert.NotNil(t, sprintf)
	assert.NotEmpty(t, sprintf.Params)
	assert.True(t, sprintf.IsVariadic)

	// fmt has the Stringer interface type.
	assert.Contains(t, pkg.Types, "Stringer")
	stringer := pkg.Types["Stringer"]
	assert.Equal(t, "interface", stringer.Kind)
}

func TestExtractPackage_slices_generics(t *testing.T) {
	goroot := runtime.GOROOT()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: goroot, OutputDir: t.TempDir()})

	pkg, err := e.extractPackage("slices")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Equal(t, "slices", pkg.ImportPath)

	// slices.Sort is a generic function.
	sortFn := pkg.Functions["Sort"]
	require.NotNil(t, sortFn, "slices.Sort must exist")
	assert.True(t, sortFn.IsGeneric, "slices.Sort must be generic")
	assert.NotEmpty(t, sortFn.TypeParams, "slices.Sort must have type params")
}

func TestExtractPackage_os(t *testing.T) {
	goroot := runtime.GOROOT()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: goroot, OutputDir: t.TempDir()})

	pkg, err := e.extractPackage("os")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	// os.Stdin, os.Stdout, os.Stderr are package-level variables.
	assert.Contains(t, pkg.Variables, "Stdin")
	assert.Contains(t, pkg.Variables, "Stdout")
	assert.Contains(t, pkg.Variables, "Stderr")

	// os has exported constants (e.g. O_RDONLY).
	assert.Greater(t, len(pkg.Constants), 0, "os has constants")

	// os.File is a struct type.
	assert.Contains(t, pkg.Types, "File")
}

func TestExtractPackage_nonexistent(t *testing.T) {
	goroot := runtime.GOROOT()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: goroot, OutputDir: t.TempDir()})

	_, err := e.extractPackage("nonexistent/package/xyz")
	assert.Error(t, err)
}

func TestWritePackageFile(t *testing.T) {
	outDir := t.TempDir()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: runtime.GOROOT(), OutputDir: outDir})

	pkg := &Package{
		ImportPath:  "fmt",
		GoVersion:   "1.26",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Functions:   map[string]*Function{"Println": {Name: "Println", Confidence: 1.0}},
		Types:       make(map[string]*Type),
		Constants:   make(map[string]*Constant),
		Variables:   make(map[string]*Variable),
	}

	size, checksum, err := e.writePackageFile(pkg)
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))
	assert.True(t, strings.HasPrefix(checksum, "sha256:"))

	// File must exist.
	path := filepath.Join(outDir, "fmt_stdlib.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// Must be valid JSON matching the package.
	var decoded Package
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "fmt", decoded.ImportPath)
}

func TestWritePackageFile_NonWritableDir(t *testing.T) {
	e := NewExtractor(Config{
		GoVersion: "1.26",
		GOROOT:    runtime.GOROOT(),
		OutputDir: "/nonexistent_dir_xyz/output",
	})
	pkg := &Package{
		ImportPath: "fmt",
		Functions:  make(map[string]*Function),
		Types:      make(map[string]*Type),
		Constants:  make(map[string]*Constant),
		Variables:  make(map[string]*Variable),
	}
	_, _, err := e.writePackageFile(pkg)
	assert.Error(t, err)
}

func TestWriteManifestFile(t *testing.T) {
	outDir := t.TempDir()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: runtime.GOROOT(), OutputDir: outDir})

	manifest := &Manifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		GoVersion:       parseGoVersion("1.26"),
		GeneratedAt:     "2026-01-01T00:00:00Z",
		Packages:        []*PackageEntry{},
		Statistics:      &RegistryStats{},
	}

	err := e.writeManifestFile(manifest)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	require.NoError(t, err)

	var decoded Manifest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "1.0.0", decoded.SchemaVersion)
	assert.Equal(t, "v1", decoded.RegistryVersion)
}

func TestWriteManifestFile_NonWritableDir(t *testing.T) {
	e := NewExtractor(Config{
		GoVersion: "1.26",
		GOROOT:    runtime.GOROOT(),
		OutputDir: "/nonexistent_dir_xyz/output",
	})
	err := e.writeManifestFile(&Manifest{
		Packages:   []*PackageEntry{},
		Statistics: &RegistryStats{},
	})
	assert.Error(t, err)
}

func TestBuildManifest(t *testing.T) {
	outDir := t.TempDir()
	e := NewExtractor(Config{GoVersion: "1.26.0", GOROOT: runtime.GOROOT(), OutputDir: outDir})

	packages := []*Package{
		{
			ImportPath: "fmt",
			Functions: map[string]*Function{
				"Println": {Name: "Println", IsGeneric: false},
			},
			Types:     map[string]*Type{"Stringer": {Name: "Stringer", IsGeneric: false}},
			Constants: map[string]*Constant{},
			Variables: map[string]*Variable{},
		},
		{
			ImportPath: "slices",
			Functions: map[string]*Function{
				"Sort": {Name: "Sort", IsGeneric: true},
			},
			Types:     map[string]*Type{},
			Constants: map[string]*Constant{},
			Variables: map[string]*Variable{},
		},
	}

	filesizes := map[string]int64{"fmt": 1000, "slices": 2000}
	checksums := map[string]string{
		"fmt":    "sha256:abc",
		"slices": "sha256:def",
	}

	manifest := e.buildManifest(packages, filesizes, checksums)

	assert.Equal(t, "1.0.0", manifest.SchemaVersion)
	assert.Equal(t, "v1", manifest.RegistryVersion)
	assert.Equal(t, 1, manifest.GoVersion.Major)
	assert.Equal(t, 26, manifest.GoVersion.Minor)
	assert.Len(t, manifest.Packages, 2)
	assert.Equal(t, 2, manifest.Statistics.TotalPackages)
	assert.Equal(t, 2, manifest.Statistics.TotalFunctions)
	assert.Equal(t, 1, manifest.Statistics.TotalTypes)
	assert.Equal(t, 1, manifest.Statistics.PackagesWithGenerics)
	assert.Contains(t, manifest.BaseURL, "go1.26.0")

	// Packages must be sorted alphabetically.
	assert.Equal(t, "fmt", manifest.Packages[0].ImportPath)
	assert.Equal(t, "slices", manifest.Packages[1].ImportPath)
}

func TestBuildManifest_GenericType(t *testing.T) {
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: runtime.GOROOT(), OutputDir: t.TempDir()})

	packages := []*Package{
		{
			ImportPath: "mypkg",
			Functions:  map[string]*Function{},
			Types: map[string]*Type{
				"GenericType": {Name: "GenericType", IsGeneric: true},
			},
			Constants: map[string]*Constant{},
			Variables: map[string]*Variable{},
		},
	}
	manifest := e.buildManifest(packages, map[string]int64{}, map[string]string{})
	assert.Equal(t, 1, manifest.Statistics.PackagesWithGenerics)
}

func TestRun_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	goroot := runtime.GOROOT()
	outDir := t.TempDir()

	// Run extraction on fmt and os only (via a mock GOROOT).
	// We use a real GOROOT but expect many packages to be extracted.
	e := NewExtractor(Config{
		GoVersion: "1.26",
		GOROOT:    goroot,
		OutputDir: outDir,
	})

	err := e.Run()
	require.NoError(t, err)

	// manifest.json must exist and be valid.
	manifestData, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	require.NoError(t, err)

	var manifest Manifest
	require.NoError(t, json.Unmarshal(manifestData, &manifest))
	assert.Greater(t, manifest.Statistics.TotalPackages, 50)
	assert.Equal(t, "1.0.0", manifest.SchemaVersion)
	assert.NotEmpty(t, manifest.GeneratedAt)

	// fmt_stdlib.json must exist.
	fmtData, err := os.ReadFile(filepath.Join(outDir, "fmt_stdlib.json"))
	require.NoError(t, err)

	var fmtPkg Package
	require.NoError(t, json.Unmarshal(fmtData, &fmtPkg))
	assert.Equal(t, "fmt", fmtPkg.ImportPath)
	assert.Contains(t, fmtPkg.Functions, "Println")

	// slices_stdlib.json must exist and contain generic functions.
	slicesData, err := os.ReadFile(filepath.Join(outDir, "slices_stdlib.json"))
	require.NoError(t, err)

	var slicesPkg Package
	require.NoError(t, json.Unmarshal(slicesData, &slicesPkg))
	sortFn := slicesPkg.Functions["Sort"]
	require.NotNil(t, sortFn)
	assert.True(t, sortFn.IsGeneric)

	// Checksums in manifest must match sha256 prefix.
	for _, entry := range manifest.Packages {
		assert.True(t, strings.HasPrefix(entry.Checksum, "sha256:"),
			"package %s checksum must start with sha256:", entry.ImportPath)
	}
}

func TestRun_InvalidOutputDir(t *testing.T) {
	e := NewExtractor(Config{
		GoVersion: "1.26",
		GOROOT:    runtime.GOROOT(),
		OutputDir: "/root/nonwritable_test_dir_xyz",
	})
	err := e.Run()
	assert.Error(t, err)
}

// TestJSONSerialization verifies that JSON output uses snake_case keys as required.
func TestJSONSerialization(t *testing.T) {
	pkg := &Package{
		ImportPath:  "fmt",
		GoVersion:   "1.26.0",
		GeneratedAt: "2026-01-01T00:00:00Z",
		Functions: map[string]*Function{
			"Println": {
				Name:          "Println",
				IsVariadic:    true,
				IsGeneric:     false,
				ReceiverType:  "",
				DeprecatedMsg: "",
			},
		},
		Types:     map[string]*Type{},
		Constants: map[string]*Constant{},
		Variables: map[string]*Variable{},
	}

	data, err := json.Marshal(pkg)
	require.NoError(t, err)
	jsonStr := string(data)

	assert.Contains(t, jsonStr, `"import_path"`)
	assert.Contains(t, jsonStr, `"go_version"`)
	assert.Contains(t, jsonStr, `"generated_at"`)
	assert.Contains(t, jsonStr, `"is_variadic"`)
	assert.Contains(t, jsonStr, `"is_generic"`)
	assert.Contains(t, jsonStr, `"receiver_type"`)
	assert.Contains(t, jsonStr, `"deprecated_msg"`)
}

func TestManifestJSONSerialization(t *testing.T) {
	manifest := &Manifest{
		SchemaVersion:    "1.0.0",
		RegistryVersion:  "v1",
		GeneratorVersion: "1.0.0",
		GoVersion: VersionInfo{
			Major:       1,
			Minor:       26,
			Patch:       0,
			Full:        "1.26.0",
			ReleaseDate: "",
		},
		Packages:   []*PackageEntry{},
		Statistics: &RegistryStats{},
	}

	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	jsonStr := string(data)

	assert.Contains(t, jsonStr, `"schema_version"`)
	assert.Contains(t, jsonStr, `"registry_version"`)
	assert.Contains(t, jsonStr, `"generator_version"`)
	assert.Contains(t, jsonStr, `"go_version"`)
	assert.Contains(t, jsonStr, `"generated_at"`)
	assert.Contains(t, jsonStr, `"release_date"`)
	assert.Contains(t, jsonStr, `"total_packages"`)
	assert.Contains(t, jsonStr, `"total_functions"`)
	assert.Contains(t, jsonStr, `"packages_with_generics"`)
}

// --- Additional coverage tests ---

// TestRun_DiscoverError tests that Run returns an error when GOROOT/src doesn't exist.
func TestRun_DiscoverError(t *testing.T) {
	// Create a fake GOROOT without a src/ directory.
	fakeGoroot := t.TempDir()
	e := NewExtractor(Config{
		GoVersion: "1.26",
		GOROOT:    fakeGoroot,
		OutputDir: t.TempDir(),
	})
	err := e.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discovering packages")
}

// TestRun_ExtractWarning tests that Run logs a warning and continues when a package fails to parse.
func TestRun_ExtractWarning(t *testing.T) {
	// Build a fake GOROOT with one valid package and one with a parse error.
	fakeGoroot := t.TempDir()
	srcDir := filepath.Join(fakeGoroot, "src")

	// Valid package: mypkg
	validPkg := filepath.Join(srcDir, "mypkg")
	require.NoError(t, os.MkdirAll(validPkg, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(validPkg, "mypkg.go"),
		[]byte("package mypkg\n\n// Hello says hello.\nfunc Hello() string { return \"hello\" }\n"),
		0o644,
	))

	// Invalid package: badpkg (syntax error)
	badPkg := filepath.Join(srcDir, "badpkg")
	require.NoError(t, os.MkdirAll(badPkg, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(badPkg, "badpkg.go"),
		[]byte("package badpkg\nfunc Broken( {}\n"), // syntax error
		0o644,
	))

	outDir := t.TempDir()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: fakeGoroot, OutputDir: outDir})
	err := e.Run()
	// Run should succeed despite the bad package (it only warns and skips).
	require.NoError(t, err)

	// Only mypkg_stdlib.json should exist (badpkg was skipped).
	_, err = os.Stat(filepath.Join(outDir, "mypkg_stdlib.json"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(outDir, "badpkg_stdlib.json"))
	assert.True(t, os.IsNotExist(err))
}

// TestRun_ManifestWriteError tests that Run returns an error when the manifest can't be written.
// We create manifest.json as a directory to block the file write.
func TestRun_ManifestWriteError(t *testing.T) {
	fakeGoroot := t.TempDir()
	srcDir := filepath.Join(fakeGoroot, "src", "mypkg")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(srcDir, "mypkg.go"),
		[]byte("package mypkg\n\n// Fn is a function.\nfunc Fn() {}\n"),
		0o644,
	))

	outDir := t.TempDir()
	// Block manifest.json by creating a directory with that name.
	require.NoError(t, os.MkdirAll(filepath.Join(outDir, "manifest.json"), 0o755))

	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: fakeGoroot, OutputDir: outDir})
	err := e.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest")
}

// TestRun_PackageWriteError tests that Run returns an error when a package file can't be written.
func TestRun_PackageWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test not meaningful as root")
	}
	fakeGoroot := t.TempDir()
	srcDir := filepath.Join(fakeGoroot, "src", "mypkg")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(srcDir, "mypkg.go"),
		[]byte("package mypkg\n\n// Fn is a function.\nfunc Fn() {}\n"),
		0o644,
	))

	outDir := t.TempDir()
	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: fakeGoroot, OutputDir: outDir})

	// Make the output directory read-only so package file writes fail.
	require.NoError(t, os.Chmod(outDir, 0o555))
	defer os.Chmod(outDir, 0o755) // restore for cleanup

	err := e.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writing")
}

// TestDiscoverPackages_UnreadableDir tests the error path when a directory can't be read.
func TestDiscoverPackages_UnreadableDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test not meaningful as root")
	}

	fakeGoroot := t.TempDir()
	srcDir := filepath.Join(fakeGoroot, "src")

	// Create a regular package.
	validPkg := filepath.Join(srcDir, "mypkg")
	require.NoError(t, os.MkdirAll(validPkg, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(validPkg, "mypkg.go"),
		[]byte("package mypkg"),
		0o644,
	))

	// Create an unreadable subdirectory.
	unreadable := filepath.Join(srcDir, "secret")
	require.NoError(t, os.MkdirAll(unreadable, 0o000))
	defer os.Chmod(unreadable, 0o755) // restore for cleanup

	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: fakeGoroot, OutputDir: t.TempDir()})
	_, err := e.discoverPackages()
	assert.Error(t, err)
}

// TestExtractPackage_TestOnlyPackage tests that a directory with only _test package is skipped.
func TestExtractPackage_TestOnlyPackage(t *testing.T) {
	// Create a fake GOROOT with a package that only has _test package files.
	fakeGoroot := t.TempDir()
	srcDir := filepath.Join(fakeGoroot, "src")

	testOnlyPkg := filepath.Join(srcDir, "testonly")
	require.NoError(t, os.MkdirAll(testOnlyPkg, 0o755))
	// File declares package testonly_test (external test package).
	require.NoError(t, os.WriteFile(
		filepath.Join(testOnlyPkg, "doc.go"),
		[]byte("package testonly_test\n"),
		0o644,
	))

	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: fakeGoroot, OutputDir: t.TempDir()})
	pkg, err := e.extractPackage("testonly")
	require.NoError(t, err)
	// All declarations were in the _test package and skipped.
	assert.Empty(t, pkg.Functions)
}

// TestReceiverTypeName_IndexListExpr tests the multi-param generic receiver case.
func TestReceiverTypeName_IndexListExpr(t *testing.T) {
	// Parse a function with a multi-param generic receiver: Map[K, V].
	src := `package p
type Map[K comparable, V any] struct{}
func (m Map[K, V]) Get(k K) V { var v V; return v }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	require.NoError(t, err)

	// Find the method declaration.
	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}
		recv := funcDecl.Recv.List[0].Type
		name := receiverTypeName(recv)
		assert.Equal(t, "Map", name)
		return
	}
	t.Fatal("no method declaration found")
}

// TestExtractFromFile_MethodOnKnownType tests that method declarations are attached to their type.
func TestExtractFromFile_MethodOnKnownType(t *testing.T) {
	fakeGoroot := t.TempDir()
	srcDir := filepath.Join(fakeGoroot, "src", "mypkg")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(srcDir, "mypkg.go"),
		[]byte(`package mypkg

// Buffer is a byte buffer.
type Buffer struct{}

// Write writes bytes to the buffer.
func (b *Buffer) Write(p []byte) (int, error) { return 0, nil }
`),
		0o644,
	))

	e := NewExtractor(Config{GoVersion: "1.26", GOROOT: fakeGoroot, OutputDir: t.TempDir()})
	pkg, err := e.extractPackage("mypkg")
	require.NoError(t, err)

	// The method must be attached to the Buffer type.
	buf := pkg.Types["Buffer"]
	require.NotNil(t, buf)
	assert.Contains(t, buf.Methods, "Write")
	// Also available as a top-level function with receiver prefix.
	assert.Contains(t, pkg.Functions, "Buffer.Write")
}
