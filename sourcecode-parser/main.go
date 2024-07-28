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
const (
	Version   = "0.0.22"
	GitCommit = "HEAD"
)

func main() {
	// accept command line param optional path to source code
	output := flag.String("output", "", "Supported output format: json")
	outputFile := flag.String("output-file", "", "Output file path")
	project := flag.String("project", "", "Project to analyze")
	query := flag.String("query", "", "Query to execute")
	stdin := flag.Bool("stdin", false, "Read query from stdin")
	versionFlag := flag.Bool("version", false, "Print the version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
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

func processQuery(input string, graph *CodeGraph, output string) (string, error) {
	fmt.Println("Executing query: " + input)
	parsedQuery := parser.ParseQuery(input)
	entities := QueryEntities(graph, parsedQuery)
	if output == "json" {
		// convert struct to query_results
		queryResults, err := json.Marshal(entities)
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
				return "", fmt.Errorf("error processing query: %w", err)
			}
		}
	} else {
		// read from command line
		result, err := processQuery(query, graph, output)
		if err != nil {
			return "", fmt.Errorf("error processing query: %w", err)
		}
		return result, nil
	}
}

// func parseQueryWithExpr(inputQuery string) {
//
//	// string replace "md." with ""
//	expression = strings.Replace(expression, "md.", "", -1)
//	fmt.Println(expression)
//	env := map[string]interface{}{
//		"getName": func() string {
//			return "onCreate"
//		},
//		"getVisibility": func() string {
//			return "public"
//		},
//		"getReturnType": func() string {
//			return "voids"
//		},
//	}
//	program, err := expr.Compile(expression, expr.Env(env))
//	if err != nil {
//		fmt.Println(err)
//	}
//	output, err := expr.Run(program, env)
//	if err != nil {
//		fmt.Println(err)
//	}
//	fmt.Println(output)
//}
