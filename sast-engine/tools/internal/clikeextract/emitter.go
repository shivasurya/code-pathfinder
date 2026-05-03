package clikeextract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// EmitOutput writes the per-header JSON files and the top-level manifest.json
// to outDir. The output layout matches what the loader (PR-02) expects:
//
//	<outDir>/manifest.json
//	<outDir>/<sanitized-header>_stdlib.json   (one per header)
//
// The manifest is computed deterministically — entries are sorted by header
// name, statistics are tallied from the actual symbol counts, and checksums
// are sha256 hashes of the per-header JSON bytes that ship to the CDN. Two
// runs over the same input produce byte-identical output.
//
// overlayApplied is the count returned by MergeOverlay across all headers,
// used to populate Statistics.OverlayOverrides for visibility in
// resolution-report.
func EmitOutput(headers []*core.CStdlibHeader, cfg Config, overlayApplied int) error {
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("EmitOutput: creating output dir %q: %w", cfg.OutputDir, err)
	}

	// Sort headers by name so the manifest's Headers slice is stable.
	sort.SliceStable(headers, func(i, j int) bool {
		return headers[i].Header < headers[j].Header
	})

	entries := make([]*core.CStdlibHeaderEntry, 0, len(headers))
	for _, h := range headers {
		entry, err := writePerHeader(h, cfg)
		if err != nil {
			return err
		}
		entries = append(entries, entry)
	}

	manifest := buildManifest(headers, entries, cfg, overlayApplied)
	return writeManifest(manifest, cfg)
}

// writePerHeader serialises one header to disk and returns the manifest entry
// describing it. The serialised bytes are also hashed (sha256) and the hash is
// embedded in the entry so loaders can verify integrity at fetch time.
func writePerHeader(h *core.CStdlibHeader, cfg Config) (*core.CStdlibHeaderEntry, error) {
	if h.GeneratedAt == "" {
		h.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("writePerHeader: marshalling %q: %w", h.Header, err)
	}

	filename := SanitizeHeaderName(h.Header) + "_stdlib.json"
	path := filepath.Join(cfg.OutputDir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("writePerHeader: writing %q: %w", path, err)
	}

	sum := sha256.Sum256(data)
	return &core.CStdlibHeaderEntry{
		Header:   h.Header,
		ModuleID: h.ModuleID,
		File:     filename,
		URL:      buildHeaderURL(cfg, filename),
		Size:     int64(len(data)),
		Checksum: "sha256:" + hex.EncodeToString(sum[:]),
	}, nil
}

// buildHeaderURL returns the absolute URL the loader will GET for a given
// header file. Format: <baseURL>/<platform>/<lang>/<registryVersion>/<file>.
func buildHeaderURL(cfg Config, filename string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		cfg.effectiveBaseURL(), cfg.Target, cfg.Language, RegistryVersion, filename)
}

// buildManifest assembles a CStdlibManifest with deterministic statistics and
// the per-header entries already on hand. Only this function knows how to
// compute aggregate counts; emitter callers must not roll their own.
func buildManifest(headers []*core.CStdlibHeader, entries []*core.CStdlibHeaderEntry,
	cfg Config, overlayApplied int) *core.CStdlibManifest {
	stats := computeStatistics(headers, overlayApplied)

	return &core.CStdlibManifest{
		SchemaVersion:    SchemaVersion,
		RegistryVersion:  RegistryVersion,
		Platform:         cfg.Target,
		Language:         cfg.Language,
		SystemTag:        firstSystemTag(headers),
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		GeneratorVersion: GeneratorVersion,
		BaseURL: fmt.Sprintf("%s/%s/%s/%s",
			cfg.effectiveBaseURL(), cfg.Target, cfg.Language, RegistryVersion),
		Headers:    entries,
		Statistics: stats,
	}
}

// computeStatistics tallies aggregate counts across all extracted headers.
// Symbol totals reflect the in-memory state AFTER overlay merge, so they
// include both extracted and overlay-only entries.
func computeStatistics(headers []*core.CStdlibHeader, overlayApplied int) *core.CStdlibStatistics {
	stats := &core.CStdlibStatistics{
		TotalHeaders:     len(headers),
		OverlayOverrides: overlayApplied,
	}
	for _, h := range headers {
		stats.TotalFunctions += len(h.Functions) + len(h.FreeFunctions)
		stats.TotalClasses += len(h.Classes)
		stats.TotalTypedefs += len(h.Typedefs)
		stats.TotalConstants += len(h.Constants)
		for _, cls := range h.Classes {
			stats.TotalFunctions += len(cls.Methods)
		}
	}
	return stats
}

// firstSystemTag returns the SystemTag of the first header in the slice, or
// the empty string if the slice is empty. All headers in a single run share
// the same SystemTag (we walk one HeaderSource at a time), so this picks the
// authoritative value without scanning all of them.
func firstSystemTag(headers []*core.CStdlibHeader) string {
	if len(headers) == 0 {
		return ""
	}
	return headers[0].SystemTag
}

// writeManifest serialises the top-level manifest.json next to the per-header
// JSONs. Like the per-header writer, the output is indented for human review
// in the local-validation gate.
func writeManifest(m *core.CStdlibManifest, cfg Config) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("writeManifest: marshalling: %w", err)
	}
	path := filepath.Join(cfg.OutputDir, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writeManifest: writing %q: %w", path, err)
	}
	return nil
}
