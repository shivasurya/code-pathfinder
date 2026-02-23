package executor

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/docker"
)

// ContainerRuleExecutor executes container security rules.
type ContainerRuleExecutor struct {
	dockerfileRules []CompiledRule
	composeRules    []CompiledRule
}

// CompiledRule represents a parsed rule from JSON IR.
// JSON tags use snake_case to match Python IR format.
//
//nolint:tagliatelle
type CompiledRule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Severity    string         `json:"severity"`
	Category    string         `json:"category"`
	CWE         string         `json:"cwe"`
	Message     string         `json:"message"`
	FilePattern string         `json:"file_pattern"`
	RuleType    string         `json:"rule_type"`
	Matcher     map[string]any `json:"matcher"`
}

// RuleMatch represents a matched security issue.
// JSON tags use snake_case to match expected output format.
//
//nolint:tagliatelle
type RuleMatch struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Severity    string `json:"severity"`
	CWE         string `json:"cwe"`
	Message     string `json:"message"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	ServiceName string `json:"service_name,omitempty"` // For compose rules
}

// LoadRules loads rules from JSON IR.
func (e *ContainerRuleExecutor) LoadRules(jsonIR []byte) error {
	var rules struct {
		Dockerfile []CompiledRule `json:"dockerfile"`
		Compose    []CompiledRule `json:"compose"`
	}

	if err := json.Unmarshal(jsonIR, &rules); err != nil {
		return err
	}

	e.dockerfileRules = rules.Dockerfile
	e.composeRules = rules.Compose
	return nil
}

// ExecuteDockerfile runs all Dockerfile rules against a parsed Dockerfile.
func (e *ContainerRuleExecutor) ExecuteDockerfile(
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	matches := make([]RuleMatch, 0, len(e.dockerfileRules))

	for _, rule := range e.dockerfileRules {
		ruleMatches := e.evaluateDockerfileRule(rule, dockerfile)
		matches = append(matches, ruleMatches...)
	}

	return matches
}

// ExecuteCompose runs all compose rules against a parsed docker-compose.
func (e *ContainerRuleExecutor) ExecuteCompose(
	compose *graph.ComposeGraph,
) []RuleMatch {
	matches := make([]RuleMatch, 0)

	for _, rule := range e.composeRules {
		for serviceName := range compose.Services {
			if match := e.evaluateComposeRule(rule, compose, serviceName); match != nil {
				matches = append(matches, *match)
			}
		}
	}

	return matches
}

func (e *ContainerRuleExecutor) evaluateDockerfileRule(
	rule CompiledRule,
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	matcherType, ok := rule.Matcher["type"].(string)
	if !ok {
		return nil
	}

	switch matcherType {
	case "missing_instruction":
		return e.evaluateMissingInstruction(rule, dockerfile)
	case "instruction":
		return e.evaluateInstruction(rule, dockerfile)
	case "all_of":
		return e.evaluateAllOf(rule, dockerfile)
	case "any_of":
		return e.evaluateAnyOf(rule, dockerfile)
	case "none_of":
		return e.evaluateNoneOf(rule, dockerfile)
	}

	return nil
}

func (e *ContainerRuleExecutor) evaluateMissingInstruction(
	rule CompiledRule,
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	instType, ok := rule.Matcher["instruction"].(string)
	if !ok {
		return nil
	}

	if !dockerfile.HasInstruction(instType) {
		return []RuleMatch{{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			Severity:   rule.Severity,
			CWE:        rule.CWE,
			Message:    rule.Message,
			FilePath:   dockerfile.FilePath,
			LineNumber: 1, // File-level issue
		}}
	}

	return nil
}

func (e *ContainerRuleExecutor) evaluateInstruction(
	rule CompiledRule,
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	instType, ok := rule.Matcher["instruction"].(string)
	if !ok {
		return nil
	}

	nodes := dockerfile.GetInstructions(instType)
	matches := make([]RuleMatch, 0)

	for _, node := range nodes {
		if e.matchesInstructionCriteria(rule.Matcher, node) {
			matches = append(matches, RuleMatch{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				Severity:   rule.Severity,
				CWE:        rule.CWE,
				Message:    rule.Message,
				FilePath:   dockerfile.FilePath,
				LineNumber: node.LineNumber,
			})
		}
	}

	return matches
}

func (e *ContainerRuleExecutor) matchesInstructionCriteria(
	matcher map[string]any,
	node *docker.DockerfileNode,
) bool {
	// Check image_tag
	if tag, ok := matcher["image_tag"].(string); ok {
		if node.ImageTag != tag {
			return false
		}
	}

	// Check user_name
	if userName, ok := matcher["user_name"].(string); ok {
		if node.UserName != userName {
			return false
		}
	}

	// Check arg_name_regex
	if argRegex, ok := matcher["arg_name_regex"].(string); ok {
		re, err := regexp.Compile(argRegex)
		if err != nil {
			return false
		}
		if !re.MatchString(node.ArgName) {
			return false
		}
	}

	// Check contains
	if contains, ok := matcher["contains"].(string); ok {
		if !strings.Contains(node.RawInstruction, contains) {
			return false
		}
	}

	// Check not_contains
	if notContains, ok := matcher["not_contains"].(string); ok {
		if strings.Contains(node.RawInstruction, notContains) {
			return false
		}
	}

	// Check port_less_than
	if portLT, ok := matcher["port_less_than"].(float64); ok {
		hasMatch := false
		for _, port := range node.Ports {
			if port < int(portLT) {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			return false
		}
	}

	// Check port_greater_than
	if portGT, ok := matcher["port_greater_than"].(float64); ok {
		hasMatch := false
		for _, port := range node.Ports {
			if port > int(portGT) {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			return false
		}
	}

	// Check missing_digest
	if missingDigest, ok := matcher["missing_digest"].(bool); ok {
		if missingDigest && node.ImageDigest != "" {
			return false
		}
		if !missingDigest && node.ImageDigest == "" {
			return false
		}
	}

	// Check base_image
	if baseImage, ok := matcher["base_image"].(string); ok {
		if node.BaseImage != baseImage {
			return false
		}
	}

	return true
}

func (e *ContainerRuleExecutor) evaluateComposeRule(
	rule CompiledRule,
	compose *graph.ComposeGraph,
	serviceName string,
) *RuleMatch {
	matcherType, ok := rule.Matcher["type"].(string)
	if !ok {
		return nil
	}

	switch matcherType {
	case "service_has":
		return e.evaluateServiceHas(rule, compose, serviceName)
	case "service_missing":
		return e.evaluateServiceMissing(rule, compose, serviceName)
	}

	return nil
}

func (e *ContainerRuleExecutor) evaluateServiceHas(
	rule CompiledRule,
	compose *graph.ComposeGraph,
	serviceName string,
) *RuleMatch {
	key, ok := rule.Matcher["key"].(string)
	if !ok {
		return nil
	}

	// Check equals
	if equals, ok := rule.Matcher["equals"]; ok {
		if compose.ServiceHas(serviceName, key, equals) {
			return &RuleMatch{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				CWE:         rule.CWE,
				Message:     rule.Message,
				FilePath:    compose.FilePath,
				ServiceName: serviceName,
				LineNumber:  compose.ServiceGetLineNumber(serviceName, key),
			}
		}
	}

	// Check contains (single string)
	if contains, ok := rule.Matcher["contains"].(string); ok {
		value := compose.ServiceGet(serviceName, key)
		lineNumber := compose.ServiceGetLineNumber(serviceName, key)
		// Handle string value
		if valueStr, ok := value.(string); ok {
			if strings.Contains(valueStr, contains) {
				return &RuleMatch{
					RuleID:      rule.ID,
					RuleName:    rule.Name,
					Severity:    rule.Severity,
					CWE:         rule.CWE,
					Message:     rule.Message,
					FilePath:    compose.FilePath,
					ServiceName: serviceName,
					LineNumber:  lineNumber,
				}
			}
		}
		// Handle array value
		if valueList, ok := value.([]any); ok {
			for _, v := range valueList {
				if vStr, ok := v.(string); ok && strings.Contains(vStr, contains) {
					return &RuleMatch{
						RuleID:      rule.ID,
						RuleName:    rule.Name,
						Severity:    rule.Severity,
						CWE:         rule.CWE,
						Message:     rule.Message,
						FilePath:    compose.FilePath,
						ServiceName: serviceName,
						LineNumber:  lineNumber,
					}
				}
			}
		}
	}

	// Check contains_any
	if containsAny, ok := rule.Matcher["contains_any"].([]any); ok {
		lineNumber := compose.ServiceGetLineNumber(serviceName, key)
		for _, val := range containsAny {
			valStr, ok := val.(string)
			if !ok {
				continue
			}
			// Check volumes for string match
			volumes := compose.ServiceGet(serviceName, key)
			if volumeList, ok := volumes.([]any); ok {
				for _, v := range volumeList {
					if vStr, ok := v.(string); ok && strings.Contains(vStr, valStr) {
						return &RuleMatch{
							RuleID:      rule.ID,
							RuleName:    rule.Name,
							Severity:    rule.Severity,
							CWE:         rule.CWE,
							Message:     rule.Message,
							FilePath:    compose.FilePath,
							ServiceName: serviceName,
							LineNumber:  lineNumber,
						}
					}
				}
			}
		}
	}

	return nil
}

func (e *ContainerRuleExecutor) evaluateServiceMissing(
	rule CompiledRule,
	compose *graph.ComposeGraph,
	serviceName string,
) *RuleMatch {
	key, ok := rule.Matcher["key"].(string)
	if !ok {
		return nil
	}

	if !compose.ServiceHasKey(serviceName, key) {
		// For missing properties, point to service declaration line
		serviceLineNumber := compose.ServiceGetLineNumber(serviceName, "")
		return &RuleMatch{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Severity:    rule.Severity,
			CWE:         rule.CWE,
			Message:     rule.Message,
			FilePath:    compose.FilePath,
			ServiceName: serviceName,
			LineNumber:  serviceLineNumber,
		}
	}

	return nil
}

// Combinator evaluators.
func (e *ContainerRuleExecutor) evaluateAllOf(
	rule CompiledRule,
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	conditions, ok := rule.Matcher["conditions"].([]any)
	if !ok {
		return nil
	}

	// Track first match to get line number
	var firstMatches []RuleMatch

	// All conditions must match
	for _, cond := range conditions {
		condMap, ok := cond.(map[string]any)
		if !ok {
			return nil
		}

		tempRule := CompiledRule{
			ID:       rule.ID,
			Name:     rule.Name,
			Severity: rule.Severity,
			CWE:      rule.CWE,
			Message:  rule.Message,
			Matcher:  condMap,
		}

		matches := e.evaluateDockerfileRule(tempRule, dockerfile)
		if len(matches) == 0 {
			// One condition didn't match, so all_of fails
			return nil
		}

		// Capture first condition's matches to get line numbers
		if len(firstMatches) == 0 {
			firstMatches = matches
		}
	}

	// All conditions matched, return first condition's matches
	return firstMatches
}

func (e *ContainerRuleExecutor) evaluateAnyOf(
	rule CompiledRule,
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	conditions, ok := rule.Matcher["conditions"].([]any)
	if !ok {
		return nil
	}

	// Collect matches from all conditions
	allMatches := make([]RuleMatch, 0)
	for _, cond := range conditions {
		condMap, ok := cond.(map[string]any)
		if !ok {
			continue
		}

		tempRule := CompiledRule{
			ID:       rule.ID,
			Name:     rule.Name,
			Severity: rule.Severity,
			CWE:      rule.CWE,
			Message:  rule.Message,
			Matcher:  condMap,
		}

		matches := e.evaluateDockerfileRule(tempRule, dockerfile)
		allMatches = append(allMatches, matches...)
	}

	return allMatches
}

func (e *ContainerRuleExecutor) evaluateNoneOf(
	rule CompiledRule,
	dockerfile *docker.DockerfileGraph,
) []RuleMatch {
	conditions, ok := rule.Matcher["conditions"].([]any)
	if !ok {
		return nil
	}

	// Collect all violations (matches that should NOT have happened)
	violations := make([]RuleMatch, 0)
	for _, cond := range conditions {
		condMap, ok := cond.(map[string]any)
		if !ok {
			continue
		}

		tempRule := CompiledRule{
			ID:       rule.ID,
			Name:     rule.Name,
			Severity: rule.Severity,
			CWE:      rule.CWE,
			Message:  rule.Message,
			Matcher:  condMap,
		}

		matches := e.evaluateDockerfileRule(tempRule, dockerfile)
		// Each match is a violation of the none_of condition
		for _, match := range matches {
			violations = append(violations, RuleMatch{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				Severity:   rule.Severity,
				CWE:        rule.CWE,
				Message:    rule.Message,
				FilePath:   dockerfile.FilePath,
				LineNumber: match.LineNumber,
			})
		}
	}

	return violations
}
