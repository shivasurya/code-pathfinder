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
