package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDockerfile(t *testing.T) {
	// Create a temporary Dockerfile
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")

	dockerfileContent := `FROM ubuntu:22.04
RUN apt-get update && apt-get install -y curl
USER appuser
EXPOSE 8080
WORKDIR /app
COPY . /app
ENV DEBUG=true
CMD ["./start.sh"]
`

	err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
	require.NoError(t, err)

	// Parse Dockerfile
	graph := NewCodeGraph()
	err = parseDockerfile(dockerfilePath, graph)
	require.NoError(t, err)

	// Verify nodes created
	assert.Len(t, graph.Nodes, 8, "Should create 8 nodes for 8 instructions")

	// Verify node types
	for _, node := range graph.Nodes {
		assert.Equal(t, "dockerfile_instruction", node.Type)
		assert.NotEmpty(t, node.ID)
		assert.NotEmpty(t, node.Name) // Instruction type (FROM, RUN, etc.)
		assert.Equal(t, dockerfilePath, node.File)
	}
}

func TestParseDockerfileWithMultiStage(t *testing.T) {
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")

	dockerfileContent := `FROM golang:1.21 AS builder
WORKDIR /build
COPY . .
RUN go build -o app

FROM alpine:latest
COPY --from=builder /build/app /app
USER nobody
CMD ["/app"]
`

	err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
	require.NoError(t, err)

	graph := NewCodeGraph()
	err = parseDockerfile(dockerfilePath, graph)
	require.NoError(t, err)

	// Should create nodes for all instructions
	assert.GreaterOrEqual(t, len(graph.Nodes), 8)
}

func TestParseDockerfileWithError(t *testing.T) {
	graph := NewCodeGraph()
	err := parseDockerfile("/nonexistent/Dockerfile", graph)
	assert.Error(t, err)
}

func TestParseDockerCompose(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	composeContent := `version: '3.8'
services:
  web:
    image: nginx:latest
    privileged: true
    network_mode: host
    ports:
      - "8080:80"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - DEBUG=true
  db:
    image: postgres:15
    read_only: true
`

	err := os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Parse docker-compose.yml
	graph := NewCodeGraph()
	err = parseDockerCompose(composePath, graph)
	require.NoError(t, err)

	// Verify nodes created (2 services)
	assert.Len(t, graph.Nodes, 2, "Should create 2 nodes for 2 services")

	// Verify node types
	for _, node := range graph.Nodes {
		assert.Equal(t, "compose_service", node.Type)
		assert.NotEmpty(t, node.ID)
		assert.NotEmpty(t, node.Name) // Service name
		assert.Equal(t, composePath, node.File)
		assert.NotEmpty(t, node.MethodArgumentsValue) // Properties
	}
}

func TestParseDockerComposeWithError(t *testing.T) {
	graph := NewCodeGraph()
	err := parseDockerCompose("/nonexistent/docker-compose.yml", graph)
	assert.Error(t, err)
}

func TestConvertDockerInstructionToNode(t *testing.T) {
	tests := []struct {
		name         string
		dockerNode   *docker.DockerfileNode
		expectedType string
		expectedName string
	}{
		{
			name: "FROM instruction",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "FROM",
				BaseImage:       "ubuntu",
				ImageTag:        "22.04",
				LineNumber:      1,
			},
			expectedType: "dockerfile_instruction",
			expectedName: "FROM",
		},
		{
			name: "USER instruction",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "USER",
				UserName:        "appuser",
				LineNumber:      5,
			},
			expectedType: "dockerfile_instruction",
			expectedName: "USER",
		},
		{
			name: "EXPOSE instruction",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "EXPOSE",
				Ports:           []int{8080, 8443},
				LineNumber:      10,
			},
			expectedType: "dockerfile_instruction",
			expectedName: "EXPOSE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := convertDockerInstructionToNode(tt.dockerNode, "/test/Dockerfile")
			assert.Equal(t, tt.expectedType, node.Type)
			assert.Equal(t, tt.expectedName, node.Name)
			assert.NotEmpty(t, node.ID)
			assert.Equal(t, uint32(tt.dockerNode.LineNumber), node.LineNumber)
		})
	}
}

func TestExtractDockerInstructionArgs(t *testing.T) {
	tests := []struct {
		name       string
		dockerNode *docker.DockerfileNode
		expected   []string
	}{
		{
			name: "FROM with tag",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "FROM",
				BaseImage:       "ubuntu",
				ImageTag:        "22.04",
			},
			expected: []string{"ubuntu", "22.04"},
		},
		{
			name: "USER instruction",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "USER",
				UserName:        "appuser",
			},
			expected: []string{"appuser"},
		},
		{
			name: "EXPOSE instruction",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "EXPOSE",
				Ports:           []int{8080, 8443},
			},
			expected: []string{"8080", "8443"},
		},
		{
			name: "ENV instruction",
			dockerNode: &docker.DockerfileNode{
				InstructionType: "ENV",
				EnvVars: map[string]string{
					"DEBUG": "true",
					"PORT":  "8080",
				},
			},
			expected: []string{"DEBUG=true", "PORT=8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := extractDockerInstructionArgs(tt.dockerNode)
			assert.ElementsMatch(t, tt.expected, args)
		})
	}
}

func TestConvertComposeServiceToNode(t *testing.T) {
	yamlGraph, err := ParseYAMLString(`
version: '3.8'
services:
  web:
    image: nginx:latest
    privileged: true
    ports:
      - "8080:80"
`, "docker-compose.yml")
	require.NoError(t, err)

	composeGraph := NewComposeGraph(yamlGraph, "docker-compose.yml")
	serviceNode := composeGraph.Services["web"]
	require.NotNil(t, serviceNode)

	node := convertComposeServiceToNode("web", serviceNode, "docker-compose.yml")

	assert.Equal(t, "compose_service", node.Type)
	assert.Equal(t, "web", node.Name)
	assert.NotEmpty(t, node.ID)
	assert.Equal(t, "docker-compose.yml", node.File)
	assert.NotEmpty(t, node.MethodArgumentsValue)
}

func TestExtractComposeServiceProperties(t *testing.T) {
	yamlGraph, err := ParseYAMLString(`
services:
  web:
    image: nginx:latest
    privileged: true
    network_mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      DEBUG: "true"
`, "docker-compose.yml")
	require.NoError(t, err)

	composeGraph := NewComposeGraph(yamlGraph, "docker-compose.yml")
	serviceNode := composeGraph.Services["web"]
	require.NotNil(t, serviceNode)

	props := extractComposeServiceProperties(serviceNode)

	// Verify expected properties are extracted
	assert.Contains(t, props, "image=nginx:latest")
	assert.Contains(t, props, "privileged=true")
	assert.Contains(t, props, "network_mode=host")

	// Verify volume contains docker socket
	hasDockerSocket := false
	for _, prop := range props {
		if contains(prop, "/var/run/docker.sock") {
			hasDockerSocket = true
			break
		}
	}
	assert.True(t, hasDockerSocket, "Should extract Docker socket volume")
}

func TestIsDockerNode(t *testing.T) {
	dockerNode := &Node{Type: "dockerfile_instruction"}
	assert.True(t, IsDockerNode(dockerNode))

	nonDockerNode := &Node{Type: "function_definition"}
	assert.False(t, IsDockerNode(nonDockerNode))
}

func TestIsComposeNode(t *testing.T) {
	composeNode := &Node{Type: "compose_service"}
	assert.True(t, IsComposeNode(composeNode))

	nonComposeNode := &Node{Type: "class_declaration"}
	assert.False(t, IsComposeNode(nonComposeNode))
}

func TestGetDockerInstructionType(t *testing.T) {
	node := &Node{
		Type: "dockerfile_instruction",
		Name: "RUN",
	}
	assert.Equal(t, "RUN", GetDockerInstructionType(node))

	nonDockerNode := &Node{Type: "function_definition"}
	assert.Empty(t, GetDockerInstructionType(nonDockerNode))
}

func TestHasDockerInstructionArg(t *testing.T) {
	node := &Node{
		Type:                 "dockerfile_instruction",
		MethodArgumentsValue: []string{"ubuntu", "22.04", "apt-get"},
	}

	assert.True(t, HasDockerInstructionArg(node, "ubuntu"))
	assert.True(t, HasDockerInstructionArg(node, "22.04"))
	assert.False(t, HasDockerInstructionArg(node, "nonexistent"))
}

func TestGetComposeServiceProperty(t *testing.T) {
	node := &Node{
		Type: "compose_service",
		MethodArgumentsValue: []string{
			"image=nginx:latest",
			"privileged=true",
			"network_mode=host",
		},
	}

	assert.Equal(t, "nginx:latest", GetComposeServiceProperty(node, "image"))
	assert.Equal(t, "true", GetComposeServiceProperty(node, "privileged"))
	assert.Equal(t, "host", GetComposeServiceProperty(node, "network_mode"))
	assert.Empty(t, GetComposeServiceProperty(node, "nonexistent"))
}

func TestHasComposeServiceProperty(t *testing.T) {
	node := &Node{
		Type: "compose_service",
		MethodArgumentsValue: []string{
			"image=nginx:latest",
			"privileged=true",
			"port=8080:80",
		},
	}

	// Check existence
	assert.True(t, HasComposeServiceProperty(node, "image"))
	assert.True(t, HasComposeServiceProperty(node, "privileged"))
	assert.False(t, HasComposeServiceProperty(node, "nonexistent"))

	// Check specific value
	assert.True(t, HasComposeServiceProperty(node, "image", "nginx:latest"))
	assert.True(t, HasComposeServiceProperty(node, "privileged", "true"))
	assert.False(t, HasComposeServiceProperty(node, "image", "apache"))
}

func TestInitializeWithDockerFiles(t *testing.T) {
	// Create test directory with Docker files
	tmpDir := t.TempDir()

	// Create Dockerfile
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	dockerfileContent := `FROM ubuntu:22.04
USER appuser
`
	err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
	require.NoError(t, err)

	// Create docker-compose.yml
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	composeContent := `version: '3.8'
services:
  web:
    image: nginx
`
	err = os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Initialize CodeGraph
	graph := Initialize(tmpDir, nil)

	// Verify both files were parsed
	assert.GreaterOrEqual(t, len(graph.Nodes), 3, "Should have nodes from both Dockerfile and docker-compose")

	// Verify we have both Docker and Compose nodes
	hasDockerNode := false
	hasComposeNode := false
	for _, node := range graph.Nodes {
		if node.Type == "dockerfile_instruction" {
			hasDockerNode = true
		}
		if node.Type == "compose_service" {
			hasComposeNode = true
		}
	}

	assert.True(t, hasDockerNode, "Should have Dockerfile nodes")
	assert.True(t, hasComposeNode, "Should have Compose nodes")
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   (len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		   (len(s) > len(substr)*2 && s[len(s)/2-len(substr)/2:len(s)/2+len(substr)/2+1] == substr)
}
