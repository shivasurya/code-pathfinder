package clikeextract

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// HeaderSource describes one (platform, language) header tree to walk. Each
// search dir is rooted at a stdlib or POSIX install location; one extractor
// run typically yields 1–3 sources (e.g. for linux/cpp: the libstdc++ tree).
type HeaderSource struct {
	// Platform / Language echo Config.Target / Config.Language for stamping
	// emitted JSON files. Stored on the source rather than re-derived later
	// because cross-platform combinations may want platform-specific tags
	// (e.g. mingw vs MSVC headers both target windows).
	Platform string
	Language string

	// SearchDirs is the absolute path(s) to walk. The walker enumerates the
	// transitive contents and yields HeaderFile entries. Order is preserved:
	// earlier dirs win on duplicate header names (rare, but happens with
	// /usr/include vs /usr/local/include layouts).
	SearchDirs []string

	// HeaderExts is the set of recognised header extensions. C uses
	// `.h` only; C++ also accepts the extensionless form (<vector>) and
	// `.hpp`/`.hxx`. The empty string in this slice opts in to extensionless
	// files — used for the C++ STL where files like /usr/include/c++/13/vector
	// have no suffix.
	HeaderExts []string

	// SystemTag identifies the source library + version, e.g. "glibc-2.39",
	// "libstdc++-13", "mingw-w64-libstdc++-13". Stamped into manifest.SystemTag
	// for downstream debugging.
	SystemTag string
}

// HeaderFile is one discovered header on disk: the header name as you'd #include
// it, plus the absolute path to read.
type HeaderFile struct {
	// Name is the canonical #include form (e.g. "stdio.h", "vector",
	// "sys/socket.h"). Computed by stripping the search-dir prefix from Path.
	Name string

	// Path is the absolute filesystem path. Used to read the source bytes.
	Path string
}

// DiscoverHeaderSources returns the list of HeaderSource entries to walk for
// the given (target, language). PR-01 supports linux/c and linux/cpp; windows
// and darwin paths are scaffolded with a clear "not yet implemented" error so
// PR-03 can fill them in without re-shaping the surface.
//
// Detection of glibc/libstdc++ versions is best-effort: the function probes
// the filesystem for canonical install paths and falls back to a generic
// "unknown" tag on failure rather than refusing to run. This keeps the
// generator usable in containerised CI environments where the version-detection
// commands might not be available.
func DiscoverHeaderSources(target, language string) ([]HeaderSource, error) {
	switch target + "/" + language {
	case core.PlatformLinux + "/" + core.LanguageC:
		return []HeaderSource{linuxCSource()}, nil

	case core.PlatformLinux + "/" + core.LanguageCpp:
		src, err := linuxCppSource()
		if err != nil {
			return nil, err
		}
		return []HeaderSource{src}, nil

	case core.PlatformWindows + "/" + core.LanguageC:
		src, err := windowsCSource()
		if err != nil {
			return nil, err
		}
		return []HeaderSource{src}, nil

	case core.PlatformWindows + "/" + core.LanguageCpp:
		src, err := windowsCppSource()
		if err != nil {
			return nil, err
		}
		return []HeaderSource{src}, nil

	case core.PlatformDarwin + "/" + core.LanguageC:
		src, err := darwinCSource()
		if err != nil {
			return nil, err
		}
		return []HeaderSource{src}, nil

	case core.PlatformDarwin + "/" + core.LanguageCpp:
		src, err := darwinCppSource()
		if err != nil {
			return nil, err
		}
		return []HeaderSource{src}, nil

	default:
		return nil, fmt.Errorf("DiscoverHeaderSources: unknown target+language combination %q+%q", target, language)
	}
}

// linuxLibcRoots is the ordered list of canonical glibc lib directories probed
// by detectGlibcTag. Exposed as a package var (rather than a literal inside the
// function) so tests can override the search list to exercise both the hit and
// miss branches without depending on the host's filesystem layout.
var linuxLibcRoots = []string{
	"/lib/x86_64-linux-gnu",
	"/lib/aarch64-linux-gnu",
	"/lib64",
	"/usr/lib/x86_64-linux-gnu",
	"/usr/lib/aarch64-linux-gnu",
	"/usr/lib64",
}

// linuxCppRoot is the libstdc++ include directory probed by findLibstdcppRoot.
// Test override knob, same rationale as linuxLibcRoots.
var linuxCppRoot = "/usr/include/c++"

// linuxCSource returns the C header source on Linux. Always succeeds because
// the search dir (/usr/include) is universally present on glibc systems and
// the walker tolerates absent dirs at WalkHeaders time anyway.
func linuxCSource() HeaderSource {
	return HeaderSource{
		Platform:   core.PlatformLinux,
		Language:   core.LanguageC,
		SearchDirs: []string{"/usr/include"},
		HeaderExts: []string{".h"},
		SystemTag:  detectGlibcTag(),
	}
}

// linuxCppSource returns the C++ header source on Linux. Probes for the libstdc++
// install via linuxCppRoot/<version>; returns an explicit error if no version is
// found because, unlike the C path, the C++ tree is unrecoverable without one
// (the directory name itself encodes the version).
func linuxCppSource() (HeaderSource, error) {
	dir, version := findLibstdcppRoot()
	if dir == "" {
		return HeaderSource{}, errors.New("linuxCppSource: no libstdc++ headers found under " + linuxCppRoot + "/* " +
			"(install libstdc++-dev / libstdc++-13-dev or run inside a container with C++ stdlib headers)")
	}
	return HeaderSource{
		Platform:   core.PlatformLinux,
		Language:   core.LanguageCpp,
		SearchDirs: []string{dir},
		HeaderExts: []string{".h", ".hpp", ".hxx", ""},
		SystemTag:  "libstdc++-" + version,
	}, nil
}

// detectGlibcTag returns "glibc-<basename>" of the first existing dir from
// linuxLibcRoots, falling back to "glibc-unknown" so the manifest always has
// *something* identifiable. Generated registries don't have to match
// `ldd --version` exactly — the tag is for spotting registry-vs-runtime
// mismatches, not for downstream feature detection.
func detectGlibcTag() string {
	for _, c := range linuxLibcRoots {
		if dirExists(c) {
			return "glibc-" + filepath.Base(c)
		}
	}
	return "glibc-unknown"
}

// findLibstdcppRoot inspects linuxCppRoot/<version> and returns the lexically-
// largest version directory (which, for purely-numeric names, is the freshest
// install) plus its version string. Returns ("","") if no install is found.
func findLibstdcppRoot() (dir, version string) {
	entries, err := os.ReadDir(linuxCppRoot)
	if err != nil {
		return "", ""
	}

	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Accept any directory name that contains at least one digit — covers
		// the canonical `13`, `13.2.0`, and similar; skips bare `experimental`.
		name := e.Name()
		if !containsDigit(name) {
			continue
		}
		versions = append(versions, name)
	}
	if len(versions) == 0 {
		return "", ""
	}

	sort.Strings(versions)
	v := versions[len(versions)-1]
	return filepath.Join(linuxCppRoot, v), v
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// WalkHeaders enumerates header files under src.SearchDirs that match the
// configured extensions. Internal/private subdirectories (`bits/`, `internal/`,
// `__support/`, etc.) are skipped because their contents are compiler-internal
// and the public namespace is sufficient.
//
// The result is deterministic — entries are sorted by header Name — so manifest
// output is stable across runs (independent of filesystem walk order).
func (s HeaderSource) WalkHeaders() ([]HeaderFile, error) {
	if len(s.SearchDirs) == 0 {
		return nil, errors.New("WalkHeaders: HeaderSource has no SearchDirs")
	}

	var found []HeaderFile
	seen := make(map[string]struct{}) // dedupe by canonical name

	for _, dir := range s.SearchDirs {
		if !dirExists(dir) {
			return nil, fmt.Errorf("WalkHeaders: search dir %q does not exist (install missing system headers)", dir)
		}

		walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if shouldSkipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if !s.matchesExt(d.Name()) {
				return nil
			}
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return fmt.Errorf("relative path of %q: %w", path, relErr)
			}
			name := filepath.ToSlash(rel)
			if _, dup := seen[name]; dup {
				return nil
			}
			seen[name] = struct{}{}
			found = append(found, HeaderFile{Name: name, Path: path})
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("walking %q: %w", dir, walkErr)
		}
	}

	sort.Slice(found, func(i, j int) bool {
		return found[i].Name < found[j].Name
	})
	return found, nil
}

// shouldSkipDir reports whether the directory at the given name should be pruned
// from the walk. The pruned set is intentionally small and conservative: only
// directories that the C/C++ standards explicitly designate as implementation
// detail.
func shouldSkipDir(name string) bool {
	switch name {
	case "bits", "internal", "__support", "ext", "experimental", "tr1", "tr2",
		"parallel", "pstl", "debug", "profile", "decimal":
		return true
	}
	return false
}

// matchesExt reports whether filename is one of this source's recognised header
// extensions. The empty extension (extensionless) is also matched: tree-sitter
// can parse `<vector>` from libstdc++ even though it has no suffix.
func (s HeaderSource) matchesExt(filename string) bool {
	ext := filepath.Ext(filename)
	return slices.Contains(s.HeaderExts, ext)
}

// IsHostLinux is a runtime helper used by tests that need to short-circuit
// when the host isn't running glibc (e.g. CI macOS runners, Windows). Kept here
// rather than in a test helper so the generator binary itself can use it for
// graceful warnings if a developer runs it on a non-Linux machine before PR-03
// adds darwin/windows support.
func IsHostLinux() bool {
	return runtime.GOOS == core.PlatformLinux
}

// HeaderName converts an absolute path under one of src's search dirs back into
// its #include form — the inverse of WalkHeaders' Name → Path mapping. Useful
// when a caller already has a path and wants the registry-key form.
//
// Returns "" if path is not under any of the source's search dirs.
func (s HeaderSource) HeaderName(path string) string {
	for _, dir := range s.SearchDirs {
		rel, err := filepath.Rel(dir, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		return filepath.ToSlash(rel)
	}
	return ""
}
