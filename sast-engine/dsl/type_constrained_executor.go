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
			if matchMethod := e.matchesCallSite(cs, minConf); matchMethod != "" {
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
					MatchMethod:     matchMethod,
				})
			}
		}
	}

	return detections
}

// matchesCallSite checks if a call site matches the type-constrained pattern.
// Returns the match method string ("type_inference", "fqn_bridge", "fqn_prefix",
// "name_fallback") or empty string if no match.
func (e *TypeConstrainedCallExecutor) matchesCallSite(cs *core.CallSite, minConf float64) string {
	// Step 1: Method name match (cheapest check)
	if !e.matchesMethodName(cs.Target) {
		return ""
	}

	// Step 2: Type inference match
	if cs.ResolvedViaTypeInference && cs.TypeConfidence >= float32(minConf) {
		if e.matchesAnyReceiverType(cs.InferredType) {
			if e.matchesArgs(cs) {
				return "type_inference"
			}
			return ""
		}
	}

	// Step 2b: FQN-to-receiver bridge
	if cs.TargetFQN != "" {
		derivedReceiver := deriveReceiverFromFQN(cs.TargetFQN, cs.Target)
		if derivedReceiver != "" && e.matchesAnyReceiverType(derivedReceiver) {
			if e.matchesArgs(cs) {
				return "fqn_bridge"
			}
			return ""
		}

		// Step 2c: FQN prefix match — check if TargetFQN starts with a receiver type.
		// Handles cases like TargetFQN="flask.request.args.get" matching "flask.request".
		if e.fqnPrefixMatchesReceiver(cs.TargetFQN) {
			if e.matchesArgs(cs) {
				return "fqn_prefix"
			}
			return ""
		}
	}

	// Step 3: Fallback behavior
	switch e.IR.FallbackMode {
	case "name":
		if e.matchesArgs(cs) {
			return "name_fallback"
		}
		return ""
	default:
		return ""
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

// fqnPrefixMatchesReceiver checks if the TargetFQN starts with any configured receiver type.
// This handles module-level proxies like flask.request (FQN "flask.request.args.get")
// matching against receiver type "flask.request", and case-insensitive class name matching
// where "flask.request" proxies "flask.Request".
func (e *TypeConstrainedCallExecutor) fqnPrefixMatchesReceiver(targetFQN string) bool {
	fqnLower := strings.ToLower(targetFQN)
	for _, rt := range e.IR.GetEffectiveReceiverTypes() {
		rtLower := strings.ToLower(rt)
		if strings.HasPrefix(fqnLower, rtLower+".") {
			return true
		}
	}
	for _, rp := range e.IR.ReceiverPatterns {
		if strings.Contains(rp, "*") && matchesFQNPrefixWildcard(targetFQN, rp) {
			return true
		}
	}
	return false
}

// matchesFQNPrefixWildcard checks if any prefix of the FQN matches a wildcard pattern.
func matchesFQNPrefixWildcard(fqn, pattern string) bool {
	parts := strings.Split(fqn, ".")
	for i := len(parts) - 1; i > 0; i-- {
		prefix := strings.Join(parts[:i], ".")
		if matchesWildcardType(prefix, pattern) {
			return true
		}
	}
	return false
}

// matchesArgs checks argument constraints on a call site.
func (e *TypeConstrainedCallExecutor) matchesArgs(cs *core.CallSite) bool {
	return MatchesArguments(cs, e.IR.PositionalArgs, e.IR.KeywordArgs)
}

// matchesAnyReceiverTypeList checks if actual matches any receiver type in the list.
// Used by attribute executor which has a single receiverType stored as a list.
func matchesAnyReceiverTypeList(actual string, receiverTypes []string, checker InheritanceChecker) bool {
	for _, rt := range receiverTypes {
		if matchesReceiverType(actual, rt, checker) {
			return true
		}
	}
	return false
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

	receiverTypes := []string{e.IR.ReceiverType}

	// Step 2: Type inference match
	if cs.ResolvedViaTypeInference && cs.TypeConfidence >= float32(minConf) {
		if matchesAnyReceiverTypeList(cs.InferredType, receiverTypes, e.ThirdPartyRemote) {
			return true
		}
	}

	// Step 2b: FQN-to-receiver bridge (ported from TypeConstrainedCallExecutor)
	if cs.TargetFQN != "" {
		derivedReceiver := deriveReceiverFromFQN(cs.TargetFQN, cs.Target)
		if derivedReceiver != "" && matchesAnyReceiverTypeList(derivedReceiver, receiverTypes, e.ThirdPartyRemote) {
			return true
		}

		// Step 2c: FQN prefix match
		for _, rt := range receiverTypes {
			if strings.HasPrefix(cs.TargetFQN, rt+".") || cs.TargetFQN == rt {
				return true
			}
		}
	}

	// Step 3: Fallback
	switch e.IR.FallbackMode {
	case "name":
		return true
	default:
		return false
	}
}
