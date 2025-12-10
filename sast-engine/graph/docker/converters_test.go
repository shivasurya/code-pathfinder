package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test FROM converter.

func TestConvertFROM_Simple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM ubuntu:20.04"))

	nodes := graph.GetInstructions("FROM")
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "ubuntu", nodes[0].BaseImage)
	assert.Equal(t, "20.04", nodes[0].ImageTag)
}

func TestConvertFROM_WithAlias(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM golang:1.21 AS builder"))

	nodes := graph.GetInstructions("FROM")
	assert.Equal(t, "golang", nodes[0].BaseImage)
	assert.Equal(t, "1.21", nodes[0].ImageTag)
	assert.Equal(t, "builder", nodes[0].StageAlias)
}

func TestConvertFROM_WithDigest(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM ubuntu@sha256:abc123"))

	nodes := graph.GetInstructions("FROM")
	assert.Equal(t, "ubuntu", nodes[0].BaseImage)
	assert.Equal(t, "sha256:abc123", nodes[0].ImageDigest)
}

func TestConvertFROM_ImplicitLatest(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM ubuntu"))

	nodes := graph.GetInstructions("FROM")
	assert.Equal(t, "ubuntu", nodes[0].BaseImage)
	assert.Equal(t, "latest", nodes[0].ImageTag)
}

// Test USER converter.

func TestConvertUSER_UserOnly(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nUSER appuser"))

	nodes := graph.GetInstructions("USER")
	assert.Equal(t, "appuser", nodes[0].UserName)
	assert.Equal(t, "", nodes[0].GroupName)
}

func TestConvertUSER_UserAndGroup(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nUSER appuser:appgroup"))

	nodes := graph.GetInstructions("USER")
	assert.Equal(t, "appuser", nodes[0].UserName)
	assert.Equal(t, "appgroup", nodes[0].GroupName)
}

// Test COPY/ADD converters.

func TestConvertCOPY_Simple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nCOPY src dest"))

	nodes := graph.GetInstructions("COPY")
	assert.Equal(t, []string{"src"}, nodes[0].SourcePaths)
	assert.Equal(t, "dest", nodes[0].DestPath)
}

func TestConvertCOPY_WithFlags(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nCOPY --from=builder /app /app"))

	nodes := graph.GetInstructions("COPY")
	assert.Equal(t, "builder", nodes[0].CopyFrom)
	assert.Equal(t, "/app", nodes[0].DestPath)
}

// Test RUN converter.

func TestConvertRUN_Shell(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nRUN apt-get update"))

	nodes := graph.GetInstructions("RUN")
	assert.Equal(t, "shell", nodes[0].CommandForm)
	assert.Equal(t, 1, len(nodes[0].Arguments))
}

func TestConvertRUN_Exec(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte(`FROM base
RUN ["apt-get", "update"]`))

	nodes := graph.GetInstructions("RUN")
	assert.Equal(t, "exec", nodes[0].CommandForm)
	assert.Equal(t, []string{"apt-get", "update"}, nodes[0].CommandArray)
}

// Test CMD converter.

func TestConvertCMD_Shell(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nCMD echo hello"))

	nodes := graph.GetInstructions("CMD")
	assert.Equal(t, "shell", nodes[0].CommandForm)
}

func TestConvertCMD_Exec(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte(`FROM base
CMD ["/bin/bash"]`))

	nodes := graph.GetInstructions("CMD")
	assert.Equal(t, "exec", nodes[0].CommandForm)
	assert.Equal(t, []string{"/bin/bash"}, nodes[0].CommandArray)
}

// Test ENV converter.

func TestConvertENV_Simple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nENV APP_ENV=production"))

	nodes := graph.GetInstructions("ENV")
	assert.Equal(t, "production", nodes[0].EnvVars["APP_ENV"])
}

func TestConvertENV_Multiple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nENV APP_ENV=production DEBUG=true"))

	nodes := graph.GetInstructions("ENV")
	assert.Equal(t, "production", nodes[0].EnvVars["APP_ENV"])
	assert.Equal(t, "true", nodes[0].EnvVars["DEBUG"])
}

// Test ARG converter.

func TestConvertARG_Simple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nARG VERSION"))

	nodes := graph.GetInstructions("ARG")
	assert.Equal(t, "VERSION", nodes[0].ArgName)
}

func TestConvertARG_WithDefault(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nARG VERSION=1.0"))

	nodes := graph.GetInstructions("ARG")
	assert.Equal(t, "VERSION", nodes[0].ArgName)
	assert.Equal(t, []string{"1.0"}, nodes[0].Arguments)
}

// Test EXPOSE converter.

func TestConvertEXPOSE_Simple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nEXPOSE 8080"))

	nodes := graph.GetInstructions("EXPOSE")
	assert.Equal(t, []int{8080}, nodes[0].Ports)
}

func TestConvertEXPOSE_WithProtocol(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nEXPOSE 8080/tcp"))

	nodes := graph.GetInstructions("EXPOSE")
	assert.Equal(t, []int{8080}, nodes[0].Ports)
	assert.Equal(t, "tcp", nodes[0].Protocol)
}

// Test WORKDIR converter.

func TestConvertWORKDIR_Absolute(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nWORKDIR /app"))

	nodes := graph.GetInstructions("WORKDIR")
	assert.Equal(t, "/app", nodes[0].WorkDir)
	assert.True(t, nodes[0].IsAbsolutePath)
}

func TestConvertWORKDIR_Relative(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nWORKDIR app"))

	nodes := graph.GetInstructions("WORKDIR")
	assert.Equal(t, "app", nodes[0].WorkDir)
	assert.False(t, nodes[0].IsAbsolutePath)
}

// Test VOLUME converter.

func TestConvertVOLUME_JSON(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte(`FROM base
VOLUME ["/data"]`))

	nodes := graph.GetInstructions("VOLUME")
	assert.Equal(t, []string{"/data"}, nodes[0].Volumes)
}

// Test LABEL converter.

func TestConvertLABEL_Simple(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nLABEL version=\"1.0\""))

	nodes := graph.GetInstructions("LABEL")
	assert.Equal(t, "1.0", nodes[0].Labels["version"])
}

// Test MAINTAINER converter.

func TestConvertMAINTAINER(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nMAINTAINER test@example.com"))

	nodes := graph.GetInstructions("MAINTAINER")
	assert.Equal(t, []string{"test@example.com"}, nodes[0].Arguments)
}

// Test STOPSIGNAL converter.

func TestConvertSTOPSIGNAL(t *testing.T) {
	parser := NewDockerfileParser()
	graph, _ := parser.Parse("Dockerfile", []byte("FROM base\nSTOPSIGNAL SIGTERM"))

	nodes := graph.GetInstructions("STOPSIGNAL")
	assert.Equal(t, "SIGTERM", nodes[0].StopSignal)
}
