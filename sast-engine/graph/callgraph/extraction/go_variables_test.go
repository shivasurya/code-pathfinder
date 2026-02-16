package extraction

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
)

// TestExtractGoVariables_Literals tests literal value assignments.
func TestExtractGoVariables_Literals(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		expectedType string
	}{
		{
			name: "string literal",
			code: `package main
func Test() {
	name := "Alice"
}`,
			expectedType: "builtin.string",
		},
		{
			name: "int literal",
			code: `package main
func Test() {
	count := 42
}`,
			expectedType: "builtin.int",
		},
		{
			name: "bool literal true",
			code: `package main
func Test() {
	flag := true
}`,
			expectedType: "builtin.bool",
		},
		{
			name: "bool literal false",
			code: `package main
func Test() {
	flag := false
}`,
			expectedType: "builtin.bool",
		},
		{
			name: "float literal",
			code: `package main
func Test() {
	pi := 3.14
}`,
			expectedType: "builtin.float64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			registry := &core.GoModuleRegistry{
				ModulePath: "test",
				DirToImport: map[string]string{
					"/test": "test",
				},
			}
			typeEngine := resolution.NewGoTypeInferenceEngine(registry)
			importMap := &core.GoImportMap{
				Aliases: make(map[string]string),
			}

			// Execute
			err := ExtractGoVariableAssignments(
				"/test/main.go",
				[]byte(tt.code),
				typeEngine,
				registry,
				importMap,
			)

			// Verify
			assert.NoError(t, err)

			// Check variable binding was created
			scope, ok := typeEngine.GetScope("test.Test")
			assert.True(t, ok, "Expected scope for test.Test")

			// Get bindings (name depends on test case, but all use same var names)
			var varName string
			switch tt.name {
			case "string literal":
				varName = "name"
			case "int literal":
				varName = "count"
			case "bool literal true", "bool literal false":
				varName = "flag"
			case "float literal":
				varName = "pi"
			}

			bindings, ok := scope.Variables[varName]
			assert.True(t, ok, "Expected binding for variable %s", varName)
			assert.Len(t, bindings, 1)
			assert.Equal(t, tt.expectedType, bindings[0].Type.TypeFQN)
			assert.Equal(t, float32(1.0), bindings[0].Type.Confidence)
		})
	}
}

// TestExtractGoVariables_FunctionCall tests function call assignments.
func TestExtractGoVariables_FunctionCall(t *testing.T) {
	code := `package main
func GetUser() *User {
	return &User{}
}
func Test() {
	user := GetUser()
}`

	// Setup
	registry := &core.GoModuleRegistry{
		ModulePath: "test",
		DirToImport: map[string]string{
			"/test": "test",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Pre-populate return type for GetUser (simulating Pass 2a)
	typeEngine.AddReturnType("test.GetUser", &core.TypeInfo{
		TypeFQN:    "User",
		Confidence: 1.0,
		Source:     "declaration",
	})

	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/test/main.go",
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify
	assert.NoError(t, err)

	scope, ok := typeEngine.GetScope("test.Test")
	assert.True(t, ok)

	bindings, ok := scope.Variables["user"]
	assert.True(t, ok)
	assert.Len(t, bindings, 1)
	assert.Equal(t, "User", bindings[0].Type.TypeFQN)
}

// TestExtractGoVariables_VariableRef tests variable reference assignments.
func TestExtractGoVariables_VariableRef(t *testing.T) {
	code := `package main
func Test() {
	name := "Alice"
	name2 := name
}`

	// Setup
	registry := &core.GoModuleRegistry{
		ModulePath: "test",
		DirToImport: map[string]string{
			"/test": "test",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)
	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/test/main.go",
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify
	assert.NoError(t, err)

	scope, ok := typeEngine.GetScope("test.Test")
	assert.True(t, ok)

	// Check name2 has same type as name
	bindings, ok := scope.Variables["name2"]
	assert.True(t, ok)
	assert.Len(t, bindings, 1)
	assert.Equal(t, "builtin.string", bindings[0].Type.TypeFQN)
}

// TestExtractGoVariables_StructLiteral tests struct literal assignments.
func TestExtractGoVariables_StructLiteral(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		varName      string
		expectedType string
	}{
		{
			name: "struct literal",
			code: `package main
type User struct {}
func Test() {
	u := User{}
}`,
			varName:      "u",
			expectedType: "User",
		},
		{
			name: "pointer to struct literal",
			code: `package main
type Config struct {}
func Test() {
	c := &Config{}
}`,
			varName:      "c",
			expectedType: "Config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			registry := &core.GoModuleRegistry{
				ModulePath: "test",
				DirToImport: map[string]string{
					"/test": "test",
				},
			}
			typeEngine := resolution.NewGoTypeInferenceEngine(registry)
			importMap := &core.GoImportMap{
				Aliases: make(map[string]string),
			}

			// Execute
			err := ExtractGoVariableAssignments(
				"/test/main.go",
				[]byte(tt.code),
				typeEngine,
				registry,
				importMap,
			)

			// Verify
			assert.NoError(t, err)

			scope, ok := typeEngine.GetScope("test.Test")
			assert.True(t, ok)

			bindings, ok := scope.Variables[tt.varName]
			assert.True(t, ok)
			assert.Len(t, bindings, 1)
			assert.Equal(t, tt.expectedType, bindings[0].Type.TypeFQN)
		})
	}
}

// TestExtractGoVariables_MultiAssignment tests multi-variable assignments.
func TestExtractGoVariables_MultiAssignment(t *testing.T) {
	code := `package main
func GetTwo() (int, error) {
	return 42, nil
}
func Test() {
	x, err := GetTwo()
}`

	// Setup
	registry := &core.GoModuleRegistry{
		ModulePath: "test",
		DirToImport: map[string]string{
			"/test": "test",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Pre-populate return type (first return value)
	typeEngine.AddReturnType("test.GetTwo", &core.TypeInfo{
		TypeFQN:    "builtin.int",
		Confidence: 1.0,
		Source:     "declaration",
	})

	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/test/main.go",
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify
	assert.NoError(t, err)

	scope, ok := typeEngine.GetScope("test.Test")
	assert.True(t, ok)

	// Both variables should have the first return type
	xBindings, ok := scope.Variables["x"]
	assert.True(t, ok)
	assert.Len(t, xBindings, 1)
	assert.Equal(t, "builtin.int", xBindings[0].Type.TypeFQN)

	errBindings, ok := scope.Variables["err"]
	assert.True(t, ok)
	assert.Len(t, errBindings, 1)
	assert.Equal(t, "builtin.int", errBindings[0].Type.TypeFQN)
}

// TestExtractGoVariables_Reassignment tests variable reassignment tracking.
func TestExtractGoVariables_Reassignment(t *testing.T) {
	code := `package main
func Test() {
	x := 42
	x = 100
}`

	// Setup
	registry := &core.GoModuleRegistry{
		ModulePath: "test",
		DirToImport: map[string]string{
			"/test": "test",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)
	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/test/main.go",
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify
	assert.NoError(t, err)

	scope, ok := typeEngine.GetScope("test.Test")
	assert.True(t, ok)

	// Should have two bindings for x
	bindings, ok := scope.Variables["x"]
	assert.True(t, ok)
	assert.Len(t, bindings, 2, "Expected 2 bindings for reassigned variable")

	// Both should be int
	assert.Equal(t, "builtin.int", bindings[0].Type.TypeFQN)
	assert.Equal(t, "builtin.int", bindings[1].Type.TypeFQN)
}

// TestExtractGoVariables_MethodContext tests variable tracking in methods.
func TestExtractGoVariables_MethodContext(t *testing.T) {
	code := `package main
type User struct {}
func (u *User) Test() {
	name := "Alice"
}`

	// Setup
	registry := &core.GoModuleRegistry{
		ModulePath: "test",
		DirToImport: map[string]string{
			"/test": "test",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)
	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/test/main.go",
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify
	assert.NoError(t, err)

	// Check method FQN is correct
	scope, ok := typeEngine.GetScope("test.User.Test")
	assert.True(t, ok, "Expected scope for method test.User.Test")

	bindings, ok := scope.Variables["name"]
	assert.True(t, ok)
	assert.Len(t, bindings, 1)
	assert.Equal(t, "builtin.string", bindings[0].Type.TypeFQN)
}

// TestExtractGoVariables_EmptyFile tests handling of empty files.
func TestExtractGoVariables_EmptyFile(t *testing.T) {
	code := `package main`

	// Setup
	registry := &core.GoModuleRegistry{
		ModulePath: "test",
		DirToImport: map[string]string{
			"/test": "test",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)
	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/test/main.go",
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify - should not error
	assert.NoError(t, err)

	// No scopes should be created
	allScopes := typeEngine.GetAllScopes()
	assert.Len(t, allScopes, 0)
}

// TestExtractGoVariables_FileNotInRegistry tests handling of files not in registry.
func TestExtractGoVariables_FileNotInRegistry(t *testing.T) {
	code := `package main
func Test() {
	x := 42
}`

	// Setup with empty registry
	registry := &core.GoModuleRegistry{
		ModulePath:  "test",
		DirToImport: make(map[string]string), // Empty - file won't be found
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)
	importMap := &core.GoImportMap{
		Aliases: make(map[string]string),
	}

	// Execute
	err := ExtractGoVariableAssignments(
		"/other/main.go", // Not in registry
		[]byte(code),
		typeEngine,
		registry,
		importMap,
	)

	// Verify - should return nil without error
	assert.NoError(t, err)

	// No scopes should be created
	allScopes := typeEngine.GetAllScopes()
	assert.Len(t, allScopes, 0)
}
