package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// IndexEnvVar is the environment variable that overrides the default index
// location. A non-empty value is used verbatim when no explicit override flag
// is passed. The precedence is: override flag > env var > default location.
const IndexEnvVar = "CODEPATHFINDER_INDEX_PATH"

// indexDirName is the per-user directory under $HOME that holds one SQLite
// index file per project root. It lives outside the project tree so there is
// nothing to .gitignore and nothing to commit by accident.
const indexDirName = ".codepathfinder"

// indexParentDirOverride, when non-empty, replaces $HOME/.codepathfinder as the
// directory that holds default-location index files. It is a test-isolation
// seam: the package test binary points it at a temp directory so tests never
// write into the developer's real home directory and never depend on $HOME
// (other resolvers in this package read $HOME for GOMODCACHE). It is never set
// in production.
var indexParentDirOverride string

// ResolveIndexPath computes the on-disk location of a project's FQN index.
//
// Precedence:
//  1. override (the --index-path flag) when non-empty.
//  2. envVar (CODEPATHFINDER_INDEX_PATH) when non-empty.
//  3. $HOME/.codepathfinder/<hash>.sqlite, where <hash> is the first 16 hex
//     chars of SHA-256 over the cleaned absolute project root. Two checkouts at
//     different absolute paths get distinct files; the same path always maps to
//     the same file regardless of the working directory a command runs from.
//
// The parent directory of the resolved path is created with 0700 perms
// (user-only) on first use, whether the path comes from an override, the env
// var, or the default location, so callers never hit a cryptic open error on a
// fresh path.
func ResolveIndexPath(projectRoot, override, envVar string) (string, error) {
	path, err := resolveIndexFile(projectRoot, override, envVar)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", fmt.Errorf("resolve index path: create %s: %w", parent, err)
	}
	return path, nil
}

// resolveIndexFile computes the index file path without touching the filesystem.
func resolveIndexFile(projectRoot, override, envVar string) (string, error) {
	if override != "" {
		return override, nil
	}
	if envVar != "" {
		return envVar, nil
	}

	abs, err := filepath.Abs(filepath.Clean(projectRoot))
	if err != nil {
		return "", fmt.Errorf("resolve index path: absolute path for %q: %w", projectRoot, err)
	}

	dir := indexParentDirOverride
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve index path: locate home directory: %w", err)
		}
		dir = filepath.Join(home, indexDirName)
	}

	sum := sha256.Sum256([]byte(abs))
	name := hex.EncodeToString(sum[:8]) + ".sqlite" // 8 bytes -> 16 hex chars
	return filepath.Join(dir, name), nil
}
