package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/analytics"
	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Execute queries on the source code",
	Run: func(cmd *cobra.Command, _ []string) {
		// Implement query execution logic here
		queryFile := cmd.Flag("query-file").Value.String()
		queryInput := cmd.Flag("query").Value.String()
		projectInput := cmd.Flag("project").Value.String()
		output := cmd.Flag("output").Value.String()
		outputFile := cmd.Flag("output-file").Value.String()
		stdin, _ := cmd.Flags().GetBool("stdin") //nolint:all

		if queryFile != "" {
			extractedQuery, err := ExtractQueryFromFile(queryFile)
			if err != nil {
				fmt.Println("Error extracting query from file:", err)
				return
			}
			queryInput = extractedQuery
		}

		result, err := executeCLIQuery(projectInput, queryInput, output, stdin)
		if err != nil {
			fmt.Println(err)
		}
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
			_, err = file.WriteString(result)
			if err != nil {
				fmt.Println("Error writing output file: ", err)
			}
		} else {
			fmt.Println(result)
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringP("output", "o", "", "Supported output format: json")
	queryCmd.Flags().StringP("output-file", "f", "", "Output file path")
	queryCmd.Flags().StringP("project", "p", "", "Project to analyze")
	queryCmd.Flags().StringP("query", "q", "", "Query to execute")
	queryCmd.Flags().Bool("stdin", false, "Read query from stdin")
	queryCmd.Flags().String("query-file", "", "File containing query to execute")
}

func initializeProject(project string) *graph.CodeGraph {
	codeGraph := graph.NewCodeGraph()
	if project != "" {
		codeGraph = graph.Initialize(project)
	}
	return codeGraph
}

func executeCLIQuery(project, query, output string, stdin bool) (string, error) {
	codeGraph := initializeProject(project)

	if stdin {
		// read from stdin
		for {
			fmt.Print("Path-Finder Query Console: \n>")
			in := bufio.NewReader(os.Stdin)

			input, err := in.ReadString('\n')
			analytics.ReportEvent(analytics.QueryCommandStdin)
			if err != nil {
				return "", fmt.Errorf("error processing query: %w", err)
			}
			// if input starts with :quit string
			if strings.HasPrefix(input, ":quit") {
				return "Okay, Bye!", nil
			}
			result, err := processQuery(input, codeGraph, output)
			if err != nil {
				analytics.ReportEvent(analytics.ErrorProcessingQuery)
				err := fmt.Errorf("PathFinder Query syntax error: %w", err)
				fmt.Println(err)
			} else {
				fmt.Println(result)
			}
		}
	} else {
		// read from command line
		result, err := processQuery(query, codeGraph, output)
		if err != nil {
			analytics.ReportEvent(analytics.ErrorProcessingQuery)
			return "", fmt.Errorf("PathFinder Query syntax error: %w", err)
		}
		return result, nil
	}
}

func processQuery(input string, codeGraph *graph.CodeGraph, output string) (string, error) {
	fmt.Println("Executing query: " + input)
	parsedQuery, err := parser.ParseQuery(input)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(input, "WHERE", 2)
	if len(parts) > 1 {
		parsedQuery.Expression = strings.SplitN(parts[1], "SELECT", 2)[0]
	}
	entities, formattedOutput := graph.QueryEntities(codeGraph, parsedQuery)
	if output == "json" || output == "sarif" {
		analytics.ReportEvent(analytics.QueryCommandJSON)
		// convert struct to query_results
		results := make(map[string]interface{})
		results["result_set"] = make([]map[string]interface{}, 0)
		results["output"] = formattedOutput
		for _, entity := range entities {
			for _, entityObject := range entity {
				result := make(map[string]interface{})
				result["file"] = entityObject.File
				result["line"] = entityObject.LineNumber
				result["code"] = entityObject.CodeSnippet

				results["result_set"] = append(results["result_set"].([]map[string]interface{}), result) //nolint:all
			}
		}
		queryResults, err := json.Marshal(results)
		if err != nil {
			return "", fmt.Errorf("error processing query results: %w", err)
		}
		return string(queryResults), nil
	}
	result := ""
	verticalLine := "|"
	yellowCode := color.New(color.FgYellow).SprintFunc()
	greenCode := color.New(color.FgGreen).SprintFunc()
	for i, entity := range entities {
		for _, entityObject := range entity {
			header := fmt.Sprintf("\tFile: %s, Line: %s \n", greenCode(entityObject.File), greenCode(entityObject.LineNumber))
			// add formatted output to result
			output := "\tResult: "
			for _, outputObject := range formattedOutput[i] {
				output += graph.FormatType(outputObject)
				output += " "
				output += verticalLine + " "
			}
			header += output + "\n"
			result += header
			result += "\n"
			codeSnippetArray := strings.Split(entityObject.CodeSnippet, "\n")
			for i := 0; i < len(codeSnippetArray); i++ {
				lineNumber := color.New(color.FgCyan).SprintfFunc()("%4d", int(entityObject.LineNumber)+i)
				result += fmt.Sprintf("%s%s %s %s\n", strings.Repeat("\t", 2), lineNumber, verticalLine, yellowCode(codeSnippetArray[i]))
			}
			result += "\n"
		}
	}
	return result, nil
}

func ExtractQueryFromFile(file string) (string, error) {
	// read from file
	queryFileContent, err := os.Open(file)
	if err != nil {
		fmt.Println("Error opening file: ", err)
		return "", err
	}
	defer func(queryFileContent *os.File) {
		err := queryFileContent.Close()
		if err != nil {
			fmt.Println("Error closing file: ", err)
			os.Exit(1)
		}
	}(queryFileContent)
	query := ""
	scanner := bufio.NewScanner(queryFileContent)
	findLineFound := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "predicate") || strings.HasPrefix(strings.TrimSpace(line), "FROM") {
			findLineFound = true
			query += line + " "
		} else if findLineFound {
			query += line + " "
		}
	}
	query = strings.TrimSpace(query)
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return query, nil
}
