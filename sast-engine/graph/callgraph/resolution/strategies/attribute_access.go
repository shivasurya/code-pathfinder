// Package strategies provides AttributeAccessStrategy for general attribute access.
package strategies

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// AttributeAccessStrategy resolves obj.attr patterns (non-self).
type AttributeAccessStrategy struct {
	BaseStrategy
}

// NewAttributeAccessStrategy creates a new AttributeAccessStrategy.
func NewAttributeAccessStrategy() *AttributeAccessStrategy {
	return &AttributeAccessStrategy{
		BaseStrategy: NewBaseStrategy("attribute_access", 70),
	}
}

// CanHandle returns true for attribute access that isn't self.
func (s *AttributeAccessStrategy) CanHandle(node *sitter.Node, ctx *InferenceContext) bool {
	if node == nil || node.Type() != "attribute" {
		return false
	}

	// Skip if this is self.attr (handled by SelfReferenceStrategy)
	objNode := node.ChildByFieldName("object")
	if objNode == nil {
		return false
	}

	if objNode.Type() == "identifier" {
		varName := GetNodeText(objNode, ctx.SourceCode)
		if varName == "self" && ctx.SelfType != nil {
			return false // Let SelfReferenceStrategy handle it
		}
	}

	return true
}

// Synthesize infers the type of obj.attr.
func (s *AttributeAccessStrategy) Synthesize(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	// Get the object
	objNode := node.ChildByFieldName("object")
	if objNode == nil {
		return &core.AnyType{Reason: "no object"}, 0.0
	}

	// Get the attribute name
	attrNode := node.ChildByFieldName("attribute")
	if attrNode == nil {
		return &core.AnyType{Reason: "no attribute"}, 0.0
	}
	attrName := GetNodeText(attrNode, ctx.SourceCode)

	// Infer object type
	objType, objConf := s.inferObjectType(objNode, ctx)
	if core.IsAnyType(objType) {
		return objType, 0.0
	}

	// Look up attribute type
	attrType, attrConf := s.lookupAttributeType(objType, attrName, ctx)

	return attrType, core.CombineConfidence(objConf, attrConf)
}

// inferObjectType determines the type of the object being accessed.
func (s *AttributeAccessStrategy) inferObjectType(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	switch node.Type() {
	case "identifier":
		varName := GetNodeText(node, ctx.SourceCode)
		if typ := ctx.Store.Lookup(varName); typ != nil {
			return typ, typ.Confidence()
		}
		return &core.AnyType{Reason: "unbound: " + varName}, 0.0

	case "attribute":
		// Chained: obj.a.b
		// Recursively synthesize
		return s.Synthesize(node, ctx)

	case "call":
		// Result of call: get_user().name
		// Defer to call resolution
		return &core.AnyType{Reason: "call result - needs call resolution"}, 0.0

	default:
		return &core.AnyType{Reason: "unhandled object: " + node.Type()}, 0.0
	}
}

// lookupAttributeType finds the type of an attribute on an object.
func (s *AttributeAccessStrategy) lookupAttributeType(objType core.Type, attrName string, ctx *InferenceContext) (core.Type, float64) {
	ct, ok := core.ExtractConcreteType(objType)
	if !ok {
		return &core.AnyType{Reason: "object not concrete"}, 0.0
	}

	classFQN := ct.FQN()

	// Check attribute registry
	if ctx.AttrRegistry != nil {
		attr := ctx.AttrRegistry.GetAttribute(classFQN, attrName)
		if attr != nil && attr.Type != nil {
			return core.NewConcreteType(attr.Type.TypeFQN, float64(attr.Type.Confidence)),
				float64(attr.Type.Confidence)
		}
	}

	// Attribute not found
	return &core.AnyType{Reason: "attribute not found: " + attrName}, 0.0
}

// Check verifies if obj.attr can produce the expected type.
func (s *AttributeAccessStrategy) Check(node *sitter.Node, expectedType core.Type, ctx *InferenceContext) bool {
	inferredType, confidence := s.Synthesize(node, ctx)
	if confidence < 0.5 {
		return false
	}
	return inferredType.Equals(expectedType)
}
