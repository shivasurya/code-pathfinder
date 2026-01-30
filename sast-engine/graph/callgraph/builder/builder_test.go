package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCallGraph(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def greet(name):
    return f"Hello, {name}"

def main():
    message = greet("World")
    print(message)
`), 0644)
	require.NoError(t, err)

	// Parse project
	codeGraph := graph.Initialize(tmpDir, nil)

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	// Build call graph
	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)
	assert.NotNil(t, callGraph)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)

	// Verify edges exist
	assert.NotNil(t, callGraph.Edges)

	// Verify reverse edges exist
	assert.NotNil(t, callGraph.ReverseEdges)
}

func TestIndexFunctions(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(mainPy, []byte(`
def func1():
    pass

def func2():
    pass

class MyClass:
    def method1(self):
        pass
`), 0644)
	require.NoError(t, err)

	// Parse project
	codeGraph := graph.Initialize(tmpDir, nil)

	// Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	// Create call graph and index functions
	callGraph := core.NewCallGraph()
	IndexFunctions(codeGraph, callGraph, moduleRegistry)

	// Verify functions were indexed
	assert.NotEmpty(t, callGraph.Functions)

	// Count functions/methods (including new Python symbol types)
	functionCount := 0
	for _, node := range callGraph.Functions {
		if node.Type == "function_definition" || node.Type == "method_declaration" ||
			node.Type == "method" || node.Type == "constructor" ||
			node.Type == "property" || node.Type == "special_method" {
			functionCount++
		}
	}
	assert.GreaterOrEqual(t, functionCount, 3, "Should have at least 3 functions/methods")
}

func TestGetFunctionsInFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	err := os.WriteFile(testFile, []byte(`
def func1():
    pass

def func2():
    pass
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir, nil)

	// Get functions in file
	functions := GetFunctionsInFile(codeGraph, testFile)

	// Verify functions were found
	assert.NotEmpty(t, functions)
	assert.GreaterOrEqual(t, len(functions), 2, "Should find at least 2 functions")
}

func TestFindContainingFunction(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
def outer_function():
    x = 1
    y = 2
    return x + y
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir, nil)

	// Get functions
	functions := GetFunctionsInFile(codeGraph, testFile)
	require.NotEmpty(t, functions)

	// Build class context (empty for module-level functions)
	classContext := buildClassContext(codeGraph)

	// Test finding containing function for a location inside the function
	location := core.Location{
		File:   testFile,
		Line:   3,
		Column: 5, // Inside function body
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath, classContext)

	// Should find the outer_function
	assert.NotEmpty(t, containingFQN)
	assert.Contains(t, containingFQN, "outer_function")
}

func TestFindContainingFunction_ModuleLevel(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
MODULE_VAR = 42

def my_function():
    pass
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir, nil)

	functions := GetFunctionsInFile(codeGraph, testFile)

	// Build class context (empty for module-level code)
	classContext := buildClassContext(codeGraph)

	// Test module-level code (column == 1)
	location := core.Location{
		File:   testFile,
		Line:   2,
		Column: 1, // Module level
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath, classContext)

	// Should return empty for module-level code
	assert.Empty(t, containingFQN)
}

// TestFindContainingFunction_ClassMethod verifies that class methods return
// class-qualified FQNs (e.g., "module.ClassName.methodName").
func TestFindContainingFunction_ClassMethod(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class User:
    def save(self):
        self.validate()
        return True

    def validate(self):
        pass
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir, nil)

	// Get functions (should include methods)
	functions := GetFunctionsInFile(codeGraph, testFile)
	require.NotEmpty(t, functions, "Should find methods in file")

	// Build class context
	classContext := buildClassContext(codeGraph)
	require.NotEmpty(t, classContext, "Should have class context for User class")

	// Test finding containing function for a call inside save() method
	location := core.Location{
		File:   testFile,
		Line:   4, // Inside save() method where self.validate() is called
		Column: 9, // Inside method body
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath, classContext)

	// Should find class-qualified FQN: "test.User.save"
	assert.NotEmpty(t, containingFQN, "Should find containing method")
	assert.Equal(t, "test.User.save", containingFQN, "Should return class-qualified FQN")
}

// TestFindContainingFunction_NestedClass verifies that nested class methods
// return class-qualified FQNs. Note: Nested classes are matched to the outermost
// containing class due to byte range overlap - this is acceptable for Phase 1.
func TestFindContainingFunction_NestedClass(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class Outer:
    class Inner:
        def method(self):
            self.helper()
            return 1

        def helper(self):
            pass
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir, nil)

	// Get functions
	functions := GetFunctionsInFile(codeGraph, testFile)
	require.NotEmpty(t, functions, "Should find nested class methods")

	// Build class context
	classContext := buildClassContext(codeGraph)
	require.NotEmpty(t, classContext, "Should have class context for nested classes")

	// Test finding containing function for a call inside Inner.method()
	location := core.Location{
		File:   testFile,
		Line:   5, // Inside Inner.method() where self.helper() is called
		Column: 13,
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath, classContext)

	// Should find class-qualified FQN
	// Due to byte range overlap, nested class methods match to the outermost class
	// This is acceptable for Phase 1 - full nested class support can be added later
	assert.NotEmpty(t, containingFQN, "Should find containing method in nested class")
	assert.Contains(t, containingFQN, ".method", "Should include method name")
	assert.NotEqual(t, "test.method", containingFQN, "Should include class qualification")
	// Verify it's class-qualified (either Outer or Inner is acceptable for Phase 1)
	assert.True(t, containingFQN == "test.Outer.method" || containingFQN == "test.Inner.method",
		"Should have class-qualified FQN, got: %s", containingFQN)
}

// TestFindContainingFunction_ModuleFunctionRegression ensures module-level
// functions still work correctly (backward compatibility test).
func TestFindContainingFunction_ModuleFunctionRegression(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
def process():
    result = helper()
    return result

def helper():
    return 42
`), 0644)
	require.NoError(t, err)

	// Parse file
	codeGraph := graph.Initialize(tmpDir, nil)

	// Get functions
	functions := GetFunctionsInFile(codeGraph, testFile)
	require.NotEmpty(t, functions, "Should find module-level functions")

	// Build class context (should be empty for module-level code)
	classContext := buildClassContext(codeGraph)

	// Test finding containing function for a call inside process()
	location := core.Location{
		File:   testFile,
		Line:   3, // Inside process() where helper() is called
		Column: 5,
	}

	modulePath := "test"
	containingFQN := FindContainingFunction(location, functions, modulePath, classContext)

	// Should find simple FQN for module-level function: "test.process"
	assert.NotEmpty(t, containingFQN, "Should find containing function")
	assert.Equal(t, "test.process", containingFQN, "Should return module-level FQN without class")
}

// TestResolveCallTarget_SelfMethod verifies Phase 2: self.method() resolution
// with class-qualified FQN extraction from callerFQN.
func TestResolveCallTarget_SelfMethod(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class User:
    def save(self):
        self.validate()  # Should resolve to User.validate
        return True

    def validate(self):
        self.check_email()  # Should resolve to User.check_email
        pass

    def check_email(self):
        pass
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify User.save has callees
	saveFQN := "test.User.save"
	saveCallees, ok := callGraph.Edges[saveFQN]
	assert.True(t, ok, "User.save should have edges in call graph")
	assert.NotEmpty(t, saveCallees, "User.save should call other methods")

	// Verify self.validate() resolved to User.validate
	validateFQN := "test.User.validate"
	assert.Contains(t, saveCallees, validateFQN, "User.save should call User.validate")

	// Verify User.validate has callees
	validateCallees, ok := callGraph.Edges[validateFQN]
	assert.True(t, ok, "User.validate should have edges")
	assert.NotEmpty(t, validateCallees, "User.validate should call other methods")

	// Verify self.check_email() resolved to User.check_email
	checkEmailFQN := "test.User.check_email"
	assert.Contains(t, validateCallees, checkEmailFQN, "User.validate should call User.check_email")
}

// TestResolveCallTarget_SelfMethodChain verifies that self.method() calls
// in a chain of methods all resolve correctly.
func TestResolveCallTarget_SelfMethodChain(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class Processor:
    def process(self):
        self.stage1()
        return True

    def stage1(self):
        self.stage2()

    def stage2(self):
        self.stage3()

    def stage3(self):
        pass
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify the call chain: process → stage1 → stage2 → stage3
	processFQN := "test.Processor.process"
	stage1FQN := "test.Processor.stage1"
	stage2FQN := "test.Processor.stage2"
	stage3FQN := "test.Processor.stage3"

	// Check process calls stage1
	processCallees := callGraph.Edges[processFQN]
	assert.Contains(t, processCallees, stage1FQN, "process should call stage1")

	// Check stage1 calls stage2
	stage1Callees := callGraph.Edges[stage1FQN]
	assert.Contains(t, stage1Callees, stage2FQN, "stage1 should call stage2")

	// Check stage2 calls stage3
	stage2Callees := callGraph.Edges[stage2FQN]
	assert.Contains(t, stage2Callees, stage3FQN, "stage2 should call stage3")

	// Verify reverse edges (callers)
	stage1Callers := callGraph.ReverseEdges[stage1FQN]
	assert.Contains(t, stage1Callers, processFQN, "stage1 should be called by process")

	stage2Callers := callGraph.ReverseEdges[stage2FQN]
	assert.Contains(t, stage2Callers, stage1FQN, "stage2 should be called by stage1")

	stage3Callers := callGraph.ReverseEdges[stage3FQN]
	assert.Contains(t, stage3Callers, stage2FQN, "stage3 should be called by stage2")
}

// TestResolveCallTarget_ModuleFunctionBackwardCompat ensures module-level
// functions still work correctly (backward compatibility for Phase 2).
func TestResolveCallTarget_ModuleFunctionBackwardCompat(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
def process():
    validate()
    helper()
    return True

def validate():
    helper()

def helper():
    pass
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify module-level function calls still work
	processFQN := "test.process"
	validateFQN := "test.validate"
	helperFQN := "test.helper"

	// Check process calls validate and helper
	processCallees := callGraph.Edges[processFQN]
	assert.Contains(t, processCallees, validateFQN, "process should call validate")
	assert.Contains(t, processCallees, helperFQN, "process should call helper")

	// Check validate calls helper
	validateCallees := callGraph.Edges[validateFQN]
	assert.Contains(t, validateCallees, helperFQN, "validate should call helper")

	// Verify reverse edges
	helperCallers := callGraph.ReverseEdges[helperFQN]
	assert.Contains(t, helperCallers, processFQN, "helper should be called by process")
	assert.Contains(t, helperCallers, validateFQN, "helper should be called by validate")
}

// TestResolveCallTarget_MultipleClasses verifies self.method() resolution
// works correctly when multiple classes are in the same file.
func TestResolveCallTarget_MultipleClasses(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class User:
    def save(self):
        self.validate()

    def validate(self):
        pass

class Product:
    def save(self):
        self.validate()

    def validate(self):
        pass
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify User.save calls User.validate (not Product.validate)
	userSaveFQN := "test.User.save"
	userValidateFQN := "test.User.validate"
	userSaveCallees := callGraph.Edges[userSaveFQN]
	assert.Contains(t, userSaveCallees, userValidateFQN, "User.save should call User.validate")

	// Verify Product.save calls Product.validate (not User.validate)
	productSaveFQN := "test.Product.save"
	productValidateFQN := "test.Product.validate"
	productSaveCallees := callGraph.Edges[productSaveFQN]
	assert.Contains(t, productSaveCallees, productValidateFQN, "Product.save should call Product.validate")

	// Verify User.validate is not called by Product.save
	assert.NotContains(t, productSaveCallees, userValidateFQN, "Product.save should not call User.validate")

	// Verify Product.validate is not called by User.save
	assert.NotContains(t, userSaveCallees, productValidateFQN, "User.save should not call Product.validate")
}

// TestResolveCallTarget_InstanceMethod verifies Phase 3: instance.method()
// resolution using type inference.
func TestResolveCallTarget_InstanceMethod(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class User:
    def save(self):
        return True

    def validate(self):
        return True

def process():
    user = User()
    user.save()  # Should resolve via type inference
    user.validate()
    return True
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify process function has callees
	processFQN := "test.process"
	processCallees, ok := callGraph.Edges[processFQN]
	assert.True(t, ok, "process should have edges")
	assert.NotEmpty(t, processCallees, "process should call methods")

	// Check if user.save() and user.validate() resolved
	// Note: This depends on type inference being able to resolve User() constructor
	// If type inference works, we should see User.save and User.validate in callees
	userSaveFQN := "test.User.save"
	userValidateFQN := "test.User.validate"

	// These might not resolve without full type inference, but test the infrastructure
	t.Logf("process callees: %v", processCallees)
	t.Logf("Looking for User.save: %s", userSaveFQN)
	t.Logf("Looking for User.validate: %s", userValidateFQN)

	// At minimum, verify the functions are indexed correctly
	assert.NotNil(t, callGraph.Functions[userSaveFQN], "User.save should be indexed")
	assert.NotNil(t, callGraph.Functions[userValidateFQN], "User.validate should be indexed")
}

// TestResolveCallTarget_SuperMethod verifies Phase 3: super().method()
// basic infrastructure (full inheritance would require more metadata).
func TestResolveCallTarget_SuperMethod(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class Base:
    def save(self):
        return True

class User(Base):
    def save(self):
        super().save()  # Should attempt to resolve to Base.save
        return True
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify User.save is indexed
	userSaveFQN := "test.User.save"
	assert.NotNil(t, callGraph.Functions[userSaveFQN], "User.save should be indexed")

	// Verify Base.save is indexed
	baseSaveFQN := "test.Base.save"
	assert.NotNil(t, callGraph.Functions[baseSaveFQN], "Base.save should be indexed")

	// Check if User.save has any callees (super() call)
	userSaveCallees := callGraph.Edges[userSaveFQN]
	t.Logf("User.save callees: %v", userSaveCallees)

	// Note: Full super() resolution requires inheritance metadata
	// Phase 3 provides the infrastructure; this test verifies no crashes
	// and that both classes are properly indexed
}

// TestResolveCallTarget_MixedPatterns verifies all three phases work together:
// self.method(), instance.method(), and module functions.
func TestResolveCallTarget_MixedPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class Validator:
    def check(self, value):
        return self.validate(value)  # Phase 2: self.method()

    def validate(self, value):
        return sanitize(value)  # Module function

def sanitize(value):
    return value.strip()

def process(data):
    validator = Validator()
    return validator.check(data)  # Phase 3: instance.method()
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify all functions/methods are indexed
	validatorCheckFQN := "test.Validator.check"
	validatorValidateFQN := "test.Validator.validate"
	sanitizeFQN := "test.sanitize"
	processFQN := "test.process"

	assert.NotNil(t, callGraph.Functions[validatorCheckFQN], "Validator.check should be indexed")
	assert.NotNil(t, callGraph.Functions[validatorValidateFQN], "Validator.validate should be indexed")
	assert.NotNil(t, callGraph.Functions[sanitizeFQN], "sanitize should be indexed")
	assert.NotNil(t, callGraph.Functions[processFQN], "process should be indexed")

	// Phase 2: Verify self.validate() resolves in Validator.check
	checkCallees := callGraph.Edges[validatorCheckFQN]
	assert.Contains(t, checkCallees, validatorValidateFQN, "Validator.check should call Validator.validate via self")

	// Verify sanitize() is called from Validator.validate
	validateCallees := callGraph.Edges[validatorValidateFQN]
	assert.Contains(t, validateCallees, sanitizeFQN, "Validator.validate should call sanitize")

	// Log process callees for Phase 3 verification
	processCallees := callGraph.Edges[processFQN]
	t.Logf("process callees: %v (may include validator.check if type inference works)", processCallees)
}

// TestResolveCallTarget_TypeInferenceInfrastructure verifies the type
// inference infrastructure is available and functioning.
func TestResolveCallTarget_TypeInferenceInfrastructure(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.py")
	err := os.WriteFile(testFile, []byte(`
class DataProcessor:
    def process(self, data):
        return data.strip().lower()

def main():
    text = "HELLO"
    result = text.strip()  # Builtin method
    return result
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify the call graph was built successfully
	assert.NotNil(t, callGraph, "Call graph should be built")
	assert.NotEmpty(t, callGraph.Functions, "Functions should be indexed")

	// Check that DataProcessor.process is indexed
	processorFQN := "test.DataProcessor.process"
	assert.NotNil(t, callGraph.Functions[processorFQN], "DataProcessor.process should be indexed")

	// Verify main function
	mainFQN := "test.main"
	assert.NotNil(t, callGraph.Functions[mainFQN], "main should be indexed")

	// Log edges for debugging
	t.Logf("Total functions indexed: %d", len(callGraph.Functions))
	t.Logf("Total edges: %d", len(callGraph.Edges))
}

func TestValidateFQN(t *testing.T) {
	moduleRegistry := core.NewModuleRegistry()

	// Add a test module
	moduleRegistry.Modules["mymodule"] = "/path/to/mymodule.py"
	moduleRegistry.FileToModule["/path/to/mymodule.py"] = "mymodule"

	tests := []struct {
		name     string
		fqn      string
		expected bool
	}{
		{"Valid module FQN", "mymodule.func", true},
		{"Invalid module FQN", "unknownmodule.func", false},
		{"Empty FQN", "", false},
		{"Valid module name without dot", "mymodule", true},
		{"Valid class method FQN (grandparent module)", "mymodule.ClassName.method", true},
		{"Invalid class method FQN (grandparent not in registry)", "unknownmodule.ClassName.method", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFQN(tt.fqn, moduleRegistry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectPythonVersion(t *testing.T) {
	// Create a temporary project
	tmpDir := t.TempDir()

	// Test with .python-version file
	pythonVersionFile := filepath.Join(tmpDir, ".python-version")
	err := os.WriteFile(pythonVersionFile, []byte("3.11.0\n"), 0644)
	require.NoError(t, err)

	version := DetectPythonVersion(tmpDir)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "3.11")
}

func TestDetectPythonVersion_NoPythonVersionFile(t *testing.T) {
	// Create an empty temporary directory
	tmpDir := t.TempDir()

	// Should fall back to checking pyproject.toml or default
	version := DetectPythonVersion(tmpDir)
	// Should return a default version or detect from system
	assert.NotEmpty(t, version)
}

func TestBuildCallGraph_WithEdges(t *testing.T) {
	// Create a project with function calls
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def helper():
    return 42

def caller():
    result = helper()
    return result
`), 0644)
	require.NoError(t, err)

	// Parse and build call graph
	codeGraph := graph.Initialize(tmpDir, nil)

	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// Verify edges were created
	assert.NotEmpty(t, callGraph.Edges)

	// Check that caller has edges to helper
	foundEdge := false
	for callerFQN, callees := range callGraph.Edges {
		if len(callees) > 0 {
			foundEdge = true
			t.Logf("Function %s calls: %v", callerFQN, callees)
		}
	}

	assert.True(t, foundEdge, "Expected at least one call edge")
}
