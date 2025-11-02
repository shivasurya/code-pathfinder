package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPythonVersion_PythonVersionFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Write .python-version file
	versionFile := filepath.Join(tmpDir, ".python-version")
	err := os.WriteFile(versionFile, []byte("3.11.5\n"), 0644)
	require.NoError(t, err)

	version := detectPythonVersion(tmpDir)
	assert.Equal(t, "3.11", version)
}

func TestDetectPythonVersion_PyprojectToml_RequiresPython(t *testing.T) {
	tmpDir := t.TempDir()

	// Write pyproject.toml with requires-python
	pyprojectContent := `[project]
name = "test-project"
requires-python = ">=3.10"
`
	pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
	err := os.WriteFile(pyprojectFile, []byte(pyprojectContent), 0644)
	require.NoError(t, err)

	version := detectPythonVersion(tmpDir)
	assert.Equal(t, "3.10", version)
}

func TestDetectPythonVersion_PyprojectToml_Poetry(t *testing.T) {
	tmpDir := t.TempDir()

	// Write pyproject.toml with poetry dependencies
	pyprojectContent := `[tool.poetry]
name = "test-project"

[tool.poetry.dependencies]
python = "^3.12"
`
	pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
	err := os.WriteFile(pyprojectFile, []byte(pyprojectContent), 0644)
	require.NoError(t, err)

	version := detectPythonVersion(tmpDir)
	assert.Equal(t, "3.12", version)
}

func TestDetectPythonVersion_Default(t *testing.T) {
	tmpDir := t.TempDir()

	// No version files - should default to 3.14
	version := detectPythonVersion(tmpDir)
	assert.Equal(t, "3.14", version)
}

func TestDetectPythonVersion_PriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create both .python-version and pyproject.toml
	// .python-version should take priority
	versionFile := filepath.Join(tmpDir, ".python-version")
	err := os.WriteFile(versionFile, []byte("3.9.0"), 0644)
	require.NoError(t, err)

	pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
	pyprojectContent := `[project]
requires-python = ">=3.11"
`
	err = os.WriteFile(pyprojectFile, []byte(pyprojectContent), 0644)
	require.NoError(t, err)

	version := detectPythonVersion(tmpDir)
	assert.Equal(t, "3.9", version, ".python-version should take priority over pyproject.toml")
}

func TestReadPythonVersionFile_Success(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "full version",
			content:  "3.14.0",
			expected: "3.14",
		},
		{
			name:     "major.minor only",
			content:  "3.11",
			expected: "3.11",
		},
		{
			name:     "with newline",
			content:  "3.12.1\n",
			expected: "3.12",
		},
		{
			name:     "with spaces",
			content:  "  3.10.5  ",
			expected: "3.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			versionFile := filepath.Join(tmpDir, ".python-version")
			err := os.WriteFile(versionFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			version := readPythonVersionFile(tmpDir)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestReadPythonVersionFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	version := readPythonVersionFile(tmpDir)
	assert.Equal(t, "", version)
}

func TestParsePyprojectToml_RequiresPython(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "requires-python >=",
			content: `[project]
requires-python = ">=3.11"
`,
			expected: "3.11",
		},
		{
			name: "requires-python ==",
			content: `[project]
requires-python = "==3.10"
`,
			expected: "3.10",
		},
		{
			name: "requires-python ~=",
			content: `[project]
requires-python = "~=3.9"
`,
			expected: "3.9",
		},
		{
			name: "requires-python with spaces",
			content: `[project]
requires-python  =  ">=3.8"
`,
			expected: "3.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
			err := os.WriteFile(pyprojectFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			version := parsePyprojectToml(tmpDir)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestParsePyprojectToml_Poetry(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "poetry ^",
			content: `[tool.poetry.dependencies]
python = "^3.12"
`,
			expected: "3.12",
		},
		{
			name: "poetry ~",
			content: `[tool.poetry.dependencies]
python = "~3.11"
`,
			expected: "3.11",
		},
		{
			name: "poetry >=",
			content: `[tool.poetry.dependencies]
python = ">=3.10"
`,
			expected: "3.10",
		},
		{
			name: "poetry with spaces",
			content: `[tool.poetry.dependencies]
python  =  "^3.9"
`,
			expected: "3.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
			err := os.WriteFile(pyprojectFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			version := parsePyprojectToml(tmpDir)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestParsePyprojectToml_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	version := parsePyprojectToml(tmpDir)
	assert.Equal(t, "", version)
}

func TestParsePyprojectToml_NoVersionInfo(t *testing.T) {
	tmpDir := t.TempDir()
	pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
	content := `[project]
name = "test-project"
description = "A test project"
`
	err := os.WriteFile(pyprojectFile, []byte(content), 0644)
	require.NoError(t, err)

	version := parsePyprojectToml(tmpDir)
	assert.Equal(t, "", version)
}

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "full version",
			version:  "3.14.0",
			expected: "3.14",
		},
		{
			name:     "major.minor only",
			version:  "3.11",
			expected: "3.11",
		},
		{
			name:     "major only",
			version:  "3",
			expected: "3",
		},
		{
			name:     "empty string",
			version:  "",
			expected: "",
		},
		{
			name:     "with patch and build",
			version:  "3.12.5.final.0",
			expected: "3.12",
		},
		{
			name:     "single digit",
			version:  "3.9",
			expected: "3.9",
		},
		{
			name:     "double digit minor",
			version:  "3.10.1",
			expected: "3.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMajorMinor(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePyprojectToml_ScannerEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with file that has matching line but scanner continues
	pyprojectFile := filepath.Join(tmpDir, "pyproject.toml")
	content := `[project]
name = "test"
# Some comment
requires-python = ">=3.8"
# More content after match
dependencies = ["requests"]
`
	err := os.WriteFile(pyprojectFile, []byte(content), 0644)
	require.NoError(t, err)

	version := parsePyprojectToml(tmpDir)
	assert.Equal(t, "3.8", version)
}
