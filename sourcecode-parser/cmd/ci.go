package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"

	"github.com/spf13/cobra"
)

type Rule struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	Impact       string `json:"impact"`
	Severity     string `json:"severity"`
	Passed       bool   `json:"passed" default:"true"`
	Query        string `json:"query"`
	RuleProvider string `json:"rule_provider"`
}

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Scan a project for vulnerabilities with ruleset in ci mode",
	Run: func(cmd *cobra.Command, _ []string) {
		rulesetConfig := cmd.Flag("ruleset").Value.String()
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

		if rulesetConfig == "" {
			fmt.Println("ruleset are not specified. Please specify a ruleset eg: cpf/java or directory path")
			os.Exit(1)
		}

		if projectInput == "" {
			fmt.Println("Project not specified")
			os.Exit(1)
		}

		ruleset, err = loadRules(rulesetConfig, strings.HasPrefix(rulesetConfig, "cpf/"))
		if err != nil {
			if verboseFlag {
				fmt.Printf("%s - error loading rules or ruleset not found: \nStacktrace: \n%s \n", rulesetConfig, err)
			}
			os.Exit(1)
		}
		codeGraph := initializeProject(projectInput)
		for _, rule := range ruleset {
			queryInput := ParseQuery(rule)
			rulesetResult := make(map[string]interface{})
			result, err := processQuery(queryInput.Query, codeGraph, output)

			if output == "json" || output == "sarif" {
				var resultObject map[string]interface{}
				json.Unmarshal([]byte(result), &resultObject) //nolint:all
				rulesetResult["query"] = queryInput.Query
				rulesetResult["rule"] = queryInput
				rulesetResult["result"] = resultObject
				outputResult = append(outputResult, rulesetResult)
			} else {
				fmt.Println(result)
			}
			if err != nil {
				fmt.Println("Error processing query: ", err)
			}
		}

		// TODO: Add sarif file support
		if output == "json" {
			if outputFile != "" {
				if graph.IsGitHubActions() {
					// append GITHUB_WORKSPACE to output file path
					outputFile = os.Getenv("GITHUB_WORKSPACE") + "/" + outputFile
				}
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
			}
		} else if output == "sarif" {
			sarifReport, err := generateSarifReport(outputResult)
			if err != nil {
				fmt.Println("Error generating sarif report: ", err)
				os.Exit(1)
			}
			if graph.IsGitHubActions() {
				// append GITHUB_WORKSPACE to output file path
				outputFile = os.Getenv("GITHUB_WORKSPACE") + "/" + outputFile
			}
			if err := sarifReport.WriteFile(outputFile); err != nil {
				fmt.Println("Error writing sarif report: ", err)
				os.Exit(1)
			}
		}
	},
}

func generateSarifReport(results []map[string]interface{}) (*sarif.Report, error) {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return nil, err
	}
	run := sarif.NewRunWithInformationURI("CodePathFinder", "https://codepathfinder.dev")
	for _, result := range results {
		localresult := result["result"].(map[string]interface{}) //nolint:all
		resultSet := localresult["result_set"].([]interface{})   //nolint:all
		pb := sarif.NewPropertyBag()
		rule := result["rule"].(Rule) //nolint:all
		pb.Add("impact", rule.Impact)
		pb.Add("ruleProvider", rule.RuleProvider)

		run.AddRule(rule.ID).
			WithDescription(rule.Description).
			WithProperties(pb.Properties).
			WithMarkdownHelp("# markdown")

		for _, finding := range resultSet {
			findingMap := finding.(map[string]interface{}) //nolint:all
			file, _ := findingMap["file"].(string)         //nolint:all
			line, _ := findingMap["line"].(float64)        //nolint:all
			// convert line to int
			lineInt := int(line)

			run.CreateResultForRule(rule.ID).
				WithLevel(strings.ToLower(rule.Severity)).
				WithMessage(sarif.NewTextMessage(rule.Description)).
				AddLocation(
					sarif.NewLocationWithPhysicalLocation(
						sarif.NewPhysicalLocation().
							WithArtifactLocation(
								sarif.NewSimpleArtifactLocation(file),
							).WithRegion(
							sarif.NewSimpleRegion(lineInt, lineInt),
						),
					),
				)
		}
	}
	report.AddRun(run)
	return report, nil
}

func init() {
	rootCmd.AddCommand(ciCmd)
	ciCmd.Flags().StringP("output", "o", "", "Supported output format: json, sarif")
	ciCmd.Flags().StringP("output-file", "f", "", "Output file path")
	ciCmd.Flags().StringP("project", "p", "", "Source code to analyze")
	ciCmd.Flags().StringP("ruleset", "r", "", "Ruleset to use example: cfp/java or directory path")
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

func ParseQuery(query string) Rule {
	// split query into lines
	lines := strings.Split(query, "\n")
	findLineFound := false
	commentLineFound := false
	query = ""
	comment := ""
	rule := Rule{}
	for _, line := range lines {
		// check if line starts with :
		if strings.HasPrefix(strings.TrimSpace(line), "/*") { //nolint:all
			comment += line
			commentLineFound = true
		} else if strings.HasPrefix(strings.TrimSpace(line), "predicate") || strings.HasPrefix(strings.TrimSpace(line), "FROM") {
			findLineFound = true
			query += line + " "
		} else if findLineFound {
			query += line + " "
		} else if commentLineFound {
			comment += line
			key, value := ParseCommentLine(line)
			switch key {
			case "@id":
				rule.ID = value
			case "@description":
				rule.Description = value
			case "@problem.severity":
				rule.Severity = value
			case "@security-severity":
				rule.Impact = value
			case "@ruleprovider":
				rule.RuleProvider = value
			}
		} else if strings.HasPrefix(strings.TrimSpace(line), "*/") {
			commentLineFound = false
		}
	}
	rule.Query = strings.TrimSpace(query)
	return rule
}

func ParseCommentLine(line string) (key, value string) {
	// parse comment start with "* @name <VALUE_MAY_CONTAIN_SPACE>"
	comment := strings.TrimSpace(line)
	comment = strings.TrimPrefix(comment, "*")
	comment = strings.TrimSpace(comment)
	parts := strings.Split(comment, " ")
	if len(parts) > 1 {
		return parts[0], strings.Join(parts[1:], " ")
	}
	return "", ""
}
