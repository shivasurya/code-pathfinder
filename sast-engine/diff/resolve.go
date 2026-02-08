package diff

import "os"

// ResolveBaseRef auto-detects the baseline ref from CI environment variables.
// Returns empty string if no baseline can be detected (full scan).
func ResolveBaseRef() string {
	// GitHub Actions.
	if ref := os.Getenv("GITHUB_BASE_REF"); ref != "" {
		return "origin/" + ref
	}
	// GitLab CI.
	if ref := os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"); ref != "" {
		return "origin/" + ref
	}
	// Explicit env var override.
	if ref := os.Getenv("PATHFINDER_BASELINE_REF"); ref != "" {
		return ref
	}
	return "" // No baseline detected → full scan.
}

// ComputeChangedFiles resolves changed files between two git refs.
func ComputeChangedFiles(baseRef, headRef, projectRoot string) ([]string, error) {
	provider, err := NewChangedFilesProvider(ProviderOptions{
		ProjectRoot: projectRoot,
		BaseRef:     baseRef,
		HeadRef:     headRef,
	})
	if err != nil {
		return nil, err
	}

	return provider.GetChangedFiles()
}
