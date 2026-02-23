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
				t.Helper()
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
				t.Helper()
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
		Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
				Matcher: map[string]any{
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
					Matcher: map[string]any{
						"type": "all_of",
						"conditions": []any{
							map[string]any{
								"type":        "instruction",
								"instruction": "FROM",
								"image_tag":   "latest",
							},
							map[string]any{
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
					Matcher: map[string]any{
						"type": "any_of",
						"conditions": []any{
							map[string]any{
								"type":        "instruction",
								"instruction": "FROM",
								"image_tag":   "latest",
							},
							map[string]any{
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
					Matcher: map[string]any{
						"type": "none_of",
						"conditions": []any{
							map[string]any{
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
					Matcher: map[string]any{
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
					Matcher: map[string]any{
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
							Value:    []any{"seccomp:unconfined"},
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
					Matcher: map[string]any{
						"type": "service_has",
						"key":  "volumes",
						"contains_any": []any{
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
							Value:    []any{"/var/run/docker.sock:/var/run/docker.sock"},
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
					Matcher: map[string]any{
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
					Matcher: map[string]any{
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
					Matcher: map[string]any{
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
					Matcher: map[string]any{
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
					Matcher: map[string]any{},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("missing_instruction with wrong type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-WRONG-TYPE",
					Matcher: map[string]any{
						"type":        "missing_instruction",
						"instruction": 123, // Wrong type - should be string
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("instruction with wrong type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-WRONG-INST",
					Matcher: map[string]any{
						"type":        "instruction",
						"instruction": 456, // Wrong type
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("instruction with no matching nodes", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-NO-MATCH",
					Matcher: map[string]any{
						"type":        "instruction",
						"instruction": "HEALTHCHECK",
					},
				},
			},
		}

		// Dockerfile without HEALTHCHECK
		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			BaseImage:       "ubuntu",
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("invalid regex in arg_name_regex", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-INVALID-REGEX",
					Matcher: map[string]any{
						"type":           "instruction",
						"instruction":    "ARG",
						"arg_name_regex": "[invalid(regex",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "ARG",
			ArgName:         "TEST_ARG",
			LineNumber:      5,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches) // Invalid regex should not match
	})

	t.Run("criteria mismatch - image_tag", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-TAG-MISMATCH",
					Matcher: map[string]any{
						"type":        "instruction",
						"instruction": "FROM",
						"image_tag":   "latest",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			ImageTag:        "22.04", // Different tag
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("criteria mismatch - user_name", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-USER-MISMATCH",
					Matcher: map[string]any{
						"type":        "instruction",
						"instruction": "USER",
						"user_name":   "root",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "USER",
			UserName:        "appuser", // Different user
			LineNumber:      10,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("port_less_than with no matching ports", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-PORT-NO-MATCH",
					Matcher: map[string]any{
						"type":           "instruction",
						"instruction":    "EXPOSE",
						"port_less_than": float64(1024),
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "EXPOSE",
			Ports:           []int{8080, 9000}, // All ports >= 1024
			LineNumber:      15,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("missing_digest false with digest present", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-DIGEST-PRESENT",
					Matcher: map[string]any{
						"type":           "instruction",
						"instruction":    "FROM",
						"missing_digest": false, // Requires digest
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			ImageDigest:     "sha256:abc123", // Has digest
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		require.Len(t, matches, 1)
	})

	t.Run("missing_digest false without digest", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-DIGEST-MISSING",
					Matcher: map[string]any{
						"type":           "instruction",
						"instruction":    "FROM",
						"missing_digest": false,
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			ImageDigest:     "", // No digest
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("base_image mismatch", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-BASE-MISMATCH",
					Matcher: map[string]any{
						"type":        "instruction",
						"instruction": "FROM",
						"base_image":  "alpine",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			BaseImage:       "ubuntu", // Different image
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("compose - service_has wrong key type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-WRONG-KEY",
					Matcher: map[string]any{
						"type": "service_has",
						"key":  123, // Wrong type
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {Type: "mapping", Children: map[string]*graph.YAMLNode{}},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("compose - service_missing wrong key type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-MISSING-WRONG",
					Matcher: map[string]any{
						"type": "service_missing",
						"key":  456,
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {Type: "mapping", Children: map[string]*graph.YAMLNode{}},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("compose - invalid matcher type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-INVALID",
					Matcher: map[string]any{
						"type": "unknown_compose_type",
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {Type: "mapping", Children: map[string]*graph.YAMLNode{}},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("compose - service_has no equals match", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-NO-MATCH",
					Matcher: map[string]any{
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
					Type: "mapping",
					Children: map[string]*graph.YAMLNode{
						"privileged": {Type: "scalar", Value: false}, // Not true
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("compose - service_has contains_any no match", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-CONTAINS-NO-MATCH",
					Matcher: map[string]any{
						"type": "service_has",
						"key":  "volumes",
						"contains_any": []any{
							"/var/run/docker.sock",
						},
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {
					Type: "mapping",
					Children: map[string]*graph.YAMLNode{
						"volumes": {
							Type:  "sequence",
							Value: []any{"/data:/data"}, // Different volume
						},
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("compose - service_has contains_any with non-string", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-NON-STRING",
					Matcher: map[string]any{
						"type": "service_has",
						"key":  "volumes",
						"contains_any": []any{
							123, // Non-string value
						},
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {
					Type: "mapping",
					Children: map[string]*graph.YAMLNode{
						"volumes": {Type: "sequence", Value: []any{"volume1"}},
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("compose - service_missing with service having key", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-HAS-KEY",
					Matcher: map[string]any{
						"type": "service_missing",
						"key":  "read_only",
					},
				},
			},
		}

		compose := &graph.ComposeGraph{
			Services: map[string]*graph.YAMLNode{
				"web": {
					Type: "mapping",
					Children: map[string]*graph.YAMLNode{
						"read_only": {Type: "scalar", Value: true}, // Key exists
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		assert.Empty(t, matches)
	})

	t.Run("all_of with malformed condition", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ALLOF-MALFORMED",
					Matcher: map[string]any{
						"type": "all_of",
						"conditions": []any{
							"not_a_map", // Invalid condition type
						},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("all_of with one condition failing", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ALLOF-FAIL",
					Matcher: map[string]any{
						"type": "all_of",
						"conditions": []any{
							map[string]any{
								"type":        "instruction",
								"instruction": "FROM",
								"image_tag":   "latest",
							},
							map[string]any{
								"type":        "instruction",
								"instruction": "HEALTHCHECK", // This won't exist
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
		assert.Empty(t, matches) // all_of should fail because HEALTHCHECK missing
	})

	t.Run("all_of with wrong conditions type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ALLOF-WRONG-TYPE",
					Matcher: map[string]any{
						"type":       "all_of",
						"conditions": "not_an_array",
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("any_of with all conditions failing", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ANYOF-ALL-FAIL",
					Matcher: map[string]any{
						"type": "any_of",
						"conditions": []any{
							map[string]any{
								"type":        "instruction",
								"instruction": "HEALTHCHECK",
							},
							map[string]any{
								"type":        "instruction",
								"instruction": "STOPSIGNAL",
							},
						},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches) // any_of should fail because no conditions match
	})

	t.Run("any_of with malformed condition", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ANYOF-MALFORMED",
					Matcher: map[string]any{
						"type": "any_of",
						"conditions": []any{
							123, // Invalid type
						},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("any_of with wrong conditions type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-ANYOF-WRONG-TYPE",
					Matcher: map[string]any{
						"type":       "any_of",
						"conditions": 789,
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("none_of with no conditions matching", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-NONEOF-NO-MATCH",
					Matcher: map[string]any{
						"type": "none_of",
						"conditions": []any{
							map[string]any{
								"type":        "instruction",
								"instruction": "HEALTHCHECK",
							},
						},
					},
				},
			},
		}

		// Dockerfile without HEALTHCHECK
		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "FROM",
			LineNumber:      1,
		})

		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches) // none_of should not trigger because no conditions matched
	})

	t.Run("none_of with malformed condition", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-NONEOF-MALFORMED",
					Matcher: map[string]any{
						"type": "none_of",
						"conditions": []any{
							false, // Invalid type
						},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("none_of with wrong conditions type", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			dockerfileRules: []CompiledRule{
				{
					ID: "TEST-NONEOF-WRONG-TYPE",
					Matcher: map[string]any{
						"type":       "none_of",
						"conditions": map[string]any{},
					},
				},
			},
		}

		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		matches := executor.ExecuteDockerfile(dockerfile)
		assert.Empty(t, matches)
	})

	t.Run("multiple services in compose", func(t *testing.T) {
		executor := &ContainerRuleExecutor{
			composeRules: []CompiledRule{
				{
					ID: "COMPOSE-MULTI",
					Matcher: map[string]any{
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
					Type:     "mapping",
					Children: map[string]*graph.YAMLNode{},
				},
				"db": {
					Type: "mapping",
					Children: map[string]*graph.YAMLNode{
						"privileged": {Type: "scalar", Value: true},
					},
				},
			},
			FilePath: "docker-compose.yml",
		}

		matches := executor.ExecuteCompose(compose)
		require.Len(t, matches, 1)
		assert.Equal(t, "db", matches[0].ServiceName)
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

func TestContainerRuleExecutor_MultiplePortViolations(t *testing.T) {
	// Test that multiple invalid ports are reported as separate findings
	executor := &ContainerRuleExecutor{}

	// Create rule with any_of combinator to catch both port < 1 and port > 65535
	rule := CompiledRule{
		ID:       "DOCKER-COR-002",
		Name:     "Invalid Port Number",
		Severity: "HIGH",
		CWE:      "CWE-20",
		Message:  "Invalid port number",
		Matcher: map[string]any{
			"type": "any_of",
			"conditions": []any{
				map[string]any{
					"type":           "instruction",
					"instruction":    "EXPOSE",
					"port_less_than": float64(1),
				},
				map[string]any{
					"type":              "instruction",
					"instruction":       "EXPOSE",
					"port_greater_than": float64(65535),
				},
			},
		},
	}

	// Create Dockerfile with multiple invalid ports
	dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
	dockerfile.AddInstruction(&docker.DockerfileNode{
		InstructionType: "EXPOSE",
		Ports:           []int{0},
		LineNumber:      2,
	})
	dockerfile.AddInstruction(&docker.DockerfileNode{
		InstructionType: "EXPOSE",
		Ports:           []int{70000},
		LineNumber:      3,
	})

	executor.dockerfileRules = []CompiledRule{rule}
	matches := executor.ExecuteDockerfile(dockerfile)

	// Should report both violations
	require.Len(t, matches, 2, "Expected 2 violations for 2 invalid ports")
	assert.Equal(t, 2, matches[0].LineNumber, "First match should be line 2")
	assert.Equal(t, 3, matches[1].LineNumber, "Second match should be line 3")
	assert.Equal(t, "DOCKER-COR-002", matches[0].RuleID)
	assert.Equal(t, "DOCKER-COR-002", matches[1].RuleID)
}

func TestContainerRuleExecutor_PortGreaterThan(t *testing.T) {
	// Test that port_greater_than matcher works correctly
	executor := &ContainerRuleExecutor{}

	rule := CompiledRule{
		ID:   "TEST-PORT-GT",
		Name: "Port too high",
		Matcher: map[string]any{
			"type":              "instruction",
			"instruction":       "EXPOSE",
			"port_greater_than": float64(65535),
		},
	}

	t.Run("matches port greater than 65535", func(t *testing.T) {
		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "EXPOSE",
			Ports:           []int{70000},
			LineNumber:      5,
		})

		executor.dockerfileRules = []CompiledRule{rule}
		matches := executor.ExecuteDockerfile(dockerfile)

		require.Len(t, matches, 1)
		assert.Equal(t, 5, matches[0].LineNumber)
	})

	t.Run("does not match port equal to 65535", func(t *testing.T) {
		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "EXPOSE",
			Ports:           []int{65535},
			LineNumber:      5,
		})

		executor.dockerfileRules = []CompiledRule{rule}
		matches := executor.ExecuteDockerfile(dockerfile)

		require.Len(t, matches, 0)
	})

	t.Run("does not match valid port", func(t *testing.T) {
		dockerfile := docker.NewDockerfileGraph("test.Dockerfile")
		dockerfile.AddInstruction(&docker.DockerfileNode{
			InstructionType: "EXPOSE",
			Ports:           []int{8080},
			LineNumber:      5,
		})

		executor.dockerfileRules = []CompiledRule{rule}
		matches := executor.ExecuteDockerfile(dockerfile)

		require.Len(t, matches, 0)
	})
}
