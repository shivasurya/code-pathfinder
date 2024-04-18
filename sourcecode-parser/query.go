package main

import (
	"bufio"
	"fmt"
	"os"
	"queryparser"
)

func main() {
	// input := "FIND class WHERE name = 'MyClass' AND visibility = 'public' AND method = 'main'"
	var input string
	fmt.Print("Enter query: \n >")
	in := bufio.NewReader(os.Stdin)

	input, _ = in.ReadString('\n')
	if input == ":quit" {
		return
	}
	fmt.Print(input)
	lex := queryparser.NewLexer(input)
	pars := queryparser.NewParser(lex)
	query := pars.ParseQuery()
	if query == nil {
		fmt.Println("Failed to parse query:")
		for _, err := range pars.Errors() {
			fmt.Println(err)
		}
	} else {
		fmt.Println(query)
	}
}
