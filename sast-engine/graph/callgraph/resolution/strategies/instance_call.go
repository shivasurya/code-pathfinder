// Package strategies provides the InstanceCallStrategy for resolving instance method calls.
package strategies

import (
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	sitter "github.com/smacker/go-tree-sitter"
)

// InstanceCallStrategy resolves instance method calls like `obj.method()`.
type InstanceCallStrategy struct {
	BaseStrategy
}

// NewInstanceCallStrategy creates a new InstanceCallStrategy.
func NewInstanceCallStrategy() *InstanceCallStrategy {
	return &InstanceCallStrategy{
		BaseStrategy: NewBaseStrategy("instance_call", 80),
	}
}

// CanHandle returns true for call expressions with attribute access.
// Pattern: identifier.method() or expression.method().
func (s *InstanceCallStrategy) CanHandle(node *sitter.Node, ctx *InferenceContext) bool {
	if node == nil || node.Type() != "call" {
		return false
	}

	// Get the function being called
	funcNode := GetChildByType(node, "attribute")
	if funcNode == nil {
		// Check for direct call: identifier()
		funcNode = GetChildByType(node, "identifier")
		if funcNode == nil {
			return false
		}
		// Direct call handled by different strategy
		return false
	}

	// Must have format: something.method
	return true
}

// Synthesize infers the return type of the method call.
func (s *InstanceCallStrategy) Synthesize(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	// call node structure:
	//   attribute (function)
	//     object
	//     identifier (method name)
	//   argument_list

	attrNode := GetChildByType(node, "attribute")
	if attrNode == nil {
		return &core.AnyType{Reason: "no attribute node"}, 0.0
	}

	// Get receiver (object)
	receiverNode := attrNode.ChildByFieldName("object")
	if receiverNode == nil {
		return &core.AnyType{Reason: "no receiver"}, 0.0
	}

	// Get method name
	methodNode := attrNode.ChildByFieldName("attribute")
	if methodNode == nil {
		return &core.AnyType{Reason: "no method name"}, 0.0
	}
	methodName := GetNodeText(methodNode, ctx.SourceCode)

	// Infer receiver type
	receiverType, receiverConf := s.inferReceiverType(receiverNode, ctx)
	if core.IsAnyType(receiverType) {
		return receiverType, 0.0
	}

	// Look up method return type
	returnType, methodConf := s.lookupMethodReturnType(receiverType, methodName, ctx)

	return returnType, core.CombineConfidence(receiverConf, methodConf)
}

// inferReceiverType infers the type of the receiver object.
func (s *InstanceCallStrategy) inferReceiverType(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	switch node.Type() {
	case "identifier":
		varName := GetNodeText(node, ctx.SourceCode)

		// Check for 'self'
		if varName == "self" && ctx.SelfType != nil {
			return ctx.SelfType, core.ConfidenceScore(core.ConfidenceAnnotation)
		}

		// Look up in store
		if typ := ctx.Store.Lookup(varName); typ != nil {
			return typ, typ.Confidence()
		}

		return &core.AnyType{Reason: "unbound: " + varName}, 0.0

	case "call":
		// Nested call: get_user().name
		// Recursively infer
		return s.Synthesize(node, ctx)

	case "attribute":
		// Chained: obj.attr.method()
		// This will be handled by ChainStrategy in PR-04
		return &core.AnyType{Reason: "chain - defer to ChainStrategy"}, 0.0

	default:
		return &core.AnyType{Reason: "unhandled receiver: " + node.Type()}, 0.0
	}
}

// lookupMethodReturnType finds the return type of a method.
func (s *InstanceCallStrategy) lookupMethodReturnType(receiverType core.Type, methodName string, ctx *InferenceContext) (core.Type, float64) {
	ct, ok := core.ExtractConcreteType(receiverType)
	if !ok {
		return &core.AnyType{Reason: "receiver not concrete"}, 0.0
	}

	classFQN := ct.FQN()

	// Check builtin registry first
	if ctx.BuiltinRegistry != nil && ctx.BuiltinRegistry.IsBuiltinType(classFQN) {
		if retType, found := ctx.BuiltinRegistry.GetMethodReturnType(classFQN, methodName); found {
			return core.NewConcreteType(retType, core.ConfidenceScore(core.ConfidenceReturnType)),
				core.ConfidenceScore(core.ConfidenceReturnType)
		}
	}

	// Check attribute registry for user-defined classes
	if ctx.AttrRegistry != nil {
		classAttrs := ctx.AttrRegistry.GetClassAttributes(classFQN)
		if classAttrs != nil {
			// Check if method exists in the class
			if slices.Contains(classAttrs.Methods, classFQN+"."+methodName) {
				// Method found - need to look up return type
				// For now, return the class type for self-returning methods
				// This is a heuristic; full implementation uses return type tracking
				return receiverType, core.ConfidenceScore(core.ConfidenceFluentHeuristic)
			}
		}
	}

	// Method not found - still return receiver type for fluent interface heuristic
	return &core.AnyType{Reason: "method not found: " + methodName}, 0.0
}

// Check verifies if the method call can return the expected type.
func (s *InstanceCallStrategy) Check(node *sitter.Node, expectedType core.Type, ctx *InferenceContext) bool {
	inferredType, confidence := s.Synthesize(node, ctx)
	if confidence < 0.5 {
		return false
	}
	return inferredType.Equals(expectedType)
}
