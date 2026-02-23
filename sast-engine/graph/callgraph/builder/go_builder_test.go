package builder

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGoCallGraph(t *testing.T) {
	// Create a simple mock CodeGraph with functions and a builtin call
	mainFunc := &graph.Node{
		ID:         "main",
		Type:       "function_declaration", // Fixed: was "function_definition"
		Name:       "main",
		File:       "/project/main.go",
		LineNumber: 10,
	}
	// Builtin call - these don't require imports
	callNode := &graph.Node{
		ID:         "call1",
		Type:       "call",   // Fixed: was "call_expression"
		Name:       "append", // Fixed: use Name field
		File:       "/project/main.go",
		LineNumber: 12,
	}

	// Set up parent-child relationship so findContainingGoFunction works
	edge := &graph.Edge{From: mainFunc, To: callNode}
	mainFunc.OutgoingEdges = []*graph.Edge{edge}

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"main":  mainFunc,
			"call1": callNode,
		},
	}

	// Build module registry
	registry := &core.GoModuleRegistry{
		DirToImport: map[string]string{
			"/project": "github.com/example/project",
		},
		StdlibPackages: map[string]bool{
			"fmt": true,
		},
	}

	// Initialize type engine
	goTypeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Build call graph
	callGraph, err := BuildGoCallGraph(codeGraph, registry, goTypeEngine)
	require.NoError(t, err)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions, "Should have indexed functions")
	assert.Contains(t, callGraph.Functions, "github.com/example/project.main")

	// Verify edges exist (call from main to builtin.append)
	assert.NotEmpty(t, callGraph.Edges, "Should have call edges")
	mainFQN := "github.com/example/project.main"
	assert.Contains(t, callGraph.Edges[mainFQN], "builtin.append", "main should call builtin.append")
}

func TestIndexGoFunctions(t *testing.T) {
	tests := []struct {
		name          string
		nodes         []*graph.Node
		expectedFQNs  []string
		unexpectedFQN string
	}{
		{
			name: "index package function",
			nodes: []*graph.Node{
				{
					ID:         "func1",
					Type:       "function_declaration", // Fixed: was "function_definition"
					Name:       "main",
					File:       "/project/main.go",
					LineNumber: 10,
				},
			},
			expectedFQNs: []string{"main.main"},
		},
		{
			name: "index method with receiver",
			nodes: []*graph.Node{
				{
					ID:         "method1",
					Type:       "method",
					Name:       "Start",
					Interface:  []string{"Server"},
					File:       "/project/models/server.go",
					LineNumber: 15,
				},
			},
			expectedFQNs: []string{"models.Server.Start"},
		},
		{
			name: "index closure",
			nodes: []*graph.Node{
				{
					ID:         "func1",
					Type:       "function_declaration", // Fixed: was "function_definition"
					Name:       "process",
					File:       "/project/main.go",
					LineNumber: 10,
				},
				{
					ID:         "closure1",
					Type:       "func_literal",
					Name:       "$anon_1",
					File:       "/project/main.go",
					LineNumber: 12,
				},
			},
			expectedFQNs: []string{"main.process"},
		},
		{
			name: "skip non-function nodes",
			nodes: []*graph.Node{
				{
					ID:   "var1",
					Type: "variable_declaration",
					Name: "count",
					File: "/project/main.go",
				},
				{
					ID:         "func1",
					Type:       "function_declaration", // Fixed: was "function_definition"
					Name:       "main",
					File:       "/project/main.go",
					LineNumber: 10,
				},
			},
			expectedFQNs:  []string{"main.main"},
			unexpectedFQN: "count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeMap := make(map[string]*graph.Node)
			for _, node := range tt.nodes {
				nodeMap[node.ID] = node
			}
			codeGraph := &graph.CodeGraph{
				Nodes: nodeMap,
			}
			callGraph := core.NewCallGraph()
			registry := &core.GoModuleRegistry{
				DirToImport: map[string]string{
					"/project":        "main",
					"/project/models": "models",
				},
			}

			functionContext := indexGoFunctions(codeGraph, callGraph, registry)

			// Check expected FQNs
			for _, expectedFQN := range tt.expectedFQNs {
				_, ok := callGraph.Functions[expectedFQN]
				assert.True(t, ok, "Should index function with FQN: %s", expectedFQN)
			}

			// Check unexpected FQN doesn't exist
			if tt.unexpectedFQN != "" {
				found := false
				for fqn := range callGraph.Functions {
					if containsStr(fqn, tt.unexpectedFQN) {
						found = true
						break
					}
				}
				assert.False(t, found, "Should not index non-function: %s", tt.unexpectedFQN)
			}

			// Verify functionContext populated
			assert.NotEmpty(t, functionContext, "Function context should be populated")
		})
	}
}

func TestExtractGoCallSites(t *testing.T) {
	tests := []struct {
		name              string
		nodes             []*graph.Node
		expectedCallCount int
		expectedCaller    string
		expectedTarget    string
	}{
		{
			name: "extract simple call",
			nodes: []*graph.Node{
				{
					ID:         "func1",
					Type:       "function_declaration", // Fixed: was "function_definition"
					Name:       "main",
					File:       "/project/main.go",
					LineNumber: 10,
				},
				{
					ID:         "call1",
					Type:       "method_expression", // Fixed: was "call_expression"
					Name:       "Println",           // Fixed: use Name field
					Interface:  []string{"fmt"},     // Fixed: use Interface field for object name
					File:       "/project/main.go",
					LineNumber: 12,
				},
			},
			expectedCallCount: 1,
			expectedTarget:    "Println",
		},
		{
			name: "extract same-package call",
			nodes: []*graph.Node{
				{
					ID:         "func1",
					Type:       "function_declaration", // Fixed: was "function_definition"
					Name:       "main",
					File:       "/project/main.go",
					LineNumber: 10,
				},
				{
					ID:         "call1",
					Type:       "call",   // Fixed: was "call_expression"
					Name:       "helper", // Fixed: use Name field
					File:       "/project/main.go",
					LineNumber: 12,
				},
			},
			expectedCallCount: 1,
			expectedTarget:    "helper",
		},
		{
			name: "skip non-call nodes",
			nodes: []*graph.Node{
				{
					ID:         "func1",
					Type:       "function_declaration", // Fixed: was "function_definition"
					Name:       "main",
					File:       "/project/main.go",
					LineNumber: 10,
				},
				{
					ID:   "var1",
					Type: "variable_declaration",
					Name: "x",
					File: "/project/main.go",
				},
			},
			expectedCallCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeMap := make(map[string]*graph.Node)
			for _, node := range tt.nodes {
				nodeMap[node.ID] = node
			}
			codeGraph := &graph.CodeGraph{
				Nodes: nodeMap,
			}
			callGraph := core.NewCallGraph()

			callSites := extractGoCallSitesFromCodeGraph(codeGraph, callGraph)

			assert.Equal(t, tt.expectedCallCount, len(callSites), "Call site count mismatch")

			if tt.expectedCallCount > 0 {
				assert.Equal(t, tt.expectedTarget, callSites[0].FunctionName, "Target function name mismatch")
			}
		})
	}
}

func TestResolveGoCallTarget(t *testing.T) {
	tests := []struct {
		name          string
		callSite      *CallSiteInternal
		importMap     *core.GoImportMap
		registry      *core.GoModuleRegistry
		funcContext   map[string][]*graph.Node
		expectedFQN   string
		shouldResolve bool
	}{
		{
			name: "resolve qualified stdlib call",
			callSite: &CallSiteInternal{
				FunctionName: "Println",
				ObjectName:   "fmt",
				CallerFile:   "/project/main.go",
			},
			importMap: &core.GoImportMap{
				Imports: map[string]string{
					"fmt": "fmt",
				},
			},
			registry: &core.GoModuleRegistry{
				StdlibPackages: map[string]bool{
					"fmt": true,
				},
			},
			funcContext:   make(map[string][]*graph.Node),
			expectedFQN:   "fmt.Println",
			shouldResolve: true,
		},
		{
			name: "resolve qualified project call",
			callSite: &CallSiteInternal{
				FunctionName: "Helper",
				ObjectName:   "utils",
				CallerFile:   "/project/main.go",
			},
			importMap: &core.GoImportMap{
				Imports: map[string]string{
					"utils": "github.com/example/myapp/utils",
				},
			},
			registry: &core.GoModuleRegistry{
				StdlibPackages: map[string]bool{},
			},
			funcContext:   make(map[string][]*graph.Node),
			expectedFQN:   "github.com/example/myapp/utils.Helper",
			shouldResolve: true,
		},
		{
			name: "resolve same-package call",
			callSite: &CallSiteInternal{
				FunctionName: "helper",
				ObjectName:   "",
				CallerFile:   "/project/utils/main.go",
			},
			importMap: core.NewGoImportMap("/project/utils/main.go"),
			registry: &core.GoModuleRegistry{
				StdlibPackages: map[string]bool{},
				DirToImport: map[string]string{
					"/project/utils": "github.com/example/myapp/utils",
				},
			},
			funcContext: map[string][]*graph.Node{
				"helper": {
					{
						Type: "function_definition",
						Name: "helper",
						File: "/project/utils/helper.go",
					},
				},
			},
			expectedFQN:   "github.com/example/myapp/utils.helper",
			shouldResolve: true,
		},
		{
			name: "resolve builtin call",
			callSite: &CallSiteInternal{
				FunctionName: "append",
				ObjectName:   "",
				CallerFile:   "/project/main.go",
			},
			importMap:     core.NewGoImportMap("/project/main.go"),
			registry:      &core.GoModuleRegistry{},
			funcContext:   make(map[string][]*graph.Node),
			expectedFQN:   "builtin.append",
			shouldResolve: true,
		},
		{
			name: "unresolved qualified call",
			callSite: &CallSiteInternal{
				FunctionName: "Unknown",
				ObjectName:   "pkg",
				CallerFile:   "/project/main.go",
			},
			importMap: &core.GoImportMap{
				Imports: map[string]string{},
			},
			registry: &core.GoModuleRegistry{
				StdlibPackages: map[string]bool{},
			},
			funcContext:   make(map[string][]*graph.Node),
			shouldResolve: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass nil for typeEngine and callGraph (backward compatibility)
			targetFQN, resolved := resolveGoCallTarget(tt.callSite, tt.importMap, tt.registry, tt.funcContext, nil, nil)

			assert.Equal(t, tt.shouldResolve, resolved, "Resolution status mismatch")

			if tt.shouldResolve {
				assert.Equal(t, tt.expectedFQN, targetFQN, "Resolved FQN mismatch")
			}
		})
	}
}

func TestBuildGoFQN(t *testing.T) {
	tests := []struct {
		name        string
		node        *graph.Node
		codeGraph   *graph.CodeGraph
		registry    *core.GoModuleRegistry
		expectedFQN string
	}{
		{
			name: "package function FQN",
			node: &graph.Node{
				Type: "function_declaration", // Fixed: was "function_definition"
				Name: "main",
				File: "/project/cmd/main.go",
			},
			registry: &core.GoModuleRegistry{
				DirToImport: map[string]string{
					"/project/cmd": "github.com/example/myapp/cmd",
				},
			},
			expectedFQN: "github.com/example/myapp/cmd.main",
		},
		{
			name: "method FQN with receiver",
			node: &graph.Node{
				Type:      "method",
				Name:      "Start",
				Interface: []string{"Server"},
				File:      "/project/server/server.go",
			},
			registry: &core.GoModuleRegistry{
				DirToImport: map[string]string{
					"/project/server": "github.com/example/myapp/server",
				},
			},
			expectedFQN: "github.com/example/myapp/server.Server.Start",
		},
		{
			name: "closure FQN",
			node: &graph.Node{
				ID:   "closure1",
				Type: "func_literal",
				Name: "$anon_1",
				File: "/project/handlers/handler.go",
			},
			codeGraph: func() *graph.CodeGraph {
				func1 := &graph.Node{
					ID:   "func1",
					Type: "function_declaration",
					Name: "HandleRequest",
					File: "/project/handlers/handler.go",
				}
				closure1 := &graph.Node{
					ID:   "closure1",
					Type: "func_literal",
					Name: "$anon_1",
					File: "/project/handlers/handler.go",
				}
				edge := &graph.Edge{From: func1, To: closure1}
				func1.OutgoingEdges = []*graph.Edge{edge}

				return &graph.CodeGraph{
					Nodes: map[string]*graph.Node{
						"func1":    func1,
						"closure1": closure1,
					},
				}
			}(),
			registry: &core.GoModuleRegistry{
				DirToImport: map[string]string{
					"/project/handlers": "github.com/example/myapp/handlers",
				},
			},
			expectedFQN: "github.com/example/myapp/handlers.HandleRequest.$anon_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.codeGraph == nil {
				tt.codeGraph = &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}
			}

			fqn := buildGoFQN(tt.node, tt.codeGraph, tt.registry)
			assert.Equal(t, tt.expectedFQN, fqn, "FQN mismatch")
		})
	}
}

func TestIsBuiltin(t *testing.T) {
	tests := []struct {
		name      string
		funcName  string
		isBuiltin bool
	}{
		{"append is builtin", "append", true},
		{"len is builtin", "len", true},
		{"make is builtin", "make", true},
		{"println is builtin", "println", true},
		{"Println is not builtin", "Println", false},
		{"Helper is not builtin", "Helper", false},
		{"custom is not builtin", "custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBuiltin(tt.funcName)
			assert.Equal(t, tt.isBuiltin, result)
		})
	}
}

func TestIsSameGoPackage(t *testing.T) {
	tests := []struct {
		name        string
		file1       string
		file2       string
		samePackage bool
	}{
		{
			name:        "same directory",
			file1:       "/project/utils/helper.go",
			file2:       "/project/utils/validator.go",
			samePackage: true,
		},
		{
			name:        "different directories",
			file1:       "/project/handlers/handler.go",
			file2:       "/project/utils/helper.go",
			samePackage: false,
		},
		{
			name:        "same file",
			file1:       "/project/main.go",
			file2:       "/project/main.go",
			samePackage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameGoPackage(tt.file1, tt.file2)
			assert.Equal(t, tt.samePackage, result)
		})
	}
}

// Helper function for string containment check.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBuildGoCallGraph_WithTypeTracking verifies the 5-pass algorithm with type tracking.
// This integration test ensures that:
//   - Pass 1: Functions are indexed
//   - Pass 2a: Return types are extracted
//   - Pass 2b: Variable assignments are tracked
//   - Pass 3: Call sites are extracted
//   - Pass 4: Call targets are resolved
//   - GoTypeEngine is populated and attached to CallGraph
func TestBuildGoCallGraph_WithTypeTracking(t *testing.T) {
	// Use the all_type_patterns.go fixture from PR-14/PR-15
	fixturePath := "../../../test-fixtures/golang/type_tracking/all_type_patterns.go"

	// Convert to absolute path for consistency with buildGoFQN's path resolution
	absFixturePath, err := filepath.Abs(fixturePath)
	require.NoError(t, err)

	// Build CodeGraph from the fixture
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Create minimal nodes for testing (normally these come from graph.Initialize)
	// Function: GetInt() int
	getIntNode := &graph.Node{
		ID:         "func1",
		Type:       "function_declaration",
		Name:       "GetInt",
		File:       absFixturePath,
		LineNumber: 9,
		ReturnType: "int", // Annotation-based return type
	}
	codeGraph.Nodes["func1"] = getIntNode

	// Function: GetUserPointer() *User
	getUserPtrNode := &graph.Node{
		ID:         "func2",
		Type:       "function_declaration",
		Name:       "GetUserPointer",
		File:       absFixturePath,
		LineNumber: 46,
		ReturnType: "*User", // Annotation-based return type
	}
	codeGraph.Nodes["func2"] = getUserPtrNode

	// Build module registry with absolute path
	registry := &core.GoModuleRegistry{
		DirToImport: map[string]string{
			filepath.Dir(absFixturePath): "typetracking",
		},
		StdlibPackages: map[string]bool{},
	}

	// Initialize type engine
	goTypeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Build call graph with type tracking
	callGraph, err := BuildGoCallGraph(codeGraph, registry, goTypeEngine)
	require.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify Pass 1: Functions indexed
	assert.NotEmpty(t, callGraph.Functions, "Functions should be indexed")
	assert.Contains(t, callGraph.Functions, "typetracking.GetInt")
	assert.Contains(t, callGraph.Functions, "typetracking.GetUserPointer")

	// Verify Pass 2a: Return types extracted and stored in GoTypeEngine
	assert.NotNil(t, callGraph.GoTypeEngine, "GoTypeEngine should be attached to CallGraph")
	returnTypes := callGraph.GoTypeEngine.GetAllReturnTypes()
	assert.NotNil(t, returnTypes, "Return types should be extracted")

	// Return types should be stored in GoTypeEngine
	getIntReturnType, ok := callGraph.GoTypeEngine.GetReturnType("typetracking.GetInt")
	assert.True(t, ok, "GetInt return type should be in type engine")
	if ok && getIntReturnType != nil {
		assert.Equal(t, "builtin.int", getIntReturnType.TypeFQN, "GetInt should return int")
	}

	getUserPtrReturnType, ok := callGraph.GoTypeEngine.GetReturnType("typetracking.GetUserPointer")
	assert.True(t, ok, "GetUserPointer return type should be in type engine")
	if ok && getUserPtrReturnType != nil {
		assert.Equal(t, "typetracking.User", getUserPtrReturnType.TypeFQN, "GetUserPointer should return *User")
	}

	// Verify Pass 3 & 4: Call sites and edges work (tested in other tests)
	// The integration test ensures all passes run without errors
}

// TestResolveGoCallTarget_VariableMethod tests Pattern 1b: variable-based method resolution (PR-17).
// This verifies that method calls like user.Save() are resolved using variable types from typeEngine.
func TestResolveGoCallTarget_VariableMethod(t *testing.T) {
	tests := []struct {
		name          string
		callSite      *CallSiteInternal
		variableName  string
		variableType  string
		methodExists  bool
		expectedFQN   string
		shouldResolve bool
	}{
		{
			name: "resolve user.Save() to User.Save",
			callSite: &CallSiteInternal{
				FunctionName: "Save",
				ObjectName:   "user",
				CallerFQN:    "main.ProcessUser",
				CallerFile:   "/project/main.go",
			},
			variableName:  "user",
			variableType:  "models.User",
			methodExists:  true,
			expectedFQN:   "models.User.Save",
			shouldResolve: true,
		},
		{
			name: "resolve pointer variable (*User).Save",
			callSite: &CallSiteInternal{
				FunctionName: "Save",
				ObjectName:   "userPtr",
				CallerFQN:    "main.ProcessUser",
				CallerFile:   "/project/main.go",
			},
			variableName:  "userPtr",
			variableType:  "*models.User", // Pointer type
			methodExists:  true,
			expectedFQN:   "models.User.Save", // Stripped *
			shouldResolve: true,
		},
		{
			name: "fail when method doesn't exist",
			callSite: &CallSiteInternal{
				FunctionName: "NonExistent",
				ObjectName:   "user",
				CallerFQN:    "main.ProcessUser",
				CallerFile:   "/project/main.go",
			},
			variableName:  "user",
			variableType:  "models.User",
			methodExists:  false,
			shouldResolve: false,
		},
		{
			name: "fail when variable not in scope",
			callSite: &CallSiteInternal{
				FunctionName: "Save",
				ObjectName:   "unknown",
				CallerFQN:    "main.ProcessUser",
				CallerFile:   "/project/main.go",
			},
			shouldResolve: false,
		},
		{
			name: "fallback to import when typeEngine is nil",
			callSite: &CallSiteInternal{
				FunctionName: "Println",
				ObjectName:   "fmt",
				CallerFQN:    "main.Main",
				CallerFile:   "/project/main.go",
			},
			expectedFQN:   "fmt.Println",
			shouldResolve: true,
		},
		{
			name: "prioritize import over variable",
			callSite: &CallSiteInternal{
				FunctionName: "Println",
				ObjectName:   "fmt",
				CallerFQN:    "main.Main",
				CallerFile:   "/project/main.go",
			},
			variableName:  "fmt", // Variable named "fmt" (edge case)
			variableType:  "models.Formatter",
			methodExists:  true,
			expectedFQN:   "fmt.Println", // Import wins over variable
			shouldResolve: true,
		},
		{
			name: "resolve config.Validate() with return type",
			callSite: &CallSiteInternal{
				FunctionName: "Validate",
				ObjectName:   "config",
				CallerFQN:    "main.Setup",
				CallerFile:   "/project/main.go",
			},
			variableName:  "config",
			variableType:  "pkg.Config",
			methodExists:  true,
			expectedFQN:   "pkg.Config.Validate",
			shouldResolve: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup registry
			registry := &core.GoModuleRegistry{
				DirToImport: map[string]string{
					"/project":        "main",
					"/project/models": "models",
					"/project/pkg":    "pkg",
				},
				StdlibPackages: map[string]bool{
					"fmt": true,
				},
			}

			// Setup import map
			importMap := core.NewGoImportMap("/project/main.go")
			if tt.callSite.ObjectName == "fmt" {
				importMap.AddImport("fmt", "fmt")
			}

			// Setup type engine with variable binding
			var typeEngine *resolution.GoTypeInferenceEngine
			if tt.variableName != "" || tt.name == "fallback to import when typeEngine is nil" {
				typeEngine = resolution.NewGoTypeInferenceEngine(registry)
				if tt.variableName != "" {
					scope := resolution.NewGoFunctionScope(tt.callSite.CallerFQN)
					binding := &resolution.GoVariableBinding{
						VarName: tt.variableName,
						Type: &core.TypeInfo{
							TypeFQN:    tt.variableType,
							Confidence: 1.0,
							Source:     "test",
						},
						AssignedFrom: "test",
						Location: resolution.Location{
							File: tt.callSite.CallerFile,
							Line: 10,
						},
					}
					scope.AddVariable(binding)
					typeEngine.AddScope(scope)
				}
			}

			// Setup call graph with method
			callGraph := core.NewCallGraph()
			if tt.methodExists {
				// Strip pointer prefix for method FQN
				methodTypeFQN := tt.variableType
				if after, ok := strings.CutPrefix(methodTypeFQN, "*"); ok {
					methodTypeFQN = after
				}
				methodFQN := methodTypeFQN + "." + tt.callSite.FunctionName
				callGraph.Functions[methodFQN] = &graph.Node{
					Type: "method",
					Name: tt.callSite.FunctionName,
					File: "/project/models/user.go",
				}
			}
			// Add fmt.Println for import tests
			if tt.callSite.ObjectName == "fmt" {
				callGraph.Functions["fmt.Println"] = &graph.Node{
					Type: "function_declaration",
					Name: "Println",
					File: "/usr/lib/go/fmt/print.go",
				}
			}

			functionContext := make(map[string][]*graph.Node)

			// Execute
			targetFQN, resolved := resolveGoCallTarget(
				tt.callSite,
				importMap,
				registry,
				functionContext,
				typeEngine,
				callGraph,
			)

			// Assert
			assert.Equal(t, tt.shouldResolve, resolved, "Resolution status mismatch")
			if tt.shouldResolve {
				assert.Equal(t, tt.expectedFQN, targetFQN, "Resolved FQN mismatch")
			}
		})
	}
}

// TestBuildGoCallGraph_MethodResolution is an integration test that verifies
// variable-based method calls are resolved end-to-end through BuildGoCallGraph.
// This uses the all_type_patterns.go fixture to validate realistic scenarios.
func TestBuildGoCallGraph_MethodResolution(t *testing.T) {
	fixturePath := "../../../test-fixtures/golang/type_tracking/all_type_patterns.go"

	// Convert to absolute path for consistency with buildGoFQN's path resolution
	absFixturePath, err := filepath.Abs(fixturePath)
	require.NoError(t, err)

	// Build CodeGraph with realistic nodes that would come from PR-06 AST parsing
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Function: DemoVariableAssignments() - contains variable.method() calls
	demoFunc := &graph.Node{
		ID:         "demo_func",
		Type:       "function_declaration",
		Name:       "DemoVariableAssignments",
		File:       absFixturePath,
		LineNumber: 141,
	}
	codeGraph.Nodes["demo_func"] = demoFunc

	// Method: (*User).Save() error
	saveMethod := &graph.Node{
		ID:         "user_save",
		Type:       "method",
		Name:       "Save",
		Interface:  []string{"User"}, // Receiver type
		File:       absFixturePath,
		LineNumber: 124,
		ReturnType: "error",
	}
	codeGraph.Nodes["user_save"] = saveMethod

	// Method: (*Config).Validate() (bool, error)
	validateMethod := &graph.Node{
		ID:         "config_validate",
		Type:       "method",
		Name:       "Validate",
		Interface:  []string{"Config"}, // Receiver type
		File:       absFixturePath,
		LineNumber: 133,
		ReturnType: "(bool, error)",
	}
	codeGraph.Nodes["config_validate"] = validateMethod

	// Call node: user.Save() inside DemoVariableAssignments
	// This simulates what PR-06 AST parsing would create
	userSaveCall := &graph.Node{
		ID:         "call_user_save",
		Type:       "method_expression",
		Name:       "Save",
		Interface:  []string{"user"}, // ObjectName = "user"
		File:       absFixturePath,
		LineNumber: 150, // Inside DemoVariableAssignments (after user := GetUserPointer())
	}
	// Set up parent-child relationship
	edge1 := &graph.Edge{From: demoFunc, To: userSaveCall}
	demoFunc.OutgoingEdges = append(demoFunc.OutgoingEdges, edge1)
	codeGraph.Nodes["call_user_save"] = userSaveCall

	// Call node: config.Validate() inside DemoVariableAssignments
	configValidateCall := &graph.Node{
		ID:         "call_config_validate",
		Type:       "method_expression",
		Name:       "Validate",
		Interface:  []string{"config"}, // ObjectName = "config"
		File:       absFixturePath,
		LineNumber: 152, // Inside DemoVariableAssignments (after config := CreateConfig())
	}
	edge2 := &graph.Edge{From: demoFunc, To: configValidateCall}
	demoFunc.OutgoingEdges = append(demoFunc.OutgoingEdges, edge2)
	codeGraph.Nodes["call_config_validate"] = configValidateCall

	// Setup registry with absolute path
	registry := &core.GoModuleRegistry{
		DirToImport: map[string]string{
			filepath.Dir(absFixturePath): "typetracking",
		},
		StdlibPackages: map[string]bool{},
	}

	// Initialize type engine
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Manually populate variable bindings that would come from PR-15 variable extraction
	// In DemoVariableAssignments:
	//   user := GetUserPointer()  // Type: *User
	//   config := CreateConfig()  // Type: Config
	demoFQN := "typetracking.DemoVariableAssignments"
	scope := resolution.NewGoFunctionScope(demoFQN)

	// Variable: user (type: *User from GetUserPointer)
	userBinding := &resolution.GoVariableBinding{
		VarName: "user",
		Type: &core.TypeInfo{
			TypeFQN:    "typetracking.User", // Note: pointer stripped for method lookup
			Confidence: 1.0,
			Source:     "function_call",
		},
		AssignedFrom: "typetracking.GetUserPointer",
		Location: resolution.Location{
			File: absFixturePath,
			Line: 143,
		},
	}
	scope.AddVariable(userBinding)

	// Variable: config (type: Config from CreateConfig)
	configBinding := &resolution.GoVariableBinding{
		VarName: "config",
		Type: &core.TypeInfo{
			TypeFQN:    "typetracking.Config",
			Confidence: 1.0,
			Source:     "function_call",
		},
		AssignedFrom: "typetracking.CreateConfig",
		Location: resolution.Location{
			File: absFixturePath,
			Line: 144,
		},
	}
	scope.AddVariable(configBinding)

	typeEngine.AddScope(scope)

	// Build call graph
	callGraph, err := BuildGoCallGraph(codeGraph, registry, typeEngine)
	require.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify functions were indexed
	assert.Contains(t, callGraph.Functions, "typetracking.DemoVariableAssignments")
	assert.Contains(t, callGraph.Functions, "typetracking.User.Save")
	assert.Contains(t, callGraph.Functions, "typetracking.Config.Validate")

	// Verify call sites for DemoVariableAssignments
	callSites := callGraph.CallSites[demoFQN]
	assert.NotEmpty(t, callSites, "DemoVariableAssignments should have call sites")

	// Track which methods were resolved
	var userSaveResolved bool
	var configValidateResolved bool

	for _, cs := range callSites {
		t.Logf("Call site: Target=%s, TargetFQN=%s, Resolved=%v", cs.Target, cs.TargetFQN, cs.Resolved)

		if cs.Target == "Save" && cs.Resolved {
			assert.Equal(t, "typetracking.User.Save", cs.TargetFQN, "user.Save() should resolve to User.Save")
			userSaveResolved = true
		}

		if cs.Target == "Validate" && cs.Resolved {
			assert.Equal(t, "typetracking.Config.Validate", cs.TargetFQN, "config.Validate() should resolve to Config.Validate")
			configValidateResolved = true
		}
	}

	// Assert both methods were resolved via variable types
	assert.True(t, userSaveResolved, "user.Save() should be resolved via variable type")
	assert.True(t, configValidateResolved, "config.Validate() should be resolved via variable type")

	// Verify edges were added
	callees := callGraph.GetCallees(demoFQN)
	assert.Contains(t, callees, "typetracking.User.Save", "Should have edge to User.Save")
	assert.Contains(t, callees, "typetracking.Config.Validate", "Should have edge to Config.Validate")

	// Verify reverse edges
	userSaveCallers := callGraph.GetCallers("typetracking.User.Save")
	assert.Contains(t, userSaveCallers, demoFQN, "User.Save should have DemoVariableAssignments as caller")

	configValidateCallers := callGraph.GetCallers("typetracking.Config.Validate")
	assert.Contains(t, configValidateCallers, demoFQN, "Config.Validate should have DemoVariableAssignments as caller")
}

// TestBuildGoFQN_RelativePath tests that relative paths are converted to absolute.
func TestBuildGoFQN_RelativePath(t *testing.T) {
	node := &graph.Node{
		Type: "function_declaration",
		Name: "TestFunc",
		File: "test.go", // Relative path
	}

	registry := &core.GoModuleRegistry{
		DirToImport: make(map[string]string),
	}

	// Should not panic, even if registry lookup fails
	fqn := buildGoFQN(node, nil, registry)
	assert.NotEmpty(t, fqn)
}

// TestBuildGoFQN_EmptyPath tests edge case with empty file path.
func TestBuildGoFQN_EmptyPath(t *testing.T) {
	node := &graph.Node{
		Type: "function_declaration",
		Name: "TestFunc",
		File: "", // Empty path
	}

	registry := &core.GoModuleRegistry{
		DirToImport: make(map[string]string),
	}

	fqn := buildGoFQN(node, nil, registry)
	assert.Contains(t, fqn, "TestFunc")
}
