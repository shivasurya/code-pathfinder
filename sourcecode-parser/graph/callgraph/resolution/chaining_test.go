package resolution

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
)

func TestParseChain(t *testing.T) {
	tests := []struct {
		name          string
		target        string
		expectedSteps int
		expectedNames []string
	}{
		{
			name:          "simple two-step chain",
			target:        "create_builder().append()",
			expectedSteps: 2,
			expectedNames: []string{"create_builder", "append"},
		},
		{
			name:          "three-step chain",
			target:        "create_builder().append().upper()",
			expectedSteps: 3,
			expectedNames: []string{"create_builder", "append", "upper"},
		},
		{
			name:          "four-step chain",
			target:        "create_builder().append().upper().build()",
			expectedSteps: 4,
			expectedNames: []string{"create_builder", "append", "upper", "build"},
		},
		{
			name:          "chain with arguments",
			target:        `create_builder().append("hello ").append("world").upper().build()`,
			expectedSteps: 5,
			// Note: method names are extracted without arguments for calls
			expectedNames: []string{"create_builder", "append", "append", "upper", "build"},
		},
		{
			name:          "builtin chain",
			target:        "text.strip().upper().split()",
			expectedSteps: 3,
			expectedNames: []string{"strip", "upper", "split"},
		},
		{
			name:          "not a chain - simple call",
			target:        "function()",
			expectedSteps: 0,
			expectedNames: nil,
		},
		{
			name:          "not a chain - attribute access",
			target:        "obj.method()",
			expectedSteps: 0,
			expectedNames: nil,
		},
		{
			name:          "not a chain - nested attribute",
			target:        "obj.attr.method()",
			expectedSteps: 0,
			expectedNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := ParseChain(tt.target)

			if tt.expectedSteps == 0 {
				assert.Nil(t, steps, "Expected no chain for: %s", tt.target)
				return
			}

			assert.NotNil(t, steps, "Expected chain for: %s", tt.target)
			assert.Equal(t, tt.expectedSteps, len(steps), "Wrong number of steps for: %s", tt.target)

			if tt.expectedNames != nil {
				for i, expectedName := range tt.expectedNames {
					if i < len(steps) {
						assert.Equal(t, expectedName, steps[i].MethodName,
							"Step %d: expected name %s, got %s", i, expectedName, steps[i].MethodName)
					}
				}
			}
		})
	}
}

func TestParseStep(t *testing.T) {
	tests := []struct {
		name           string
		expr           string
		expectedName   string
		expectedIsCall bool
	}{
		{
			name:           "simple call",
			expr:           "function()",
			expectedName:   "function",
			expectedIsCall: true,
		},
		{
			name:           "method call",
			expr:           "obj.method()",
			expectedName:   "method",
			expectedIsCall: true,
		},
		{
			name:           "nested method call",
			expr:           "obj.attr.method()",
			expectedName:   "method",
			expectedIsCall: true,
		},
		{
			name:           "variable access",
			expr:           "variable",
			expectedName:   "variable",
			expectedIsCall: false,
		},
		{
			name:           "attribute access",
			expr:           "obj.attr",
			expectedName:   "obj.attr",
			expectedIsCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := parseStep(tt.expr)

			assert.NotNil(t, step)
			assert.Equal(t, tt.expectedName, step.MethodName)
			assert.Equal(t, tt.expectedIsCall, step.IsCall)
			assert.Equal(t, tt.expr, step.Expression)
		})
	}
}

func TestResolveChainedCall(t *testing.T) {
	// Setup test environment
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	builtins := registry.NewBuiltinRegistry()
	codeGraph := &graph.CodeGraph{}
	callGraph := core.NewCallGraph()

	// Add a function with known return type
	typeEngine.ReturnTypes["myapp.create_builder"] = &core.TypeInfo{
		TypeFQN:    "builtins.str",
		Confidence: 1.0,
		Source:     "return_type",
	}

	tests := []struct {
		name             string
		target           string
		callerFQN        string
		currentModule    string
		expectedResolved bool
		expectedType     string
	}{
		{
			name:             "simple two-step chain with builtin",
			target:           "create_builder().upper()",
			callerFQN:        "myapp.test",
			currentModule:    "myapp",
			expectedResolved: true,
			expectedType:     "builtins.str",
		},
		{
			name:             "not a chain - single call",
			target:           "function()",
			callerFQN:        "myapp.test",
			currentModule:    "myapp",
			expectedResolved: false,
			expectedType:     "",
		},
		{
			name:             "not a chain - simple attribute",
			target:           "obj.method()",
			callerFQN:        "myapp.test",
			currentModule:    "myapp",
			expectedResolved: false,
			expectedType:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetFQN, resolved, typeInfo := ResolveChainedCall(
				tt.target,
				typeEngine,
				builtins,
				moduleRegistry,
				codeGraph,
				tt.callerFQN,
				tt.currentModule,
				callGraph,
			)

			assert.Equal(t, tt.expectedResolved, resolved, "Resolution status mismatch")

			if tt.expectedResolved {
				assert.NotEmpty(t, targetFQN, "Should have resolved FQN")
				assert.NotNil(t, typeInfo, "Should have type info")
				assert.Equal(t, tt.expectedType, typeInfo.TypeFQN)
				assert.Equal(t, "method_chain", typeInfo.Source)
			}
		})
	}
}

func TestResolveFirstChainStep(t *testing.T) {
	// Setup
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	callGraph := core.NewCallGraph()

	// Add a function with return type
	typeEngine.ReturnTypes["myapp.create_builder"] = &core.TypeInfo{
		TypeFQN:    "myapp.Builder",
		Confidence: 1.0,
		Source:     "return_type",
	}

	// Add function to call graph
	builderNode := &graph.Node{
		ID:   "myapp.create_builder",
		Type: "function_declaration",
		Name: "create_builder",
	}
	callGraph.Functions["myapp.create_builder"] = builderNode

	tests := []struct {
		name          string
		step          ChainStep
		callerFQN     string
		currentModule string
		expectedOK    bool
		expectedType  string
	}{
		{
			name: "function call with return type",
			step: ChainStep{
				MethodName: "create_builder",
				Expression: "create_builder()",
				IsCall:     true,
			},
			callerFQN:     "myapp.test",
			currentModule: "myapp",
			expectedOK:    true,
			expectedType:  "myapp.Builder",
		},
		{
			name: "unknown function call",
			step: ChainStep{
				MethodName: "unknown_function",
				Expression: "unknown_function()",
				IsCall:     true,
			},
			callerFQN:     "myapp.test",
			currentModule: "myapp",
			expectedOK:    false,
			expectedType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo, fqn, ok := resolveFirstChainStep(
				tt.step,
				typeEngine,
				tt.callerFQN,
				tt.currentModule,
				moduleRegistry,
				callGraph,
			)

			assert.Equal(t, tt.expectedOK, ok, "Resolution OK status mismatch")

			if tt.expectedOK {
				assert.NotNil(t, typeInfo, "Should have type info")
				assert.Equal(t, tt.expectedType, typeInfo.TypeFQN)
				assert.NotEmpty(t, fqn, "Should have FQN")
			}
		})
	}
}

func TestResolveChainMethod(t *testing.T) {
	// Setup
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Attributes = registry.NewAttributeRegistry()
	builtins := registry.NewBuiltinRegistry()
	callGraph := core.NewCallGraph()

	tests := []struct {
		name         string
		step         ChainStep
		currentType  *core.TypeInfo
		expectedOK   bool
		expectedType string
	}{
		{
			name: "builtin method on string",
			step: ChainStep{
				MethodName: "upper",
				Expression: "upper()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true,
			expectedType: "builtins.str",
		},
		{
			name: "builtin method on list",
			step: ChainStep{
				MethodName: "append",
				Expression: "append()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "builtins.list",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true,
			expectedType: "builtins.NoneType",
		},
		{
			name: "unknown method on builtin treated as call",
			step: ChainStep{
				MethodName: "nonexistent_method_xyz",
				Expression: "nonexistent_method_xyz()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true, // Changed - unknown methods are still resolved
			expectedType: "builtins.str", // Returns same type when method not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo, fqn, ok := resolveChainMethod(
				tt.step,
				tt.currentType,
				builtins,
				typeEngine,
				moduleRegistry,
				callGraph,
			)

			assert.Equal(t, tt.expectedOK, ok, "Resolution OK status mismatch")

			if tt.expectedOK {
				assert.NotNil(t, typeInfo, "Should have type info")
				assert.Equal(t, tt.expectedType, typeInfo.TypeFQN)
				assert.NotEmpty(t, fqn, "Should have FQN")
			}
		})
	}
}
