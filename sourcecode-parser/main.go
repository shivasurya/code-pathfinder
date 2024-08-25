package main

import (
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
