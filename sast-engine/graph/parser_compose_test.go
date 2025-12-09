package graph

import (
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

// Helper to parse YAML string for testing.
func parseComposeFromString(yaml string) *ComposeGraph {
	yamlGraph, _ := ParseYAMLString(yaml)
	return NewComposeGraph(yamlGraph, "docker-compose.yml")
}
