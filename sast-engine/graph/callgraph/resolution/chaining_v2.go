// Package resolution provides enhanced chain resolution.
package resolution

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution/strategies"
)

// ResolveInlineInstantiation resolves ClassName().method() patterns.
// Returns the resolved class type.
func ResolveInlineInstantiation(
	callNode *sitter.Node,
	sourceCode []byte,
	attrRegistry strategies.AttributeRegistryInterface,
	moduleRegistry strategies.ModuleRegistryInterface,
	filePath string,
) (core.Type, float64) {
	chainStrategy := strategies.NewChainStrategy()

	ctx := &strategies.InferenceContext{
		SourceCode:     sourceCode,
		FilePath:       filePath,
		Store:          NewTypeStore(),
		AttrRegistry:   attrRegistry,
		ModuleRegistry: moduleRegistry,
	}

	if !chainStrategy.CanHandle(callNode, ctx) {
		return &core.AnyType{Reason: "not a chain"}, 0.0
	}

	return chainStrategy.Synthesize(callNode, ctx)
}

// ResolveDeepAttributeChain resolves self.a.b.c.method() patterns.
// Takes the chain as a slice of attribute names.
func ResolveDeepAttributeChain(
	attributeNames []string,
	startingType core.Type,
	attrRegistry strategies.AttributeRegistryInterface,
) (core.Type, float64) {
	if len(attributeNames) == 0 {
		return startingType, 1.0
	}

	currentType := startingType
	confidence := 1.0

	for _, attrName := range attributeNames {
		ct, ok := core.ExtractConcreteType(currentType)
		if !ok {
			return &core.AnyType{Reason: "type not concrete at: " + attrName}, 0.0
		}

		classFQN := ct.FQN()

		// Look up attribute
		if attrRegistry != nil {
			attr := attrRegistry.GetAttribute(classFQN, attrName)
			if attr != nil && attr.Type != nil {
				currentType = core.NewConcreteType(attr.Type.TypeFQN, float64(attr.Type.Confidence))
				confidence = core.CombineConfidence(confidence, float64(attr.Type.Confidence))
				continue
			}
		}

		// Attribute not found
		return &core.AnyType{Reason: "attribute not found: " + attrName}, 0.0
	}

	return currentType, confidence
}

// ChainResolver provides a fluent interface for chain resolution.
type ChainResolver struct {
	store        *TypeStore
	attrRegistry strategies.AttributeRegistryInterface
	builtinReg   strategies.BuiltinRegistryInterface
	moduleReg    strategies.ModuleRegistryInterface
	filePath     string
	sourceCode   []byte
	selfType     core.Type
	classFQN     string
}

// NewChainResolver creates a new ChainResolver.
func NewChainResolver(
	attrReg strategies.AttributeRegistryInterface,
	builtinReg strategies.BuiltinRegistryInterface,
	moduleReg strategies.ModuleRegistryInterface,
) *ChainResolver {
	return &ChainResolver{
		store:        NewTypeStore(),
		attrRegistry: attrReg,
		builtinReg:   builtinReg,
		moduleReg:    moduleReg,
	}
}

// WithContext sets the resolution context.
func (r *ChainResolver) WithContext(filePath string, sourceCode []byte) *ChainResolver {
	r.filePath = filePath
	r.sourceCode = sourceCode
	return r
}

// WithSelf sets the self type for method resolution.
func (r *ChainResolver) WithSelf(selfType core.Type, classFQN string) *ChainResolver {
	r.selfType = selfType
	r.classFQN = classFQN
	return r
}

// WithVariable registers a known variable.
func (r *ChainResolver) WithVariable(name string, typ core.Type) *ChainResolver {
	r.store.Set(name, typ, core.ConfidenceAssignment, r.filePath, 0, 0)
	return r
}

// Resolve resolves a chain call node.
func (r *ChainResolver) Resolve(node *sitter.Node) (core.Type, float64) {
	chainStrategy := strategies.NewChainStrategy()

	ctx := &strategies.InferenceContext{
		SourceCode:      r.sourceCode,
		FilePath:        r.filePath,
		Store:           r.store,
		SelfType:        r.selfType,
		ClassFQN:        r.classFQN,
		AttrRegistry:    r.attrRegistry,
		ModuleRegistry:  r.moduleReg,
		BuiltinRegistry: r.builtinReg,
	}

	return chainStrategy.Synthesize(node, ctx)
}
