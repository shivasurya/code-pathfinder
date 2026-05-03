package clikeextract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeOverlay is a test helper that materialises a YAML overlay in a temp dir
// and returns its absolute path.
func writeOverlay(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "overlay.yaml")
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
	return p
}

func TestLoadOverlay_EmptyPathReturnsNil(t *testing.T) {
	o, err := LoadOverlay("", core.LanguageC)
	require.NoError(t, err)
	assert.Nil(t, o)
}

func TestLoadOverlay_MissingFile(t *testing.T) {
	_, err := LoadOverlay("/nonexistent-overlay-pr01.yaml", core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestLoadOverlay_InvalidYAML(t *testing.T) {
	p := writeOverlay(t, "::: not yaml :::")
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing YAML")
}

func TestLoadOverlay_MissingSchemaVersion(t *testing.T) {
	p := writeOverlay(t, `language: c
overrides: []
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version")
}

func TestLoadOverlay_MissingLanguage(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
overrides: []
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "language is required")
}

func TestLoadOverlay_LanguageMismatch(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: cpp
overrides: []
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `declares language="cpp" but extractor wants "c"`)
}

func TestLoadOverlay_UnsupportedLanguage(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: rust
overrides: []
`)
	_, err := LoadOverlay(p, "rust")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "language must be")
}

func TestLoadOverlay_ValidEmpty(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides: []
`)
	o, err := LoadOverlay(p, core.LanguageC)
	require.NoError(t, err)
	require.NotNil(t, o)
	assert.Equal(t, "1.0.0", o.SchemaVersion)
	assert.Equal(t, core.LanguageC, o.Language)
}

func TestLoadOverlay_FunctionOverride(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides:
  - header: stdio.h
    function: printf
    return_type: int
    params:
      - { name: format, type: const char* }
    confidence: 0.9
    security_tag: format_string_sink
`)
	o, err := LoadOverlay(p, core.LanguageC)
	require.NoError(t, err)
	require.Len(t, o.Overrides, 1)
	assert.Equal(t, "printf", o.Overrides[0].Function)
	assert.InDelta(t, 0.9, o.Overrides[0].Confidence, 0.001)
}

func TestLoadOverlay_OverrideMissingHeader(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides:
  - function: printf
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "header is required")
}

func TestLoadOverlay_OverrideMissingIdentifier(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides:
  - header: stdio.h
    return_type: int
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of function, method, typedef, constant")
}

func TestLoadOverlay_OverrideMultipleIdentifiers(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides:
  - header: stdio.h
    function: printf
    typedef: FILE
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "got multiple")
}

func TestLoadOverlay_MethodOverrideWithoutClass(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: cpp
overrides:
  - header: vector
    method: push_back
`)
	_, err := LoadOverlay(p, core.LanguageCpp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "method override requires class")
}

func TestLoadOverlay_MethodOverrideInCOverlayRejected(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides:
  - header: stdio.h
    class: FILE
    method: read
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "class+method only valid in cpp")
}

func TestLoadOverlay_SkipNeedsExactlyOne(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides: []
skip:
  - prefix: __builtin_
    exact: malloc
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of prefix or exact")
}

func TestLoadOverlay_AliasNeedsBothFields(t *testing.T) {
	p := writeOverlay(t, `schema_version: "1.0.0"
language: c
overrides: []
cross_platform_aliases:
  - alias: _stdio.h
`)
	_, err := LoadOverlay(p, core.LanguageC)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alias and canonical")
}

// Merge tests. Each builds a CStdlibHeader, applies an overlay, asserts the
// final state.

//nolint:unparam // name is parameterised for future test cases on other headers.
func newTestHeader(name string) *core.CStdlibHeader {
	h := core.NewCStdlibHeader()
	h.Header = name
	h.ModuleID = "c::" + SanitizeHeaderName(name)
	h.Language = core.LanguageC
	return h
}

func TestMergeOverlay_NilOverlayIsNoop(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Functions["printf"] = &core.CStdlibFunction{FQN: "c::stdio::printf", Source: core.SourceHeader}

	count := MergeOverlay(h, nil)
	assert.Equal(t, 0, count)
	assert.Equal(t, core.SourceHeader, h.Functions["printf"].Source)
}

func TestMergeOverlay_NilHeader(t *testing.T) {
	o := &Overlay{Language: core.LanguageC, SchemaVersion: "1.0.0"}
	count := MergeOverlay(nil, o)
	assert.Equal(t, 0, count)
}

func TestMergeOverlay_FunctionOverride_HeaderOnlyBecomesMerged(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Functions["printf"] = &core.CStdlibFunction{
		FQN: "c::stdio::printf", ReturnType: "int", Source: core.SourceHeader,
	}
	o := &Overlay{
		SchemaVersion: "1.0.0",
		Language:      core.LanguageC,
		Overrides: []OverlayOverride{
			{Header: "stdio.h", Function: "printf", SecurityTag: "format_string_sink"},
		},
	}

	applied := MergeOverlay(h, o)
	assert.Equal(t, 1, applied)
	got := h.Functions["printf"]
	assert.Equal(t, core.SourceMerged, got.Source)
	assert.Equal(t, "format_string_sink", got.SecurityTag)
	assert.Equal(t, "int", got.ReturnType, "untouched fields preserved")
}

func TestMergeOverlay_FunctionOverride_OverlayOnlyBecomesOverlay(t *testing.T) {
	h := newTestHeader("stdio.h")
	o := &Overlay{
		SchemaVersion: "1.0.0",
		Language:      core.LanguageC,
		Overrides: []OverlayOverride{
			{Header: "stdio.h", Function: "system", ReturnType: "int", SecurityTag: "command_injection_sink"},
		},
	}

	applied := MergeOverlay(h, o)
	assert.Equal(t, 1, applied)
	got := h.Functions["system"]
	require.NotNil(t, got)
	assert.Equal(t, core.SourceOverlay, got.Source)
	assert.Equal(t, "command_injection_sink", got.SecurityTag)
	assert.Equal(t, "c::stdio::system", got.FQN)
	assert.InDelta(t, float32(1.0), got.Confidence, 0.001, "default confidence")
}

func TestMergeOverlay_FunctionOverride_HeaderMismatchIsSkipped(t *testing.T) {
	h := newTestHeader("stdio.h")
	o := &Overlay{
		SchemaVersion: "1.0.0",
		Language:      core.LanguageC,
		Overrides: []OverlayOverride{
			{Header: "string.h", Function: "strcpy"},
		},
	}
	applied := MergeOverlay(h, o)
	assert.Equal(t, 0, applied)
	assert.Empty(t, h.Functions)
}

func TestMergeOverlay_MethodOverride(t *testing.T) {
	h := core.NewCStdlibHeader()
	h.Header = "vector"
	h.ModuleID = "std::vector"
	h.Language = core.LanguageCpp

	o := &Overlay{
		SchemaVersion: "1.0.0",
		Language:      core.LanguageCpp,
		Overrides: []OverlayOverride{
			{Header: "vector", Class: "std::vector", Method: "push_back", ReturnType: "void"},
			{Header: "vector", Class: "std::vector", Method: "at", ReturnType: "T&", Throws: "std::out_of_range"},
		},
	}
	applied := MergeOverlay(h, o)
	assert.Equal(t, 2, applied)

	cls := h.Classes["std::vector"]
	require.NotNil(t, cls)
	pb := cls.Methods["push_back"]
	require.NotNil(t, pb)
	assert.Equal(t, "void", pb.ReturnType)
	assert.Equal(t, core.SourceOverlay, pb.Source)
	assert.Equal(t, "std::vector::push_back", pb.FQN)

	at := cls.Methods["at"]
	require.NotNil(t, at)
	assert.Equal(t, "std::out_of_range", at.Throws)
}

func TestMergeOverlay_MethodOverride_ExistingClassAndMethodMerged(t *testing.T) {
	h := core.NewCStdlibHeader()
	h.Header = "vector"
	cls := core.NewCppStdlibClass("std::vector")
	cls.Methods["size"] = &core.CStdlibFunction{
		FQN: "std::vector::size", ReturnType: "size_t", Source: core.SourceHeader,
	}
	h.Classes["std::vector"] = cls

	o := &Overlay{
		SchemaVersion: "1.0.0",
		Language:      core.LanguageCpp,
		Overrides: []OverlayOverride{
			{Header: "vector", Class: "std::vector", Method: "size", Throws: ""},
			{
				Header: "vector", Class: "std::vector", Method: "size",
				ReturnType: "std::size_t", Attribute: "nodiscard",
				Params: []OverlayParam{},
			},
		},
	}
	applied := MergeOverlay(h, o)
	assert.Equal(t, 2, applied)
	got := h.Classes["std::vector"].Methods["size"]
	assert.Equal(t, "std::size_t", got.ReturnType)
	assert.Equal(t, "nodiscard", got.Attribute)
	assert.Equal(t, core.SourceMerged, got.Source)
}

func TestMergeOverlay_TypedefOverride(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Typedefs["FILE"] = &core.CStdlibTypedef{Type: "struct __FILE", Source: core.SourceHeader}

	o := &Overlay{
		SchemaVersion: "1.0.0", Language: core.LanguageC,
		Overrides: []OverlayOverride{
			{Header: "stdio.h", Typedef: "FILE", Type: "struct _IO_FILE"},
			{Header: "stdio.h", Typedef: "fpos_t", Type: "long"},
		},
	}
	applied := MergeOverlay(h, o)
	assert.Equal(t, 2, applied)
	assert.Equal(t, "struct _IO_FILE", h.Typedefs["FILE"].Type)
	assert.Equal(t, core.SourceMerged, h.Typedefs["FILE"].Source)
	assert.Equal(t, "long", h.Typedefs["fpos_t"].Type)
	assert.Equal(t, core.SourceOverlay, h.Typedefs["fpos_t"].Source)
}

func TestMergeOverlay_ConstantOverride(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Constants["EOF"] = &core.CStdlibConstant{Type: "int", Value: "-1", Source: core.SourceHeader}

	o := &Overlay{
		SchemaVersion: "1.0.0", Language: core.LanguageC,
		Overrides: []OverlayOverride{
			{Header: "stdio.h", Constant: "EOF", Value: "(-1)"},
			{Header: "stdio.h", Constant: "BUFSIZ", Type: "int", Value: "8192"},
		},
	}
	applied := MergeOverlay(h, o)
	assert.Equal(t, 2, applied)
	assert.Equal(t, "(-1)", h.Constants["EOF"].Value)
	assert.Equal(t, core.SourceMerged, h.Constants["EOF"].Source)
	assert.Equal(t, "8192", h.Constants["BUFSIZ"].Value)
	assert.Equal(t, core.SourceOverlay, h.Constants["BUFSIZ"].Source)
}

func TestMergeOverlay_SkipPrefixDropsFunctions(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Functions["printf"] = &core.CStdlibFunction{FQN: "c::stdio::printf", Source: core.SourceHeader}
	h.Functions["__builtin_printf"] = &core.CStdlibFunction{FQN: "c::stdio::__builtin_printf", Source: core.SourceHeader}
	h.FreeFunctions["__internal_free"] = &core.CStdlibFunction{Source: core.SourceHeader}
	h.Typedefs["__attr_t"] = &core.CStdlibTypedef{Type: "int", Source: core.SourceHeader}
	h.Constants["__BUFSIZ"] = &core.CStdlibConstant{Source: core.SourceHeader}

	o := &Overlay{
		SchemaVersion: "1.0.0", Language: core.LanguageC,
		Skip: []OverlaySkip{{Prefix: "__"}},
	}
	applied := MergeOverlay(h, o)
	assert.Equal(t, 0, applied)
	assert.NotNil(t, h.Functions["printf"])
	assert.Nil(t, h.Functions["__builtin_printf"])
	assert.Nil(t, h.FreeFunctions["__internal_free"])
	assert.Nil(t, h.Typedefs["__attr_t"])
	assert.Nil(t, h.Constants["__BUFSIZ"])
}

func TestMergeOverlay_SkipExactDropsExact(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Functions["printf"] = &core.CStdlibFunction{Source: core.SourceHeader}
	h.Functions["fopen"] = &core.CStdlibFunction{Source: core.SourceHeader}

	o := &Overlay{
		SchemaVersion: "1.0.0", Language: core.LanguageC,
		Skip: []OverlaySkip{{Exact: "fopen"}},
	}
	MergeOverlay(h, o)
	assert.NotNil(t, h.Functions["printf"])
	assert.Nil(t, h.Functions["fopen"])
}

func TestMatchesAnySkip(t *testing.T) {
	skips := []OverlaySkip{{Prefix: "__"}, {Exact: "deprecated_thing"}}
	assert.True(t, matchesAnySkip("__internal", skips))
	assert.True(t, matchesAnySkip("deprecated_thing", skips))
	assert.False(t, matchesAnySkip("public_func", skips))
}

func TestApplySkipRules_Empty(t *testing.T) {
	h := newTestHeader("stdio.h")
	h.Functions["printf"] = &core.CStdlibFunction{Source: core.SourceHeader}
	applySkipRules(h, nil)
	assert.NotNil(t, h.Functions["printf"])
}

func TestParamListFromOverlay(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		assert.Empty(t, paramListFromOverlay(nil))
	})

	t.Run("default required true", func(t *testing.T) {
		got := paramListFromOverlay([]OverlayParam{{Name: "x", Type: "int"}})
		require.Len(t, got, 1)
		assert.True(t, got[0].Required)
	})

	t.Run("explicit required false", func(t *testing.T) {
		f := false
		got := paramListFromOverlay([]OverlayParam{{Name: "x", Type: "int", Required: &f}})
		require.Len(t, got, 1)
		assert.False(t, got[0].Required)
	})
}

func TestBuildFunctionFQN(t *testing.T) {
	h := &core.CStdlibHeader{ModuleID: "c::stdio"}
	assert.Equal(t, "c::stdio::printf", buildFunctionFQN(h, "printf"))

	h = &core.CStdlibHeader{}
	assert.Equal(t, "printf", buildFunctionFQN(h, "printf"))
}

func TestApplyFunctionOverride_NilFunctionsMap(t *testing.T) {
	h := &core.CStdlibHeader{Header: "stdio.h", ModuleID: "c::stdio"}
	applyFunctionOverride(h, OverlayOverride{Header: "stdio.h", Function: "printf", ReturnType: "int"})
	require.NotNil(t, h.Functions)
	assert.NotNil(t, h.Functions["printf"])
}

func TestApplyTypedefOverride_NilTypedefsMap(t *testing.T) {
	h := &core.CStdlibHeader{Header: "stdio.h"}
	applyTypedefOverride(h, OverlayOverride{Header: "stdio.h", Typedef: "FILE", Type: "struct __FILE"})
	require.NotNil(t, h.Typedefs)
	assert.NotNil(t, h.Typedefs["FILE"])
}

func TestApplyConstantOverride_NilConstantsMap(t *testing.T) {
	h := &core.CStdlibHeader{Header: "stdio.h"}
	applyConstantOverride(h, OverlayOverride{Header: "stdio.h", Constant: "EOF", Value: "-1"})
	require.NotNil(t, h.Constants)
	assert.NotNil(t, h.Constants["EOF"])
}

func TestApplyMethodOverride_NilClassesMap(t *testing.T) {
	h := &core.CStdlibHeader{Header: "vector"}
	applyMethodOverride(h, OverlayOverride{
		Header: "vector", Class: "std::vector", Method: "push_back", ReturnType: "void",
	})
	require.NotNil(t, h.Classes)
	assert.NotNil(t, h.Classes["std::vector"])
}

func TestOverrideKind_AllKinds(t *testing.T) {
	tests := []struct {
		name string
		ov   OverlayOverride
		want overrideEntryKind
	}{
		{"function", OverlayOverride{Function: "printf"}, overrideFunction},
		{"method", OverlayOverride{Class: "C", Method: "f"}, overrideMethod},
		{"typedef", OverlayOverride{Typedef: "T"}, overrideTypedef},
		{"constant", OverlayOverride{Constant: "K"}, overrideConstant},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := overrideKind(tt.ov)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyFunctionOverride_PreservesParamsWhenOverlayHasNone(t *testing.T) {
	h := &core.CStdlibHeader{Header: "stdio.h"}
	h.Functions = map[string]*core.CStdlibFunction{
		"printf": {
			FQN:    "c::stdio::printf",
			Params: []*core.CStdlibParam{{Name: "fmt", Type: "const char*"}},
			Source: core.SourceHeader,
		},
	}
	applyFunctionOverride(h, OverlayOverride{Header: "stdio.h", Function: "printf", SecurityTag: "x"})
	got := h.Functions["printf"]
	assert.Len(t, got.Params, 1, "extracted params preserved when overlay supplies none")
	assert.Equal(t, "x", got.SecurityTag)
}

func TestApplyMethodOverride_PreservesParamsWhenOverlayHasNone(t *testing.T) {
	h := core.NewCStdlibHeader()
	h.Header = "vector"
	cls := core.NewCppStdlibClass("std::vector")
	cls.Methods["at"] = &core.CStdlibFunction{
		FQN:    "std::vector::at",
		Params: []*core.CStdlibParam{{Name: "pos", Type: "size_t"}},
		Source: core.SourceHeader,
	}
	h.Classes["std::vector"] = cls

	applyMethodOverride(h, OverlayOverride{
		Header: "vector", Class: "std::vector", Method: "at",
		Throws: "std::out_of_range",
	})
	got := h.Classes["std::vector"].Methods["at"]
	assert.Len(t, got.Params, 1)
	assert.Equal(t, "std::out_of_range", got.Throws)
}
