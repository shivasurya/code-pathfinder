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

// CallMatcherIR represents call_matcher JSON IR.
type CallMatcherIR struct {
	Type      string   `json:"type"`      // "call_matcher"
	Patterns  []string `json:"patterns"`  // ["eval", "exec"]
	Wildcard  bool     `json:"wildcard"`  // true if any pattern has *
	MatchMode string   `json:"matchMode"` // "any" (OR) or "all" (AND)
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
	Type        string           `json:"type"`        // "dataflow"
	Sources     []CallMatcherIR  `json:"sources"`     // Where taint originates
	Sinks       []CallMatcherIR  `json:"sinks"`       // Dangerous functions
	Sanitizers  []CallMatcherIR  `json:"sanitizers"`  // Taint-removing functions
	Propagation []PropagationIR  `json:"propagation"` // How taint flows (for future use)
	Scope       string           `json:"scope"`       // "local" or "global"
}

// GetType returns the IR type.
func (d *DataflowIR) GetType() IRType {
	return IRTypeDataflow
}

// PropagationIR represents propagation primitives (currently informational only).
type PropagationIR struct {
	Type     string                 `json:"type"`     // "assignment", "function_args", etc.
	Metadata map[string]interface{} `json:"metadata"` // Future use
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
	Matcher interface{} `json:"matcher"` // Will be one of *MatcherIR types
}
