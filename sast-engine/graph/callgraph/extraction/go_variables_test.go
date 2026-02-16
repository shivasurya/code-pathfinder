package extraction

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				Imports: make(map[string]string),
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
			scope := typeEngine.GetScope("test.Test")

			assert.NotNil(t, scope, "Expected scope for test.Test")

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
		Imports: make(map[string]string),
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

	scope := typeEngine.GetScope("test.Test")


	if scope == nil {


		t.Fatal("Expected scope but got nil")


	}

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
		Imports: make(map[string]string),
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

	scope := typeEngine.GetScope("test.Test")


	if scope == nil {


		t.Fatal("Expected scope but got nil")


	}

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
			expectedType: "test.User", // Resolved to full FQN with package
		},
		{
			name: "pointer to struct literal",
			code: `package main
type Config struct {}
func Test() {
	c := &Config{}
}`,
			varName:      "c",
			expectedType: "test.Config", // Resolved to full FQN with package
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
				Imports: make(map[string]string),
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

			scope := typeEngine.GetScope("test.Test")


			if scope == nil {


				t.Fatal("Expected scope but got nil")


			}

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
		Imports: make(map[string]string),
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

	scope := typeEngine.GetScope("test.Test")


	if scope == nil {


		t.Fatal("Expected scope but got nil")


	}

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
		Imports: make(map[string]string),
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

	scope := typeEngine.GetScope("test.Test")


	if scope == nil {


		t.Fatal("Expected scope but got nil")


	}

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
		Imports: make(map[string]string),
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
	scope := typeEngine.GetScope("test.User.Test")

	assert.NotNil(t, scope, "Expected scope for method test.User.Test")

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
		Imports: make(map[string]string),
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
		Imports: make(map[string]string),
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

// TestExtractGoVariables_Integration tests with real fixture file.
func TestExtractGoVariables_Integration(t *testing.T) {
	fixturePath := "../../../test-fixtures/golang/type_tracking/all_type_patterns.go"

	// Convert to absolute path for consistency with ExtractGoVariableAssignments
	absFixturePath, err := filepath.Abs(fixturePath)
	require.NoError(t, err)

	// Read fixture
	sourceCode, err := os.ReadFile(absFixturePath)
	if err != nil {
		t.Skip("Fixture file not found, skipping integration test")
		return
	}

	// Setup with absolute path
	registry := &core.GoModuleRegistry{
		ModulePath: "github.com/test/typetracking",
		DirToImport: map[string]string{
			filepath.Dir(absFixturePath): "github.com/test/typetracking",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Pre-populate return types (simulating Pass 2a)
	returnTypes := map[string]string{
		"github.com/test/typetracking.GetInt":         "builtin.int",
		"github.com/test/typetracking.GetString":      "builtin.string",
		"github.com/test/typetracking.GetBool":        "builtin.bool",
		"github.com/test/typetracking.GetUserPointer": "User",
		"github.com/test/typetracking.CreateConfig":   "Config",
		"github.com/test/typetracking.LoadConfig":     "builtin.string",
		"github.com/test/typetracking.GetTwoInts":     "builtin.int",
	}

	for fqn, typeFQN := range returnTypes {
		typeEngine.AddReturnType(fqn, &core.TypeInfo{
			TypeFQN:    typeFQN,
			Confidence: 1.0,
			Source:     "declaration",
		})
	}

	importMap := &core.GoImportMap{
		Imports: make(map[string]string),
	}

	// Execute
	err = ExtractGoVariableAssignments(
		absFixturePath,
		sourceCode,
		typeEngine,
		registry,
		importMap,
	)

	// Verify
	assert.NoError(t, err)

	// Check DemoVariableAssignments function scope
	scope := typeEngine.GetScope("github.com/test/typetracking.DemoVariableAssignments")

	assert.NotNil(t, scope, "Expected scope for DemoVariableAssignments")

	// Test function call assignments
	expectedFunctionCallTypes := map[string]string{
		"user":    "User",
		"config":  "Config",
		"name":    "builtin.string",
		"intVal":  "builtin.int",
		"boolVal": "builtin.bool",
	}

	for varName, expectedType := range expectedFunctionCallTypes {
		bindings, ok := scope.Variables[varName]
		assert.True(t, ok, "Expected binding for %s", varName)
		if ok && len(bindings) > 0 {
			assert.Equal(t, expectedType, bindings[0].Type.TypeFQN, "Wrong type for %s", varName)
		}
	}

	// Test literal assignments
	expectedLiteralTypes := map[string]string{
		"str":       "builtin.string",
		"num":       "builtin.int",
		"floatNum":  "builtin.float64",
		"flag":      "builtin.bool",
		"falsyFlag": "builtin.bool",
	}

	for varName, expectedType := range expectedLiteralTypes {
		bindings, ok := scope.Variables[varName]
		assert.True(t, ok, "Expected binding for %s", varName)
		if ok && len(bindings) > 0 {
			assert.Equal(t, expectedType, bindings[0].Type.TypeFQN, "Wrong type for %s", varName)
		}
	}

	// Test variable reference
	user2Bindings, ok := scope.Variables["user2"]
	assert.True(t, ok, "Expected binding for user2")
	if ok && len(user2Bindings) > 0 {
		assert.Equal(t, "User", user2Bindings[0].Type.TypeFQN)
	}

	// Check DemoComplexAssignments scope
	complexScope := typeEngine.GetScope("github.com/test/typetracking.DemoComplexAssignments")

	assert.NotNil(t, complexScope, "Expected scope for DemoComplexAssignments")

	// Test multi-assignment
	xBindings, ok := complexScope.Variables["x"]
	assert.True(t, ok, "Expected binding for x")
	if ok && len(xBindings) > 0 {
		assert.Equal(t, "builtin.int", xBindings[0].Type.TypeFQN)
	}
}
