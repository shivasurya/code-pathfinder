package graph

import (
	"maps"
	"strconv"
	"strings"
)

// ComposeGraph wraps a YAMLGraph with docker-compose specific indexing.
type ComposeGraph struct {
	// Embedded YAML graph
	YAMLGraph *YAMLGraph

	// Compose-specific indexes
	Services map[string]*YAMLNode
	Volumes  map[string]*YAMLNode
	Networks map[string]*YAMLNode

	// Metadata
	Version  string
	FilePath string
}

// ParseDockerCompose parses a docker-compose.yml file.
func ParseDockerCompose(filePath string) (*ComposeGraph, error) {
	yamlGraph, err := ParseYAML(filePath)
	if err != nil {
		return nil, err
	}

	return NewComposeGraph(yamlGraph, filePath), nil
}

// NewComposeGraph creates a ComposeGraph from a YAMLGraph.
func NewComposeGraph(yamlGraph *YAMLGraph, filePath string) *ComposeGraph {
	compose := &ComposeGraph{
		YAMLGraph: yamlGraph,
		Services:  make(map[string]*YAMLNode),
		Volumes:   make(map[string]*YAMLNode),
		Networks:  make(map[string]*YAMLNode),
		FilePath:  filePath,
	}

	// Index services
	servicesNode := yamlGraph.Query("services")
	if servicesNode != nil && servicesNode.Children != nil {
		maps.Copy(compose.Services, servicesNode.Children)
	}

	// Index volumes
	volumesNode := yamlGraph.Query("volumes")
	if volumesNode != nil && volumesNode.Children != nil {
		maps.Copy(compose.Volumes, volumesNode.Children)
	}

	// Index networks
	networksNode := yamlGraph.Query("networks")
	if networksNode != nil && networksNode.Children != nil {
		maps.Copy(compose.Networks, networksNode.Children)
	}

	// Get version
	versionNode := yamlGraph.Query("version")
	if versionNode != nil {
		compose.Version = versionNode.StringValue()
	}

	return compose
}

// --- Service Query Methods ---

// GetServices returns all service names.
func (c *ComposeGraph) GetServices() []string {
	names := make([]string, 0, len(c.Services))
	for name := range c.Services {
		names = append(names, name)
	}
	return names
}

// ServiceHas checks if a service has a property with specified value.
func (c *ComposeGraph) ServiceHas(serviceName, key string, value any) bool {
	service, exists := c.Services[serviceName]
	if !exists {
		return false
	}
	return c.nodeHasValue(service, key, value)
}

// ServiceHasKey checks if a service has a property defined.
func (c *ComposeGraph) ServiceHasKey(serviceName, key string) bool {
	service, exists := c.Services[serviceName]
	if !exists {
		return false
	}
	return service.HasChild(key)
}

// ServiceGet retrieves a service property value.
func (c *ComposeGraph) ServiceGet(serviceName, key string) any {
	service, exists := c.Services[serviceName]
	if !exists {
		return nil
	}
	child := service.GetChild(key)
	if child == nil {
		return nil
	}
	return child.Value
}

// ServiceGetLineNumber retrieves the line number for a service property.
// Returns the line number of the property's value, or the service line if property doesn't exist.
func (c *ComposeGraph) ServiceGetLineNumber(serviceName, key string) int {
	service, exists := c.Services[serviceName]
	if !exists {
		return 1
	}

	// Try to get the specific property
	child := service.GetChild(key)
	if child != nil && child.LineNumber > 0 {
		return child.LineNumber
	}

	// Fall back to service line number
	if service.LineNumber > 0 {
		return service.LineNumber
	}

	return 1
}

// --- Security Query Methods ---

// GetPrivilegedServices returns services with privileged: true.
func (c *ComposeGraph) GetPrivilegedServices() []string {
	privileged := make([]string, 0)
	for name, service := range c.Services {
		if c.nodeHasValue(service, "privileged", true) {
			privileged = append(privileged, name)
		}
	}
	return privileged
}

// ServicesWithDockerSocket returns services that mount Docker socket.
func (c *ComposeGraph) ServicesWithDockerSocket() []string {
	exposed := make([]string, 0)

	for name, service := range c.Services {
		volumesNode := service.GetChild("volumes")
		if volumesNode == nil {
			continue
		}

		for _, vol := range volumesNode.ListValues() {
			volStr, ok := vol.(string)
			if !ok {
				continue
			}
			if strings.Contains(volStr, "/var/run/docker.sock") ||
				strings.Contains(volStr, "/run/docker.sock") {
				exposed = append(exposed, name)
				break
			}
		}
	}

	return exposed
}

// ServiceHasSecurityOpt checks for specific security_opt value.
func (c *ComposeGraph) ServiceHasSecurityOpt(serviceName, optValue string) bool {
	service, exists := c.Services[serviceName]
	if !exists {
		return false
	}

	secOptNode := service.GetChild("security_opt")
	if secOptNode == nil {
		return false
	}

	for _, opt := range secOptNode.ListValues() {
		if optStr, ok := opt.(string); ok && optStr == optValue {
			return true
		}
	}

	return false
}

// ServiceHasCapability checks for capability in cap_add or cap_drop.
func (c *ComposeGraph) ServiceHasCapability(serviceName, capability, capType string) bool {
	service, exists := c.Services[serviceName]
	if !exists {
		return false
	}

	capNode := service.GetChild(capType) // "cap_add" or "cap_drop"
	if capNode == nil {
		return false
	}

	for _, cap := range capNode.ListValues() {
		if capStr, ok := cap.(string); ok && capStr == capability {
			return true
		}
	}

	return false
}

// ServicesWithHostNetwork returns services using network_mode: host.
func (c *ComposeGraph) ServicesWithHostNetwork() []string {
	hostMode := make([]string, 0)
	for name, service := range c.Services {
		if c.nodeHasValue(service, "network_mode", "host") {
			hostMode = append(hostMode, name)
		}
	}
	return hostMode
}

// ServicesWithoutReadOnly returns services without read_only: true.
func (c *ComposeGraph) ServicesWithoutReadOnly() []string {
	writable := make([]string, 0)
	for name, service := range c.Services {
		if !service.HasChild("read_only") {
			writable = append(writable, name)
		} else if !c.nodeHasValue(service, "read_only", true) {
			writable = append(writable, name)
		}
	}
	return writable
}

// ServiceExposesPort checks if a service exposes a specific port.
func (c *ComposeGraph) ServiceExposesPort(serviceName string, port int) bool {
	service, exists := c.Services[serviceName]
	if !exists {
		return false
	}

	portsNode := service.GetChild("ports")
	if portsNode == nil {
		return false
	}

	for _, portMapping := range portsNode.ListValues() {
		portStr, ok := portMapping.(string)
		if !ok {
			continue
		}
		// Parse formats: "8080:80", "8080", "8080:80/tcp"
		parts := strings.SplitSeq(strings.Split(portStr, "/")[0], ":")
		for p := range parts {
			if portNum, err := strconv.Atoi(p); err == nil && portNum == port {
				return true
			}
		}
	}

	return false
}

// ServiceHasEnvVar checks if service has environment variable.
func (c *ComposeGraph) ServiceHasEnvVar(serviceName, varName string) bool {
	service, exists := c.Services[serviceName]
	if !exists {
		return false
	}

	envNode := service.GetChild("environment")
	if envNode == nil {
		return false
	}

	// Handle map format: {VAR: value}
	if envNode.Children != nil {
		if _, exists := envNode.Children[varName]; exists {
			return true
		}
	}

	// Handle array format: ["VAR=value"]
	for _, env := range envNode.ListValues() {
		if envStr, ok := env.(string); ok {
			if strings.HasPrefix(envStr, varName+"=") || envStr == varName {
				return true
			}
		}
	}

	return false
}

// --- Helper Methods ---

func (c *ComposeGraph) nodeHasValue(node *YAMLNode, key string, expected any) bool {
	if node == nil {
		return false
	}
	child := node.GetChild(key)
	if child == nil {
		return false
	}
	return child.Value == expected
}
