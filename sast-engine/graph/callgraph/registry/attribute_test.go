package registry

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestNewAttributeRegistry(t *testing.T) {
	registry := NewAttributeRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Classes)
	assert.Equal(t, 0, registry.Size())
}

func TestAddClassAttributes(t *testing.T) {
	registry := NewAttributeRegistry()

	classAttrs := &core.ClassAttributes{
		ClassFQN:   "myapp.User",
		Attributes: make(map[string]*core.ClassAttribute),
		Methods:    []string{"__init__", "save"},
		FilePath:   "/path/to/user.py",
	}

	registry.AddClassAttributes(classAttrs)

	assert.Equal(t, 1, registry.Size())
	assert.True(t, registry.HasClass("myapp.User"))
}

func TestGetClassAttributes(t *testing.T) {
	registry := NewAttributeRegistry()

	classAttrs := &core.ClassAttributes{
		ClassFQN:   "myapp.User",
		Attributes: make(map[string]*core.ClassAttribute),
		FilePath:   "/path/to/user.py",
	}

	registry.AddClassAttributes(classAttrs)

	retrieved := registry.GetClassAttributes("myapp.User")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "myapp.User", retrieved.ClassFQN)
	assert.Equal(t, "/path/to/user.py", retrieved.FilePath)

	// Test non-existent class
	nonExistent := registry.GetClassAttributes("myapp.NonExistent")
	assert.Nil(t, nonExistent)
}

func TestAddAttribute(t *testing.T) {
	registry := NewAttributeRegistry()

	attr := &core.ClassAttribute{
		Name: "name",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
		AssignedIn: "__init__",
		Location:   &graph.SourceLocation{File: "/path/to/user.py", StartByte: 100, EndByte: 120},
		Confidence: 1.0,
	}

	// Add attribute to non-existent class (should create class)
	registry.AddAttribute("myapp.User", attr)

	assert.True(t, registry.HasClass("myapp.User"))
	classAttrs := registry.GetClassAttributes("myapp.User")
	assert.NotNil(t, classAttrs)
	assert.Equal(t, 1, len(classAttrs.Attributes))
	assert.Equal(t, "name", classAttrs.Attributes["name"].Name)

	// Add another attribute to same class
	attr2 := &core.ClassAttribute{
		Name: "email",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
		AssignedIn: "__init__",
		Confidence: 1.0,
	}

	registry.AddAttribute("myapp.User", attr2)

	classAttrs = registry.GetClassAttributes("myapp.User")
	assert.Equal(t, 2, len(classAttrs.Attributes))
}

func TestGetAttribute(t *testing.T) {
	registry := NewAttributeRegistry()

	attr := &core.ClassAttribute{
		Name: "name",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "literal",
		},
		AssignedIn: "__init__",
		Confidence: 1.0,
	}

	registry.AddAttribute("myapp.User", attr)

	// Get existing attribute
	retrieved := registry.GetAttribute("myapp.User", "name")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "name", retrieved.Name)
	assert.Equal(t, "builtins.str", retrieved.Type.TypeFQN)

	// Get non-existent attribute
	nonExistent := registry.GetAttribute("myapp.User", "nonexistent")
	assert.Nil(t, nonExistent)

	// Get attribute from non-existent class
	nonExistent = registry.GetAttribute("myapp.NonExistent", "name")
	assert.Nil(t, nonExistent)
}

func TestGetAllClasses(t *testing.T) {
	registry := NewAttributeRegistry()

	// Add multiple classes
	registry.AddClassAttributes(&core.ClassAttributes{ClassFQN: "myapp.User"})
	registry.AddClassAttributes(&core.ClassAttributes{ClassFQN: "myapp.Product"})
	registry.AddClassAttributes(&core.ClassAttributes{ClassFQN: "myapp.Order"})

	classes := registry.GetAllClasses()
	assert.Equal(t, 3, len(classes))
	assert.Contains(t, classes, "myapp.User")
	assert.Contains(t, classes, "myapp.Product")
	assert.Contains(t, classes, "myapp.Order")
}

func TestAttributeTypeInference(t *testing.T) {
	tests := []struct {
		name          string
		attributeName string
		typeFQN       string
		source        string
		expectedConf  float64
		assignedIn    string
	}{
		{
			name:          "Literal string assignment",
			attributeName: "name",
			typeFQN:       "builtins.str",
			source:        "literal",
			expectedConf:  1.0,
			assignedIn:    "__init__",
		},
		{
			name:          "Class instantiation",
			attributeName: "user",
			typeFQN:       "myapp.models.User",
			source:        "class_instantiation",
			expectedConf:  0.9,
			assignedIn:    "__init__",
		},
		{
			name:          "Function call propagation",
			attributeName: "result",
			typeFQN:       "builtins.dict",
			source:        "function_call_propagation",
			expectedConf:  0.8,
			assignedIn:    "setup",
		},
		{
			name:          "Constructor parameter",
			attributeName: "client",
			typeFQN:       "requests.HttpClient",
			source:        "constructor_param",
			expectedConf:  0.95,
			assignedIn:    "__init__",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewAttributeRegistry()

			attr := &core.ClassAttribute{
				Name: tt.attributeName,
				Type: &core.TypeInfo{
					TypeFQN:    tt.typeFQN,
					Confidence: float32(tt.expectedConf),
					Source:     tt.source,
				},
				AssignedIn: tt.assignedIn,
				Confidence: tt.expectedConf,
			}

			registry.AddAttribute("test.TestClass", attr)

			retrieved := registry.GetAttribute("test.TestClass", tt.attributeName)
			assert.NotNil(t, retrieved)
			assert.Equal(t, tt.typeFQN, retrieved.Type.TypeFQN)
			assert.Equal(t, tt.source, retrieved.Type.Source)
			assert.Equal(t, tt.expectedConf, retrieved.Confidence)
			assert.Equal(t, tt.assignedIn, retrieved.AssignedIn)
		})
	}
}

func TestThreadSafety(t *testing.T) {
	registry := NewAttributeRegistry()

	// Simulate concurrent adds
	done := make(chan bool, 10)

	for i := range 10 {
		go func(_ int) {
			attr := &core.ClassAttribute{
				Name: "attr",
				Type: &core.TypeInfo{
					TypeFQN:    "builtins.str",
					Confidence: 1.0,
					Source:     "literal",
				},
				Confidence: 1.0,
			}
			registry.AddAttribute("test.Class", attr)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Verify class was created and attribute added
	assert.True(t, registry.HasClass("test.Class"))
	assert.NotNil(t, registry.GetAttribute("test.Class", "attr"))
}

func TestMultipleAttributesPerClass(t *testing.T) {
	registry := NewAttributeRegistry()

	attributes := []struct {
		name    string
		typeFQN string
	}{
		{"name", "builtins.str"},
		{"age", "builtins.int"},
		{"email", "builtins.str"},
		{"active", "builtins.bool"},
		{"created_at", "datetime.datetime"},
	}

	// Add all attributes
	for _, attrSpec := range attributes {
		attr := &core.ClassAttribute{
			Name: attrSpec.name,
			Type: &core.TypeInfo{
				TypeFQN:    attrSpec.typeFQN,
				Confidence: 1.0,
				Source:     "literal",
			},
			AssignedIn: "__init__",
			Confidence: 1.0,
		}
		registry.AddAttribute("myapp.User", attr)
	}

	// Verify all attributes exist
	classAttrs := registry.GetClassAttributes("myapp.User")
	assert.NotNil(t, classAttrs)
	assert.Equal(t, len(attributes), len(classAttrs.Attributes))

	for _, attrSpec := range attributes {
		retrieved := registry.GetAttribute("myapp.User", attrSpec.name)
		assert.NotNil(t, retrieved, "Attribute %s should exist", attrSpec.name)
		assert.Equal(t, attrSpec.typeFQN, retrieved.Type.TypeFQN)
	}
}

func TestUpdateExistingAttribute(t *testing.T) {
	registry := NewAttributeRegistry()

	// Add initial attribute
	attr1 := &core.ClassAttribute{
		Name: "value",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 0.5,
			Source:     "heuristic",
		},
		AssignedIn: "__init__",
		Confidence: 0.5,
	}
	registry.AddAttribute("test.Class", attr1)

	// Update with better type information
	attr2 := &core.ClassAttribute{
		Name: "value",
		Type: &core.TypeInfo{
			TypeFQN:    "builtins.str",
			Confidence: 1.0,
			Source:     "annotation",
		},
		AssignedIn: "__init__",
		Confidence: 1.0,
	}
	registry.AddAttribute("test.Class", attr2)

	// Verify updated
	retrieved := registry.GetAttribute("test.Class", "value")
	assert.NotNil(t, retrieved)
	assert.Equal(t, float32(1.0), retrieved.Type.Confidence)
	assert.Equal(t, "annotation", retrieved.Type.Source)
}
