package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	sarif "github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// SARIFFormatter formats enriched detections as SARIF 2.1.0.
type SARIFFormatter struct {
	writer  io.Writer
	options *OutputOptions
}

// NewSARIFFormatter creates a SARIF formatter.
func NewSARIFFormatter(opts *OutputOptions) *SARIFFormatter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &SARIFFormatter{
		writer:  os.Stdout,
		options: opts,
	}
}

// NewSARIFFormatterWithWriter creates a formatter with custom writer (for testing).
func NewSARIFFormatterWithWriter(w io.Writer, opts *OutputOptions) *SARIFFormatter {
	sf := NewSARIFFormatter(opts)
	sf.writer = w
	return sf
}

// Format outputs all detections as SARIF.
func (f *SARIFFormatter) Format(detections []*dsl.EnrichedDetection, scanInfo ScanInfo) error {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return err
	}

	run := sarif.NewRunWithInformationURI("Code Pathfinder", "https://github.com/shivasurya/code-pathfinder")

	// Build rules from unique rule IDs
	f.buildRules(detections, run)

	// Build results — skip detections with no resolvable file path since
	// GitHub Code Scanning requires every result to have at least one location.
	for _, det := range detections {
		if det.Location.RelPath == "" && det.Location.FilePath == "" {
			continue
		}
		f.buildResult(det, run)
	}

	report.AddRun(run)

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func (f *SARIFFormatter) buildRules(detections []*dsl.EnrichedDetection, run *sarif.Run) map[string]bool {
	seen := make(map[string]bool)

	for _, det := range detections {
		if seen[det.Rule.ID] {
			continue
		}
		seen[det.Rule.ID] = true

		// Build full description with CWE and OWASP
		fullDesc := det.Rule.Description
		if len(det.Rule.CWE) > 0 || len(det.Rule.OWASP) > 0 {
			parts := []string{}
			if len(det.Rule.CWE) > 0 {
				parts = append(parts, strings.Join(det.Rule.CWE, ", "))
			}
			if len(det.Rule.OWASP) > 0 {
				parts = append(parts, strings.Join(det.Rule.OWASP, ", "))
			}
			fullDesc += " (" + strings.Join(parts, ", ") + ")"
		}

		sarifRule := run.AddRule(det.Rule.ID).
			WithDescription(fullDesc).
			WithName(det.Rule.Name).
			WithHelpURI("https://github.com/shivasurya/code-pathfinder")

		// Map severity to SARIF level
		level := f.severityToLevelString(det.Rule.Severity)
		sarifRule.WithDefaultConfiguration(sarif.NewReportingConfiguration().WithLevel(level))

		// Add properties for GitHub
		sarifRule.WithProperties(f.buildRuleProperties(det.Rule))
	}

	return seen
}

func (f *SARIFFormatter) buildHelpMarkdown(rule dsl.RuleMetadata) string {
	var markdown strings.Builder
	markdown.WriteString("## " + rule.Name + "\n\n")
	if rule.Description != "" {
		markdown.WriteString(rule.Description + "\n\n")
	}

	if len(rule.CWE) > 0 {
		markdown.WriteString("### References\n\n")
		for _, cwe := range rule.CWE {
			cweNum := extractCWENumber(cwe)
			markdown.WriteString("- [" + cwe + "](https://cwe.mitre.org/data/definitions/" + cweNum + ".html)\n")
		}
	}

	return markdown.String()
}

func extractCWENumber(cwe string) string {
	// CWE-78 -> 78
	if len(cwe) > 4 && cwe[:4] == "CWE-" {
		return cwe[4:]
	}
	return cwe
}

func (f *SARIFFormatter) severityToLevelString(severity string) string {
	switch severity {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	case "low", "info":
		return "note"
	default:
		return "warning"
	}
}

func (f *SARIFFormatter) buildRuleProperties(rule dsl.RuleMetadata) map[string]any {
	props := make(map[string]any)

	// Tags for filtering
	props["tags"] = []string{"security"}

	// Security severity for GitHub
	props["security-severity"] = f.severityToScore(rule.Severity)

	// Precision indicator
	props["precision"] = "high"

	return props
}

func (f *SARIFFormatter) severityToScore(severity string) string {
	switch severity {
	case "critical":
		return "9.0"
	case "high":
		return "7.0"
	case "medium":
		return "5.0"
	case "low":
		return "3.0"
	default:
		return "5.0"
	}
}

func (f *SARIFFormatter) buildResult(det *dsl.EnrichedDetection, run *sarif.Run) {
	message := det.Rule.Description
	if det.Detection.SinkCall != "" {
		message += fmt.Sprintf(" (sink: %s, confidence: %.0f%%)", det.Detection.SinkCall, det.Detection.Confidence*100)
	}

	result := run.CreateResultForRule(det.Rule.ID).
		WithMessage(sarif.NewTextMessage(message))

	// Primary location
	f.addLocation(det, result)

	// Code flows for taint detections
	if det.DetectionType == dsl.DetectionTypeTaintLocal || det.DetectionType == dsl.DetectionTypeTaintGlobal {
		f.addCodeFlow(det, result)
	}
}

func (f *SARIFFormatter) addLocation(det *dsl.EnrichedDetection, result *sarif.Result) {
	filePath := det.Location.RelPath
	if filePath == "" {
		filePath = det.Location.FilePath
	}

	// Skip adding location if file path is empty — SARIF results with empty
	// artifact URIs are rejected by GitHub Code Scanning.
	if filePath == "" {
		return
	}

	region := sarif.NewRegion().
		WithStartLine(det.Location.Line)

	if det.Location.Column > 0 {
		region.WithStartColumn(det.Location.Column)
	}

	location := sarif.NewLocation().
		WithPhysicalLocation(
			sarif.NewPhysicalLocation().
				WithArtifactLocation(
					sarif.NewArtifactLocation().WithUri(filePath),
				).
				WithRegion(region),
		)

	result.AddLocation(location)
}

func (f *SARIFFormatter) addCodeFlow(det *dsl.EnrichedDetection, result *sarif.Result) {
	if det.Detection.SourceLine == 0 || det.Detection.SinkLine == 0 {
		return
	}

	filePath := det.Location.RelPath
	if filePath == "" {
		filePath = det.Location.FilePath
	}

	// Skip code flow if file path is empty — empty artifact URIs are invalid SARIF.
	if filePath == "" {
		return
	}

	// Create thread flow locations
	sourceMsg := "Taint source"
	if det.Detection.TaintedVar != "" {
		sourceMsg += ": " + det.Detection.TaintedVar
	}

	sinkMsg := "Taint sink"
	if det.Detection.SinkCall != "" {
		sinkMsg += ": " + det.Detection.SinkCall
	}

	sourceLocation := sarif.NewLocation().
		WithPhysicalLocation(
			sarif.NewPhysicalLocation().
				WithArtifactLocation(sarif.NewArtifactLocation().WithUri(filePath)).
				WithRegion(sarif.NewRegion().WithStartLine(det.Detection.SourceLine)),
		).
		WithMessage(sarif.NewTextMessage(sourceMsg))

	sinkLocation := sarif.NewLocation().
		WithPhysicalLocation(
			sarif.NewPhysicalLocation().
				WithArtifactLocation(sarif.NewArtifactLocation().WithUri(filePath)).
				WithRegion(sarif.NewRegion().WithStartLine(det.Detection.SinkLine)),
		).
		WithMessage(sarif.NewTextMessage(sinkMsg))

	threadFlow := sarif.NewThreadFlow().
		WithLocations([]*sarif.ThreadFlowLocation{
			sarif.NewThreadFlowLocation().WithLocation(sourceLocation),
			sarif.NewThreadFlowLocation().WithLocation(sinkLocation),
		})

	flowMsg := fmt.Sprintf("Taint flow from line %d to line %d", det.Detection.SourceLine, det.Detection.SinkLine)
	codeFlow := sarif.NewCodeFlow().
		WithThreadFlows([]*sarif.ThreadFlow{threadFlow}).
		WithMessage(sarif.NewTextMessage(flowMsg))

	result.WithCodeFlows([]*sarif.CodeFlow{codeFlow})

	// Also add as related location for visibility
	relatedLocation := sarif.NewLocation().
		WithPhysicalLocation(
			sarif.NewPhysicalLocation().
				WithArtifactLocation(sarif.NewArtifactLocation().WithUri(filePath)).
				WithRegion(sarif.NewRegion().WithStartLine(det.Detection.SourceLine)),
		).
		WithMessage(sarif.NewTextMessage(sourceMsg))

	result.WithRelatedLocations([]*sarif.Location{relatedLocation})
}
