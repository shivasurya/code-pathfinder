package diff

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ValidateGitRef checks that a git ref exists and is reachable.
// Accepts: branch names (origin/main), tags (v1.0), commit SHAs (abc123),
// relative refs (HEAD~1), and special refs (HEAD).
// Returns a clear error message suggesting fetch-depth: 0 on failure.
func ValidateGitRef(projectRoot, ref string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", ref)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git rev-parse timed out after 30s for ref '%s'", ref)
		}
		return fmt.Errorf("invalid git ref '%s': not found in repository. "+
			"Ensure the ref exists and is fetched. "+
			"For CI, you may need 'fetch-depth: 0' in your checkout step.\n"+
			"git error: %s", ref, strings.TrimSpace(string(output)))
	}

	return nil
}
