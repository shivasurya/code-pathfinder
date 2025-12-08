package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDockerfileGraph(t *testing.T) {
	graph := NewDockerfileGraph("/path/to/Dockerfile")

	assert.Equal(t, "/path/to/Dockerfile", graph.FilePath)
	assert.NotNil(t, graph.Instructions)
	assert.NotNil(t, graph.Stages)
	assert.NotNil(t, graph.InstructionIndex)
	assert.NotNil(t, graph.Environment)
	assert.Equal(t, 0, graph.TotalInstructions)
}

func TestDockerfileGraph_AddInstruction(t *testing.T) {
	graph := NewDockerfileGraph("Dockerfile")

	fromNode := NewDockerfileNode("FROM", 1)
	fromNode.BaseImage = "ubuntu"
	graph.AddInstruction(fromNode)

	runNode := NewDockerfileNode("RUN", 2)
	graph.AddInstruction(runNode)

	assert.Equal(t, 2, graph.TotalInstructions)
	assert.Equal(t, 1, len(graph.GetInstructions("FROM")))
	assert.Equal(t, 1, len(graph.GetInstructions("RUN")))
}

func TestDockerfileGraph_HasInstruction(t *testing.T) {
	graph := NewDockerfileGraph("Dockerfile")

	assert.False(t, graph.HasInstruction("USER"))

	userNode := NewDockerfileNode("USER", 5)
	graph.AddInstruction(userNode)

	assert.True(t, graph.HasInstruction("USER"))
}

func TestDockerfileGraph_GetFinalUser(t *testing.T) {
	graph := NewDockerfileGraph("Dockerfile")

	// No USER instruction
	assert.Nil(t, graph.GetFinalUser())

	// Add first USER
	user1 := NewDockerfileNode("USER", 5)
	user1.UserName = "root"
	graph.AddInstruction(user1)

	assert.Equal(t, user1, graph.GetFinalUser())

	// Add second USER - should be final
	user2 := NewDockerfileNode("USER", 10)
	user2.UserName = "appuser"
	graph.AddInstruction(user2)

	assert.Equal(t, user2, graph.GetFinalUser())
	assert.Equal(t, "appuser", graph.GetFinalUser().UserName)
}

func TestDockerfileGraph_IsRunningAsRoot(t *testing.T) {
	tests := []struct {
		name     string
		users    []string
		expected bool
	}{
		{"no USER instruction", nil, true},
		{"final user is root", []string{"appuser", "root"}, true},
		{"final user is non-root", []string{"root", "appuser"}, false},
		{"single non-root user", []string{"appuser"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewDockerfileGraph("Dockerfile")
			for i, user := range tt.users {
				node := NewDockerfileNode("USER", i+1)
				node.UserName = user
				graph.AddInstruction(node)
			}
			assert.Equal(t, tt.expected, graph.IsRunningAsRoot())
		})
	}
}

func TestDockerfileGraph_AnalyzeBuildStages(t *testing.T) {
	graph := NewDockerfileGraph("Dockerfile")

	// Stage 1: builder
	from1 := NewDockerfileNode("FROM", 1)
	from1.BaseImage = "golang"
	from1.ImageTag = "1.21"
	from1.StageAlias = "builder"
	graph.AddInstruction(from1)

	run1 := NewDockerfileNode("RUN", 2)
	graph.AddInstruction(run1)

	// Stage 2: final
	from2 := NewDockerfileNode("FROM", 5)
	from2.BaseImage = "alpine"
	from2.ImageTag = "3.18"
	graph.AddInstruction(from2)

	copyNode := NewDockerfileNode("COPY", 6)
	copyNode.Flags["from"] = "builder"
	graph.AddInstruction(copyNode)

	stages := graph.GetStages()

	assert.Equal(t, 2, len(stages))
	assert.Equal(t, "builder", stages[0].Alias)
	assert.Equal(t, "golang", stages[0].BaseImage)
	assert.Equal(t, "1.21", stages[0].ImageTag)
	assert.Equal(t, 2, len(stages[0].Instructions))

	assert.Equal(t, "", stages[1].Alias)
	assert.Equal(t, "alpine", stages[1].BaseImage)
	assert.Equal(t, 2, len(stages[1].Instructions))
}

func TestDockerfileGraph_IsMultiStage(t *testing.T) {
	// Single stage
	single := NewDockerfileGraph("Dockerfile")
	single.AddInstruction(NewDockerfileNode("FROM", 1))
	assert.False(t, single.IsMultiStage())

	// Multi-stage
	multi := NewDockerfileGraph("Dockerfile")
	multi.AddInstruction(NewDockerfileNode("FROM", 1))
	multi.AddInstruction(NewDockerfileNode("FROM", 5))
	assert.True(t, multi.IsMultiStage())
}

func TestDockerfileGraph_GetStageByAlias(t *testing.T) {
	graph := NewDockerfileGraph("Dockerfile")

	from1 := NewDockerfileNode("FROM", 1)
	from1.BaseImage = "golang"
	from1.StageAlias = "builder"
	graph.AddInstruction(from1)

	from2 := NewDockerfileNode("FROM", 5)
	from2.BaseImage = "alpine"
	graph.AddInstruction(from2)

	builder := graph.GetStageByAlias("builder")
	assert.NotNil(t, builder)
	assert.Equal(t, "golang", builder.BaseImage)

	nonexistent := graph.GetStageByAlias("nonexistent")
	assert.Nil(t, nonexistent)
}

func TestDockerfileGraph_GetFinalStage(t *testing.T) {
	graph := NewDockerfileGraph("Dockerfile")

	from1 := NewDockerfileNode("FROM", 1)
	from1.StageAlias = "builder"
	graph.AddInstruction(from1)

	from2 := NewDockerfileNode("FROM", 5)
	from2.BaseImage = "alpine"
	graph.AddInstruction(from2)

	final := graph.GetFinalStage()
	assert.NotNil(t, final)
	assert.Equal(t, "alpine", final.BaseImage)
}
