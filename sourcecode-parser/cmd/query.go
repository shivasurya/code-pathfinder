package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
		page, _ := cmd.Flags().GetInt("page")
		size, _ := cmd.Flags().GetInt("size")

		if queryFile != "" {
			extractedQuery, err := ExtractQueryFromFile(queryFile)
			if err != nil {
				fmt.Println("Error extracting query from file:", err)
				return
			}
			queryInput = extractedQuery
		}

		result, err := executeCLIQuery(projectInput, queryInput, output, stdin, page, size)
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
	// Pagination flags – optional. Page is 1‑based; size 0 disables pagination.
	queryCmd.Flags().Int("page", 0, "Page number (1‑based, 0 for no pagination)")
	queryCmd.Flags().Int("size", 0, "Page size (number of results per page, 0 for all)")
}

func initializeProject(project string) *graph.CodeGraph {
	codeGraph := graph.NewCodeGraph()
	if project != "" {
		codeGraph = graph.Initialize(project)
	}
	return codeGraph
}

func executeCLIQuery(project, query, output string, stdin bool, page, size int) (string, error) {
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
		result, err := processQuery(input, codeGraph, output, page, size)
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
		result, err := processQuery(query, codeGraph, output, page, size)
		if err != nil {
			analytics.ReportEvent(analytics.ErrorProcessingQuery)
			return "", fmt.Errorf("PathFinder Query syntax error: %w", err)
		}
		return result, nil
	}
}

func processQuery(input string, codeGraph *graph.CodeGraph, output string, page, size int) (string, error) {
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

	// Sort results deterministically for consistent pagination.
	// Sorting by: File (primary), LineNumber (secondary), ID (tertiary).
	// Need to keep entities and formattedOutput in sync during sort.
	type resultPair struct {
		entity []*graph.Node
		output []interface{}
	}

	// Combine entities and formattedOutput into pairs
	pairs := make([]resultPair, len(entities))
	for i := range entities {
		pairs[i] = resultPair{
			entity: entities[i],
			output: formattedOutput[i],
		}
	}

	// Sort the pairs
	sort.SliceStable(pairs, func(i, j int) bool {
		// Get first node from each result set for comparison
		if len(pairs[i].entity) == 0 || len(pairs[j].entity) == 0 {
			return len(pairs[i].entity) > 0
		}
		nodeI := pairs[i].entity[0]
		nodeJ := pairs[j].entity[0]

		// Compare by File first
		if nodeI.File != nodeJ.File {
			return nodeI.File < nodeJ.File
		}
		// Then by LineNumber
		if nodeI.LineNumber != nodeJ.LineNumber {
			return nodeI.LineNumber < nodeJ.LineNumber
		}
		// Finally by ID for tie-breaking
		return nodeI.ID < nodeJ.ID
	})

	// Split pairs back into entities and formattedOutput
	for i := range pairs {
		entities[i] = pairs[i].entity
		formattedOutput[i] = pairs[i].output
	}

	// Apply pagination if requested (page is 1‑based). If size == 0, return all.
	if size > 0 && page > 0 {
		total := len(entities)
		start := (page - 1) * size
		if start >= total {
			// Empty result set for out‑of‑range page
			entities = [][]*graph.Node{}
			formattedOutput = [][]interface{}{}
		} else {
			end := start + size
			if end > total {
				end = total
			}
			entities = entities[start:end]
			formattedOutput = formattedOutput[start:end]
		}
	}
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
				result["code"] = entityObject.GetCodeSnippet()

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
			codeSnippetArray := strings.Split(entityObject.GetCodeSnippet(), "\n")
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
