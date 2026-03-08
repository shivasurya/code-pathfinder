package dsl

import (
	"encoding/json"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// IRType represents the type of IR node.
type IRType string

const (
	IRTypeCallMatcher              IRType = "call_matcher"
	IRTypeVariableMatcher          IRType = "variable_matcher"
	IRTypeDataflow                 IRType = "dataflow"
	IRTypeLogicAnd                 IRType = "logic_and"
	IRTypeLogicOr                  IRType = "logic_or"
	IRTypeLogicNot                 IRType = "logic_not"
	IRTypeTypeConstrainedCall      IRType = "type_constrained_call"
	IRTypeTypeConstrainedAttribute IRType = "type_constrained_attribute"
)

// MatcherIR is the base interface for all matcher IR types.
type MatcherIR interface {
	GetType() IRType
}

// ArgumentConstraint represents a constraint on a single argument value.
type ArgumentConstraint struct {
	// Value is the expected argument value(s).
	// Can be a single value or a list of acceptable values (OR logic).
	// Examples: "0.0.0.0", "true", "777", ["Loader", "UnsafeLoader"]
	Value any `json:"value"`

	// Wildcard enables pattern matching with * and ? in Value.
	// Example: "0o7*" matches "0o777", "0o755", etc.
	Wildcard bool `json:"wildcard"`

	// Comparator specifies the comparison mode for the value.
	// Supported: "lt", "gt", "lte", "gte", "regex", "missing", "" (exact/wildcard).
	Comparator string `json:"comparator,omitempty"`
}

// CallMatcherIR represents call_matcher JSON IR.
type CallMatcherIR struct {
	Type      string   `json:"type"`      // "call_matcher"
	Patterns  []string `json:"patterns"`  // ["eval", "exec"]
	Wildcard  bool     `json:"wildcard"`  // true if any pattern has *
	MatchMode string   `json:"matchMode"` // "any" (OR) or "all" (AND)

	// PositionalArgs maps positional argument index (as string) to expected value(s).
	// Example: {"0": ArgumentConstraint{Value: "0.0.0.0"}}
	// Position is stored as string key to match JSON format from Python DSL.
	// This field is optional and will be omitted from JSON if empty.
	PositionalArgs map[string]ArgumentConstraint `json:"positionalArgs,omitempty"`

	// KeywordArgs maps keyword argument name to expected value(s).
	// Example: {"debug": ArgumentConstraint{Value: true}}
	// This field is optional and will be omitted from JSON if empty.
	KeywordArgs map[string]ArgumentConstraint `json:"keywordArgs,omitempty"`
}

// GetType returns the IR type.
func (c *CallMatcherIR) GetType() IRType {
	return IRTypeCallMatcher
}

// VariableMatcherIR represents variable_matcher JSON IR.
type VariableMatcherIR struct {
	Type     string `json:"type"`     // "variable_matcher"
	Pattern  string `json:"pattern"`  // "user_input" or "user_*"
	Wildcard bool   `json:"wildcard"` // true if pattern has *
}

// GetType returns the IR type.
func (v *VariableMatcherIR) GetType() IRType {
	return IRTypeVariableMatcher
}

// DataflowIR represents dataflow (taint analysis) JSON IR from Python DSL.
// Sources/Sinks/Sanitizers accept any matcher type (CallMatcherIR or TypeConstrainedCallIR).
type DataflowIR struct {
	Type        string            `json:"type"`        // "dataflow"
	Sources     []json.RawMessage `json:"sources"`     // Any matcher IR
	Sinks       []json.RawMessage `json:"sinks"`       // Any matcher IR
	Sanitizers  []json.RawMessage `json:"sanitizers"`  // Any matcher IR
	Propagation []PropagationIR   `json:"propagation"` // How taint flows (for future use)
	Scope       string            `json:"scope"`       // "local" or "global"
}

// GetType returns the IR type.
func (d *DataflowIR) GetType() IRType {
	return IRTypeDataflow
}

// PropagationIR represents propagation primitives (currently informational only).
type PropagationIR struct {
	Type     string         `json:"type"`     // "assignment", "function_args", etc.
	Metadata map[string]any `json:"metadata"` // Future use
}

// DataflowDetection represents a detected taint flow.
type DataflowDetection struct {
	FunctionFQN     string          // Function containing the vulnerability
	SourceLine      int             // Line where taint originates
	SinkLine        int             // Line where taint reaches sink
	TaintedVar      string          // Variable name that is tainted
	SinkCall        string          // Sink function name
	Confidence      float64         // 0.0-1.0 confidence score
	Sanitized       bool            // Was sanitization detected?
	Scope           string          // "local" or "global"
	MatchedCallSite *core.CallSite  // Internal: matched call site for DataflowExecutor use
}

// TypeConstrainedCallIR represents type_constrained_call JSON IR.
// Matches call sites where the receiver variable has a specific inferred type.
//
//nolint:tagliatelle // JSON tags match Python DSL format.
type TypeConstrainedCallIR struct {
	Type             string  `json:"type"`                       // "type_constrained_call"
	ReceiverType     string  `json:"receiverType,omitempty"`     // backward compat: single FQN
	ReceiverTypes    []string `json:"receiverTypes,omitempty"`   // multiple exact FQNs
	ReceiverPatterns []string `json:"receiverPatterns,omitempty"` // wildcard patterns
	MatchSubclasses  bool    `json:"matchSubclasses"`            // MRO inheritance matching
	MethodName       string  `json:"methodName,omitempty"`       // backward compat: single method
	MethodNames      []string `json:"methodNames,omitempty"`     // multiple method names
	MinConfidence    float64 `json:"minConfidence"`              // default 0.5
	FallbackMode     string  `json:"fallbackMode"`               // "name", "none"

	// Argument matching (reuses ArgumentConstraint)
	PositionalArgs map[string]ArgumentConstraint `json:"positionalArgs,omitempty"`
	KeywordArgs    map[string]ArgumentConstraint `json:"keywordArgs,omitempty"`
}

// GetEffectiveReceiverTypes returns the receiver types, merging legacy single field.
func (t *TypeConstrainedCallIR) GetEffectiveReceiverTypes() []string {
	types := make([]string, 0, len(t.ReceiverTypes)+1)
	if t.ReceiverType != "" {
		types = append(types, t.ReceiverType)
	}
	types = append(types, t.ReceiverTypes...)
	return types
}

// GetEffectiveMethodNames returns the method names, merging legacy single field.
func (t *TypeConstrainedCallIR) GetEffectiveMethodNames() []string {
	names := make([]string, 0, len(t.MethodNames)+1)
	if t.MethodName != "" {
		names = append(names, t.MethodName)
	}
	names = append(names, t.MethodNames...)
	return names
}

// GetType returns the IR type.
func (t *TypeConstrainedCallIR) GetType() IRType {
	return IRTypeTypeConstrainedCall
}

// TypeConstrainedAttributeIR represents type_constrained_attribute JSON IR.
// Matches attribute access on variables with a specific inferred type.
//
//nolint:tagliatelle // JSON tags match Python DSL format.
type TypeConstrainedAttributeIR struct {
	Type          string  `json:"type"`          // "type_constrained_attribute"
	ReceiverType  string  `json:"receiverType"`  // e.g., "django.http.HttpRequest"
	AttributeName string  `json:"attributeName"` // e.g., "GET"
	MinConfidence float64 `json:"minConfidence"` // default 0.5
	FallbackMode  string  `json:"fallbackMode"`  // "name", "none"
}

// GetType returns the IR type.
func (t *TypeConstrainedAttributeIR) GetType() IRType {
	return IRTypeTypeConstrainedAttribute
}

// RuleIR represents a complete rule with metadata.
type RuleIR struct {
	Rule struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Severity    string `json:"severity"`
		CWE         string `json:"cwe"`
		OWASP       string `json:"owasp"`
		Description string `json:"description"`
	} `json:"rule"`
	Matcher any `json:"matcher"` // Will be one of *MatcherIR types
}
