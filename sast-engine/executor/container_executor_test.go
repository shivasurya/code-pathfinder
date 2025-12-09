package executor

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerRuleExecutor_LoadRules(t *testing.T) {
	tests := []struct {
		name    string
		jsonIR  string
		wantErr bool
		checkFn func(*testing.T, *ContainerRuleExecutor)
	}{
		{
			name: "valid rules",
			jsonIR: `{
				"dockerfile": [
					{
						"id": "TEST-001",
						"name": "Test Rule",
						"severity": "HIGH",
						"category": "security",
						"cwe": "CWE-250",
						"message": "Test message",
						"file_pattern": "Dockerfile*",
						"rule_type": "dockerfile",
						"matcher": {"type": "missing_instruction", "instruction": "USER"}
					}
				],
				"compose": [
					{
						"id": "COMPOSE-001",
						"name": "Compose Test",
						"severity": "HIGH",
						"category": "security",
						"cwe": "CWE-250",
						"message": "Test",
						"file_pattern": "*.yml",
						"rule_type": "compose",
						"matcher": {"type": "service_has", "key": "privileged", "equals": true}
					}
				]
			}`,
			wantErr: false,
			checkFn: func(t *testing.T, e *ContainerRuleExecutor) {
				assert.Len(t, e.dockerfileRules, 1)
				assert.Len(t, e.composeRules, 1)
				assert.Equal(t, "TEST-001", e.dockerfileRules[0].ID)
				assert.Equal(t, "COMPOSE-001", e.composeRules[0].ID)
			},
		},
		{
			name:    "invalid json",
			jsonIR:  `{invalid}`,
			wantErr: true,
		},
		{
			name: "empty rules",
			jsonIR: `{
				"dockerfile": [],
				"compose": []
			}`,
			wantErr: false,
			checkFn: func(t *testing.T, e *ContainerRuleExecutor) {
				assert.Len(t, e.dockerfileRules, 0)
				assert.Len(t, e.composeRules, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ContainerRuleExecutor{}
			err := executor.LoadRules([]byte(tt.jsonIR))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFn != nil {
					tt.checkFn(t, executor)
				}
			}
		})
	}
}

func TestContainerRuleExecutor_ExecuteDockerfile_MissingInstruction(t *testing.T) {
	executor := &ContainerRuleExecutor{}
	rule := CompiledRule{
		ID:       "TEST-001",
		Name:     "Missing USER",
		Severity: "HIGH",
		CWE:      "CWE-250",
		Message:  "No USER instruction",
		Matcher: map[string]interface{}{
			"type":        "missing_instruction",
			"instruction": "USER",
		},
	}
	executor.dockerfileRules = []CompiledRule{rule}

	// Dockerfile without USER
	dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
	from := &docker.DockerfileNode{
		InstructionType: "FROM",
		BaseImage:       "ubuntu",
		ImageTag:        "20.04",
		LineNumber:      1,
	}
	dockerfile.AddInstruction(from)

	matches := executor.ExecuteDockerfile(dockerfile)

	require.Len(t, matches, 1)
	assert.Equal(t, "TEST-001", matches[0].RuleID)
	assert.Equal(t, "Missing USER", matches[0].RuleName)
	assert.Equal(t, "HIGH", matches[0].Severity)
	assert.Equal(t, "test.Dockerfile", matches[0].FilePath)
}

func TestContainerRuleExecutor_ExecuteDockerfile_Instruction(t *testing.T) {
	tests := []struct {
		name       string
		rule       CompiledRule
		dockerfile *docker.DockerfileGraph
		wantMatch  bool
	}{
		{
			name: "image tag latest",
			rule: CompiledRule{
				ID:       "TEST-002",
				Name:     "Using latest tag",
				Severity: "MEDIUM",
				Matcher: map[string]interface{}{
					"type":        "instruction",
					"instruction": "FROM",
					"image_tag":   "latest",
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "FROM",
					BaseImage:       "ubuntu",
					ImageTag:        "latest",
					LineNumber:      1,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "user name root",
			rule: CompiledRule{
				ID:   "TEST-003",
				Name: "Running as root",
				Matcher: map[string]interface{}{
					"type":        "instruction",
					"instruction": "USER",
					"user_name":   "root",
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "USER",
					UserName:        "root",
					LineNumber:      5,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "arg name regex - secret",
			rule: CompiledRule{
				ID:   "TEST-004",
				Name: "Secret in ARG",
				Matcher: map[string]interface{}{
					"type":           "instruction",
					"instruction":    "ARG",
					"arg_name_regex": "(?i).*password.*",
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "ARG",
					ArgName:         "DB_PASSWORD",
					LineNumber:      3,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "contains substring",
			rule: CompiledRule{
				ID: "TEST-005",
				Matcher: map[string]interface{}{
					"type":        "instruction",
					"instruction": "RUN",
					"contains":    "apt-get install",
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "RUN",
					RawInstruction:  "RUN apt-get update && apt-get install -y curl",
					LineNumber:      10,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "not contains",
			rule: CompiledRule{
				ID: "TEST-006",
				Matcher: map[string]interface{}{
					"type":         "instruction",
					"instruction":  "RUN",
					"contains":     "apt-get install",
					"not_contains": "--no-install-recommends",
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "RUN",
					RawInstruction:  "RUN apt-get install -y curl",
					LineNumber:      10,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "port less than 1024",
			rule: CompiledRule{
				ID: "TEST-007",
				Matcher: map[string]interface{}{
					"type":           "instruction",
					"instruction":    "EXPOSE",
					"port_less_than": float64(1024),
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "EXPOSE",
					Ports:           []int{22, 80, 443},
					LineNumber:      15,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "missing digest",
			rule: CompiledRule{
				ID: "TEST-008",
				Matcher: map[string]interface{}{
					"type":           "instruction",
					"instruction":    "FROM",
					"missing_digest": true,
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "FROM",
					BaseImage:       "ubuntu",
					ImageTag:        "20.04",
					ImageDigest:     "", // No digest
					LineNumber:      1,
				})
				return g
			}(),
			wantMatch: true,
		},
		{
			name: "base image match",
			rule: CompiledRule{
				ID: "TEST-009",
				Matcher: map[string]interface{}{
					"type":        "instruction",
					"instruction": "FROM",
					"base_image":  "ubuntu",
				},
			},
			dockerfile: func() *docker.DockerfileGraph {
				g := docker.NewDockerfileGraph("test.Dockerfile")
				g.AddInstruction(&docker.DockerfileNode{
					InstructionType: "FROM",
					BaseImage:       "ubuntu",
					ImageTag:        "22.04",
					LineNumber:      1,
				})
				return g
			}(),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ContainerRuleExecutor{
				dockerfileRules: []CompiledRule{tt.rule},
			}

			matches := executor.ExecuteDockerfile(tt.dockerfile)

			if tt.wantMatch {
				require.Len(t, matches, 1, "Expected match but got none")
				assert.Equal(t, tt.rule.ID, matches[0].RuleID)
			} else {
				assert.Empty(t, matches, "Expected no match but got one")
			}
		})
	}
}

func TestContainerRuleExecutor_Combinators(t *testing.T) {
	t.Run("all_of - all conditions match", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID:   "TEST-ALL-01",
					Name: "All conditions",
					Matcher: map[string]interface{}{
						"type": "all_of",
						"conditions": []interface{}{
							map[string]interface{}{
								"type":        "instruction",
								"instruction": "FROM",
								"image_tag":   "latest",
							},
							map[string]interface{}{
								"type":        "missing_instruction",
								"instruction": "USER",
							},
						},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			ImageTag:        "latest",
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		require.Len(t, matches, 1)
	})

	t.Run("any_of - one condition matches", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ANY-01",
					Matcher: map[string]interface{}{
						"type": "any_of",
						"conditions": []interface{}{
							map[string]interface{}{
								"type":        "instruction",
								"instruction": "FROM",
								"image_tag":   "latest",
							},
							map[string]interface{}{
								"type":        "instruction",
								"instruction": "FROM",
								"base_image":  "scratch",
							},
						},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			ImageTag:        "latest",
			BaseImage:       "ubuntu",
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		require.Len(t, matches, 1)
	})

	t.Run("none_of - condition matches triggers rule", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-NONE-01",
					Matcher: map[string]interface{}{
						"type": "none_of",
						"conditions": []interface{}{
							map[string]interface{}{
								"type":        "instruction",
								"instruction": "HEALTHCHECK",
							},
						},
					},
				},
			},
		}

		// Dockerfile with HEALTHCHECK - should trigger none_of
		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "HEALTHCHECK",
			LineNumber:      5,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		require.Len(t, matches, 1)
	})
}

func TestContainerRuleExecutor_ExecuteCompose(t *testing.T) {
	t.Run("service_has with equals", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID:       "COMPOSE-001",
					Name:     "Privileged service",
					Severity: "CRITICAL",
					Matcher: map[string]interface{}{
						"type":   "service_has",
						"key":    "privileged",
						"equals": true,
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {
					Type:  "mapping",
					Value: nil,
					Children: map[string]*graph.YAMLNode{
						"privileged": {
							Type:     "scalar",
							Value:    true,
							Children: nil,
						},
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		require.Len(t, matches, 1)
		assert.Equal(t, "COMPOSE-001", matches[0].RuleID)
		assert.Equal(t, "web", matches[0].ServiceName)
	})

	t.Run("service_has with contains", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-002",
					Matcher: map[string]interface{}{
						"type":     "service_has",
						"key":      "security_opt",
						"contains": "seccomp:unconfined",
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"app": {
					Type:  "mapping",
					Value: nil,
					Children: map[string]*graph.YAMLNode{
						"security_opt": {
							Type:     "sequence",
							Value:    []interface{}{"seccomp:unconfined"},
							Children: nil,
						},
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		require.Len(t, matches, 1)
	})

	t.Run("service_has with contains_any", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-003",
					Matcher: map[string]interface{}{
						"type": "service_has",
						"key":  "volumes",
						"contains_any": []interface{}{
							"/var/run/docker.sock",
							"/run/docker.sock",
						},
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {
					Type:  "mapping",
					Value: nil,
					Children: map[string]*graph.YAMLNode{
						"volumes": {
							Type:     "sequence",
							Value:    []interface{}{"/var/run/docker.sock:/var/run/docker.sock"},
							Children: nil,
						},
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		require.Len(t, matches, 1)
	})

	t.Run("service_missing", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-004",
					Matcher: map[string]interface{}{
						"type": "service_missing",
						"key":  "read_only",
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {
					Type:     "mapping",
					Value:    nil,
					Children: map[string]*graph.YAMLNode{},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		require.Len(t, matches, 1)
		assert.Equal(t, "COMPOSE-004", matches[0].RuleID)
	})
}

func TestContainerRuleExecutor_EdgeCases(t *testing.T) {
	t.Run("empty dockerfile", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-001",
					Matcher: map[string]interface{}{
						"type":        "missing_instruction",
						"instruction": "USER",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		require.Len(t, matches, 1) // Missing USER should match
	})

	t.Run("empty compose", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-001",
					Matcher: map[string]interface{}{
						"type":   "service_has",
						"key":    "privileged",
						"equals": true,
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("invalid matcher type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-INVALID",
					Matcher: map[string]interface{}{
						"type": "unknown_type",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("malformed matcher - no type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID:      "TEST-NO-TYPE",
					Matcher: map[string]interface{}{},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})
}

func TestRuleMatch_JSONSerialization(t *testing.T) {
	match := RuleMatch{
		RuleID:      "TEST-001",
		RuleName:    "Test Rule",
		Severity:    "HIGH",
		CWE:         "CWE-250",
		Message:     "Test message",
		FilePath:    "test.Dockerfile",
		LineNumber:  10,
		ServiceName: "web",
	}

	data, err := json.Marshal(match)
	require.NoError(t, err)

	var decoded RuleMatch
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, match.RuleID, decoded.RuleID)
	assert.Equal(t, match.ServiceName, decoded.ServiceName)
}
