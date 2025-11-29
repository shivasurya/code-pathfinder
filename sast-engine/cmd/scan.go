package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan code for security vulnerabilities using Python DSL rules",
	Long: `Scan codebase using Python DSL security rules.

Examples:
  # Scan with a single rules file
  pathfinder scan --rules rules/owasp_top10.py --project /path/to/project

  # Scan with a directory of rules
  pathfinder scan --rules rules/ --project /path/to/project

  # Scan with custom rules
  pathfinder scan --rules my_rules.py --project .`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rulesPath, _ := cmd.Flags().GetString("rules")
		projectPath, _ := cmd.Flags().GetString("project")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		failOnStr, _ := cmd.Flags().GetString("fail-on")

		// Setup logger with appropriate verbosity
		verbosity := output.VerbosityDefault
		if debug {
			verbosity = output.VerbosityDebug
		} else if verbose {
			verbosity = output.VerbosityVerbose
		}
		logger := output.NewLogger(verbosity)

		// Parse and validate --fail-on severities
		failOn := output.ParseFailOn(failOnStr)
		if len(failOn) > 0 {
			if err := output.ValidateSeverities(failOn); err != nil {
				return err
			}
		}

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
		logger.Progress("Building code graph from %s...", projectPath)
		codeGraph := graph.Initialize(projectPath)
		if len(codeGraph.Nodes) == 0 {
			return fmt.Errorf("no source files found in project")
		}
		logger.Statistic("Code graph built: %d nodes", len(codeGraph.Nodes))

		// Step 2: Build module registry
		logger.Progress("Building module registry...")
		moduleRegistry, err := registry.BuildModuleRegistry(projectPath)
		if err != nil {
			logger.Warning("failed to build module registry: %v", err)
			// Create empty registry as fallback
			moduleRegistry = core.NewModuleRegistry()
		}

		// Step 3: Build callgraph
		logger.Progress("Building callgraph...")
		cg, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
		if err != nil {
			return fmt.Errorf("failed to build callgraph: %w", err)
		}
		logger.Statistic("Callgraph built: %d functions, %d call sites",
			len(cg.Functions), countTotalCallSites(cg))

		// Step 4: Load Python DSL rules
		logger.Progress("Loading rules from %s...", rulesPath)
		loader := dsl.NewRuleLoader(rulesPath)
		rules, err := loader.LoadRules()
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}
		logger.Statistic("Loaded %d rules", len(rules))

		// Step 5: Execute rules against callgraph
		logger.Progress("Running security scan...")

		// Create enricher for adding context to detections
		enricher := output.NewEnricher(cg, &output.OutputOptions{
			ProjectRoot:  projectPath,
			ContextLines: 3,
			Verbosity:    verbosity,
		})

		// Execute all rules and collect enriched detections
		var allEnriched []*dsl.EnrichedDetection
		var scanErrors bool
		for _, rule := range rules {
			detections, err := loader.ExecuteRule(&rule, cg)
			if err != nil {
				logger.Warning("Error executing rule %s: %v", rule.Rule.ID, err)
				scanErrors = true
				continue
			}

			if len(detections) > 0 {
				enriched, _ := enricher.EnrichAll(detections, rule)
				allEnriched = append(allEnriched, enriched...)
			}
		}

		// Step 6: Format and display results
		summary := output.BuildSummary(allEnriched, len(rules))
		formatter := output.NewTextFormatter(&output.OutputOptions{
			Verbosity: verbosity,
		}, logger)

		if err := formatter.Format(allEnriched, summary); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}

		// Determine exit code based on findings and --fail-on flag
		exitCode := output.DetermineExitCode(allEnriched, failOn, scanErrors)
		if exitCode != output.ExitCodeSuccess {
			os.Exit(int(exitCode))
		}

		return nil
	},
}

func countTotalCallSites(cg *core.CallGraph) int {
	total := 0
	for _, sites := range cg.CallSites {
		total += len(sites)
	}
	return total
}

// printDetections outputs detections in simple format (used by query command).
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
	scanCmd.Flags().StringP("rules", "r", "", "Path to Python DSL rules file or directory (required)")
	scanCmd.Flags().StringP("project", "p", "", "Path to project directory to scan (required)")
	scanCmd.Flags().BoolP("verbose", "v", false, "Show progress and statistics")
	scanCmd.Flags().Bool("debug", false, "Show debug diagnostics with timestamps")
	scanCmd.Flags().String("fail-on", "", "Fail with exit code 1 if findings match severities (e.g., critical,high)")
	scanCmd.MarkFlagRequired("rules")
	scanCmd.MarkFlagRequired("project")
}
