package extraction

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
)

// TestExtractVariableAssignments_StringLiterals tests string literal type inference.
func TestExtractVariableAssignments_StringLiterals(t *testing.T) {
	sourceCode := []byte(`
def test_function():
    name = "Alice"
    greeting = 'Hello'
    multiline = """Multi
    line
    string"""
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.test_function")
	assert.NotNil(t, scope)

	// Check 'name' variable
	nameBinding := scope.Variables["name"]
	assert.NotNil(t, nameBinding)
	assert.Equal(t, "builtins.str", nameBinding.Type.TypeFQN)
	assert.Equal(t, float32(1.0), nameBinding.Type.Confidence)
	assert.Equal(t, "literal", nameBinding.Type.Source)

	// Check 'greeting' variable
	greetingBinding := scope.Variables["greeting"]
	assert.NotNil(t, greetingBinding)
	assert.Equal(t, "builtins.str", greetingBinding.Type.TypeFQN)

	// Check 'multiline' variable
	multilineBinding := scope.Variables["multiline"]
	assert.NotNil(t, multilineBinding)
	assert.Equal(t, "builtins.str", multilineBinding.Type.TypeFQN)
}

// TestExtractVariableAssignments_NumericLiterals tests numeric literal type inference.
func TestExtractVariableAssignments_NumericLiterals(t *testing.T) {
	sourceCode := []byte(`
def calculate():
    count = 42
    price = 19.99
    negative = -5
    scientific = 1.5e10
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.calculate")
	assert.NotNil(t, scope)

	// Check 'count' variable (int)
	countBinding := scope.Variables["count"]
	assert.NotNil(t, countBinding)
	assert.Equal(t, "builtins.int", countBinding.Type.TypeFQN)

	// Check 'price' variable (float)
	priceBinding := scope.Variables["price"]
	assert.NotNil(t, priceBinding)
	assert.Equal(t, "builtins.float", priceBinding.Type.TypeFQN)

	// Check 'negative' variable (int)
	negativeBinding := scope.Variables["negative"]
	assert.NotNil(t, negativeBinding)
	assert.Equal(t, "builtins.int", negativeBinding.Type.TypeFQN)

	// Check 'scientific' variable (float)
	scientificBinding := scope.Variables["scientific"]
	assert.NotNil(t, scientificBinding)
	assert.Equal(t, "builtins.float", scientificBinding.Type.TypeFQN)
}

// TestExtractVariableAssignments_CollectionLiterals tests collection literal inference.
func TestExtractVariableAssignments_CollectionLiterals(t *testing.T) {
	sourceCode := []byte(`
def process_data():
    items = [1, 2, 3]
    config = {"key": "value"}
    unique = {1, 2, 3}
    coords = (10, 20)
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.process_data")
	assert.NotNil(t, scope)

	// Check 'items' variable (list)
	itemsBinding := scope.Variables["items"]
	assert.NotNil(t, itemsBinding)
	assert.Equal(t, "builtins.list", itemsBinding.Type.TypeFQN)

	// Check 'config' variable (dict)
	configBinding := scope.Variables["config"]
	assert.NotNil(t, configBinding)
	assert.Equal(t, "builtins.dict", configBinding.Type.TypeFQN)

	// Check 'unique' variable (set)
	uniqueBinding := scope.Variables["unique"]
	assert.NotNil(t, uniqueBinding)
	assert.Equal(t, "builtins.set", uniqueBinding.Type.TypeFQN)

	// Check 'coords' variable (tuple)
	coordsBinding := scope.Variables["coords"]
	assert.NotNil(t, coordsBinding)
	assert.Equal(t, "builtins.tuple", coordsBinding.Type.TypeFQN)
}

// TestExtractVariableAssignments_BooleanAndNone tests boolean and None literals.
func TestExtractVariableAssignments_BooleanAndNone(t *testing.T) {
	sourceCode := []byte(`
def check_status():
    is_active = True
    is_deleted = False
    result = None
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.check_status")
	assert.NotNil(t, scope)

	// Check 'is_active' variable
	isActiveBinding := scope.Variables["is_active"]
	assert.NotNil(t, isActiveBinding)
	assert.Equal(t, "builtins.bool", isActiveBinding.Type.TypeFQN)

	// Check 'is_deleted' variable
	isDeletedBinding := scope.Variables["is_deleted"]
	assert.NotNil(t, isDeletedBinding)
	assert.Equal(t, "builtins.bool", isDeletedBinding.Type.TypeFQN)

	// Check 'result' variable
	resultBinding := scope.Variables["result"]
	assert.NotNil(t, resultBinding)
	assert.Equal(t, "builtins.NoneType", resultBinding.Type.TypeFQN)
}

// TestExtractVariableAssignments_MultipleVariables tests multiple variables in one function.
func TestExtractVariableAssignments_MultipleVariables(t *testing.T) {
	sourceCode := []byte(`
def process():
    name = "Alice"
    age = 30
    items = [1, 2, 3]
    is_valid = True
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.process")
	assert.NotNil(t, scope)
	assert.Equal(t, 4, len(scope.Variables))

	assert.NotNil(t, scope.Variables["name"])
	assert.NotNil(t, scope.Variables["age"])
	assert.NotNil(t, scope.Variables["items"])
	assert.NotNil(t, scope.Variables["is_valid"])
}

// TestExtractVariableAssignments_NestedFunctions tests nested function scopes.
func TestExtractVariableAssignments_NestedFunctions(t *testing.T) {
	sourceCode := []byte(`
def outer():
    x = "outer"

    def inner():
        y = "inner"
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify outer function scope
	outerScope := typeEngine.GetScope("test.outer")
	assert.NotNil(t, outerScope)
	assert.NotNil(t, outerScope.Variables["x"])
	assert.Equal(t, "builtins.str", outerScope.Variables["x"].Type.TypeFQN)

	// Verify inner function scope
	innerScope := typeEngine.GetScope("test.outer.inner")
	assert.NotNil(t, innerScope)
	assert.NotNil(t, innerScope.Variables["y"])
	assert.Equal(t, "builtins.str", innerScope.Variables["y"].Type.TypeFQN)
}

// TestExtractVariableAssignments_VariableReassignment tests variable reassignment.
func TestExtractVariableAssignments_VariableReassignment(t *testing.T) {
	sourceCode := []byte(`
def reassign():
    x = 10
    x = "string"
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify - should have the last assignment
	scope := typeEngine.GetScope("test.reassign")
	assert.NotNil(t, scope)

	xBinding := scope.Variables["x"]
	assert.NotNil(t, xBinding)
	// Last assignment wins (string)
	assert.Equal(t, "builtins.str", xBinding.Type.TypeFQN)
}

// TestExtractVariableAssignments_EmptyFile tests empty file handling.
func TestExtractVariableAssignments_EmptyFile(t *testing.T) {
	sourceCode := []byte("")

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments - should not error
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// No scopes should be created
	assert.Equal(t, 0, len(typeEngine.Scopes))
}

// TestExtractVariableAssignments_NoAssignments tests function with no assignments.
func TestExtractVariableAssignments_NoAssignments(t *testing.T) {
	sourceCode := []byte(`
def empty_function():
    pass
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Scope should exist but be empty
	scope := typeEngine.GetScope("test.empty_function")
	assert.NotNil(t, scope)
	assert.Equal(t, 0, len(scope.Variables))
}

// TestExtractVariableAssignments_LocationTracking tests source location tracking.
func TestExtractVariableAssignments_LocationTracking(t *testing.T) {
	sourceCode := []byte(`
def test():
    x = 10
    y = 20
`)

	// Setup
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, sourceCode, 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins)
	assert.NoError(t, err)

	// Verify locations
	scope := typeEngine.GetScope("test.test")
	assert.NotNil(t, scope)

	xBinding := scope.Variables["x"]
	assert.NotNil(t, xBinding)
	assert.Equal(t, filePath, xBinding.Location.File)
	assert.Equal(t, uint32(3), xBinding.Location.Line)

	yBinding := scope.Variables["y"]
	assert.NotNil(t, yBinding)
	assert.Equal(t, filePath, yBinding.Location.File)
	assert.Equal(t, uint32(4), yBinding.Location.Line)
}

// TestInferTypeFromExpression_DirectCalls tests type inference helper.
func TestInferTypeFromExpression(t *testing.T) {
	builtinRegistry := registry.NewBuiltinRegistry()

	tests := []struct {
		name         string
		code         string
		expectedType string
	}{
		{name: "string literal", code: `x = "test"`, expectedType: "builtins.str"},
		{name: "integer literal", code: `x = 42`, expectedType: "builtins.int"},
		{name: "float literal", code: `x = 3.14`, expectedType: "builtins.float"},
		{name: "list literal", code: `x = [1, 2, 3]`, expectedType: "builtins.list"},
		{name: "dict literal", code: `x = {}`, expectedType: "builtins.dict"},
		{name: "boolean True", code: `x = True`, expectedType: "builtins.bool"},
		{name: "boolean False", code: `x = False`, expectedType: "builtins.bool"},
		{name: "None", code: `x = None`, expectedType: "builtins.NoneType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.py")
			sourceCode := []byte("def test():\n    " + tt.code)
			err := os.WriteFile(filePath, sourceCode, 0644)
			assert.NoError(t, err)

			modRegistry, err := registry.BuildModuleRegistry(tmpDir)
			assert.NoError(t, err)

			typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
			typeEngine.Builtins = builtinRegistry

			err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, builtinRegistry)
			assert.NoError(t, err)

			scope := typeEngine.GetScope("test.test")
			assert.NotNil(t, scope)

			xBinding := scope.Variables["x"]
			assert.NotNil(t, xBinding, "Variable 'x' should be bound for code: %s", tt.code)
			assert.Equal(t, tt.expectedType, xBinding.Type.TypeFQN)
		})
	}
}
