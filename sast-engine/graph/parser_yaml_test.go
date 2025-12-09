package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseYAML_ValidFile(t *testing.T) {
	// Create temp YAML file
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yaml")
	content := `
version: "3.8"
services:
  web:
    image: nginx
    ports:
      - "80:80"
`
	err := os.WriteFile(yamlPath, []byte(content), 0644)
	assert.NoError(t, err)

	graph, err := ParseYAML(yamlPath)
	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, yamlPath, graph.FilePath)
	assert.NotNil(t, graph.Root)
}

func TestParseYAML_FileNotFound(t *testing.T) {
	graph, err := ParseYAML("/nonexistent/file.yaml")
	assert.Error(t, err)
	assert.Nil(t, graph)
	assert.Contains(t, err.Error(), "failed to read YAML file")
}

func TestParseYAMLString_Valid(t *testing.T) {
	yaml := `
key: value
number: 42
bool: true
`
	graph, err := ParseYAMLString(yaml)
	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, "docker-compose.yml", graph.FilePath)
}

func TestParseYAMLString_WithCustomPath(t *testing.T) {
	yaml := `key: value`
	graph, err := ParseYAMLString(yaml, "custom.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "custom.yaml", graph.FilePath)
}

func TestParseYAMLString_Invalid(t *testing.T) {
	yaml := `
invalid: yaml: content:
  - unbalanced
  brackets: [
`
	graph, err := ParseYAMLString(yaml)
	assert.Error(t, err)
	assert.Nil(t, graph)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestConvertToYAMLNode_Nil(t *testing.T) {
	node := convertToYAMLNode(nil)
	assert.NotNil(t, node)
	assert.Equal(t, "scalar", node.Type)
	assert.Nil(t, node.Value)
}

func TestConvertToYAMLNode_StringScalar(t *testing.T) {
	node := convertToYAMLNode("test")
	assert.Equal(t, "scalar", node.Type)
	assert.Equal(t, "test", node.Value)
}

func TestConvertToYAMLNode_IntScalar(t *testing.T) {
	node := convertToYAMLNode(42)
	assert.Equal(t, "scalar", node.Type)
	assert.Equal(t, 42, node.Value)
}

func TestConvertToYAMLNode_BoolScalar(t *testing.T) {
	node := convertToYAMLNode(true)
	assert.Equal(t, "scalar", node.Type)
	assert.Equal(t, true, node.Value)
}

func TestConvertToYAMLNode_MapStringInterface(t *testing.T) {
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	node := convertToYAMLNode(data)
	assert.Equal(t, "mapping", node.Type)
	assert.NotNil(t, node.Children)
	assert.Equal(t, 2, len(node.Children))
	assert.Equal(t, "value1", node.Children["key1"].Value)
	assert.Equal(t, 42, node.Children["key2"].Value)
}

func TestConvertToYAMLNode_MapInterfaceInterface(t *testing.T) {
	data := map[interface{}]interface{}{
		"key1": "value1",
		"key2": 42,
		123:    "numeric key",
	}
	node := convertToYAMLNode(data)
	assert.Equal(t, "mapping", node.Type)
	assert.NotNil(t, node.Children)
	assert.Equal(t, 3, len(node.Children))
	assert.Equal(t, "value1", node.Children["key1"].Value)
	assert.Equal(t, 42, node.Children["key2"].Value)
	assert.Equal(t, "numeric key", node.Children["123"].Value)
}

func TestConvertToYAMLNode_Sequence(t *testing.T) {
	data := []interface{}{"item1", "item2", 42}
	node := convertToYAMLNode(data)
	assert.Equal(t, "sequence", node.Type)
	assert.Equal(t, data, node.Value)
}

func TestConvertToYAMLNode_NestedStructure(t *testing.T) {
	data := map[string]interface{}{
		"services": map[string]interface{}{
			"web": map[string]interface{}{
				"image": "nginx",
				"ports": []interface{}{"80:80", "443:443"},
			},
		},
	}
	node := convertToYAMLNode(data)
	assert.Equal(t, "mapping", node.Type)

	services := node.Children["services"]
	assert.Equal(t, "mapping", services.Type)

	web := services.Children["web"]
	assert.Equal(t, "mapping", web.Type)
	assert.Equal(t, "nginx", web.Children["image"].Value)

	ports := web.Children["ports"]
	assert.Equal(t, "sequence", ports.Type)
}

func TestYAMLGraph_Query(t *testing.T) {
	yaml := `
version: "3.8"
services:
  web:
    image: nginx
`
	graph, _ := ParseYAMLString(yaml)

	versionNode := graph.Query("version")
	assert.NotNil(t, versionNode)
	assert.Equal(t, "3.8", versionNode.Value)

	servicesNode := graph.Query("services")
	assert.NotNil(t, servicesNode)
	assert.Equal(t, "mapping", servicesNode.Type)
}

func TestYAMLGraph_Query_NotFound(t *testing.T) {
	yaml := `key: value`
	graph, _ := ParseYAMLString(yaml)

	node := graph.Query("nonexistent")
	assert.Nil(t, node)
}

func TestYAMLGraph_Query_NilRoot(t *testing.T) {
	graph := &YAMLGraph{Root: nil}
	node := graph.Query("anything")
	assert.Nil(t, node)
}

func TestYAMLGraph_Query_NoChildren(t *testing.T) {
	graph := &YAMLGraph{
		Root: &YAMLNode{Type: "scalar", Value: "test"},
	}
	node := graph.Query("anything")
	assert.Nil(t, node)
}

func TestYAMLNode_HasChild(t *testing.T) {
	node := &YAMLNode{
		Type: "mapping",
		Children: map[string]*YAMLNode{
			"key1": {Type: "scalar", Value: "value1"},
		},
	}

	assert.True(t, node.HasChild("key1"))
	assert.False(t, node.HasChild("key2"))
}

func TestYAMLNode_HasChild_Nil(t *testing.T) {
	var node *YAMLNode
	assert.False(t, node.HasChild("anything"))
}

func TestYAMLNode_HasChild_NoChildren(t *testing.T) {
	node := &YAMLNode{Type: "scalar", Value: "test"}
	assert.False(t, node.HasChild("anything"))
}

func TestYAMLNode_GetChild(t *testing.T) {
	node := &YAMLNode{
		Type: "mapping",
		Children: map[string]*YAMLNode{
			"key1": {Type: "scalar", Value: "value1"},
		},
	}

	child := node.GetChild("key1")
	assert.NotNil(t, child)
	assert.Equal(t, "value1", child.Value)

	missing := node.GetChild("missing")
	assert.Nil(t, missing)
}

func TestYAMLNode_GetChild_Nil(t *testing.T) {
	var node *YAMLNode
	assert.Nil(t, node.GetChild("anything"))
}

func TestYAMLNode_ListValues(t *testing.T) {
	data := []interface{}{"item1", "item2", 42}
	node := &YAMLNode{
		Type:  "sequence",
		Value: data,
	}

	list := node.ListValues()
	assert.Equal(t, data, list)
	assert.Equal(t, 3, len(list))
}

func TestYAMLNode_ListValues_Nil(t *testing.T) {
	var node *YAMLNode
	assert.Nil(t, node.ListValues())
}

func TestYAMLNode_ListValues_NotSequence(t *testing.T) {
	node := &YAMLNode{Type: "scalar", Value: "test"}
	assert.Nil(t, node.ListValues())
}

func TestYAMLNode_ListValues_InvalidType(t *testing.T) {
	node := &YAMLNode{Type: "sequence", Value: "not a slice"}
	assert.Nil(t, node.ListValues())
}

func TestYAMLNode_StringValue(t *testing.T) {
	tests := []struct {
		name     string
		node     *YAMLNode
		expected string
	}{
		{
			name:     "string value",
			node:     &YAMLNode{Value: "test"},
			expected: "test",
		},
		{
			name:     "int value",
			node:     &YAMLNode{Value: 42},
			expected: "42",
		},
		{
			name:     "bool value",
			node:     &YAMLNode{Value: true},
			expected: "true",
		},
		{
			name:     "nil node",
			node:     nil,
			expected: "",
		},
		{
			name:     "nil value",
			node:     &YAMLNode{Value: nil},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.StringValue())
		})
	}
}

func TestYAMLNode_BoolValue(t *testing.T) {
	tests := []struct {
		name     string
		node     *YAMLNode
		expected bool
	}{
		{
			name:     "true value",
			node:     &YAMLNode{Value: true},
			expected: true,
		},
		{
			name:     "false value",
			node:     &YAMLNode{Value: false},
			expected: false,
		},
		{
			name:     "non-bool value",
			node:     &YAMLNode{Value: "true"},
			expected: false,
		},
		{
			name:     "nil node",
			node:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.node.BoolValue())
		})
	}
}
