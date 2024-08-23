package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
)

// Version is the current version of the application.
var (
	Version            = "0.0.24"
	GitCommit          = "HEAD"
	AnalyticsPublicKey = "test"
	enableMetrics      = true
)

func main() {
	// accept command line param optional path to source code
	output := flag.String("output", "", "Supported output format: json")
	outputFile := flag.String("output-file", "", "Output file path")
	project := flag.String("project", "", "Project to analyze")
	query := flag.String("query", "", "Query to execute")
	stdin := flag.Bool("stdin", false, "Read query from stdin")
	versionFlag := flag.Bool("version", false, "Print the version information and exit")
	disableMetrics := flag.Bool("disable-metrics", false, "Disable metrics")
	queryFile := flag.String("query-file", "", "File containing query to execute")
	flag.Parse()

	if *disableMetrics {
		enableMetrics = false
	}

	loadEnvFile()

	if *versionFlag {
		reportEvent(VersionCommand, AnalyticsPublicKey)
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		return
	}

	if *queryFile != "" {
		extractedQuery, err := ExtractQueryFromFile(*queryFile)
		if err != nil {
			fmt.Println("Error extracting query from file:", err)
			return
		}
		*query = extractedQuery
	}

	result, err := executeCLIQuery(*project, *query, *output, *stdin)
	if err != nil {
		fmt.Println(err)
	}
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
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

func processQuery(input string, graph *CodeGraph, output string) (string, error) {
	fmt.Println("Executing query: " + input)
	parsedQuery := parser.ParseQuery(input)
	// split the input string if WHERE
	parsedQuery.Expression = strings.Split(input, "WHERE")[1]
	entities := QueryEntities(graph, parsedQuery)
	if output == "json" {
		reportEvent(QueryCommandJSON, AnalyticsPublicKey)
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

func InitializeProject(project string) *CodeGraph {
	graph := NewCodeGraph()
	if project != "" {
		graph = Initialize(project)
	}
	return graph
}

func executeCLIQuery(project, query, output string, stdin bool) (string, error) {
	graph := InitializeProject(project)

	if stdin {
		// read from stdin
		for {
			fmt.Print("Path-Finder Query Console: \n>")
			in := bufio.NewReader(os.Stdin)

			input, err := in.ReadString('\n')
			reportEvent(QueryCommandStdin, AnalyticsPublicKey)
			if err != nil {
				return "", fmt.Errorf("error processing query: %w", err)
			}
			// if input starts with :quit string
			if strings.HasPrefix(input, ":quit") {
				return "Okay, Bye!", nil
			}
			result, err := processQuery(input, graph, output)
			fmt.Println(result)
			if err != nil {
				reportEvent(ErrorProcessingQuery, AnalyticsPublicKey)
				return "", fmt.Errorf("error processing query: %w", err)
			}
		}
	} else {
		// read from command line
		result, err := processQuery(query, graph, output)
		if err != nil {
			reportEvent(ErrorProcessingQuery, AnalyticsPublicKey)
			return "", fmt.Errorf("error processing query: %w", err)
		}
		return result, nil
	}
}
