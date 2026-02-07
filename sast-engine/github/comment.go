package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// summaryMarker is an invisible HTML comment embedded in every summary comment.
// Used to find and update existing comments instead of creating duplicates.
const summaryMarker = "<!-- code-pathfinder-summary -->"

// ScanMetrics captures aggregate scan statistics for the summary comment.
type ScanMetrics struct {
	FilesScanned  int
	RulesExecuted int
}

// CommentManager handles creating and updating PR summary comments.
type CommentManager struct {
	client   *Client
	prNumber int
}

// NewCommentManager creates a comment manager for the given PR.
func NewCommentManager(client *Client, prNumber int) *CommentManager {
	return &CommentManager{client: client, prNumber: prNumber}
}

// PostOrUpdate posts a new summary comment or updates the existing one.
// It searches for a comment containing the marker to avoid duplicates.
func (cm *CommentManager) PostOrUpdate(ctx context.Context, markdown string) error {
	body := summaryMarker + "\n" + markdown

	existingID, err := cm.findExisting(ctx)
	if err != nil {
		return fmt.Errorf("find existing comment: %w", err)
	}

	if existingID != 0 {
		_, err = cm.client.UpdateComment(ctx, existingID, body)
		if err != nil {
			return fmt.Errorf("update summary comment: %w", err)
		}
		return nil
	}

	_, err = cm.client.CreateComment(ctx, cm.prNumber, body)
	if err != nil {
		return fmt.Errorf("create summary comment: %w", err)
	}
	return nil
}

// findExisting returns the ID of an existing summary comment, or 0 if none.
func (cm *CommentManager) findExisting(ctx context.Context) (int64, error) {
	comments, err := cm.client.ListComments(ctx, cm.prNumber)
	if err != nil {
		return 0, err
	}
	for _, c := range comments {
		if strings.Contains(c.Body, summaryMarker) {
			return c.ID, nil
		}
	}
	return 0, nil
}

// FormatSummaryComment builds the markdown body for a PR summary comment.
func FormatSummaryComment(findings []*dsl.EnrichedDetection, metrics ScanMetrics) string {
	counts := countBySeverity(findings)
	var sb strings.Builder

	sb.WriteString("## Code Pathfinder Security Scan\n\n")

	// Status and severity badges.
	if counts.Critical == 0 && counts.High == 0 && counts.Medium == 0 && counts.Low == 0 {
		sb.WriteString(statusBadge("Pass", "success"))
	} else {
		sb.WriteString(statusBadge("Issues Found", "critical"))
	}
	sb.WriteString(" ")
	sb.WriteString(severityBadge("Critical", counts.Critical))
	sb.WriteString(" ")
	sb.WriteString(severityBadge("High", counts.High))
	sb.WriteString(" ")
	sb.WriteString(severityBadge("Medium", counts.Medium))
	sb.WriteString("\n\n")

	if len(findings) == 0 {
		sb.WriteString("**No security issues detected.**\n\n")
	} else {
		writeFindingsTable(&sb, findings)
		writeDetails(&sb, findings)
		if counts.Critical > 0 {
			sb.WriteString(fmt.Sprintf("> **%d critical issue(s)** require attention.\n\n", counts.Critical))
		}
	}

	// Metrics table.
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|:-------|------:|\n")
	sb.WriteString(fmt.Sprintf("| Files Scanned | %d |\n", metrics.FilesScanned))
	sb.WriteString(fmt.Sprintf("| Rules | %d |\n", metrics.RulesExecuted))

	sb.WriteString("\n---\n")
	sb.WriteString("<sub>Powered by <a href=\"https://codepathfinder.dev\">Code Pathfinder</a></sub>\n")

	return sb.String()
}

// severityCounts holds per-severity finding totals.
type severityCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
}

func countBySeverity(findings []*dsl.EnrichedDetection) severityCounts {
	var c severityCounts
	for _, f := range findings {
		switch strings.ToLower(f.Rule.Severity) {
		case "critical":
			c.Critical++
		case "high":
			c.High++
		case "medium":
			c.Medium++
		case "low":
			c.Low++
		}
	}
	return c
}

func statusBadge(label, color string) string {
	safe := strings.ReplaceAll(label, " ", "_")
	return fmt.Sprintf("![%s](https://img.shields.io/badge/Security-%s-%s?style=flat-square)", label, safe, color)
}

func severityBadge(label string, count int) string {
	color := "lightgrey"
	switch label {
	case "Critical":
		if count > 0 {
			color = "critical"
		} else {
			color = "success"
		}
	case "High":
		if count > 0 {
			color = "orange"
		} else {
			color = "success"
		}
	case "Medium":
		if count > 0 {
			color = "yellow"
		} else {
			color = "success"
		}
	}
	return fmt.Sprintf("![%s](https://img.shields.io/badge/%s-%d-%s?style=flat-square)", label, label, count, color)
}

func severityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "\xf0\x9f\x94\xb4" // red circle
	case "high":
		return "\xf0\x9f\x9f\xa0" // orange circle
	case "medium":
		return "\xf0\x9f\x9f\xa1" // yellow circle
	case "low":
		return "\xf0\x9f\x94\xb5" // blue circle
	default:
		return ""
	}
}

func severityLabel(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return severityEmoji("critical") + " **Critical**"
	case "high":
		return severityEmoji("high") + " High"
	case "medium":
		return severityEmoji("medium") + " Medium"
	case "low":
		return severityEmoji("low") + " Low"
	default:
		return severity
	}
}

func writeFindingsTable(sb *strings.Builder, findings []*dsl.EnrichedDetection) {
	sb.WriteString("### Findings\n\n")
	sb.WriteString("| Severity | File | Line | Issue |\n")
	sb.WriteString("|:---------|:-----|-----:|:------|\n")
	for _, f := range findings {
		sb.WriteString(fmt.Sprintf("| %s | `%s` | %d | %s |\n",
			severityLabel(f.Rule.Severity),
			f.Location.RelPath,
			f.Location.Line,
			f.Rule.Name,
		))
	}
	sb.WriteString("\n")
}

func writeDetails(sb *strings.Builder, findings []*dsl.EnrichedDetection) {
	sb.WriteString("<details>\n<summary>View Details</summary>\n\n")
	for _, f := range findings {
		sb.WriteString(fmt.Sprintf("### %s %s\n", severityEmoji(f.Rule.Severity), f.Rule.Name))
		sb.WriteString(fmt.Sprintf("**%s:%d**", f.Location.RelPath, f.Location.Line))
		if len(f.Rule.CWE) > 0 {
			sb.WriteString(fmt.Sprintf(" \u00b7 %s", strings.Join(f.Rule.CWE, ", ")))
		}
		sb.WriteString("\n\n")
		if f.Rule.Description != "" {
			sb.WriteString(f.Rule.Description)
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("</details>\n\n")
}
