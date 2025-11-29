package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/diagnostic"
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
		provider := cmd.Flag("provider").Value.String()
		apiKey := cmd.Flag("api-key").Value.String()
		outputDir := cmd.Flag("output").Value.String()
		maxFunctions, _ := cmd.Flags().GetInt("max-functions")
		concurrency, _ := cmd.Flags().GetInt("concurrency")

		if projectInput == "" {
			fmt.Println("Error: --project flag is required")
			return
		}

		startTime := time.Now()

		// Create LLM client based on provider
		var llmClient *diagnostic.LLMClient
		if provider == "openai" {
			if apiKey == "" {
				fmt.Println("Error: --api-key is required for OpenAI-compatible providers")
				return
			}
			llmClient = diagnostic.NewOpenAIClient(llmURL, modelName, apiKey)
		} else {
			llmClient = diagnostic.NewLLMClient(llmURL, modelName)
		}

		// Step 1: Extract functions
		fmt.Println("===============================================================================")
		fmt.Println("                    DIAGNOSTIC VALIDATION STARTING")
		fmt.Println("===============================================================================")
		fmt.Println()
		fmt.Printf("Project:        %s\n", projectInput)
		fmt.Printf("LLM Endpoint:   %s\n", llmURL)
		fmt.Printf("Model:          %s\n", modelName)
		fmt.Printf("Provider:       %s\n", provider)
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
		llmResults, llmErrors := llmClient.AnalyzeBatch(functions, concurrency)
		fmt.Printf("✓ Analyzed %d functions (%d errors)\n", len(llmResults), len(llmErrors))

		// Print errors (always show if there are any)
		if len(llmErrors) > 0 {
			fmt.Println("\n⚠️  LLM Analysis Errors:")
			count := 0
			for fqn, err := range llmErrors {
				if count >= 5 {
					fmt.Printf("  ... and %d more errors\n", len(llmErrors)-5)
					break
				}
				fmt.Printf("  ❌ %s:\n", fqn)
				fmt.Printf("     %v\n", err)
				count++
			}
			fmt.Printf("\n💡 Tip: Failed responses saved to %s/llm_errors.txt\n", outputDir)
		}
		fmt.Println()

		// Step 3: Tool Analysis + Comparison
		fmt.Println("Step 3/4: Running tool analysis and comparison...")

		comparisons := []*diagnostic.DualLevelComparison{}
		functionsMap := make(map[string]*diagnostic.FunctionMetadata)

		for _, fn := range functions {
			functionsMap[fn.FQN] = fn

			llmResult, hasLLM := llmResults[fn.FQN]
			if !hasLLM {
				continue // Skip functions with LLM errors
			}

			// Extract unique source/sink/sanitizer patterns from LLM-discovered patterns
			sourcePatterns := make(map[string]bool)
			sinkPatterns := make(map[string]bool)
			sanitizerPatterns := make(map[string]bool)

			for _, src := range llmResult.DiscoveredPatterns.Sources {
				sourcePatterns[src.Pattern] = true
			}
			for _, snk := range llmResult.DiscoveredPatterns.Sinks {
				sinkPatterns[snk.Pattern] = true
			}
			for _, san := range llmResult.DiscoveredPatterns.Sanitizers {
				sanitizerPatterns[san.Pattern] = true
			}

			// Convert to slices and clean patterns
			// Strip () from patterns since tool matching doesn't expect them
			sources := []string{}
			for pattern := range sourcePatterns {
				cleanPattern := strings.TrimSuffix(pattern, "()")
				sources = append(sources, cleanPattern)
			}
			sinks := []string{}
			for pattern := range sinkPatterns {
				cleanPattern := strings.TrimSuffix(pattern, "()")
				sinks = append(sinks, cleanPattern)
			}
			sanitizers := []string{}
			for pattern := range sanitizerPatterns {
				cleanPattern := strings.TrimSuffix(pattern, "()")
				sanitizers = append(sanitizers, cleanPattern)
			}

			if verboseFlag {
				fmt.Printf("  %s: LLM found %d sources, %d sinks, %d sanitizers\n",
					fn.FQN, len(sources), len(sinks), len(sanitizers))
				if len(sources) > 0 {
					fmt.Printf("    Sources: %v\n", sources)
				}
				if len(sinks) > 0 {
					fmt.Printf("    Sinks: %v\n", sinks)
				}
			}

			// If no patterns discovered, use empty lists (tool will find nothing, matching LLM)
			if len(sources) == 0 && len(sinks) == 0 {
				// No patterns = no flows expected
				toolResult := &diagnostic.FunctionTaintResult{
					FunctionFQN:  fn.FQN,
					HasTaintFlow: false,
					TaintFlows:   []diagnostic.ToolTaintFlow{},
				}
				comparison := diagnostic.CompareFunctionResults(fn, toolResult, llmResult)
				comparisons = append(comparisons, comparison)
				continue
			}

			// Run tool with LLM-discovered patterns
			toolResult, err := diagnostic.AnalyzeSingleFunction(fn, sources, sinks, sanitizers)
			if err != nil {
				if verboseFlag {
					fmt.Printf("  Tool error for %s: %v\n", fn.FQN, err)
				}
				continue
			}

			if verboseFlag && toolResult != nil {
				fmt.Printf("    Tool found %d flows (HasTaintFlow=%v)\n",
					len(toolResult.TaintFlows), toolResult.HasTaintFlow)
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
	diagnoseCmd.Flags().String("llm-url", "http://localhost:11434", "LLM endpoint base URL")
	diagnoseCmd.Flags().String("model", "qwen2.5-coder:3b", "LLM model name")
	diagnoseCmd.Flags().String("provider", "ollama", "LLM provider: ollama, openai (for xAI Grok, vLLM, etc.)")
	diagnoseCmd.Flags().String("api-key", "", "API key for OpenAI-compatible providers (e.g., xAI Grok)")
	diagnoseCmd.Flags().StringP("output", "o", "./diagnostic_output", "Output directory for reports")
	diagnoseCmd.Flags().IntP("max-functions", "m", 50, "Maximum functions to analyze")
	diagnoseCmd.Flags().IntP("concurrency", "c", 3, "LLM request concurrency")
	diagnoseCmd.MarkFlagRequired("project") //nolint:all
}
