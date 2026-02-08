package diff

import "fmt"

// ChangedFilesProvider abstracts how changed files are obtained.
type ChangedFilesProvider interface {
	// GetChangedFiles returns relative file paths changed between base and head.
	GetChangedFiles() ([]string, error)
}

// ProviderOptions configures how changed files are computed.
type ProviderOptions struct {
	// ProjectRoot is the absolute path to the project directory.
	ProjectRoot string

	// BaseRef is the baseline git ref (branch, tag, or commit SHA).
	BaseRef string

	// HeadRef is the head git ref to compare against baseline. Defaults to "HEAD".
	HeadRef string
}

// NewChangedFilesProvider creates a git-based ChangedFilesProvider.
func NewChangedFilesProvider(opts ProviderOptions) (ChangedFilesProvider, error) {
	if opts.BaseRef == "" {
		return nil, fmt.Errorf("no baseline ref provided: set --base flag")
	}

	headRef := opts.HeadRef
	if headRef == "" {
		headRef = "HEAD"
	}
	return &GitDiffProvider{
		ProjectRoot: opts.ProjectRoot,
		BaseRef:     opts.BaseRef,
		HeadRef:     headRef,
	}, nil
}
