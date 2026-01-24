// Package strategies provides pluggable type inference strategies.
package strategies

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// InferenceContext provides context for type inference.
type InferenceContext struct {
	// Source information
	SourceCode []byte
	FilePath   string

	// Type environment
	Store      TypeStore
	SelfType   core.Type  // Type of 'self' in current method
	ClassFQN   string     // Current class FQN

	// Registries (from existing system)
	AttrRegistry    AttributeRegistryInterface
	ModuleRegistry  ModuleRegistryInterface
	BuiltinRegistry BuiltinRegistryInterface

	// Current scope
	FunctionFQN string
	ScopeDepth  int
}

// InferenceStrategy defines the interface for type inference strategies.
// Each strategy handles specific AST node patterns.
type InferenceStrategy interface {
	// Name returns a unique identifier for this strategy.
	Name() string

	// CanHandle returns true if this strategy can process the given node.
	CanHandle(node *sitter.Node, ctx *InferenceContext) bool

	// Synthesize infers the type of a node (forward/bottom-up).
	// Returns the inferred type and confidence score.
	Synthesize(node *sitter.Node, ctx *InferenceContext) (core.Type, float64)

	// Check verifies if a node matches an expected type (backward/top-down).
	// Returns true if the node can produce the expected type.
	Check(node *sitter.Node, expectedType core.Type, ctx *InferenceContext) bool

	// Priority returns the strategy priority (higher = checked first).
	Priority() int
}

// BaseStrategy provides common functionality for strategies.
type BaseStrategy struct {
	name     string
	priority int
}

func (s *BaseStrategy) Name() string {
	return s.name
}

func (s *BaseStrategy) Priority() int {
	return s.priority
}

// NewBaseStrategy creates a new BaseStrategy.
func NewBaseStrategy(name string, priority int) BaseStrategy {
	return BaseStrategy{name: name, priority: priority}
}

// =============================================================================
// REGISTRY INTERFACES (for decoupling from concrete types)
// =============================================================================

// AttributeRegistryInterface abstracts the AttributeRegistry.
type AttributeRegistryInterface interface {
	GetClassAttributes(classFQN string) *core.ClassAttributes
	GetAttribute(classFQN, attrName string) *core.ClassAttribute
	HasClass(classFQN string) bool
}

// ModuleRegistryInterface abstracts the ModuleRegistry.
type ModuleRegistryInterface interface {
	GetModulePath(filePath string) string
	ResolveImport(importPath string, fromFile string) (string, bool)
}

// BuiltinRegistryInterface abstracts the BuiltinRegistry.
type BuiltinRegistryInterface interface {
	GetMethodReturnType(typeFQN, methodName string) (string, bool)
	IsBuiltinType(typeFQN string) bool
}

// TypeStore interface for strategies package.
// This interface is satisfied by resolution.TypeStore.
type TypeStore interface {
	Lookup(varName string) core.Type
	CurrentScopeDepth() int
}

// =============================================================================
// STRATEGY REGISTRY
// =============================================================================

// StrategyRegistry manages registered inference strategies.
type StrategyRegistry struct {
	strategies []InferenceStrategy
	sorted     bool
}

// NewStrategyRegistry creates a new StrategyRegistry.
func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make([]InferenceStrategy, 0),
		sorted:     false,
	}
}

// Register adds a strategy to the registry.
func (r *StrategyRegistry) Register(strategy InferenceStrategy) {
	r.strategies = append(r.strategies, strategy)
	r.sorted = false
}

// GetStrategies returns all strategies sorted by priority (descending).
func (r *StrategyRegistry) GetStrategies() []InferenceStrategy {
	if !r.sorted {
		r.sortByPriority()
		r.sorted = true
	}
	return r.strategies
}

// FindStrategy returns the first strategy that can handle the node.
func (r *StrategyRegistry) FindStrategy(node *sitter.Node, ctx *InferenceContext) InferenceStrategy {
	for _, s := range r.GetStrategies() {
		if s.CanHandle(node, ctx) {
			return s
		}
	}
	return nil
}

func (r *StrategyRegistry) sortByPriority() {
	// Simple bubble sort (strategies list is small)
	n := len(r.strategies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if r.strategies[j].Priority() < r.strategies[j+1].Priority() {
				r.strategies[j], r.strategies[j+1] = r.strategies[j+1], r.strategies[j]
			}
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// GetNodeText extracts the text content of an AST node.
func GetNodeText(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	return string(source[node.StartByte():node.EndByte()])
}

// GetChildByType finds the first child with the given type.
func GetChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			return child
		}
	}
	return nil
}

// GetChildrenByType finds all children with the given type.
func GetChildrenByType(node *sitter.Node, nodeType string) []*sitter.Node {
	var children []*sitter.Node
	if node == nil {
		return children
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeType {
			children = append(children, child)
		}
	}
	return children
}
