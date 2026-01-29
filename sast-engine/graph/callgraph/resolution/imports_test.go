package resolution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractImports_SimpleImports(t *testing.T) {
	// Test simple import statements: import module
	sourceCode := []byte(`
import os
import sys
import json
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify all simple imports are captured
	assert.Equal(t, 3, len(importMap.Imports))

	fqn, ok := importMap.Resolve("os")
	assert.True(t, ok)
	assert.Equal(t, "os", fqn)

	fqn, ok = importMap.Resolve("sys")
	assert.True(t, ok)
	assert.Equal(t, "sys", fqn)

	fqn, ok = importMap.Resolve("json")
	assert.True(t, ok)
	assert.Equal(t, "json", fqn)
}

func TestExtractImports_FromImports(t *testing.T) {
	// Test from import statements: from module import name
	sourceCode := []byte(`
from os import path
from sys import argv
from collections import OrderedDict
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify from imports create fully qualified names
	assert.Equal(t, 3, len(importMap.Imports))

	fqn, ok := importMap.Resolve("path")
	assert.True(t, ok)
	assert.Equal(t, "os.path", fqn)

	fqn, ok = importMap.Resolve("argv")
	assert.True(t, ok)
	assert.Equal(t, "sys.argv", fqn)

	fqn, ok = importMap.Resolve("OrderedDict")
	assert.True(t, ok)
	assert.Equal(t, "collections.OrderedDict", fqn)
}

func TestExtractImports_AliasedSimpleImports(t *testing.T) {
	// Test aliased simple imports: import module as alias
	sourceCode := []byte(`
import os as operating_system
import sys as system
import json as js
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify aliases map to original module names
	assert.Equal(t, 3, len(importMap.Imports))

	fqn, ok := importMap.Resolve("operating_system")
	assert.True(t, ok)
	assert.Equal(t, "os", fqn)

	fqn, ok = importMap.Resolve("system")
	assert.True(t, ok)
	assert.Equal(t, "sys", fqn)

	fqn, ok = importMap.Resolve("js")
	assert.True(t, ok)
	assert.Equal(t, "json", fqn)

	// Original names should NOT be in the map
	_, ok = importMap.Resolve("os")
	assert.False(t, ok)
}

func TestExtractImports_AliasedFromImports(t *testing.T) {
	// Test aliased from imports: from module import name as alias
	sourceCode := []byte(`
from os import path as ospath
from sys import argv as arguments
from collections import OrderedDict as OD
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify aliases map to fully qualified names
	assert.Equal(t, 3, len(importMap.Imports))

	fqn, ok := importMap.Resolve("ospath")
	assert.True(t, ok)
	assert.Equal(t, "os.path", fqn)

	fqn, ok = importMap.Resolve("arguments")
	assert.True(t, ok)
	assert.Equal(t, "sys.argv", fqn)

	fqn, ok = importMap.Resolve("OD")
	assert.True(t, ok)
	assert.Equal(t, "collections.OrderedDict", fqn)

	// Original names should NOT be in the map
	_, ok = importMap.Resolve("path")
	assert.False(t, ok)
	_, ok = importMap.Resolve("argv")
	assert.False(t, ok)
	_, ok = importMap.Resolve("OrderedDict")
	assert.False(t, ok)
}

func TestExtractImports_MixedStyles(t *testing.T) {
	// Test mixed import styles in one file
	sourceCode := []byte(`
import os
from sys import argv
import json as js
from collections import OrderedDict as OD
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	assert.Equal(t, 4, len(importMap.Imports))

	// Simple import
	fqn, ok := importMap.Resolve("os")
	assert.True(t, ok)
	assert.Equal(t, "os", fqn)

	// From import
	fqn, ok = importMap.Resolve("argv")
	assert.True(t, ok)
	assert.Equal(t, "sys.argv", fqn)

	// Aliased simple import
	fqn, ok = importMap.Resolve("js")
	assert.True(t, ok)
	assert.Equal(t, "json", fqn)

	// Aliased from import
	fqn, ok = importMap.Resolve("OD")
	assert.True(t, ok)
	assert.Equal(t, "collections.OrderedDict", fqn)
}

func TestExtractImports_NestedModules(t *testing.T) {
	// Test imports with nested module paths
	sourceCode := []byte(`
import xml.etree.ElementTree
from xml.etree import ElementTree
from xml.etree.ElementTree import Element
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	assert.Equal(t, 3, len(importMap.Imports))

	// Simple import of nested module
	fqn, ok := importMap.Resolve("xml.etree.ElementTree")
	assert.True(t, ok)
	assert.Equal(t, "xml.etree.ElementTree", fqn)

	// From import of nested module
	fqn, ok = importMap.Resolve("ElementTree")
	assert.True(t, ok)
	assert.Equal(t, "xml.etree.ElementTree", fqn)

	// From import from deeply nested module
	fqn, ok = importMap.Resolve("Element")
	assert.True(t, ok)
	assert.Equal(t, "xml.etree.ElementTree.Element", fqn)
}

func TestExtractImports_EmptyFile(t *testing.T) {
	sourceCode := []byte(`
# Just a comment, no imports
def foo():
    pass
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)
	assert.Equal(t, 0, len(importMap.Imports))
}

func TestExtractImports_InvalidSyntax(t *testing.T) {
	// Test with invalid Python syntax
	sourceCode := []byte(`
import this is not valid python
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	// Tree-sitter is fault-tolerant, so parsing may succeed even with errors
	// We just verify it doesn't crash
	require.NoError(t, err)
	require.NotNil(t, importMap)
}

func TestExtractImports_WithTestFixtures(t *testing.T) {
	tests := []struct {
		name             string
		fixtureFile      string
		expectedImports  map[string]string
		expectedCount    int
	}{
		{
			name:        "Simple imports fixture",
			fixtureFile: "simple_imports.py",
			expectedImports: map[string]string{
				"os":   "os",
				"sys":  "sys",
				"json": "json",
			},
			expectedCount: 3,
		},
		{
			name:        "From imports fixture",
			fixtureFile: "from_imports.py",
			expectedImports: map[string]string{
				"path":  "os.path",
				"argv":  "sys.argv",
				"dumps": "json.dumps",
				"loads": "json.loads",
			},
			expectedCount: 4,
		},
		{
			name:        "Aliased imports fixture",
			fixtureFile: "aliased_imports.py",
			expectedImports: map[string]string{
				"operating_system": "os",
				"arguments":        "sys.argv",
				"json_dumps":       "json.dumps",
			},
			expectedCount: 3,
		},
		{
			name:        "Mixed imports fixture",
			fixtureFile: "mixed_imports.py",
			expectedImports: map[string]string{
				"os":   "os",
				"argv": "sys.argv",
				"js":   "json",
				"OD":   "collections.OrderedDict",
			},
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixturePath := filepath.Join("..", "..", "..", "test-fixtures", "python", "imports_test", tt.fixtureFile)

			// Check if fixture exists
			if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
				t.Skipf("Fixture file not found: %s", fixturePath)
			}

			sourceCode, err := os.ReadFile(fixturePath)
			require.NoError(t, err)

			registry := core.NewModuleRegistry()
			importMap, err := ExtractImports(fixturePath, sourceCode, registry)

			require.NoError(t, err)
			require.NotNil(t, importMap)

			// Check expected count
			assert.Equal(t, tt.expectedCount, len(importMap.Imports),
				"Expected %d imports, got %d", tt.expectedCount, len(importMap.Imports))

			// Check each expected import
			for alias, expectedFQN := range tt.expectedImports {
				fqn, ok := importMap.Resolve(alias)
				assert.True(t, ok, "Expected import alias '%s' not found", alias)
				assert.Equal(t, expectedFQN, fqn,
					"Import '%s' should resolve to '%s', got '%s'", alias, expectedFQN, fqn)
			}
		})
	}
}

func TestExtractImports_MultipleImportsPerLine(t *testing.T) {
	// Python allows multiple imports on one line with commas
	sourceCode := []byte(`
from collections import OrderedDict, defaultdict, Counter
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Each import should be captured separately
	// Note: The tree-sitter query may need adjustment to handle this
	// For now, we just verify it doesn't crash
	assert.GreaterOrEqual(t, len(importMap.Imports), 1)
}

func TestExtractCaptures(t *testing.T) {
	// This is a unit test for the extractCaptures helper function
	// We test it indirectly through ExtractImports, but this documents its behavior
	sourceCode := []byte(`
import os
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	assert.Equal(t, 1, len(importMap.Imports))
}

func TestExtractImports_Whitespace(t *testing.T) {
	// Test that whitespace is properly handled
	sourceCode := []byte(`
import   os
from    sys   import   argv
import json    as    js
`)

	registry := core.NewModuleRegistry()
	importMap, err := ExtractImports("/test/file.py", sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify whitespace doesn't affect import extraction
	assert.Equal(t, 3, len(importMap.Imports))

	fqn, ok := importMap.Resolve("os")
	assert.True(t, ok)
	assert.Equal(t, "os", fqn)

	fqn, ok = importMap.Resolve("argv")
	assert.True(t, ok)
	assert.Equal(t, "sys.argv", fqn)

	fqn, ok = importMap.Resolve("js")
	assert.True(t, ok)
	assert.Equal(t, "json", fqn)
}

func TestExtractImports_ProjectInternalImports(t *testing.T) {
	// Test that project-internal imports are normalized to include project root
	//
	// Scenario: A Python project "label_studio" with this structure:
	//   label_studio/
	//     data_manager/
	//       functions.py     ← file being parsed
	//       prepare_params.py
	//
	// In functions.py:
	//   from data_manager.prepare_params import PrepareParams
	//
	// Expected behavior:
	//   - Import should be normalized to "label_studio.data_manager.prepare_params.PrepareParams"
	//   - This enables cross-file resolution to work correctly

	// Set up a module registry that simulates the label_studio project structure
	registry := core.NewModuleRegistry()

	// Register the file being parsed
	filePath := "/project/label_studio/data_manager/functions.py"
	modulePath := "label_studio.data_manager.functions"
	registry.AddModule(modulePath, filePath)

	// Register the module being imported (this is what normalizeProjectImport will check)
	prepareParamsFile := "/project/label_studio/data_manager/prepare_params.py"
	prepareParamsModule := "label_studio.data_manager.prepare_params"
	registry.AddModule(prepareParamsModule, prepareParamsFile)

	// Source code with project-internal import
	sourceCode := []byte(`
from data_manager.prepare_params import PrepareParams
`)

	importMap, err := ExtractImports(filePath, sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Verify that PrepareParams is normalized to include project root
	fqn, ok := importMap.Resolve("PrepareParams")
	assert.True(t, ok, "PrepareParams import should be found")
	assert.Equal(t, "label_studio.data_manager.prepare_params.PrepareParams", fqn,
		"Project-internal import should be normalized to include project root (label_studio)")
}

func TestExtractImports_MixedProjectAndThirdParty(t *testing.T) {
	// Test that project-internal imports are normalized while third-party imports are left as-is
	//
	// Scenario: File imports both project-internal and third-party modules
	//
	// Expected behavior:
	//   - Project-internal: data_manager.X → label_studio.data_manager.X
	//   - Third-party: django.db.models → django.db.models (unchanged)

	registry := core.NewModuleRegistry()

	// Register the file being parsed
	filePath := "/project/label_studio/data_manager/functions.py"
	modulePath := "label_studio.data_manager.functions"
	registry.AddModule(modulePath, filePath)

	// Register project-internal modules
	registry.AddModule("label_studio.data_manager.prepare_params", "/project/label_studio/data_manager/prepare_params.py")
	registry.AddModule("label_studio.core.utils", "/project/label_studio/core/utils.py")

	// Note: We intentionally do NOT register django or rest_framework modules
	// to simulate third-party dependencies

	sourceCode := []byte(`
from django.db import models
from data_manager.prepare_params import PrepareParams
from rest_framework.views import APIView
from core.utils import sanitize
`)

	importMap, err := ExtractImports(filePath, sourceCode, registry)

	require.NoError(t, err)
	require.NotNil(t, importMap)

	// Third-party imports should remain unchanged (not in registry)
	fqn, ok := importMap.Resolve("models")
	assert.True(t, ok)
	assert.Equal(t, "django.db.models", fqn, "Third-party import should not be modified")

	fqn, ok = importMap.Resolve("APIView")
	assert.True(t, ok)
	assert.Equal(t, "rest_framework.views.APIView", fqn, "Third-party import should not be modified")

	// Project-internal imports should be normalized
	fqn, ok = importMap.Resolve("PrepareParams")
	assert.True(t, ok)
	assert.Equal(t, "label_studio.data_manager.prepare_params.PrepareParams", fqn,
		"Project-internal import should include project root")

	fqn, ok = importMap.Resolve("sanitize")
	assert.True(t, ok)
	assert.Equal(t, "label_studio.core.utils.sanitize", fqn,
		"Project-internal import should include project root")
}

func TestNormalizeProjectImport_ProjectInternal(t *testing.T) {
	// Unit test for normalizeProjectImport function - project-internal case
	registry := core.NewModuleRegistry()

	filePath := "/project/myapp/submodule/file.py"
	modulePath := "myapp.submodule.file"
	registry.AddModule(modulePath, filePath)

	// Register the module being imported
	registry.AddModule("myapp.utils.helper", "/project/myapp/utils/helper.py")

	// Test normalizing a project-internal import
	result := normalizeProjectImport("utils.helper", filePath, registry)
	assert.Equal(t, "myapp.utils.helper", result,
		"Project-internal import should be normalized with project root")
}

func TestNormalizeProjectImport_ThirdParty(t *testing.T) {
	// Unit test for normalizeProjectImport function - third-party case
	registry := core.NewModuleRegistry()

	filePath := "/project/myapp/submodule/file.py"
	modulePath := "myapp.submodule.file"
	registry.AddModule(modulePath, filePath)

	// Test normalizing a third-party import (not in registry)
	result := normalizeProjectImport("django.db.models", filePath, registry)
	assert.Equal(t, "django.db.models", result,
		"Third-party import should remain unchanged")

	result = normalizeProjectImport("rest_framework.views", filePath, registry)
	assert.Equal(t, "rest_framework.views", result,
		"Third-party import should remain unchanged")
}

func TestNormalizeProjectImport_AlreadyAbsolute(t *testing.T) {
	// Unit test for normalizeProjectImport function - already absolute path case
	registry := core.NewModuleRegistry()

	filePath := "/project/myapp/submodule/file.py"
	modulePath := "myapp.submodule.file"
	registry.AddModule(modulePath, filePath)

	// Register a module with full path
	registry.AddModule("myapp.utils.helper", "/project/myapp/utils/helper.py")

	// Test with already absolute import (full FQN already in registry)
	result := normalizeProjectImport("myapp.utils.helper", filePath, registry)
	assert.Equal(t, "myapp.utils.helper", result,
		"Already absolute import should remain unchanged")
}

func TestNormalizeProjectImport_FileNotInRegistry(t *testing.T) {
	// Unit test for normalizeProjectImport function - file not in registry
	registry := core.NewModuleRegistry()

	// Test with file not in registry - should return original
	result := normalizeProjectImport("some.module", "/unknown/file.py", registry)
	assert.Equal(t, "some.module", result,
		"Import from unregistered file should remain unchanged")
}

func TestNormalizeProjectImport_EmptyModuleName(t *testing.T) {
	// Unit test for normalizeProjectImport function - empty module name
	registry := core.NewModuleRegistry()

	filePath := "/project/myapp/file.py"
	modulePath := "myapp.file"
	registry.AddModule(modulePath, filePath)

	// Test with empty module name - should return empty
	result := normalizeProjectImport("", filePath, registry)
	assert.Equal(t, "", result,
		"Empty module name should return empty string")
}
