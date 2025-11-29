package main

import (
	"fmt"
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
