package registry

import (
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// AttributeRegistry is the global registry of class attributes
// It provides thread-safe access to class attribute information.
type AttributeRegistry struct {
	Classes map[string]*core.ClassAttributes // Map from class FQN to class attributes
	mu      sync.RWMutex                     // Protects concurrent access
}

// NewAttributeRegistry creates a new empty AttributeRegistry.
func NewAttributeRegistry() *AttributeRegistry {
	return &AttributeRegistry{
		Classes: make(map[string]*core.ClassAttributes),
	}
}

// GetClassAttributes retrieves attributes for a given class FQN
// Returns nil if class is not in registry.
func (ar *AttributeRegistry) GetClassAttributes(classFQN string) *core.ClassAttributes {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return ar.Classes[classFQN]
}

// GetAttribute retrieves a specific attribute from a class
// Returns nil if class or attribute is not found.
func (ar *AttributeRegistry) GetAttribute(classFQN, attrName string) *core.ClassAttribute {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	classAttrs, exists := ar.Classes[classFQN]
	if !exists || classAttrs == nil {
		return nil
	}

	return classAttrs.Attributes[attrName]
}

// AddClassAttributes adds or updates attributes for a class
// Thread-safe for concurrent modifications.
func (ar *AttributeRegistry) AddClassAttributes(classAttrs *core.ClassAttributes) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.Classes[classAttrs.ClassFQN] = classAttrs
}

// AddAttribute adds a single attribute to a class
// Creates the ClassAttributes entry if it doesn't exist.
func (ar *AttributeRegistry) AddAttribute(classFQN string, attr *core.ClassAttribute) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	classAttrs, exists := ar.Classes[classFQN]
	if !exists {
		classAttrs = &core.ClassAttributes{
			ClassFQN:   classFQN,
			Attributes: make(map[string]*core.ClassAttribute),
			Methods:    []string{},
		}
		ar.Classes[classFQN] = classAttrs
	}

	classAttrs.Attributes[attr.Name] = attr
}

// HasClass checks if a class is registered.
func (ar *AttributeRegistry) HasClass(classFQN string) bool {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	_, exists := ar.Classes[classFQN]
	return exists
}

// GetAllClasses returns a list of all registered class FQNs.
func (ar *AttributeRegistry) GetAllClasses() []string {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	classes := make([]string, 0, len(ar.Classes))
	for classFQN := range ar.Classes {
		classes = append(classes, classFQN)
	}
	return classes
}

// Size returns the number of registered classes.
func (ar *AttributeRegistry) Size() int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return len(ar.Classes)
}
