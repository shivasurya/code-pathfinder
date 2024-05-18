package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"queryparser"
	"strings"
)

func processQuery(input string, graph *CodeGraph, output string) (string, error) {
	fmt.Println("Executing query: " + input)
	lex := queryparser.NewLexer(input)
	pars := queryparser.NewParser(lex)
	query := pars.ParseQuery()
	if query == nil {
		return "", fmt.Errorf("failed to parse query: %v", pars.Errors())
	}
	entities := QueryEntities(graph, query)
	if output == "json" {
		// convert struct to query_results
		queryResults, err := json.Marshal(entities)
		if err != nil {
			return "", fmt.Errorf("error processing query results: %w", err)
		}
		return string(queryResults), nil
	}
	var result strings.Builder
	result.WriteString("------Query Results------\n")
	for _, entity := range entities {
		result.WriteString("-------------------\n")
		result.WriteString(entity.CodeSnippet + "\n")
		result.WriteString(entity.File + "\n")
		result.WriteString("-------------------\n")
	}
	result.WriteString("-------------------\n")
	return result.String(), nil
}

func executeProject(project, query, output string, stdin bool) (string, error) {
	graph := NewCodeGraph()
	if project != "" {
		graph = Initialize(project)
	}

	if stdin {
		// read from stdin
		for {
			fmt.Print("Path-Finder Query Console: \n>")
			in := bufio.NewReader(os.Stdin)

			input, err := in.ReadString('\n')
			if err != nil {
				return "", fmt.Errorf("error processing query: %w", err)
			}
			// if input starts with :quit string
			if strings.HasPrefix(input, ":quit") {
				return "Okay, Bye!", nil
			}
			result, err := processQuery(input, graph, output)
			if err != nil {
				return "", fmt.Errorf("error processing query: %w", err)
			}
			fmt.Println(result)
		}
	} else if output != "" && query != "" {
		return processQuery(query, graph, output)
	}
	return "", fmt.Errorf("output and query parameters are required")
}

func main() {
	// accept command line param optional path to source code
	output := flag.String("output", "", "Supported output format: json")
	outputFile := flag.String("output-file", "", "Output file path")
	query := flag.String("query", "", "Query to execute")
	project := flag.String("project", "", "Project to analyze")
	stdin := flag.Bool("stdin", false, "Read query from stdin")
	flag.Parse()

	result, err := executeProject(*project, *query, *output, *stdin)
	if err != nil {
		fmt.Println(err)
		return
	}
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Println("Error creating output file: ", err)
			return
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Println("Error closing output file: ", err)
				return
			}
		}(file)
		_, err = file.WriteString(result)
		if err != nil {
			fmt.Println("Error writing output file: ", err)
			return
		}
	} else {
		fmt.Println(result)
	}
}
