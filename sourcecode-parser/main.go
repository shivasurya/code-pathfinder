package main

import (
	"bufio"
	"fmt"
	"os"
	"queryparser"
	"strings"
)

func main() {
	// accept command line param optional path to source code
	sourceDirectory := "."
	graph := NewCodeGraph()
	if len(os.Args) > 1 {
		sourceDirectory = os.Args[1]
		graph = Initialize(sourceDirectory)
	}
	// loop to accept queries
	for {
		var input string
		fmt.Print("Path-Finder Query Console: \n>")
		in := bufio.NewReader(os.Stdin)

		input, _ = in.ReadString('\n')
		// if input starts with :quit string
		if strings.HasPrefix(input, ":quit") {
			return
		}
		fmt.Print("Executing query: " + input)
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
			fmt.Println("------Query Results------")
			for _, entity := range entities {
				fmt.Println(entity.CodeSnippet)
			}
			fmt.Println("-------------------")
		}
	}
}
