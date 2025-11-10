package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/dsl"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan code for security vulnerabilities using Python DSL rules",
	Long: `Scan codebase using Python DSL security rules.

Examples:
  # Scan with OWASP rules
  pathfinder scan --rules rules/owasp_top10.py --project /path/to/project

  # Scan with custom rules
  pathfinder scan --rules my_rules.py --project .`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rulesPath, _ := cmd.Flags().GetString("rules")
		projectPath, _ := cmd.Flags().GetString("project")

		if rulesPath == "" {
			return fmt.Errorf("--rules flag is required")
		}

		if projectPath == "" {
			return fmt.Errorf("--project flag is required")
		}

		// Convert project path to absolute path to ensure consistency
		absProjectPath, err := filepath.Abs(projectPath)
		if err != nil {
			return fmt.Errorf("failed to resolve project path: %w", err)
		}
		projectPath = absProjectPath

		// Step 1: Build code graph (AST)
		log.Printf("Building code graph from %s...\n", projectPath)
		codeGraph := graph.Initialize(projectPath)
		if len(codeGraph.Nodes) == 0 {
			return fmt.Errorf("no source files found in project")
		}
		log.Printf("Code graph built: %d nodes\n", len(codeGraph.Nodes))

		// Step 2: Build module registry
		log.Printf("Building module registry...\n")
		registry, err := callgraph.BuildModuleRegistry(projectPath)
		if err != nil {
			log.Printf("Warning: failed to build module registry: %v\n", err)
			// Create empty registry as fallback
			registry = callgraph.NewModuleRegistry()
		}

		// Step 3: Build callgraph
		log.Printf("Building callgraph...\n")
		cg, err := callgraph.BuildCallGraph(codeGraph, registry, projectPath)
		if err != nil {
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		log.Printf("Callgraph built: %d functions, %d call sites\n",
			len(cg.Functions), countTotalCallSites(cg))

		// Step 4: Load Python DSL rules
		log.Printf("Loading rules from %s...\n", rulesPath)
		loader := dsl.NewRuleLoader(rulesPath)
		rules, err := loader.LoadRules()
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}
		log.Printf("Loaded %d rules\n", len(rules))

		// Step 5: Execute rules against callgraph
		log.Printf("\n=== Running Security Scan ===\n")
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

		// Step 6: Print summary
		log.Printf("\n=== Scan Complete ===\n")
		log.Printf("Total vulnerabilities found: %d\n", totalDetections)

		if totalDetections > 0 {
			os.Exit(1) // Exit with error code if vulnerabilities found
		}

		return nil
	},
}

func countTotalCallSites(cg *callgraph.CallGraph) int {
	total := 0
	for _, sites := range cg.CallSites {
		total += len(sites)
	}
	return total
}

func printDetections(rule dsl.RuleIR, detections []dsl.DataflowDetection) {
	fmt.Printf("\n[%s] %s (%s)\n", rule.Rule.Severity, rule.Rule.ID, rule.Rule.Name)
	fmt.Printf("  CWE: %s | OWASP: %s\n", rule.Rule.CWE, rule.Rule.OWASP)
	fmt.Printf("  %s\n", rule.Rule.Description)

	for _, detection := range detections {
		fmt.Printf("\n  → %s:%d\n", detection.FunctionFQN, detection.SinkLine)
		if detection.SourceLine > 0 {
			fmt.Printf("    Source: line %d\n", detection.SourceLine)
		}
		if detection.SinkCall != "" {
			fmt.Printf("    Sink: %s (line %d)\n", detection.SinkCall, detection.SinkLine)
		}
		if detection.TaintedVar != "" {
			fmt.Printf("    Tainted variable: %s\n", detection.TaintedVar)
		}
		fmt.Printf("    Confidence: %.0f%%\n", detection.Confidence*100)
		fmt.Printf("    Scope: %s\n", detection.Scope)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file (required)")
	scanCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	scanCmd.MarkFlagRequired("rules")
	scanCmd.MarkFlagRequired("project")
}
