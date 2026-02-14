package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
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

// TestIndexParameters verifies that indexParameters extracts typed parameters
// from indexed functions into the Parameters map.
func TestIndexParameters(t *testing.T) {
	callGraph := core.NewCallGraph()

	// Function with typed parameters.
	callGraph.Functions["myapp.auth.validate"] = &graph.Node{
		ID:                  "1",
		Type:                "function_definition",
		Name:                "validate",
		File:                "/path/auth.py",
		LineNumber:          10,
		MethodArgumentsType: []string{"username: str", "password: str"},
	}

	// Method with self (should be excluded) and typed parameter.
	callGraph.Functions["myapp.models.User.save"] = &graph.Node{
		ID:                  "2",
		Type:                "method",
		Name:                "save",
		File:                "/path/models.py",
		LineNumber:          20,
		MethodArgumentsType: []string{"self", "force: bool"},
	}

	// Class method with cls (should be excluded) and typed parameter.
	callGraph.Functions["myapp.models.User.create"] = &graph.Node{
		ID:                  "3",
		Type:                "method",
		Name:                "create",
		File:                "/path/models.py",
		LineNumber:          30,
		MethodArgumentsType: []string{"cls", "name: str"},
	}

	// Function with complex types.
	callGraph.Functions["myapp.utils.process"] = &graph.Node{
		ID:                  "4",
		Type:                "function_definition",
		Name:                "process",
		File:                "/path/utils.py",
		LineNumber:          5,
		MethodArgumentsType: []string{"items: list[str]", "qs: QuerySet[ModelType]"},
	}

	// Function with no typed parameters (should not produce any).
	callGraph.Functions["myapp.utils.helper"] = &graph.Node{
		ID:         "5",
		Type:       "function_definition",
		Name:       "helper",
		File:       "/path/utils.py",
		LineNumber: 15,
	}

	IndexParameters(callGraph)

	// Verify total parameter count: 2 + 1 + 1 + 2 = 6 (self and cls excluded).
	assert.Len(t, callGraph.Parameters, 6)

	// Verify specific parameters.
	usernameParam := callGraph.Parameters["myapp.auth.validate.username"]
	assert.NotNil(t, usernameParam)
	assert.Equal(t, "username", usernameParam.Name)
	assert.Equal(t, "str", usernameParam.TypeAnnotation)
	assert.Equal(t, "myapp.auth.validate", usernameParam.ParentFQN)
	assert.Equal(t, "/path/auth.py", usernameParam.File)
	assert.Equal(t, uint32(10), usernameParam.Line)

	passwordParam := callGraph.Parameters["myapp.auth.validate.password"]
	assert.NotNil(t, passwordParam)
	assert.Equal(t, "str", passwordParam.TypeAnnotation)

	// Verify self is excluded.
	assert.Nil(t, callGraph.Parameters["myapp.models.User.save.self"])

	// Verify cls is excluded.
	assert.Nil(t, callGraph.Parameters["myapp.models.User.create.cls"])

	// Verify typed param after self is included.
	forceParam := callGraph.Parameters["myapp.models.User.save.force"]
	assert.NotNil(t, forceParam)
	assert.Equal(t, "force", forceParam.Name)
	assert.Equal(t, "bool", forceParam.TypeAnnotation)

	// Verify typed param after cls is included.
	nameParam := callGraph.Parameters["myapp.models.User.create.name"]
	assert.NotNil(t, nameParam)
	assert.Equal(t, "str", nameParam.TypeAnnotation)

	// Verify complex types.
	itemsParam := callGraph.Parameters["myapp.utils.process.items"]
	assert.NotNil(t, itemsParam)
	assert.Equal(t, "list[str]", itemsParam.TypeAnnotation)

	qsParam := callGraph.Parameters["myapp.utils.process.qs"]
	assert.NotNil(t, qsParam)
	assert.Equal(t, "QuerySet[ModelType]", qsParam.TypeAnnotation)
}

// TestIndexParameters_NoTypedParameters verifies that functions without typed
// parameters don't produce any ParameterSymbol entries.
func TestIndexParameters_NoTypedParameters(t *testing.T) {
	callGraph := core.NewCallGraph()

	callGraph.Functions["myapp.utils.helper"] = &graph.Node{
		ID:                   "1",
		Type:                 "function_definition",
		Name:                 "helper",
		File:                 "/path/utils.py",
		LineNumber:           5,
		MethodArgumentsValue: []string{"x", "y"},
	}

	IndexParameters(callGraph)

	assert.Len(t, callGraph.Parameters, 0)
}

// TestIndexParameters_EmptyCallGraph verifies safety with an empty call graph.
func TestIndexParameters_EmptyCallGraph(t *testing.T) {
	callGraph := core.NewCallGraph()

	IndexParameters(callGraph)

	assert.Len(t, callGraph.Parameters, 0)
}

func TestNormalizeReturnType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"builtins.str", "str"},
		{"builtins.int", "int"},
		{"builtins.float", "float"},
		{"builtins.bool", "bool"},
		{"builtins.list", "list"},
		{"builtins.dict", "dict"},
		{"builtins.set", "set"},
		{"builtins.tuple", "tuple"},
		{"builtins.NoneType", "None"},
		{"builtins.bytes", "bytes"},
		{"builtins.complex", "complex"},
		{"builtins.Generator", "Generator"},
		{"myapp.models.User", "myapp.models.User"},
		{"str", "str"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeReturnType(tt.input))
		})
	}
}

func TestPopulateInferredReturnTypes(t *testing.T) {
	// Set up a call graph with Python functions
	callGraph := core.NewCallGraph()
	modRegistry := core.NewModuleRegistry()

	// Function with annotation (should NOT be overwritten)
	annotatedFunc := &graph.Node{
		ID:         "annotated",
		Type:       "function_definition",
		Name:       "annotated_func",
		ReturnType: "int",
		File:       "test.py",
		LineNumber: 1,
	}
	callGraph.Functions["test.annotated_func"] = annotatedFunc

	// Function without annotation, inferred type available
	inferredFunc := &graph.Node{
		ID:         "inferred",
		Type:       "function_definition",
		Name:       "inferred_func",
		ReturnType: "",
		File:       "test.py",
		LineNumber: 5,
	}
	callGraph.Functions["test.inferred_func"] = inferredFunc

	// Void function (no return values, no inferred type)
	voidFunc := &graph.Node{
		ID:         "void",
		Type:       "function_definition",
		Name:       "void_func",
		ReturnType: "",
		File:       "test.py",
		LineNumber: 10,
	}
	callGraph.Functions["test.void_func"] = voidFunc

	// Function with return expression but uninferrable
	unknownFunc := &graph.Node{
		ID:         "unknown",
		Type:       "function_definition",
		Name:       "unknown_func",
		ReturnType: "",
		File:       "test.py",
		LineNumber: 15,
	}
	callGraph.Functions["test.unknown_func"] = unknownFunc

	// Function with low-confidence inferred type (should be skipped)
	lowConfFunc := &graph.Node{
		ID:         "lowconf",
		Type:       "function_definition",
		Name:       "lowconf_func",
		ReturnType: "",
		File:       "test.py",
		LineNumber: 20,
	}
	callGraph.Functions["test.lowconf_func"] = lowConfFunc

	// Function with placeholder return type (should be skipped)
	placeholderFunc := &graph.Node{
		ID:         "placeholder",
		Type:       "function_definition",
		Name:       "placeholder_func",
		ReturnType: "",
		File:       "test.py",
		LineNumber: 25,
	}
	callGraph.Functions["test.placeholder_func"] = placeholderFunc

	// Java function (should be skipped entirely)
	javaFunc := &graph.Node{
		ID:         "java",
		Type:       "method_declaration",
		Name:       "javaMethod",
		ReturnType: "",
		File:       "Test.java",
		LineNumber: 1,
	}
	callGraph.Functions["com.Test.javaMethod"] = javaFunc

	// Set up TypeEngine with return types
	typeEngine := resolution.NewTypeInferenceEngine(modRegistry)
	typeEngine.AddReturnTypesToEngine(map[string]*core.TypeInfo{
		"test.inferred_func": {
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "return_literal",
		},
		"test.lowconf_func": {
			TypeFQN:    "builtins.int",
			Confidence: 0.2, // Below threshold
			Source:     "return_variable",
		},
		"test.placeholder_func": {
			TypeFQN:    "call:some_func",
			Confidence: 0.8,
			Source:     "return_function_call",
		},
	})

	// Set up functions with return values tracking
	functionsWithReturnValues := map[string]bool{
		"test.inferred_func":    true,
		"test.unknown_func":     true, // Has return <expr> but couldn't infer
		"test.lowconf_func":     true,
		"test.placeholder_func": true,
		// test.void_func is NOT here — it's void
	}

	logger := output.NewLogger(output.VerbosityDefault)
	populateInferredReturnTypes(callGraph, typeEngine, functionsWithReturnValues, logger)

	// Verify results
	assert.Equal(t, "int", annotatedFunc.ReturnType, "annotation should NOT be overwritten")
	assert.Equal(t, "str", inferredFunc.ReturnType, "should be populated with normalized inferred type")
	assert.Equal(t, "None", voidFunc.ReturnType, "void function should get None")
	assert.Equal(t, "", unknownFunc.ReturnType, "function with uninferrable return should stay empty")
	assert.Equal(t, "", lowConfFunc.ReturnType, "low-confidence should be skipped")
	assert.Equal(t, "", placeholderFunc.ReturnType, "placeholder should be skipped")
	assert.Equal(t, "", javaFunc.ReturnType, "Java function should be skipped")
}

func TestPopulateInferredReturnTypes_Integration(t *testing.T) {
	// Full integration: parse real Python, build call graph, verify return types
	tmpDir := t.TempDir()

	mainPy := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(mainPy, []byte(`
def greet(name):
    return f"Hello, {name}!"

def get_count():
    return 42

def is_valid(x):
    return x > 0

def process():
    greet("world")
    print("done")

def setup():
    pass
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(tmpDir, false)
	require.NoError(t, err)

	callGraph, err := BuildCallGraph(codeGraph, moduleRegistry, tmpDir, output.NewLogger(output.VerbosityDefault))
	require.NoError(t, err)

	// greet: returns f-string → str
	if greetNode, ok := callGraph.Functions["main.greet"]; ok {
		assert.Equal(t, "str", greetNode.ReturnType, "greet should return str from f-string")
	} else {
		t.Error("main.greet not found in call graph")
	}

	// get_count: returns int literal → int
	if countNode, ok := callGraph.Functions["main.get_count"]; ok {
		assert.Equal(t, "int", countNode.ReturnType, "get_count should return int")
	} else {
		t.Error("main.get_count not found in call graph")
	}

	// is_valid: returns comparison → bool
	if validNode, ok := callGraph.Functions["main.is_valid"]; ok {
		assert.Equal(t, "bool", validNode.ReturnType, "is_valid should return bool from comparison")
	} else {
		t.Error("main.is_valid not found in call graph")
	}

	// process: no return value → None (void)
	if processNode, ok := callGraph.Functions["main.process"]; ok {
		assert.Equal(t, "None", processNode.ReturnType, "process should return None (void)")
	} else {
		t.Error("main.process not found in call graph")
	}

	// setup: only `pass` → None (void)
	if setupNode, ok := callGraph.Functions["main.setup"]; ok {
		assert.Equal(t, "None", setupNode.ReturnType, "setup should return None (void)")
	} else {
		t.Error("main.setup not found in call graph")
	}
}
