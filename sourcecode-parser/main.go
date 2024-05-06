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

func processQuery(input string, graph *CodeGraph, output string) {
	lex := queryparser.NewLexer(input)
	pars := queryparser.NewParser(lex)
	query := pars.ParseQuery()
	if query == nil {
		fmt.Println("Failed to parse query:")
		for _, err := range pars.Errors() {
			fmt.Println(err)
		}
	} else {
		entities := QueryEntities(graph, query)
		if output == "json" {
			// convert struct to query_results
			queryResults, err := json.Marshal(entities)
			if err != nil {
				fmt.Println("Error processing query results")
			} else {
				fmt.Println(string(queryResults))
			}
		} else {
			fmt.Println("------Query Results------")
			for _, entity := range entities {
				fmt.Println(entity.CodeSnippet)
			}
			fmt.Println("-------------------")
		}
	}
}

func main() {
	// accept command line param optional path to source code
	graph := NewCodeGraph()
	output := flag.String("output", "", "Supported output format: json")
	query := flag.String("query", "", "Query to execute")
	project := flag.String("project", "", "Project to analyze")
	flag.Parse()
	// loop to accept queries
	if project != nil {
		graph = Initialize(*project)
	}
	// check if output and query are provided
	if output != nil && query != nil && *output != "" && *query != "" {
		processQuery(*query, graph, *output)
	} else {
		for {
			var input string
			fmt.Print("Path-Finder Query Console: \n>")
			in := bufio.NewReader(os.Stdin)

			input, err := in.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input")
				return
			}
			// if input starts with :quit string
			if strings.HasPrefix(input, ":quit") {
				return
			}
			fmt.Print("Executing query: " + input)
			processQuery(input, graph, "text")
		}
	}
}
