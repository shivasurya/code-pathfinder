package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan a project for vulnerabilities with ruleset",
	Run: func(cmd *cobra.Command, _ []string) {
		ruleset := cmd.Flag("ruleset").Value.String()
		project := cmd.Flag("project").Value.String()

		if ruleset == "" {
			fmt.Println("Please provide a ruleset directory path")
			return
		}

		if project == "" {
			fmt.Println("Please provide a project directory path")
			return
		}

		queryFiles := getAllRulesetFile(ruleset)

		for _, queryFile := range queryFiles {
			extractedQuery, err := ExtractQueryFromFile(queryFile)
			if err != nil {
				fmt.Println("Error extracting query from file:", err)
				return
			}
			result, err := executeCLIQuery(project, extractedQuery, "json", false)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(result)
		}
	},
}

func getAllRulesetFile(directory string) []string {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if filepath.Ext(path) == ".cql" {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Please provide a valid ruleset directory path")
		return nil
	}
	return files
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringP("ruleset", "o", "", "Ruleset directory path")
	scanCmd.Flags().StringP("project", "f", "", "project to scan path")
}
