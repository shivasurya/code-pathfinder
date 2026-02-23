// Package strategies provides the SelfReferenceStrategy for resolving self.attr patterns.
package strategies

import (
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	sitter "github.com/smacker/go-tree-sitter"
)

// SelfReferenceStrategy resolves self.attribute access patterns.
type SelfReferenceStrategy struct {
	BaseStrategy
}

// NewSelfReferenceStrategy creates a new SelfReferenceStrategy.
func NewSelfReferenceStrategy() *SelfReferenceStrategy {
	return &SelfReferenceStrategy{
		BaseStrategy: NewBaseStrategy("self_reference", 90), // Higher priority than instance_call
	}
}

// CanHandle returns true for attribute access on 'self'.
func (s *SelfReferenceStrategy) CanHandle(node *sitter.Node, ctx *InferenceContext) bool {
	if node == nil || node.Type() != "attribute" {
		return false
	}

	// Get the object being accessed
	objNode := node.ChildByFieldName("object")
	if objNode == nil {
		return false
	}

	// Check if it's 'self'
	if objNode.Type() == "identifier" {
		varName := GetNodeText(objNode, ctx.SourceCode)
		return varName == "self" && ctx.SelfType != nil
	}

	return false
}

// Synthesize infers the type of self.attribute.
func (s *SelfReferenceStrategy) Synthesize(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	if ctx.SelfType == nil {
		return &core.AnyType{Reason: "no self type"}, 0.0
	}

	// Get attribute name
	attrNode := node.ChildByFieldName("attribute")
	if attrNode == nil {
		return &core.AnyType{Reason: "no attribute"}, 0.0
	}
	attrName := GetNodeText(attrNode, ctx.SourceCode)

	// Get class FQN from self type
	ct, ok := core.ExtractConcreteType(ctx.SelfType)
	if !ok {
		return &core.AnyType{Reason: "self not concrete"}, 0.0
	}
	classFQN := ct.FQN()

	// Look up attribute in registry
	if ctx.AttrRegistry != nil {
		attr := ctx.AttrRegistry.GetAttribute(classFQN, attrName)
		if attr != nil && attr.Type != nil {
			return core.NewConcreteType(attr.Type.TypeFQN, float64(attr.Type.Confidence)),
				float64(attr.Type.Confidence)
		}

		// Check if it's a method
		classAttrs := ctx.AttrRegistry.GetClassAttributes(classFQN)
		if classAttrs != nil {
			if slices.Contains(classAttrs.Methods, classFQN+"."+attrName) {
				// It's a method - return a callable type
				// For simplicity, return FunctionType with unknown signature
				return &core.FunctionType{
					Parameters: nil, // Unknown
					ReturnType: nil, // Unknown until called
				}, core.ConfidenceScore(core.ConfidenceAttribute)
			}
		}
	}

	// Attribute not found in registry
	// Could be dynamically set or inherited - return Any
	return &core.AnyType{Reason: "attribute not found: " + attrName}, 0.0
}

// Check verifies if self.attr can produce the expected type.
func (s *SelfReferenceStrategy) Check(node *sitter.Node, expectedType core.Type, ctx *InferenceContext) bool {
	inferredType, confidence := s.Synthesize(node, ctx)
	if confidence < 0.5 {
		return false
	}
	return inferredType.Equals(expectedType)
}
