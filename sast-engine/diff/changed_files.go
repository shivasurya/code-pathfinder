package diff

import "fmt"

// ChangedFilesProvider abstracts how changed files are obtained.
type ChangedFilesProvider interface {
	// GetChangedFiles returns relative file paths changed between base and head.
	GetChangedFiles() ([]string, error)
}

// ProviderOptions configures how changed files are computed.
type ProviderOptions struct {
	// ProjectRoot is the absolute path to the project directory (required for git provider).
	ProjectRoot string

	// BaseRef is the baseline git ref (branch, tag, or commit SHA).
	BaseRef string

	// HeadRef is the head git ref to compare against baseline. Defaults to "HEAD".
	HeadRef string

	// GitHubToken is the GitHub API token for authenticated requests.
	GitHubToken string

	// Owner is the GitHub repository owner (e.g., "shivasurya").
	Owner string

	// Repo is the GitHub repository name (e.g., "code-pathfinder").
	Repo string

	// PRNumber is the pull request number for GitHub API-based diff.
	PRNumber int
}

// NewChangedFilesProvider creates a ChangedFilesProvider based on available options.
// GitHub API is preferred when token and PR number are available (more reliable,
// immune to merge commit confusion). Falls back to git-based diff otherwise.
func NewChangedFilesProvider(opts ProviderOptions) (ChangedFilesProvider, error) {
	if opts.BaseRef == "" && !hasGitHubPRContext(opts) {
		return nil, fmt.Errorf("no baseline ref provided: set --base or provide GitHub PR context (--github-token, --github-repo, --github-pr)")
	}

	// GitHub API takes priority when available (more reliable).
	if hasGitHubPRContext(opts) {
		owner, repo, err := parseOwnerRepo(opts)
		if err != nil {
			return nil, err
		}
		return &GitHubAPIDiffProvider{
			Token:    opts.GitHubToken,
			Owner:    owner,
			Repo:     repo,
			PRNumber: opts.PRNumber,
		}, nil
	}

	// Fallback to git-based diff.
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

// hasGitHubPRContext returns true if GitHub API-based diff is possible.
func hasGitHubPRContext(opts ProviderOptions) bool {
	return opts.GitHubToken != "" && opts.PRNumber > 0 && (opts.Owner != "" || opts.Repo != "")
}

// parseOwnerRepo extracts owner and repo from ProviderOptions.
// If Owner and Repo are set directly, uses those. Otherwise returns an error.
func parseOwnerRepo(opts ProviderOptions) (string, string, error) {
	if opts.Owner != "" && opts.Repo != "" {
		return opts.Owner, opts.Repo, nil
	}
	return "", "", fmt.Errorf("github-repo must specify both owner and repo (e.g., --github-repo owner/repo)")
}
