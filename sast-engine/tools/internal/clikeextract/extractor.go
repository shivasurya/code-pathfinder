package clikeextract

import (
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Extractor stitches the four pipeline stages — discovery, walk, extract, merge,
// emit — into a single Run() entrypoint. It mirrors the API of the sibling
// goextract.Extractor: construct with NewExtractor(cfg), call Run(), get an
// error or success.
type Extractor struct {
	cfg     Config
	overlay *Overlay
	logf    func(format string, args ...any)
}

// NewExtractor constructs an Extractor with the given configuration. The
// caller MUST have validated cfg with cfg.Validate() before this point —
// invalid configuration is a programming error.
func NewExtractor(cfg Config) *Extractor {
	return &Extractor{
		cfg: cfg,
		// Default logger writes to stderr; the entry-point binary uses this,
		// while tests can swap it out via SetLogger to capture output.
		logf: defaultLogf,
	}
}

// SetLogger overrides the warning-emit destination. Used by tests to capture
// continue-on-failure messages without writing to stderr.
func (e *Extractor) SetLogger(logf func(format string, args ...any)) {
	e.logf = logf
}

// Run executes the full pipeline. Errors abort the run only when they cannot
// be recovered from (missing search dirs, invalid overlay, unwritable output);
// per-header parse failures log a warning and continue, so a single bad header
// does not poison the whole registry.
func (e *Extractor) Run() error {
	if err := e.cfg.Validate(); err != nil {
		return fmt.Errorf("Run: invalid config: %w", err)
	}

	overlay, err := LoadOverlay(e.cfg.OverlayPath, e.cfg.Language)
	if err != nil {
		return err
	}
	e.overlay = overlay

	sources, err := discoverHeaderSourcesFn(e.cfg.Target, e.cfg.Language)
	if err != nil {
		return err
	}

	allHeaders := make([]*core.CStdlibHeader, 0, 256)
	overlayApplied := 0

	for _, src := range sources {
		files, err := src.WalkHeaders()
		if err != nil {
			return err
		}
		for _, f := range files {
			h, perHeaderErr := e.extractOne(f, src)
			if perHeaderErr != nil {
				e.logf("warning: skipping header %q: %v", f.Name, perHeaderErr)
				continue
			}
			overlayApplied += MergeOverlay(h, overlay)
			allHeaders = append(allHeaders, h)
		}
	}

	return EmitOutput(allHeaders, e.cfg, overlayApplied)
}

// extractOne dispatches to the language-specific extractor. Kept on the
// receiver (rather than a free function) so future fields on Extractor —
// caching, parallelism — have somewhere natural to land.
func (e *Extractor) extractOne(f HeaderFile, src HeaderSource) (*core.CStdlibHeader, error) {
	switch e.cfg.Language {
	case core.LanguageC:
		return extractCHeader(f, src)
	case core.LanguageCpp:
		return extractCppHeader(f, src)
	default:
		// Validate() should have caught this; defensive fallback.
		return nil, fmt.Errorf("extractOne: unsupported language %q", e.cfg.Language)
	}
}

func defaultLogf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// discoverHeaderSourcesFn is the package-level indirection that lets tests
// substitute a synthetic source list. Production code never re-binds this.
var discoverHeaderSourcesFn = DiscoverHeaderSources
