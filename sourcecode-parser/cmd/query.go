package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

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
			fmt.Println(result)
			if err != nil {
				analytics.ReportEvent(analytics.ErrorProcessingQuery)
				return "", fmt.Errorf("error processing query: %w", err)
			}
		}
	} else {
		// read from command line
		result, err := processQuery(query, codeGraph, output)
		if err != nil {
			analytics.ReportEvent(analytics.ErrorProcessingQuery)
			return "", fmt.Errorf("error processing query: %w", err)
		}
		return result, nil
	}
}

func processQuery(input string, codeGraph *graph.CodeGraph, output string) (string, error) {
	fmt.Println("Executing query: " + input)
	parsedQuery := parser.ParseQuery(input)
	// split the input string if WHERE
	parsedQuery.Expression = strings.Split(input, "WHERE")[1]
	entities := graph.QueryEntities(codeGraph, parsedQuery)
	if output == "json" {
		analytics.ReportEvent(analytics.QueryCommandJSON)
		// convert struct to query_results
		results := []map[string]interface{}{}
		for _, entity := range entities {
			result := make(map[string]interface{})
			result["file"] = entity.File
			result["line"] = entity.LineNumber
			result["code"] = entity.CodeSnippet
			results = append(results, result)
		}
		queryResults, err := json.Marshal(results)
		if err != nil {
			return "", fmt.Errorf("error processing query results: %w", err)
		}
		return string(queryResults), nil
	}
	result := ""
	for _, entity := range entities {
		// add blockquotes to string
		result += entity.File + ":" + strconv.Itoa(int(entity.LineNumber)) + "\n"
		result += "------------\n"
		result += "> " + entity.CodeSnippet + "\n"
		result += "------------"
		result += "\n"
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
		if strings.HasPrefix(strings.TrimSpace(line), "FIND") {
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
