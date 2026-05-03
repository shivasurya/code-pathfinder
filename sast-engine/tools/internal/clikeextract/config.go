package clikeextract

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// GeneratorVersion is the version of the clikeextract pipeline. Embedded in the
// generated manifest for downstream debugging — when a registry file is misbehaving,
// the consumer can correlate it with the generator that produced it.
const GeneratorVersion = "1.0.0"

// SchemaVersion is the schema_version stamped on every emitted JSON file. Loaders
// (PR-02) read this and reject anything outside the supported range. Bump only on
// breaking schema changes (renamed/removed fields); additive changes (new fields,
// new headers) stay on the same SchemaVersion.
const SchemaVersion = "1.0.0"

// RegistryVersion is the URL-path segment between language and per-header file
// (e.g. .../linux/c/v1/stdio_stdlib.json). Distinct from SchemaVersion: schema can
// move forward without rev'ing the URL path, and vice versa.
const RegistryVersion = "v1"

// DefaultBaseURL is the registry root on the production CDN. Used to construct
// per-header URLs in manifest entries. The URL does not have to resolve at
// generation time — the loader fetches at scan time.
const DefaultBaseURL = "https://assets.codepathfinder.dev/registries"

// Config carries the parameters for one Extractor.Run invocation. One run produces
// one (target, language) pair of registry files; producing the full Linux+Windows+
// Darwin matrix takes six runs (handled by the GitHub Actions workflow in PR-03).
type Config struct {
	// Target is the platform the generated registry is for ("linux", "windows",
	// "darwin"). Used to drive header discovery (which dirs to walk) and stamped
	// into every emitted file's Platform field.
	Target string

	// Language is "c" or "cpp". Drives extractor dispatch and the URL-path
	// segment under which the registry is published.
	Language string

	// OutputDir is the directory where manifest.json and per-header JSONs are
	// written. Created if it does not exist; existing files are overwritten.
	OutputDir string

	// OverlayPath is the path to the YAML overlay file that augments tree-sitter
	// extraction with hand-curated entries. If empty, the extractor runs without
	// an overlay (every entry stays Source="header"). Mismatch between overlay's
	// declared language and Config.Language is a hard error at load time.
	OverlayPath string

	// BaseURL overrides DefaultBaseURL when stamping per-header URL fields. Tests
	// set this to a local file:// path so generated manifests can be replayed
	// without hitting the network.
	BaseURL string
}

// Validate reports the first inconsistency in cfg, or nil if cfg is usable.
// Validation is intentionally minimal — most invariants (search dirs exist,
// overlay parses) are enforced later in Run() where the error has more context.
func (c Config) Validate() error {
	if c.Target == "" {
		return fmt.Errorf("config: Target is required (one of %q, %q, %q)",
			core.PlatformLinux, core.PlatformWindows, core.PlatformDarwin)
	}
	switch c.Target {
	case core.PlatformLinux, core.PlatformWindows, core.PlatformDarwin:
	default:
		return fmt.Errorf("config: unsupported Target %q (allowed: %q, %q, %q)",
			c.Target, core.PlatformLinux, core.PlatformWindows, core.PlatformDarwin)
	}

	switch c.Language {
	case core.LanguageC, core.LanguageCpp:
	case "":
		return fmt.Errorf("config: Language is required (%q or %q)", core.LanguageC, core.LanguageCpp)
	default:
		return fmt.Errorf("config: unsupported Language %q (allowed: %q, %q)",
			c.Language, core.LanguageC, core.LanguageCpp)
	}

	if c.OutputDir == "" {
		return fmt.Errorf("config: OutputDir is required")
	}
	return nil
}

// effectiveBaseURL returns cfg.BaseURL if set, otherwise DefaultBaseURL.
func (c Config) effectiveBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}
