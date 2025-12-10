package docker

import (
	"context"
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
)

// DockerfileParser handles parsing of Dockerfile content using tree-sitter.
type DockerfileParser struct {
	parser *sitter.Parser
}

// NewDockerfileParser creates a new Dockerfile parser.
func NewDockerfileParser() *DockerfileParser {
	parser := sitter.NewParser()
	parser.SetLanguage(dockerfile.GetLanguage())
	return &DockerfileParser{parser: parser}
}

// ParseFile parses a Dockerfile from a file path.
func (dp *DockerfileParser) ParseFile(filePath string) (*DockerfileGraph, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
	}
	return dp.Parse(filePath, content)
}

// Parse parses Dockerfile content and returns a DockerfileGraph.
func (dp *DockerfileParser) Parse(filePath string, content []byte) (*DockerfileGraph, error) {
	// Parse into tree-sitter AST
	tree, err := dp.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile: %w", err)
	}
	defer tree.Close()

	rootNode := tree.RootNode()

	// Check for syntax errors
	if rootNode.HasError() {
		// Log warning but continue (partial parsing is useful)
		// log.Printf("Warning: Dockerfile has syntax errors: %s", filePath)
	}

	// Create graph
	graph := NewDockerfileGraph(filePath)

	// Convert AST to DockerfileGraph
	dp.convertASTToGraph(rootNode, content, graph)

	return graph, nil
}

// convertASTToGraph traverses the tree-sitter AST and populates DockerfileGraph.
func (dp *DockerfileParser) convertASTToGraph(
	rootNode *sitter.Node,
	source []byte,
	graph *DockerfileGraph,
) {
	// Iterate through all child nodes
	for i := 0; i < int(rootNode.ChildCount()); i++ {
		child := rootNode.Child(i)

		// Skip non-instruction nodes (comments, blank lines).
		if !isInstructionNode(child) {
			continue
		}

		// Convert to DockerfileNode (implemented in PR #3).
		node := dp.convertInstruction(child, source)

		graph.AddInstruction(node)
	}

	// Analyze build stages after all instructions parsed.
	graph.AnalyzeBuildStages()
}

// isInstructionNode checks if a tree-sitter node represents a Dockerfile instruction.
func isInstructionNode(node *sitter.Node) bool {
	nodeType := node.Type()
	instructionTypes := map[string]bool{
		"from_instruction":        true,
		"run_instruction":         true,
		"copy_instruction":        true,
		"add_instruction":         true,
		"env_instruction":         true,
		"arg_instruction":         true,
		"user_instruction":        true,
		"expose_instruction":      true,
		"workdir_instruction":     true,
		"cmd_instruction":         true,
		"entrypoint_instruction":  true,
		"volume_instruction":      true,
		"shell_instruction":       true,
		"healthcheck_instruction": true,
		"label_instruction":       true,
		"onbuild_instruction":     true,
		"stopsignal_instruction":  true,
		"maintainer_instruction":  true,
	}
	return instructionTypes[nodeType]
}

// convertInstruction is a placeholder for PR #3.
// It will be replaced with actual conversion logic.
func (dp *DockerfileParser) convertInstruction(
	node *sitter.Node,
	source []byte,
) *DockerfileNode {
	// Placeholder implementation - creates basic node with type and line.
	// Full implementation in PR #3.

	nodeType := node.Type()
	instructionType := extractInstructionType(nodeType)

	dockerNode := NewDockerfileNode(
		instructionType,
		int(node.StartPoint().Row)+1, // 1-indexed line number
	)
	dockerNode.RawInstruction = node.Content(source)

	return dockerNode
}

// extractInstructionType converts tree-sitter node type to instruction name.
// For example, "from_instruction" becomes "FROM".
func extractInstructionType(nodeType string) string {
	typeMap := map[string]string{
		"from_instruction":        "FROM",
		"run_instruction":         "RUN",
		"copy_instruction":        "COPY",
		"add_instruction":         "ADD",
		"env_instruction":         "ENV",
		"arg_instruction":         "ARG",
		"user_instruction":        "USER",
		"expose_instruction":      "EXPOSE",
		"workdir_instruction":     "WORKDIR",
		"cmd_instruction":         "CMD",
		"entrypoint_instruction":  "ENTRYPOINT",
		"volume_instruction":      "VOLUME",
		"shell_instruction":       "SHELL",
		"healthcheck_instruction": "HEALTHCHECK",
		"label_instruction":       "LABEL",
		"onbuild_instruction":     "ONBUILD",
		"stopsignal_instruction":  "STOPSIGNAL",
		"maintainer_instruction":  "MAINTAINER",
	}
	if t, ok := typeMap[nodeType]; ok {
		return t
	}
	return nodeType
}
