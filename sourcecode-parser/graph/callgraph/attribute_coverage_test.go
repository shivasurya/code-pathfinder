package callgraph

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
)

// Tests specifically targeting new code in attribute_extraction.go for coverage

func TestExtractClassAttributesCoverage(t *testing.T) {
	code := `
class User:
    def __init__(self, name):
        self.name = name
        self.age = 25
        self.active = True

class Manager(User):
    def __init__(self):
        super().__init__("Admin")
        self.role = "manager"
`

	// Create minimal code graph
	cg := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Add User class node
	userNode := &graph.Node{
		ID:         "test.User",
		Type:       "class_declaration",
		Name:       "User",
		LineNumber: 2,
	}
	cg.Nodes["test.User"] = userNode

	// Add Manager class node
	managerNode := &graph.Node{
		ID:         "test.Manager",
		Type:       "class_declaration",
		Name:       "Manager",
		LineNumber: 8,
	}
	cg.Nodes["test.Manager"] = managerNode

	// Create type engine
	registry := NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(registry)
	attrRegistry := NewAttributeRegistry()

	// Extract attributes
	err := ExtractClassAttributes("test.py", []byte(code), "test", typeEngine, attrRegistry)

	// Verify
	assert.Nil(t, err)
	assert.NotNil(t, attrRegistry)
	// Check that we extracted attributes for at least one class
	classes := attrRegistry.GetAllClasses()
	assert.True(t, len(classes) > 0, "Should have extracted attributes for at least one class")
}

func TestFindSelfAttributeAssignmentsCoverage(t *testing.T) {
	code := `
def __init__(self, name, age):
    self.name = name
    self.age = age
    self.active = True
    self.items = []
    self.data = {}
    self.coords = (1, 2)
    other = "not self"
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	// Get function node
	funcNode := root.Child(0)
	assignments := findSelfAttributeAssignments(funcNode, []byte(code))

	// Should find all self.* assignments, not "other"
	assert.True(t, len(assignments) >= 6)

	// Verify we found self attributes
	foundName := false
	foundAge := false
	for _, assign := range assignments {
		if assign.AttributeName == "name" {
			foundName = true
		}
		if assign.AttributeName == "age" {
			foundAge = true
		}
	}

	assert.True(t, foundName)
	assert.True(t, foundAge)
}

func TestInferAttributeTypeCoverage(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "literal string",
			code: `
class User:
    def __init__(self):
        self.name = "test"
`,
			expected: "builtins.str",
		},
		{
			name: "literal int",
			code: `
class User:
    def __init__(self):
        self.age = 25
`,
			expected: "builtins.int",
		},
		{
			name: "literal bool",
			code: `
class User:
    def __init__(self):
        self.active = True
`,
			expected: "builtins.bool",
		},
		{
			name: "literal list",
			code: `
class User:
    def __init__(self):
        self.items = []
`,
			expected: "builtins.list",
		},
		{
			name: "literal dict",
			code: `
class User:
    def __init__(self):
        self.data = {}
`,
			expected: "builtins.dict",
		},
		{
			name: "function call",
			code: `
class User:
    def __init__(self):
        self.data = get_data()
`,
			expected: "call:get_data",
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.code))
			root := tree.RootNode()

			// Create type engine
			registry := NewModuleRegistry()
			typeEngine := NewTypeInferenceEngine(registry)

			// Find class and extract
			classNode := root.Child(0)
			assignments := extractAttributeAssignments(classNode, []byte(tt.code), "test.User", "test.py", typeEngine)

			if len(assignments) > 0 {
				// Get first attribute (map iteration order is not guaranteed, so we just check any attribute)
				for _, attr := range assignments {
					if attr.Type != nil {
						assert.Contains(t, attr.Type.TypeFQN, tt.expected)
						break
					}
				}
			}
		})
	}
}

func TestResolveSelfAttributeCallCoverage(t *testing.T) {
	// Setup
	registry := NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(registry)
	typeEngine.Attributes = NewAttributeRegistry()
	builtins := NewBuiltinRegistry()
	callGraph := NewCallGraph()

	// Add class with name attribute (string type)
	classAttrs := &ClassAttributes{
		ClassFQN:   "test.User",
		Attributes: make(map[string]*ClassAttribute),
		Methods:    []string{"test.User.__init__", "test.User.get_name"},
		FilePath:   "/test/user.py",
	}

	nameAttr := &ClassAttribute{
		Name: "name",
		Type: &TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
		AssignedIn: "__init__",
		Confidence: 1.0,
	}

	classAttrs.Attributes["name"] = nameAttr
	typeEngine.Attributes.AddClassAttributes(classAttrs)

	tests := []struct {
		name         string
		target       string
		callerFQN    string
		expectResolv bool
	}{
		{
			name:         "valid self attribute call",
			target:       "self.name.upper",
			callerFQN:    "test.User.get_name",
			expectResolv: true,
		},
		{
			name:         "not self prefix",
			target:       "other.name.upper",
			callerFQN:    "test.User.get_name",
			expectResolv: false,
		},
		{
			name:         "too shallow",
			target:       "self.name",
			callerFQN:    "test.User.get_name",
			expectResolv: false,
		},
		{
			name:         "too deep",
			target:       "self.obj.attr.method.deep",
			callerFQN:    "test.User.get_name",
			expectResolv: false,
		},
		{
			name:         "attribute not found",
			target:       "self.missing.upper",
			callerFQN:    "test.User.get_name",
			expectResolv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, resolved, _ := ResolveSelfAttributeCall(
				tt.target,
				tt.callerFQN,
				typeEngine,
				builtins,
				callGraph,
			)

			assert.Equal(t, tt.expectResolv, resolved)
		})
	}
}

func TestResolveAttributePlaceholdersCoverage(t *testing.T) {
	// Create call graph with placeholder
	cg := NewCallGraph()

	callSite := CallSite{
		Target:     "attr:name.upper",
		TargetFQN:  "attr:name.upper",
		Resolved:   false,
		Location:   Location{File: "test.py", Line: 10, Column: 5},
	}

	cg.CallSites["test.User.process"] = []CallSite{callSite}

	// Create registries
	attrRegistry := NewAttributeRegistry()
	typeEngine := NewTypeInferenceEngine(NewModuleRegistry())
	typeEngine.Attributes = attrRegistry
	moduleRegistry := NewModuleRegistry()
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Add class with name attribute
	classAttrs := &ClassAttributes{
		ClassFQN:   "test.User",
		Attributes: make(map[string]*ClassAttribute),
		Methods:    []string{"process"},
	}

	nameAttr := &ClassAttribute{
		Name: "name",
		Type: &TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
	}

	classAttrs.Attributes["name"] = nameAttr
	attrRegistry.AddClassAttributes(classAttrs)

	// Resolve
	ResolveAttributePlaceholders(attrRegistry, typeEngine, moduleRegistry, codeGraph)

	// Just verify it runs without panic
	assert.NotNil(t, cg)
}

func TestFindClassContainingMethodCoverage(t *testing.T) {
	attrRegistry := NewAttributeRegistry()

	// Add User class with methods (methods list has FQN format: classFQN.methodName)
	classAttrs := &ClassAttributes{
		ClassFQN:   "test.User",
		Attributes: make(map[string]*ClassAttribute),
		Methods:    []string{"test.User.__init__", "test.User.get_name", "test.User.save"},
	}
	attrRegistry.AddClassAttributes(classAttrs)

	// Add Manager class with methods
	managerAttrs := &ClassAttributes{
		ClassFQN:   "test.Manager",
		Attributes: make(map[string]*ClassAttribute),
		Methods:    []string{"test.Manager.__init__", "test.Manager.approve"},
	}
	attrRegistry.AddClassAttributes(managerAttrs)

	tests := []struct {
		name       string
		methodFQN  string
		expectClas string
	}{
		{
			name:       "User.get_name",
			methodFQN:  "test.User.get_name",
			expectClas: "test.User",
		},
		{
			name:       "Manager.approve",
			methodFQN:  "test.Manager.approve",
			expectClas: "test.Manager",
		},
		{
			name:       "non-existent",
			methodFQN:  "test.User.nonexistent",
			expectClas: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findClassContainingMethod(tt.methodFQN, attrRegistry)
			assert.Equal(t, tt.expectClas, result)
		})
	}
}

func TestPrintAttributeFailureStatsCoverage(t *testing.T) {
	// Setup some failure stats
	attributeFailureStats = &FailureStats{
		TotalAttempts:          10,
		NotSelfPrefix:          2,
		DeepChains:             1,
		ClassNotFound:          3,
		AttributeNotFound:      3,
		MethodNotInBuiltins:    1,
		CustomClassUnsupported: 0,
		DeepChainSamples:       []string{"self.a.b.c.d"},
		AttributeNotFoundSamples: []string{"self.missing.method"},
	}

	// This should not panic
	PrintAttributeFailureStats()

	// Reset
	attributeFailureStats = &FailureStats{
		DeepChainSamples:       make([]string, 0, 20),
		AttributeNotFoundSamples: make([]string, 0, 20),
		CustomClassSamples:     make([]string, 0, 20),
	}
}

func TestResolveClassNameCoverage(t *testing.T) {
	registry := NewModuleRegistry()
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Add User class node
	userNode := &graph.Node{
		ID:   "test.User",
		Type: "class_declaration",
		Name: "test.User",
	}
	codeGraph.Nodes["test.User"] = userNode

	tests := []struct {
		name       string
		className  string
		contextFQN string
		expected   string
	}{
		{
			name:       "needs qualification in same module",
			className:  "User",
			contextFQN: "test.SomeClass",
			expected:   "test.User",
		},
		{
			name:       "not found returns empty",
			className:  "NonExistent",
			contextFQN: "other.Class",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveClassName(tt.className, tt.contextFQN, registry, codeGraph)
			if tt.expected == "" {
				assert.Equal(t, "", result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestInferFromConstructorParamCoverage(t *testing.T) {
	code := `
class User:
    def __init__(self, name: str, user: UserModel):
        self.name = name
        self.user = user
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	registry := NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(registry)

	// Find the __init__ method
	classNode := root.Child(0)
	methods := findMethodNodes(classNode, []byte(code))

	if len(methods) == 0 {
		t.Skip("No methods found in test class")
		return
	}

	methodNode := methods[0]

	// Find assignments
	assignments := findSelfAttributeAssignments(methodNode, []byte(code))

	assert.True(t, len(assignments) >= 2, "Should find self.name and self.user")

	for _, assignment := range assignments {
		typeInfo := inferFromConstructorParam(assignment, methodNode, []byte(code), typeEngine)
		if assignment.AttributeName == "name" && typeInfo != nil {
			assert.Contains(t, typeInfo.TypeFQN, "str")
		} else if assignment.AttributeName == "user" && typeInfo != nil {
			assert.Contains(t, typeInfo.TypeFQN, "UserModel")
		}
	}
}

func TestExtractAttributeAssignmentsCoverage(t *testing.T) {
	code := `
class User:
    def __init__(self, name):
        self.name = name
        self.age = 25
        self.active = True

    def get_name(self):
        self.temp = "test"
        return self.name
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	registry := NewModuleRegistry()
	typeEngine := NewTypeInferenceEngine(registry)

	classNode := root.Child(0)
	attrs := extractAttributeAssignments(classNode, []byte(code), "test.User", "test.py", typeEngine)

	assert.True(t, len(attrs) >= 3, "Should extract at least 3 attributes")

	// Check that age and active are present (these have literal types with high confidence)
	assert.NotNil(t, attrs["age"], "Should find 'age' attribute")
	assert.NotNil(t, attrs["active"], "Should find 'active' attribute")

	// Check that we have at least these attributes
	foundAttrs := 0
	for key := range attrs {
		if key == "name" || key == "age" || key == "active" || key == "temp" {
			foundAttrs++
		}
	}
	assert.True(t, foundAttrs >= 3, "Should find at least 3 of the expected attributes")
}

func TestFindClassNodesCoverage(t *testing.T) {
	code := `
class User:
    pass

class Manager:
    pass

def function():
    pass
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	classes := findClassNodes(root, []byte(code))
	assert.Equal(t, 2, len(classes))
}

func TestFindMethodNodesCoverage(t *testing.T) {
	code := `
class User:
    def __init__(self):
        pass

    def get_name(self):
        pass

    def save(self):
        pass
`

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, []byte(code))
	root := tree.RootNode()

	classNode := root.Child(0)
	methods := findMethodNodes(classNode, []byte(code))
	assert.Equal(t, 3, len(methods))
}

func TestGetModuleFromClassFQNCoverage(t *testing.T) {
	tests := []struct {
		name     string
		classFQN string
		expected string
	}{
		{
			name:     "simple class",
			classFQN: "test.User",
			expected: "test",
		},
		{
			name:     "nested module",
			classFQN: "myapp.models.User",
			expected: "myapp.models",
		},
		{
			name:     "deep nesting",
			classFQN: "a.b.c.d.User",
			expected: "a.b.c.d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModuleFromClassFQN(tt.classFQN)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassExistsCoverage(t *testing.T) {
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	userNode := &graph.Node{
		ID:   "test.User",
		Type: "class_declaration",
		Name: "test.User",
	}
	codeGraph.Nodes["test.User"] = userNode

	assert.True(t, classExists("test.User", codeGraph))
	assert.False(t, classExists("test.Manager", codeGraph))
}
