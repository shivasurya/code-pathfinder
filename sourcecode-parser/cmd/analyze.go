package cmd

import (
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze source code for security vulnerabilities using call graph",
	Run: func(cmd *cobra.Command, _ []string) {
		projectInput := cmd.Flag("project").Value.String()

		if projectInput == "" {
			fmt.Println("Error: --project flag is required")
			return
		}

		fmt.Println("Building code graph...")
		codeGraph := graph.Initialize(projectInput)

		fmt.Println("Building call graph and analyzing security patterns...")
		cg, registry, patternRegistry, err := callgraph.InitializeCallGraph(codeGraph, projectInput)
		if err != nil {
			fmt.Println("Error building call graph:", err)
			return
		}

		fmt.Printf("Call graph built successfully: %d functions indexed\n", len(cg.Functions))
		fmt.Printf("Module registry: %d modules\n", len(registry.Modules))

		// Run security analysis
		matches := callgraph.AnalyzePatterns(cg, patternRegistry)

		if len(matches) == 0 {
			fmt.Println("\n✓ No security issues found!")
			return
		}

		fmt.Printf("\n⚠ Found %d potential security issues:\n\n", len(matches))
		for i, match := range matches {
			fmt.Printf("%d. [%s] %s\n", i+1, match.Severity, match.PatternName)
			fmt.Printf("   Description: %s\n", match.Description)
			fmt.Printf("   CWE: %s, OWASP: %s\n\n", match.CWE, match.OWASP)

			// Display source information
			if match.SourceFQN != "" {
				if match.SourceCall != "" {
					fmt.Printf("   Source: %s() calls %s()\n", match.SourceFQN, match.SourceCall)
				} else {
					fmt.Printf("   Source: %s\n", match.SourceFQN)
				}
				if match.SourceFile != "" {
					fmt.Printf("           at %s:%d\n", match.SourceFile, match.SourceLine)
					if match.SourceCode != "" {
						printCodeSnippet(match.SourceCode, int(match.SourceLine))
					}
				}
				fmt.Println()
			}

			// Display sink information
			if match.SinkFQN != "" {
				if match.SinkCall != "" {
					fmt.Printf("   Sink:   %s() calls %s()\n", match.SinkFQN, match.SinkCall)
				} else {
					fmt.Printf("   Sink:   %s\n", match.SinkFQN)
				}
				if match.SinkFile != "" {
					fmt.Printf("           at %s:%d\n", match.SinkFile, match.SinkLine)
					if match.SinkCode != "" {
						printCodeSnippet(match.SinkCode, int(match.SinkLine))
					}
				}
				fmt.Println()
			}

			// Display data flow path
			if len(match.DataFlowPath) > 0 {
				fmt.Printf("   Data flow path (%d steps):\n", len(match.DataFlowPath))
				for j, step := range match.DataFlowPath {
					if j == 0 {
						fmt.Printf("      %s (source)\n", step)
					} else if j == len(match.DataFlowPath)-1 {
						fmt.Printf("      └─> %s (sink)\n", step)
					} else {
						fmt.Printf("      └─> %s\n", step)
					}
				}
				fmt.Println()
			}
		}
	},
}

func printCodeSnippet(code string, startLine int) {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line != "" {
			fmt.Printf("           %4d | %s\n", startLine+i, line)
		}
	}
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringP("project", "p", "", "Project directory to analyze (required)")
	analyzeCmd.MarkFlagRequired("project") //nolint:all
}
