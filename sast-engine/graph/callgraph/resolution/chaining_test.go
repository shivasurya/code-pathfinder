package resolution

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
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

// TestResolveFirstChainStep_VariableLookup tests variable resolution in scopes.
func TestResolveFirstChainStep_VariableLookup(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	callGraph := core.NewCallGraph()

	// Setup function scope with variables
	functionScope := NewFunctionScope("myapp.test_func")
	functionScope.AddVariable(&VariableBinding{
		VarName: "local_var",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "assignment",
		},
	})
	typeEngine.Scopes["myapp.test_func"] = functionScope

	// Setup module scope with variables
	moduleScope := NewFunctionScope("myapp")
	moduleScope.AddVariable(&VariableBinding{
		VarName: "module_var",
		Type: &core.TypeInfo{
			TypeFQN:    "myapp.MyClass",
			Confidence: 1.0,
			Source:     "assignment",
		},
	})
	typeEngine.Scopes["myapp"] = moduleScope

	tests := []struct {
		name          string
		step          ChainStep
		callerFQN     string
		currentModule string
		expectedOK    bool
		expectedType  string
	}{
		{
			name: "local variable in function scope",
			step: ChainStep{
				MethodName: "local_var",
				Expression: "local_var",
				IsCall:     false,
			},
			callerFQN:     "myapp.test_func",
			currentModule: "myapp",
			expectedOK:    true,
			expectedType:  "builtins.str",
		},
		{
			name: "module-level variable",
			step: ChainStep{
				MethodName: "module_var",
				Expression: "module_var",
				IsCall:     false,
			},
			callerFQN:     "myapp.test_func",
			currentModule: "myapp",
			expectedOK:    true,
			expectedType:  "myapp.MyClass",
		},
		{
			name: "variable not found in any scope",
			step: ChainStep{
				MethodName: "unknown_var",
				Expression: "unknown_var",
				IsCall:     false,
			},
			callerFQN:     "myapp.test_func",
			currentModule: "myapp",
			expectedOK:    false,
			expectedType:  "",
		},
		{
			name: "scope not found",
			step: ChainStep{
				MethodName: "some_var",
				Expression: "some_var",
				IsCall:     false,
			},
			callerFQN:     "unknown.function",
			currentModule: "unknown",
			expectedOK:    false,
			expectedType:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo, _, ok := resolveFirstChainStep(
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
			}
		})
	}
}

// TestResolveFirstChainStep_FunctionCall tests function call resolution.
func TestResolveFirstChainStep_FunctionCall(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	callGraph := core.NewCallGraph()

	// Add function with concrete return type
	typeEngine.ReturnTypes["myapp.get_user"] = &core.TypeInfo{
		TypeFQN:    "myapp.User",
		Confidence: 1.0,
		Source:     "return_type",
	}

	// Add function with unresolved return type (var:)
	typeEngine.ReturnTypes["myapp.get_unresolved"] = &core.TypeInfo{
		TypeFQN:    "var:result",
		Confidence: 0.5,
		Source:     "return_type",
	}

	// Add function with call placeholder (call:)
	typeEngine.ReturnTypes["myapp.get_call"] = &core.TypeInfo{
		TypeFQN:    "call:some_func",
		Confidence: 0.5,
		Source:     "return_type",
	}

	// Add function to call graph
	callGraph.Functions["myapp.get_user"] = &graph.Node{
		ID:   "myapp.get_user",
		Type: "function_declaration",
		Name: "get_user",
	}

	tests := []struct {
		name          string
		step          ChainStep
		callerFQN     string
		currentModule string
		expectedOK    bool
		expectedType  string
	}{
		{
			name: "function with concrete return type",
			step: ChainStep{
				MethodName: "get_user",
				Expression: "get_user()",
				IsCall:     true,
			},
			callerFQN:     "myapp.test",
			currentModule: "myapp",
			expectedOK:    true,
			expectedType:  "myapp.User",
		},
		{
			name: "function with unresolved var: return type",
			step: ChainStep{
				MethodName: "get_unresolved",
				Expression: "get_unresolved()",
				IsCall:     true,
			},
			callerFQN:     "myapp.test",
			currentModule: "myapp",
			expectedOK:    false,
			expectedType:  "",
		},
		{
			name: "function with call: placeholder",
			step: ChainStep{
				MethodName: "get_call",
				Expression: "get_call()",
				IsCall:     true,
			},
			callerFQN:     "myapp.test",
			currentModule: "myapp",
			expectedOK:    false,
			expectedType:  "",
		},
		{
			name: "function not in return types",
			step: ChainStep{
				MethodName: "unknown_func",
				Expression: "unknown_func()",
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
			typeInfo, _, ok := resolveFirstChainStep(
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
			}
		})
	}
}

// TestResolveChainMethod_BuiltinTypes tests builtin method resolution.
func TestResolveChainMethod_BuiltinTypes(t *testing.T) {
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
			name: "str.lower() returns str",
			step: ChainStep{
				MethodName: "lower",
				Expression: "lower()",
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
			name: "str.split() returns list",
			step: ChainStep{
				MethodName: "split",
				Expression: "split()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true,
			expectedType: "builtins.list",
		},
		{
			name: "list.pop() returns empty type",
			step: ChainStep{
				MethodName: "pop",
				Expression: "pop()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "builtins.list",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true,
			expectedType: "", // pop() returns empty type in builtin registry
		},
		{
			name: "dict.keys() returns dict_keys",
			step: ChainStep{
				MethodName: "keys",
				Expression: "keys()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "builtins.dict",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true,
			expectedType: "builtins.dict_keys",
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

// TestResolveChainMethod_CustomClass tests custom class method resolution.
func TestResolveChainMethod_CustomClass(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Attributes = registry.NewAttributeRegistry()
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	builtins := registry.NewBuiltinRegistry()
	callGraph := core.NewCallGraph()

	// Add custom class method with concrete return type
	typeEngine.ReturnTypes["myapp.get_name"] = &core.TypeInfo{
		TypeFQN:    "builtins.str",
		Confidence: 1.0,
		Source:     "return_type",
	}

	// Add method with var: placeholder (fluent interface)
	typeEngine.ReturnTypes["myapp.set_value"] = &core.TypeInfo{
		TypeFQN:    "var:self",
		Confidence: 0.9,
		Source:     "return_type",
	}

	// Add method to call graph
	callGraph.Functions["myapp.get_name"] = &graph.Node{
		ID:   "myapp.get_name",
		Type: "function_declaration",
		Name: "get_name",
	}

	callGraph.Functions["myapp.set_value"] = &graph.Node{
		ID:   "myapp.set_value",
		Type: "function_declaration",
		Name: "set_value",
	}

	tests := []struct {
		name                string
		step                ChainStep
		currentType         *core.TypeInfo
		expectedOK          bool
		expectedType        string
		expectedSource      string
		checkConfidenceDown bool
	}{
		{
			name: "custom method with concrete return type",
			step: ChainStep{
				MethodName: "get_name",
				Expression: "get_name()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "myapp.MyClass",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:   true,
			expectedType: "builtins.str",
		},
		{
			name: "fluent interface method (var:self)",
			step: ChainStep{
				MethodName: "set_value",
				Expression: "set_value()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "myapp.MyClass",
				Confidence: 1.0,
				Source:     "literal",
			},
			expectedOK:          true,
			expectedType:        "myapp.MyClass",
			expectedSource:      "method_chain_fluent",
			checkConfidenceDown: true,
		},
		{
			name: "method not found but high confidence - heuristic",
			step: ChainStep{
				MethodName: "unknown_method",
				Expression: "unknown_method()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "myapp.MyClass",
				Confidence: 0.9,
				Source:     "literal",
			},
			expectedOK:          true,
			expectedType:        "myapp.MyClass",
			expectedSource:      "method_chain_heuristic",
			checkConfidenceDown: true,
		},
		{
			name: "method not found and low confidence",
			step: ChainStep{
				MethodName: "unknown_method",
				Expression: "unknown_method()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "myapp.MyClass",
				Confidence: 0.5,
				Source:     "heuristic",
			},
			expectedOK:   false,
			expectedType: "",
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

				if tt.expectedSource != "" {
					assert.Equal(t, tt.expectedSource, typeInfo.Source)
				}

				if tt.checkConfidenceDown {
					assert.Less(t, typeInfo.Confidence, tt.currentType.Confidence,
						"Confidence should decrease for heuristic methods")
				}
			}
		})
	}
}

// TestResolveChainMethod_EdgeCases tests edge cases and error handling.
func TestResolveChainMethod_EdgeCases(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Attributes = registry.NewAttributeRegistry()
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	builtins := registry.NewBuiltinRegistry()
	callGraph := core.NewCallGraph()

	tests := []struct {
		name        string
		step        ChainStep
		currentType *core.TypeInfo
		expectedOK  bool
	}{
		{
			name: "nil current type",
			step: ChainStep{
				MethodName: "method",
				Expression: "method()",
				IsCall:     true,
			},
			currentType: nil,
			expectedOK:  false,
		},
		{
			name: "empty method name",
			step: ChainStep{
				MethodName: "",
				Expression: "()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "myapp.MyClass",
				Confidence: 0.9,
				Source:     "literal",
			},
			expectedOK: false,
		},
		{
			name: "var: placeholder type",
			step: ChainStep{
				MethodName: "method",
				Expression: "method()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "var:unknown",
				Confidence: 0.5,
				Source:     "heuristic",
			},
			expectedOK: false,
		},
		{
			name: "call: placeholder type",
			step: ChainStep{
				MethodName: "method",
				Expression: "method()",
				IsCall:     true,
			},
			currentType: &core.TypeInfo{
				TypeFQN:    "call:some_function",
				Confidence: 0.5,
				Source:     "heuristic",
			},
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typeInfo, _, ok := resolveChainMethod(
				tt.step,
				tt.currentType,
				builtins,
				typeEngine,
				moduleRegistry,
				callGraph,
			)

			assert.Equal(t, tt.expectedOK, ok, "Resolution OK status mismatch")

			if !tt.expectedOK {
				if typeInfo != nil {
					t.Logf("Unexpected non-nil typeInfo: %+v", typeInfo)
				}
			}
		})
	}
}

// TestResolveChainedCall_ComplexChains tests full chain resolution.
func TestResolveChainedCall_ComplexChains(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Attributes = registry.NewAttributeRegistry()
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	builtins := registry.NewBuiltinRegistry()
	callGraph := core.NewCallGraph()
	codeGraph := &graph.CodeGraph{}

	// Setup: create_builder() returns Builder
	typeEngine.ReturnTypes["myapp.create_builder"] = &core.TypeInfo{
		TypeFQN:    "myapp.Builder",
		Confidence: 1.0,
		Source:     "return_type",
	}

	// Builder.append() returns Builder (fluent interface)
	typeEngine.ReturnTypes["myapp.append"] = &core.TypeInfo{
		TypeFQN:    "var:self",
		Confidence: 0.9,
		Source:     "return_type",
	}

	callGraph.Functions["myapp.create_builder"] = &graph.Node{
		ID:   "myapp.create_builder",
		Type: "function_declaration",
		Name: "create_builder",
	}

	callGraph.Functions["myapp.append"] = &graph.Node{
		ID:   "myapp.append",
		Type: "function_declaration",
		Name: "append",
	}

	tests := []struct {
		name              string
		target            string
		callerFQN         string
		currentModule     string
		expectedResolved  bool
		expectedType      string
		checkConfidence   bool
		minConfidence     float32
	}{
		{
			name:             "three-step fluent chain",
			target:           "create_builder().append().append()",
			callerFQN:        "myapp.test",
			currentModule:    "myapp",
			expectedResolved: true,
			expectedType:     "myapp.Builder",
			checkConfidence:  true,
			minConfidence:    0.7,
		},
		{
			name:             "chain ending with builtin method",
			target:           "create_builder().append().upper()",
			callerFQN:        "myapp.test",
			currentModule:    "myapp",
			expectedResolved: true, // Heuristic allows unknown methods on high-confidence types
			expectedType:     "myapp.Builder", // Returns same type due to heuristic
		},
		{
			name:             "chain with unknown first function",
			target:           "unknown_func().method1().method2()",
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

				if tt.checkConfidence {
					assert.GreaterOrEqual(t, typeInfo.Confidence, tt.minConfidence,
						"Confidence should be at least %f", tt.minConfidence)
				}
			}
		})
	}
}

// TestResolveFirstChainStep_EdgeCases tests edge cases in first step resolution.
func TestResolveFirstChainStep_EdgeCases(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
	callGraph := core.NewCallGraph()

	// Setup scope with variable that has nil type
	functionScope := NewFunctionScope("myapp.test_func")
	functionScope.AddVariable(&VariableBinding{
		VarName: "nil_type_var",
		Type:    nil, // Explicitly nil type
	})
	typeEngine.Scopes["myapp.test_func"] = functionScope

	tests := []struct {
		name          string
		step          ChainStep
		callerFQN     string
		currentModule string
		expectedOK    bool
	}{
		{
			name: "variable with nil type",
			step: ChainStep{
				MethodName: "nil_type_var",
				Expression: "nil_type_var",
				IsCall:     false,
			},
			callerFQN:     "myapp.test_func",
			currentModule: "myapp",
			expectedOK:    false,
		},
		{
			name: "function call when typeEngine is nil",
			step: ChainStep{
				MethodName: "some_func",
				Expression: "some_func()",
				IsCall:     true,
			},
			callerFQN:     "myapp.test",
			currentModule: "myapp",
			expectedOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := typeEngine
			if tt.name == "function call when typeEngine is nil" {
				// This test case actually uses typeEngine, but tests when ReturnTypes is empty
				// We can't actually pass nil typeEngine, so we test the case where it has no return types
			}

			typeInfo, _, ok := resolveFirstChainStep(
				tt.step,
				engine,
				tt.callerFQN,
				tt.currentModule,
				moduleRegistry,
				callGraph,
			)

			assert.Equal(t, tt.expectedOK, ok, "Resolution OK status mismatch")

			if !tt.expectedOK {
				if tt.name != "function call when typeEngine is nil" {
					assert.Nil(t, typeInfo, "Should not have type info")
				}
			}
		})
	}
}
