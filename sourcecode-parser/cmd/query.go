package cmd

import (
	"fmt"
	"log"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/dsl"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query code using Python DSL rules",
	Long: `Query codebase using Python DSL security rules.

Similar to scan but designed for ad-hoc queries and exploration.

Examples:
  # Query with a single rule
  pathfinder query --rules my_rule.py --project /path/to/project

  # Query specific files
  pathfinder query --rules rule.py --project /path/to/file.py`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rulesPath, _ := cmd.Flags().GetString("rules")
		projectPath, _ := cmd.Flags().GetString("project")

		if rulesPath == "" {
			return fmt.Errorf("--rules flag is required")
		}

		if projectPath == "" {
			return fmt.Errorf("--project flag is required")
		}

		// Build code graph (AST)
		log.Printf("Building code graph from %s...\n", projectPath)
		codeGraph := graph.Initialize(projectPath)
		if len(codeGraph.Nodes) == 0 {
			return fmt.Errorf("no source files found in project")
		}
		log.Printf("Code graph built: %d nodes\n", len(codeGraph.Nodes))

		// Build module registry
		log.Printf("Building module registry...\n")
		registry, err := callgraph.BuildModuleRegistry(projectPath)
		if err != nil {
			log.Printf("Warning: failed to build module registry: %v\n", err)
			registry = callgraph.NewModuleRegistry()
		}

		// Build callgraph
		log.Printf("Building callgraph...\n")
		cg, err := callgraph.BuildCallGraph(codeGraph, registry, projectPath)
		if err != nil {
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		log.Printf("Callgraph built: %d functions, %d call sites\n",
			len(cg.Functions), countTotalCallSites(cg))

		// Load Python DSL rules
		log.Printf("Loading rules from %s...\n", rulesPath)
		loader := dsl.NewRuleLoader(rulesPath)
		rules, err := loader.LoadRules()
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}
		log.Printf("Loaded %d rules\n", len(rules))

		// Execute rules against callgraph
		log.Printf("\n=== Query Results ===\n")
		totalDetections := 0
		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				log.Printf("Error executing rule %s: %v\n", rule.Rule.ID, err)
				continue
			}

			if len(detections) > 0 {
				printDetections(rule, detections)
				totalDetections += len(detections)
			}
		}

		log.Printf("\n=== Query Complete ===\n")
		log.Printf("Total matches: %d\n", totalDetections)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file (required)")
	queryCmd.Flags().StringP("project", "p", "", "Path to project directory to query (required)")
	queryCmd.MarkFlagRequired("rules")
	queryCmd.MarkFlagRequired("project")
}
