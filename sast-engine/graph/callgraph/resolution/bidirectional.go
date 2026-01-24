// Package resolution provides bidirectional type inference.
package resolution

import (
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution/strategies"
)

// BidirectionalInferencer orchestrates type inference using strategies.
type BidirectionalInferencer struct {
	strategies *strategies.StrategyRegistry
	cache      *TypeCache
	mutex      sync.RWMutex

	// Integration with existing system
	attrRegistry    strategies.AttributeRegistryInterface
	moduleRegistry  strategies.ModuleRegistryInterface
	builtinRegistry strategies.BuiltinRegistryInterface
}

// NewBidirectionalInferencer creates a new BidirectionalInferencer.
func NewBidirectionalInferencer(
	attrReg strategies.AttributeRegistryInterface,
	modReg strategies.ModuleRegistryInterface,
	builtinReg strategies.BuiltinRegistryInterface,
	cacheCapacity int,
) *BidirectionalInferencer {
	bi := &BidirectionalInferencer{
		strategies:      strategies.NewStrategyRegistry(),
		cache:           NewTypeCache(cacheCapacity),
		attrRegistry:    attrReg,
		moduleRegistry:  modReg,
		builtinRegistry: builtinReg,
	}

	// Register default strategies (PR-03, PR-04)
	bi.RegisterStrategy(strategies.NewSelfReferenceStrategy())    // Priority: 90
	bi.RegisterStrategy(strategies.NewChainStrategy())             // Priority: 85 (PR-04)
	bi.RegisterStrategy(strategies.NewInstanceCallStrategy())      // Priority: 80
	bi.RegisterStrategy(strategies.NewAttributeAccessStrategy())   // Priority: 70

	return bi
}

// RegisterStrategy adds a strategy to the inferencer.
func (bi *BidirectionalInferencer) RegisterStrategy(strategy strategies.InferenceStrategy) {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()
	bi.strategies.Register(strategy)
}

// InferType infers the type of an AST node using registered strategies.
// This is the main entry point for type inference.
func (bi *BidirectionalInferencer) InferType(
	node *sitter.Node,
	store *TypeStore,
	sourceCode []byte,
	filePath string,
	selfType core.Type,
	classFQN string,
	functionFQN string,
) (core.Type, float64) {
	if node == nil {
		return &core.AnyType{Reason: "nil node"}, 0.0
	}

	// Check cache first
	cacheKey := MakeCacheKey(filePath, int(node.StartPoint().Row), int(node.StartPoint().Column), node.Type())
	if cached, found := bi.cache.Get(cacheKey); found {
		return cached, cached.Confidence()
	}

	// Build context
	ctx := &strategies.InferenceContext{
		SourceCode:      sourceCode,
		FilePath:        filePath,
		Store:           store,
		SelfType:        selfType,
		ClassFQN:        classFQN,
		AttrRegistry:    bi.attrRegistry,
		ModuleRegistry:  bi.moduleRegistry,
		BuiltinRegistry: bi.builtinRegistry,
		FunctionFQN:     functionFQN,
		ScopeDepth:      store.CurrentScopeDepth(),
	}

	// Find and apply strategy
	bi.mutex.RLock()
	strategy := bi.strategies.FindStrategy(node, ctx)
	bi.mutex.RUnlock()

	var typ core.Type
	var confidence float64

	if strategy == nil {
		// Fallback: try to synthesize from node type
		typ, confidence = bi.fallbackSynthesize(node, ctx)
	} else {
		typ, confidence = strategy.Synthesize(node, ctx)
	}

	// Cache result
	bi.cache.Put(cacheKey, typ, filePath)

	return typ, confidence
}

// CheckType verifies if a node can produce an expected type.
func (bi *BidirectionalInferencer) CheckType(
	node *sitter.Node,
	expectedType core.Type,
	store *TypeStore,
	sourceCode []byte,
	filePath string,
	selfType core.Type,
	classFQN string,
) bool {
	if node == nil || expectedType == nil {
		return false
	}

	ctx := &strategies.InferenceContext{
		SourceCode:      sourceCode,
		FilePath:        filePath,
		Store:           store,
		SelfType:        selfType,
		ClassFQN:        classFQN,
		AttrRegistry:    bi.attrRegistry,
		ModuleRegistry:  bi.moduleRegistry,
		BuiltinRegistry: bi.builtinRegistry,
	}

	bi.mutex.RLock()
	strategy := bi.strategies.FindStrategy(node, ctx)
	bi.mutex.RUnlock()

	if strategy == nil {
		return false
	}

	return strategy.Check(node, expectedType, ctx)
}

// fallbackSynthesize attempts basic type inference for unhandled nodes.
func (bi *BidirectionalInferencer) fallbackSynthesize(node *sitter.Node, ctx *strategies.InferenceContext) (core.Type, float64) {
	nodeType := node.Type()

	switch nodeType {
	case "string", "string_content":
		return core.NewConcreteType("builtins.str", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "integer":
		return core.NewConcreteType("builtins.int", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "float":
		return core.NewConcreteType("builtins.float", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "true", "false":
		return core.NewConcreteType("builtins.bool", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "none":
		return &core.NoneType{}, 1.0

	case "list":
		return core.NewConcreteType("builtins.list", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "dictionary":
		return core.NewConcreteType("builtins.dict", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "tuple":
		return core.NewConcreteType("builtins.tuple", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "set":
		return core.NewConcreteType("builtins.set", core.ConfidenceScore(core.ConfidenceLiteral)), core.ConfidenceScore(core.ConfidenceLiteral)

	case "identifier":
		// Look up in store
		varName := strategies.GetNodeText(node, ctx.SourceCode)
		if typ := ctx.Store.Lookup(varName); typ != nil {
			return typ, typ.Confidence()
		}
		return &core.AnyType{Reason: "unbound variable: " + varName}, 0.0

	default:
		return &core.AnyType{Reason: "unhandled node type: " + nodeType}, 0.0
	}
}

// InvalidateFile clears cached types for a modified file.
func (bi *BidirectionalInferencer) InvalidateFile(filePath string) int {
	return bi.cache.InvalidateFile(filePath)
}

// CacheStats returns cache hit/miss statistics.
func (bi *BidirectionalInferencer) CacheStats() (hits, misses int64, size int) {
	return bi.cache.Stats()
}

// =============================================================================
// INTEGRATION ADAPTER
// =============================================================================

// TypeStoreAdapter adapts TypeStore to strategies.InferenceContext.
// This ensures the interface contract is maintained.
type TypeStoreAdapter struct {
	*TypeStore
}

// Ensure TypeStore is compatible with strategies package.
func (ts *TypeStore) AsInterface() *TypeStore {
	return ts
}
