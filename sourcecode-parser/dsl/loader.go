package dsl

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// RuleLoader loads Python DSL rules and executes them.
type RuleLoader struct {
	RulesPath string // Path to .py rules file or directory
}

// NewRuleLoader creates a new rule loader.
func NewRuleLoader(rulesPath string) *RuleLoader {
	return &RuleLoader{RulesPath: rulesPath}
}

// LoadRules loads and executes Python DSL rules.
//
// Algorithm:
//  1. Execute Python rules file with timeout: python3 rules.py
//  2. Capture JSON IR output from stdout
//  3. Parse JSON IR into RuleIR structs
//  4. Return list of rules
func (l *RuleLoader) LoadRules() ([]RuleIR, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute Python script with context
	cmd := exec.CommandContext(ctx, "python3", l.RulesPath)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python rule execution timed out after 30s")
		}
		return nil, fmt.Errorf("failed to execute Python rules: %w", err)
	}

	// Parse JSON IR
	var rules []RuleIR
	if err := json.Unmarshal(output, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rule JSON IR: %w", err)
	}

	return rules, nil
}

// ExecuteRule executes a single rule against callgraph.
func (l *RuleLoader) ExecuteRule(rule *RuleIR, cg *core.CallGraph) ([]DataflowDetection, error) {
	// Determine matcher type and execute
	matcherMap, ok := rule.Matcher.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("matcher is not a map")
	}

	matcherType, ok := matcherMap["type"].(string)
	if !ok {
		return nil, fmt.Errorf("matcher type not found")
	}

	switch matcherType {
	case "call_matcher":
		return l.executeCallMatcher(matcherMap, cg)

	case "variable_matcher":
		return l.executeVariableMatcher(matcherMap, cg)

	case "dataflow":
		return l.executeDataflow(matcherMap, cg)

	case "logic_and", "logic_or", "logic_not":
		return l.executeLogic(matcherType, matcherMap, cg)

	default:
		return nil, fmt.Errorf("unknown matcher type: %s", matcherType)
	}
}

func (l *RuleLoader) executeCallMatcher(matcherMap map[string]interface{}, cg *core.CallGraph) ([]DataflowDetection, error) {
	// Convert map to CallMatcherIR
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal call_matcher: %w", err)
	}

	var ir CallMatcherIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call_matcher: %w", err)
	}

	executor := NewCallMatcherExecutor(&ir, cg)
	matches := executor.ExecuteWithContext()

	// Convert to DataflowDetection for consistent return type
	detections := []DataflowDetection{}
	for _, match := range matches {
		detections = append(detections, DataflowDetection{
			FunctionFQN: match.FunctionFQN,
			SourceLine:  match.Line,
			SinkLine:    match.Line,
			SinkCall:    match.CallSite.Target,
			Confidence:  1.0,
			Scope:       "local",
		})
	}

	return detections, nil
}

func (l *RuleLoader) executeDataflow(matcherMap map[string]interface{}, cg *core.CallGraph) ([]DataflowDetection, error) {
	// Convert map to DataflowIR
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dataflow: %w", err)
	}

	var ir DataflowIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dataflow: %w", err)
	}

	executor := NewDataflowExecutor(&ir, cg)
	return executor.Execute(), nil
}

func (l *RuleLoader) executeVariableMatcher(matcherMap map[string]interface{}, cg *core.CallGraph) ([]DataflowDetection, error) {
	// Convert map to VariableMatcherIR
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variable_matcher: %w", err)
	}

	var ir VariableMatcherIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variable_matcher: %w", err)
	}

	executor := NewVariableMatcherExecutor(&ir, cg)
	matches := executor.Execute()

	// Convert to DataflowDetection
	detections := []DataflowDetection{}
	for _, match := range matches {
		detections = append(detections, DataflowDetection{
			FunctionFQN: match.FunctionFQN,
			SourceLine:  match.Line,
			SinkLine:    match.Line,
			TaintedVar:  match.VariableName,
			Confidence:  1.0,
			Scope:       "local",
		})
	}

	return detections, nil
}

//nolint:unparam // Will be implemented in future PRs
func (l *RuleLoader) executeLogic(logicType string, matcherMap map[string]interface{}, cg *core.CallGraph) ([]DataflowDetection, error) {
	// TODO: Handle And/Or/Not logic operators
	// This requires recursive execution of nested matchers
	// For now, return empty detections as placeholder
	return []DataflowDetection{}, nil
}
