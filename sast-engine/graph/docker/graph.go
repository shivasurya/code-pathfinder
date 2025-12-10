package docker

// DockerfileGraph represents a complete parsed Dockerfile.
// It provides indexed access to instructions and multi-stage build analysis.
type DockerfileGraph struct {
	// All instructions in order of appearance
	Instructions []*DockerfileNode

	// Build stages (for multi-stage builds)
	Stages []*BuildStage

	// Index by instruction type for O(1) lookup
	InstructionIndex map[string][]*DockerfileNode

	// Environment variables accumulated during parsing
	Environment map[string]string

	// FinalUser is the last USER instruction (nil if none)
	FinalUser *DockerfileNode

	// Metadata
	FilePath          string
	TotalInstructions int
}

// BuildStage represents a single stage in a multi-stage Dockerfile.
type BuildStage struct {
	// Alias is the AS name (e.g., "builder" in "FROM golang AS builder")
	Alias string

	// BaseImage is the image name without tag
	BaseImage string

	// ImageTag is the tag (e.g., "1.21" in "golang:1.21")
	ImageTag string

	// Index is the 0-based stage index
	Index int

	// StartLine is the line number of the FROM instruction
	StartLine int

	// EndLine is the last line of this stage (before next FROM or EOF)
	EndLine int

	// Instructions contains all instructions in this stage
	Instructions []*DockerfileNode
}

// NewDockerfileGraph creates an empty DockerfileGraph.
func NewDockerfileGraph(filePath string) *DockerfileGraph {
	return &DockerfileGraph{
		Instructions:     make([]*DockerfileNode, 0),
		Stages:           make([]*BuildStage, 0),
		InstructionIndex: make(map[string][]*DockerfileNode),
		Environment:      make(map[string]string),
		FilePath:         filePath,
	}
}

// AddInstruction adds an instruction and updates indexes.
func (g *DockerfileGraph) AddInstruction(node *DockerfileNode) {
	g.Instructions = append(g.Instructions, node)
	g.InstructionIndex[node.InstructionType] = append(
		g.InstructionIndex[node.InstructionType], node)
	g.TotalInstructions++

	// Track final USER
	if node.InstructionType == "USER" {
		g.FinalUser = node
	}
}

// GetInstructions returns all instructions of a given type.
func (g *DockerfileGraph) GetInstructions(instructionType string) []*DockerfileNode {
	return g.InstructionIndex[instructionType]
}

// HasInstruction checks if any instruction of the given type exists.
func (g *DockerfileGraph) HasInstruction(instructionType string) bool {
	return len(g.InstructionIndex[instructionType]) > 0
}

// GetFinalUser returns the last USER instruction, or nil if none.
func (g *DockerfileGraph) GetFinalUser() *DockerfileNode {
	return g.FinalUser
}

// IsRunningAsRoot returns true if no USER instruction or last USER is root.
func (g *DockerfileGraph) IsRunningAsRoot() bool {
	if g.FinalUser == nil {
		return true // No USER instruction = runs as root
	}
	return g.FinalUser.IsRootUser()
}

// GetStages returns all build stages (analyzes if not already done).
func (g *DockerfileGraph) GetStages() []*BuildStage {
	if len(g.Stages) == 0 && len(g.Instructions) > 0 {
		g.AnalyzeBuildStages()
	}
	return g.Stages
}

// IsMultiStage returns true if Dockerfile has multiple FROM instructions.
func (g *DockerfileGraph) IsMultiStage() bool {
	return len(g.GetInstructions("FROM")) > 1
}

// AnalyzeBuildStages populates the Stages slice from Instructions.
func (g *DockerfileGraph) AnalyzeBuildStages() {
	g.Stages = make([]*BuildStage, 0)
	var currentStage *BuildStage
	stageIndex := 0

	for _, node := range g.Instructions {
		if node.InstructionType == "FROM" {
			// Close previous stage
			if currentStage != nil {
				currentStage.EndLine = node.LineNumber - 1
				g.Stages = append(g.Stages, currentStage)
			}

			// Start new stage
			currentStage = &BuildStage{
				Alias:        node.StageAlias,
				BaseImage:    node.BaseImage,
				ImageTag:     node.ImageTag,
				Index:        stageIndex,
				StartLine:    node.LineNumber,
				Instructions: make([]*DockerfileNode, 0),
			}
			stageIndex++
		}

		// Add instruction to current stage
		if currentStage != nil {
			currentStage.Instructions = append(currentStage.Instructions, node)
			node.StageIndex = currentStage.Index
		}
	}

	// Close final stage
	if currentStage != nil {
		if len(g.Instructions) > 0 {
			currentStage.EndLine = g.Instructions[len(g.Instructions)-1].LineNumber
		}
		g.Stages = append(g.Stages, currentStage)
	}
}

// GetStageByAlias finds a build stage by its AS alias.
func (g *DockerfileGraph) GetStageByAlias(alias string) *BuildStage {
	for _, stage := range g.GetStages() {
		if stage.Alias == alias {
			return stage
		}
	}
	return nil
}

// GetFinalStage returns the last build stage (the one that produces the image).
func (g *DockerfileGraph) GetFinalStage() *BuildStage {
	stages := g.GetStages()
	if len(stages) == 0 {
		return nil
	}
	return stages[len(stages)-1]
}
