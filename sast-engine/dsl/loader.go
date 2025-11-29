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

// RuleLoader loads Python DSL rules and executes them.
type RuleLoader struct {
	RulesPath string // Path to .py rules file or directory
}

// NewRuleLoader creates a new rule loader.
func NewRuleLoader(rulesPath string) *RuleLoader {
	return &RuleLoader{RulesPath: rulesPath}
}

// isSandboxEnabled checks if nsjail sandboxing is enabled via environment variable.
// Returns true if PATHFINDER_SANDBOX_ENABLED is set to "true" (case-insensitive).
func isSandboxEnabled() bool {
	enabled := os.Getenv("PATHFINDER_SANDBOX_ENABLED")
	return strings.ToLower(strings.TrimSpace(enabled)) == "true"
}

// buildNsjailCommand constructs an nsjail command for sandboxed Python execution.
// Security features:
//   - Network isolation (--iface_no_lo)
//   - Filesystem isolation (chroot to /tmp/nsjail_root)
//   - Process isolation (PID namespace)
//   - User isolation (run as nobody)
//   - Resource limits: 512MB memory, 30s CPU, 1MB file size, 30s wall time
//   - Read-only system mounts (/usr, /lib)
//   - Writable /tmp for output
func buildNsjailCommand(ctx context.Context, filePath string) *exec.Cmd {
	args := []string{
		"-Mo",                          // Mode: ONCE (run once and exit)
		"--user", "nobody",             // Run as nobody (UID 65534)
		"--chroot", "/tmp/nsjail_root", // Isolated root filesystem
		"--iface_no_lo",                // Block all network access (no loopback)
		"--disable_proc",               // Disable /proc (no process visibility)
		"--bindmount_ro", "/usr:/usr",  // Read-only /usr
		"--bindmount_ro", "/lib:/lib",  // Read-only /lib
		"--bindmount", "/tmp:/tmp",     // Writable /tmp (for output)
		"--cwd", "/tmp",                // Working directory
		"--rlimit_as", "512",           // Memory limit: 512MB
		"--rlimit_cpu", "30",           // CPU time limit: 30 seconds
		"--rlimit_fsize", "1",          // File size limit: 1MB
		"--rlimit_nofile", "64",        // Max open files: 64
		"--time_limit", "30",           // Wall time limit: 30 seconds
		"--quiet",                      // Suppress nsjail logs
		"--",                           // End of nsjail args
		"/usr/bin/python3", filePath,   // Command to execute
	}

	return exec.CommandContext(ctx, "nsjail", args...)
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
func (l *RuleLoader) LoadRules() ([]RuleIR, error) {
	// Check if path is file or directory
	info, err := os.Stat(l.RulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access rules path: %w", err)
	}

	// If single file, load directly
	if !info.IsDir() {
		return l.loadRulesFromFile(l.RulesPath)
	}

	// If directory, find all .py files and load them
	return l.loadRulesFromDirectory(l.RulesPath)
}

// loadRulesFromFile loads rules from a single Python file.
// Uses nsjail sandboxing if PATHFINDER_SANDBOX_ENABLED=true, otherwise runs Python directly.
func (l *RuleLoader) loadRulesFromFile(filePath string) ([]RuleIR, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build command based on sandbox configuration
	var cmd *exec.Cmd
	if isSandboxEnabled() {
		// Use nsjail for sandboxed execution (production mode)
		cmd = buildNsjailCommand(ctx, filePath)
	} else {
		// Direct Python execution (development mode)
		cmd = exec.CommandContext(ctx, "python3", filePath)
	}

	// Execute Python script with context
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("Python rule execution timed out after 30s for file: %s", filePath)
		}
		return nil, fmt.Errorf("failed to execute Python rules from %s: %w", filePath, err)
	}

	// Parse JSON IR
	var rules []RuleIR
	if err := json.Unmarshal(output, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rule JSON IR from %s: %w", filePath, err)
	}

	return rules, nil
}

// loadRulesFromDirectory loads rules from all .py files in a directory.
func (l *RuleLoader) loadRulesFromDirectory(dirPath string) ([]RuleIR, error) {
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

		// Load rules from this file
		rules, err := l.loadRulesFromFile(path)
		if err != nil {
			// Log error but continue processing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to load rules from %s: %v\n", path, err)
			return nil
		}

		allRules = append(allRules, rules...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	if len(allRules) == 0 {
		return nil, fmt.Errorf("no rules found in directory: %s", dirPath)
	}

	return allRules, nil
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
