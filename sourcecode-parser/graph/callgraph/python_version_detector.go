package callgraph

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// detectPythonVersion infers Python version from project files.
// It checks in order:
//  1. .python-version file
//  2. pyproject.toml [tool.poetry.dependencies] or [project] requires-python
//  3. Defaults to "3.14"
//
// Parameters:
//   - projectPath: absolute path to the project root
//
// Returns:
//   - Python version string (e.g., "3.14", "3.11", "3.9")
func detectPythonVersion(projectPath string) string {
	// 1. Check .python-version file
	if version := readPythonVersionFile(projectPath); version != "" {
		return version
	}

	// 2. Check pyproject.toml
	if version := parsePyprojectToml(projectPath); version != "" {
		return version
	}

	// 3. Default to 3.14
	return "3.14"
}

// readPythonVersionFile reads version from .python-version file.
// Format: "3.14.0" or "3.14" (we extract major.minor)
//
// Parameters:
//   - projectPath: absolute path to the project root
//
// Returns:
//   - Python version string (e.g., "3.14"), or empty string if not found
func readPythonVersionFile(projectPath string) string {
	versionFile := filepath.Join(projectPath, ".python-version")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return ""
	}

	version := strings.TrimSpace(string(data))
	return extractMajorMinor(version)
}

// parsePyprojectToml extracts Python version from pyproject.toml.
// Supports:
//   - [project] requires-python = ">=3.11"
//   - [tool.poetry.dependencies] python = "^3.11"
//
// Parameters:
//   - projectPath: absolute path to the project root
//
// Returns:
//   - Python version string (e.g., "3.11"), or empty string if not found
func parsePyprojectToml(projectPath string) string {
	tomlFile := filepath.Join(projectPath, "pyproject.toml")
	file, err := os.Open(tomlFile)
	if err != nil {
		return ""
	}
	defer file.Close()

	// Patterns to match:
	// requires-python = ">=3.11"
	// python = "^3.11"
	// python = "~3.11"
	requiresPythonRe := regexp.MustCompile(`requires-python\s*=\s*"[><=~^]*(\d+\.\d+)`)
	poetryPythonRe := regexp.MustCompile(`python\s*=\s*"[\^~>=<]*(\d+\.\d+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check requires-python pattern
		if matches := requiresPythonRe.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1]
		}

		// Check poetry python pattern
		if matches := poetryPythonRe.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// extractMajorMinor extracts major.minor version from full version string.
// Examples:
//   - "3.14.0" -> "3.14"
//   - "3.11" -> "3.11"
//   - "3" -> "3"
//
// Parameters:
//   - version: full version string
//
// Returns:
//   - major.minor version string, or original if no dots found
func extractMajorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return ""
}
