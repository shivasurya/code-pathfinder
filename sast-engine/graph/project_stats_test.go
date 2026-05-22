package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- languageOf -----------------------------------------------------------

func TestLanguageOf_SupportedExtensions(t *testing.T) {
	cases := map[string]string{
		"/repo/Main.java":                "Java",
		"/repo/app.py":                   "Python",
		"/repo/types.pyi":                "Python",
		"/repo/main.go":                  "Go",
		"/repo/main.c":                   "C/C++",
		"/repo/main.h":                   "C/C++",
		"/repo/main.cpp":                 "C/C++",
		"/repo/main.cc":                  "C/C++",
		"/repo/main.cxx":                 "C/C++",
		"/repo/main.hpp":                 "C/C++",
		"/repo/main.hh":                  "C/C++",
		"/repo/main.hxx":                 "C/C++",
		"/repo/Dockerfile":               "Dockerfile",
		"/repo/Dockerfile.dev":           "Dockerfile",
		"/repo/dockerfile":               "Dockerfile",
		"/repo/docker-compose.yml":       "docker-compose",
		"/repo/docker-compose.yaml":      "docker-compose",
		"/repo/docker-compose.prod.yml":  "docker-compose",
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			assert.Equal(t, want, languageOf(path))
		})
	}
}

func TestLanguageOf_UnsupportedExtensions(t *testing.T) {
	cases := map[string]string{
		"/repo/app.ts":     "TypeScript",
		"/repo/app.tsx":    "TypeScript",
		"/repo/app.js":     "JavaScript",
		"/repo/app.jsx":    "JavaScript",
		"/repo/app.mjs":    "JavaScript",
		"/repo/app.cjs":    "JavaScript",
		"/repo/app.rb":     "Ruby",
		"/repo/app.php":    "PHP",
		"/repo/app.rs":     "Rust",
		"/repo/App.kt":     "Kotlin",
		"/repo/build.kts":  "Kotlin",
		"/repo/App.swift":  "Swift",
		"/repo/App.scala":  "Scala",
		"/repo/App.cs":     "C#",
		"/repo/run.sh":     "Shell",
		"/repo/run.bash":   "Shell",
		"/repo/run.zsh":    "Shell",
		"/repo/conf.yaml":  "YAML",
		"/repo/conf.yml":   "YAML",
		"/repo/conf.json":  "JSON",
		"/repo/conf.xml":   "XML",
		"/repo/conf.toml":  "TOML",
		"/repo/README.md":  "Markdown",
		"/repo/index.html": "HTML",
		"/repo/index.htm":  "HTML",
		"/repo/styles.css": "CSS",
		"/repo/styles.scss": "CSS",
		"/repo/q.sql":      "SQL",
		"/repo/api.proto":  "Protobuf",
		"/repo/main.tf":    "Terraform",
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			assert.Equal(t, want, languageOf(path))
		})
	}
}

func TestLanguageOf_Uncounted(t *testing.T) {
	// Extensions we deliberately don't bucket so the "Detected:" line stays
	// honest about what's actually source code.
	cases := []string{
		"/repo/binary",            // no extension
		"/repo/image.png",         // binary
		"/repo/package-lock.json", // technically JSON; still rendered as JSON
		"/repo/Makefile",          // no opinion
		"/repo/.gitignore",        // dotfile, no opinion
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			got := languageOf(path)
			// "package-lock.json" is intentionally still JSON; everything
			// else is empty.
			if path == "/repo/package-lock.json" {
				assert.Equal(t, "JSON", got)
			} else {
				assert.Equal(t, "", got)
			}
		})
	}
}

func TestLanguageOf_DockerfileMustBePrefix(t *testing.T) {
	// "Dockerfile" prefix match — "Dockerfilexyz" still matches. That's
	// fine because the actual file walker only feeds in real Dockerfile
	// variants; the test pins the rule explicitly so a future refactor
	// doesn't tighten it unintentionally without us noticing.
	assert.Equal(t, "Dockerfile", languageOf("/repo/dockerfilexyz"))
}

func TestLanguageOf_DockerComposeRequiresYamlExt(t *testing.T) {
	// "docker-compose.json" or "docker-compose.txt" should NOT match the
	// supported docker-compose bucket.
	assert.Equal(t, "JSON", languageOf("/repo/docker-compose.json"))
	assert.Equal(t, "", languageOf("/repo/docker-compose.txt"))
}

// --- SupportedLanguages ----------------------------------------------------

func TestSupportedLanguages_ReturnsCopy(t *testing.T) {
	a := SupportedLanguages()
	b := SupportedLanguages()
	assert.Equal(t, a, b)
	// Mutate a; b should be unaffected.
	a[0] = "MUTATED"
	assert.NotEqual(t, a[0], b[0], "SupportedLanguages must return a defensive copy")
}

func TestSupportedLanguages_Contents(t *testing.T) {
	got := SupportedLanguages()
	assert.Equal(t, []string{"Java", "Python", "Go", "C/C++", "Dockerfile", "docker-compose"}, got)
}

// --- ProjectStats.recordFile -----------------------------------------------

func TestProjectStats_RecordFile_SupportedAndUnsupported(t *testing.T) {
	var s ProjectStats
	s.recordFile("/repo/Main.java", true)
	s.recordFile("/repo/app.ts", false)
	s.recordFile("/repo/conf.json", false)
	s.recordFile("/repo/binary", false) // not counted into ByLanguage

	assert.Equal(t, 4, s.TotalFiles)
	assert.Equal(t, 1, s.ScannedFiles)
	assert.Equal(t, map[string]int{
		"Java":       1,
		"TypeScript": 1,
		"JSON":       1,
	}, s.ByLanguage)
}

func TestProjectStats_RecordFile_NilMapInitialised(t *testing.T) {
	var s ProjectStats
	assert.Nil(t, s.ByLanguage)
	s.recordFile("/repo/foo.java", true)
	assert.NotNil(t, s.ByLanguage, "first recordFile must initialise the map")
}

// --- ProjectStats.UnsupportedFileCount ------------------------------------

func TestProjectStats_UnsupportedFileCount_Empty(t *testing.T) {
	var s ProjectStats
	assert.Equal(t, 0, s.UnsupportedFileCount())
}

func TestProjectStats_UnsupportedFileCount_AllSupported(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{"Java": 5, "Python": 3, "Go": 2}}
	assert.Equal(t, 0, s.UnsupportedFileCount())
}

func TestProjectStats_UnsupportedFileCount_MixedAndUnsupported(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{
		"Java":       5,
		"TypeScript": 32,
		"JavaScript": 8,
		"JSON":       5,
	}}
	assert.Equal(t, 45, s.UnsupportedFileCount())
}

// --- ProjectStats.UnsupportedSummary ---------------------------------------

func TestProjectStats_UnsupportedSummary_Empty(t *testing.T) {
	var s ProjectStats
	assert.Equal(t, "", s.UnsupportedSummary(5))
}

func TestProjectStats_UnsupportedSummary_OnlySupported(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{"Java": 5}}
	assert.Equal(t, "", s.UnsupportedSummary(5))
}

func TestProjectStats_UnsupportedSummary_SortedByCountDesc(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{
		"TypeScript": 32,
		"JavaScript": 8,
		"JSON":       5,
		"Markdown":   2,
	}}
	assert.Equal(t, "TypeScript (32), JavaScript (8), JSON (5), Markdown (2)", s.UnsupportedSummary(0))
}

func TestProjectStats_UnsupportedSummary_TopNCap(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{
		"TypeScript": 32,
		"JavaScript": 8,
		"JSON":       5,
		"Markdown":   2,
		"YAML":       1,
	}}
	assert.Equal(t, "TypeScript (32), JavaScript (8), JSON (5)", s.UnsupportedSummary(3))
}

func TestProjectStats_UnsupportedSummary_NegativeTopNMeansUnbounded(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{"TypeScript": 1, "Ruby": 1, "Rust": 1}}
	// 3 entries, all unsupported, topN <= 0 → all shown
	got := s.UnsupportedSummary(-1)
	assert.Contains(t, got, "TypeScript (1)")
	assert.Contains(t, got, "Ruby (1)")
	assert.Contains(t, got, "Rust (1)")
}

func TestProjectStats_UnsupportedSummary_TiesBrokenAlphabetically(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{
		"Ruby":       3,
		"Rust":       3,
		"TypeScript": 3,
	}}
	assert.Equal(t, "Ruby (3), Rust (3), TypeScript (3)", s.UnsupportedSummary(0))
}

func TestProjectStats_UnsupportedSummary_ExcludesSupported(t *testing.T) {
	s := ProjectStats{ByLanguage: map[string]int{
		"Java":       100, // supported, must not appear
		"TypeScript": 1,
	}}
	assert.Equal(t, "TypeScript (1)", s.UnsupportedSummary(0))
}
