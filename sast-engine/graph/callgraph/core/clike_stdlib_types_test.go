package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCStdlibRegistryPrealloc(t *testing.T) {
	r := NewCStdlibRegistry()
	require.NotNil(t, r)
	require.NotNil(t, r.Headers)
	assert.Equal(t, 0, r.HeaderCount())
}

func TestCStdlibRegistry_HeaderAccessors(t *testing.T) {
	r := NewCStdlibRegistry()
	assert.False(t, r.HasHeader("stdio.h"))
	assert.Nil(t, r.GetHeader("stdio.h"))

	h := NewCStdlibHeader()
	h.Header = "stdio.h"
	r.Headers["stdio.h"] = h

	assert.True(t, r.HasHeader("stdio.h"))
	assert.Equal(t, h, r.GetHeader("stdio.h"))
	assert.Equal(t, 1, r.HeaderCount())
}

func TestCStdlibRegistry_GetFunction(t *testing.T) {
	r := NewCStdlibRegistry()
	// Missing header
	assert.Nil(t, r.GetFunction("stdio.h", "printf"))

	h := NewCStdlibHeader()
	h.Functions["printf"] = &CStdlibFunction{FQN: "c::stdio::printf"}
	h.FreeFunctions["std::move"] = &CStdlibFunction{FQN: "std::move"}
	r.Headers["stdio.h"] = h

	// Hits Functions map
	got := r.GetFunction("stdio.h", "printf")
	require.NotNil(t, got)
	assert.Equal(t, "c::stdio::printf", got.FQN)

	// Hits FreeFunctions fallback
	got = r.GetFunction("stdio.h", "std::move")
	require.NotNil(t, got)
	assert.Equal(t, "std::move", got.FQN)

	// Missing function in present header
	assert.Nil(t, r.GetFunction("stdio.h", "unknown_func"))
}

func TestCStdlibRegistry_GetClassAndMethod(t *testing.T) {
	r := NewCStdlibRegistry()
	// Missing header
	assert.Nil(t, r.GetClass("vector", "std::vector"))
	assert.Nil(t, r.GetMethod("vector", "std::vector", "push_back"))

	h := NewCStdlibHeader()
	cls := NewCppStdlibClass("std::vector")
	cls.Methods["push_back"] = &CStdlibFunction{FQN: "std::vector::push_back"}
	h.Classes["std::vector"] = cls
	r.Headers["vector"] = h

	// Class hit
	got := r.GetClass("vector", "std::vector")
	require.NotNil(t, got)
	assert.Equal(t, "std::vector", got.FQN)

	// Method hit
	method := r.GetMethod("vector", "std::vector", "push_back")
	require.NotNil(t, method)
	assert.Equal(t, "std::vector::push_back", method.FQN)

	// Missing class in present header
	assert.Nil(t, r.GetClass("vector", "std::string"))
	assert.Nil(t, r.GetMethod("vector", "std::string", "c_str"))

	// Missing method in present class
	assert.Nil(t, r.GetMethod("vector", "std::vector", "unknown_method"))
}

func TestCStdlibManifest_HasAndGetHeaderEntry(t *testing.T) {
	m := NewCStdlibManifest()
	require.NotNil(t, m)
	require.NotNil(t, m.Headers)
	require.NotNil(t, m.Statistics)

	assert.False(t, m.HasHeader("stdio.h"))
	assert.Nil(t, m.GetHeaderEntry("stdio.h"))

	entry := &CStdlibHeaderEntry{
		Header:   "stdio.h",
		ModuleID: "c::stdio",
		File:     "stdio_stdlib.json",
		URL:      "https://assets.codepathfinder.dev/registries/linux/c/v1/stdio_stdlib.json",
		Size:     1024,
		Checksum: "sha256:abc",
	}
	m.Headers = append(m.Headers, entry)

	assert.True(t, m.HasHeader("stdio.h"))
	assert.Equal(t, entry, m.GetHeaderEntry("stdio.h"))

	// Non-matching name
	assert.False(t, m.HasHeader("string.h"))
	assert.Nil(t, m.GetHeaderEntry("string.h"))
}

func TestNewCStdlibHeader_PreallocatesAllMaps(t *testing.T) {
	h := NewCStdlibHeader()
	require.NotNil(t, h.Functions)
	require.NotNil(t, h.Typedefs)
	require.NotNil(t, h.Constants)
	require.NotNil(t, h.Classes)
	require.NotNil(t, h.FreeFunctions)
}

func TestNewCppStdlibClass_PreallocatesMethodsAndArgs(t *testing.T) {
	cls := NewCppStdlibClass("std::vector")
	assert.Equal(t, "std::vector", cls.FQN)
	require.NotNil(t, cls.Methods)
	require.NotNil(t, cls.DefaultTemplateArgs)
}

// Round-trip a fully populated manifest through JSON. Asserts every field survives.
func TestCStdlibManifest_JSONRoundTrip(t *testing.T) {
	m := &CStdlibManifest{
		SchemaVersion:    "1.0.0",
		RegistryVersion:  "v1",
		Platform:         PlatformLinux,
		Language:         LanguageC,
		SystemTag:        "glibc-2.39",
		GeneratedAt:      "2026-05-04T10:30:00Z",
		GeneratorVersion: "1.0.0",
		BaseURL:          "https://assets.codepathfinder.dev/registries/linux/c/v1",
		Headers: []*CStdlibHeaderEntry{
			{
				Header:   "stdio.h",
				ModuleID: "c::stdio",
				File:     "stdio_stdlib.json",
				URL:      "https://example/stdio_stdlib.json",
				Size:     2048,
				Checksum: "sha256:abc123",
			},
		},
		Statistics: &CStdlibStatistics{
			TotalHeaders:     1,
			TotalFunctions:   42,
			TotalClasses:     0,
			TotalTypedefs:    7,
			TotalConstants:   12,
			OverlayOverrides: 3,
		},
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var got CStdlibManifest
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, *m, got)
}

func TestCStdlibHeader_JSONRoundTrip_C(t *testing.T) {
	h := &CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "stdio.h",
		ModuleID:      "c::stdio",
		Language:      LanguageC,
		Platform:      PlatformLinux,
		SystemTag:     "glibc-2.39",
		GeneratedAt:   "2026-05-04T10:30:00Z",
		Functions: map[string]*CStdlibFunction{
			"printf": {
				FQN:        "c::stdio::printf",
				ReturnType: "int",
				Params: []*CStdlibParam{
					{Name: "format", Type: "const char*", Required: true, Attribute: "format(printf, 1, 2)"},
					{Name: "...", Type: "variadic", Required: false},
				},
				Confidence:  1.0,
				Source:      SourceOverlay,
				SecurityTag: "format_string_sink",
			},
		},
		Typedefs: map[string]*CStdlibTypedef{
			"FILE": {Type: "struct __FILE", PlatformSpecific: false, Source: SourceHeader},
		},
		Constants: map[string]*CStdlibConstant{
			"EOF": {Type: "int", Value: "-1", Source: SourceHeader},
		},
	}

	data, err := json.Marshal(h)
	require.NoError(t, err)

	// Confirm omitempty drops Classes / FreeFunctions / Namespaces for C-only headers.
	assert.NotContains(t, string(data), `"classes":`)
	assert.NotContains(t, string(data), `"free_functions":`)
	assert.NotContains(t, string(data), `"namespaces":`)

	var got CStdlibHeader
	require.NoError(t, json.Unmarshal(data, &got))
	// Empty maps decode to nil; copy them across before comparing.
	assert.Equal(t, h.Functions, got.Functions)
	assert.Equal(t, h.Typedefs, got.Typedefs)
	assert.Equal(t, h.Constants, got.Constants)
	assert.Equal(t, h.Header, got.Header)
}

func TestCStdlibHeader_JSONRoundTrip_Cpp(t *testing.T) {
	h := &CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "vector",
		ModuleID:      "std::vector",
		Language:      LanguageCpp,
		Platform:      PlatformLinux,
		SystemTag:     "libstdc++-13",
		GeneratedAt:   "2026-05-04T10:30:00Z",
		Namespaces:    []string{"std"},
		Classes: map[string]*CppStdlibClass{
			"std::vector": {
				FQN:        "std::vector",
				TypeParams: []string{"T", "Allocator"},
				DefaultTemplateArgs: map[string]string{
					"Allocator": "std::allocator<T>",
				},
				Methods: map[string]*CStdlibFunction{
					"push_back": {
						FQN:        "std::vector::push_back",
						ReturnType: "void",
						Params:     []*CStdlibParam{{Name: "value", Type: "const T&", Required: true}},
						Confidence: 1.0,
						Source:     SourceOverlay,
					},
					"at": {
						FQN:        "std::vector::at",
						ReturnType: "T&",
						Params:     []*CStdlibParam{{Name: "pos", Type: "size_t", Required: true}},
						Confidence: 1.0,
						Source:     SourceOverlay,
						Throws:     "std::out_of_range",
					},
				},
				Constructors: []*CppStdlibConstructor{
					{Params: []*CStdlibParam{}, Source: SourceHeader},
				},
			},
		},
		FreeFunctions: map[string]*CStdlibFunction{
			"std::swap": {
				FQN:        "std::swap",
				ReturnType: "void",
				Source:     SourceHeader,
			},
		},
	}

	data, err := json.Marshal(h)
	require.NoError(t, err)

	var got CStdlibHeader
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, h.Classes, got.Classes)
	assert.Equal(t, h.FreeFunctions, got.FreeFunctions)
	assert.Equal(t, h.Namespaces, got.Namespaces)
}

func TestSourceConstants_AreDistinct(t *testing.T) {
	assert.NotEqual(t, SourceHeader, SourceOverlay)
	assert.NotEqual(t, SourceOverlay, SourceMerged)
	assert.NotEqual(t, SourceHeader, SourceMerged)
	assert.Equal(t, "header", SourceHeader)
	assert.Equal(t, "overlay", SourceOverlay)
	assert.Equal(t, "merged", SourceMerged)
}

func TestLanguageAndPlatformConstants(t *testing.T) {
	assert.Equal(t, "c", LanguageC)
	assert.Equal(t, "cpp", LanguageCpp)
	assert.Equal(t, "linux", PlatformLinux)
	assert.Equal(t, "windows", PlatformWindows)
	assert.Equal(t, "darwin", PlatformDarwin)
}
