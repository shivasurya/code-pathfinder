package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSemanticDockerTestServer creates a Server with comprehensive Docker test fixtures.
func createSemanticDockerTestServer() *Server {
	callGraph := core.NewCallGraph()

	// Add a baseline function node
	callGraph.Functions["myapp.main"] = &graph.Node{
		ID:         "func1",
		Type:       "function_definition",
		Name:       "main",
		File:       "/test/app.py",
		LineNumber: 10,
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"myapp": "/test/app.py"},
		FileToModule: map[string]string{"/test/app.py": "myapp"},
		ShortNames:   map[string][]string{"myapp": {"/test/app.py"}},
	}

	// Create CodeGraph with comprehensive Docker nodes
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			// FROM instructions (unpinned and pinned)
			"from1": {
				ID:         "from1",
				Type:       "dockerfile_instruction",
				Name:       "FROM",
				File:       "/test/Dockerfile",
				LineNumber: 1,
				MethodArgumentsValue: []string{
					"FROM python:3.11-slim",
					"python:3.11-slim",
				},
			},
			"from2": {
				ID:         "from2",
				Type:       "dockerfile_instruction",
				Name:       "FROM",
				File:       "/test/Dockerfile",
				LineNumber: 10,
				MethodArgumentsValue: []string{
					"FROM alpine:3.18@sha256:abc123 AS builder",
					"alpine:3.18@sha256:abc123",
					"AS",
					"builder",
				},
			},
			"from3": {
				ID:         "from3",
				Type:       "dockerfile_instruction",
				Name:       "FROM",
				File:       "/test/Dockerfile",
				LineNumber: 20,
				MethodArgumentsValue: []string{
					"FROM node:18",
					"node:18",
				},
			},
			// USER instructions (root and non-root)
			"user1": {
				ID:         "user1",
				Type:       "dockerfile_instruction",
				Name:       "USER",
				File:       "/test/Dockerfile",
				LineNumber: 8,
				MethodArgumentsValue: []string{
					"USER appuser:appgroup",
					"appuser",
					"appgroup",
				},
			},
			"user2": {
				ID:         "user2",
				Type:       "dockerfile_instruction",
				Name:       "USER",
				File:       "/test/Dockerfile.root",
				LineNumber: 5,
				MethodArgumentsValue: []string{
					"USER root",
					"root",
				},
			},
			// EXPOSE instructions
			"expose1": {
				ID:         "expose1",
				Type:       "dockerfile_instruction",
				Name:       "EXPOSE",
				File:       "/test/Dockerfile",
				LineNumber: 9,
				MethodArgumentsValue: []string{
					"EXPOSE 8080/tcp",
					"8080",
				},
			},
			"expose2": {
				ID:         "expose2",
				Type:       "dockerfile_instruction",
				Name:       "EXPOSE",
				File:       "/test/Dockerfile",
				LineNumber: 10,
				MethodArgumentsValue: []string{
					"EXPOSE 443/tcp",
					"443",
				},
			},
			// WORKDIR instruction
			"workdir1": {
				ID:         "workdir1",
				Type:       "dockerfile_instruction",
				Name:       "WORKDIR",
				File:       "/test/Dockerfile",
				LineNumber: 3,
				MethodArgumentsValue: []string{
					"WORKDIR /app",
					"/app",
				},
			},
			// COPY instruction with flags
			"copy1": {
				ID:         "copy1",
				Type:       "dockerfile_instruction",
				Name:       "COPY",
				File:       "/test/Dockerfile",
				LineNumber: 15,
				MethodArgumentsValue: []string{
					"COPY --from=builder --chown=appuser:appgroup /build /app",
					"--from=builder",
					"--chown=appuser:appgroup",
					"/build",
					"/app",
				},
			},
			// RUN instruction
			"run1": {
				ID:         "run1",
				Type:       "dockerfile_instruction",
				Name:       "RUN",
				File:       "/test/Dockerfile",
				LineNumber: 4,
				MethodArgumentsValue: []string{
					"RUN pip install -r requirements.txt",
				},
			},
			// HEALTHCHECK instruction
			"healthcheck1": {
				ID:         "healthcheck1",
				Type:       "dockerfile_instruction",
				Name:       "HEALTHCHECK",
				File:       "/test/Dockerfile",
				LineNumber: 11,
				MethodArgumentsValue: []string{
					"HEALTHCHECK CMD curl -f http://localhost:8080/health",
				},
			},
			// Compose services (with various security configurations)
			"compose1": {
				ID:         "compose1",
				Type:       "compose_service",
				Name:       "web",
				File:       "/test/docker-compose.yml",
				LineNumber: 3,
				MethodArgumentsValue: []string{
					"image=nginx:latest",
					"port=8080:80",
					"port=443:443",
					"volume=/app/data:/data",
				},
			},
			"compose2": {
				ID:         "compose2",
				Type:       "compose_service",
				Name:       "db",
				File:       "/test/docker-compose.yml",
				LineNumber: 10,
				MethodArgumentsValue: []string{
					"image=postgres:15",
					"port=5432:5432",
					"env=POSTGRES_PASSWORD=secret",
				},
			},
			"compose3": {
				ID:         "compose3",
				Type:       "compose_service",
				Name:       "privileged-service",
				File:       "/test/docker-compose.yml",
				LineNumber: 20,
				MethodArgumentsValue: []string{
					"image=alpine:latest",
					"privileged=true",
					"volume=/var/run/docker.sock:/var/run/docker.sock",
				},
			},
			"compose4": {
				ID:         "compose4",
				Type:       "compose_service",
				Name:       "host-network-service",
				File:       "/test/docker-compose.yml",
				LineNumber: 30,
				MethodArgumentsValue: []string{
					"image=ubuntu:latest",
					"network_mode=host",
				},
			},
			"compose5": {
				ID:         "compose5",
				Type:       "compose_service",
				Name:       "dangerous-caps-service",
				File:       "/test/docker-compose.yml",
				LineNumber: 40,
				MethodArgumentsValue: []string{
					"image=debian:latest",
					"cap_add=SYS_ADMIN",
					"cap_add=NET_ADMIN",
				},
			},
		},
		Edges: []*graph.Edge{},
	}

	return NewServer("/test/project", "3.11", callGraph, moduleRegistry, codeGraph, time.Second, false)
}

// ============================================================================
// Test 1: find_dockerfile_instructions - Basic Queries
// ============================================================================

func TestFindDockerfileInstructions_Basic(t *testing.T) {
	server := createSemanticDockerTestServer()

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected int
	}{
		{
			name:     "Find all FROM instructions",
			args:     map[string]interface{}{"instruction_type": "FROM"},
			expected: 3,
		},
		{
			name:     "Find all USER instructions",
			args:     map[string]interface{}{"instruction_type": "USER"},
			expected: 2,
		},
		{
			name:     "Find all EXPOSE instructions",
			args:     map[string]interface{}{"instruction_type": "EXPOSE"},
			expected: 2,
		},
		{
			name:     "Find all COPY instructions",
			args:     map[string]interface{}{"instruction_type": "COPY"},
			expected: 1,
		},
		{
			name:     "Find all WORKDIR instructions",
			args:     map[string]interface{}{"instruction_type": "WORKDIR"},
			expected: 1,
		},
		{
			name:     "Find all HEALTHCHECK instructions",
			args:     map[string]interface{}{"instruction_type": "HEALTHCHECK"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isError := server.toolFindDockerfileInstructions(tt.args)
			require.False(t, isError, "Expected no error")

			var parsed map[string]interface{}
			err := json.Unmarshal([]byte(result), &parsed)
			require.NoError(t, err, "Expected valid JSON")

			matches, ok := parsed["matches"].([]interface{})
			require.True(t, ok, "Expected matches array")
			assert.Equal(t, tt.expected, len(matches), "Match count mismatch")
		})
	}
}

// ============================================================================
// Test 2: find_dockerfile_instructions - Filters
// ============================================================================

func TestFindDockerfileInstructions_Filters(t *testing.T) {
	server := createSemanticDockerTestServer()

	t.Run("Filter by base_image", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"instruction_type": "FROM",
			"base_image":       "python",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Contains(t, match["raw_content"], "python")
	})

	t.Run("Filter by port", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"instruction_type": "EXPOSE",
			"port":             float64(8080),
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, float64(8080), match["port"])
	})

	t.Run("Filter by user", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"instruction_type": "USER",
			"user":             "appuser",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "appuser", match["user"])
	})

	t.Run("Filter by file_path", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"file_path": "Dockerfile.root",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))
	})
}

// ============================================================================
// Test 3: find_dockerfile_instructions - Security Filters
// ============================================================================

func TestFindDockerfileInstructions_SecurityFilters(t *testing.T) {
	server := createSemanticDockerTestServer()

	t.Run("Find unpinned base images", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"instruction_type": "FROM",
			"has_digest":       false,
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 2, len(matches), "Expected 2 unpinned FROM instructions")

		// Verify security issue is flagged
		for _, m := range matches {
			match := m.(map[string]interface{})
			assert.Contains(t, match, "security_issue")
			assert.Equal(t, "No digest pinning (CWE-1188)", match["security_issue"])
			assert.Equal(t, "MEDIUM", match["risk_level"])
		}
	})

	t.Run("Find pinned base images", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"instruction_type": "FROM",
			"has_digest":       true,
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches), "Expected 1 pinned FROM instruction")

		match := matches[0].(map[string]interface{})
		assert.Contains(t, match, "digest")
		assert.NotEmpty(t, match["digest"])
	})

	t.Run("Find root users", func(t *testing.T) {
		result, isError := server.toolFindDockerfileInstructions(map[string]interface{}{
			"instruction_type": "USER",
			"user":             "root",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "root", match["user"])
		assert.Equal(t, "Container runs as root", match["security_issue"])
		assert.Equal(t, "HIGH", match["risk_level"])
	})
}

// ============================================================================
// Test 4: find_compose_services - Basic
// ============================================================================

func TestFindComposeServices_Basic(t *testing.T) {
	server := createSemanticDockerTestServer()

	t.Run("Find all services", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{})
		require.False(t, isError)

		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 5, len(matches), "Expected 5 compose services")
	})

	t.Run("Find service by name", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{
			"service_name": "web",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "web", match["service_name"])
	})

	t.Run("Filter by exposes_port", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{
			"exposes_port": float64(8080),
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "web", match["service_name"])
	})
}

// ============================================================================
// Test 5: find_compose_services - Security Filters
// ============================================================================

func TestFindComposeServices_SecurityFilters(t *testing.T) {
	server := createSemanticDockerTestServer()

	t.Run("Find privileged containers", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{
			"has_privileged": true,
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "privileged-service", match["service_name"])
		assert.Equal(t, true, match["privileged"])

		// Verify security analysis
		securityIssues, ok := match["security_issues"].([]interface{})
		require.True(t, ok)
		assert.Greater(t, len(securityIssues), 0)
		assert.Equal(t, "CRITICAL", match["risk_level"])
	})

	t.Run("Find Docker socket exposure", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{
			"has_volume": "/var/run/docker.sock",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "privileged-service", match["service_name"])
		assert.Equal(t, "CRITICAL", match["risk_level"])
	})

	t.Run("Find host network mode services", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{
			"service_name": "host-network",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Equal(t, "host", match["network_mode"])
		assert.Contains(t, match, "security_issues")
		assert.Equal(t, "HIGH", match["risk_level"])
	})

	t.Run("Find dangerous capabilities", func(t *testing.T) {
		result, isError := server.toolFindComposeServices(map[string]interface{}{
			"service_name": "dangerous-caps",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		matches, ok := parsed["matches"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 1, len(matches))

		match := matches[0].(map[string]interface{})
		assert.Contains(t, match, "security_issues")
		assert.Equal(t, "HIGH", match["risk_level"])
	})
}

// ============================================================================
// Test 6: get_dockerfile_details - Complete
// ============================================================================

func TestGetDockerfileDetails_Complete(t *testing.T) {
	server := createSemanticDockerTestServer()

	result, isError := server.toolGetDockerfileDetails(map[string]interface{}{
		"file_path": "/test/Dockerfile",
	})

	require.False(t, isError)

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	// Verify basic structure
	assert.Equal(t, "/test/Dockerfile", parsed["file"])
	assert.Equal(t, float64(10), parsed["total_instructions"]) // 10 instructions in /test/Dockerfile (user2 is in Dockerfile.root)

	// Verify instructions array
	instructions, ok := parsed["instructions"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, 10, len(instructions))

	// Verify multi-stage detection
	multiStage, ok := parsed["multi_stage"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, multiStage["is_multi_stage"])
	assert.NotEmpty(t, multiStage["base_image"])

	stages, ok := multiStage["stages"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, stages, "builder")

	// Verify security summary
	securitySummary, ok := parsed["security_summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, securitySummary["has_user_instruction"])
	assert.Equal(t, true, securitySummary["has_healthcheck"])
	assert.Equal(t, float64(2), securitySummary["unpinned_images"])
}

// ============================================================================
// Test 7: get_dockerfile_details - Security Summary
// ============================================================================

func TestGetDockerfileDetails_SecuritySummary(t *testing.T) {
	server := createSemanticDockerTestServer()

	t.Run("Dockerfile with unpinned images", func(t *testing.T) {
		result, isError := server.toolGetDockerfileDetails(map[string]interface{}{
			"file_path": "/test/Dockerfile",
		})

		require.False(t, isError)
		var parsed map[string]interface{}
		err := json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)

		securitySummary := parsed["security_summary"].(map[string]interface{})
		assert.Equal(t, float64(2), securitySummary["unpinned_images"])

		issues, ok := securitySummary["issues"].([]interface{})
		require.True(t, ok)
		hasUnpinnedIssue := false
		for _, issue := range issues {
			if issueStr, ok := issue.(string); ok && issueStr == "2 unpinned base image(s)" {
				hasUnpinnedIssue = true
			}
		}
		assert.True(t, hasUnpinnedIssue, "Expected unpinned images issue")
	})

	t.Run("File not found", func(t *testing.T) {
		result, isError := server.toolGetDockerfileDetails(map[string]interface{}{
			"file_path": "/nonexistent/Dockerfile",
		})

		require.True(t, isError)
		assert.Contains(t, result, "error")
	})

	t.Run("Missing file_path parameter", func(t *testing.T) {
		result, isError := server.toolGetDockerfileDetails(map[string]interface{}{})

		require.True(t, isError)
		assert.Contains(t, result, "file_path parameter is required")
	})
}

// ============================================================================
// Test 8: Parsing Functions - FROM Instruction
// ============================================================================

func TestParseFromInstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FromDetails
	}{
		{
			name:  "FROM with tag and digest and stage",
			input: "FROM python:3.11@sha256:abc123 AS builder",
			expected: FromDetails{
				BaseImage:  "python",
				Tag:        "3.11",
				Digest:     "sha256:abc123",
				StageAlias: "builder",
			},
		},
		{
			name:  "FROM with tag only",
			input: "FROM alpine:3.18",
			expected: FromDetails{
				BaseImage: "alpine",
				Tag:       "3.18",
				Digest:    "",
			},
		},
		{
			name:  "FROM without tag (defaults to latest)",
			input: "FROM ubuntu",
			expected: FromDetails{
				BaseImage: "ubuntu",
				Tag:       "latest",
				Digest:    "",
			},
		},
		{
			name:  "FROM with digest but no tag",
			input: "FROM nginx@sha256:def456",
			expected: FromDetails{
				BaseImage: "nginx",
				Tag:       "latest",
				Digest:    "sha256:def456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFromInstruction(tt.input)
			assert.Equal(t, tt.expected.BaseImage, result.BaseImage)
			assert.Equal(t, tt.expected.Tag, result.Tag)
			assert.Equal(t, tt.expected.Digest, result.Digest)
			assert.Equal(t, tt.expected.StageAlias, result.StageAlias)
		})
	}
}

// ============================================================================
// Test 9: Parsing Functions - USER Instruction
// ============================================================================

func TestParseUserInstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected UserDetails
	}{
		{
			name:  "USER with user and group",
			input: "USER appuser:appgroup",
			expected: UserDetails{
				User:  "appuser",
				Group: "appgroup",
			},
		},
		{
			name:  "USER with user only",
			input: "USER root",
			expected: UserDetails{
				User:  "root",
				Group: "",
			},
		},
		{
			name:  "USER with UID and GID",
			input: "USER 1000:1000",
			expected: UserDetails{
				User:  "1000",
				Group: "1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUserInstruction(tt.input)
			assert.Equal(t, tt.expected.User, result.User)
			assert.Equal(t, tt.expected.Group, result.Group)
		})
	}
}

// ============================================================================
// Test 10: Parsing Functions - EXPOSE Instruction
// ============================================================================

func TestParseExposeInstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ExposeDetails
	}{
		{
			name:  "EXPOSE with protocol",
			input: "EXPOSE 8080/tcp",
			expected: ExposeDetails{
				Port:     8080,
				Protocol: "tcp",
			},
		},
		{
			name:  "EXPOSE without protocol (defaults to tcp)",
			input: "EXPOSE 443",
			expected: ExposeDetails{
				Port:     443,
				Protocol: "tcp",
			},
		},
		{
			name:  "EXPOSE with udp protocol",
			input: "EXPOSE 53/udp",
			expected: ExposeDetails{
				Port:     53,
				Protocol: "udp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseExposeInstruction(tt.input)
			assert.Equal(t, tt.expected.Port, result.Port)
			assert.Equal(t, tt.expected.Protocol, result.Protocol)
		})
	}
}

// ============================================================================
// Test 11: Parsing Functions - COPY Instruction
// ============================================================================

func TestParseCopyInstruction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected CopyDetails
	}{
		{
			name:  "COPY with all flags",
			input: "COPY --from=builder --chown=appuser:appgroup /build/app /app",
			expected: CopyDetails{
				Source:      "/build/app",
				Destination: "/app",
				FromStage:   "builder",
				Chown:       "appuser:appgroup",
			},
		},
		{
			name:  "COPY without flags",
			input: "COPY src/ /app/",
			expected: CopyDetails{
				Source:      "src/",
				Destination: "/app/",
				FromStage:   "",
				Chown:       "",
			},
		},
		{
			name:  "COPY with from flag only",
			input: "COPY --from=builder /dist /app",
			expected: CopyDetails{
				Source:      "/dist",
				Destination: "/app",
				FromStage:   "builder",
				Chown:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCopyInstruction(tt.input)
			assert.Equal(t, tt.expected.Source, result.Source)
			assert.Equal(t, tt.expected.Destination, result.Destination)
			assert.Equal(t, tt.expected.FromStage, result.FromStage)
			assert.Equal(t, tt.expected.Chown, result.Chown)
		})
	}
}
