package dsl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Logger interface for verbose logging (avoids import cycle with output package).
type Logger interface {
	Debug(format string, args ...any)
	Statistic(format string, args ...any)
	IsDebug() bool
	IsVerbose() bool
}

// RuleLoader loads Python SDK rules and executes them.
type RuleLoader struct {
	RulesPath   string               // Path to .py rules file or directory
	Config      *QueryTypeConfig     // Execution config (nil → defaults)
	Diagnostics *DiagnosticCollector  // Optional diagnostic collector (nil → no diagnostics)
}

// NewRuleLoader creates a new rule loader.
func NewRuleLoader(rulesPath string) *RuleLoader {
	return &RuleLoader{RulesPath: rulesPath}
}

// LoadRules loads and executes Python SDK rules.
//
// Algorithm:
//  1. Check if path is file or directory
//  2. If directory, find all .py files recursively
//  3. Execute each Python file with timeout: python3 rules.py
//  4. Capture JSON IR output from stdout
//  5. Parse and consolidate JSON IR into RuleIR structs
//  6. Return combined list of rules
func (l *RuleLoader) LoadRules(logger Logger) ([]RuleIR, error) {
	// Check if path is file or directory
	info, err := os.Stat(l.RulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access rules path: %w", err)
	}

	// If single file, check for rule decorators first
	if !info.IsDir() {
		// Skip files without code analysis rules (consistent with directory behavior)
		// This allows pure container rule files to be used with --rules flag
		if !hasCodeAnalysisRuleDecorators(l.RulesPath) {
			return []RuleIR{}, nil
		}
		return l.loadRulesFromFile(l.RulesPath, logger)
	}

	// If directory, find all .py files and load them
	return l.loadRulesFromDirectory(l.RulesPath, logger)
}

// loadRulesFromFile loads rules from a single Python file.
func (l *RuleLoader) loadRulesFromFile(filePath string, logger Logger) ([]RuleIR, error) {
	// Create context with timeout to prevent hanging
	timeout := l.Config.getExecutionTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", filePath)

	// Execute Python script with context
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python rule execution timed out after %s for file: %s", timeout, filePath)
		}
		return nil, fmt.Errorf("failed to execute Python rules from %s: %w", filePath, err)
	}

	// Parse JSON IR - try array format first (code analysis rules)
	var rules []RuleIR
	if err := json.Unmarshal(output, &rules); err != nil {
		// If array parsing fails, check if it's a container rule (object format)
		var containerTest struct {
			Dockerfile []any `json:"dockerfile"`
			Compose    []any `json:"compose"`
		}
		if containerErr := json.Unmarshal(output, &containerTest); containerErr == nil {
			// This is a container rule file, skip it (handled by LoadContainerRules)
			return []RuleIR{}, nil
		}
		return nil, fmt.Errorf("failed to parse rule JSON IR from %s: %w", filePath, err)
	}

	// Log loaded rules in verbose mode
	if logger != nil && logger.IsVerbose() {
		for _, rule := range rules {
			logger.Statistic("  - Loaded rule %s from %s", rule.Rule.ID, filePath)
		}
	}

	return rules, nil
}

// loadRulesFromDirectory loads rules from all .py files in a directory.
func (l *RuleLoader) loadRulesFromDirectory(dirPath string, logger Logger) ([]RuleIR, error) {
	var allRules []RuleIR

	// Walk directory and find all .py files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Python files
		if info.IsDir() || filepath.Ext(path) != ".py" {
			return nil
		}

		// Skip files without code analysis rule decorators (early filtering to avoid executing non-rule files)
		if !hasCodeAnalysisRuleDecorators(path) {
			return nil
		}

		// Load rules from this file
		rules, err := l.loadRulesFromFile(path, logger)
		if err != nil {
			// Log the error for debugging, then skip (may be container rules)
			if logger != nil {
				logger.Debug("skipping rule file %s: %v", path, err)
			}
			//nolint:nilerr // Intentionally skip files that aren't code analysis rules
			return nil
		}

		allRules = append(allRules, rules...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	// It's OK to have zero code analysis rules (directory might only contain container rules)
	return allRules, nil
}

// hasCodeAnalysisRuleDecorators checks if a Python file contains code analysis rule decorators.
// It scans for @rule decorator or codepathfinder imports.
func hasCodeAnalysisRuleDecorators(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	fileContent := string(content)
	// NOTE: This string-based decorator detection is fragile — it could match
	// @rule( inside comments or string literals. A proper AST-based parser
	// is tracked as a separate tech spec. For now, this works for standard
	// rule files where @rule appears at the top level.
	return strings.Contains(fileContent, "@rule(") ||
		strings.Contains(fileContent, "@go_rule(") ||
		strings.Contains(fileContent, "@c_rule(") ||
		strings.Contains(fileContent, "@cpp_rule(") ||
		strings.Contains(fileContent, "from codepathfinder import") ||
		strings.Contains(fileContent, "import codepathfinder")
}

// hasContainerRuleDecorators checks if a Python file contains container rule decorators.
// It scans for @dockerfile_rule or @compose_rule annotations.
func hasContainerRuleDecorators(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Check for container rule decorators
	fileContent := string(content)
	return strings.Contains(fileContent, "@dockerfile_rule") ||
		strings.Contains(fileContent, "@compose_rule")
}

// hasAnyContainerRulesInPath checks if any Python files in the given path contain container rule decorators.
func (l *RuleLoader) hasAnyContainerRulesInPath() bool {
	info, err := os.Stat(l.RulesPath)
	if err != nil {
		return false
	}

	// If single file, check directly
	if !info.IsDir() {
		return hasContainerRuleDecorators(l.RulesPath)
	}

	// If directory, check all .py files
	hasRules := false
	filepath.Walk(l.RulesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || hasRules {
			//nolint:nilerr // Intentionally ignore errors during walk - just return false
			return nil
		}

		// Check Python files only
		if !info.IsDir() && filepath.Ext(path) == ".py" {
			if hasContainerRuleDecorators(path) {
				hasRules = true
			}
		}
		return nil
	})

	return hasRules
}

// LoadContainerRules loads container rules (Dockerfile/Compose) from Python SDK files.
// Returns JSON IR in format: {"dockerfile": [...], "compose": [...]}.
func (l *RuleLoader) LoadContainerRules(logger Logger) ([]byte, error) {
	// Check if path is file or directory (check existence first)
	info, err := os.Stat(l.RulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access rules path: %w", err)
	}

	// Early filtering: check if any files contain container rule decorators
	if !l.hasAnyContainerRulesInPath() {
		return nil, fmt.Errorf("no container rules detected (no @dockerfile_rule or @compose_rule decorators found)")
	}

	var containerRulesJSON struct {
		Dockerfile []map[string]any `json:"dockerfile"`
		Compose    []map[string]any `json:"compose"`
	}

	// If single file, load directly
	if !info.IsDir() {
		jsonIR, err := l.loadContainerRulesFromFile(l.RulesPath, logger)
		if err != nil {
			return nil, err
		}
		// Parse and merge
		var fileRules struct {
			Dockerfile []map[string]any `json:"dockerfile"`
			Compose    []map[string]any `json:"compose"`
		}
		if err := json.Unmarshal(jsonIR, &fileRules); err != nil {
			return nil, fmt.Errorf("failed to parse container rules JSON: %w", err)
		}
		containerRulesJSON.Dockerfile = append(containerRulesJSON.Dockerfile, fileRules.Dockerfile...)
		containerRulesJSON.Compose = append(containerRulesJSON.Compose, fileRules.Compose...)
	} else {
		// If directory, find all .py files and load them
		err := filepath.Walk(l.RulesPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip non-Python files
			if info.IsDir() || filepath.Ext(path) != ".py" {
				return nil
			}

			// Skip files without container rule decorators
			if !hasContainerRuleDecorators(path) {
				return nil
			}

			// Load container rules from this file
			jsonIR, err := l.loadContainerRulesFromFile(path, logger)
			if err != nil {
				// Skip files that don't contain container rules (they might be code analysis rules)
				//nolint:nilerr // Intentionally skip files that aren't container rules
				return nil
			}

			// Parse and merge
			var fileRules struct {
				Dockerfile []map[string]any `json:"dockerfile"`
				Compose    []map[string]any `json:"compose"`
			}
			if err := json.Unmarshal(jsonIR, &fileRules); err != nil {
				// Skip files with invalid JSON (might not be container rules)
				//nolint:nilerr // Intentionally skip files with wrong format
				return nil
			}

			containerRulesJSON.Dockerfile = append(containerRulesJSON.Dockerfile, fileRules.Dockerfile...)
			containerRulesJSON.Compose = append(containerRulesJSON.Compose, fileRules.Compose...)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", l.RulesPath, err)
		}
	}

	// Log loaded container rules in verbose mode (removed - logging happens in loadContainerRulesFromFile with paths)

	// Return combined JSON
	return json.Marshal(containerRulesJSON)
}

// loadContainerRulesFromFile loads container rules from a single Python file or directory.
// Creates a temporary Python script to import and compile all rules, then executes it.
func (l *RuleLoader) loadContainerRulesFromFile(rulesPath string, logger Logger) ([]byte, error) {
	// Create context with timeout
	timeout := l.Config.getExecutionTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create a temporary Python script that compiles rules from the given path
	compileScript := fmt.Sprintf(`
import json
import importlib.util
from pathlib import Path

from rules import container_decorators, container_ir

# Import rule file(s)
rule_path = Path('%s')

if rule_path.is_file():
    # Single file - import it
    spec = importlib.util.spec_from_file_location("user_rule", rule_path)
    if spec and spec.loader:
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
elif rule_path.is_dir():
    # Directory - import all .py files
    for rule_file in rule_path.glob("*.py"):
        if rule_file.name == "__init__.py":
            continue
        try:
            spec = importlib.util.spec_from_file_location(rule_file.stem, rule_file)
            if spec and spec.loader:
                module = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(module)
        except Exception:
            pass  # Skip files that fail to import

# Compile and output
json_ir = container_ir.compile_all_rules()
print(json.dumps(json_ir))
`, rulesPath)

	cmd := exec.CommandContext(ctx, "python3", "-c", compileScript)

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python rule execution timed out after %s", timeout)
		}
		return nil, fmt.Errorf("failed to compile container rules: %w", err)
	}

	// Validate it's valid JSON and log loaded rules in verbose mode
	var containerRules struct {
		Dockerfile []map[string]any `json:"dockerfile"`
		Compose    []map[string]any `json:"compose"`
	}
	if err := json.Unmarshal(output, &containerRules); err != nil {
		return nil, fmt.Errorf("invalid JSON output from container rules: %w", err)
	}

	// Log loaded rules in verbose mode
	if logger != nil && logger.IsVerbose() {
		for _, dockerfileRule := range containerRules.Dockerfile {
			if id, ok := dockerfileRule["id"].(string); ok {
				logger.Statistic("  - Loaded Dockerfile rule %s from %s", id, rulesPath)
			}
		}
		for _, composeRule := range containerRules.Compose {
			if id, ok := composeRule["id"].(string); ok {
				logger.Statistic("  - Loaded docker-compose rule %s from %s", id, rulesPath)
			}
		}
	}

	return output, nil
}

// ExecuteRule executes a single rule against callgraph.
func (l *RuleLoader) ExecuteRule(rule *RuleIR, cg *core.CallGraph) ([]DataflowDetection, error) {
	// Determine matcher type and execute
	matcherMap, ok := rule.Matcher.(map[string]any)
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

	case "type_constrained_call":
		return l.executeTypeConstrainedCall(matcherMap, cg)

	case "type_constrained_attribute":
		return l.executeTypeConstrainedAttribute(matcherMap, cg)

	// Container matchers - skip silently (handled by ContainerRuleExecutor)
	case "missing_instruction", "instruction", "service_has", "service_missing", "any_of", "all_of", "none_of":
		return []DataflowDetection{}, nil

	default:
		return nil, fmt.Errorf("unknown matcher type: %s", matcherType)
	}
}

func (l *RuleLoader) executeCallMatcher(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
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
			FunctionFQN:  match.FunctionFQN,
			SourceLine:   match.Line,
			SourceColumn: match.CallSite.Location.Column,
			SinkLine:     match.Line,
			SinkColumn:   match.CallSite.Location.Column,
			SinkCall:     match.CallSite.Target,
			Confidence:   1.0,
			Scope:        "local",
		})
	}

	return detections, nil
}

func (l *RuleLoader) executeDataflow(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	// Convert map to DataflowIR.
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dataflow: %w", err)
	}

	var ir DataflowIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dataflow: %w", err)
	}

	if err := validateDataflowIR(&ir, l.Diagnostics); err != nil {
		l.Diagnostics.Addf("skip", "ir_validation", "skipping dataflow: %v", err)
		return []DataflowDetection{}, nil
	}

	executor := NewDataflowExecutor(&ir, cg)
	executor.Config = l.Config
	executor.Diagnostics = l.Diagnostics
	return safeExecute(executor.Execute, l.Diagnostics), nil
}

func (l *RuleLoader) executeVariableMatcher(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
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

func (l *RuleLoader) executeTypeConstrainedCall(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal type_constrained_call: %w", err)
	}

	var ir TypeConstrainedCallIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal type_constrained_call: %w", err)
	}

	if err := validateTypeConstrainedCallIR(&ir, l.Diagnostics); err != nil {
		l.Diagnostics.Addf("skip", "ir_validation", "skipping type_constrained_call: %v", err)
		return []DataflowDetection{}, nil
	}

	executor := &TypeConstrainedCallExecutor{
		IR:               &ir,
		CallGraph:        cg,
		Config:           l.Config,
		ThirdPartyRemote: extractInheritanceChecker(cg),
		Diagnostics:      l.Diagnostics,
	}
	return safeExecute(executor.Execute, l.Diagnostics), nil
}

func (l *RuleLoader) executeTypeConstrainedAttribute(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal type_constrained_attribute: %w", err)
	}

	var ir TypeConstrainedAttributeIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal type_constrained_attribute: %w", err)
	}

	if err := validateTypeConstrainedAttributeIR(&ir, l.Diagnostics); err != nil {
		l.Diagnostics.Addf("skip", "ir_validation", "skipping type_constrained_attribute: %v", err)
		return []DataflowDetection{}, nil
	}

	executor := &TypeConstrainedAttributeExecutor{
		IR:               &ir,
		CallGraph:        cg,
		Config:           l.Config,
		ThirdPartyRemote: extractInheritanceChecker(cg),
		Diagnostics:      l.Diagnostics,
	}
	return safeExecute(executor.Execute, l.Diagnostics), nil
}

func (l *RuleLoader) executeLogic(logicType string, matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	switch logicType {
	case "logic_or":
		return l.executeLogicOr(matcherMap, cg)
	case "logic_and":
		return l.executeLogicAnd(matcherMap, cg)
	case "logic_not":
		return l.executeLogicNot(matcherMap, cg)
	default:
		return nil, fmt.Errorf("unknown logic type: %s", logicType)
	}
}

func (l *RuleLoader) executeLogicOr(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	matchers, ok := matcherMap["matchers"].([]any)
	if !ok {
		return nil, fmt.Errorf("logic_or requires 'matchers' array")
	}

	var all []DataflowDetection
	for _, m := range matchers {
		mMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		ruleIR := &RuleIR{Matcher: mMap}
		dets, err := l.ExecuteRule(ruleIR, cg)
		if err != nil {
			return nil, err
		}
		all = append(all, dets...)
	}
	return deduplicateDetections(all), nil
}

func (l *RuleLoader) executeLogicAnd(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	matchers, ok := matcherMap["matchers"].([]any)
	if !ok {
		return nil, fmt.Errorf("logic_and requires 'matchers' array")
	}

	if len(matchers) == 0 {
		return nil, nil
	}

	// Execute first matcher
	mMap, ok := matchers[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("matcher is not a map")
	}
	ruleIR := &RuleIR{Matcher: mMap}
	result, err := l.ExecuteRule(ruleIR, cg)
	if err != nil {
		return nil, err
	}

	// Intersect with remaining matchers
	for _, m := range matchers[1:] {
		mMap, ok = m.(map[string]any)
		if !ok {
			continue
		}
		ruleIR = &RuleIR{Matcher: mMap}
		dets, err := l.ExecuteRule(ruleIR, cg)
		if err != nil {
			return nil, err
		}
		result = intersectDetections(result, dets)
	}
	return result, nil
}

// executeLogicNot implements Not(M1, M2, ...) semantics:
// Universe = all call sites in CallGraph, Result = Universe - Union(M1, M2, ...).
func (l *RuleLoader) executeLogicNot(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	if cg == nil || cg.CallSites == nil {
		l.Diagnostics.Addf("warning", "logic_not", "CallGraph or CallSites is nil, returning empty")
		return nil, nil
	}

	// Build universe: all (functionFQN, line) pairs from call graph.
	type csKey struct {
		FunctionFQN string
		Line        int
		Target      string
	}
	universe := make(map[csKey]bool)
	var universeKeys []csKey
	for funcFQN, callSites := range cg.CallSites {
		for _, cs := range callSites {
			key := csKey{funcFQN, cs.Location.Line, cs.Target}
			if !universe[key] {
				universe[key] = true
				universeKeys = append(universeKeys, key)
			}
		}
	}

	// Execute nested matchers, collect matched keys.
	matchers, ok := matcherMap["matchers"].([]any)
	if !ok {
		// No matchers array → return entire universe.
		l.Diagnostics.Addf("warning", "logic_not", "no 'matchers' array, returning entire universe (%d call sites)", len(universeKeys))
	}

	matched := make(map[csKey]bool)
	for _, m := range matchers {
		mMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		ruleIR := &RuleIR{Matcher: mMap}
		dets, err := l.ExecuteRule(ruleIR, cg)
		if err != nil {
			return nil, err
		}
		for _, d := range dets {
			matched[csKey{d.FunctionFQN, d.SourceLine, d.SinkCall}] = true
		}
	}

	// Subtract: universe - matched.
	var result []DataflowDetection
	for _, key := range universeKeys {
		if !matched[key] {
			result = append(result, DataflowDetection{
				FunctionFQN: key.FunctionFQN,
				SourceLine:  key.Line,
				SinkLine:    key.Line,
				SinkCall:    key.Target,
				Confidence:  1.0,
				MatchMethod: "logic_not",
				Scope:       "local",
			})
		}
	}

	l.Diagnostics.Addf("debug", "logic_not",
		"universe=%d, matched=%d, result=%d", len(universe), len(matched), len(result))

	return result, nil
}

// extractInheritanceChecker extracts InheritanceCheckers from a CallGraph's
// ThirdPartyRemote and StdlibRemote fields, returning a composite checker
// that queries both registries. Returns nil if neither is set.
func extractInheritanceChecker(cg *core.CallGraph) InheritanceChecker {
	if cg == nil {
		return nil
	}

	var checkers []InheritanceChecker

	if cg.ThirdPartyRemote != nil {
		if checker, ok := cg.ThirdPartyRemote.(InheritanceChecker); ok {
			checkers = append(checkers, checker)
		}
	}
	if cg.StdlibRemote != nil {
		if checker, ok := cg.StdlibRemote.(InheritanceChecker); ok {
			checkers = append(checkers, checker)
		}
	}

	switch len(checkers) {
	case 0:
		return nil
	case 1:
		return checkers[0]
	default:
		return &compositeInheritanceChecker{checkers: checkers}
	}
}

// compositeInheritanceChecker delegates to multiple InheritanceChecker instances,
// checking each in order until one matches (third-party, then stdlib).
type compositeInheritanceChecker struct {
	checkers []InheritanceChecker
}

func (c *compositeInheritanceChecker) HasModule(moduleName string) bool {
	for _, ch := range c.checkers {
		if ch.HasModule(moduleName) {
			return true
		}
	}
	return false
}

func (c *compositeInheritanceChecker) IsSubclassSimple(moduleName, className, parentFQN string) bool {
	for _, ch := range c.checkers {
		if ch.HasModule(moduleName) && ch.IsSubclassSimple(moduleName, className, parentFQN) {
			return true
		}
	}
	return false
}

func (c *compositeInheritanceChecker) GetClassMRO(moduleName, className string) []string {
	for _, ch := range c.checkers {
		if mro := ch.GetClassMRO(moduleName, className); len(mro) > 0 {
			return mro
		}
	}
	return nil
}

func dedupKey(d DataflowDetection) string {
	return fmt.Sprintf("%s:%d:%d:%d:%d:%s:%s",
		d.FunctionFQN, d.SourceLine, d.SourceColumn, d.SinkLine, d.SinkColumn, d.SinkCall, d.MatchMethod)
}

func deduplicateDetections(dets []DataflowDetection) []DataflowDetection {
	best := make(map[string]DataflowDetection)
	for _, d := range dets {
		key := dedupKey(d)
		if existing, ok := best[key]; !ok || d.Confidence > existing.Confidence {
			best[key] = d
		}
	}
	result := make([]DataflowDetection, 0, len(best))
	for _, d := range best {
		result = append(result, d)
	}
	return result
}

func intersectKey(d DataflowDetection) string {
	return fmt.Sprintf("%s:%d:%d:%d:%d",
		d.FunctionFQN, d.SourceLine, d.SourceColumn, d.SinkLine, d.SinkColumn)
}

func intersectDetections(a, b []DataflowDetection) []DataflowDetection {
	bSet := make(map[string]bool)
	for _, d := range b {
		bSet[intersectKey(d)] = true
	}
	var result []DataflowDetection
	for _, d := range a {
		if bSet[intersectKey(d)] {
			result = append(result, d)
		}
	}
	return result
}
