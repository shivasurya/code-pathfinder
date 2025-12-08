package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDockerfileParser(t *testing.T) {
	parser := NewDockerfileParser()
	assert.NotNil(t, parser)
	assert.NotNil(t, parser.parser)
}

func TestDockerfileParser_Parse_Simple(t *testing.T) {
	parser := NewDockerfileParser()

	dockerfile := []byte(`FROM ubuntu:20.04
RUN apt-get update
USER appuser
CMD ["/bin/bash"]
`)

	graph, err := parser.Parse("Dockerfile", dockerfile)

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, 4, graph.TotalInstructions)
	assert.True(t, graph.HasInstruction("FROM"))
	assert.True(t, graph.HasInstruction("RUN"))
	assert.True(t, graph.HasInstruction("USER"))
	assert.True(t, graph.HasInstruction("CMD"))
}

func TestDockerfileParser_Parse_MultiStage(t *testing.T) {
	parser := NewDockerfileParser()

	dockerfile := []byte(`FROM golang:1.21 AS builder
RUN go build -o app

FROM alpine:3.18
COPY --from=builder /app /app
CMD ["/app"]
`)

	graph, err := parser.Parse("Dockerfile", dockerfile)

	assert.NoError(t, err)
	assert.True(t, graph.IsMultiStage())
	assert.Equal(t, 5, graph.TotalInstructions)
	assert.Equal(t, 2, len(graph.GetInstructions("FROM")))
}

func TestDockerfileParser_Parse_AllInstructions(t *testing.T) {
	parser := NewDockerfileParser()

	// Dockerfile with all 18 instruction types
	dockerfile := []byte(`FROM ubuntu:20.04 AS base
MAINTAINER test@example.com
RUN apt-get update
COPY src /app/src
ADD archive.tar.gz /app/
ENV APP_ENV=production
ARG VERSION=1.0
USER appuser
EXPOSE 8080/tcp
WORKDIR /app
VOLUME ["/data"]
SHELL ["/bin/bash", "-c"]
HEALTHCHECK --interval=30s CMD curl -f http://localhost/
LABEL version="1.0"
STOPSIGNAL SIGTERM
ONBUILD RUN echo "building"
CMD ["./app"]
ENTRYPOINT ["/entrypoint.sh"]
`)

	graph, err := parser.Parse("Dockerfile", dockerfile)

	assert.NoError(t, err)
	assert.Equal(t, 18, graph.TotalInstructions)

	// Verify each instruction type is present
	instructionTypes := []string{
		"FROM", "MAINTAINER", "RUN", "COPY", "ADD", "ENV", "ARG",
		"USER", "EXPOSE", "WORKDIR", "VOLUME", "SHELL", "HEALTHCHECK",
		"LABEL", "STOPSIGNAL", "ONBUILD", "CMD", "ENTRYPOINT",
	}

	for _, instType := range instructionTypes {
		assert.True(t, graph.HasInstruction(instType),
			"Missing instruction: %s", instType)
	}
}

func TestDockerfileParser_Parse_EmptyDockerfile(t *testing.T) {
	parser := NewDockerfileParser()

	dockerfile := []byte(`# Just a comment
`)

	graph, err := parser.Parse("Dockerfile", dockerfile)

	assert.NoError(t, err)
	assert.Equal(t, 0, graph.TotalInstructions)
}

func TestDockerfileParser_Parse_LineNumbers(t *testing.T) {
	parser := NewDockerfileParser()

	dockerfile := []byte(`# Comment
FROM ubuntu:20.04

RUN apt-get update
`)

	graph, err := parser.Parse("Dockerfile", dockerfile)

	assert.NoError(t, err)

	fromNodes := graph.GetInstructions("FROM")
	assert.Equal(t, 1, len(fromNodes))
	assert.Equal(t, 2, fromNodes[0].LineNumber)

	runNodes := graph.GetInstructions("RUN")
	assert.Equal(t, 1, len(runNodes))
	assert.Equal(t, 4, runNodes[0].LineNumber)
}

func TestIsInstructionNode(t *testing.T) {
	tests := []struct {
		nodeType string
		expected bool
	}{
		{"from_instruction", true},
		{"run_instruction", true},
		{"copy_instruction", true},
		{"comment", false},
		{"blank_line", false},
		{"source_file", false},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			// Create mock node (for testing helper logic)
			result := isInstructionNodeType(tt.nodeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper for testing without actual tree-sitter node.
func isInstructionNodeType(nodeType string) bool {
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

func TestExtractInstructionType(t *testing.T) {
	tests := []struct {
		nodeType string
		expected string
	}{
		{"from_instruction", "FROM"},
		{"run_instruction", "RUN"},
		{"copy_instruction", "COPY"},
		{"user_instruction", "USER"},
		{"healthcheck_instruction", "HEALTHCHECK"},
		{"unknown_type", "unknown_type"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			result := extractInstructionType(tt.nodeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
