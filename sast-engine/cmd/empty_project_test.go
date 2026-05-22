package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

func TestEmptyProjectMessage_TotallyEmptyDir(t *testing.T) {
	msg := emptyProjectMessage(graph.ProjectStats{})
	assert.Equal(t, "No files to analyze. Project directory is empty (after applying skip and exclude rules).", msg)
}

func TestEmptyProjectMessage_OnlySupportedFilesButZeroNodes(t *testing.T) {
	// Files existed, all in supported languages, but graph ended empty
	// (e.g. every file failed to parse). Headline should reflect that
	// the walk did happen, distinct from the "empty directory" case.
	stats := graph.ProjectStats{
		TotalFiles:   3,
		ScannedFiles: 3,
		ByLanguage:   map[string]int{"Java": 3},
	}
	msg := emptyProjectMessage(stats)
	assert.Equal(t, "Scanned 3 file(s) but no analyzable content was produced.", msg)
}

func TestEmptyProjectMessage_OnlyUnsupportedFiles_MatchesDesignSpec(t *testing.T) {
	// Matches the example from the brainstorm:
	//   Scanned 0 of 47 files (47 unsupported).
	//   Detected:  TypeScript (32), JavaScript (8), JSON (5), Markdown (2)
	//   Supported: Java, Python, Go, C/C++, Dockerfile, docker-compose
	//
	//   No files to analyze. (Pathfinder doesn't analyze these languages yet.)
	stats := graph.ProjectStats{
		TotalFiles:   47,
		ScannedFiles: 0,
		ByLanguage: map[string]int{
			"TypeScript": 32,
			"JavaScript": 8,
			"JSON":       5,
			"Markdown":   2,
		},
	}
	msg := emptyProjectMessage(stats)
	want := "Scanned 0 of 47 files (47 unsupported).\n" +
		"Detected:  TypeScript (32), JavaScript (8), JSON (5), Markdown (2)\n" +
		"Supported: Java, Python, Go, C/C++, Dockerfile, docker-compose\n" +
		"\nNo files to analyze. (Pathfinder doesn't analyze these languages yet.)"
	assert.Equal(t, want, msg)
}

func TestEmptyProjectMessage_MixedButGraphEmpty(t *testing.T) {
	// Walk found 50 files: 5 supported (parsed but produced 0 nodes) and 45
	// unsupported. Treated as "files unsupported > 0" branch since the user
	// still has a noisy unsupported population to know about.
	stats := graph.ProjectStats{
		TotalFiles:   50,
		ScannedFiles: 5,
		ByLanguage: map[string]int{
			"Java":       5,
			"TypeScript": 45,
		},
	}
	msg := emptyProjectMessage(stats)
	assert.Contains(t, msg, "Scanned 0 of 50 files (45 unsupported).")
	assert.Contains(t, msg, "Detected:  TypeScript (45)")
	assert.Contains(t, msg, "Supported: Java, Python, Go, C/C++, Dockerfile, docker-compose")
}

func TestEmptyProjectMessage_FilesExistButNoneCategorised(t *testing.T) {
	// Walk saw N files but every one was a binary blob or had an
	// unrecognised extension (languageOf returns ""). ScannedFiles is 0
	// and UnsupportedFileCount is also 0 because nothing got bucketed.
	// Falls through to the default branch, which still surfaces that the
	// walk happened.
	stats := graph.ProjectStats{TotalFiles: 3}
	msg := emptyProjectMessage(stats)
	assert.Equal(t, "Scanned 3 file(s) but no analyzable content was produced.", msg)
}

func TestReportEmptyProject_AlwaysVisibleAtDefaultVerbosity(t *testing.T) {
	// Logger.Info must print regardless of verbosity (unlike Progress
	// which is gated to verbose+). Verify by piping a logger through a
	// buffer at the lowest verbosity and checking the output is non-empty.
	var buf bytes.Buffer
	logger := output.NewLoggerWithWriter(output.VerbosityDefault, &buf)
	stats := graph.ProjectStats{
		TotalFiles:   1,
		ScannedFiles: 0,
		ByLanguage:   map[string]int{"TypeScript": 1},
	}
	reportEmptyProject(logger, stats)
	got := buf.String()
	assert.True(t, strings.Contains(got, "Scanned 0 of 1 files"),
		"reportEmptyProject must write at default verbosity, got: %q", got)
	assert.True(t, strings.Contains(got, "Supported: Java"),
		"reportEmptyProject must include Supported line, got: %q", got)
}
