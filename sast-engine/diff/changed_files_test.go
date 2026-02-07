package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChangedFilesProvider_Git(t *testing.T) {
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
		BaseRef:     "origin/main",
		HeadRef:     "feature-branch",
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	gitProvider, ok := provider.(*GitDiffProvider)
	require.True(t, ok, "expected GitDiffProvider")
	assert.Equal(t, "/some/project", gitProvider.ProjectRoot)
	assert.Equal(t, "origin/main", gitProvider.BaseRef)
	assert.Equal(t, "feature-branch", gitProvider.HeadRef)
}

func TestNewChangedFilesProvider_DefaultHead(t *testing.T) {
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
		BaseRef:     "origin/main",
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	gitProvider, ok := provider.(*GitDiffProvider)
	require.True(t, ok, "expected GitDiffProvider")
	assert.Equal(t, "HEAD", gitProvider.HeadRef)
}

func TestNewChangedFilesProvider_NoBaseRef(t *testing.T) {
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
	}

	provider, err := NewChangedFilesProvider(opts)
	assert.Nil(t, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no baseline ref provided")
}

func TestNewChangedFilesProvider_EmptyOptions(t *testing.T) {
	provider, err := NewChangedFilesProvider(ProviderOptions{})
	assert.Nil(t, provider)
	assert.Error(t, err)
}

func TestNewChangedFilesProvider_AllFields(t *testing.T) {
	opts := ProviderOptions{
		ProjectRoot: "/project",
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	gitProvider, ok := provider.(*GitDiffProvider)
	require.True(t, ok)
	assert.Equal(t, "/project", gitProvider.ProjectRoot)
	assert.Equal(t, "main", gitProvider.BaseRef)
	assert.Equal(t, "HEAD", gitProvider.HeadRef)
}
