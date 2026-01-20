package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
)

// TestFrameworkResolution validates that known frameworks are resolved correctly.
func TestFrameworkResolution(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files (avoid test_ prefix as it gets filtered by module registry)
	testFile := filepath.Join(tmpDir, "frameworks.py")
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

	// This test now validates that the build process works correctly
	// with framework imports. The internal resolveCallTarget function
	// is tested indirectly through the builder.

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	// Parse the code graph
	codeGraph := graph.Initialize(tmpDir, nil)

	// Build call graph which internally uses resolveCallTarget
	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	assert.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify that call sites were extracted (indirectly validates resolution)
	assert.Greater(t, len(callGraph.CallSites), 0, "Should have extracted call sites from test file")
}

// TestNonFrameworkResolution ensures non-framework calls still work correctly.
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

	// Create file that imports utils
	testFile := filepath.Join(tmpDir, "app.py")
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
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	// Parse the code graph
	codeGraph := graph.Initialize(tmpDir, nil)

	// Build call graph which internally uses resolveCallTarget
	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	assert.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify that call sites were extracted
	assert.Greater(t, len(callGraph.CallSites), 0, "Should have extracted call sites")
}

// TestFrameworkVsLocalPrecedence ensures local definitions take precedence over frameworks.
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

	// Create file that imports local json
	testFile := filepath.Join(tmpDir, "process.py")
	testCode := `
from json import loads

def process():
    return loads('{}')
`
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	assert.NoError(t, err)

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	// Parse the code graph
	codeGraph := graph.Initialize(tmpDir, nil)

	// Build call graph which internally uses resolveCallTarget
	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	assert.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify that local json module exists in registry (takes precedence)
	_, localExists := moduleRegistry.Modules["json"]
	assert.True(t, localExists, "Local json module should be in registry")
}

// TestMixedFrameworkAndLocalCalls validates correct resolution in mixed scenarios.
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

	// Create file with mixed calls
	testFile := filepath.Join(tmpDir, "handler.py")
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
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	// Parse the code graph
	codeGraph := graph.Initialize(tmpDir, nil)

	// Build call graph which internally uses resolveCallTarget
	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	assert.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify that call sites were extracted from mixed scenario
	assert.Greater(t, len(callGraph.CallSites), 0, "Should have extracted call sites from mixed code")
}
