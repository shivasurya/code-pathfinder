package docker

// DockerfileNode represents a single instruction in a Dockerfile.
// This is a unified structure that handles all 18 Dockerfile instruction types.
//
// Design rationale:
// - Single type simplifies iteration and storage
// - Optional fields populated based on InstructionType
// - Flags map handles all instruction-specific flags (--from, --chown, etc.)
type DockerfileNode struct {
	// InstructionType identifies the instruction: FROM, RUN, COPY, ADD, ENV,
	// ARG, USER, EXPOSE, WORKDIR, CMD, ENTRYPOINT, VOLUME, SHELL, HEALTHCHECK,
	// LABEL, ONBUILD, STOPSIGNAL, MAINTAINER
	InstructionType string `json:"instructionType"`

	// RawInstruction contains the full instruction line (for debugging/display)
	RawInstruction string `json:"rawInstruction"`

	// Arguments contains parsed positional arguments
	Arguments []string `json:"arguments,omitempty"`

	// Flags contains instruction flags (e.g., --from=builder, --chown=user:group)
	// Key: flag name without --, Value: flag value
	Flags map[string]string `json:"flags,omitempty"`

	// LineNumber is the 1-indexed line in the Dockerfile
	LineNumber int `json:"lineNumber"`

	// StageIndex indicates which build stage (0-indexed, for multi-stage)
	StageIndex int `json:"stageIndex"`

	// StageAlias is the AS alias for this stage (only for FROM instructions)
	StageAlias string `json:"stageAlias,omitempty"`

	// --- FROM instruction fields ---
	BaseImage   string `json:"baseImage,omitempty"`
	ImageTag    string `json:"imageTag,omitempty"`
	ImageDigest string `json:"imageDigest,omitempty"`

	// --- USER instruction fields ---
	UserName  string `json:"userName,omitempty"`
	GroupName string `json:"groupName,omitempty"`

	// --- EXPOSE instruction fields ---
	Ports    []int  `json:"ports,omitempty"`
	Protocol string `json:"protocol,omitempty"` // tcp or udp

	// --- ENV/ARG instruction fields ---
	EnvVars map[string]string `json:"envVars,omitempty"`
	ArgName string            `json:"argName,omitempty"`

	// --- COPY/ADD instruction fields ---
	SourcePaths []string `json:"sourcePaths,omitempty"`
	DestPath    string   `json:"destPath,omitempty"`
	CopyFrom    string   `json:"copyFrom,omitempty"` // --from flag
	Chown       string   `json:"chown,omitempty"`    // --chown flag

	// --- HEALTHCHECK instruction fields ---
	HealthcheckType        string `json:"healthcheckType,omitempty"` // CMD or NONE
	HealthcheckCmd         string `json:"healthcheckCmd,omitempty"`
	HealthcheckInterval    string `json:"healthcheckInterval,omitempty"`
	HealthcheckTimeout     string `json:"healthcheckTimeout,omitempty"`
	HealthcheckStartPeriod string `json:"healthcheckStartPeriod,omitempty"`
	HealthcheckRetries     int    `json:"healthcheckRetries,omitempty"`

	// --- LABEL instruction fields ---
	Labels map[string]string `json:"labels,omitempty"`

	// --- CMD/ENTRYPOINT instruction fields ---
	CommandForm  string   `json:"commandForm,omitempty"` // "shell" or "exec"
	CommandArray []string `json:"commandArray,omitempty"`

	// --- WORKDIR instruction fields ---
	WorkDir        string `json:"workdir,omitempty"`
	IsAbsolutePath bool   `json:"isAbsolutePath,omitempty"`

	// --- VOLUME instruction fields ---
	Volumes []string `json:"volumes,omitempty"`

	// --- SHELL instruction fields ---
	Shell []string `json:"shell,omitempty"`

	// --- ONBUILD instruction fields ---
	OnBuildInstruction string `json:"onbuildInstruction,omitempty"`

	// --- STOPSIGNAL instruction fields ---
	StopSignal string `json:"stopSignal,omitempty"`

	// --- Multi-line tracking ---
	IsContinuation bool `json:"isContinuation,omitempty"`
}

// NewDockerfileNode creates a new DockerfileNode with initialized maps.
func NewDockerfileNode(instructionType string, lineNumber int) *DockerfileNode {
	return &DockerfileNode{
		InstructionType: instructionType,
		LineNumber:      lineNumber,
		Flags:           make(map[string]string),
		EnvVars:         make(map[string]string),
		Labels:          make(map[string]string),
		Arguments:       make([]string, 0),
	}
}

// GetFlag retrieves a flag value by name, returns empty string if not present.
func (n *DockerfileNode) GetFlag(name string) string {
	if n.Flags == nil {
		return ""
	}
	return n.Flags[name]
}

// HasFlag checks if a flag is present.
func (n *DockerfileNode) HasFlag(name string) bool {
	if n.Flags == nil {
		return false
	}
	_, exists := n.Flags[name]
	return exists
}

// IsRootUser checks if this is a USER instruction with root user.
func (n *DockerfileNode) IsRootUser() bool {
	return n.InstructionType == "USER" &&
		(n.UserName == "root" || n.UserName == "0")
}

// UsesLatestTag checks if this FROM instruction uses the :latest tag.
func (n *DockerfileNode) UsesLatestTag() bool {
	return n.InstructionType == "FROM" &&
		(n.ImageTag == "latest" || n.ImageTag == "")
}
