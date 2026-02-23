package dsl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Logger interface for verbose logging (avoids import cycle with output package).
type Logger interface {
	Debug(format string, args ...any)
	Statistic(format string, args ...any)
	IsDebug() bool
	IsVerbose() bool
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", filePath)

	// Execute Python script with context
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python rule execution timed out after 30s for file: %s", filePath)
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
			// Silently skip files that fail to load (may be container rules)
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
	// Check for rule decorator or codepathfinder imports
	return strings.Contains(fileContent, "@rule(") ||
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

// LoadContainerRules loads container rules (Dockerfile/Compose) from Python DSL files.
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
			return nil, fmt.Errorf("Python rule execution timed out after 30s")
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

func (l *RuleLoader) executeDataflow(matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
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

//nolint:unparam // Will be implemented in future PRs
func (l *RuleLoader) executeLogic(logicType string, matcherMap map[string]any, cg *core.CallGraph) ([]DataflowDetection, error) {
	// TODO: Handle And/Or/Not logic operators
	// This requires recursive execution of nested matchers
	// For now, return empty detections as placeholder
	return []DataflowDetection{}, nil
}
