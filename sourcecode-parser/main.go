package main

import (
	"context"
	"fmt"
	"os"

	ql "github.com/shivasurya/code-pathfinder/sourcecode-parser/internal/ql/go"
	sitter "github.com/smacker/go-tree-sitter"
)

func main() {
	parser := sitter.NewParser()
	parser.SetLanguage(ql.GetLanguage())

	content, err := os.ReadFile("/Users/shiva/src/shivasurya/test/test.ql")
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		os.Exit(1)
	}

	rootNode := tree.RootNode()
	fmt.Println("Root node type:", rootNode.Type())
	fmt.Println("Root node start byte:", rootNode.StartByte())
	fmt.Println("Root node end byte:", rootNode.EndByte())
	fmt.Println("Root node start point:", rootNode.StartPoint())
	fmt.Println("Root node end point:", rootNode.EndPoint())
	fmt.Println("Root node child count:", rootNode.ChildCount())
	fmt.Println(rootNode.String())

	// if err := cmd.Execute(); err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
}
