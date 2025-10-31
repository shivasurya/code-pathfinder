package callgraph

import (
	"sync"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
)

// ClassAttribute represents a single attribute of a class
type ClassAttribute struct {
	Name       string    // Attribute name (e.g., "value", "user")
	Type       *TypeInfo // Inferred type of the attribute
	AssignedIn string    // Method where assigned (e.g., "__init__", "setup")
	Location   *graph.SourceLocation
	Confidence float64 // Confidence in type inference (0.0-1.0)
}

// ClassAttributes holds all attributes for a single class
type ClassAttributes struct {
	ClassFQN   string                        // Fully qualified class name (e.g., "myapp.models.User")
	Attributes map[string]*ClassAttribute    // Map from attribute name to attribute info
	Methods    []string                      // List of method FQNs in this class
	FilePath   string                        // Source file path where class is defined
}

// AttributeRegistry is the global registry of class attributes
// It provides thread-safe access to class attribute information
type AttributeRegistry struct {
	Classes map[string]*ClassAttributes // Map from class FQN to class attributes
	mu      sync.RWMutex                // Protects concurrent access
}

// NewAttributeRegistry creates a new empty AttributeRegistry
func NewAttributeRegistry() *AttributeRegistry {
	return &AttributeRegistry{
		Classes: make(map[string]*ClassAttributes),
	}
}

// GetClassAttributes retrieves attributes for a given class FQN
// Returns nil if class is not in registry
func (ar *AttributeRegistry) GetClassAttributes(classFQN string) *ClassAttributes {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return ar.Classes[classFQN]
}

// GetAttribute retrieves a specific attribute from a class
// Returns nil if class or attribute is not found
func (ar *AttributeRegistry) GetAttribute(classFQN, attrName string) *ClassAttribute {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	classAttrs, exists := ar.Classes[classFQN]
	if !exists || classAttrs == nil {
		return nil
	}

	return classAttrs.Attributes[attrName]
}

// AddClassAttributes adds or updates attributes for a class
// Thread-safe for concurrent modifications
func (ar *AttributeRegistry) AddClassAttributes(classAttrs *ClassAttributes) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.Classes[classAttrs.ClassFQN] = classAttrs
}

// AddAttribute adds a single attribute to a class
// Creates the ClassAttributes entry if it doesn't exist
func (ar *AttributeRegistry) AddAttribute(classFQN string, attr *ClassAttribute) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	classAttrs, exists := ar.Classes[classFQN]
	if !exists {
		classAttrs = &ClassAttributes{
			ClassFQN:   classFQN,
			Attributes: make(map[string]*ClassAttribute),
			Methods:    []string{},
		}
		ar.Classes[classFQN] = classAttrs
	}

	classAttrs.Attributes[attr.Name] = attr
}

// HasClass checks if a class is registered
func (ar *AttributeRegistry) HasClass(classFQN string) bool {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	_, exists := ar.Classes[classFQN]
	return exists
}

// GetAllClasses returns a list of all registered class FQNs
func (ar *AttributeRegistry) GetAllClasses() []string {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	classes := make([]string, 0, len(ar.Classes))
	for classFQN := range ar.Classes {
		classes = append(classes, classFQN)
	}
	return classes
}

// Size returns the number of registered classes
func (ar *AttributeRegistry) Size() int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return len(ar.Classes)
}
