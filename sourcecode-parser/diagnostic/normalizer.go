package diagnostic

import (
	"strings"
)

// NormalizedTaintFlow is the common format for comparison.
// Both tool and LLM results are converted to this format.
type NormalizedTaintFlow struct {
	SourceLine        int
	SourceVariable    string
	SourceCategory    string // Semantic: "user_input", "file_read", etc.

	SinkLine          int
	SinkVariable      string
	SinkCategory      string // Semantic: "sql_execution", "command_exec", etc.

	VulnerabilityType string // "SQL_INJECTION", "XSS", etc.
	Confidence        float64
}

// NormalizeToolResult converts our tool's result to normalized format.
func NormalizeToolResult(toolResult *FunctionTaintResult) []NormalizedTaintFlow {
	normalized := make([]NormalizedTaintFlow, 0, len(toolResult.TaintFlows))

	for _, flow := range toolResult.TaintFlows {
		normalized = append(normalized, NormalizedTaintFlow{
			SourceLine:        flow.SourceLine,
			SourceVariable:    flow.SourceVariable,
			SourceCategory:    flow.SourceCategory,
			SinkLine:          flow.SinkLine,
			SinkVariable:      flow.SinkVariable,
			SinkCategory:      flow.SinkCategory,
			VulnerabilityType: flow.VulnerabilityType,
			Confidence:        flow.Confidence,
		})
	}

	return normalized
}

// NormalizeLLMResult converts LLM test cases to normalized format.
func NormalizeLLMResult(llmResult *LLMAnalysisResult) []NormalizedTaintFlow {
	normalized := make([]NormalizedTaintFlow, 0, len(llmResult.DataflowTestCases))

	for _, testCase := range llmResult.DataflowTestCases {
		// Only include test cases where LLM expects detection
		if testCase.ExpectedDetection {
			normalized = append(normalized, NormalizedTaintFlow{
				SourceLine:        testCase.Source.Line,
				SourceVariable:    testCase.Source.Variable,
				SourceCategory:    categorizeLLMPattern(testCase.Source.Pattern),
				SinkLine:          testCase.Sink.Line,
				SinkVariable:      testCase.Sink.Variable,
				SinkCategory:      categorizeLLMPattern(testCase.Sink.Pattern),
				VulnerabilityType: normalizeVulnType(testCase.VulnerabilityType),
				Confidence:        testCase.Confidence,
			})
		}
	}

	return normalized
}

// categorizeLLMPattern maps LLM pattern string to semantic category.
func categorizeLLMPattern(pattern string) string {
	patternLower := strings.ToLower(pattern)

	// User input
	if strings.Contains(patternLower, "request") ||
		strings.Contains(patternLower, "input") ||
		strings.Contains(patternLower, "get[") ||
		strings.Contains(patternLower, "post[") {
		return "user_input"
	}

	// SQL
	if strings.Contains(patternLower, "execute") ||
		strings.Contains(patternLower, "sql") ||
		strings.Contains(patternLower, "cursor") {
		return "sql_execution"
	}

	// Command execution
	if strings.Contains(patternLower, "system") ||
		strings.Contains(patternLower, "subprocess") ||
		strings.Contains(patternLower, "popen") ||
		strings.Contains(patternLower, "call") {
		return "command_exec"
	}

	// Code execution
	if strings.Contains(patternLower, "eval") ||
		strings.Contains(patternLower, "exec") {
		return "code_exec"
	}

	return "other"
}

// normalizeVulnType normalizes vulnerability type names.
func normalizeVulnType(vulnType string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(vulnType, " ", "_"))

	// Handle variations
	equivalenceMap := map[string]string{
		"SQLI":                 "SQL_INJECTION",
		"SQL INJECTION":        "SQL_INJECTION",
		"CMD_INJECTION":        "COMMAND_INJECTION",
		"COMMAND INJECTION":    "COMMAND_INJECTION",
		"OS_COMMAND_INJECTION": "COMMAND_INJECTION",
		"CODE INJECTION":       "CODE_INJECTION",
		"CROSS_SITE_SCRIPTING": "XSS",
		"CROSS SITE SCRIPTING": "XSS",
		"PATH_TRAVERSAL":       "PATH_TRAVERSAL",
		"DIRECTORY_TRAVERSAL":  "PATH_TRAVERSAL",
	}

	if canonical, ok := equivalenceMap[normalized]; ok {
		return canonical
	}

	return normalized
}

// MatchConfig specifies how lenient fuzzy matching should be.
type MatchConfig struct {
	// LineThreshold: Accept matches within ±N lines (default: 2)
	LineThreshold int

	// AllowVariableAliases: Match user_input vs user_input_1 (SSA) (default: true)
	AllowVariableAliases bool

	// SemanticVulnTypes: "SQL_INJECTION" == "sqli" (default: true)
	SemanticVulnTypes bool
}

// DefaultMatchConfig returns default fuzzy matching configuration.
func DefaultMatchConfig() MatchConfig {
	return MatchConfig{
		LineThreshold:        2,
		AllowVariableAliases: true,
		SemanticVulnTypes:    true,
	}
}

// FlowsMatch checks if two normalized flows match (fuzzy matching).
func FlowsMatch(f1, f2 NormalizedTaintFlow, config MatchConfig) bool {
	// 1. Line numbers within threshold
	sourceLineMatch := abs(f1.SourceLine-f2.SourceLine) <= config.LineThreshold
	sinkLineMatch := abs(f1.SinkLine-f2.SinkLine) <= config.LineThreshold

	if !sourceLineMatch || !sinkLineMatch {
		return false
	}

	// 2. Variable names match (with optional aliases)
	sourceVarMatch := variablesMatch(f1.SourceVariable, f2.SourceVariable, config.AllowVariableAliases)
	sinkVarMatch := variablesMatch(f1.SinkVariable, f2.SinkVariable, config.AllowVariableAliases)

	if !sourceVarMatch || !sinkVarMatch {
		return false
	}

	// 3. Categories match (semantic comparison)
	categoryMatch := (f1.SourceCategory == f2.SourceCategory) &&
		(f1.SinkCategory == f2.SinkCategory)

	if !categoryMatch {
		return false
	}

	// 4. Vulnerability type match (with semantic equivalence)
	vulnMatch := vulnTypesMatch(f1.VulnerabilityType, f2.VulnerabilityType, config.SemanticVulnTypes)

	return vulnMatch
}

// variablesMatch checks if two variable names match (with optional alias support).
func variablesMatch(v1, v2 string, allowAliases bool) bool {
	if v1 == v2 {
		return true
	}

	if allowAliases {
		// Strip SSA suffixes: user_input_1 → user_input
		base1 := stripSSASuffix(v1)
		base2 := stripSSASuffix(v2)
		return base1 == base2
	}

	return false
}

// stripSSASuffix removes SSA renaming suffix.
// Example: "user_input_1" → "user_input".
func stripSSASuffix(varName string) string {
	// Simple heuristic: remove _N suffix where N is digit
	parts := strings.Split(varName, "_")
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		// Check if last part is a number
		if len(lastPart) > 0 && lastPart[0] >= '0' && lastPart[0] <= '9' {
			return strings.Join(parts[:len(parts)-1], "_")
		}
	}
	return varName
}

// vulnTypesMatch checks if two vulnerability types match semantically.
func vulnTypesMatch(t1, t2 string, semantic bool) bool {
	if t1 == t2 {
		return true
	}

	if semantic {
		// Normalize both
		t1Norm := normalizeVulnType(t1)
		t2Norm := normalizeVulnType(t2)
		return t1Norm == t2Norm
	}

	return false
}

// abs returns absolute value of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
