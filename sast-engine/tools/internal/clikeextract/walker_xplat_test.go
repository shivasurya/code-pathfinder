package clikeextract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTempMingwRoot wires a temporary directory in as the mingw root for
// the duration of t. Restores the original on cleanup so tests are
// independent of order.
func withTempMingwRoot(t *testing.T, root string) {
	t.Helper()
	origMingw := windowsMingwRoot
	origUbuntu := ubuntuMingwGccRoot
	windowsMingwRoot = root
	// Point the Ubuntu probe at a fresh empty dir so tests on a host with
	// real mingw installed don't see Ubuntu's libstdc++ tree as a
	// "phantom" fallback.
	ubuntuMingwGccRoot = filepath.Join(t.TempDir(), "ubuntu-mingw-absent")
	t.Cleanup(func() {
		windowsMingwRoot = origMingw
		ubuntuMingwGccRoot = origUbuntu
	})
}

// withTempUbuntuMingwRoot points the Ubuntu mingw probe at root for the
// duration of t. Used by tests that exercise the Ubuntu layout
// specifically — independent of withTempMingwRoot so callers can mix and
// match upstream / Ubuntu probe state.
func withTempUbuntuMingwRoot(t *testing.T, root string) {
	t.Helper()
	orig := ubuntuMingwGccRoot
	ubuntuMingwGccRoot = root
	t.Cleanup(func() { ubuntuMingwGccRoot = orig })
}

// withTempDarwinRoots replaces both the C and C++ Darwin probe lists for t.
func withTempDarwinRoots(t *testing.T, sdkRoots, cppRoots []string) {
	t.Helper()
	origC := darwinSDKRoots
	origCpp := darwinCppRoots
	darwinSDKRoots = sdkRoots
	darwinCppRoots = cppRoots
	t.Cleanup(func() {
		darwinSDKRoots = origC
		darwinCppRoots = origCpp
	})
}

func TestWindowsCSource_Found(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include", "c++", "13"), 0o755))
	withTempMingwRoot(t, root)

	src, err := windowsCSource()
	require.NoError(t, err)
	assert.Equal(t, core.PlatformWindows, src.Platform)
	assert.Equal(t, core.LanguageC, src.Language)
	assert.Equal(t, []string{filepath.Join(root, "include")}, src.SearchDirs)
	assert.Equal(t, "mingw-w64-13", src.SystemTag)
}

func TestWindowsCSource_Missing(t *testing.T) {
	withTempMingwRoot(t, filepath.Join(t.TempDir(), "absent"))
	_, err := windowsCSource()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mingw-w64 headers not found")
}

func TestWindowsCSource_VersionUnknownWhenCppTreeMissing(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include"), 0o755))
	withTempMingwRoot(t, root)

	src, err := windowsCSource()
	require.NoError(t, err)
	assert.Equal(t, "mingw-w64-unknown", src.SystemTag)
}

func TestWindowsCppSource_Found_UpstreamLayout(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include", "c++", "13"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include", "c++", "12"), 0o755))
	withTempMingwRoot(t, root)

	src, err := windowsCppSource()
	require.NoError(t, err)
	assert.Equal(t, core.PlatformWindows, src.Platform)
	assert.Equal(t, core.LanguageCpp, src.Language)
	assert.Equal(t, []string{filepath.Join(root, "include", "c++", "13")}, src.SearchDirs,
		"freshest version directory must win")
	assert.Equal(t, "mingw-w64-libstdc++-13-upstream", src.SystemTag)
}

// TestWindowsCppSource_Found_UbuntuLayout pins the Debian/Ubuntu
// packaging shape where libstdc++ lives under
// /usr/lib/gcc/x86_64-w64-mingw32/<ver>-<thread>/include/c++.
// The posix-threading variant must win when both are present.
func TestWindowsCppSource_Found_UbuntuLayout(t *testing.T) {
	// Upstream layout absent.
	withTempMingwRoot(t, filepath.Join(t.TempDir(), "no-upstream"))

	ubuntu := t.TempDir()
	for _, ver := range []string{"13-posix", "13-win32"} {
		require.NoError(t, os.MkdirAll(filepath.Join(ubuntu, ver, "include", "c++"), 0o755))
	}
	withTempUbuntuMingwRoot(t, ubuntu)

	src, err := windowsCppSource()
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(ubuntu, "13-posix", "include", "c++")}, src.SearchDirs,
		"posix threading variant must win over win32")
	assert.Equal(t, "mingw-w64-libstdc++-13-posix-ubuntu", src.SystemTag)
}

// TestWindowsCppSource_Found_UbuntuLayout_Win32Fallback covers the case
// where only the win32 threading variant is installed.
func TestWindowsCppSource_Found_UbuntuLayout_Win32Fallback(t *testing.T) {
	withTempMingwRoot(t, filepath.Join(t.TempDir(), "no-upstream"))

	ubuntu := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(ubuntu, "13-win32", "include", "c++"), 0o755))
	withTempUbuntuMingwRoot(t, ubuntu)

	src, err := windowsCppSource()
	require.NoError(t, err)
	assert.Equal(t, "mingw-w64-libstdc++-13-win32-ubuntu", src.SystemTag)
}

func TestWindowsCppSource_Missing(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "include"), 0o755))
	withTempMingwRoot(t, root)

	_, err := windowsCppSource()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no mingw libstdc++ headers")
}

func TestDarwinCSource_Found(t *testing.T) {
	sdkInclude := filepath.Join(t.TempDir(), "MacOSX.sdk", "usr", "include")
	require.NoError(t, os.MkdirAll(sdkInclude, 0o755))
	withTempDarwinRoots(t, []string{sdkInclude}, []string{"/nope"})

	src, err := darwinCSource()
	require.NoError(t, err)
	assert.Equal(t, core.PlatformDarwin, src.Platform)
	assert.Equal(t, core.LanguageC, src.Language)
	assert.Equal(t, "darwin-MacOSX.sdk", src.SystemTag)
}

func TestDarwinCSource_Missing(t *testing.T) {
	withTempDarwinRoots(t, []string{"/no", "/where"}, []string{"/no"})
	_, err := darwinCSource()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "macOS SDK headers not found")
}

func TestDarwinCppSource_Found(t *testing.T) {
	cppInclude := filepath.Join(t.TempDir(), "v1")
	require.NoError(t, os.MkdirAll(cppInclude, 0o755))
	withTempDarwinRoots(t, []string{"/no"}, []string{cppInclude})

	src, err := darwinCppSource()
	require.NoError(t, err)
	assert.Equal(t, core.PlatformDarwin, src.Platform)
	assert.Equal(t, core.LanguageCpp, src.Language)
	assert.Equal(t, "libc++-darwin-v1", src.SystemTag)
}

func TestDarwinCppSource_Missing(t *testing.T) {
	withTempDarwinRoots(t, []string{"/no"}, []string{"/no", "/where"})
	_, err := darwinCppSource()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "libc++ headers not found")
}

func TestFindVersionedDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "13"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "12"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "experimental"), 0o755))

	dir, ver := findVersionedDir(root)
	assert.Equal(t, filepath.Join(root, "13"), dir)
	assert.Equal(t, "13", ver)
}

func TestFindVersionedDir_MissingRoot(t *testing.T) {
	dir, ver := findVersionedDir(filepath.Join(t.TempDir(), "missing"))
	assert.Empty(t, dir)
	assert.Empty(t, ver)
}

func TestFindVersionedDir_NoVersionedEntries(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "experimental"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal"), 0o755))
	dir, ver := findVersionedDir(root)
	assert.Empty(t, dir)
	assert.Empty(t, ver)
}

func TestFirstExistingDir(t *testing.T) {
	good := t.TempDir()
	assert.Equal(t, good, firstExistingDir([]string{"/missing", good}))
	assert.Empty(t, firstExistingDir([]string{"/missing", "/also-missing"}))
}

func TestDetectDarwinSDKTag(t *testing.T) {
	assert.Equal(t, "MacOSX.sdk",
		detectDarwinSDKTag("/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include"))
}

func TestDetectDarwinCppTag(t *testing.T) {
	assert.Equal(t, "v1", detectDarwinCppTag("/Library/Developer/CommandLineTools/usr/include/c++/v1"))
}

// TestDiscoverHeaderSources_DarwinAndWindowsDispatched is the integration
// test for DiscoverHeaderSources: it confirms each (target, language) reaches
// the right per-platform source builder when fixture trees exist on disk.
func TestDiscoverHeaderSources_DarwinAndWindowsDispatched(t *testing.T) {
	mingw := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(mingw, "include"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(mingw, "include", "c++", "13"), 0o755))
	withTempMingwRoot(t, mingw)

	sdkInclude := filepath.Join(t.TempDir(), "MacOSX.sdk", "usr", "include")
	require.NoError(t, os.MkdirAll(sdkInclude, 0o755))
	cppInclude := filepath.Join(t.TempDir(), "v1")
	require.NoError(t, os.MkdirAll(cppInclude, 0o755))
	withTempDarwinRoots(t, []string{sdkInclude}, []string{cppInclude})

	for _, tt := range []struct {
		target, language, wantTag string
	}{
		{core.PlatformWindows, core.LanguageC, "mingw-w64-13"},
		{core.PlatformWindows, core.LanguageCpp, "mingw-w64-libstdc++-13-upstream"},
		{core.PlatformDarwin, core.LanguageC, "darwin-MacOSX.sdk"},
		{core.PlatformDarwin, core.LanguageCpp, "libc++-darwin-v1"},
	} {
		t.Run(tt.target+"/"+tt.language, func(t *testing.T) {
			sources, err := DiscoverHeaderSources(tt.target, tt.language)
			require.NoError(t, err)
			require.Len(t, sources, 1)
			assert.Equal(t, tt.wantTag, sources[0].SystemTag)
		})
	}
}

// TestDiscoverHeaderSources_DarwinAndWindowsErrorWhenAbsent surfaces the
// error path: missing toolchain → clear error message.
func TestDiscoverHeaderSources_DarwinAndWindowsErrorWhenAbsent(t *testing.T) {
	withTempMingwRoot(t, "/definitely/missing")
	withTempDarwinRoots(t, []string{"/missing/sdk"}, []string{"/missing/cpp"})

	for _, tc := range []struct {
		target, language, fragment string
	}{
		{core.PlatformWindows, core.LanguageC, "mingw-w64 headers not found"},
		{core.PlatformWindows, core.LanguageCpp, "no mingw libstdc++"},
		{core.PlatformDarwin, core.LanguageC, "macOS SDK headers not found"},
		{core.PlatformDarwin, core.LanguageCpp, "libc++ headers not found"},
	} {
		t.Run(tc.target+"/"+tc.language, func(t *testing.T) {
			_, err := DiscoverHeaderSources(tc.target, tc.language)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.fragment)
		})
	}
}
