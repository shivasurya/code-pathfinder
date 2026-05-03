package registry

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// DetectClikeTarget chooses the stdlib registry target for a project — one of
// "linux", "windows", or "darwin". The decision uses a three-source heuristic:
//
//  1. CLI override (the --target flag value, passed in as `override`). If
//     non-empty and matches a known platform, returned verbatim. An unknown
//     value falls through to the heuristic so the user gets a sensible
//     default rather than a startup error.
//  2. Macro signal — count occurrences of `_WIN32`, `_WIN64`, `__APPLE__`,
//     `__MACH__`, `__linux__`, `__GLIBC__` across the project's C/C++ source
//     and header files.
//  3. Path signal — count `/win32/`, `/windows/`, `/darwin/`, `/macos/`,
//     `/linux/`, `/unix/` segments in file paths. Each match is worth 5 points
//     (heavier than a single macro mention because path conventions are more
//     deliberate).
//
// On a tie or empty signal, returns "linux" — the default target for most
// security-critical C/C++ codebases.
//
// The walk is bounded: we read at most platformProbeMaxBytes per file and
// platformProbeMaxFiles files in total. A multi-million-line project still
// returns a result in well under a second.
func DetectClikeTarget(projectPath, override string) string {
	if t := normaliseTarget(override); t != "" {
		return t
	}

	macroScores := scanPlatformMacros(projectPath)
	pathScores := scanPathHints(projectPath)

	scores := map[string]int{
		core.PlatformLinux: macroScores["__linux__"] + macroScores["__GLIBC__"] +
			5*pathScores["linux"],
		core.PlatformWindows: macroScores["_WIN32"] + macroScores["_WIN64"] +
			5*pathScores["windows"],
		core.PlatformDarwin: macroScores["__APPLE__"] + macroScores["__MACH__"] +
			5*pathScores["darwin"],
	}

	best := core.PlatformLinux
	bestScore := 0
	for plat, score := range scores {
		if score > bestScore {
			best = plat
			bestScore = score
		}
	}
	return best
}

// normaliseTarget validates an explicit --target flag value. Returns the
// canonical platform string on a recognised input, or "" so the caller knows
// to fall through to the heuristic.
func normaliseTarget(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return ""
	case "linux", "ubuntu", "debian", "alpine":
		return core.PlatformLinux
	case "windows", "win32", "win64", "win", "msvc", "mingw":
		return core.PlatformWindows
	case "darwin", "macos", "macosx", "osx", "apple":
		return core.PlatformDarwin
	default:
		return ""
	}
}

const (
	// platformProbeMaxBytes caps the per-file read length. Macros usually
	// appear within the first kilobyte; reading more wastes I/O on enormous
	// generated headers.
	platformProbeMaxBytes = 4096

	// platformProbeMaxFiles caps the total number of files inspected. The
	// signal saturates well before we hit it.
	platformProbeMaxFiles = 5000
)

// scanPlatformMacros walks the project tree and counts platform-defining
// macro mentions in source files. Recognises the canonical preprocessor
// guards used by glibc, MSVCRT, and Apple SDK headers.
func scanPlatformMacros(projectPath string) map[string]int {
	counts := map[string]int{
		"_WIN32":     0,
		"_WIN64":     0,
		"__APPLE__":  0,
		"__MACH__":   0,
		"__linux__":  0,
		"__GLIBC__":  0,
	}

	scanned := 0
	_ = filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // ignore unreadable dirs; signal will sample what's reachable
		}
		if d.IsDir() {
			if shouldSkipProjectDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if scanned >= platformProbeMaxFiles {
			return fs.SkipAll
		}
		if !isClikeSourceExt(filepath.Ext(d.Name())) {
			return nil
		}
		scanned++

		data, readErr := readFileLimited(path, platformProbeMaxBytes)
		if readErr != nil {
			// Skip unreadable files (permissions, transient I/O errors) without
			// failing the whole detection — best-effort probe by design.
			return nil //nolint:nilerr // intentional: skip unreadable file
		}
		text := string(data)
		for macro := range counts {
			counts[macro] += strings.Count(text, macro)
		}
		return nil
	})

	return counts
}

// scanPathHints counts platform-named directories in the project tree. Maps
// a project layout convention ("src/linux/...", "third_party/win32/...") into
// a strong platform vote.
func scanPathHints(projectPath string) map[string]int {
	hits := map[string]int{
		"linux":   0,
		"windows": 0,
		"darwin":  0,
	}

	_ = filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // unreadable dirs are skipped silently
		}
		if !d.IsDir() {
			return nil
		}
		if shouldSkipProjectDir(d.Name()) {
			return fs.SkipDir
		}
		switch strings.ToLower(d.Name()) {
		case "linux", "unix":
			hits["linux"]++
		case "windows", "win32", "win64", "msvc":
			hits["windows"]++
		case "darwin", "macos", "apple":
			hits["darwin"]++
		}
		return nil
	})
	return hits
}

// shouldSkipProjectDir prunes directories that should not contribute to the
// platform signal — vendored dependencies, build artifacts, version control
// metadata. The list is conservative; everything else is walked.
func shouldSkipProjectDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn",
		"node_modules", "vendor", "third_party", "_build", "build",
		"target", "dist", "out", ".venv", "__pycache__":
		return true
	}
	return false
}

// isClikeSourceExt reports whether the file extension marks a C/C++ source
// or header. Used to avoid reading every file in the project.
func isClikeSourceExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".hxx":
		return true
	}
	return false
}

// readFileLimited opens path and reads up to limit bytes, then closes the
// file. Returns the bytes read (possibly fewer than limit on a short file)
// or any I/O error.
func readFileLimited(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from filepath.WalkDir under projectPath
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, limit)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return nil, err
	}
	return buf[:n], nil
}
