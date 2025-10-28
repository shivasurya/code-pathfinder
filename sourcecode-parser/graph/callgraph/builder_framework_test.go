package callgraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFrameworkResolution validates that known frameworks are resolved correctly
func TestFrameworkResolution(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tmpDir, "test_frameworks.py")
	testCode := `
import django.db.models as models
from rest_framework import serializers
import pytest
import json
import logging

def test_django():
    # Django ORM call (should be resolved as external framework)
    user = models.User.objects.get(id=1)
    return user

def test_rest_framework():
    # REST framework call (should be resolved as external framework)
    serializer = serializers.ModelSerializer()
    return serializer

def test_pytest():
    # pytest call (should be resolved as external framework)
    fixture = pytest.fixture()
    return fixture

def test_stdlib():
    # stdlib calls (should be resolved as external framework)
    data = json.loads('{}')
    logger = logging.getLogger(__name__)
    return data
`
	err := os.WriteFile(testFile, []byte(testCode), 0644)
	assert.NoError(t, err)

	// Build module registry
	registry, err := BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	// Build import map cache
	cache := NewImportMapCache()
	sourceCode, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	importMap, err := cache.GetOrExtract(testFile, sourceCode, registry)
	assert.NoError(t, err)

	// Get module path
	modulePath, ok := registry.FileToModule[testFile]
	assert.True(t, ok)

	// Test Django models resolution
	targetFQN, resolved := resolveCallTarget("models.User", importMap, registry, modulePath)
	assert.True(t, resolved, "Django models.User should be resolved")
	assert.Equal(t, "django.db.models.User", targetFQN)

	// Test REST framework resolution
	targetFQN, resolved = resolveCallTarget("serializers.ModelSerializer", importMap, registry, modulePath)
	assert.True(t, resolved, "REST framework serializers should be resolved")
	assert.Equal(t, "rest_framework.serializers.ModelSerializer", targetFQN)

	// Test pytest resolution
	targetFQN, resolved = resolveCallTarget("pytest.fixture", importMap, registry, modulePath)
	assert.True(t, resolved, "pytest.fixture should be resolved")
	assert.Equal(t, "pytest.fixture", targetFQN)

	// Test json (stdlib) resolution
	targetFQN, resolved = resolveCallTarget("json.loads", importMap, registry, modulePath)
	assert.True(t, resolved, "json.loads should be resolved")
	assert.Equal(t, "json.loads", targetFQN)

	// Test logging (stdlib) resolution
	targetFQN, resolved = resolveCallTarget("logging.getLogger", importMap, registry, modulePath)
	assert.True(t, resolved, "logging.getLogger should be resolved")
	assert.Equal(t, "logging.getLogger", targetFQN)
}

// TestNonFrameworkResolution ensures non-framework calls still work correctly
func TestNonFrameworkResolution(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create utils module
	utilsFile := filepath.Join(tmpDir, "utils.py")
	utilsCode := `
def sanitize(value):
    return value.strip()

def validate(data):
    return True
`
	err := os.WriteFile(utilsFile, []byte(utilsCode), 0644)
	assert.NoError(t, err)

	// Create test file that imports utils
	testFile := filepath.Join(tmpDir, "test.py")
	testCode := `
from utils import sanitize, validate

def process():
    result = sanitize("  test  ")
    valid = validate(result)
    return valid
`
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	assert.NoError(t, err)

	// Build module registry
	registry, err := BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	// Build import map
	cache := NewImportMapCache()
	sourceCode, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	importMap, err := cache.GetOrExtract(testFile, sourceCode, registry)
	assert.NoError(t, err)

	// Get module path
	modulePath, ok := registry.FileToModule[testFile]
	assert.True(t, ok)

	// Test local function resolution (should resolve to local module)
	targetFQN, resolved := resolveCallTarget("sanitize", importMap, registry, modulePath)
	assert.True(t, resolved, "Local function sanitize should be resolved")
	assert.Contains(t, targetFQN, "utils.sanitize")

	targetFQN, resolved = resolveCallTarget("validate", importMap, registry, modulePath)
	assert.True(t, resolved, "Local function validate should be resolved")
	assert.Contains(t, targetFQN, "utils.validate")
}

// TestFrameworkVsLocalPrecedence ensures local definitions take precedence over frameworks
func TestFrameworkVsLocalPrecedence(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create a local module named "json" (shadowing stdlib)
	jsonFile := filepath.Join(tmpDir, "json.py")
	jsonCode := `
def loads(data):
    return "custom loads"
`
	err := os.WriteFile(jsonFile, []byte(jsonCode), 0644)
	assert.NoError(t, err)

	// Create test file that imports local json
	testFile := filepath.Join(tmpDir, "test.py")
	testCode := `
from json import loads

def process():
    return loads('{}')
`
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	assert.NoError(t, err)

	// Build module registry
	registry, err := BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	// Build import map
	cache := NewImportMapCache()
	sourceCode, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	importMap, err := cache.GetOrExtract(testFile, sourceCode, registry)
	assert.NoError(t, err)

	// Get module path
	modulePath, ok := registry.FileToModule[testFile]
	assert.True(t, ok)

	// Test that local json takes precedence over stdlib
	targetFQN, resolved := resolveCallTarget("loads", importMap, registry, modulePath)
	assert.True(t, resolved, "Local json.loads should be resolved")
	// When there's a local module that shadows stdlib, it resolves to local
	// The FQN will be json.loads but from the local module, not stdlib
	assert.Contains(t, targetFQN, "json.loads", "Should resolve to json.loads")

	// Verify it's actually from local module by checking registry
	_, localExists := registry.Modules[targetFQN[:strings.LastIndex(targetFQN, ".")]]
	assert.True(t, localExists, "Should resolve to local json module in registry")
}

// TestMixedFrameworkAndLocalCalls validates correct resolution in mixed scenarios
func TestMixedFrameworkAndLocalCalls(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create local utils
	utilsFile := filepath.Join(tmpDir, "utils.py")
	utilsCode := `
def helper():
    pass
`
	err := os.WriteFile(utilsFile, []byte(utilsCode), 0644)
	assert.NoError(t, err)

	// Create test file with mixed calls
	testFile := filepath.Join(tmpDir, "test.py")
	testCode := `
import json
from utils import helper

def process():
    # Local call
    helper()
    # Framework call
    data = json.loads('{}')
    return data
`
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	assert.NoError(t, err)

	// Build module registry
	registry, err := BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	// Build import map
	cache := NewImportMapCache()
	sourceCode, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	importMap, err := cache.GetOrExtract(testFile, sourceCode, registry)
	assert.NoError(t, err)

	modulePath, ok := registry.FileToModule[testFile]
	assert.True(t, ok)

	// Test local function resolution
	targetFQN, resolved := resolveCallTarget("helper", importMap, registry, modulePath)
	assert.True(t, resolved, "Local helper should be resolved")
	assert.Contains(t, targetFQN, "utils.helper")

	// Test framework resolution
	targetFQN, resolved = resolveCallTarget("json.loads", importMap, registry, modulePath)
	assert.True(t, resolved, "json.loads should be resolved as framework")
	assert.Equal(t, "json.loads", targetFQN)
}
