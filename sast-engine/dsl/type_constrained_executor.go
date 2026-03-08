package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// InheritanceChecker provides inheritance-aware type checking via MRO.
// Implemented by cgregistry.ThirdPartyRegistryRemote (avoids import cycle).
type InheritanceChecker interface {
	HasModule(moduleName string) bool
	IsSubclassSimple(moduleName, className, parentFQN string) bool
	GetClassMRO(moduleName, className string) []string
}

// TypeConstrainedCallExecutor executes type_constrained_call IR against callgraph.
// It finds call sites where the receiver variable has a specific inferred type,
// with inheritance-aware matching via MRO lookups.
type TypeConstrainedCallExecutor struct {
	IR               *TypeConstrainedCallIR
	CallGraph        *core.CallGraph
	ThirdPartyRemote InheritanceChecker
}

// Execute finds all call sites matching the type-constrained pattern.
//
// Algorithm:
//  1. Iterate all call sites
//  2. Check method name match (cheapest check)
//  3. Check receiver type via type inference, FQN bridge, or fallback
//  4. Check argument constraints if type matches
//  5. Return matching call sites as detections
func (e *TypeConstrainedCallExecutor) Execute() []DataflowDetection {
	var detections []DataflowDetection
	minConf := e.IR.MinConfidence
	if minConf <= 0 {
		minConf = 0.5
	}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for i := range callSites {
			cs := &callSites[i]
			if e.matchesCallSite(cs, minConf) {
				conf := float64(cs.TypeConfidence)
				if conf == 0 {
					conf = 0.7 // FQN bridge confidence
				}
				detections = append(detections, DataflowDetection{
					FunctionFQN:     functionFQN,
					SourceLine:      cs.Location.Line,
					SinkLine:        cs.Location.Line,
					SinkCall:        cs.Target,
					Confidence:      conf,
					Scope:           "local",
					MatchedCallSite: cs,
				})
			}
		}
	}

	return detections
}

// matchesCallSite checks if a call site matches the type-constrained pattern.
func (e *TypeConstrainedCallExecutor) matchesCallSite(cs *core.CallSite, minConf float64) bool {
	// Step 1: Method name match (cheapest check)
	if !e.matchesMethodName(cs.Target) {
		return false
	}

	// Step 2: Type inference match
	if cs.ResolvedViaTypeInference && cs.TypeConfidence >= float32(minConf) {
		if e.matchesAnyReceiverType(cs.InferredType) {
			return e.matchesArgs(cs)
		}
	}

	// Step 2b: FQN-to-receiver bridge
	if cs.TargetFQN != "" {
		derivedReceiver := deriveReceiverFromFQN(cs.TargetFQN, cs.Target)
		if derivedReceiver != "" && e.matchesAnyReceiverType(derivedReceiver) {
			return e.matchesArgs(cs)
		}
	}

	// Step 3: Fallback behavior
	switch e.IR.FallbackMode {
	case "name":
		return e.matchesArgs(cs)
	default:
		return false
	}
}

// matchesMethodName checks if the call target ends with any expected method name.
func (e *TypeConstrainedCallExecutor) matchesMethodName(target string) bool {
	methodNames := e.IR.GetEffectiveMethodNames()
	if len(methodNames) == 0 {
		return true // No method constraint
	}

	for _, methodName := range methodNames {
		if target == methodName {
			return true
		}
		if strings.HasSuffix(target, "."+methodName) {
			return true
		}
	}
	return false
}

// matchesAnyReceiverType checks if an actual type matches any of the configured receiver types or patterns.
func (e *TypeConstrainedCallExecutor) matchesAnyReceiverType(actual string) bool {
	if actual == "" {
		return false
	}

	// Check exact receiver types
	for _, rt := range e.IR.GetEffectiveReceiverTypes() {
		if matchesReceiverType(actual, rt, e.ThirdPartyRemote) {
			return true
		}
	}

	// Check wildcard receiver patterns
	for _, rp := range e.IR.ReceiverPatterns {
		if matchesReceiverType(actual, rp, e.ThirdPartyRemote) {
			return true
		}
	}

	return false
}

// matchesArgs checks argument constraints on a call site.
func (e *TypeConstrainedCallExecutor) matchesArgs(cs *core.CallSite) bool {
	return MatchesArguments(cs, e.IR.PositionalArgs, e.IR.KeywordArgs)
}

// deriveReceiverFromFQN extracts receiver module/class from TargetFQN.
// Examples: "os.system" → "os", "pickle.loads" → "pickle".
func deriveReceiverFromFQN(targetFQN, target string) string {
	// Extract method name from target
	methodName := target
	if idx := strings.LastIndex(target, "."); idx >= 0 {
		methodName = target[idx+1:]
	}

	if strings.HasSuffix(targetFQN, "."+methodName) {
		return strings.TrimSuffix(targetFQN, "."+methodName)
	}
	return ""
}

// matchesReceiverType checks if an actual inferred type matches the expected pattern,
// with support for exact match, short name match, wildcard match, and inheritance-aware
// MRO-based matching via the third-party registry.
//
// Match priority (most specific to least):
//  1. Exact FQN: actual == pattern
//  2. Short name: "View" matches "django.views.View"
//  3. Wildcard: "*Cursor", "sqlite3.*"
//  4. Inheritance: IsSubclass via MRO (requires CDN data)
func matchesReceiverType(actual, pattern string, thirdPartyRemote InheritanceChecker) bool {
	if actual == "" || pattern == "" {
		return false
	}

	// 1. Exact FQN match
	if actual == pattern {
		return true
	}

	// 2. Short name match (pattern has no dots)
	if !strings.Contains(pattern, ".") {
		if strings.HasSuffix(actual, "."+pattern) {
			return true
		}
	}

	// 3. Wildcard match
	if strings.Contains(pattern, "*") {
		if matchesWildcardType(actual, pattern) {
			return true
		}
	}

	// 4. Inheritance-aware match via MRO
	if thirdPartyRemote != nil {
		moduleName, className := splitTypeModuleAndClass(actual)
		if moduleName != "" && className != "" && thirdPartyRemote.HasModule(moduleName) {
			// Direct IsSubclass check (FQN pattern)
			if thirdPartyRemote.IsSubclassSimple(moduleName, className, pattern) {
				return true
			}
			// Short name expansion: "View" → check MRO for any "*.View"
			if !strings.Contains(pattern, ".") {
				for _, ancestor := range thirdPartyRemote.GetClassMRO(moduleName, className) {
					if strings.HasSuffix(ancestor, "."+pattern) {
						return true
					}
				}
			}
		}
	}

	return false
}

// matchesWildcardType performs simple wildcard matching on type FQNs.
func matchesWildcardType(actual, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(actual, strings.Trim(pattern, "*"))
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(actual, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(actual, pattern[:len(pattern)-1])
	}
	return actual == pattern
}

// splitTypeModuleAndClass splits "django.views.View" into module="django", class="views.View".
func splitTypeModuleAndClass(fqn string) (string, string) {
	if idx := strings.Index(fqn, "."); idx > 0 {
		return fqn[:idx], fqn[idx+1:]
	}
	return fqn, ""
}

// TypeConstrainedAttributeExecutor executes type_constrained_attribute IR.
// Matches attribute access patterns on typed variables.
type TypeConstrainedAttributeExecutor struct {
	IR               *TypeConstrainedAttributeIR
	CallGraph        *core.CallGraph
	ThirdPartyRemote InheritanceChecker
}

// Execute finds call sites matching typed attribute access patterns.
//
// Looks for call targets like "self.attr" or "var.attr" where the variable's
// inferred type matches ReceiverType and the attribute matches AttributeName.
func (e *TypeConstrainedAttributeExecutor) Execute() []DataflowDetection {
	var detections []DataflowDetection
	minConf := e.IR.MinConfidence
	if minConf <= 0 {
		minConf = 0.5
	}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			if e.matchesAttributeAccess(&cs, minConf) {
				detections = append(detections, DataflowDetection{
					FunctionFQN: functionFQN,
					SourceLine:  cs.Location.Line,
					SinkLine:    cs.Location.Line,
					SinkCall:    cs.Target,
					Confidence:  float64(cs.TypeConfidence),
					Scope:       "local",
				})
			}
		}
	}

	return detections
}

// matchesAttributeAccess checks if a call site represents a typed attribute access.
func (e *TypeConstrainedAttributeExecutor) matchesAttributeAccess(cs *core.CallSite, minConf float64) bool {
	// Check if target contains the attribute name
	attrName := e.IR.AttributeName
	if attrName == "" {
		return false
	}

	// Target should end with ".attributeName" (e.g., "request.GET", "self.request.GET")
	if !strings.HasSuffix(cs.Target, "."+attrName) {
		return false
	}

	// If resolved via type inference, check the receiver type
	if cs.ResolvedViaTypeInference && cs.TypeConfidence >= float32(minConf) {
		return matchesReceiverType(cs.InferredType, e.IR.ReceiverType, e.ThirdPartyRemote)
	}

	// Fallback
	switch e.IR.FallbackMode {
	case "name":
		return true
	default:
		return false
	}
}
