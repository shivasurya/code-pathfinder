package cmd

import (
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// emptyProjectTopLanguages caps the "Detected:" line so a polyglot repo with
// twenty extensions doesn't produce an unreadable scroll of language counts.
const emptyProjectTopLanguages = 5

// reportEmptyProject prints a user-facing explanation of why the scan ended
// up with zero analyzable files. Always visible (verbosity-independent) so a
// user pointed at the wrong directory or running pathfinder on a repo we
// don't support yet sees something concrete instead of a successful but
// silent run.
func reportEmptyProject(logger *output.Logger, stats graph.ProjectStats) {
	logger.Info(emptyProjectMessage(stats))
}

// emptyProjectMessage formats the multi-line explanation. Split out from
// reportEmptyProject so unit tests can assert the exact output without
// piping a logger through a buffer.
func emptyProjectMessage(stats graph.ProjectStats) string {
	var sb strings.Builder
	unsupported := stats.UnsupportedFileCount()

	switch {
	case stats.TotalFiles == 0:
		// Walk found nothing: empty directory, or everything was filtered
		// out by skip-dirs (vendor/, node_modules/, ...) and --exclude.
		sb.WriteString("No files to analyze. Project directory is empty (after applying skip and exclude rules).")

	case unsupported > 0:
		// Mixed or all-unsupported: this is the pathfinder-api case.
		// "Scanned 0 of 47 files (47 unsupported)."
		fmt.Fprintf(&sb, "Scanned 0 of %d files (%d unsupported).\n", stats.TotalFiles, unsupported)
		if summary := stats.UnsupportedSummary(emptyProjectTopLanguages); summary != "" {
			fmt.Fprintf(&sb, "Detected:  %s\n", summary)
		}
		fmt.Fprintf(&sb, "Supported: %s\n", strings.Join(graph.SupportedLanguages(), ", "))
		sb.WriteString("\nNo files to analyze. (Pathfinder doesn't analyze these languages yet.)")

	default:
		// Files existed and were all in supported languages, but the
		// parsers still produced zero graph nodes (e.g. every file
		// failed to parse, or the only files were tests excluded via
		// --skip-tests). Surface the count so the user knows the walk
		// did happen.
		fmt.Fprintf(&sb, "Scanned %d file(s) but no analyzable content was produced.", stats.TotalFiles)
	}
	return sb.String()
}
