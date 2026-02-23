package graph

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLNode represents a node in the YAML tree.
type YAMLNode struct {
	Value      any
	Children   map[string]*YAMLNode
	Type       string // "scalar", "mapping", "sequence"
	LineNumber int    // Line number in source file (1-indexed)
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
	var node yaml.Node
	err := yaml.Unmarshal([]byte(content), &node)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	path := "docker-compose.yml"
	if len(filePath) > 0 {
		path = filePath[0]
	}

	// The root node is a document node, get its content (first child)
	var rootContent *yaml.Node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		rootContent = node.Content[0]
	} else {
		rootContent = &node
	}

	return &YAMLGraph{
		Root:     convertYAMLNodeToInternal(rootContent),
		FilePath: path,
	}, nil
}

// convertYAMLNodeToInternal converts yaml.Node to our internal YAMLNode with line numbers.
func convertYAMLNodeToInternal(node *yaml.Node) *YAMLNode {
	if node == nil {
		return &YAMLNode{Type: "scalar", Value: nil, LineNumber: 0}
	}

	result := &YAMLNode{
		LineNumber: node.Line,
	}

	switch node.Kind {
	case yaml.MappingNode:
		result.Type = "mapping"
		result.Children = make(map[string]*YAMLNode)

		// Mapping nodes have alternating key-value pairs in Content
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) {
				keyNode := node.Content[i]
				valueNode := node.Content[i+1]

				key := keyNode.Value
				result.Children[key] = convertYAMLNodeToInternal(valueNode)
			}
		}

	case yaml.SequenceNode:
		result.Type = "sequence"
		var items []any
		for _, item := range node.Content {
			converted := convertYAMLNodeToInternal(item)
			if converted.Type == "scalar" {
				items = append(items, converted.Value)
			} else {
				items = append(items, converted)
			}
		}
		result.Value = items

	case yaml.ScalarNode:
		result.Type = "scalar"
		// Decode scalar value to proper type (bool, int, float, string)
		var decoded any
		if err := node.Decode(&decoded); err == nil {
			result.Value = decoded
		} else {
			// Fall back to string value if decoding fails
			result.Value = node.Value
		}

	case yaml.AliasNode:
		// Dereference alias
		return convertYAMLNodeToInternal(node.Alias)

	default:
		result.Type = "scalar"
		result.Value = nil
	}

	return result
}

// convertToYAMLNode converts a generic interface{} to YAMLNode (deprecated, use convertYAMLNodeToInternal).
func convertToYAMLNode(data any) *YAMLNode {
	if data == nil {
		return &YAMLNode{Type: "scalar", Value: nil}
	}

	switch v := data.(type) {
	case map[string]any:
		node := &YAMLNode{
			Type:     "mapping",
			Children: make(map[string]*YAMLNode),
		}
		for key, val := range v {
			node.Children[key] = convertToYAMLNode(val)
		}
		return node

	case map[any]any:
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

	case []any:
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
func (yn *YAMLNode) ListValues() []any {
	if yn == nil {
		return nil
	}
	if yn.Type == "sequence" {
		if list, ok := yn.Value.([]any); ok {
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
