package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var verboseFlag bool

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Scan a project for vulnerabilities with ruleset in ci mode",
	Run: func(cmd *cobra.Command, _ []string) {
		rulesetConfig := cmd.Flag("ruleset").Value.String()
		rulesetDirectory := cmd.Flag("rules-directory").Value.String()
		projectInput := cmd.Flag("project").Value.String()
		output := cmd.Flag("output").Value.String()
		outputFile := cmd.Flag("output-file").Value.String()
		verboseFlag, _ = cmd.Flags().GetBool("verbose") //nolint:all

		if verboseFlag {
			fmt.Println("Executing in CI mode")
		}

		if rulesetConfig == "" || rulesetDirectory == "" {
			fmt.Println("Ruleset or rules directory not specified")
			os.Exit(1)
		}

		if projectInput == "" {
			fmt.Println("Project not specified")
			os.Exit(1)
		}

		if !strings.HasPrefix(rulesetConfig, "cpf/") {
			fmt.Println("Ruleset not specified")
			os.Exit(1)
		}
		ruleset, err := loadRules(rulesetConfig, rulesetConfig != "")
		if err != nil {
			fmt.Println("Error loading rules: ", err)
			os.Exit(1)
		}
		codeGraph := initializeProject(projectInput)
		for _, rule := range ruleset {
			queryInput := ParseQuery(rule)
			result, err := processQuery(queryInput, codeGraph, output)
			if err != nil {
				fmt.Println("Error processing query: ", err)
				os.Exit(1)
			}
			fmt.Println(outputFile)
			fmt.Println(result)
		}
	},
}

func init() {
	rootCmd.AddCommand(ciCmd)
	ciCmd.Flags().StringP("output", "o", "", "Supported output format: json")
	ciCmd.Flags().StringP("output-file", "f", "", "Output file path")
	ciCmd.Flags().StringP("project", "p", "", "Project to analyze")
	ciCmd.Flags().StringP("ruleset", "q", "", "Ruleset to use example: cfp/java")
	ciCmd.Flags().Bool("rules-directory", false, "Ruleset directory")
}

func loadRules(rulesDirectory string, isHosted bool) ([]string, error) {
	var rules []string

	if isHosted {
		rules = downloadRuleset(rulesDirectory)
	} else {
		entries, err := os.ReadDir(rulesDirectory)
		// check if rules directory exists
		if err != nil {
			fmt.Println("Rules directory does not exist")
			return nil, err
		}
		// read all cql files in rules directory
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// check if file is a cql file
			if strings.HasSuffix(entry.Name(), ".cql") {
				// read file contents
				contents, err := os.ReadFile(filepath.Join(rulesDirectory, entry.Name()))
				if err != nil {
					fmt.Println("Error reading file: ", err)
					continue
				}
				// add file contents to rules
				rules = append(rules, string(contents))
			}
		}
	}

	return rules, nil
}

func downloadRuleset(ruleset string) []string {
	rules := []string{}
	ruleset = strings.TrimPrefix(ruleset, "cpf/")
	url := "https://codepathfinder.dev/rules/" + ruleset + ".json"
	//nolint:all
	resp, err := http.Get(url)
	if err != nil {
		err := fmt.Errorf("error downloading ruleset: %w", err)
		fmt.Println(err)
		return rules
	}
	defer resp.Body.Close()
	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("error reading response body: %w", err)
		fmt.Println(err)
		return rules
	}
	// parse response body
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error parsing response body: ", err)
		return rules
	}
	// add rules to rules
	if files, ok := response["files"].([]interface{}); ok {
		for _, file := range files {
			if rule, ok := file.(map[string]interface{}); ok {
				if content, ok := rule["content"].(string); ok {
					rules = append(rules, content)
				}
			}
		}
	}
	return rules
}

func ParseQuery(query string) string {
	// split query into lines
	lines := strings.Split(query, "\n")
	findLineFound := false
	query = ""
	for _, line := range lines {
		// check if line starts with :
		if strings.HasPrefix(strings.TrimSpace(line), "predicate") || strings.HasPrefix(strings.TrimSpace(line), "FROM") {
			findLineFound = true
			query += line + " "
		} else if findLineFound {
			query += line + " "
		}
	}
	query = strings.TrimSpace(query)
	return query
}
