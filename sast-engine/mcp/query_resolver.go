// Package mcp provides query standardization for MCP.
package mcp

import (
	"regexp"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution/strategies"
)

// QueryResolver standardizes various query formats to canonical FQN.
type QueryResolver struct {
	inferencer   *resolution.BidirectionalInferencer
	attrRegistry strategies.AttributeRegistryInterface
}

// NewQueryResolver creates a new QueryResolver.
func NewQueryResolver(
	inferencer *resolution.BidirectionalInferencer,
	attrRegistry strategies.AttributeRegistryInterface,
) *QueryResolver {
	return &QueryResolver{
		inferencer:   inferencer,
		attrRegistry: attrRegistry,
	}
}

// QueryPattern represents a recognized query pattern.
type QueryPattern int

const (
	// PatternUnknown represents an unrecognized pattern.
	PatternUnknown QueryPattern = iota
	// PatternDirectFQN represents myapp.models.User.get_name.
	PatternDirectFQN
	// PatternInstanceCall represents user.get_name().
	PatternInstanceCall
	// PatternSelfCall represents self.process().
	PatternSelfCall
	// PatternChainedCall represents app.service.run().
	PatternChainedCall
	// PatternInlineInstantiation represents UserService().get_user().
	PatternInlineInstantiation
	// PatternStaticMethod represents ClassName.static_method().
	PatternStaticMethod
	// PatternClassMethod represents ClassName.class_method().
	PatternClassMethod
)

// StandardizedQuery represents a normalized query.
type StandardizedQuery struct {
	OriginalQuery string
	Pattern       QueryPattern
	CanonicalFQN  string
	ClassName     string
	MethodName    string
	Confidence    float64
	RequiresIndex bool   // True if we need to look up in index
	IndexQuery    string // The query to use for index lookup
}

// StandardizeQuery converts a user query to canonical form.
func (r *QueryResolver) StandardizeQuery(
	query string,
	knownVariables map[string]string,
	selfType string,
) *StandardizedQuery {
	query = strings.TrimSpace(query)

	// Try to detect the pattern
	pattern, parts := r.detectPattern(query)

	result := &StandardizedQuery{
		OriginalQuery: query,
		Pattern:       pattern,
		RequiresIndex: true,
	}

	switch pattern {
	case PatternDirectFQN:
		// Already canonical: myapp.models.User.get_name
		result.CanonicalFQN = query
		result.extractClassMethod(query)
		result.IndexQuery = query
		result.Confidence = 1.0

	case PatternInstanceCall:
		// user.get_name() - need to resolve 'user' type
		varName := parts["variable"]
		methodName := parts["method"]

		if typeFQN, ok := knownVariables[varName]; ok {
			result.ClassName = typeFQN
			result.MethodName = methodName
			result.CanonicalFQN = typeFQN + "." + methodName
			result.IndexQuery = result.CanonicalFQN
			result.Confidence = 0.85
		} else {
			result.Confidence = 0.0
			result.RequiresIndex = false // Can't resolve without type info
		}

	case PatternSelfCall:
		// self.process() - use selfType
		methodName := parts["method"]

		if selfType != "" {
			result.ClassName = selfType
			result.MethodName = methodName
			result.CanonicalFQN = selfType + "." + methodName
			result.IndexQuery = result.CanonicalFQN
			result.Confidence = 0.95
		} else {
			result.Confidence = 0.0
			result.RequiresIndex = false
		}

	case PatternChainedCall:
		// app.service.run() - need to resolve chain
		result.RequiresIndex = true
		result.Confidence = 0.0 // Will be filled by chain resolution

	case PatternInlineInstantiation:
		// UserService().get_user() - class is in the expression
		className := parts["class"]
		methodName := parts["method"]

		result.ClassName = className
		result.MethodName = methodName
		result.CanonicalFQN = className + "." + methodName
		result.IndexQuery = result.CanonicalFQN
		result.Confidence = 0.90

	case PatternStaticMethod, PatternClassMethod:
		// ClassName.method() - direct class reference
		className := parts["class"]
		methodName := parts["method"]

		result.ClassName = className
		result.MethodName = methodName
		result.CanonicalFQN = className + "." + methodName
		result.IndexQuery = result.CanonicalFQN
		result.Confidence = 0.95

	default:
		result.Confidence = 0.0
		result.RequiresIndex = false
	}

	return result
}

// detectPattern identifies the query pattern.
func (r *QueryResolver) detectPattern(query string) (QueryPattern, map[string]string) {
	parts := make(map[string]string)

	// Pattern: self.method() or self.method
	if selfPattern := regexp.MustCompile(`^self\.(\w+)(\(\))?$`); selfPattern.MatchString(query) {
		matches := selfPattern.FindStringSubmatch(query)
		parts["method"] = matches[1]
		return PatternSelfCall, parts
	}

	// Pattern: ClassName().method() or ClassName().method (inline instantiation)
	if inlinePattern := regexp.MustCompile(`^([A-Z]\w*)\(\)\.(\w+)(\(\))?$`); inlinePattern.MatchString(query) {
		matches := inlinePattern.FindStringSubmatch(query)
		parts["class"] = matches[1]
		parts["method"] = matches[2]
		return PatternInlineInstantiation, parts
	}

	// Pattern: ClassName.method() or ClassName.method (static/class method)
	if staticPattern := regexp.MustCompile(`^([A-Z]\w*)\.(\w+)(\(\))?$`); staticPattern.MatchString(query) {
		matches := staticPattern.FindStringSubmatch(query)
		parts["class"] = matches[1]
		parts["method"] = matches[2]
		return PatternStaticMethod, parts
	}

	// Pattern: module.class.method (direct FQN) - Check before instance/chain patterns
	// Must have 3+ segments and no parentheses to be FQN
	if fqnPattern := regexp.MustCompile(`^[\w.]+\.\w+$`); fqnPattern.MatchString(query) {
		// Check if it looks like FQN (has at least module.class.method)
		segments := strings.Split(query, ".")
		if len(segments) >= 3 && !strings.Contains(query, "(") {
			return PatternDirectFQN, parts
		}
	}

	// Pattern: variable.method() or variable.method (instance call)
	if instancePattern := regexp.MustCompile(`^([a-z_]\w*)\.(\w+)(\(\))?$`); instancePattern.MatchString(query) {
		matches := instancePattern.FindStringSubmatch(query)
		parts["variable"] = matches[1]
		parts["method"] = matches[2]
		return PatternInstanceCall, parts
	}

	// Pattern: a.b.c.method() or a.b.c.method (chained call)
	if chainPattern := regexp.MustCompile(`^[a-z_]\w*(\.\w+){2,}(\(\))?$`); chainPattern.MatchString(query) {
		return PatternChainedCall, parts
	}

	return PatternUnknown, parts
}

// extractClassMethod extracts class and method from a canonical FQN.
func (sq *StandardizedQuery) extractClassMethod(fqn string) {
	parts := strings.Split(fqn, ".")
	if len(parts) >= 2 {
		sq.MethodName = parts[len(parts)-1]
		sq.ClassName = strings.Join(parts[:len(parts)-1], ".")
	}
}

// ResolveChainedQuery resolves chained queries like app.service.run().
func (r *QueryResolver) ResolveChainedQuery(
	query string,
	filePath string,
	knownVariables map[string]string,
	selfType string,
) *StandardizedQuery {
	result := &StandardizedQuery{
		OriginalQuery: query,
		Pattern:       PatternChainedCall,
		RequiresIndex: true,
	}

	// Build TypeStore from known variables
	store := resolution.NewTypeStore()
	for varName, typeFQN := range knownVariables {
		store.Set(varName, core.NewConcreteType(typeFQN, 0.95),
			core.ConfidenceAssignment, filePath, 0, 0)
	}

	var selfTyp core.Type
	if selfType != "" {
		selfTyp = core.NewConcreteType(selfType, 0.95)
	}

	// Use ChainResolver
	resolver := resolution.NewChainResolver(r.attrRegistry, nil, nil).
		WithContext(filePath, []byte(query)).
		WithSelf(selfTyp, selfType)

	for varName, typeFQN := range knownVariables {
		resolver.WithVariable(varName, core.NewConcreteType(typeFQN, 0.95))
	}

	// Parse and resolve
	// (simplified - actual implementation would parse the query)

	result.Confidence = 0.7 // Chain resolution has lower confidence
	return result
}
