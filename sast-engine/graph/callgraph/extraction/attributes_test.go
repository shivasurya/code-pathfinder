package extraction

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractClassAttributes_DottedCallPlaceholder(t *testing.T) {
	source := []byte(`
import sqlite3
import configparser

class DbWrapper:
    def __init__(self, path):
        self.conn = sqlite3.connect(path)

class ConfigWrapper:
    def __init__(self):
        self.parser = configparser.ConfigParser()

class SimpleWrapper:
    def __init__(self):
        self.data = {}
        self.name = "test"
        self.items = []
`)

	moduleRegistry := core.NewModuleRegistry()
	typeEngine := resolution.NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Attributes = registry.NewAttributeRegistry()

	err := ExtractClassAttributes("test.py", source, "test_module", typeEngine, typeEngine.Attributes)
	require.NoError(t, err)

	// Dotted call: sqlite3.connect → call:sqlite3.connect
	attr := typeEngine.Attributes.GetAttribute("test_module.DbWrapper", "conn")
	require.NotNil(t, attr, "conn attribute should be extracted")
	assert.Equal(t, "call:sqlite3.connect", attr.Type.TypeFQN)

	// Dotted constructor: configparser.ConfigParser → call:configparser.ConfigParser
	attr = typeEngine.Attributes.GetAttribute("test_module.ConfigWrapper", "parser")
	require.NotNil(t, attr, "parser attribute should be extracted")
	assert.Equal(t, "call:configparser.ConfigParser", attr.Type.TypeFQN)

	// Literal dict
	attr = typeEngine.Attributes.GetAttribute("test_module.SimpleWrapper", "data")
	require.NotNil(t, attr)
	assert.Equal(t, "builtins.dict", attr.Type.TypeFQN)
}
