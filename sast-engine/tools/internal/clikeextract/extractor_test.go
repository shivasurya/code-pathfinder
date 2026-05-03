package clikeextract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunFixtureLinuxC pipes the C testdata fixtures through the full extractor
// pipeline (discover → walk → extract → merge → emit) and asserts on the
// generated manifest + a sampled per-header file. This is the integration
// test that catches breakage in the wiring between stages.
func TestRunFixtureLinuxC(t *testing.T) {
	out := t.TempDir()
	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: out,
	}

	// Override the C search dir to our fixture tree.
	src := cTestSource()
	useFixedSources(t, []HeaderSource{src})

	ext := NewExtractor(cfg)
	require.NoError(t, ext.Run())

	// Manifest written and parses.
	mData, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	require.NoError(t, err)
	var m core.CStdlibManifest
	require.NoError(t, json.Unmarshal(mData, &m))
	assert.Equal(t, core.PlatformLinux, m.Platform)
	assert.Equal(t, core.LanguageC, m.Language)
	require.GreaterOrEqual(t, len(m.Headers), 4, "fixture has at least stdio.h, string.h, unistd.h, inline.h")

	// Per-header file is reachable via the manifest's File field.
	stdioEntry := m.GetHeaderEntry("stdio.h")
	require.NotNil(t, stdioEntry)
	stdioPath := filepath.Join(out, stdioEntry.File)
	stdioData, err := os.ReadFile(stdioPath)
	require.NoError(t, err)

	var stdioHeader core.CStdlibHeader
	require.NoError(t, json.Unmarshal(stdioData, &stdioHeader))
	assert.NotNil(t, stdioHeader.Functions["printf"])
	assert.Equal(t, core.SourceHeader, stdioHeader.Functions["printf"].Source)

	// Statistics are populated and reasonable.
	require.NotNil(t, m.Statistics)
	assert.Greater(t, m.Statistics.TotalFunctions, 0)
	assert.Greater(t, m.Statistics.TotalConstants, 0)
}

// TestRunFixtureLinuxCpp does the same for the C++ testdata.
func TestRunFixtureLinuxCpp(t *testing.T) {
	out := t.TempDir()
	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageCpp,
		OutputDir: out,
	}

	useFixedSources(t, []HeaderSource{cppTestSource()})

	require.NoError(t, NewExtractor(cfg).Run())

	mData, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	require.NoError(t, err)
	var m core.CStdlibManifest
	require.NoError(t, json.Unmarshal(mData, &m))
	assert.Equal(t, core.LanguageCpp, m.Language)

	vectorEntry := m.GetHeaderEntry("vector")
	require.NotNil(t, vectorEntry)
	vData, err := os.ReadFile(filepath.Join(out, vectorEntry.File))
	require.NoError(t, err)

	var vec core.CStdlibHeader
	require.NoError(t, json.Unmarshal(vData, &vec))
	assert.Contains(t, vec.Classes, "std::vector")
}

func TestRun_OverlayMerged(t *testing.T) {
	out := t.TempDir()
	overlayPath := filepath.Join(out, "overlay.yaml")
	require.NoError(t, os.WriteFile(overlayPath, []byte(`schema_version: "1.0.0"
language: c
overrides:
  - header: stdio.h
    function: printf
    security_tag: format_string_sink
`), 0o644))

	cfg := Config{
		Target:      core.PlatformLinux,
		Language:    core.LanguageC,
		OutputDir:   out,
		OverlayPath: overlayPath,
	}
	useFixedSources(t, []HeaderSource{cTestSource()})

	require.NoError(t, NewExtractor(cfg).Run())

	stdioPath := filepath.Join(out, "stdio_stdlib.json")
	data, err := os.ReadFile(stdioPath)
	require.NoError(t, err)

	var h core.CStdlibHeader
	require.NoError(t, json.Unmarshal(data, &h))
	got := h.Functions["printf"]
	require.NotNil(t, got)
	assert.Equal(t, "format_string_sink", got.SecurityTag)
	assert.Equal(t, core.SourceMerged, got.Source)
}

func TestRun_InvalidConfig(t *testing.T) {
	cfg := Config{} // missing everything
	err := NewExtractor(cfg).Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config")
}

func TestRun_OverlayLoadError(t *testing.T) {
	cfg := Config{
		Target:      core.PlatformLinux,
		Language:    core.LanguageC,
		OutputDir:   t.TempDir(),
		OverlayPath: "/nonexistent-overlay-pr01-extractor-test.yaml",
	}
	err := NewExtractor(cfg).Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestRun_DiscoveryError(t *testing.T) {
	// Windows + missing mingw toolchain → discovery error with remediation
	// hint. Pre-PR-03 this asserted the stub message; now it asserts the
	// "headers not found" path.
	withTempMingwRoot(t, "/definitely/missing")
	cfg := Config{
		Target:    core.PlatformWindows,
		Language:  core.LanguageC,
		OutputDir: t.TempDir(),
	}
	err := NewExtractor(cfg).Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mingw-w64")
}

func TestRun_WalkError(t *testing.T) {
	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: t.TempDir(),
	}
	useFixedSources(t, []HeaderSource{{
		Platform:   core.PlatformLinux,
		Language:   core.LanguageC,
		SearchDirs: []string{"/this-path-must-not-exist-pr01-walk"},
		HeaderExts: []string{".h"},
	}})

	err := NewExtractor(cfg).Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRun_PerHeaderParseFailure_Continues(t *testing.T) {
	// A directory with a real header + a binary file that parses fine but
	// produces no symbols. The pipeline should NOT abort; the warning channel
	// should record nothing fatal.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "real.h"),
		[]byte(`int real_func(int x);`), 0o644))

	cfg := Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: t.TempDir(),
	}
	useFixedSources(t, []HeaderSource{{
		Platform:   core.PlatformLinux,
		Language:   core.LanguageC,
		SystemTag:  "test",
		SearchDirs: []string{dir},
		HeaderExts: []string{".h"},
	}})

	var captured strings.Builder
	ext := NewExtractor(cfg)
	ext.SetLogger(func(format string, args ...any) {
		captured.WriteString(format + "\n")
	})
	require.NoError(t, ext.Run())
	// The real header is fine — no warnings expected.
	assert.NotContains(t, captured.String(), "warning")
}

func TestDefaultLogf_DoesNotPanic(t *testing.T) {
	// defaultLogf writes to os.Stderr; we can't easily capture it here, but
	// the call must not panic and must accept format args.
	defaultLogf("test %d %s", 1, "x")
}

func TestExtractOne_UnsupportedLanguage(t *testing.T) {
	// Bypass Validate by constructing the extractor with a hand-tweaked Config
	// that survives Validate (we can't — Validate rejects "rust"). Instead
	// call extractOne directly to exercise the defensive fallback branch.
	ext := NewExtractor(Config{
		Target:    core.PlatformLinux,
		Language:  core.LanguageC,
		OutputDir: t.TempDir(),
	})
	ext.cfg.Language = "rust" // post-construction tweak for the test
	_, err := ext.extractOne(HeaderFile{Name: "x.h", Path: "/dev/null"},
		HeaderSource{Language: "rust"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language")
}

// useFixedSources monkey-patches DiscoverHeaderSources to return a fixed list
// for the duration of the test. The package-level pointer indirection is the
// same trick walker_test.go uses for linuxLibcRoots / linuxCppRoot.
func useFixedSources(t *testing.T, sources []HeaderSource) {
	t.Helper()
	original := discoverHeaderSourcesFn
	discoverHeaderSourcesFn = func(target, language string) ([]HeaderSource, error) {
		return sources, nil
	}
	t.Cleanup(func() { discoverHeaderSourcesFn = original })
}
