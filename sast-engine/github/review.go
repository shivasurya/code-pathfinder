package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// ReviewManager handles posting inline review comments on a PR.
type ReviewManager struct {
	client    *Client
	prNumber  int
	commitSHA string
}

// NewReviewManager creates a review manager for the given PR and commit.
func NewReviewManager(client *Client, prNumber int, commitSHA string) *ReviewManager {
	return &ReviewManager{
		client:    client,
		prNumber:  prNumber,
		commitSHA: commitSHA,
	}
}

// PostInlineComments posts inline review comments for critical and high findings.
// Findings are batched into a single review request (atomic).
// Existing comments with matching markers are updated; new ones are created.
func (rm *ReviewManager) PostInlineComments(ctx context.Context, findings []*dsl.EnrichedDetection) error {
	// Filter to inline-eligible findings.
	eligible := filterEligible(findings)
	if len(eligible) == 0 {
		return nil
	}

	// Fetch existing review comments for marker comparison.
	existing, err := rm.client.ListReviewComments(ctx, rm.prNumber)
	if err != nil {
		return fmt.Errorf("list existing review comments: %w", err)
	}
	existingByMarker := indexByMarker(existing)

	// Separate findings into updates vs new comments.
	newComments := make([]ReviewCommentInput, 0, len(eligible))
	for _, f := range eligible {
		marker := ReviewCommentMarker(f)
		body := FormatInlineComment(f)

		if commentID, ok := existingByMarker[marker]; ok {
			// Update existing review comment in-place (uses pulls/comments endpoint).
			if _, err := rm.client.UpdateReviewComment(ctx, commentID, body); err != nil {
				return fmt.Errorf("update inline comment: %w", err)
			}
			continue
		}

		newComments = append(newComments, ReviewCommentInput{
			Path: f.Location.RelPath,
			Line: f.Location.Line,
			Side: "RIGHT",
			Body: body,
		})
	}

	// Post new comments as a single atomic review.
	if len(newComments) > 0 {
		if err := rm.client.CreateReview(ctx, rm.prNumber, rm.commitSHA, "", newComments); err != nil {
			return fmt.Errorf("create review: %w", err)
		}
	}

	return nil
}

// ShouldPostInline returns true if the severity warrants an inline comment.
// Only critical and high findings get inline comments; medium and low go in the summary only.
func ShouldPostInline(severity string) bool {
	s := strings.ToLower(severity)
	return s == "critical" || s == "high"
}

// ReviewCommentMarker generates a hidden HTML marker for a finding.
// Used to match existing comments for update-in-place.
func ReviewCommentMarker(f *dsl.EnrichedDetection) string {
	return fmt.Sprintf("<!-- cpf-%s-%s-%d -->", f.Rule.ID, f.Location.RelPath, f.Location.Line)
}

// FormatInlineComment builds the markdown body for a single inline comment.
func FormatInlineComment(f *dsl.EnrichedDetection) string {
	var sb strings.Builder

	// Severity + rule name header.
	sb.WriteString(fmt.Sprintf("%s **%s**\n\n", severityEmoji(f.Rule.Severity), f.Rule.Name))

	// Description.
	if f.Rule.Description != "" {
		sb.WriteString(f.Rule.Description)
		sb.WriteString("\n\n")
	}

	// Taint flow path.
	if len(f.TaintPath) >= 2 {
		writeTaintFlow(&sb, f.TaintPath)
	}

	// CWE and OWASP references.
	writeReferences(&sb, f.Rule.CWE, f.Rule.OWASP)

	// Hidden marker for update-in-place.
	// Trim trailing whitespace to avoid excess blank lines.
	body := strings.TrimRight(sb.String(), "\n")
	return body + "\n\n" + ReviewCommentMarker(f) + "\n"
}

// filterEligible returns only critical and high findings with valid locations.
func filterEligible(findings []*dsl.EnrichedDetection) []*dsl.EnrichedDetection {
	result := make([]*dsl.EnrichedDetection, 0, len(findings))
	for _, f := range findings {
		if ShouldPostInline(f.Rule.Severity) && f.Location.RelPath != "" && f.Location.Line > 0 {
			result = append(result, f)
		}
	}
	return result
}

// indexByMarker builds a map from marker string to comment ID for existing comments.
func indexByMarker(comments []*ReviewComment) map[string]int64 {
	m := make(map[string]int64, len(comments))
	for _, c := range comments {
		// Extract marker from comment body.
		if idx := strings.Index(c.Body, "<!-- cpf-"); idx != -1 {
			end := strings.Index(c.Body[idx:], "-->")
			if end != -1 {
				marker := c.Body[idx : idx+end+3]
				m[marker] = c.ID
			}
		}
	}
	return m
}

// writeTaintFlow writes the source→sink flow section.
func writeTaintFlow(sb *strings.Builder, path []dsl.TaintPathNode) {
	var source, sink *dsl.TaintPathNode
	for i := range path {
		if path[i].IsSource {
			source = &path[i]
		}
		if path[i].IsSink {
			sink = &path[i]
		}
	}
	if source == nil || sink == nil {
		return
	}

	sb.WriteString("**Flow:**\n")
	sourceFile := source.Location.RelPath
	if sourceFile == "" {
		sourceFile = source.Location.FilePath
	}
	sb.WriteString(fmt.Sprintf("- Source: `%s:%d`", sourceFile, source.Location.Line))
	if source.Variable != "" {
		sb.WriteString(fmt.Sprintf(" \u2014 `%s`", source.Variable))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("- Sink: `%s:%d`", sink.Location.RelPath, sink.Location.Line))
	if sink.Variable != "" {
		sb.WriteString(fmt.Sprintf(" \u2014 `%s`", sink.Variable))
	}
	sb.WriteString("\n\n")
}

// writeReferences writes CWE and OWASP reference line.
func writeReferences(sb *strings.Builder, cwes, owasps []string) {
	parts := make([]string, 0, 2)
	if len(cwes) > 0 {
		parts = append(parts, strings.Join(cwes, ", "))
	}
	if len(owasps) > 0 {
		parts = append(parts, strings.Join(owasps, ", "))
	}
	if len(parts) > 0 {
		sb.WriteString(strings.Join(parts, " \u00b7 "))
		sb.WriteString("\n")
	}
}
