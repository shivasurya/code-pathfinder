package graph

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLNode represents a node in the YAML tree.
type YAMLNode struct {
	Value    interface{}
	Children map[string]*YAMLNode
	Type     string // "scalar", "mapping", "sequence"
}

// YAMLGraph represents a parsed YAML document.
type YAMLGraph struct {
	Root     *YAMLNode
	FilePath string
}

// ParseYAML parses a YAML file and returns a YAMLGraph.
func ParseYAML(filePath string) (*YAMLGraph, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	return ParseYAMLString(string(content), filePath)
}

// ParseYAMLString parses a YAML string and returns a YAMLGraph.
func ParseYAMLString(content string, filePath ...string) (*YAMLGraph, error) {
	var data interface{}
	err := yaml.Unmarshal([]byte(content), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	path := "docker-compose.yml"
	if len(filePath) > 0 {
		path = filePath[0]
	}

	return &YAMLGraph{
		Root:     convertToYAMLNode(data),
		FilePath: path,
	}, nil
}

// convertToYAMLNode converts a generic interface{} to YAMLNode.
func convertToYAMLNode(data interface{}) *YAMLNode {
	if data == nil {
		return &YAMLNode{Type: "scalar", Value: nil}
	}

	switch v := data.(type) {
	case map[string]interface{}:
		node := &YAMLNode{
			Type:     "mapping",
			Children: make(map[string]*YAMLNode),
		}
		for key, val := range v {
			node.Children[key] = convertToYAMLNode(val)
		}
		return node

	case map[interface{}]interface{}:
		// YAML v3 sometimes returns map[interface{}]interface{}
		node := &YAMLNode{
			Type:     "mapping",
			Children: make(map[string]*YAMLNode),
		}
		for key, val := range v {
			keyStr := fmt.Sprint(key)
			node.Children[keyStr] = convertToYAMLNode(val)
		}
		return node

	case []interface{}:
		node := &YAMLNode{
			Type:  "sequence",
			Value: v,
		}
		return node

	default:
		return &YAMLNode{
			Type:  "scalar",
			Value: v,
		}
	}
}

// Query retrieves a top-level node by key.
func (yg *YAMLGraph) Query(key string) *YAMLNode {
	if yg.Root == nil || yg.Root.Children == nil {
		return nil
	}
	return yg.Root.Children[key]
}

// HasChild checks if a node has a child with the given key.
func (yn *YAMLNode) HasChild(key string) bool {
	if yn == nil || yn.Children == nil {
		return false
	}
	_, exists := yn.Children[key]
	return exists
}

// GetChild retrieves a child node by key.
func (yn *YAMLNode) GetChild(key string) *YAMLNode {
	if yn == nil || yn.Children == nil {
		return nil
	}
	return yn.Children[key]
}

// ListValues returns the value as a slice (for sequence nodes).
func (yn *YAMLNode) ListValues() []interface{} {
	if yn == nil {
		return nil
	}
	if yn.Type == "sequence" {
		if list, ok := yn.Value.([]interface{}); ok {
			return list
		}
	}
	return nil
}

// StringValue returns the value as a string.
func (yn *YAMLNode) StringValue() string {
	if yn == nil || yn.Value == nil {
		return ""
	}
	return fmt.Sprint(yn.Value)
}

// BoolValue returns the value as a boolean.
func (yn *YAMLNode) BoolValue() bool {
	if yn == nil {
		return false
	}
	if b, ok := yn.Value.(bool); ok {
		return b
	}
	return false
}
