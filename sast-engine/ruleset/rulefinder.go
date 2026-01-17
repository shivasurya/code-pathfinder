package ruleset

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RuleFinder finds Python rule files by rule ID in a rules directory.
type RuleFinder struct {
	rulesDir string
}

// NewRuleFinder creates a new RuleFinder.
func NewRuleFinder(rulesDir string) *RuleFinder {
	return &RuleFinder{
		rulesDir: rulesDir,
	}
}

// FindRuleFile searches for a Python file containing the specified rule ID.
// Returns the absolute path to the file, or an error if not found.
func (rf *RuleFinder) FindRuleFile(spec *RuleSpec) (string, error) {
	languageDir := filepath.Join(rf.rulesDir, spec.Language)

	// Check if language directory exists
	if _, err := os.Stat(languageDir); os.IsNotExist(err) {
		return "", fmt.Errorf("language directory not found: %s", languageDir)
	}

	var foundFile string

	// Walk through all subdirectories
	err := filepath.Walk(languageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Python files
		if info.IsDir() || !strings.HasSuffix(path, ".py") {
			return nil
		}

		// Skip __init__.py and other special files
		if strings.HasPrefix(info.Name(), "__") {
			return nil
		}

		// Check if this file contains the rule ID
		// Ignore errors from individual files and continue searching
		contains, _ := fileContainsRuleID(path, spec.RuleID)

		if contains {
			foundFile = path
			return filepath.SkipDir // Stop searching once found
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error searching for rule: %w", err)
	}

	if foundFile == "" {
		return "", fmt.Errorf("rule %s not found in %s", spec.RuleID, languageDir)
	}

	return foundFile, nil
}

// fileContainsRuleID checks if a Python file contains a decorator with the specified rule ID.
// Looks for patterns like: id="DOCKER-BP-007".
func fileContainsRuleID(filePath string, ruleID string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Look for: id="RULE-ID" or id='RULE-ID'
	searchPattern := fmt.Sprintf(`id="%s"`, ruleID)
	searchPatternAlt := fmt.Sprintf(`id='%s'`, ruleID)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, searchPattern) || strings.Contains(line, searchPatternAlt) {
			return true, nil
		}
	}

	return false, scanner.Err()
}
