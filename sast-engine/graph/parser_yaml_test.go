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
	data := map[string]any{
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
	data := map[any]any{
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
	data := []any{"item1", "item2", 42}
	node := convertToYAMLNode(data)
	assert.Equal(t, "sequence", node.Type)
	assert.Equal(t, data, node.Value)
}

func TestConvertToYAMLNode_NestedStructure(t *testing.T) {
	data := map[string]any{
		"services": map[string]any{
			"web": map[string]any{
				"image": "nginx",
				"ports": []any{"80:80", "443:443"},
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
	data := []any{"item1", "item2", 42}
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

func TestConvertYAMLNodeToInternal_LineNumbers(t *testing.T) {
	yaml := `version: '3.8'
services:
  web:
    image: nginx
    ports:
      - "80:80"
    privileged: true
`
	graph, err := ParseYAMLString(yaml)
	assert.NoError(t, err)

	t.Run("preserves line numbers for scalar values", func(t *testing.T) {
		versionNode := graph.Query("version")
		assert.NotNil(t, versionNode)
		assert.Greater(t, versionNode.LineNumber, 0)
		assert.Equal(t, "3.8", versionNode.Value)
	})

	t.Run("preserves line numbers for mapping nodes", func(t *testing.T) {
		servicesNode := graph.Query("services")
		assert.NotNil(t, servicesNode)
		assert.Greater(t, servicesNode.LineNumber, 0)
		assert.Equal(t, "mapping", servicesNode.Type)
	})

	t.Run("preserves line numbers for nested properties", func(t *testing.T) {
		servicesNode := graph.Query("services")
		webNode := servicesNode.GetChild("web")
		assert.NotNil(t, webNode)
		assert.Greater(t, webNode.LineNumber, 0)

		imageNode := webNode.GetChild("image")
		assert.NotNil(t, imageNode)
		assert.Greater(t, imageNode.LineNumber, 0)
	})

	t.Run("decodes bool values correctly", func(t *testing.T) {
		servicesNode := graph.Query("services")
		webNode := servicesNode.GetChild("web")
		privilegedNode := webNode.GetChild("privileged")
		assert.NotNil(t, privilegedNode)
		assert.Equal(t, true, privilegedNode.Value)
		assert.True(t, privilegedNode.BoolValue())
	})
}

func TestConvertYAMLNodeToInternal_SequenceNodes(t *testing.T) {
	yaml := `
items:
  - item1
  - item2
  - 42
`
	graph, err := ParseYAMLString(yaml)
	assert.NoError(t, err)

	itemsNode := graph.Query("items")
	assert.NotNil(t, itemsNode)
	assert.Equal(t, "sequence", itemsNode.Type)
	assert.Greater(t, itemsNode.LineNumber, 0)

	list := itemsNode.ListValues()
	assert.NotNil(t, list)
	assert.Equal(t, 3, len(list))
	assert.Equal(t, "item1", list[0])
	assert.Equal(t, "item2", list[1])
	assert.Equal(t, 42, list[2])
}

func TestConvertYAMLNodeToInternal_ScalarTypeDecoding(t *testing.T) {
	yaml := `
string_val: "test"
int_val: 42
float_val: 3.14
bool_true: true
bool_false: false
null_val: null
`
	graph, err := ParseYAMLString(yaml)
	assert.NoError(t, err)

	t.Run("decodes string values", func(t *testing.T) {
		node := graph.Query("string_val")
		assert.NotNil(t, node)
		assert.Equal(t, "test", node.Value)
		assert.Equal(t, "scalar", node.Type)
	})

	t.Run("decodes int values", func(t *testing.T) {
		node := graph.Query("int_val")
		assert.NotNil(t, node)
		assert.Equal(t, 42, node.Value)
	})

	t.Run("decodes float values", func(t *testing.T) {
		node := graph.Query("float_val")
		assert.NotNil(t, node)
		assert.Equal(t, 3.14, node.Value)
	})

	t.Run("decodes bool true", func(t *testing.T) {
		node := graph.Query("bool_true")
		assert.NotNil(t, node)
		assert.Equal(t, true, node.Value)
		assert.True(t, node.BoolValue())
	})

	t.Run("decodes bool false", func(t *testing.T) {
		node := graph.Query("bool_false")
		assert.NotNil(t, node)
		assert.Equal(t, false, node.Value)
		assert.False(t, node.BoolValue())
	})

	t.Run("decodes null values", func(t *testing.T) {
		node := graph.Query("null_val")
		assert.NotNil(t, node)
		assert.Nil(t, node.Value)
	})
}

func TestConvertYAMLNodeToInternal_NilNode(t *testing.T) {
	result := convertYAMLNodeToInternal(nil)
	assert.NotNil(t, result)
	assert.Equal(t, "scalar", result.Type)
	assert.Nil(t, result.Value)
	assert.Equal(t, 0, result.LineNumber)
}

func TestConvertYAMLNodeToInternal_SequenceWithNestedMaps(t *testing.T) {
	yaml := `
volumes:
  - name: data
    path: /data
  - name: logs
    path: /logs
`
	graph, err := ParseYAMLString(yaml)
	assert.NoError(t, err)

	volumesNode := graph.Query("volumes")
	assert.NotNil(t, volumesNode)
	assert.Equal(t, "sequence", volumesNode.Type)

	list := volumesNode.ListValues()
	assert.NotNil(t, list)
	assert.Equal(t, 2, len(list))

	// First item should be a YAMLNode (mapping)
	firstItem, ok := list[0].(*YAMLNode)
	assert.True(t, ok)
	assert.Equal(t, "mapping", firstItem.Type)
	assert.NotNil(t, firstItem.Children["name"])
	assert.Equal(t, "data", firstItem.Children["name"].Value)
}
