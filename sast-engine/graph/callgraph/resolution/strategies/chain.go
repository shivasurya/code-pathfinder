// Package strategies provides ChainStrategy for deep chain resolution.
package strategies

import (
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	sitter "github.com/smacker/go-tree-sitter"
)

// ChainStep represents a single step in a method/attribute chain.
type ChainStep struct {
	Name string
	Kind ChainStepKind
	Node *sitter.Node
	Args []*sitter.Node // For call steps
}

// ChainStepKind identifies the type of chain step.
type ChainStepKind int

const (
	// StepIdentifier represents a starting variable: obj.
	StepIdentifier ChainStepKind = iota
	// StepAttribute represents attribute access: .attr.
	StepAttribute
	// StepMethodCall represents method call: .method().
	StepMethodCall
	// StepInstantiation represents class instantiation: Class().
	StepInstantiation
)

// ChainStrategy resolves deep chains like a.b.c.method() or Builder().x().y().
type ChainStrategy struct {
	BaseStrategy
}

// NewChainStrategy creates a new ChainStrategy.
func NewChainStrategy() *ChainStrategy {
	return &ChainStrategy{
		BaseStrategy: NewBaseStrategy("chain", 85), // Higher than instance_call
	}
}

// CanHandle returns true for chains with 2+ levels.
func (s *ChainStrategy) CanHandle(node *sitter.Node, ctx *InferenceContext) bool {
	steps := s.parseChain(node, ctx.SourceCode)
	return len(steps) >= 2
}

// Synthesize walks the chain and infers the final type.
func (s *ChainStrategy) Synthesize(node *sitter.Node, ctx *InferenceContext) (core.Type, float64) {
	steps := s.parseChain(node, ctx.SourceCode)
	if len(steps) == 0 {
		return &core.AnyType{Reason: "empty chain"}, 0.0
	}

	// Enforce max depth
	if len(steps) > MaxChainDepth {
		return &core.AnyType{Reason: "chain too deep"}, 0.0
	}

	// Start with first step
	currentType, confidence := s.resolveFirstStep(steps[0], ctx)
	if core.IsAnyType(currentType) {
		return currentType, 0.0
	}

	// Walk through remaining steps
	for i := 1; i < len(steps); i++ {
		step := steps[i]
		var stepConf float64

		currentType, stepConf = s.resolveStep(currentType, step, ctx)
		confidence = core.CombineConfidence(confidence, stepConf)

		// Early exit if confidence too low
		if confidence < MinConfidenceThreshold {
			return &core.AnyType{Reason: "confidence too low in chain"}, 0.0
		}

		if core.IsAnyType(currentType) {
			return currentType, 0.0
		}
	}

	return currentType, confidence
}

// parseChain extracts the chain steps from an AST node.
func (s *ChainStrategy) parseChain(node *sitter.Node, source []byte) []*ChainStep {
	var steps []*ChainStep

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		switch n.Type() {
		case "call":
			// Check if this is a method call or instantiation
			funcNode := n.ChildByFieldName("function")
			if funcNode == nil {
				return
			}

			if funcNode.Type() == "attribute" {
				// Method call: obj.method()
				walk(funcNode.ChildByFieldName("object"))

				methodNode := funcNode.ChildByFieldName("attribute")
				if methodNode != nil {
					steps = append(steps, &ChainStep{
						Name: GetNodeText(methodNode, source),
						Kind: StepMethodCall,
						Node: n,
						Args: s.getCallArgs(n),
					})
				}
			} else if funcNode.Type() == "identifier" {
				// Could be Class() instantiation or function call
				name := GetNodeText(funcNode, source)
				// Heuristic: capitalized = class instantiation
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					steps = append(steps, &ChainStep{
						Name: name,
						Kind: StepInstantiation,
						Node: n,
						Args: s.getCallArgs(n),
					})
				} else {
					// Function call - start of chain
					steps = append(steps, &ChainStep{
						Name: name,
						Kind: StepMethodCall,
						Node: n,
						Args: s.getCallArgs(n),
					})
				}
			}

		case "attribute":
			// Attribute access: obj.attr
			walk(n.ChildByFieldName("object"))

			attrNode := n.ChildByFieldName("attribute")
			if attrNode != nil {
				steps = append(steps, &ChainStep{
					Name: GetNodeText(attrNode, source),
					Kind: StepAttribute,
					Node: n,
				})
			}

		case "identifier":
			// Starting variable
			steps = append(steps, &ChainStep{
				Name: GetNodeText(n, source),
				Kind: StepIdentifier,
				Node: n,
			})
		}
	}

	walk(node)
	return steps
}

// getCallArgs extracts argument nodes from a call.
func (s *ChainStrategy) getCallArgs(callNode *sitter.Node) []*sitter.Node {
	argList := GetChildByType(callNode, "argument_list")
	if argList == nil {
		return nil
	}

	var args []*sitter.Node
	for i := 0; i < int(argList.ChildCount()); i++ {
		child := argList.Child(i)
		if child.Type() != "(" && child.Type() != ")" && child.Type() != "," {
			args = append(args, child)
		}
	}
	return args
}

// resolveFirstStep resolves the starting point of the chain.
func (s *ChainStrategy) resolveFirstStep(step *ChainStep, ctx *InferenceContext) (core.Type, float64) {
	switch step.Kind {
	case StepIdentifier:
		// Variable lookup
		if step.Name == "self" && ctx.SelfType != nil {
			return ctx.SelfType, core.ConfidenceScore(core.ConfidenceAnnotation)
		}
		if typ := ctx.Store.Lookup(step.Name); typ != nil {
			return typ, typ.Confidence()
		}
		return &core.AnyType{Reason: "unbound: " + step.Name}, 0.0

	case StepInstantiation:
		// Class instantiation: ClassName()
		return s.resolveInstantiation(step.Name, ctx)

	case StepMethodCall:
		// Function call (not method)
		return s.resolveFunctionCall(step.Name, ctx)

	default:
		return &core.AnyType{Reason: "invalid first step"}, 0.0
	}
}

// resolveInstantiation resolves ClassName() to the class type.
func (s *ChainStrategy) resolveInstantiation(className string, ctx *InferenceContext) (core.Type, float64) {
	// Try to find the class in the registry
	if ctx.AttrRegistry != nil {
		// Try with module prefix
		if ctx.ModuleRegistry != nil {
			modulePath := ctx.ModuleRegistry.GetModulePath(ctx.FilePath)
			fullFQN := modulePath + "." + className

			if ctx.AttrRegistry.HasClass(fullFQN) {
				return core.NewConcreteType(fullFQN, core.ConfidenceScore(core.ConfidenceConstructor)),
					core.ConfidenceScore(core.ConfidenceConstructor)
			}
		}

		// Try just the class name (might be imported)
		if ctx.AttrRegistry.HasClass(className) {
			return core.NewConcreteType(className, core.ConfidenceScore(core.ConfidenceConstructor)),
				core.ConfidenceScore(core.ConfidenceConstructor)
		}
	}

	// Class not found in registry - still create type optimistically
	return core.NewConcreteType(className, core.ConfidenceScore(core.ConfidenceFluentHeuristic)),
		core.ConfidenceScore(core.ConfidenceFluentHeuristic)
}

// resolveFunctionCall resolves a function call's return type.
func (s *ChainStrategy) resolveFunctionCall(funcName string, _ *InferenceContext) (core.Type, float64) {
	// Look up in return type registry
	// For now, return Any - full implementation would check return type registry
	return &core.AnyType{Reason: "function return type unknown: " + funcName}, 0.0
}

// resolveStep resolves a single step in the chain.
func (s *ChainStrategy) resolveStep(currentType core.Type, step *ChainStep, ctx *InferenceContext) (core.Type, float64) {
	ct, ok := core.ExtractConcreteType(currentType)
	if !ok {
		return &core.AnyType{Reason: "current type not concrete"}, 0.0
	}

	classFQN := ct.FQN()

	switch step.Kind {
	case StepAttribute:
		return s.resolveAttributeStep(classFQN, step.Name, ctx)

	case StepMethodCall:
		return s.resolveMethodStep(classFQN, step.Name, currentType, ctx)

	default:
		return &core.AnyType{Reason: "invalid step kind"}, 0.0
	}
}

// resolveAttributeStep resolves .attr access.
func (s *ChainStrategy) resolveAttributeStep(classFQN, attrName string, ctx *InferenceContext) (core.Type, float64) {
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

// resolveMethodStep resolves .method() call.
func (s *ChainStrategy) resolveMethodStep(classFQN, methodName string, currentType core.Type, ctx *InferenceContext) (core.Type, float64) {
	// Check builtin registry
	if ctx.BuiltinRegistry != nil && ctx.BuiltinRegistry.IsBuiltinType(classFQN) {
		if retType, found := ctx.BuiltinRegistry.GetMethodReturnType(classFQN, methodName); found {
			return core.NewConcreteType(retType, core.ConfidenceScore(core.ConfidenceReturnType)),
				core.ConfidenceScore(core.ConfidenceReturnType)
		}
	}

	// Check if method exists in class
	if ctx.AttrRegistry != nil {
		classAttrs := ctx.AttrRegistry.GetClassAttributes(classFQN)
		if classAttrs != nil {
			if slices.Contains(classAttrs.Methods, classFQN+"."+methodName) {
				// Method found - use fluent heuristic (returns self)
				return currentType, core.ConfidenceScore(core.ConfidenceFluentHeuristic)
			}
		}
	}

	// Method not found - still apply fluent heuristic for builder patterns
	// (common pattern: methods return self for chaining)
	return currentType, core.ConfidenceScore(core.ConfidenceFluentHeuristic)
}

// Check verifies if the chain produces the expected type.
func (s *ChainStrategy) Check(node *sitter.Node, expectedType core.Type, ctx *InferenceContext) bool {
	inferredType, confidence := s.Synthesize(node, ctx)
	if confidence < 0.5 {
		return false
	}
	return inferredType.Equals(expectedType)
}

// =============================================================================
// CHAIN DEPTH CONFIGURATION
// =============================================================================

const (
	// MaxChainDepth is the maximum supported chain depth.
	MaxChainDepth = 10

	// MinConfidenceThreshold is the minimum confidence to continue resolving.
	MinConfidenceThreshold = 0.3
)
