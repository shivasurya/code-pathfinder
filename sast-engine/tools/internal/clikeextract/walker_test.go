package clikeextract

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverHeaderSources_LinuxC(t *testing.T) {
	got, err := DiscoverHeaderSources(core.PlatformLinux, core.LanguageC)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, core.PlatformLinux, got[0].Platform)
	assert.Equal(t, core.LanguageC, got[0].Language)
	assert.Equal(t, []string{".h"}, got[0].HeaderExts)
	assert.Contains(t, got[0].SystemTag, "glibc-")
}

func TestDiscoverHeaderSources_LinuxCpp_Found(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "13"), 0o755))
	withCppRoot(t, root)

	got, err := DiscoverHeaderSources(core.PlatformLinux, core.LanguageCpp)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, core.LanguageCpp, got[0].Language)
	assert.Contains(t, got[0].HeaderExts, "")
}

func TestDiscoverHeaderSources_LinuxCpp_NotInstalled(t *testing.T) {
	withCppRoot(t, filepath.Join(t.TempDir(), "no-libstdcpp"))
	got, err := DiscoverHeaderSources(core.PlatformLinux, core.LanguageCpp)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "libstdc++")
}

// TestDiscoverHeaderSources_CrossPlatformHeadersMissing verifies that when
// the cross-platform toolchains aren't installed on the host, each target
// surfaces a remediation hint instead of crashing. PR-03 implements the
// dispatch; the host-installation case is covered by walker_xplat_test.go.
func TestDiscoverHeaderSources_CrossPlatformHeadersMissing(t *testing.T) {
	withTempMingwRoot(t, "/definitely/missing")
	withTempDarwinRoots(t, []string{"/missing/sdk"}, []string{"/missing/cpp"})

	cases := []struct {
		platform, language, fragment string
	}{
		{core.PlatformWindows, core.LanguageC, "mingw-w64"},
		{core.PlatformWindows, core.LanguageCpp, "mingw libstdc++"},
		{core.PlatformDarwin, core.LanguageC, "macOS SDK"},
		{core.PlatformDarwin, core.LanguageCpp, "libc++"},
	}
	for _, tt := range cases {
		_, err := DiscoverHeaderSources(tt.platform, tt.language)
		require.Error(t, err)
		assert.Contains(t, err.Error(), tt.fragment)
	}
}

func TestDiscoverHeaderSources_UnknownCombination(t *testing.T) {
	_, err := DiscoverHeaderSources("freebsd", "rust")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown target+language")
}

func TestWalkHeaders_FlatDir(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "stdio.h"), "// stdio")
	mustWriteFile(t, filepath.Join(dir, "string.h"), "// string")
	mustWriteFile(t, filepath.Join(dir, "ignore.txt"), "should be filtered out")

	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".h"}}
	got, err := src.WalkHeaders()
	require.NoError(t, err)
	require.Len(t, got, 2)

	names := []string{got[0].Name, got[1].Name}
	sort.Strings(names)
	assert.Equal(t, []string{"stdio.h", "string.h"}, names)
}

func TestWalkHeaders_SubdirNamePreserved(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sys"), 0o755))
	mustWriteFile(t, filepath.Join(dir, "sys", "socket.h"), "// socket")

	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".h"}}
	got, err := src.WalkHeaders()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "sys/socket.h", got[0].Name)
}

func TestWalkHeaders_SkipsPrivateSubdirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "bits"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "internal"), 0o755))
	mustWriteFile(t, filepath.Join(dir, "stdio.h"), "// public")
	mustWriteFile(t, filepath.Join(dir, "bits", "types.h"), "// private")
	mustWriteFile(t, filepath.Join(dir, "internal", "x.h"), "// private")

	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".h"}}
	got, err := src.WalkHeaders()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "stdio.h", got[0].Name)
}

func TestWalkHeaders_ExtensionlessCpp(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "vector"), "// vector")
	mustWriteFile(t, filepath.Join(dir, "string"), "// string")
	mustWriteFile(t, filepath.Join(dir, "memory.hpp"), "// memory")
	mustWriteFile(t, filepath.Join(dir, "ignore.txt"), "// ignore")

	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".hpp", ""}}
	got, err := src.WalkHeaders()
	require.NoError(t, err)
	require.Len(t, got, 3)
}

func TestWalkHeaders_NoSearchDirs(t *testing.T) {
	src := HeaderSource{HeaderExts: []string{".h"}}
	_, err := src.WalkHeaders()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SearchDirs")
}

func TestWalkHeaders_MissingSearchDir(t *testing.T) {
	src := HeaderSource{
		SearchDirs: []string{"/nonexistent-dir-c8a2-pr01-test"},
		HeaderExts: []string{".h"},
	}
	_, err := src.WalkHeaders()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestWalkHeaders_DedupAcrossSearchDirs(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	mustWriteFile(t, filepath.Join(dirA, "stdio.h"), "// A")
	mustWriteFile(t, filepath.Join(dirB, "stdio.h"), "// B (overrides — but dedup keeps A)")

	src := HeaderSource{SearchDirs: []string{dirA, dirB}, HeaderExts: []string{".h"}}
	got, err := src.WalkHeaders()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, filepath.Join(dirA, "stdio.h"), got[0].Path,
		"first SearchDir should win on duplicate header names")
}

func TestWalkHeaders_PermissionDeniedSubdir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 0 does not block reads")
	}
	dir := t.TempDir()
	bad := filepath.Join(dir, "locked")
	require.NoError(t, os.MkdirAll(bad, 0o755))
	mustWriteFile(t, filepath.Join(bad, "leaf.h"), "//")
	require.NoError(t, os.Chmod(bad, 0o000))
	t.Cleanup(func() { _ = os.Chmod(bad, 0o755) })

	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".h"}}
	_, err := src.WalkHeaders()
	require.Error(t, err)
}

func TestWalkHeaders_DeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"zebra.h", "alpha.h", "mango.h"} {
		mustWriteFile(t, filepath.Join(dir, name), "//")
	}
	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".h"}}

	for range 3 {
		got, err := src.WalkHeaders()
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "alpha.h", got[0].Name)
		assert.Equal(t, "mango.h", got[1].Name)
		assert.Equal(t, "zebra.h", got[2].Name)
	}
}

func TestHeaderName(t *testing.T) {
	dir := t.TempDir()
	src := HeaderSource{SearchDirs: []string{dir}, HeaderExts: []string{".h"}}

	abs := filepath.Join(dir, "sys", "socket.h")
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	mustWriteFile(t, abs, "//")

	assert.Equal(t, "sys/socket.h", src.HeaderName(abs))
	assert.Equal(t, "", src.HeaderName("/etc/passwd"), "outside-tree path → empty")
}

func TestShouldSkipDir(t *testing.T) {
	skips := []string{"bits", "internal", "__support", "ext", "experimental", "tr1", "tr2", "parallel", "pstl", "debug", "profile", "decimal"}
	for _, s := range skips {
		assert.True(t, shouldSkipDir(s), s)
	}
	keeps := []string{"sys", "linux", "asm", "net"}
	for _, s := range keeps {
		assert.False(t, shouldSkipDir(s), s)
	}
}

func TestIsHostLinux(t *testing.T) {
	got := IsHostLinux()
	// We don't know the host, but the function must not panic and must return a bool.
	assert.IsType(t, true, got)
}

func TestContainsDigit(t *testing.T) {
	assert.True(t, containsDigit("13"))
	assert.True(t, containsDigit("13.2.0"))
	assert.True(t, containsDigit("c++23"))
	assert.False(t, containsDigit("experimental"))
	assert.False(t, containsDigit(""))
}

func TestDirExists(t *testing.T) {
	d := t.TempDir()
	assert.True(t, dirExists(d))
	assert.False(t, dirExists(filepath.Join(d, "nope")))

	// File, not dir
	f := filepath.Join(d, "file.txt")
	mustWriteFile(t, f, "x")
	assert.False(t, dirExists(f))
}

func TestDetectGlibcTag_HitFirstCandidate(t *testing.T) {
	probe := t.TempDir()
	libDir := filepath.Join(probe, "x86_64-linux-gnu")
	require.NoError(t, os.MkdirAll(libDir, 0o755))

	withRoots(t, []string{libDir, "/this/does/not/exist"})
	assert.Equal(t, "glibc-x86_64-linux-gnu", detectGlibcTag())
}

func TestDetectGlibcTag_FallbackUnknown(t *testing.T) {
	withRoots(t, []string{"/no-libc-1", "/no-libc-2"})
	assert.Equal(t, "glibc-unknown", detectGlibcTag())
}

func TestFindLibstdcppRoot_PicksLargestVersion(t *testing.T) {
	root := t.TempDir()
	for _, v := range []string{"11", "13", "12.2", "experimental"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, v), 0o755))
	}
	withCppRoot(t, root)

	dir, version := findLibstdcppRoot()
	assert.Equal(t, "13", version)
	assert.Equal(t, filepath.Join(root, "13"), dir)
}

func TestFindLibstdcppRoot_NoVersions(t *testing.T) {
	root := t.TempDir()
	// Only non-versioned entries, plus a stray file.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "experimental"), 0o755))
	mustWriteFile(t, filepath.Join(root, "stray-file"), "")
	withCppRoot(t, root)

	dir, version := findLibstdcppRoot()
	assert.Equal(t, "", dir)
	assert.Equal(t, "", version)
}

func TestFindLibstdcppRoot_RootMissing(t *testing.T) {
	withCppRoot(t, filepath.Join(t.TempDir(), "missing-cpp-root"))
	dir, version := findLibstdcppRoot()
	assert.Equal(t, "", dir)
	assert.Equal(t, "", version)
}

func TestLinuxCppSource_Found(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "13"), 0o755))
	withCppRoot(t, root)

	src, err := linuxCppSource()
	require.NoError(t, err)
	assert.Equal(t, core.LanguageCpp, src.Language)
	assert.Equal(t, "libstdc++-13", src.SystemTag)
	assert.Contains(t, src.HeaderExts, "")
}

func TestLinuxCppSource_NotFound(t *testing.T) {
	withCppRoot(t, filepath.Join(t.TempDir(), "no-cpp"))
	_, err := linuxCppSource()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no libstdc++")
}

// withRoots overrides linuxLibcRoots for the duration of the test and restores
// it via t.Cleanup so other tests see the original list.
func withRoots(t *testing.T, roots []string) {
	t.Helper()
	original := linuxLibcRoots
	linuxLibcRoots = roots
	t.Cleanup(func() { linuxLibcRoots = original })
}

func withCppRoot(t *testing.T, root string) {
	t.Helper()
	original := linuxCppRoot
	linuxCppRoot = root
	t.Cleanup(func() { linuxCppRoot = original })
}

// mustWriteFile is a test helper that writes a small file and fails the test on error.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
