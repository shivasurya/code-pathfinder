package graph

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
)

// parseDockerfile parses a Dockerfile and adds nodes to the CodeGraph.
// Each Dockerfile instruction becomes a node with unique ID including line and column.
func parseDockerfile(filePath string, graph *CodeGraph) error {
	// Create Docker parser
	parser := docker.NewDockerfileParser()

	// Parse Dockerfile
	dockerGraph, err := parser.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse Dockerfile: %w", err)
	}

	// Convert each instruction to a CodeGraph node
	for _, instruction := range dockerGraph.Instructions {
		node := convertDockerInstructionToNode(instruction, filePath)
		graph.AddNode(node)
	}

	return nil
}

// convertDockerInstructionToNode converts a DockerfileNode to a CodeGraph Node.
func convertDockerInstructionToNode(dockerNode *docker.DockerfileNode, filePath string) *Node {
	// Generate unique ID with line number and column (column is always 1 for Dockerfile instructions)
	// Format: "dockerfile:<file>:<instruction>:<line>:<column>"
	lineNumber := dockerNode.LineNumber
	columnNumber := 1 // Dockerfile instructions start at column 1
	nodeID := GenerateSha256(fmt.Sprintf("dockerfile:%s:%s:%d:%d",
		filePath, dockerNode.InstructionType, lineNumber, columnNumber))

	// Create CodeGraph node
	node := &Node{
		ID:                   nodeID,
		Type:                 "dockerfile_instruction",
		Name:                 dockerNode.InstructionType,
		LineNumber:           uint32(lineNumber),
		File:                 filePath,
		MethodArgumentsValue: []string{dockerNode.RawInstruction},
		SourceLocation: &SourceLocation{
			File:      filePath,
			StartByte: 0, // Will be set if we need lazy loading
			EndByte:   0,
		},
		Metadata: make(map[string]any),
	}

	// Store instruction-specific details in MethodArgumentsValue
	// This allows DSL rules to query instruction arguments
	node.MethodArgumentsValue = append(node.MethodArgumentsValue,
		extractDockerInstructionArgs(dockerNode)...)

	// Store stage information for multi-stage Dockerfiles
	if dockerNode.InstructionType == "FROM" {
		node.Metadata["stage_index"] = dockerNode.StageIndex
		if dockerNode.StageAlias != "" {
			node.Metadata["stage_name"] = dockerNode.StageAlias
		}
	}

	// Track COPY --from dependencies
	if dockerNode.InstructionType == "COPY" && dockerNode.CopyFrom != "" {
		node.Metadata["copy_from"] = dockerNode.CopyFrom
		node.Metadata["stage_index"] = dockerNode.StageIndex
	}

	return node
}

// extractDockerInstructionArgs extracts arguments from a Docker instruction.
func extractDockerInstructionArgs(dockerNode *docker.DockerfileNode) []string {
	args := []string{}

	switch dockerNode.InstructionType {
	case "FROM":
		if dockerNode.BaseImage != "" {
			args = append(args, dockerNode.BaseImage)
			if dockerNode.ImageTag != "" {
				args = append(args, dockerNode.ImageTag)
			}
			if dockerNode.StageAlias != "" {
				args = append(args, "AS", dockerNode.StageAlias)
			}
		}
	case "USER":
		if dockerNode.UserName != "" {
			args = append(args, dockerNode.UserName)
		}
		if dockerNode.GroupName != "" {
			args = append(args, dockerNode.GroupName)
		}
	case "EXPOSE":
		for _, port := range dockerNode.Ports {
			args = append(args, strconv.Itoa(port))
		}
	case "ENV":
		for key, value := range dockerNode.EnvVars {
			args = append(args, key+"="+value)
		}
	case "ARG":
		if dockerNode.ArgName != "" {
			args = append(args, dockerNode.ArgName)
		}
	case "LABEL":
		for key, value := range dockerNode.Labels {
			args = append(args, key+"="+value)
		}
	case "RUN", "CMD", "ENTRYPOINT":
		if len(dockerNode.CommandArray) > 0 {
			args = append(args, dockerNode.CommandArray...)
		} else {
			args = append(args, dockerNode.Arguments...)
		}
	case "COPY", "ADD":
		if len(dockerNode.SourcePaths) > 0 {
			args = append(args, dockerNode.SourcePaths...)
		}
		if dockerNode.DestPath != "" {
			args = append(args, dockerNode.DestPath)
		}
	case "WORKDIR":
		if dockerNode.WorkDir != "" {
			args = append(args, dockerNode.WorkDir)
		}
	case "VOLUME":
		args = append(args, dockerNode.Volumes...)
	case "HEALTHCHECK":
		if dockerNode.HealthcheckCmd != "" {
			args = append(args, dockerNode.HealthcheckCmd)
		}
	case "SHELL":
		args = append(args, dockerNode.Shell...)
	case "ONBUILD":
		if dockerNode.OnBuildInstruction != "" {
			args = append(args, dockerNode.OnBuildInstruction)
		}
	case "STOPSIGNAL":
		if dockerNode.StopSignal != "" {
			args = append(args, dockerNode.StopSignal)
		}
	}

	return args
}

// parseDockerCompose parses a docker-compose.yml file and adds nodes to the CodeGraph.
// Each service becomes a node with unique ID including line number.
func parseDockerCompose(filePath string, graph *CodeGraph) error {
	// Parse docker-compose file
	composeGraph, err := ParseDockerCompose(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse docker-compose: %w", err)
	}

	// Convert each service to a CodeGraph node
	for serviceName, serviceNode := range composeGraph.Services {
		node := convertComposeServiceToNode(serviceName, serviceNode, filePath)
		graph.AddNode(node)
	}

	return nil
}

// convertComposeServiceToNode converts a docker-compose service to a CodeGraph Node.
func convertComposeServiceToNode(serviceName string, serviceNode *YAMLNode, filePath string) *Node {
	// Generate unique ID with line number (YAML doesn't provide column, default to 1)
	// For YAML, we don't have exact line numbers from the parser, so we use a hash
	// Format: "compose:<file>:<service>"
	nodeID := GenerateSha256(fmt.Sprintf("compose:%s:%s", filePath, serviceName))

	// Create CodeGraph node
	node := &Node{
		ID:         nodeID,
		Type:       "compose_service",
		Name:       serviceName,
		LineNumber: 1, // YAML parser doesn't provide line numbers, would need enhancement
		File:       filePath,
		SourceLocation: &SourceLocation{
			File:      filePath,
			StartByte: 0,
			EndByte:   0,
		},
		Metadata: make(map[string]any),
	}

	// Extract service properties and store in MethodArgumentsValue
	// This allows DSL rules to query service configuration
	node.MethodArgumentsValue = extractComposeServiceProperties(serviceNode)

	// Extract and store depends_on in Metadata for dependency graph traversal
	depends := []any{}
	for _, prop := range node.MethodArgumentsValue {
		if after, ok := strings.CutPrefix(prop, "depends_on="); ok {
			depends = append(depends, after)
		}
	}
	if len(depends) > 0 {
		node.Metadata["depends_on"] = depends
	}
	node.Metadata["service_type"] = "compose_service"

	return node
}

// extractComposeServiceProperties extracts properties from a docker-compose service.
func extractComposeServiceProperties(serviceNode *YAMLNode) []string {
	props := []string{}

	// Extract common security-relevant properties
	if imageNode := serviceNode.GetChild("image"); imageNode != nil {
		props = append(props, "image="+imageNode.StringValue())
	}

	if privileged := serviceNode.GetChild("privileged"); privileged != nil && privileged.BoolValue() {
		props = append(props, "privileged=true")
	}

	if networkMode := serviceNode.GetChild("network_mode"); networkMode != nil {
		props = append(props, "network_mode="+networkMode.StringValue())
	}

	if readOnly := serviceNode.GetChild("read_only"); readOnly != nil && readOnly.BoolValue() {
		props = append(props, "read_only=true")
	}

	// Extract volumes (check for Docker socket exposure)
	if volumesNode := serviceNode.GetChild("volumes"); volumesNode != nil {
		for _, vol := range volumesNode.ListValues() {
			if volStr, ok := vol.(string); ok {
				props = append(props, "volume="+volStr)
			}
		}
	}

	// Extract ports
	if portsNode := serviceNode.GetChild("ports"); portsNode != nil {
		for _, port := range portsNode.ListValues() {
			if portStr, ok := port.(string); ok {
				props = append(props, "port="+portStr)
			}
		}
	}

	// Extract capabilities
	if capAdd := serviceNode.GetChild("cap_add"); capAdd != nil {
		for _, cap := range capAdd.ListValues() {
			if capStr, ok := cap.(string); ok {
				props = append(props, "cap_add="+capStr)
			}
		}
	}

	if capDrop := serviceNode.GetChild("cap_drop"); capDrop != nil {
		for _, cap := range capDrop.ListValues() {
			if capStr, ok := cap.(string); ok {
				props = append(props, "cap_drop="+capStr)
			}
		}
	}

	// Extract security_opt
	if secOpt := serviceNode.GetChild("security_opt"); secOpt != nil {
		for _, opt := range secOpt.ListValues() {
			if optStr, ok := opt.(string); ok {
				props = append(props, "security_opt="+optStr)
			}
		}
	}

	// Extract environment variables
	if envNode := serviceNode.GetChild("environment"); envNode != nil {
		// Handle map format
		if envNode.Children != nil {
			for key := range envNode.Children {
				props = append(props, "env="+key)
			}
		}
		// Handle array format
		for _, env := range envNode.ListValues() {
			if envStr, ok := env.(string); ok {
				props = append(props, "env="+envStr)
			}
		}
	}

	// Extract depends_on (for dependency graph)
	if dependsNode := serviceNode.GetChild("depends_on"); dependsNode != nil {
		// depends_on can be array format: ["db", "redis"]
		// or object format: {db: {condition: service_healthy}}
		dependsList := []string{}

		// Handle array format
		for _, dep := range dependsNode.ListValues() {
			if depStr, ok := dep.(string); ok {
				dependsList = append(dependsList, depStr)
			}
		}

		// Handle object format (keys are service names)
		if dependsNode.Children != nil {
			for serviceName := range dependsNode.Children {
				// Avoid duplicates from array format
				found := slices.Contains(dependsList, serviceName)
				if !found {
					dependsList = append(dependsList, serviceName)
				}
			}
		}

		// Store in properties
		for _, dep := range dependsList {
			props = append(props, "depends_on="+dep)
		}
	}

	// Extract build context (for dependency analysis)
	if buildNode := serviceNode.GetChild("build"); buildNode != nil {
		// build can be string (context path) or object
		if buildStr := buildNode.StringValue(); buildStr != "" {
			props = append(props, "build="+buildStr)
		} else if contextNode := buildNode.GetChild("context"); contextNode != nil {
			props = append(props, "build="+contextNode.StringValue())
		}
	}

	return props
}

// Helper functions to query Docker/Compose nodes (for DSL executor)

// IsDockerNode checks if a node represents a Dockerfile instruction.
func IsDockerNode(node *Node) bool {
	return node.Type == "dockerfile_instruction"
}

// IsComposeNode checks if a node represents a docker-compose service.
func IsComposeNode(node *Node) bool {
	return node.Type == "compose_service"
}

// GetDockerInstructionType returns the instruction type for Docker nodes (e.g., "RUN", "FROM").
func GetDockerInstructionType(node *Node) string {
	if !IsDockerNode(node) {
		return ""
	}
	return node.Name
}

// HasDockerInstructionArg checks if a Docker node has a specific argument.
func HasDockerInstructionArg(node *Node, arg string) bool {
	if !IsDockerNode(node) {
		return false
	}
	return slices.Contains(node.MethodArgumentsValue, arg)
}

// GetComposeServiceProperty gets a property value from a compose service node.
func GetComposeServiceProperty(node *Node, property string) string {
	if !IsComposeNode(node) {
		return ""
	}
	prefix := property + "="
	for _, value := range node.MethodArgumentsValue {
		if len(value) > len(prefix) && value[:len(prefix)] == prefix {
			return value[len(prefix):]
		}
	}
	return ""
}

// HasComposeServiceProperty checks if a compose service has a specific property.
func HasComposeServiceProperty(node *Node, property string, expectedValue ...string) bool {
	if !IsComposeNode(node) {
		return false
	}

	if len(expectedValue) == 0 {
		// Just check if property exists
		prefix := property + "="
		for _, value := range node.MethodArgumentsValue {
			if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
				return true
			}
		}
		return false
	}

	// Check for specific value
	expected := property + "=" + expectedValue[0]
	return slices.Contains(node.MethodArgumentsValue, expected)
}
