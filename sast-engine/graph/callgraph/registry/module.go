package registry

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// skipDirs lists directory names that should be excluded during module registry building.
// These are typically build artifacts, virtual environments, and version control directories.
var skipDirs = map[string]bool{
	"__pycache__":   true,
	"venv":          true,
	"env":           true,
	".venv":         true,
	".env":          true,
	"node_modules":  true,
	".git":          true,
	".svn":          true,
	"dist":          true,
	"build":         true,
	"_build":        true,
	".eggs":         true,
	"*.egg-info":    true,
	".tox":          true,
	".pytest_cache": true,
	".mypy_cache":   true,
	".coverage":     true,
	"htmlcov":       true,
}

// BuildModuleRegistry walks a directory tree and builds a complete module registry.
// It discovers all Python files and maps them to their corresponding module paths.
//
// The registry enables:
//   - Resolving fully qualified names (FQNs) for functions
//   - Mapping import statements to actual files
//   - Detecting ambiguous module names
//
// Algorithm:
//  1. Walk directory tree recursively
//  2. Skip common non-source directories (venv, __pycache__, etc.)
//  3. Convert file paths to Python module paths
//  4. Index both full module paths and short names
//
// Parameters:
//   - rootPath: absolute path to the project root directory
//
// Returns:
//   - *core.ModuleRegistry: populated registry with all discovered modules
//   - error: if root path doesn't exist or is inaccessible
//
// Example:
//
//	registry, err := BuildModuleRegistry("/path/to/myapp")
//	// Discovers:
//	//   /path/to/myapp/views.py → "myapp.views"
//	//   /path/to/myapp/utils/helpers.py → "myapp.utils.helpers"
func BuildModuleRegistry(rootPath string) (*core.ModuleRegistry, error) {
	registry := core.NewModuleRegistry()

	// Verify root path exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, err
	}

	// Get absolute path to ensure consistency
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		// This error is practically impossible to trigger in normal operation
		// Would require corrupted OS state or invalid memory
		return nil, err // nolint:wrapcheck // Defensive check, untestable
	}

	// Walk directory tree
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that should be excluded
		if info.IsDir() {
			if shouldSkipDirectory(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process Python files
		if !strings.HasSuffix(path, ".py") {
			return nil
		}

		// Convert file path to module path
		modulePath, convertErr := convertToModulePath(path, absRoot)
		if convertErr != nil {
			// Skip files that can't be converted (e.g., outside project)
			// We intentionally ignore this error and continue walking
			//nolint:nilerr // Returning nil continues filepath.Walk
			return nil
		}

		// Register the module
		registry.AddModule(modulePath, path)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return registry, nil
}

// convertToModulePath converts a file system path to a Python module path.
//
// Conversion rules:
//  1. Remove root path prefix
//  2. Remove .py extension
//  3. Remove __init__ suffix (package __init__.py files)
//  4. Replace path separators with dots
//
// Parameters:
//   - filePath: absolute path to a Python file
//   - rootPath: absolute path to the project root
//
// Returns:
//   - string: Python module path (e.g., "myapp.utils.helpers")
//   - error: if filePath is not under rootPath
//
// Examples:
//
//	"/project/myapp/views.py", "/project"
//	  → "myapp.views"
//
//	"/project/myapp/utils/__init__.py", "/project"
//	  → "myapp.utils"
//
//	"/project/myapp/utils/helpers.py", "/project"
//	  → "myapp.utils.helpers"
func convertToModulePath(filePath, rootPath string) (string, error) {
	// Ensure both paths are absolute
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		// Defensive error check - practically impossible to trigger
		return "", err // nolint:wrapcheck // Untestable OS error
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		// Defensive error check - practically impossible to trigger
		return "", err // nolint:wrapcheck // Untestable OS error
	}

	// Get relative path from root
	relPath, err := filepath.Rel(absRoot, absFile)
	if err != nil {
		return "", err
	}

	// Remove .py extension
	relPath = strings.TrimSuffix(relPath, ".py")

	// Handle __init__.py files (they represent the package itself)
	// e.g., "myapp/utils/__init__" → "myapp.utils"
	relPath = strings.TrimSuffix(relPath, string(filepath.Separator)+"__init__")
	relPath = strings.TrimSuffix(relPath, "__init__")

	// Convert path separators to dots
	// On Windows: backslashes → dots
	// On Unix: forward slashes → dots
	modulePath := filepath.ToSlash(relPath) // Normalize to forward slashes
	modulePath = strings.ReplaceAll(modulePath, "/", ".")

	return modulePath, nil
}

// shouldSkipDirectory determines if a directory should be excluded from scanning.
//
// Skipped directories include:
//   - Virtual environments (venv, env, .venv)
//   - Build artifacts (__pycache__, dist, build)
//   - Version control (.git, .svn)
//   - Testing artifacts (.pytest_cache, .tox, .coverage)
//   - Package metadata (.eggs, *.egg-info)
//
// This significantly improves performance by avoiding:
//   - Scanning thousands of dependency files in venv
//   - Processing bytecode in __pycache__
//   - Indexing build artifacts
//
// Parameters:
//   - dirName: the basename of the directory (not full path)
//
// Returns:
//   - bool: true if directory should be skipped
//
// Example:
//
//	shouldSkipDirectory("venv") → true
//	shouldSkipDirectory("myapp") → false
//	shouldSkipDirectory("__pycache__") → true
func shouldSkipDirectory(dirName string) bool {
	return skipDirs[dirName]
}
