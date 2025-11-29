package dsl

// EnrichedDetection contains a detection with all metadata needed for output.
// This is the canonical output structure used by all formatters.
type EnrichedDetection struct {
	// Original detection data
	Detection DataflowDetection

	// Resolved location information
	Location LocationInfo

	// Code snippet with context
	Snippet CodeSnippet

	// Rule metadata (from RuleIR)
	Rule RuleMetadata

	// Taint path for inter-procedural flows (nil for pattern matches)
	TaintPath []TaintPathNode

	// Detection classification
	DetectionType DetectionType
}

// LocationInfo contains resolved file path and position.
type LocationInfo struct {
	FilePath  string // Absolute path: /project/auth/login.py
	RelPath   string // Relative path: auth/login.py
	Line      int    // 1-indexed line number
	Column    int    // 1-indexed column (0 if unknown)
	EndLine   int    // End line for multi-line (0 if single line)
	EndColumn int    // End column (0 if unknown)
	Function  string // Function name containing the finding
	ClassName string // Class name if applicable (empty string if none)
}

// CodeSnippet contains source code context around the finding.
type CodeSnippet struct {
	Lines         []SnippetLine // Code lines with context
	StartLine     int           // First line number in snippet
	HighlightLine int           // Line to highlight (the finding)
}

// SnippetLine represents a single line in a code snippet.
type SnippetLine struct {
	Number      int    // Line number
	Content     string // Line content (without trailing newline)
	IsHighlight bool   // True if this is the finding line
}

// RuleMetadata contains rule information for display.
type RuleMetadata struct {
	ID          string   // Rule ID: "command-injection"
	Name        string   // Human name: "Command Injection"
	Severity    string   // "critical", "high", "medium", "low"
	Description string   // Rule description
	CWE         []string // CWE IDs: ["CWE-78"]
	OWASP       []string // OWASP refs: ["A1:2017"]
	References  []string // Documentation URLs
}

// TaintPathNode represents a step in an inter-procedural taint flow.
type TaintPathNode struct {
	Location    LocationInfo
	Description string // "Taint originates from user input"
	Variable    string // Variable name at this step
	IsSource    bool   // True if this is the source
	IsSink      bool   // True if this is the sink
}

// DetectionType classifies how the vulnerability was detected.
type DetectionType string

const (
	DetectionTypePattern     DetectionType = "pattern"      // Structural pattern match
	DetectionTypeTaintLocal  DetectionType = "taint-local"  // Intra-procedural taint
	DetectionTypeTaintGlobal DetectionType = "taint-global" // Inter-procedural taint
)

// ConfidenceLevel returns human-readable confidence.
func (e *EnrichedDetection) ConfidenceLevel() string {
	switch {
	case e.Detection.Confidence >= 0.8:
		return "high"
	case e.Detection.Confidence >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

// DetectionBadge returns the display badge for this detection type.
func (e *EnrichedDetection) DetectionBadge() string {
	switch e.DetectionType {
	case DetectionTypePattern:
		return "[Pattern]"
	case DetectionTypeTaintLocal:
		return "[Taint-Local]"
	case DetectionTypeTaintGlobal:
		return "[Taint-Global]"
	default:
		return "[Unknown]"
	}
}
