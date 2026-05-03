package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeStdlibFixture materialises a minimal C/C++ stdlib registry on
// disk in the layout the file:// loader expects:
//
//	<base>/<platform>/c/v1/manifest.json
//	<base>/<platform>/c/v1/stdio_stdlib.json
//	<base>/<platform>/cpp/v1/manifest.json
//	<base>/<platform>/cpp/v1/vector_stdlib.json
//
// Returns the base directory.
func writeStdlibFixture(t *testing.T, platform string) string {
	t.Helper()
	base := t.TempDir()

	cDir := filepath.Join(base, platform, "c", "v1")
	require.NoError(t, os.MkdirAll(cDir, 0o755))
	stdio := &core.CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "stdio.h",
		ModuleID:      "c::stdio",
		Language:      core.LanguageC,
		Platform:      platform,
		Functions: map[string]*core.CStdlibFunction{
			"printf": {FQN: "c::stdio::printf", ReturnType: "int", Source: core.SourceOverlay, Confidence: 1.0},
		},
	}
	stdioBytes, err := json.Marshal(stdio)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cDir, "stdio_stdlib.json"), stdioBytes, 0o644))

	cManifest := &core.CStdlibManifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		Platform:        platform,
		Language:        core.LanguageC,
		Headers: []*core.CStdlibHeaderEntry{
			{Header: "stdio.h", ModuleID: "c::stdio", File: "stdio_stdlib.json"},
		},
		Statistics: &core.CStdlibStatistics{TotalHeaders: 1, TotalFunctions: 1},
	}
	cManifestBytes, err := json.Marshal(cManifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cDir, "manifest.json"), cManifestBytes, 0o644))

	cppDir := filepath.Join(base, platform, "cpp", "v1")
	require.NoError(t, os.MkdirAll(cppDir, 0o755))
	vec := &core.CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "vector",
		ModuleID:      "std::vector",
		Language:      core.LanguageCpp,
		Classes: map[string]*core.CppStdlibClass{
			"std::vector": {
				FQN: "std::vector", TypeParams: []string{"T"},
				Methods: map[string]*core.CStdlibFunction{
					"push_back": {FQN: "std::vector::push_back", ReturnType: "void", Source: core.SourceOverlay, Confidence: 1.0},
				},
			},
		},
	}
	vecBytes, err := json.Marshal(vec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cppDir, "vector_stdlib.json"), vecBytes, 0o644))

	cppManifest := &core.CStdlibManifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		Platform:        platform,
		Language:        core.LanguageCpp,
		Headers: []*core.CStdlibHeaderEntry{
			{Header: "vector", ModuleID: "std::vector", File: "vector_stdlib.json"},
		},
		Statistics: &core.CStdlibStatistics{TotalHeaders: 1, TotalClasses: 1, TotalFunctions: 1},
	}
	cppManifestBytes, err := json.Marshal(cppManifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cppDir, "manifest.json"), cppManifestBytes, 0o644))

	return base
}

func TestInitClikeStdlib_DisabledWhenBaseURLEmpty(t *testing.T) {
	cfg, wired := initClikeStdlib(t.TempDir(), "linux", "", newTestLogger())
	assert.False(t, wired, "empty base URL must leave loaders unwired")
	assert.Nil(t, cfg.cLoader)
	assert.Nil(t, cfg.cppLoader)
	assert.Equal(t, "linux", cfg.platform)
}

func TestInitClikeStdlib_FileSchemeWiresBothLoaders(t *testing.T) {
	base := writeStdlibFixture(t, "linux")
	cfg, wired := initClikeStdlib(t.TempDir(), "linux", "file://"+base, newTestLogger())
	require.True(t, wired)
	require.NotNil(t, cfg.cLoader)
	require.NotNil(t, cfg.cppLoader)
	assert.Equal(t, 1, cfg.cLoader.HeaderCount())
	assert.Equal(t, 1, cfg.cppLoader.HeaderCount())
}

func TestInitClikeStdlib_BarePathTreatedAsFile(t *testing.T) {
	base := writeStdlibFixture(t, "linux")
	cfg, wired := initClikeStdlib(t.TempDir(), "linux", base, newTestLogger())
	require.True(t, wired)
	require.NotNil(t, cfg.cLoader)
	require.NotNil(t, cfg.cppLoader)
}

func TestInitClikeStdlib_HTTPSchemeFailsGracefullyOnUnreachableHost(t *testing.T) {
	// PR-03 wires HTTP up — when the URL doesn't resolve and there's no
	// disk cache to fall back on, both loaders fail to load and stay
	// nil. The scan continues under Phase 1 behavior.
	//
	// Point the cache at a fresh temp dir so a previously populated
	// developer cache (e.g. from running another test) cannot serve a
	// stale manifest and turn this into a false-positive success.
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())

	cfg, wired := initClikeStdlib(t.TempDir(), "linux", "http://127.0.0.1:1/registries", newTestLogger())
	assert.False(t, wired)
	assert.Nil(t, cfg.cLoader)
	assert.Nil(t, cfg.cppLoader)
}

func TestInitClikeStdlib_MissingFixtureDegradesGracefully(t *testing.T) {
	// Point at a directory that contains no manifest. The loader is
	// constructed but LoadManifest fails — the function must return a
	// usable (empty) config rather than panic.
	cfg, wired := initClikeStdlib(t.TempDir(), "linux", "file://"+t.TempDir(), newTestLogger())
	assert.False(t, wired)
	assert.Nil(t, cfg.cLoader)
	assert.Nil(t, cfg.cppLoader)
	assert.Equal(t, "linux", cfg.platform)
}

func TestBuildCStdlibLoader_SchemeDispatch(t *testing.T) {
	t.Run("file scheme", func(t *testing.T) {
		require.NotNil(t, buildCStdlibLoader("file:///tmp/foo", "linux"))
	})
	t.Run("https scheme", func(t *testing.T) {
		require.NotNil(t, buildCStdlibLoader("https://example.test/", "linux"))
	})
	t.Run("http scheme", func(t *testing.T) {
		require.NotNil(t, buildCStdlibLoader("http://example.test/", "linux"))
	})
	t.Run("bare path", func(t *testing.T) {
		require.NotNil(t, buildCStdlibLoader("/var/cache/pf", "linux"))
	})
}

func TestBuildCppStdlibLoader_SchemeDispatch(t *testing.T) {
	t.Run("file scheme", func(t *testing.T) {
		require.NotNil(t, buildCppStdlibLoader("file:///tmp/foo", "linux"))
	})
	t.Run("https scheme", func(t *testing.T) {
		require.NotNil(t, buildCppStdlibLoader("https://example.test/", "linux"))
	})
	t.Run("http scheme", func(t *testing.T) {
		require.NotNil(t, buildCppStdlibLoader("http://example.test/", "linux"))
	})
	t.Run("bare path", func(t *testing.T) {
		require.NotNil(t, buildCppStdlibLoader("/var/cache/pf", "linux"))
	})
}

func TestStdlibLoggerAdapter_ForwardsToLogger(t *testing.T) {
	// Smoke-level: each forward simply must not panic with a real logger.
	a := stdlibLoggerAdapter{logger: newTestLogger()}
	a.Debug("debug %s", "msg")
	a.Statistic("stat %d", 42)
	a.Warning("warn %s", "msg")

	// Nil logger must also be tolerated — the adapter is the only call
	// path for the loader, so a misconfigured caller must degrade
	// silently rather than panic.
	nilA := stdlibLoggerAdapter{logger: nil}
	nilA.Debug("x")
	nilA.Statistic("x")
	nilA.Warning("x")
}

func TestBuildClikeCallGraphs_StdlibConfigPropagates(t *testing.T) {
	root := "/projects/app"
	codeGraph := graph.NewCodeGraph()
	codeGraph.AddNode(&graph.Node{
		ID:         "fn:src/main.c::main",
		Type:       "function_definition",
		Name:       "main",
		File:       root + "/src/main.c",
		Language:   "c",
		ReturnType: "int",
	})

	base := writeStdlibFixture(t, "linux")
	cLoader := registry.NewCStdlibRegistryFile(filepath.Join(base, "linux", "c", "v1"), "linux")
	require.NoError(t, cLoader.LoadManifest(stdlibLoggerAdapter{logger: newTestLogger()}))

	cfg := clikeStdlibConfig{cLoader: cLoader, platform: "linux"}
	cg := core.NewCallGraph()
	buildClikeCallGraphs(cg, codeGraph, root, newTestLogger(), cfg)

	// The call graph must still be populated; loader presence cannot
	// regress Phase 1 behavior.
	assert.Contains(t, cg.Functions, "src/main.c::main")
}
