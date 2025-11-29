package resolution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRelativeImport_SingleDot(t *testing.T) {
	// Test single dot relative import: from . import module
	// File: myapp/submodule/handler.py (module: myapp.submodule.handler)
	// Import: from . import utils
	// Expected: myapp.submodule.utils

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp.submodule.handler", "/project/myapp/submodule/handler.py")
	registry.AddModule("myapp.submodule.utils", "/project/myapp/submodule/utils.py")

	result := resolveRelativeImport("/project/myapp/submodule/handler.py", 1, "utils", registry)
	assert.Equal(t, "myapp.submodule.utils", result)
}

func TestResolveRelativeImport_SingleDotNoSuffix(t *testing.T) {
	// Test single dot with no suffix: from . import *
	// File: myapp/submodule/handler.py (module: myapp.submodule.handler)
	// Import: from . import *
	// Expected: myapp.submodule

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp.submodule.handler", "/project/myapp/submodule/handler.py")

	result := resolveRelativeImport("/project/myapp/submodule/handler.py", 1, "", registry)
	assert.Equal(t, "myapp.submodule", result)
}

func TestResolveRelativeImport_TwoDots(t *testing.T) {
	// Test two dots relative import: from .. import module
	// File: myapp/submodule/handler.py (module: myapp.submodule.handler)
	// Import: from .. import config
	// Expected: myapp.config

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp.submodule.handler", "/project/myapp/submodule/handler.py")
	registry.AddModule("myapp.config", "/project/myapp/config/__init__.py")

	result := resolveRelativeImport("/project/myapp/submodule/handler.py", 2, "config", registry)
	assert.Equal(t, "myapp.config", result)
}

func TestResolveRelativeImport_TwoDotsNoSuffix(t *testing.T) {
	// Test two dots with no suffix: from .. import *
	// File: myapp/submodule/handler.py (module: myapp.submodule.handler)
	// Import: from .. import *
	// Expected: myapp

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp.submodule.handler", "/project/myapp/submodule/handler.py")

	result := resolveRelativeImport("/project/myapp/submodule/handler.py", 2, "", registry)
	assert.Equal(t, "myapp", result)
}

func TestResolveRelativeImport_ThreeDots(t *testing.T) {
	// Test three dots relative import: from ... import module
	// File: myapp/submodule/deep/handler.py (module: myapp.submodule.deep.handler)
	// Import: from ... import utils
	// Expected: myapp.utils

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp.submodule.deep.handler", "/project/myapp/submodule/deep/handler.py")
	registry.AddModule("myapp.utils", "/project/myapp/utils/__init__.py")

	result := resolveRelativeImport("/project/myapp/submodule/deep/handler.py", 3, "utils", registry)
	assert.Equal(t, "myapp.utils", result)
}

func TestResolveRelativeImport_TooManyDots(t *testing.T) {
	// Test excessive dots (more than hierarchy depth)
	// File: myapp/handler.py (module: myapp.handler)
	// Import: from ... import something (3 dots but only 1 level deep)
	// Expected: something (clamped to root)

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp.handler", "/project/myapp/handler.py")

	result := resolveRelativeImport("/project/myapp/handler.py", 3, "something", registry)
	assert.Equal(t, "something", result)
}

func TestResolveRelativeImport_NotInRegistry(t *testing.T) {
	// Test file not in registry
	// Expected: return suffix as-is

	registry := core.NewModuleRegistry()

	result := resolveRelativeImport("/project/unknown/file.py", 2, "module", registry)
	assert.Equal(t, "module", result)
}

func TestResolveRelativeImport_RootPackage(t *testing.T) {
	// Test relative import from root package file
	// File: myapp/__init__.py (module: myapp)
	// Import: from . import utils
	// Expected: utils (no parent package)

	registry := core.NewModuleRegistry()
	registry.AddModule("myapp", "/project/myapp/__init__.py")

	result := resolveRelativeImport("/project/myapp/__init__.py", 1, "utils", registry)
	assert.Equal(t, "utils", result)
}

func TestExtractImports_RelativeImports(t *testing.T) {
	// Test extraction of relative imports from source code
	sourceCode := []byte(`
from . import utils
from .. import config
from ..utils import helper
from ..config import settings
`)

	// Build registry for the test structure
	registry := core.NewModuleRegistry()
	filePath := "/project/myapp/submodule/handler.py"
	registry.AddModule("myapp.submodule.handler", filePath)
	registry.AddModule("myapp.submodule.utils", "/project/myapp/submodule/utils.py")
	registry.AddModule("myapp.config", "/project/myapp/config/__init__.py")
	registry.AddModule("myapp.utils.helper", "/project/myapp/utils/helper.py")
	registry.AddModule("myapp.config.settings", "/project/myapp/config/settings.py")

	importMap, err := ExtractImports(filePath, sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify relative imports are resolved
	fqn, ok := importMap.Resolve("utils")
	assert.True(t, ok)
	assert.Equal(t, "myapp.submodule.utils", fqn)

	fqn, ok = importMap.Resolve("config")
	assert.True(t, ok)
	assert.Equal(t, "myapp.config", fqn)

	fqn, ok = importMap.Resolve("helper")
	assert.True(t, ok)
	assert.Equal(t, "myapp.utils.helper", fqn)

	fqn, ok = importMap.Resolve("settings")
	assert.True(t, ok)
	assert.Equal(t, "myapp.config.settings", fqn)
}

func TestExtractImports_MixedAbsoluteAndRelative(t *testing.T) {
	// Test mixing absolute and relative imports
	sourceCode := []byte(`
import os
from sys import argv
from . import utils
from ..config import settings
`)

	registry := core.NewModuleRegistry()
	filePath := "/project/myapp/submodule/handler.py"
	registry.AddModule("myapp.submodule.handler", filePath)
	registry.AddModule("myapp.submodule.utils", "/project/myapp/submodule/utils.py")
	registry.AddModule("myapp.config.settings", "/project/myapp/config/settings.py")

	importMap, err := ExtractImports(filePath, sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Absolute imports
	fqn, ok := importMap.Resolve("os")
	assert.True(t, ok)
	assert.Equal(t, "os", fqn)

	fqn, ok = importMap.Resolve("argv")
	assert.True(t, ok)
	assert.Equal(t, "sys.argv", fqn)

	// Relative imports
	fqn, ok = importMap.Resolve("utils")
	assert.True(t, ok)
	assert.Equal(t, "myapp.submodule.utils", fqn)

	fqn, ok = importMap.Resolve("settings")
	assert.True(t, ok)
	assert.Equal(t, "myapp.config.settings", fqn)
}

func TestExtractImports_WithTestFixture_RelativeImports(t *testing.T) {
	// Build module registry for the test fixture - use absolute path from start
	// Note: This file is now in resolution/ subpackage, so we need one extra ..
	projectRoot := filepath.Join("..", "..", "..", "test-fixtures", "python", "relative_imports_test")
	absProjectRoot, err := filepath.Abs(projectRoot)
	require.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(absProjectRoot)
	require.NoError(t, err)

	// Test with actual fixture file - construct from absolute project root
	fixturePath := filepath.Join(absProjectRoot, "myapp", "submodule", "handler.py")

	// Check if fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Fixture file not found: %s", fixturePath)
	}

	sourceCode, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	importMap, err := ExtractImports(fixturePath, sourceCode, modRegistry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Expected imports based on handler.py content:
	// from . import utils -> myapp.submodule.utils
	// from .. import config -> myapp.config
	// from ..utils import helper -> myapp.utils.helper
	// from ..config import settings -> myapp.config.settings

	expectedImports := map[string]string{
		"utils":    "myapp.submodule.utils",
		"config":   "myapp.config",
		"helper":   "myapp.utils.helper",
		"settings": "myapp.config.settings",
	}

	assert.Equal(t, len(expectedImports), len(importMap.Imports),
		"Expected %d imports, got %d", len(expectedImports), len(importMap.Imports))

	for alias, expectedFQN := range expectedImports {
		fqn, ok := importMap.Resolve(alias)
		assert.True(t, ok, "Expected import alias '%s' not found", alias)
		assert.Equal(t, expectedFQN, fqn,
			"Import '%s' should resolve to '%s', got '%s'", alias, expectedFQN, fqn)
	}
}

func TestResolveRelativeImport_NestedPackages(t *testing.T) {
	// Test deeply nested package hierarchy
	tests := []struct {
		name         string
		filePath     string
		modulePath   string
		dotCount     int
		moduleSuffix string
		expected     string
	}{
		{
			name:         "Deep nesting - single dot",
			filePath:     "/project/a/b/c/d/file.py",
			modulePath:   "a.b.c.d.file",
			dotCount:     1,
			moduleSuffix: "utils",
			expected:     "a.b.c.d.utils",
		},
		{
			name:         "Deep nesting - four dots",
			filePath:     "/project/a/b/c/d/file.py",
			modulePath:   "a.b.c.d.file",
			dotCount:     4,
			moduleSuffix: "utils",
			expected:     "a.utils",
		},
		{
			name:         "Deep nesting - three dots",
			filePath:     "/project/a/b/c/file.py",
			modulePath:   "a.b.c.file",
			dotCount:     3,
			moduleSuffix: "utils",
			expected:     "a.utils",
		},
		{
			name:         "Deep nesting - four dots (exceeds hierarchy)",
			filePath:     "/project/a/b/c/file.py",
			modulePath:   "a.b.c.file",
			dotCount:     4,
			moduleSuffix: "utils",
			expected:     "utils", // Clamped to root
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := core.NewModuleRegistry()
			registry.AddModule(tt.modulePath, tt.filePath)

			result := resolveRelativeImport(tt.filePath, tt.dotCount, tt.moduleSuffix, registry)
			assert.Equal(t, tt.expected, result)
		})
	}
}
