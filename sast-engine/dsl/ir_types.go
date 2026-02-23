package dsl

// IRType represents the type of IR node.
type IRType string

const (
	IRTypeCallMatcher     IRType = "call_matcher"
	IRTypeVariableMatcher IRType = "variable_matcher"
	IRTypeDataflow        IRType = "dataflow"
	IRTypeLogicAnd        IRType = "logic_and"
	IRTypeLogicOr         IRType = "logic_or"
	IRTypeLogicNot        IRType = "logic_not"
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
type DataflowIR struct {
	Type        string          `json:"type"`        // "dataflow"
	Sources     []CallMatcherIR `json:"sources"`     // Where taint originates
	Sinks       []CallMatcherIR `json:"sinks"`       // Dangerous functions
	Sanitizers  []CallMatcherIR `json:"sanitizers"`  // Taint-removing functions
	Propagation []PropagationIR `json:"propagation"` // How taint flows (for future use)
	Scope       string          `json:"scope"`       // "local" or "global"
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
	FunctionFQN string  // Function containing the vulnerability
	SourceLine  int     // Line where taint originates
	SinkLine    int     // Line where taint reaches sink
	TaintedVar  string  // Variable name that is tainted
	SinkCall    string  // Sink function name
	Confidence  float64 // 0.0-1.0 confidence score
	Sanitized   bool    // Was sanitization detected?
	Scope       string  // "local" or "global"
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
