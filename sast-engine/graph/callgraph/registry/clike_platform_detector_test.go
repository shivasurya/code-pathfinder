package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDetectClikeTarget_DefaultsToLinux(t *testing.T) {
	dir := t.TempDir()
	// Empty project — no signal.
	assert.Equal(t, core.PlatformLinux, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_OverrideWinsOverHeuristic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", "#ifdef __linux__\n#endif\n")
	assert.Equal(t, core.PlatformDarwin, DetectClikeTarget(dir, "darwin"))
	assert.Equal(t, core.PlatformWindows, DetectClikeTarget(dir, "windows"))
}

func TestDetectClikeTarget_OverrideAliases(t *testing.T) {
	dir := t.TempDir()
	for _, alias := range []string{"linux", "Ubuntu", "ALPINE", "  debian  "} {
		assert.Equal(t, core.PlatformLinux, DetectClikeTarget(dir, alias), alias)
	}
	for _, alias := range []string{"windows", "WIN", "msvc", "mingw"} {
		assert.Equal(t, core.PlatformWindows, DetectClikeTarget(dir, alias), alias)
	}
	for _, alias := range []string{"darwin", "macos", "MacOSX", "osx", "apple"} {
		assert.Equal(t, core.PlatformDarwin, DetectClikeTarget(dir, alias), alias)
	}
}

func TestDetectClikeTarget_OverrideUnknownFallsThrough(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", "#ifdef __APPLE__\n#endif\n")
	// Unknown override → use heuristic, which sees __APPLE__.
	assert.Equal(t, core.PlatformDarwin, DetectClikeTarget(dir, "haiku"))
}

func TestDetectClikeTarget_LinuxByMacros(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", `
#ifdef __linux__
#include <linux/specific.h>
#endif
#ifdef __GLIBC__
int x;
#endif
`)
	assert.Equal(t, core.PlatformLinux, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_WindowsByMacros(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", `
#ifdef _WIN32
#include <windows.h>
#endif
#ifdef _WIN64
#include <winnt.h>
#endif
`)
	assert.Equal(t, core.PlatformWindows, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_DarwinByMacros(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.c", `
#ifdef __APPLE__
#include <CoreFoundation/CoreFoundation.h>
#endif
#ifdef __MACH__
int port;
#endif
`)
	assert.Equal(t, core.PlatformDarwin, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_PathHintsBeatSparseMacros(t *testing.T) {
	dir := t.TempDir()
	// Single __linux__ mention — weak macro signal.
	writeFile(t, dir, "main.c", "// __linux__\n")
	// But three windows directories — strong path signal (worth 5 pts each).
	writeFile(t, dir, "win32/a.c", "//\n")
	writeFile(t, dir, "win32/b/c.c", "//\n")
	writeFile(t, dir, "msvc/d.c", "//\n")
	assert.Equal(t, core.PlatformWindows, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_IgnoresVendoredDirs(t *testing.T) {
	dir := t.TempDir()
	// Only signal lives in vendor/ — should be skipped.
	writeFile(t, dir, "vendor/foo/x.c", "#ifdef _WIN32\n#endif\n")
	writeFile(t, dir, "main.c", "// nothing\n")
	assert.Equal(t, core.PlatformLinux, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_IgnoresGitMetadata(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".git/HEAD", "ref: refs/heads/main\n")
	writeFile(t, dir, "main.c", "#ifdef __linux__\n#endif\n")
	assert.Equal(t, core.PlatformLinux, DetectClikeTarget(dir, ""))
}

func TestDetectClikeTarget_NonexistentPathDefaultsToLinux(t *testing.T) {
	// Path that doesn't exist — walker silently fails; we fall through to
	// the default rather than panicking.
	got := DetectClikeTarget("/nonexistent-project-pr02-test", "")
	assert.Equal(t, core.PlatformLinux, got)
}

func TestNormaliseTarget(t *testing.T) {
	assert.Equal(t, "", normaliseTarget(""))
	assert.Equal(t, "", normaliseTarget("freebsd"))
	assert.Equal(t, "", normaliseTarget("solaris"))
	assert.Equal(t, core.PlatformLinux, normaliseTarget("LINUX"))
	assert.Equal(t, core.PlatformLinux, normaliseTarget("  ubuntu  "))
}

func TestIsClikeSourceExt(t *testing.T) {
	for _, ext := range []string{".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".hxx", ".C", ".HPP"} {
		assert.True(t, isClikeSourceExt(ext), ext)
	}
	for _, ext := range []string{"", ".go", ".py", ".java", ".rs"} {
		assert.False(t, isClikeSourceExt(ext), ext)
	}
}

func TestShouldSkipProjectDir(t *testing.T) {
	for _, n := range []string{".git", "node_modules", "vendor", "third_party", "build"} {
		assert.True(t, shouldSkipProjectDir(n), n)
	}
	for _, n := range []string{"src", "include", "linux", "win32", "darwin"} {
		assert.False(t, shouldSkipProjectDir(n), n)
	}
}

func TestReadFileLimited(t *testing.T) {
	dir := t.TempDir()

	// Short file: read fewer bytes than limit.
	short := filepath.Join(dir, "short.txt")
	require.NoError(t, os.WriteFile(short, []byte("abc"), 0o644))
	got, err := readFileLimited(short, 100)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(got))

	// Long file: capped at limit.
	long := filepath.Join(dir, "long.txt")
	require.NoError(t, os.WriteFile(long, []byte("0123456789"), 0o644))
	got, err = readFileLimited(long, 4)
	require.NoError(t, err)
	assert.Equal(t, "0123", string(got))

	// Missing file → error.
	_, err = readFileLimited(filepath.Join(dir, "nope"), 100)
	require.Error(t, err)
}

func TestScanPlatformMacros_FileSizeCap(t *testing.T) {
	dir := t.TempDir()
	// File with the macro AFTER the read cap → not counted.
	pad := make([]byte, platformProbeMaxBytes+1024)
	for i := range pad {
		pad[i] = ' '
	}
	content := string(pad) + "__linux__\n"
	writeFile(t, dir, "huge.c", content)

	counts := scanPlatformMacros(dir)
	assert.Equal(t, 0, counts["__linux__"], "macro past byte cap not counted")
}

func TestScanPathHints_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Linux/x.c", "//\n")
	writeFile(t, dir, "DARWIN/y.c", "//\n")
	writeFile(t, dir, "WiN32/z.c", "//\n")

	hits := scanPathHints(dir)
	assert.Equal(t, 1, hits["linux"])
	assert.Equal(t, 1, hits["darwin"])
	assert.Equal(t, 1, hits["windows"])
}
