package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule IDs are uppercase alphanumeric + dash, length 1..64. Mirrors the
// shape rule authors actually use in code-pathfinder/rules (e.g.
// SAST-CMD-001, GO-SSRF-001, DOCKER-BP-005). Anything outside that
// alphabet is rejected at the boundary so weird input from a downstream
// caller (cpf-executor passing argv from a D1 lookup) can't smuggle in
// shell metacharacters or path-traversal patterns.
var ruleIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// validateDisableRules normalizes and validates a list of rule IDs to
// disable. Empty input returns nil. Whitespace is trimmed; exact
// duplicates after trim are deduped. Each remaining ID must match
// ruleIDPattern, otherwise the whole list is rejected so a single bad
// ID doesn't silently strip itself away.
func validateDisableRules(input []string) ([]string, error) {
	if len(input) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(input))
	var out []string
	for _, raw := range input {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if len(id) > 64 {
			return nil, fmt.Errorf("rule id too long: %q (max 64)", id)
		}
		if !ruleIDPattern.MatchString(id) {
			return nil, fmt.Errorf("invalid rule id %q: must match [A-Za-z0-9_-]{1,64}", id)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}
