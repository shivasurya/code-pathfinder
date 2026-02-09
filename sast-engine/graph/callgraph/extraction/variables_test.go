package extraction

import (
	"os"
	"path/filepath"
	"strings"
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.test_function")
	assert.NotNil(t, scope)

	// Check 'name' variable
	nameBindings := scope.Variables["name"]
	assert.Len(t, nameBindings, 1)
	nameBinding := nameBindings[0]
	assert.Equal(t, "builtins.str", nameBinding.Type.TypeFQN)
	assert.Equal(t, float32(1.0), nameBinding.Type.Confidence)
	assert.Equal(t, "literal", nameBinding.Type.Source)

	// Check 'greeting' variable
	greetingBindings := scope.Variables["greeting"]
	assert.Len(t, greetingBindings, 1)
	greetingBinding := greetingBindings[0]
	assert.Equal(t, "builtins.str", greetingBinding.Type.TypeFQN)

	// Check 'multiline' variable
	multilineBindings := scope.Variables["multiline"]
	assert.Len(t, multilineBindings, 1)
	multilineBinding := multilineBindings[0]
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.calculate")
	assert.NotNil(t, scope)

	// Check 'count' variable (int)
	countBindings := scope.Variables["count"]
	assert.Len(t, countBindings, 1)
	countBinding := countBindings[0]
	assert.Equal(t, "builtins.int", countBinding.Type.TypeFQN)

	// Check 'price' variable (float)
	priceBindings := scope.Variables["price"]
	assert.Len(t, priceBindings, 1)
	priceBinding := priceBindings[0]
	assert.Equal(t, "builtins.float", priceBinding.Type.TypeFQN)

	// Check 'negative' variable (int)
	negativeBindings := scope.Variables["negative"]
	assert.Len(t, negativeBindings, 1)
	negativeBinding := negativeBindings[0]
	assert.Equal(t, "builtins.int", negativeBinding.Type.TypeFQN)

	// Check 'scientific' variable (float)
	scientificBindings := scope.Variables["scientific"]
	assert.Len(t, scientificBindings, 1)
	scientificBinding := scientificBindings[0]
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.process_data")
	assert.NotNil(t, scope)

	// Check 'items' variable (list)
	itemsBindings := scope.Variables["items"]
	assert.Len(t, itemsBindings, 1)
	itemsBinding := itemsBindings[0]
	assert.Equal(t, "builtins.list", itemsBinding.Type.TypeFQN)

	// Check 'config' variable (dict)
	configBindings := scope.Variables["config"]
	assert.Len(t, configBindings, 1)
	configBinding := configBindings[0]
	assert.Equal(t, "builtins.dict", configBinding.Type.TypeFQN)

	// Check 'unique' variable (set)
	uniqueBindings := scope.Variables["unique"]
	assert.Len(t, uniqueBindings, 1)
	uniqueBinding := uniqueBindings[0]
	assert.Equal(t, "builtins.set", uniqueBinding.Type.TypeFQN)

	// Check 'coords' variable (tuple)
	coordsBindings := scope.Variables["coords"]
	assert.Len(t, coordsBindings, 1)
	coordsBinding := coordsBindings[0]
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.check_status")
	assert.NotNil(t, scope)

	// Check 'is_active' variable
	isActiveBindings := scope.Variables["is_active"]
	assert.Len(t, isActiveBindings, 1)
	isActiveBinding := isActiveBindings[0]
	assert.Equal(t, "builtins.bool", isActiveBinding.Type.TypeFQN)

	// Check 'is_deleted' variable
	isDeletedBindings := scope.Variables["is_deleted"]
	assert.Len(t, isDeletedBindings, 1)
	isDeletedBinding := isDeletedBindings[0]
	assert.Equal(t, "builtins.bool", isDeletedBinding.Type.TypeFQN)

	// Check 'result' variable
	resultBindings := scope.Variables["result"]
	assert.Len(t, resultBindings, 1)
	resultBinding := resultBindings[0]
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify
	scope := typeEngine.GetScope("test.process")
	assert.NotNil(t, scope)
	assert.Equal(t, 4, len(scope.Variables))

	assert.True(t, len(scope.Variables["name"]) > 0)
	assert.True(t, len(scope.Variables["age"]) > 0)
	assert.True(t, len(scope.Variables["items"]) > 0)
	assert.True(t, len(scope.Variables["is_valid"]) > 0)
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify outer function scope
	outerScope := typeEngine.GetScope("test.outer")
	assert.NotNil(t, outerScope)
	assert.True(t, len(outerScope.Variables["x"]) > 0)
	assert.Equal(t, "builtins.str", outerScope.Variables["x"][0].Type.TypeFQN)

	// Verify inner function scope
	innerScope := typeEngine.GetScope("test.outer.inner")
	assert.NotNil(t, innerScope)
	assert.True(t, len(innerScope.Variables["y"]) > 0)
	assert.Equal(t, "builtins.str", innerScope.Variables["y"][0].Type.TypeFQN)
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify - should have both assignments in the slice
	scope := typeEngine.GetScope("test.reassign")
	assert.NotNil(t, scope)

	xBindings := scope.Variables["x"]
	assert.Len(t, xBindings, 2)
	// Last assignment is string (at index 1)
	xBinding := xBindings[len(xBindings)-1]
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments - should not error
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
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

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	// Extract assignments
	err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	// Verify locations
	scope := typeEngine.GetScope("test.test")
	assert.NotNil(t, scope)

	xBindings := scope.Variables["x"]
	assert.Len(t, xBindings, 1)
	xBinding := xBindings[0]
	assert.Equal(t, filePath, xBinding.Location.File)
	assert.Equal(t, uint32(3), xBinding.Location.Line)

	yBindings := scope.Variables["y"]
	assert.Len(t, yBindings, 1)
	yBinding := yBindings[0]
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

			modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
			assert.NoError(t, err)

			typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
			typeEngine.Builtins = builtinRegistry

			err = ExtractVariableAssignments(filePath, sourceCode, typeEngine, modRegistry, builtinRegistry, nil)
			assert.NoError(t, err)

			scope := typeEngine.GetScope("test.test")
			assert.NotNil(t, scope)

			xBindings := scope.Variables["x"]
			assert.True(t, len(xBindings) > 0, "Variable 'x' should be bound for code: %s", tt.code)
			xBinding := xBindings[0]
			assert.Equal(t, tt.expectedType, xBinding.Type.TypeFQN)
		})
	}
}

// TestExtractVariableAssignments_BooleanOperator tests local variable type inference from boolean operators.
// Validates that local variable assignments using boolean operators (or, and) correctly
// infer types from operands with appropriate confidence penalties.
//
// Example tested:
//   - config or Settings() → infer Settings (confidence: 0.76)
//
// This complements the class attribute boolean operator support (attributes.go) by
// handling local variables in regular functions.
func TestExtractVariableAssignments_BooleanOperator(t *testing.T) {
	tests := []struct {
		name         string
		sourceCode   string
		varName      string
		expectedType string
		expectedConf float32
	}{
		{
			name: "or with class instantiation",
			sourceCode: `
def process(config):
    settings = config or Settings()
    return settings
`,
			varName:      "settings",
			expectedType: "test.Settings", // Resolved via import map
			expectedConf: 0.76,           // 0.8 (base for class) * 0.95 (boolean penalty)
		},
		{
			name: "or with list literal",
			sourceCode: `
def get_items(items):
    result = items or []
    return result
`,
			varName:      "result",
			expectedType: "builtins.list",
			expectedConf: 0.95, // 1.0 (base) * 0.95 (penalty)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.py")
			err := os.WriteFile(filePath, []byte(tt.sourceCode), 0644)
			assert.NoError(t, err)

			modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
			assert.NoError(t, err)

			typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
			typeEngine.Builtins = registry.NewBuiltinRegistry()

			// Extract assignments
			err = ExtractVariableAssignments(filePath, []byte(tt.sourceCode), typeEngine, modRegistry, typeEngine.Builtins, nil)
			assert.NoError(t, err)

			// Determine function name from source code
			var funcName string
			switch {
			case strings.Contains(tt.sourceCode, "def process"):
				funcName = "test.process"
			case strings.Contains(tt.sourceCode, "def get_items"):
				funcName = "test.get_items"
			default:
				t.Fatalf("Unknown function in source code")
			}

			// Verify
			scope := typeEngine.GetScope(funcName)
			assert.NotNil(t, scope, "Scope should exist for function %s", funcName)

			bindings := scope.Variables[tt.varName]
			assert.True(t, len(bindings) > 0, "Variable %s should be bound", tt.varName)
			binding := bindings[0]
			assert.Equal(t, tt.expectedType, binding.Type.TypeFQN, "Type FQN mismatch for %s", tt.varName)
			assert.InDelta(t, tt.expectedConf, binding.Type.Confidence, 0.01, "Confidence mismatch for %s", tt.varName)

			// Verify source includes "boolean_or_" or "boolean_and_" prefix
			assert.True(t,
				strings.HasPrefix(binding.Type.Source, "boolean_or_") ||
					strings.HasPrefix(binding.Type.Source, "boolean_and_"),
				"Source should have boolean prefix, got: %s", binding.Type.Source)
		})
	}
}

// TestInferFromBooleanOp_EdgeCases tests edge cases for inferFromBooleanOp.
// Ensures boolean operator type inference handles various operand combinations.
func TestInferFromBooleanOp_EdgeCases(t *testing.T) {
	code := `
def test():
    x = "left" or "right"
`
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(filePath, []byte(code), 0644)
	assert.NoError(t, err)

	modRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	assert.NoError(t, err)

	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.Builtins = registry.NewBuiltinRegistry()

	err = ExtractVariableAssignments(filePath, []byte(code), typeEngine, modRegistry, typeEngine.Builtins, nil)
	assert.NoError(t, err)

	scope := typeEngine.GetScope("test.test")
	assert.NotNil(t, scope, "Scope should exist")

	xBindings := scope.Variables["x"]
	assert.True(t, len(xBindings) > 0, "Variable x should be bound")
	binding := xBindings[0]
	assert.Equal(t, "builtins.str", binding.Type.TypeFQN, "Should infer string type from right operand")
}
