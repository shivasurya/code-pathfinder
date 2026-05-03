package clikeextract

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/clike"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cppFixtureDir = "testdata/cpp"

func cppTestSource() HeaderSource {
	return HeaderSource{
		Platform:   core.PlatformLinux,
		Language:   core.LanguageCpp,
		SystemTag:  "libstdc++-test",
		SearchDirs: []string{cppFixtureDir},
		HeaderExts: []string{".h", ".hpp", ""},
	}
}

func TestExtractCppHeader_VectorClassWithTemplateAndMethods(t *testing.T) {
	src := cppTestSource()
	hf := HeaderFile{Name: "vector", Path: filepath.Join(cppFixtureDir, "vector")}

	h, err := extractCppHeader(hf, src)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "vector", h.Header)
	assert.Equal(t, "std::vector", h.ModuleID)
	assert.Equal(t, core.LanguageCpp, h.Language)

	// Class extracted with FQN.
	cls := h.Classes["std::vector"]
	require.NotNil(t, cls, "std::vector class must be extracted")
	assert.Equal(t, "std::vector", cls.FQN)

	// Template parameter names captured.
	assert.Equal(t, []string{"T", "Allocator"}, cls.TypeParams)

	// Methods present.
	for _, name := range []string{"push_back", "at", "operator[]", "size", "empty", "data", "clear"} {
		assert.Contains(t, cls.Methods, name, "method %q expected", name)
	}
	pushBack := cls.Methods["push_back"]
	require.NotNil(t, pushBack)
	assert.Equal(t, "void", pushBack.ReturnType)
	assert.Equal(t, "std::vector::push_back", pushBack.FQN)
	assert.Equal(t, core.SourceHeader, pushBack.Source)

	// Constructors recognised.
	assert.GreaterOrEqual(t, len(cls.Constructors), 3)

	// Free function in std namespace lands in FreeFunctions.
	swap := h.FreeFunctions["std::swap"]
	require.NotNil(t, swap)
	assert.Equal(t, "std::swap", swap.FQN)

	// Top-level namespace recorded.
	assert.Contains(t, h.Namespaces, "std")
}

func TestExtractCppHeader_StringClassAndTypedef(t *testing.T) {
	src := cppTestSource()
	hf := HeaderFile{Name: "string", Path: filepath.Join(cppFixtureDir, "string")}

	h, err := extractCppHeader(hf, src)
	require.NoError(t, err)

	cls := h.Classes["std::basic_string"]
	require.NotNil(t, cls)
	assert.Equal(t, []string{"CharT"}, cls.TypeParams)

	cstr := cls.Methods["c_str"]
	require.NotNil(t, cstr)

	// operator+= captured (operator overloads share the function_declarator shape).
	assert.NotNil(t, cls.Methods["operator+="])

	// Typedefs present (size_t at namespace scope, string as the alias).
	require.Contains(t, h.Typedefs, "size_t")
	require.Contains(t, h.Typedefs, "string")
}

func TestExtractCppHeader_PrivateNamespaceSkipped(t *testing.T) {
	src := cppTestSource()
	hf := HeaderFile{Name: "utility", Path: filepath.Join(cppFixtureDir, "utility")}

	h, err := extractCppHeader(hf, src)
	require.NoError(t, err)

	// Templated free functions land in FreeFunctions (under namespace std).
	assert.NotNil(t, h.FreeFunctions["std::move"])
	assert.NotNil(t, h.FreeFunctions["std::forward"])

	// Private/library-internal names dropped.
	assert.Nil(t, h.FreeFunctions["std::_internal_helper"])
	assert.Nil(t, h.FreeFunctions["std::__detail_helper"])

	// File-scope free function lands in Functions, not FreeFunctions.
	assert.NotNil(t, h.Functions["file_scope_func"])
	assert.Nil(t, h.FreeFunctions["file_scope_func"])

	// Compiler-internal namespace `__detail` was skipped wholesale.
	for fqn := range h.FreeFunctions {
		assert.NotContains(t, fqn, "__detail")
	}
	for fqn := range h.Classes {
		assert.NotContains(t, fqn, "__detail")
	}
}

func TestExtractCppHeader_FileNotFound(t *testing.T) {
	_, err := extractCppHeader(HeaderFile{Name: "absent", Path: "/no-such-cpp"}, cppTestSource())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestExtractTemplateParamNames(t *testing.T) {
	t.Run("nil list yields nil", func(t *testing.T) {
		got := extractTemplateParamNames(nil, nil)
		assert.Nil(t, got)
	})
	// Other branches are exercised end-to-end via the vector and string fixtures
	// (TestExtractCppHeader_VectorClassWithTemplateAndMethods,
	// TestExtractCppHeader_StringClassAndTypedef).
}

func TestFinaliseNamespaces_DedupsAndOrders(t *testing.T) {
	h := core.NewCStdlibHeader()
	h.FreeFunctions["std::move"] = &core.CStdlibFunction{}
	h.FreeFunctions["std::forward"] = &core.CStdlibFunction{}
	h.FreeFunctions["boost::lexical_cast"] = &core.CStdlibFunction{}
	h.Classes["std::vector"] = &core.CppStdlibClass{}
	h.Classes["std::map"] = &core.CppStdlibClass{}

	finaliseNamespaces(h)

	sort.Strings(h.Namespaces)
	assert.Equal(t, []string{"boost", "std"}, h.Namespaces)
}

func TestFinaliseNamespaces_BareNamesIgnored(t *testing.T) {
	h := core.NewCStdlibHeader()
	h.FreeFunctions["bare_func"] = &core.CStdlibFunction{}
	finaliseNamespaces(h)
	assert.Empty(t, h.Namespaces)
}

func TestQualified(t *testing.T) {
	w := &cppWalker{}
	assert.Equal(t, "foo", w.qualified("foo"))

	w.namespaceStack = []string{"std"}
	assert.Equal(t, "std::foo", w.qualified("foo"))

	w.namespaceStack = []string{"std", "detail"}
	assert.Equal(t, "std::detail::foo", w.qualified("foo"))
}

func TestFindFirstIdentifier_Nil(t *testing.T) {
	assert.Equal(t, "", findFirstIdentifier(nil, nil))
}

// TestExtractCppHeader_InlineFunctionDefinition exercises
// extractFreeFunctionFromDefinition (the function_definition branch in walk).
func TestExtractCppHeader_InlineFunctionDefinition(t *testing.T) {
	dir := t.TempDir()
	hp := filepath.Join(dir, "inline_funcs")
	mustWriteFile(t, hp, `
namespace std {
inline int square(int x) { return x * x; }
}
inline int file_scope_inline(int x) { return x; }
`)

	src := HeaderSource{
		Platform: core.PlatformLinux, Language: core.LanguageCpp,
		SystemTag: "test", SearchDirs: []string{dir}, HeaderExts: []string{""},
	}
	h, err := extractCppHeader(HeaderFile{Name: "inline_funcs", Path: hp}, src)
	require.NoError(t, err)

	// Inline function inside namespace lands in FreeFunctions under the FQN.
	require.NotNil(t, h.FreeFunctions["std::square"])

	// File-scope inline lands in Functions.
	require.NotNil(t, h.Functions["file_scope_inline"])
}

func TestParamListFromInfo_Variadic(t *testing.T) {
	got := paramListFromInfo(&clike.FunctionInfo{
		ParamNames: []string{"x", "..."},
		ParamTypes: []string{"int", "..."},
	})
	require.Len(t, got, 2)
	assert.False(t, got[1].Required)
	assert.Equal(t, "variadic", got[1].Type)
}

func TestParamListFromInfo_NormalParams(t *testing.T) {
	got := paramListFromInfo(&clike.FunctionInfo{
		ParamNames: []string{"a", "b"},
		ParamTypes: []string{"int", "const char*"},
	})
	require.Len(t, got, 2)
	assert.True(t, got[0].Required)
	assert.Equal(t, "int", got[0].Type)
	assert.Equal(t, "const char*", got[1].Type)
}

func TestParamListFromInfo_NameTypeMismatch(t *testing.T) {
	// Defensive: when ParamTypes is shorter than ParamNames, we emit an empty
	// type rather than panicking.
	got := paramListFromInfo(&clike.FunctionInfo{
		ParamNames: []string{"a", "b"},
		ParamTypes: []string{"int"},
	})
	require.Len(t, got, 2)
	assert.Equal(t, "", got[1].Type)
}
