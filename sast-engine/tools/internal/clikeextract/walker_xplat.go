package clikeextract

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Windows headers, accessed cross-platform via mingw-w64 on Ubuntu.
//
// `apt install mingw-w64` on ubuntu-latest places the Win32 + MSVCRT headers
// under /usr/x86_64-w64-mingw32/include and the mingw libstdc++ tree at
// /usr/x86_64-w64-mingw32/include/c++/<version>. Using mingw on Linux beats
// running a Windows GitHub Actions runner for cost and simplicity, and gives
// us a faithful Win32 surface for stdlib resolution.
//
// All paths exposed as package vars (rather than literals) so tests can
// override them to exercise both the hit and miss branches without depending
// on whether the host actually has mingw installed.
var (
	// windowsMingwRoot is the canonical mingw-w64 install root. Subdirectories
	// `include` (C) and `include/c++/<version>` (C++) live underneath.
	windowsMingwRoot = "/usr/x86_64-w64-mingw32"

	// darwinSDKRoots is the ordered list of macOS SDK include directories the
	// generator probes. Command Line Tools first because that's the lighter
	// install used in CI; Xcode.app is a fallback for full developer setups.
	darwinSDKRoots = []string{
		"/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include",
		"/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/usr/include",
	}

	// darwinCppRoots is the ordered list of clang-shipped libc++ trees under
	// the Apple toolchain. macos-latest's xcrun typically lands the headers
	// under Command Line Tools.
	darwinCppRoots = []string{
		"/Library/Developer/CommandLineTools/usr/include/c++/v1",
		"/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/include/c++/v1",
	}
)

// windowsCSource constructs the HeaderSource for Win32 + MSVCRT C headers.
// Probes the canonical mingw include dir; on miss, returns an error with a
// concrete remediation hint so CI logs make the cause obvious.
func windowsCSource() (HeaderSource, error) {
	dir := filepath.Join(windowsMingwRoot, "include")
	if !dirExists(dir) {
		return HeaderSource{}, fmt.Errorf("windowsCSource: mingw-w64 headers not found at %s "+
			"(install with: apt install mingw-w64)", dir)
	}
	return HeaderSource{
		Platform:   core.PlatformWindows,
		Language:   core.LanguageC,
		SearchDirs: []string{dir},
		HeaderExts: []string{".h"},
		SystemTag:  "mingw-w64-" + detectMingwVersion(),
	}, nil
}

// windowsCppSource finds the mingw libstdc++ header tree. Returns an error
// when no version directory exists — without one, the C++ surface is
// unrecoverable (the directory name encodes the version).
//
// The walk lists the C++ STL tree only. Win32 C headers are exposed via
// windowsCSource; mixing both into one source would conflate languages.
func windowsCppSource() (HeaderSource, error) {
	root := filepath.Join(windowsMingwRoot, "include", "c++")
	dir, version := findVersionedDir(root)
	if dir == "" {
		return HeaderSource{}, fmt.Errorf("windowsCppSource: no mingw libstdc++ headers under %s "+
			"(install with: apt install g++-mingw-w64)", root)
	}
	return HeaderSource{
		Platform:   core.PlatformWindows,
		Language:   core.LanguageCpp,
		SearchDirs: []string{dir},
		HeaderExts: []string{".h", ".hpp", ".hxx", ""},
		SystemTag:  "mingw-w64-libstdc++-" + version,
	}, nil
}

// darwinCSource probes the canonical macOS SDK include locations and uses
// the first one that exists. Apple ships Command Line Tools and full Xcode
// installs in different subtrees; the loader tries CLT first because that's
// the cheaper macos-latest layout.
func darwinCSource() (HeaderSource, error) {
	dir := firstExistingDir(darwinSDKRoots)
	if dir == "" {
		return HeaderSource{}, errors.New("darwinCSource: macOS SDK headers not found at any of " +
			fmt.Sprint(darwinSDKRoots) + " (install Command Line Tools: xcode-select --install)")
	}
	return HeaderSource{
		Platform:   core.PlatformDarwin,
		Language:   core.LanguageC,
		SearchDirs: []string{dir},
		HeaderExts: []string{".h"},
		SystemTag:  "darwin-" + detectDarwinSDKTag(dir),
	}, nil
}

// darwinCppSource probes the libc++ tree shipped with the Apple toolchain.
// Apple's libc++ has a notably different surface from libstdc++ — different
// container ABI, different `__1::` inline namespace — but the same public
// API; the manifest is generated against the actual installed headers so
// resolution stays correct on the host platform.
func darwinCppSource() (HeaderSource, error) {
	dir := firstExistingDir(darwinCppRoots)
	if dir == "" {
		return HeaderSource{}, errors.New("darwinCppSource: libc++ headers not found at any of " +
			fmt.Sprint(darwinCppRoots) + " (install Xcode or Command Line Tools)")
	}
	return HeaderSource{
		Platform:   core.PlatformDarwin,
		Language:   core.LanguageCpp,
		SearchDirs: []string{dir},
		HeaderExts: []string{".h", ".hpp", ".hxx", ""},
		SystemTag:  "libc++-darwin-" + detectDarwinCppTag(dir),
	}, nil
}

// findVersionedDir lists root and returns the lexically-largest entry name
// containing a digit (canonical "13", "13.2.0", "v1") together with its
// version. Returns ("","") on missing root or empty result.
//
// Shared between windowsCppSource (looks for c++/<gcc-version>) and the
// darwin probes (looks for c++/v<n>) — both want the freshest version dir
// without parsing semver explicitly.
func findVersionedDir(root string) (dir, version string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", ""
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
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
	return filepath.Join(root, v), v
}

// firstExistingDir returns the first directory in the list that exists on
// disk, or "" if none do. Order matters — callers list the cheaper / more
// likely option first.
func firstExistingDir(candidates []string) string {
	for _, c := range candidates {
		if dirExists(c) {
			return c
		}
	}
	return ""
}

// detectMingwVersion derives a version string from the libstdc++ directory
// name embedded under the mingw root. Returns "unknown" when the tree has
// not been probed yet (windowsCppSource hits this path before the C source
// builder runs). Lightweight on purpose: parsing `gcc --version` would add
// an exec dependency that complicates testing for marginal accuracy gain.
func detectMingwVersion() string {
	root := filepath.Join(windowsMingwRoot, "include", "c++")
	_, v := findVersionedDir(root)
	if v == "" {
		return "unknown"
	}
	return v
}

// detectDarwinSDKTag returns a short identifier for the SDK whose include
// dir was selected. Currently uses the parent directory name (e.g.
// "MacOSX.sdk") which is enough to tell CommandLineTools apart from Xcode.
func detectDarwinSDKTag(includeDir string) string {
	// includeDir ends in `.../MacOSX.sdk/usr/include`.
	parent := filepath.Base(filepath.Dir(filepath.Dir(includeDir)))
	if parent == "" {
		return "unknown"
	}
	return parent
}

// detectDarwinCppTag returns the libc++ version directory name (typically
// "v1") so the manifest can distinguish future ABI bumps.
func detectDarwinCppTag(includeDir string) string {
	base := filepath.Base(includeDir)
	if base == "" {
		return "unknown"
	}
	return base
}
