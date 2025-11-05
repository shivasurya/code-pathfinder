package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/diagnostic"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Validate intra-procedural taint analysis against LLM ground truth",
	Long: `The diagnose command validates the accuracy of intra-procedural taint analysis
by comparing tool results against LLM-based ground truth analysis.

It extracts functions, runs both tool and LLM analysis, compares results,
and generates diagnostic reports with precision, recall, and failure analysis.`,
	Run: func(cmd *cobra.Command, _ []string) {
		projectInput := cmd.Flag("project").Value.String()
		llmURL := cmd.Flag("llm-url").Value.String()
		modelName := cmd.Flag("model").Value.String()
		outputDir := cmd.Flag("output").Value.String()
		maxFunctions, _ := cmd.Flags().GetInt("max-functions")
		concurrency, _ := cmd.Flags().GetInt("concurrency")

		if projectInput == "" {
			fmt.Println("Error: --project flag is required")
			return
		}

		startTime := time.Now()

		// Step 1: Extract functions
		fmt.Println("===============================================================================")
		fmt.Println("                    DIAGNOSTIC VALIDATION STARTING")
		fmt.Println("===============================================================================")
		fmt.Println()
		fmt.Printf("Project:        %s\n", projectInput)
		fmt.Printf("LLM Endpoint:   %s\n", llmURL)
		fmt.Printf("Model:          %s\n", modelName)
		fmt.Printf("Max Functions:  %d\n", maxFunctions)
		fmt.Printf("Concurrency:    %d\n", concurrency)
		fmt.Println()

		fmt.Println("Step 1/4: Extracting functions from codebase...")
		functions, err := diagnostic.ExtractAllFunctions(projectInput)
		if err != nil {
			fmt.Printf("Error extracting functions: %v\n", err)
			return
		}

		// Limit to maxFunctions if specified
		if maxFunctions > 0 && len(functions) > maxFunctions {
			functions = functions[:maxFunctions]
		}

		fmt.Printf("✓ Extracted %d functions\n", len(functions))
		fmt.Println()

		// Step 2: LLM Analysis
		fmt.Println("Step 2/4: Running LLM analysis (this may take a while)...")
		llmClient := diagnostic.NewLLMClient(llmURL, modelName)
		llmResults, llmErrors := llmClient.AnalyzeBatch(functions, concurrency)
		fmt.Printf("✓ Analyzed %d functions (%d errors)\n", len(llmResults), len(llmErrors))
		fmt.Println()

		// Step 3: Tool Analysis + Comparison
		fmt.Println("Step 3/4: Running tool analysis and comparison...")
		sources := []string{
			"request.GET", "request.POST", "request.FILES",
			"input(", "sys.argv",
		}
		sinks := []string{
			".execute(", "cursor.execute",
			"os.system(", "subprocess",
			"eval(", "exec(",
			"open(",
		}
		sanitizers := []string{
			"escape", "sanitize", "clean",
			"validate", "filter",
		}

		comparisons := []*diagnostic.DualLevelComparison{}
		functionsMap := make(map[string]*diagnostic.FunctionMetadata)

		for _, fn := range functions {
			functionsMap[fn.FQN] = fn

			llmResult, hasLLM := llmResults[fn.FQN]
			if !hasLLM {
				continue // Skip functions with LLM errors
			}

			toolResult, err := diagnostic.AnalyzeSingleFunction(fn, sources, sinks, sanitizers)
			if err != nil {
				continue // Skip functions with tool errors
			}

			comparison := diagnostic.CompareFunctionResults(fn, toolResult, llmResult)
			comparisons = append(comparisons, comparison)
		}

		fmt.Printf("✓ Compared %d functions\n", len(comparisons))
		fmt.Println()

		// Step 4: Generate Reports
		fmt.Println("Step 4/4: Generating reports...")
		metrics := diagnostic.CalculateOverallMetrics(comparisons, startTime)
		metrics.TopFailures = diagnostic.ExtractTopFailures(comparisons, functionsMap, 5)

		// Console report
		err = diagnostic.GenerateConsoleReport(metrics, outputDir)
		if err != nil {
			fmt.Printf("Error generating console report: %v\n", err)
			return
		}

		// JSON report
		if outputDir != "" {
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				fmt.Printf("Error creating output directory: %v\n", err)
				return
			}

			jsonPath := filepath.Join(outputDir, "diagnostic_report.json")
			err = diagnostic.GenerateJSONReport(metrics, comparisons, jsonPath)
			if err != nil {
				fmt.Printf("Error generating JSON report: %v\n", err)
				return
			}

			fmt.Printf("✓ JSON report saved to: %s\n", jsonPath)
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
	diagnoseCmd.Flags().StringP("project", "p", "", "Project directory to analyze (required)")
	diagnoseCmd.Flags().String("llm-url", "http://localhost:11434/api/generate", "LLM endpoint URL")
	diagnoseCmd.Flags().String("model", "qwen2.5-coder:3b", "LLM model name")
	diagnoseCmd.Flags().StringP("output", "o", "./diagnostic_output", "Output directory for reports")
	diagnoseCmd.Flags().IntP("max-functions", "m", 50, "Maximum functions to analyze")
	diagnoseCmd.Flags().IntP("concurrency", "c", 3, "LLM request concurrency")
	diagnoseCmd.MarkFlagRequired("project") //nolint:all
}
