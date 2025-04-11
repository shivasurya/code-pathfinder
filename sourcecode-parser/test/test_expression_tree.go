package main

import (
	"encoding/json"
	"fmt"
	"log"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
)

func main() {
	// Test query with a WHERE clause that matches the grammar
	testQuery := `FROM method AS m WHERE m.name("GetUser") SELECT m.name()`

	// Parse the query
	result, err := parser.ParseQuery(testQuery)
	if err != nil {
		log.Fatalf("Error parsing query: %v", err)
	}

	// Print the parsed query information
	fmt.Println("Select List:")
	for _, item := range result.SelectList {
		fmt.Printf("  Entity: %s, Alias: %s\n", item.Entity, item.Alias)
	}

	fmt.Println("\nConditions:")
	for _, cond := range result.Condition {
		fmt.Printf("  %s\n", cond)
	}

	fmt.Println("\nExpression Tree:")
	if result.ExpressionTree != nil {
		treeJSON, err := json.MarshalIndent(result.ExpressionTree, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling expression tree: %v", err)
		}
		fmt.Println(string(treeJSON))
	} else {
		fmt.Println("  No expression tree available")
	}

	// Test another query with different operators
	testQuery2 := `FROM method AS m WHERE m.complexity() > 10 SELECT m.name()`

	// Parse the second query
	result2, err := parser.ParseQuery(testQuery2)
	if err != nil {
		log.Fatalf("Error parsing query: %v", err)
	}

	fmt.Println("\n\nSecond Query - Expression Tree:")
	if result2.ExpressionTree != nil {
		treeJSON, err := json.MarshalIndent(result2.ExpressionTree, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling expression tree: %v", err)
		}
		fmt.Println(string(treeJSON))
	} else {
		fmt.Println("  No expression tree available")
	}

	// Test a more complex query with AND and OR operators
	testQuery3 := `FROM method AS m WHERE m.complexity() > 10 && m.name("Controller") || m.lines() <= 100 SELECT m.name()`

	// Parse the third query
	result3, err := parser.ParseQuery(testQuery3)
	if err != nil {
		log.Fatalf("Error parsing query: %v", err)
	}

	fmt.Println("\n\nThird Query - Expression Tree:")
	if result3.ExpressionTree != nil {
		treeJSON, err := json.MarshalIndent(result3.ExpressionTree, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling expression tree: %v", err)
		}
		fmt.Println(string(treeJSON))
	} else {
		fmt.Println("  No expression tree available")
	}
}
