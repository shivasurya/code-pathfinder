package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"queryparser"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	Version   = "dev"
	GitCommit = "none"
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

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "File", "Line Number", "Type", "Name", "Code Snippet"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:     "File",
			WidthMin: 6,
			WidthMax: 40,
		},
		{
			Name:     "Code Snippet",
			WidthMin: 6,
			WidthMax: 60,
		},
	})
	for i, entity := range entities {
		t.AppendRow([]interface{}{i + 1, entity.File, entity.LineNumber, entity.Type, entity.Name, entity.CodeSnippet})
		t.AppendSeparator()
	}
	t.SetStyle(table.StyleLight)
	t.Render()
	return "", nil
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
			_, err = processQuery(input, graph, output)
			if err != nil {
				return "", fmt.Errorf("error processing query: %w", err)
			}
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
	versionFlag := flag.Bool("version", false, "Print the version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

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
