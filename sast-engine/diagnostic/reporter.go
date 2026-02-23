package diagnostic

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// GenerateConsoleReport prints human-readable report to stdout.
func GenerateConsoleReport(metrics *OverallMetrics, outputDir string) error {
	fmt.Println("===============================================================================")
	fmt.Println("                 INTRA-PROCEDURAL TAINT ANALYSIS DIAGNOSTIC")
	fmt.Println("===============================================================================")
	fmt.Println()

	// Overall stats
	fmt.Printf("Functions Analyzed: %d\n", metrics.TotalFunctions)
	fmt.Printf("Processing Time: %s\n", metrics.TotalProcessingTime)
	fmt.Printf("Speed: %.1f functions/second\n", metrics.FunctionsPerSecond)
	fmt.Println()

	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println("OVERALL METRICS")
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println()

	fmt.Printf("Agreement with LLM: %.1f%% (%d / %d)\n",
		metrics.Agreement*100,
		metrics.TruePositives+metrics.TrueNegatives,
		metrics.TotalFunctions)
	fmt.Printf("Precision:          %.1f%% (%d / %d)\n",
		metrics.Precision*100,
		metrics.TruePositives,
		metrics.TruePositives+metrics.FalsePositives)
	fmt.Printf("Recall:             %.1f%% (%d / %d)\n",
		metrics.Recall*100,
		metrics.TruePositives,
		metrics.TruePositives+metrics.FalseNegatives)
	fmt.Printf("F1 Score:           %.1f%%\n", metrics.F1Score*100)
	fmt.Println()

	fmt.Println("Confusion Matrix:")
	fmt.Printf("  True Positives:   %-6d (Tool detected, LLM confirmed)\n", metrics.TruePositives)
	fmt.Printf("  False Positives:  %-6d (Tool detected, LLM says safe)\n", metrics.FalsePositives)
	fmt.Printf("  False Negatives:  %-6d (Tool missed, LLM found vuln)\n", metrics.FalseNegatives)
	fmt.Printf("  True Negatives:   %-6d (Tool skipped, LLM confirmed safe)\n", metrics.TrueNegatives)
	fmt.Println()

	// Failure breakdown
	if len(metrics.FailuresByCategory) > 0 {
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println("FAILURE BREAKDOWN")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println()

		// Sort categories by count
		type categoryCount struct {
			category string
			count    int
		}
		categories := make([]categoryCount, 0, len(metrics.FailuresByCategory))
		for cat, count := range metrics.FailuresByCategory {
			categories = append(categories, categoryCount{cat, count})
		}
		sort.Slice(categories, func(i, j int) bool {
			return categories[i].count > categories[j].count
		})

		totalFailures := metrics.FalsePositives + metrics.FalseNegatives

		fmt.Printf("Top Failure Categories: %d total\n", totalFailures)
		for i, cc := range categories {
			percentage := 0.0
			if totalFailures > 0 {
				percentage = float64(cc.count) / float64(totalFailures) * 100
			}
			marker := ""
			if i == 0 {
				marker = " <- FIX THIS FIRST"
			}
			fmt.Printf("  %d. %-25s %d cases (%.1f%%)%s\n",
				i+1, cc.category+":", cc.count, percentage, marker)
		}
		fmt.Println()
	}

	// Top failures
	if len(metrics.TopFailures) > 0 {
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println("TOP FAILURE EXAMPLES")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println()

		count := 0
		for _, failure := range metrics.TopFailures {
			if count >= 5 {
				break
			}
			count++

			typeStr := "FALSE NEGATIVE"
			if failure.Type == "FALSE_POSITIVE" {
				typeStr = "FALSE POSITIVE"
			}

			fmt.Printf("%d. %s (%s)\n", count, failure.FunctionFQN, failure.Category)
			fmt.Printf("   File: %s:%d\n", failure.FunctionFile, failure.FunctionLine)
			fmt.Printf("   Type: %s\n", typeStr)
			fmt.Println()
			if failure.Reason != "" {
				fmt.Printf("   Reason: %s\n", wrapText(failure.Reason, 70, "   "))
				fmt.Println()
			}
		}
	}

	// Recommendations
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println("NEXT STEPS")
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println()
	fmt.Println("1. Review top failure examples above")
	fmt.Println("2. Focus on the top failure category to maximize impact")
	fmt.Println("3. Re-run diagnostic after improvements to measure progress")
	fmt.Println()

	// Save location
	if outputDir != "" {
		fmt.Printf("Full report saved to: %s/\n", outputDir)
		fmt.Println()
	}

	fmt.Println("===============================================================================")

	return nil
}

// GenerateJSONReport writes machine-readable JSON report.
func GenerateJSONReport(
	metrics *OverallMetrics,
	comparisons []*DualLevelComparison,
	outputPath string,
) error {
	report := map[string]any{
		"metrics":     metrics,
		"comparisons": comparisons,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	jsonBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(outputPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

// Helper functions

func wrapText(text string, width int, prefix string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	lineLength := 0

	for _, word := range words {
		if lineLength+len(word)+1 > width {
			result.WriteString("\n" + prefix)
			lineLength = 0
		}
		if lineLength > 0 {
			result.WriteString(" ")
			lineLength++
		}
		result.WriteString(word)
		lineLength += len(word)
	}

	return result.String()
}
