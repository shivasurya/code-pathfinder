package clikeextract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestHeader(name string) *core.CStdlibHeader {
	h := core.NewCStdlibHeader()
	h.SchemaVersion = SchemaVersion
	h.Header = name
	h.ModuleID = "c::" + SanitizeHeaderName(name)
	h.Language = core.LanguageC
	h.Platform = core.PlatformLinux
	h.SystemTag = "glibc-test"
	h.Functions["printf"] = &core.CStdlibFunction{
		FQN:        "c::stdio::printf",
		ReturnType: "int",
		Source:     core.SourceHeader,
	}
	h.Typedefs["FILE"] = &core.CStdlibTypedef{Type: "struct __FILE", Source: core.SourceHeader}
	h.Constants["EOF"] = &core.CStdlibConstant{Type: "int", Value: "-1", Source: core.SourceHeader}
	return h
}

func TestEmitOutput_ProducesManifestAndPerHeaderFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: dir,
	}
	headers := []*core.CStdlibHeader{
		makeTestHeader("stdio.h"),
		makeTestHeader("string.h"),
	}

	require.NoError(t, EmitOutput(headers, cfg, 7))

	// Files exist.
	assert.FileExists(t, filepath.Join(dir, "manifest.json"))
	assert.FileExists(t, filepath.Join(dir, "stdio_stdlib.json"))
	assert.FileExists(t, filepath.Join(dir, "string_stdlib.json"))

	// Manifest parses and has expected fields.
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	var got core.CStdlibManifest
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, SchemaVersion, got.SchemaVersion)
	assert.Equal(t, RegistryVersion, got.RegistryVersion)
	assert.Equal(t, core.PlatformLinux, got.Platform)
	assert.Equal(t, core.LanguageC, got.Language)
	assert.Equal(t, "glibc-test", got.SystemTag)
	assert.Equal(t, GeneratorVersion, got.GeneratorVersion)

	require.Len(t, got.Headers, 2)
	// Sorted alphabetically.
	assert.Equal(t, "stdio.h", got.Headers[0].Header)
	assert.Equal(t, "string.h", got.Headers[1].Header)
	assert.Equal(t, "stdio_stdlib.json", got.Headers[0].File)

	// Statistics correct.
	require.NotNil(t, got.Statistics)
	assert.Equal(t, 2, got.Statistics.TotalHeaders)
	assert.Equal(t, 2, got.Statistics.TotalFunctions)
	assert.Equal(t, 2, got.Statistics.TotalTypedefs)
	assert.Equal(t, 2, got.Statistics.TotalConstants)
	assert.Equal(t, 7, got.Statistics.OverlayOverrides)

	// Checksum is well-formed and matches the file we wrote.
	for _, e := range got.Headers {
		assert.True(t, len(e.Checksum) > len("sha256:"))
		fileBytes, err := os.ReadFile(filepath.Join(dir, e.File))
		require.NoError(t, err)
		want := "sha256:" + hex.EncodeToString(sha256SliceToHexed(sha256.Sum256(fileBytes)))
		assert.Equal(t, want, e.Checksum)
		assert.Equal(t, int64(len(fileBytes)), e.Size)
	}
}

func sha256SliceToHexed(sum [32]byte) []byte {
	return sum[:]
}

func TestEmitOutput_DefaultBaseURL(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: dir,
	}
	require.NoError(t, EmitOutput([]*core.CStdlibHeader{makeTestHeader("stdio.h")}, cfg, 0))

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	var m core.CStdlibManifest
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "https://assets.codepathfinder.dev/registries/linux/c/v1", m.BaseURL)
	require.Len(t, m.Headers, 1)
	assert.Equal(t, "https://assets.codepathfinder.dev/registries/linux/c/v1/stdio_stdlib.json",
		m.Headers[0].URL)
}

func TestEmitOutput_OverrideBaseURL(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: dir,
		BaseURL:   "file:///tmp/registries",
	}
	require.NoError(t, EmitOutput([]*core.CStdlibHeader{makeTestHeader("stdio.h")}, cfg, 0))

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	var m core.CStdlibManifest
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "file:///tmp/registries/linux/c/v1", m.BaseURL)
	assert.Equal(t, "file:///tmp/registries/linux/c/v1/stdio_stdlib.json", m.Headers[0].URL)
}

func TestEmitOutput_DeterministicAcrossRuns(t *testing.T) {
	headers := []*core.CStdlibHeader{
		makeTestHeader("string.h"),
		makeTestHeader("stdio.h"),
	}
	// Set GeneratedAt explicitly to avoid the time-based fields differing.
	for _, h := range headers {
		h.GeneratedAt = "2026-05-04T00:00:00Z"
	}

	dir1 := t.TempDir()
	dir2 := t.TempDir()
	cfg1 := Config{Target: core.PlatformLinux, Language: core.LanguageC, OutputDir: dir1}
	cfg2 := Config{Target: core.PlatformLinux, Language: core.LanguageC, OutputDir: dir2}

	require.NoError(t, EmitOutput(headers, cfg1, 0))
	// Re-create the headers to avoid in-place mutations from the first run.
	for _, h := range headers {
		h.GeneratedAt = "2026-05-04T00:00:00Z"
	}
	require.NoError(t, EmitOutput(headers, cfg2, 0))

	for _, name := range []string{"stdio_stdlib.json", "string_stdlib.json"} {
		a, _ := os.ReadFile(filepath.Join(dir1, name))
		b, _ := os.ReadFile(filepath.Join(dir2, name))
		assert.Equal(t, a, b, "per-header file %q should be byte-identical across runs", name)
	}
}

func TestEmitOutput_OutputDirCreated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	cfg := Config{Target: core.PlatformLinux, Language: core.LanguageC, OutputDir: dir}
	require.NoError(t, EmitOutput([]*core.CStdlibHeader{makeTestHeader("stdio.h")}, cfg, 0))
	assert.FileExists(t, filepath.Join(dir, "manifest.json"))
}

func TestEmitOutput_WriteFails_OutputUnderFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: file masquerading as dir does not block writes")
	}
	tmp := t.TempDir()
	clash := filepath.Join(tmp, "clash")
	require.NoError(t, os.WriteFile(clash, []byte("plain file"), 0o644))

	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: clash, // a file, not a dir → MkdirAll fails
	}
	err := EmitOutput([]*core.CStdlibHeader{makeTestHeader("stdio.h")}, cfg, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating output dir")
}

func TestComputeStatistics_WithCppContent(t *testing.T) {
	cppHeader := core.NewCStdlibHeader()
	cppHeader.Header = "vector"
	cls := core.NewCppStdlibClass("std::vector")
	cls.Methods["push_back"] = &core.CStdlibFunction{}
	cls.Methods["size"] = &core.CStdlibFunction{}
	cppHeader.Classes["std::vector"] = cls
	cppHeader.FreeFunctions["std::swap"] = &core.CStdlibFunction{}

	stats := computeStatistics([]*core.CStdlibHeader{cppHeader}, 0)
	assert.Equal(t, 1, stats.TotalHeaders)
	assert.Equal(t, 1, stats.TotalClasses)
	// 0 free + 1 namespaced free + 2 methods = 3.
	assert.Equal(t, 3, stats.TotalFunctions)
}

func TestFirstSystemTag(t *testing.T) {
	assert.Equal(t, "", firstSystemTag(nil))
	assert.Equal(t, "", firstSystemTag([]*core.CStdlibHeader{}))

	h := makeTestHeader("stdio.h")
	assert.Equal(t, "glibc-test", firstSystemTag([]*core.CStdlibHeader{h}))
}

func TestBuildHeaderURL(t *testing.T) {
	cfg := Config{
		Target:   core.PlatformLinux,
		Language: core.LanguageC,
		BaseURL:  "https://example.com/registries",
	}
	got := buildHeaderURL(cfg, "stdio_stdlib.json")
	assert.Equal(t, "https://example.com/registries/linux/c/v1/stdio_stdlib.json", got)
}
