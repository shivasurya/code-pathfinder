package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDockerfileNode(t *testing.T) {
	node := NewDockerfileNode("FROM", 1)

	assert.Equal(t, "FROM", node.InstructionType)
	assert.Equal(t, 1, node.LineNumber)
	assert.NotNil(t, node.Flags)
	assert.NotNil(t, node.EnvVars)
	assert.NotNil(t, node.Labels)
	assert.NotNil(t, node.Arguments)
}

func TestDockerfileNode_GetFlag(t *testing.T) {
	node := NewDockerfileNode("COPY", 10)
	node.Flags["from"] = "builder"

	assert.Equal(t, "builder", node.GetFlag("from"))
	assert.Equal(t, "", node.GetFlag("nonexistent"))
}

func TestDockerfileNode_HasFlag(t *testing.T) {
	node := NewDockerfileNode("COPY", 10)
	node.Flags["chown"] = "user:group"

	assert.True(t, node.HasFlag("chown"))
	assert.False(t, node.HasFlag("chmod"))
}

func TestDockerfileNode_IsRootUser(t *testing.T) {
	tests := []struct {
		name     string
		instType string
		userName string
		expected bool
	}{
		{"root user", "USER", "root", true},
		{"uid 0", "USER", "0", true},
		{"non-root", "USER", "appuser", false},
		{"non-USER instruction", "FROM", "root", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewDockerfileNode(tt.instType, 1)
			node.UserName = tt.userName
			assert.Equal(t, tt.expected, node.IsRootUser())
		})
	}
}

func TestDockerfileNode_UsesLatestTag(t *testing.T) {
	tests := []struct {
		name     string
		instType string
		tag      string
		expected bool
	}{
		{"explicit latest", "FROM", "latest", true},
		{"implicit latest", "FROM", "", true},
		{"specific version", "FROM", "1.21", false},
		{"non-FROM instruction", "RUN", "latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewDockerfileNode(tt.instType, 1)
			node.ImageTag = tt.tag
			assert.Equal(t, tt.expected, node.UsesLatestTag())
		})
	}
}
