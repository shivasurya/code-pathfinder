package builder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveIndexPath_OverrideWins(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom", "idx.sqlite")
	got, err := ResolveIndexPath("/some/project", want, "/env/path.sqlite")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	// Parent is created so a fresh override path opens cleanly.
	assert.DirExists(t, filepath.Dir(want))
}

func TestResolveIndexPath_EnvVarUsedWhenNoOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "env", "idx.sqlite")
	got, err := ResolveIndexPath("/some/project", "", want)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.DirExists(t, filepath.Dir(want))
}

func TestResolveIndexPath_OverrideBeatsEnvVar(t *testing.T) {
	override := filepath.Join(t.TempDir(), "override.sqlite")
	got, err := ResolveIndexPath("/some/project", override, filepath.Join(t.TempDir(), "env.sqlite"))
	require.NoError(t, err)
	assert.Equal(t, override, got)
}

func TestResolveIndexPath_DefaultLocation(t *testing.T) {
	// TestMain points indexParentDirOverride at a temp dir, so the default
	// location resolves under it rather than the real $HOME.
	got, err := ResolveIndexPath("/some/project", "", "")
	require.NoError(t, err)
	assert.Equal(t, indexParentDirOverride, filepath.Dir(got))
	assert.True(t, strings.HasSuffix(got, ".sqlite"), "index file should end with .sqlite, got %q", got)
	// 16 hex chars + ".sqlite".
	assert.Len(t, filepath.Base(got), 16+len(".sqlite"))
}

func TestResolveIndexPath_DeterministicForSameRoot(t *testing.T) {
	a, err := ResolveIndexPath("/some/project", "", "")
	require.NoError(t, err)
	b, err := ResolveIndexPath("/some/project", "", "")
	require.NoError(t, err)
	assert.Equal(t, a, b, "same project root must map to the same index file")
}

func TestResolveIndexPath_RelativeAndAbsoluteRootMatch(t *testing.T) {
	// A relative path and its absolute form should hash to the same file.
	cwd, err := os.Getwd()
	require.NoError(t, err)
	rel := "somedir"
	abs := filepath.Join(cwd, rel)

	fromRel, err := ResolveIndexPath(rel, "", "")
	require.NoError(t, err)
	fromAbs, err := ResolveIndexPath(abs, "", "")
	require.NoError(t, err)
	assert.Equal(t, fromAbs, fromRel, "relative and absolute forms of the same root must match")
}

func TestResolveIndexPath_DistinctRootsDistinctFiles(t *testing.T) {
	a, err := ResolveIndexPath("/project/one", "", "")
	require.NoError(t, err)
	b, err := ResolveIndexPath("/project/two", "", "")
	require.NoError(t, err)
	assert.NotEqual(t, a, b, "distinct project roots must map to distinct index files")
}

func TestResolveIndexPath_MkdirParentFails(t *testing.T) {
	// Plant a regular file, then ask for an index path nested under it. The
	// override path is honoured, but creating its parent directory must fail.
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	_, err := ResolveIndexPath("/some/project", filepath.Join(blocker, "nested", "idx.sqlite"), "")
	assert.Error(t, err)
}

func TestResolveIndexFile_DefaultUnderHome(t *testing.T) {
	// With the test override cleared and a real HOME, the default location is
	// $HOME/.codepathfinder/<hash>.sqlite. This exercises the production
	// home-resolution branch that TestMain's override otherwise bypasses.
	saved := indexParentDirOverride
	indexParentDirOverride = ""
	t.Cleanup(func() { indexParentDirOverride = saved })
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := resolveIndexFile("/some/project", "", "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".codepathfinder"), filepath.Dir(got))
	assert.True(t, strings.HasSuffix(got, ".sqlite"), "got %q", got)
}

func TestResolveIndexPath_PropagatesResolveError(t *testing.T) {
	// ResolveIndexPath must surface an error from the inner resolver (here, an
	// unresolvable home) rather than proceeding to MkdirAll.
	saved := indexParentDirOverride
	indexParentDirOverride = ""
	t.Cleanup(func() { indexParentDirOverride = saved })
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")

	_, err := ResolveIndexPath("/some/project", "", "")
	if err == nil {
		t.Skip("platform resolved a home directory despite cleared env; nothing to assert")
	}
	assert.Error(t, err)
}

func TestResolveIndexFile_NoHomeNoOverride(t *testing.T) {
	// Clear the test override and HOME so os.UserHomeDir fails and the default
	// branch surfaces an actionable error rather than panicking. resolveIndexFile
	// is exercised directly (no filesystem side effects) for a deterministic hit
	// on the home-lookup error path.
	saved := indexParentDirOverride
	indexParentDirOverride = ""
	t.Cleanup(func() { indexParentDirOverride = saved })
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "") // windows
	t.Setenv("home", "")        // plan9

	_, err := resolveIndexFile("/some/project", "", "")
	if err == nil {
		t.Skip("platform resolved a home directory despite cleared env; nothing to assert")
	}
	assert.Contains(t, err.Error(), "home directory")
}
