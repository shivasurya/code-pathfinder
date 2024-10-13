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

		var ruleset []string
		var outputResult []map[string]interface{}
		var err error

		if verboseFlag {
			fmt.Println("Executing in CI mode")
		}

		if rulesetConfig == "" && rulesetDirectory == "" {
			fmt.Println("Ruleset or rules directory not specified")
			os.Exit(1)
		}

		if projectInput == "" {
			fmt.Println("Project not specified")
			os.Exit(1)
		}

		if rulesetConfig != "" {
			if !strings.HasPrefix(rulesetConfig, "cpf/") {
				fmt.Println("Ruleset not specified")
				os.Exit(1)
			}
			ruleset, err = loadRules(rulesetConfig, true)
			if err != nil {
				if verboseFlag {
					fmt.Printf("%s - error loading rules or ruleset not found: \nStacktrace: \n%s \n", rulesetConfig, err)
				}
				os.Exit(1)
			}
		} else if rulesetDirectory != "" {
			ruleset, err = loadRules(rulesetDirectory, false)
			if err != nil {
				if verboseFlag {
					fmt.Printf("%s - error loading rules or ruleset not found: \nStacktrace: \n%s \n", rulesetDirectory, err)
				}
				os.Exit(1)
			}
		}
		codeGraph := initializeProject(projectInput)
		for _, rule := range ruleset {
			queryInput := ParseQuery(rule)
			// create json object with queryinput and result
			rulesetResult := make(map[string]interface{})
			result, err := processQuery(queryInput, codeGraph, output)
			// unstringify json result
			var resultObject map[string]interface{}
			json.Unmarshal([]byte(result), &resultObject) //nolint:all
			rulesetResult["query"] = queryInput
			rulesetResult["result"] = resultObject
			outputResult = append(outputResult, rulesetResult)
			if err != nil {
				fmt.Println("Error processing query: ", err)
			}
		}

		// TODO: Add sarif file support
		if output == "json" {
			if outputFile != "" {
				file, err := os.Create(outputFile)
				if err != nil {
					fmt.Println("Error creating output file: ", err)
				}
				defer func(file *os.File) {
					err := file.Close()
					if err != nil {
						fmt.Println("Error closing output file: ", err)
						os.Exit(1)
					}
				}(file)
				// convert outputResult to json
				outputResultJSON, err := json.MarshalIndent(outputResult, "", "  ")
				if err != nil {
					fmt.Println("Error converting output to json: ", err)
				}
				_, err = file.WriteString(string(outputResultJSON))
				if err != nil {
					fmt.Println("Error writing output file: ", err)
				}
			} else {
				fmt.Println(outputResult)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(ciCmd)
	ciCmd.Flags().StringP("output", "o", "", "Supported output format: json")
	ciCmd.Flags().StringP("output-file", "f", "", "Output file path")
	ciCmd.Flags().StringP("project", "p", "", "Project to analyze")
	ciCmd.Flags().StringP("ruleset", "q", "", "Ruleset to use example: cfp/java")
	ciCmd.Flags().StringP("rules-directory", "r", "", "Rules directory to use")
}

func loadRules(rulesDirectory string, isHosted bool) ([]string, error) {
	var rules []string
	var err error

	if isHosted {
		rules, err = downloadRuleset(rulesDirectory)
		if err != nil {
			return nil, err
		}
	} else {
		err = filepath.Walk(rulesDirectory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".cql") {
				contents, err := os.ReadFile(path)
				if err != nil {
					fmt.Printf("Error reading file %s: %v\n", path, err)
					return nil
				}
				rules = append(rules, string(contents))
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking through rules directory: %w", err)
		}
	}

	return rules, nil
}

func downloadRuleset(ruleset string) ([]string, error) {
	rules := []string{}
	ruleset = strings.TrimPrefix(ruleset, "cpf/")
	url := "https://codepathfinder.dev/rules/" + ruleset + ".json"
	//nolint:all
	resp, err := http.Get(url)
	if err != nil {
		err := fmt.Errorf("error downloading ruleset: %w", err)
		return nil, err
	}
	defer resp.Body.Close()
	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("error downloading ruleset: %w", err)
		return nil, err
	}
	// parse response body
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		err := fmt.Errorf("error downloading ruleset: %w", err)
		return nil, err
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
	return rules, nil
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
