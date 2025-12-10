package graph

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewComposeGraph(t *testing.T) {
	yaml := `
version: "3.8"
services:
  web:
    image: nginx
  db:
    image: postgres
volumes:
  data:
networks:
  backend:
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, "3.8", graph.Version)
	assert.Equal(t, 2, len(graph.Services))
	assert.Equal(t, 1, len(graph.Volumes))
	assert.Equal(t, 1, len(graph.Networks))
}

func TestComposeGraph_GetServices(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
  db:
    image: postgres
`
	graph := parseComposeFromString(yaml)
	services := graph.GetServices()

	assert.Contains(t, services, "web")
	assert.Contains(t, services, "db")
}

func TestComposeGraph_GetPrivilegedServices(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    privileged: true
  db:
    image: postgres
  admin:
    image: alpine
    privileged: true
`
	graph := parseComposeFromString(yaml)
	privileged := graph.GetPrivilegedServices()

	assert.Equal(t, 2, len(privileged))
	assert.Contains(t, privileged, "web")
	assert.Contains(t, privileged, "admin")
}

func TestComposeGraph_ServicesWithDockerSocket(t *testing.T) {
	yaml := `
services:
  dind:
    image: docker:dind
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
  web:
    image: nginx
    volumes:
      - ./html:/usr/share/nginx/html
`
	graph := parseComposeFromString(yaml)
	exposed := graph.ServicesWithDockerSocket()

	assert.Equal(t, 1, len(exposed))
	assert.Contains(t, exposed, "dind")
}

func TestComposeGraph_ServicesWithDockerSocket_RunVariant(t *testing.T) {
	yaml := `
services:
  dind:
    image: docker:dind
    volumes:
      - /run/docker.sock:/run/docker.sock
`
	graph := parseComposeFromString(yaml)
	exposed := graph.ServicesWithDockerSocket()

	assert.Equal(t, 1, len(exposed))
	assert.Contains(t, exposed, "dind")
}

func TestComposeGraph_ServiceHasSecurityOpt(t *testing.T) {
	yaml := `
services:
  insecure:
    image: alpine
    security_opt:
      - seccomp:unconfined
  secure:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHasSecurityOpt("insecure", "seccomp:unconfined"))
	assert.False(t, graph.ServiceHasSecurityOpt("secure", "seccomp:unconfined"))
}

func TestComposeGraph_ServiceHasCapability(t *testing.T) {
	yaml := `
services:
  privileged:
    image: alpine
    cap_add:
      - SYS_ADMIN
      - NET_ADMIN
  normal:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHasCapability("privileged", "SYS_ADMIN", "cap_add"))
	assert.True(t, graph.ServiceHasCapability("privileged", "NET_ADMIN", "cap_add"))
	assert.False(t, graph.ServiceHasCapability("normal", "SYS_ADMIN", "cap_add"))
}

func TestComposeGraph_ServicesWithHostNetwork(t *testing.T) {
	yaml := `
services:
  host:
    image: alpine
    network_mode: host
  bridge:
    image: nginx
`
	graph := parseComposeFromString(yaml)
	hostMode := graph.ServicesWithHostNetwork()

	assert.Equal(t, 1, len(hostMode))
	assert.Contains(t, hostMode, "host")
}

func TestComposeGraph_ServiceExposesPort(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    ports:
      - "80:80"
      - "443:443"
  backend:
    image: app
    ports:
      - "8080"
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceExposesPort("web", 80))
	assert.True(t, graph.ServiceExposesPort("web", 443))
	assert.True(t, graph.ServiceExposesPort("backend", 8080))
	assert.False(t, graph.ServiceExposesPort("web", 8080))
}

func TestComposeGraph_ServiceExposesPort_WithProtocol(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    ports:
      - "80:80/tcp"
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceExposesPort("web", 80))
}

func TestComposeGraph_ServiceHasEnvVar_ArrayFormat(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    environment:
      - DATABASE_URL=postgres://...
      - SECRET_KEY
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHasEnvVar("web", "DATABASE_URL"))
	assert.True(t, graph.ServiceHasEnvVar("web", "SECRET_KEY"))
	assert.False(t, graph.ServiceHasEnvVar("web", "POSTGRES_PASSWORD"))
}

func TestComposeGraph_ServiceHasEnvVar_MapFormat(t *testing.T) {
	yaml := `
services:
  db:
    image: postgres
    environment:
      POSTGRES_PASSWORD: secret
      POSTGRES_USER: admin
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHasEnvVar("db", "POSTGRES_PASSWORD"))
	assert.True(t, graph.ServiceHasEnvVar("db", "POSTGRES_USER"))
	assert.False(t, graph.ServiceHasEnvVar("db", "SECRET_KEY"))
}

func TestComposeGraph_ServicesWithoutReadOnly(t *testing.T) {
	yaml := `
services:
  secure:
    image: nginx
    read_only: true
  insecure:
    image: alpine
  writable:
    image: app
    read_only: false
`
	graph := parseComposeFromString(yaml)
	writable := graph.ServicesWithoutReadOnly()

	assert.Contains(t, writable, "insecure")
	assert.Contains(t, writable, "writable")
	assert.NotContains(t, writable, "secure")
}

func TestComposeGraph_ServiceHas(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    restart: always
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHas("web", "restart", "always"))
	assert.False(t, graph.ServiceHas("web", "restart", "never"))
}

func TestComposeGraph_ServiceHasKey(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    restart: always
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHasKey("web", "restart"))
	assert.True(t, graph.ServiceHasKey("web", "image"))
	assert.False(t, graph.ServiceHasKey("web", "privileged"))
}

func TestComposeGraph_ServiceGet(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    restart: always
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, "nginx", graph.ServiceGet("web", "image"))
	assert.Equal(t, "always", graph.ServiceGet("web", "restart"))
	assert.Nil(t, graph.ServiceGet("web", "nonexistent"))
	assert.Nil(t, graph.ServiceGet("nonexistent", "image"))
}

func TestComposeGraph_EmptyCompose(t *testing.T) {
	yaml := `
version: "3.8"
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, "3.8", graph.Version)
	assert.Equal(t, 0, len(graph.Services))
	assert.Equal(t, 0, len(graph.Volumes))
	assert.Equal(t, 0, len(graph.Networks))
}

func TestComposeGraph_NoServicesSection(t *testing.T) {
	yaml := `
version: "3.8"
volumes:
  data:
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, 0, len(graph.Services))
	assert.Equal(t, 1, len(graph.Volumes))
}

func TestParseDockerCompose_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := tmpDir + "/docker-compose.yml"
	content := `
version: "3.8"
services:
  web:
    image: nginx
`
	err := os.WriteFile(composePath, []byte(content), 0644)
	assert.NoError(t, err)

	graph, err := ParseDockerCompose(composePath)
	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, composePath, graph.FilePath)
	assert.Equal(t, 1, len(graph.Services))
}

func TestParseDockerCompose_FileNotFound(t *testing.T) {
	graph, err := ParseDockerCompose("/nonexistent/docker-compose.yml")
	assert.Error(t, err)
	assert.Nil(t, graph)
}

func TestComposeGraph_ServiceHas_NonExistentService(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceHas("nonexistent", "image", "nginx"))
}

func TestComposeGraph_ServiceGet_DifferentValueTypes(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    replicas: 3
    privileged: true
`
	graph := parseComposeFromString(yaml)

	// String value
	assert.Equal(t, "nginx", graph.ServiceGet("web", "image"))
	// Int value
	assert.Equal(t, 3, graph.ServiceGet("web", "replicas"))
	// Bool value
	assert.Equal(t, true, graph.ServiceGet("web", "privileged"))
}

func TestComposeGraph_GetPrivilegedServices_NonePrivileged(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
  db:
    image: postgres
`
	graph := parseComposeFromString(yaml)
	privileged := graph.GetPrivilegedServices()

	assert.Equal(t, 0, len(privileged))
}

func TestComposeGraph_ServicesWithDockerSocket_NoVolumes(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)
	exposed := graph.ServicesWithDockerSocket()

	assert.Equal(t, 0, len(exposed))
}

func TestComposeGraph_ServicesWithDockerSocket_NonStringVolume(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    volumes:
      data: /data
`
	graph := parseComposeFromString(yaml)
	exposed := graph.ServicesWithDockerSocket()

	assert.Equal(t, 0, len(exposed))
}

func TestComposeGraph_ServiceHasSecurityOpt_NoSecurityOpt(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceHasSecurityOpt("web", "seccomp:unconfined"))
}

func TestComposeGraph_ServiceHasCapability_NoCapabilities(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceHasCapability("web", "SYS_ADMIN", "cap_add"))
}

func TestComposeGraph_ServiceHasCapability_NonExistentService(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceHasCapability("nonexistent", "SYS_ADMIN", "cap_add"))
}

func TestComposeGraph_ServiceHasCapability_CapDrop(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    cap_drop:
      - ALL
`
	graph := parseComposeFromString(yaml)

	assert.True(t, graph.ServiceHasCapability("web", "ALL", "cap_drop"))
	assert.False(t, graph.ServiceHasCapability("web", "SYS_ADMIN", "cap_drop"))
}

func TestComposeGraph_ServicesWithHostNetwork_None(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)
	hostMode := graph.ServicesWithHostNetwork()

	assert.Equal(t, 0, len(hostMode))
}

func TestComposeGraph_ServiceExposesPort_NoPorts(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceExposesPort("web", 80))
}

func TestComposeGraph_ServiceExposesPort_NonStringPort(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    ports:
      80: 80
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceExposesPort("web", 80))
}

func TestComposeGraph_ServiceExposesPort_InvalidPortFormat(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    ports:
      - "invalid"
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceExposesPort("web", 80))
}

func TestComposeGraph_ServiceHasEnvVar_NoEnv(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceHasEnvVar("web", "DATABASE_URL"))
}

func TestComposeGraph_ServiceHasEnvVar_NonExistentService(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.False(t, graph.ServiceHasEnvVar("nonexistent", "VAR"))
}

func TestComposeGraph_ServicesWithoutReadOnly_ExplicitTrue(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    read_only: true
`
	graph := parseComposeFromString(yaml)
	writable := graph.ServicesWithoutReadOnly()

	assert.Equal(t, 0, len(writable))
}

func TestComposeGraph_NoVersion(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, "", graph.Version)
}

func TestComposeGraph_NetworksIndexing(t *testing.T) {
	yaml := `
networks:
  frontend:
  backend:
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, 2, len(graph.Networks))
	assert.NotNil(t, graph.Networks["frontend"])
	assert.NotNil(t, graph.Networks["backend"])
}

func TestComposeGraph_VolumesIndexing(t *testing.T) {
	yaml := `
volumes:
  data:
  logs:
`
	graph := parseComposeFromString(yaml)

	assert.Equal(t, 2, len(graph.Volumes))
	assert.NotNil(t, graph.Volumes["data"])
	assert.NotNil(t, graph.Volumes["logs"])
}

func TestComposeGraph_ServiceGetLineNumber(t *testing.T) {
	yaml := `version: '3.8'
services:
  web:
    image: nginx
    ports:
      - "80:80"
    privileged: true
    volumes:
      - ./data:/data
`
	graph := parseComposeFromString(yaml)

	t.Run("returns line number for existing property", func(t *testing.T) {
		line := graph.ServiceGetLineNumber("web", "privileged")
		assert.Greater(t, line, 1)
		assert.LessOrEqual(t, line, 10)
	})

	t.Run("returns service line for missing property", func(t *testing.T) {
		line := graph.ServiceGetLineNumber("web", "read_only")
		// Should return service line (not 0, not 1)
		assert.Greater(t, line, 1)
	})

	t.Run("returns 1 for nonexistent service", func(t *testing.T) {
		line := graph.ServiceGetLineNumber("nonexistent", "any")
		assert.Equal(t, 1, line)
	})

	t.Run("returns line number for nested property", func(t *testing.T) {
		line := graph.ServiceGetLineNumber("web", "ports")
		assert.Greater(t, line, 1)
	})

	t.Run("returns service line for empty key", func(t *testing.T) {
		line := graph.ServiceGetLineNumber("web", "")
		assert.Greater(t, line, 1)
	})
}

// Helper to parse YAML string for testing.
func parseComposeFromString(yaml string) *ComposeGraph {
	yamlGraph, _ := ParseYAMLString(yaml)
	return NewComposeGraph(yamlGraph, "docker-compose.yml")
}
