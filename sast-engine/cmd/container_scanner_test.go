package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/executor"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverContainerFiles(t *testing.T) {
	logger := output.NewLogger(output.VerbosityDefault)

	t.Run("Finds Dockerfile and compose files", func(t *testing.T) {
		// Create temp directory with container files
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("version: '3'"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("docs"), 0644))

		files, err := DiscoverContainerFiles(tmpDir, logger)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		// Check types
		types := make(map[string]int)
		for _, f := range files {
			types[f.Type]++
		}
		assert.Equal(t, 1, types["dockerfile"])
		assert.Equal(t, 1, types["compose"])
	})

	t.Run("Finds multiple Dockerfiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile.dev"), []byte("FROM node"), 0644))

		files, err := DiscoverContainerFiles(tmpDir, logger)
		require.NoError(t, err)
		assert.Len(t, files, 2)
		for _, f := range files {
			assert.Equal(t, "dockerfile", f.Type)
		}
	})

	t.Run("Skips common directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		nodeModules := filepath.Join(tmpDir, "node_modules")
		require.NoError(t, os.Mkdir(nodeModules, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(nodeModules, "Dockerfile"), []byte("FROM alpine"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644))

		files, err := DiscoverContainerFiles(tmpDir, logger)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.NotContains(t, files[0].Path, "node_modules")
	})

	t.Run("Returns empty for non-existent directory", func(t *testing.T) {
		files, err := DiscoverContainerFiles("/nonexistent/path", logger)
		assert.Error(t, err)
		assert.Nil(t, files)
	})

	t.Run("Returns empty when no container files", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.py"), []byte("print('hello')"), 0644))

		files, err := DiscoverContainerFiles(tmpDir, logger)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestCompileContainerRules(t *testing.T) {
	logger := output.NewLogger(output.VerbosityDefault)

	t.Run("Returns error when script not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := CompileContainerRules(tmpDir, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Returns error when script fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		pythonDSL := filepath.Join(tmpDir, "python-dsl")
		require.NoError(t, os.Mkdir(pythonDSL, 0755))
		
		scriptPath := filepath.Join(pythonDSL, "compile_container_rules.py")
		require.NoError(t, os.WriteFile(scriptPath, []byte("import sys\nsys.exit(1)"), 0755))

		_, err := CompileContainerRules(tmpDir, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compile")
	})
}

func TestFilterByType(t *testing.T) {
	files := []ContainerFile{
		{Path: "/a/Dockerfile", Type: "dockerfile"},
		{Path: "/a/docker-compose.yml", Type: "compose"},
		{Path: "/b/Dockerfile", Type: "dockerfile"},
	}

	t.Run("Filters dockerfiles", func(t *testing.T) {
		dockerfiles := filterByType(files, "dockerfile")
		assert.Len(t, dockerfiles, 2)
		for _, f := range dockerfiles {
			assert.Equal(t, "dockerfile", f.Type)
		}
	})

	t.Run("Filters compose files", func(t *testing.T) {
		composeFiles := filterByType(files, "compose")
		assert.Len(t, composeFiles, 1)
		assert.Equal(t, "compose", composeFiles[0].Type)
	})

	t.Run("Returns empty for unknown type", func(t *testing.T) {
		result := filterByType(files, "unknown")
		assert.Empty(t, result)
	})
}

func TestConvertToEnrichedDetection(t *testing.T) {
	projectPath := "/project"
	filePath := "/project/Dockerfile"

	t.Run("Converts RuleMatch correctly", func(t *testing.T) {
		match := executor.RuleMatch{
			RuleID:      "DOCKER-SEC-001",
			RuleName:    "Missing USER",
			Severity:    "high",
			Message:     "Container running as root",
			LineNumber:  5,
			CWE:         "CWE-250",
			ServiceName: "",
		}

		enriched := convertToEnrichedDetection(match, filePath, projectPath)

		assert.Equal(t, "DOCKER-SEC-001", enriched.Rule.ID)
		assert.Equal(t, "Missing USER", enriched.Rule.Name)
		assert.Equal(t, "high", enriched.Rule.Severity)
		assert.Equal(t, "Container running as root", enriched.Rule.Description)
		assert.Equal(t, 5, enriched.Location.Line)
		assert.Equal(t, filePath, enriched.Location.FilePath)
		assert.Equal(t, "Dockerfile", enriched.Location.RelPath)
		assert.Equal(t, dsl.DetectionTypePattern, enriched.DetectionType)
		assert.Equal(t, 1.0, enriched.Detection.Confidence)
		assert.Equal(t, "container", enriched.Detection.Scope)
		assert.Len(t, enriched.Rule.CWE, 1)
		assert.Equal(t, "CWE-250", enriched.Rule.CWE[0])
	})

	t.Run("Handles service name", func(t *testing.T) {
		match := executor.RuleMatch{
			RuleID:      "COMPOSE-SEC-001",
			RuleName:    "Privileged Mode",
			Severity:    "critical",
			Message:     "Running in privileged mode",
			LineNumber:  10,
			CWE:         "CWE-250",
			ServiceName: "web",
		}

		enriched := convertToEnrichedDetection(match, filePath, projectPath)
		assert.Contains(t, enriched.Rule.Description, "web")
	})

	t.Run("Handles missing CWE", func(t *testing.T) {
		match := executor.RuleMatch{
			RuleID:     "TEST-001",
			RuleName:   "Test Rule",
			Severity:   "low",
			Message:    "Test message",
			LineNumber: 1,
			CWE:        "",
		}

		enriched := convertToEnrichedDetection(match, filePath, projectPath)
		assert.Empty(t, enriched.Rule.CWE)
	})
}

func TestGetContainerSummary(t *testing.T) {
	t.Run("Returns 'no issues' for empty findings", func(t *testing.T) {
		summary := getContainerSummary([]*dsl.EnrichedDetection{})
		assert.Equal(t, "no issues", summary)
	})

	t.Run("Counts severities correctly", func(t *testing.T) {
		findings := []*dsl.EnrichedDetection{
			{Rule: dsl.RuleMetadata{Severity: "CRITICAL"}},
			{Rule: dsl.RuleMetadata{Severity: "HIGH"}},
			{Rule: dsl.RuleMetadata{Severity: "HIGH"}},
			{Rule: dsl.RuleMetadata{Severity: "MEDIUM"}},
			{Rule: dsl.RuleMetadata{Severity: "LOW"}},
			{Rule: dsl.RuleMetadata{Severity: "LOW"}},
			{Rule: dsl.RuleMetadata{Severity: "LOW"}},
		}

		summary := getContainerSummary(findings)
		assert.Contains(t, summary, "7 issues")
		assert.Contains(t, summary, "1 critical")
		assert.Contains(t, summary, "2 high")
		assert.Contains(t, summary, "1 medium")
		assert.Contains(t, summary, "3 low")
	})

	t.Run("Handles only critical", func(t *testing.T) {
		findings := []*dsl.EnrichedDetection{
			{Rule: dsl.RuleMetadata{Severity: "CRITICAL"}},
		}

		summary := getContainerSummary(findings)
		assert.Equal(t, "1 issues (1 critical)", summary)
	})
}

func TestFindProjectRoot(t *testing.T) {
	t.Run("Finds python-dsl directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		pythonDSL := filepath.Join(tmpDir, "python-dsl")
		require.NoError(t, os.Mkdir(pythonDSL, 0755))
		
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.Mkdir(subDir, 0755))

		root := findProjectRoot(subDir)
		assert.Equal(t, tmpDir, root)
	})

	t.Run("Returns original path if python-dsl not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		root := findProjectRoot(tmpDir)
		assert.Equal(t, tmpDir, root)
	})

	t.Run("Stops at filesystem root", func(t *testing.T) {
		// Use a deeply nested temp directory
		tmpDir := t.TempDir()
		deepDir := filepath.Join(tmpDir, "a", "b", "c", "d")
		require.NoError(t, os.MkdirAll(deepDir, 0755))

		root := findProjectRoot(deepDir)
		// Should return deepDir since python-dsl not found
		assert.Equal(t, deepDir, root)
	})
}

func TestScanContainerFiles(t *testing.T) {
	logger := output.NewLogger(output.VerbosityDefault)

	t.Run("Returns nil for empty files", func(t *testing.T) {
		findings, err := ScanContainerFiles([]ContainerFile{}, []byte("{}"), "/tmp", logger)
		assert.NoError(t, err)
		assert.Nil(t, findings)
	})

	t.Run("Returns error for invalid rules JSON", func(t *testing.T) {
		files := []ContainerFile{{Path: "/tmp/Dockerfile", Type: "dockerfile"}}
		findings, err := ScanContainerFiles(files, []byte("invalid json"), "/tmp", logger)
		assert.Error(t, err)
		assert.Nil(t, findings)
	})
}

func TestTryContainerScan(t *testing.T) {
	logger := output.NewLogger(output.VerbosityDefault)

	t.Run("Returns nil when no container files", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.py"), []byte("print('hello')"), 0644))

		findings := TryContainerScan(tmpDir, tmpDir, logger)
		assert.Nil(t, findings)
	})

	t.Run("Returns nil when rules unavailable", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644))

		findings := TryContainerScan(tmpDir, tmpDir, logger)
		assert.Nil(t, findings)
	})
}
