package resolution

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
)

// TestResolveSelfAttributeCall tests the core attribute resolution function.
func TestResolveSelfAttributeCall(t *testing.T) {
	// Reset failure stats before each test
	attributeFailureStats = &FailureStats{
		DeepChainSamples:         make([]string, 0, 20),
		AttributeNotFoundSamples: make([]string, 0, 20),
		CustomClassSamples:       make([]string, 0, 20),
	}

	tests := []struct {
		name             string
		target           string
		callerFQN        string
		setupFunc        func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph)
		expectedResolved bool
		expectedFQN      string
		expectedTypeInfo *core.TypeInfo
		checkStats       func(*testing.T)
	}{
		{
			name:      "basic self.attr.method() with builtin type",
			target:    "self.value.upper",
			callerFQN: "test_module.StringBuilder.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()

				// Add class with attribute
				typeEngine.Attributes.AddAttribute("test_module.StringBuilder", &core.ClassAttribute{
					Name: "value",
					Type: &core.TypeInfo{
						TypeFQN:    "builtins.str",
						Confidence: 1.0,
						Source:     "annotation",
					},
					Confidence: 1.0,
				})

				// Add method to class
				classAttrs := typeEngine.Attributes.GetClassAttributes("test_module.StringBuilder")
				classAttrs.Methods = append(classAttrs.Methods, "test_module.StringBuilder.process")

				return typeEngine, builtins, callGraph
			},
			expectedResolved: true,
			expectedFQN:      "builtins.str.upper",
			expectedTypeInfo: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 1.0,
				Source:     "self_attribute",
			},
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 0, attributeFailureStats.NotSelfPrefix)
				assert.Equal(t, 0, attributeFailureStats.DeepChains)
			},
		},
		{
			name:      "non-self prefix should fail",
			target:    "other.value.upper",
			callerFQN: "test_module.StringBuilder.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()
				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 1, attributeFailureStats.NotSelfPrefix)
			},
		},
		{
			name:      "deep chain (3+ levels) should fail",
			target:    "self.obj.attr.method",
			callerFQN: "test_module.MyClass.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()
				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 1, attributeFailureStats.DeepChains)
				assert.Equal(t, 1, len(attributeFailureStats.DeepChainSamples))
				assert.Equal(t, "self.obj.attr.method", attributeFailureStats.DeepChainSamples[0])
			},
		},
		{
			name:      "class not found should fail",
			target:    "self.value.upper",
			callerFQN: "unknown_module.UnknownClass.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()
				// Don't add any class
				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 1, attributeFailureStats.ClassNotFound)
			},
		},
		{
			name:      "attribute not found should fail",
			target:    "self.missing_attr.upper",
			callerFQN: "test_module.StringBuilder.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()

				// Add class with a dummy attribute and method, but not the target attribute
				typeEngine.Attributes.AddClassAttributes(&core.ClassAttributes{
					ClassFQN:   "test_module.StringBuilder",
					Attributes: make(map[string]*core.ClassAttribute),
					Methods:    []string{"test_module.StringBuilder.process"},
				})

				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 1, attributeFailureStats.AttributeNotFound)
				assert.Equal(t, 1, len(attributeFailureStats.AttributeNotFoundSamples))
			},
		},
		{
			name:      "method not in builtins should fail",
			target:    "self.value.nonexistent_method",
			callerFQN: "test_module.StringBuilder.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()

				// Add class with attribute
				typeEngine.Attributes.AddAttribute("test_module.StringBuilder", &core.ClassAttribute{
					Name: "value",
					Type: &core.TypeInfo{
						TypeFQN:    "builtins.str",
						Confidence: 1.0,
						Source:     "annotation",
					},
					Confidence: 1.0,
				})

				// Add method to class
				classAttrs := typeEngine.Attributes.GetClassAttributes("test_module.StringBuilder")
				classAttrs.Methods = append(classAttrs.Methods, "test_module.StringBuilder.process")

				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 1, attributeFailureStats.MethodNotInBuiltins)
			},
		},
		{
			name:      "custom class type (unsupported for now)",
			target:    "self.user.get_name",
			callerFQN: "test_module.Controller.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()

				// Add class with custom type attribute
				typeEngine.Attributes.AddAttribute("test_module.Controller", &core.ClassAttribute{
					Name: "user",
					Type: &core.TypeInfo{
						TypeFQN:    "test_module.User",
						Confidence: 1.0,
						Source:     "annotation",
					},
					Confidence: 1.0,
				})

				// Add method to class
				classAttrs := typeEngine.Attributes.GetClassAttributes("test_module.Controller")
				classAttrs.Methods = append(classAttrs.Methods, "test_module.Controller.process")

				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 1, attributeFailureStats.CustomClassUnsupported)
				assert.Equal(t, 1, len(attributeFailureStats.CustomClassSamples))
			},
		},
		{
			name:      "too few dots (self.method)",
			target:    "self.method",
			callerFQN: "test_module.MyClass.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()
				return typeEngine, builtins, callGraph
			},
			expectedResolved: false,
			expectedFQN:      "",
			expectedTypeInfo: nil,
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
			},
		},
		{
			name:      "list type with append method",
			target:    "self.items.append",
			callerFQN: "test_module.Container.add_item",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()

				// Add class with list attribute
				typeEngine.Attributes.AddAttribute("test_module.Container", &core.ClassAttribute{
					Name: "items",
					Type: &core.TypeInfo{
						TypeFQN:    "builtins.list",
						Confidence: 1.0,
						Source:     "annotation",
					},
					Confidence: 1.0,
				})

				// Add method to class
				classAttrs := typeEngine.Attributes.GetClassAttributes("test_module.Container")
				classAttrs.Methods = append(classAttrs.Methods, "test_module.Container.add_item")

				return typeEngine, builtins, callGraph
			},
			expectedResolved: true,
			expectedFQN:      "builtins.list.append",
			expectedTypeInfo: &core.TypeInfo{
				TypeFQN:    "builtins.NoneType",
				Confidence: 1.0,
				Source:     "self_attribute",
			},
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
				assert.Equal(t, 0, attributeFailureStats.NotSelfPrefix)
			},
		},
		{
			name:      "attribute with lower confidence",
			target:    "self.value.upper",
			callerFQN: "test_module.StringBuilder.process",
			setupFunc: func() (*TypeInferenceEngine, *registry.BuiltinRegistry, *core.CallGraph) {
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.Attributes = registry.NewAttributeRegistry()
				builtins := registry.NewBuiltinRegistry()
				callGraph := core.NewCallGraph()

				// Add class with low-confidence attribute
				typeEngine.Attributes.AddAttribute("test_module.StringBuilder", &core.ClassAttribute{
					Name: "value",
					Type: &core.TypeInfo{
						TypeFQN:    "builtins.str",
						Confidence: 0.5,
						Source:     "heuristic",
					},
					Confidence: 0.5,
				})

				// Add method to class
				classAttrs := typeEngine.Attributes.GetClassAttributes("test_module.StringBuilder")
				classAttrs.Methods = append(classAttrs.Methods, "test_module.StringBuilder.process")

				return typeEngine, builtins, callGraph
			},
			expectedResolved: true,
			expectedFQN:      "builtins.str.upper",
			expectedTypeInfo: &core.TypeInfo{
				TypeFQN:    "builtins.str",
				Confidence: 0.5, // Inherits attribute confidence
				Source:     "self_attribute",
			},
			checkStats: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 1, attributeFailureStats.TotalAttempts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset stats for this test
			attributeFailureStats = &FailureStats{
				DeepChainSamples:         make([]string, 0, 20),
				AttributeNotFoundSamples: make([]string, 0, 20),
				CustomClassSamples:       make([]string, 0, 20),
			}

			typeEngine, builtins, callGraph := tt.setupFunc()

			resolvedFQN, resolved, typeInfo := ResolveSelfAttributeCall(
				tt.target,
				tt.callerFQN,
				typeEngine,
				builtins,
				callGraph,
			)

			assert.Equal(t, tt.expectedResolved, resolved, "Resolution status mismatch")
			assert.Equal(t, tt.expectedFQN, resolvedFQN, "Resolved FQN mismatch")

			if tt.expectedTypeInfo != nil {
				assert.NotNil(t, typeInfo, "Expected type info but got nil")
				assert.Equal(t, tt.expectedTypeInfo.TypeFQN, typeInfo.TypeFQN)
				assert.Equal(t, tt.expectedTypeInfo.Confidence, typeInfo.Confidence)
				assert.Equal(t, tt.expectedTypeInfo.Source, typeInfo.Source)
			} else {
				assert.Nil(t, typeInfo, "Expected nil type info")
			}

			if tt.checkStats != nil {
				tt.checkStats(t)
			}
		})
	}
}

// TestPrintAttributeFailureStats tests the statistics printing function.
func TestPrintAttributeFailureStats(t *testing.T) {
	tests := []struct {
		name          string
		setupStats    func()
		expectOutput  bool
		checkOutput   func(*testing.T, string)
	}{
		{
			name: "no attempts - should be silent",
			setupStats: func() {
				attributeFailureStats = &FailureStats{
					TotalAttempts:            0,
					DeepChainSamples:         make([]string, 0, 20),
					AttributeNotFoundSamples: make([]string, 0, 20),
					CustomClassSamples:       make([]string, 0, 20),
				}
			},
			expectOutput: false,
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				assert.Empty(t, output, "Expected no output when no attempts")
			},
		},
		{
			name: "with attempts and various failures",
			setupStats: func() {
				attributeFailureStats = &FailureStats{
					TotalAttempts:          100,
					NotSelfPrefix:          20,
					DeepChains:             15,
					ClassNotFound:          10,
					AttributeNotFound:      25,
					MethodNotInBuiltins:    20,
					CustomClassUnsupported: 10,
					DeepChainSamples:       []string{"self.a.b.c", "self.x.y.z"},
					AttributeNotFoundSamples: []string{"self.missing.method (in class test.MyClass)"},
					CustomClassSamples:     []string{"self.user.get_name (type: myapp.User)"},
				}
			},
			expectOutput: true,
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "[ATTR_FAILURE_ANALYSIS]")
				assert.Contains(t, output, "Total attempts:              100")
				assert.Contains(t, output, "Not self prefix:           20")
				assert.Contains(t, output, "Deep chains (3+ levels):   15")
				assert.Contains(t, output, "Class not found:           10")
				assert.Contains(t, output, "Attribute not found:       25")
				assert.Contains(t, output, "Method not in builtins:    20")
				assert.Contains(t, output, "Custom class unsupported:  10")
				assert.Contains(t, output, "Deep chain samples")
				assert.Contains(t, output, "self.a.b.c")
				assert.Contains(t, output, "Attribute not found samples")
				assert.Contains(t, output, "Custom class samples")
			},
		},
		{
			name: "with many samples (should limit to 10)",
			setupStats: func() {
				samples := make([]string, 20)
				for i := 0; i < 20; i++ {
					samples[i] = "sample" + string(rune('0'+i))
				}
				attributeFailureStats = &FailureStats{
					TotalAttempts:            50,
					DeepChains:               20,
					DeepChainSamples:         samples,
					AttributeNotFoundSamples: make([]string, 0, 20),
					CustomClassSamples:       make([]string, 0, 20),
				}
			},
			expectOutput: true,
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				// Count how many samples are printed (should be max 10)
				lines := strings.Split(output, "\n")
				sampleLines := 0
				inSampleSection := false
				for _, line := range lines {
					switch {
					case strings.Contains(line, "Deep chain samples"):
						inSampleSection = true
					case inSampleSection && strings.HasPrefix(strings.TrimSpace(line), "-"):
						sampleLines++
					case inSampleSection && !strings.HasPrefix(strings.TrimSpace(line), "-") && strings.TrimSpace(line) != "":
						goto exitLoop
					}
				}
			exitLoop:
				assert.LessOrEqual(t, sampleLines, 10, "Should print at most 10 samples")
			},
		},
		{
			name: "empty sample lists should not print sections",
			setupStats: func() {
				attributeFailureStats = &FailureStats{
					TotalAttempts:            10,
					NotSelfPrefix:            10,
					DeepChainSamples:         make([]string, 0, 20),
					AttributeNotFoundSamples: make([]string, 0, 20),
					CustomClassSamples:       make([]string, 0, 20),
				}
			},
			expectOutput: true,
			checkOutput: func(t *testing.T, output string) {
				t.Helper()
				assert.NotContains(t, output, "Deep chain samples")
				assert.NotContains(t, output, "Attribute not found samples")
				assert.NotContains(t, output, "Custom class samples")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupStats()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PrintAttributeFailureStats()

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tt.expectOutput {
				assert.NotEmpty(t, output, "Expected output but got none")
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

// TestFindClassContainingMethod tests the internal class lookup function.
func TestFindClassContainingMethod(t *testing.T) {
	tests := []struct {
		name           string
		methodFQN      string
		setupRegistry  func() *registry.AttributeRegistry
		expectedClass  string
	}{
		{
			name:      "method found in class",
			methodFQN: "test_module.process",
			setupRegistry: func() *registry.AttributeRegistry {
				reg := registry.NewAttributeRegistry()
				reg.AddClassAttributes(&core.ClassAttributes{
					ClassFQN:   "test_module.MyClass",
					Attributes: make(map[string]*core.ClassAttribute),
					Methods:    []string{"test_module.MyClass.process"},
				})
				return reg
			},
			expectedClass: "test_module.MyClass",
		},
		{
			name:      "method not found",
			methodFQN: "test_module.unknown",
			setupRegistry: func() *registry.AttributeRegistry {
				reg := registry.NewAttributeRegistry()
				reg.AddClassAttributes(&core.ClassAttributes{
					ClassFQN:   "test_module.MyClass",
					Attributes: make(map[string]*core.ClassAttribute),
					Methods:    []string{"test_module.MyClass.process"},
				})
				return reg
			},
			expectedClass: "",
		},
		{
			name:      "multiple classes - finds correct one",
			methodFQN: "test_module.calculate",
			setupRegistry: func() *registry.AttributeRegistry {
				reg := registry.NewAttributeRegistry()
				reg.AddClassAttributes(&core.ClassAttributes{
					ClassFQN:   "test_module.ClassA",
					Attributes: make(map[string]*core.ClassAttribute),
					Methods:    []string{"test_module.ClassA.process"},
				})
				reg.AddClassAttributes(&core.ClassAttributes{
					ClassFQN:   "test_module.ClassB",
					Attributes: make(map[string]*core.ClassAttribute),
					Methods:    []string{"test_module.ClassB.calculate"},
				})
				return reg
			},
			expectedClass: "test_module.ClassB",
		},
		{
			name:      "method name without module",
			methodFQN: "process",
			setupRegistry: newAttributeRegistryWithClass("test_module.MyClass", []string{"test_module.MyClass.process"}),
			expectedClass: "test_module.MyClass",
		},
		{
			name:          "empty registry",
			methodFQN:     "test_module.process",
			setupRegistry: registry.NewAttributeRegistry,
			expectedClass: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := tt.setupRegistry()
			result := findClassContainingMethod(tt.methodFQN, reg)
			assert.Equal(t, tt.expectedClass, result)
		})
	}
}

// TestResolveAttributePlaceholders tests placeholder resolution.
func TestResolveAttributePlaceholders(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph)
		checkFunc     func(*testing.T, *registry.AttributeRegistry)
	}{
		{
			name: "resolve class: placeholder",
			setupFunc: func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph) {
				attrRegistry := registry.NewAttributeRegistry()
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				userNode := &graph.Node{
					ID:   "test_module.User",
					Type: "class_declaration",
					Name: "test_module.User",
				}
				codeGraph := &graph.CodeGraph{
					Nodes: map[string]*graph.Node{
						"test_module.User": userNode,
					},
				}

				// Add class with class: placeholder
				attrRegistry.AddAttribute("test_module.MyClass", &core.ClassAttribute{
					Name: "user",
					Type: &core.TypeInfo{
						TypeFQN:    "class:User",
						Confidence: 0.5,
						Source:     "heuristic",
					},
					Confidence: 0.5,
				})

				return attrRegistry, typeEngine, moduleRegistry, codeGraph
			},
			checkFunc: func(t *testing.T, reg *registry.AttributeRegistry) {
				t.Helper()
				attr := reg.GetAttribute("test_module.MyClass", "user")
				assert.NotNil(t, attr)
				assert.Equal(t, "test_module.User", attr.Type.TypeFQN)
				assert.Equal(t, float32(0.9), attr.Type.Confidence)
			},
		},
		{
			name: "resolve call: placeholder",
			setupFunc: func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph) {
				attrRegistry := registry.NewAttributeRegistry()
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
				codeGraph := &graph.CodeGraph{}

				// Add return type for function
				typeEngine.ReturnTypes["test_module.get_user"] = &core.TypeInfo{
					TypeFQN:    "test_module.User",
					Confidence: 1.0,
					Source:     "return_analysis",
				}

				// Add class with call: placeholder
				attrRegistry.AddAttribute("test_module.MyClass", &core.ClassAttribute{
					Name: "user",
					Type: &core.TypeInfo{
						TypeFQN:    "call:get_user",
						Confidence: 0.5,
						Source:     "call",
					},
					Confidence: 0.5,
				})

				return attrRegistry, typeEngine, moduleRegistry, codeGraph
			},
			checkFunc: func(t *testing.T, reg *registry.AttributeRegistry) {
				t.Helper()
				attr := reg.GetAttribute("test_module.MyClass", "user")
				assert.NotNil(t, attr)
				assert.Equal(t, "test_module.User", attr.Type.TypeFQN)
				assert.Equal(t, float32(0.8), attr.Type.Confidence) // Decayed
				assert.Equal(t, "function_call_attribute", attr.Type.Source)
			},
		},
		{
			name: "resolve param: placeholder",
			setupFunc: func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph) {
				attrRegistry := registry.NewAttributeRegistry()
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				userNode := &graph.Node{
					ID:   "test_module.User",
					Type: "class_declaration",
					Name: "test_module.User",
				}
				codeGraph := &graph.CodeGraph{
					Nodes: map[string]*graph.Node{
						"test_module.User": userNode,
					},
				}

				// Add class with param: placeholder
				attrRegistry.AddAttribute("test_module.MyClass", &core.ClassAttribute{
					Name: "user",
					Type: &core.TypeInfo{
						TypeFQN:    "param:User",
						Confidence: 0.5,
						Source:     "parameter",
					},
					Confidence: 0.5,
				})

				return attrRegistry, typeEngine, moduleRegistry, codeGraph
			},
			checkFunc: func(t *testing.T, reg *registry.AttributeRegistry) {
				t.Helper()
				attr := reg.GetAttribute("test_module.MyClass", "user")
				assert.NotNil(t, attr)
				assert.Equal(t, "test_module.User", attr.Type.TypeFQN)
				assert.Equal(t, float32(0.95), attr.Type.Confidence)
			},
		},
		{
			name: "already resolved type should not change",
			setupFunc: func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph) {
				attrRegistry := registry.NewAttributeRegistry()
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				codeGraph := &graph.CodeGraph{
					Nodes: make(map[string]*graph.Node),
				}

				// Add class with already resolved type
				attrRegistry.AddAttribute("test_module.MyClass", &core.ClassAttribute{
					Name: "value",
					Type: &core.TypeInfo{
						TypeFQN:    "builtins.str",
						Confidence: 1.0,
						Source:     "annotation",
					},
					Confidence: 1.0,
				})

				return attrRegistry, typeEngine, moduleRegistry, codeGraph
			},
			checkFunc: func(t *testing.T, reg *registry.AttributeRegistry) {
				t.Helper()
				attr := reg.GetAttribute("test_module.MyClass", "value")
				assert.NotNil(t, attr)
				assert.Equal(t, "builtins.str", attr.Type.TypeFQN)
				assert.Equal(t, float32(1.0), attr.Type.Confidence)
				assert.Equal(t, "annotation", attr.Type.Source)
			},
		},
		{
			name: "class: placeholder not found",
			setupFunc: func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph) {
				attrRegistry := registry.NewAttributeRegistry()
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				codeGraph := &graph.CodeGraph{
					Nodes: make(map[string]*graph.Node),
				}

				// Add class with class: placeholder that won't resolve
				attrRegistry.AddAttribute("test_module.MyClass", &core.ClassAttribute{
					Name: "user",
					Type: &core.TypeInfo{
						TypeFQN:    "class:NonExistent",
						Confidence: 0.5,
						Source:     "heuristic",
					},
					Confidence: 0.5,
				})

				return attrRegistry, typeEngine, moduleRegistry, codeGraph
			},
			checkFunc: func(t *testing.T, reg *registry.AttributeRegistry) {
				t.Helper()
				attr := reg.GetAttribute("test_module.MyClass", "user")
				assert.NotNil(t, attr)
				// Should remain as placeholder
				assert.Equal(t, "class:NonExistent", attr.Type.TypeFQN)
				assert.Equal(t, float32(0.5), attr.Type.Confidence)
			},
		},
		{
			name: "call: placeholder function not found",
			setupFunc: func() (*registry.AttributeRegistry, *TypeInferenceEngine, *core.ModuleRegistry, *graph.CodeGraph) {
				attrRegistry := registry.NewAttributeRegistry()
				moduleRegistry := core.NewModuleRegistry()
				typeEngine := NewTypeInferenceEngine(moduleRegistry)
				typeEngine.ReturnTypes = make(map[string]*core.TypeInfo)
				codeGraph := &graph.CodeGraph{
					Nodes: make(map[string]*graph.Node),
				}

				// Add class with call: placeholder for non-existent function
				attrRegistry.AddAttribute("test_module.MyClass", &core.ClassAttribute{
					Name: "result",
					Type: &core.TypeInfo{
						TypeFQN:    "call:nonexistent_func",
						Confidence: 0.5,
						Source:     "call",
					},
					Confidence: 0.5,
				})

				return attrRegistry, typeEngine, moduleRegistry, codeGraph
			},
			checkFunc: func(t *testing.T, reg *registry.AttributeRegistry) {
				t.Helper()
				attr := reg.GetAttribute("test_module.MyClass", "result")
				assert.NotNil(t, attr)
				// Should remain as placeholder
				assert.Equal(t, "call:nonexistent_func", attr.Type.TypeFQN)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrRegistry, typeEngine, moduleRegistry, codeGraph := tt.setupFunc()

			ResolveAttributePlaceholders(attrRegistry, typeEngine, moduleRegistry, codeGraph)

			tt.checkFunc(t, attrRegistry)
		})
	}
}

// TestResolveClassName tests class name resolution.
func TestResolveClassName(t *testing.T) {
	tests := []struct {
		name            string
		className       string
		contextClassFQN string
		setupFunc       func() (*core.ModuleRegistry, *graph.CodeGraph)
		expectedFQN     string
	}{
		{
			name:            "class in same module",
			className:       "User",
			contextClassFQN: "test_module.MyClass",
			setupFunc: func() (*core.ModuleRegistry, *graph.CodeGraph) {
				moduleRegistry := core.NewModuleRegistry()
				userNode := &graph.Node{
					ID:   "test_module.User",
					Type: "class_declaration",
					Name: "test_module.User",
				}
				codeGraph := &graph.CodeGraph{
					Nodes: map[string]*graph.Node{
						"test_module.User": userNode,
					},
				}
				return moduleRegistry, codeGraph
			},
			expectedFQN: "test_module.User",
		},
		{
			name:            "class in different module via short name",
			className:       "User",
			contextClassFQN: "test_module.MyClass",
			setupFunc: func() (*core.ModuleRegistry, *graph.CodeGraph) {
				moduleRegistry := core.NewModuleRegistry()
				moduleRegistry.ShortNames["User"] = []string{"/path/to/models.py"}
				moduleRegistry.FileToModule["/path/to/models.py"] = "myapp.models"
				codeGraph := &graph.CodeGraph{}
				return moduleRegistry, codeGraph
			},
			expectedFQN: "myapp.models.User",
		},
		{
			name:            "class not found",
			className:       "NonExistent",
			contextClassFQN: "test_module.MyClass",
			setupFunc: func() (*core.ModuleRegistry, *graph.CodeGraph) {
				moduleRegistry := core.NewModuleRegistry()
				codeGraph := &graph.CodeGraph{
					Nodes: make(map[string]*graph.Node),
				}
				return moduleRegistry, codeGraph
			},
			expectedFQN: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moduleRegistry, codeGraph := tt.setupFunc()
			result := resolveClassName(tt.className, tt.contextClassFQN, moduleRegistry, codeGraph)
			assert.Equal(t, tt.expectedFQN, result)
		})
	}
}

// TestClassExists tests class existence checking.
func TestClassExists(t *testing.T) {
	tests := []struct {
		name      string
		classFQN  string
		codeGraph *graph.CodeGraph
		expected  bool
	}{
		{
			name:     "class exists",
			classFQN: "test_module.MyClass",
			codeGraph: &graph.CodeGraph{
				Nodes: map[string]*graph.Node{
					"test_module.MyClass": {
						ID:   "test_module.MyClass",
						Type: "class_declaration",
						Name: "test_module.MyClass",
					},
				},
			},
			expected: true,
		},
		{
			name:     "class does not exist",
			classFQN: "test_module.NonExistent",
			codeGraph: &graph.CodeGraph{
				Nodes: map[string]*graph.Node{
					"test_module.MyClass": {
						ID:   "test_module.MyClass",
						Type: "class_declaration",
						Name: "test_module.MyClass",
					},
				},
			},
			expected: false,
		},
		{
			name:     "empty graph",
			classFQN: "test_module.MyClass",
			codeGraph: &graph.CodeGraph{
				Nodes: make(map[string]*graph.Node),
			},
			expected: false,
		},
		{
			name:     "wrong node type",
			classFQN: "test_module.MyClass",
			codeGraph: &graph.CodeGraph{
				Nodes: map[string]*graph.Node{
					"test_module.MyClass": {
						ID:   "test_module.MyClass",
						Type: "function_declaration",
						Name: "test_module.MyClass",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classExists(tt.classFQN, tt.codeGraph)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetModuleFromClassFQN tests module extraction from class FQN.
func TestGetModuleFromClassFQN(t *testing.T) {
	tests := []struct {
		name           string
		classFQN       string
		expectedModule string
	}{
		{
			name:           "simple two-part FQN",
			classFQN:       "test_module.MyClass",
			expectedModule: "test_module",
		},
		{
			name:           "multi-level module",
			classFQN:       "myapp.models.User",
			expectedModule: "myapp.models",
		},
		{
			name:           "deeply nested",
			classFQN:       "com.example.app.models.User",
			expectedModule: "com.example.app.models",
		},
		{
			name:           "single part (no module)",
			classFQN:       "MyClass",
			expectedModule: "MyClass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModuleFromClassFQN(tt.classFQN)
			assert.Equal(t, tt.expectedModule, result)
		})
	}
}

// TestFailureStats_SampleLimit tests that samples are limited to 20.
func TestFailureStats_SampleLimit(t *testing.T) {
	// Reset stats
	attributeFailureStats = &FailureStats{
		DeepChainSamples:         make([]string, 0, 20),
		AttributeNotFoundSamples: make([]string, 0, 20),
		CustomClassSamples:       make([]string, 0, 20),
	}

	moduleRegistry := core.NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(moduleRegistry)
	typeEngine.Attributes = registry.NewAttributeRegistry()
	builtins := registry.NewBuiltinRegistry()
	callGraph := core.NewCallGraph()

	// Try to add 30 deep chain samples
	for i := 0; i < 30; i++ {
		target := "self.a.b.c"
		callerFQN := "test.Class.method"
		ResolveSelfAttributeCall(target, callerFQN, typeEngine, builtins, callGraph)
	}

	// Should only have 20 samples
	assert.Equal(t, 20, len(attributeFailureStats.DeepChainSamples))
	assert.Equal(t, 30, attributeFailureStats.TotalAttempts)
	assert.Equal(t, 30, attributeFailureStats.DeepChains)
}

// newAttributeRegistryWithClass is a helper to create an AttributeRegistry with a single class.
func newAttributeRegistryWithClass(classFQN string, methods []string) func() *registry.AttributeRegistry {
	return func() *registry.AttributeRegistry {
		reg := registry.NewAttributeRegistry()
		reg.AddClassAttributes(&core.ClassAttributes{
			ClassFQN:   classFQN,
			Attributes: make(map[string]*core.ClassAttribute),
			Methods:    methods,
		})
		return reg
	}
}
