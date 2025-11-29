package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// TextFormatter formats enriched detections as human-readable text.
type TextFormatter struct {
	writer  io.Writer
	options *OutputOptions
	logger  *Logger
}

// NewTextFormatter creates a text formatter.
func NewTextFormatter(opts *OutputOptions, logger *Logger) *TextFormatter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &TextFormatter{
		writer:  os.Stdout,
		options: opts,
		logger:  logger,
	}
}

// NewTextFormatterWithWriter creates a formatter with custom writer (for testing).
func NewTextFormatterWithWriter(w io.Writer, opts *OutputOptions, logger *Logger) *TextFormatter {
	tf := NewTextFormatter(opts, logger)
	tf.writer = w
	return tf
}

// Format outputs all detections as formatted text.
func (f *TextFormatter) Format(detections []*dsl.EnrichedDetection, summary *Summary) error {
	if len(detections) == 0 {
		f.writeNoFindings()
		return nil
	}

	f.writeHeader()
	f.writeResults(detections)
	f.writeSummary(summary)

	if f.options.ShouldShowStatistics() {
		f.writeStatistics(summary)
	}

	return nil
}

func (f *TextFormatter) writeHeader() {
	fmt.Fprintln(f.writer, "Code Pathfinder Security Scan")
	fmt.Fprintln(f.writer)
}

func (f *TextFormatter) writeNoFindings() {
	fmt.Fprintln(f.writer, "Code Pathfinder Security Scan")
	fmt.Fprintln(f.writer)
	fmt.Fprintln(f.writer, "No security issues found.")
}

func (f *TextFormatter) writeResults(detections []*dsl.EnrichedDetection) {
	fmt.Fprintln(f.writer, "Results:")
	fmt.Fprintln(f.writer)

	// Group by severity
	grouped := f.groupBySeverity(detections)

	// Output in severity order: critical, high, medium, low
	severityOrder := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range severityOrder {
		if dets, ok := grouped[sev]; ok && len(dets) > 0 {
			f.writeSeverityGroup(sev, dets)
		}
	}
}

func (f *TextFormatter) groupBySeverity(detections []*dsl.EnrichedDetection) map[string][]*dsl.EnrichedDetection {
	grouped := make(map[string][]*dsl.EnrichedDetection)
	for _, det := range detections {
		sev := det.Rule.Severity
		grouped[sev] = append(grouped[sev], det)
	}
	return grouped
}

func (f *TextFormatter) writeSeverityGroup(severity string, detections []*dsl.EnrichedDetection) {
	// Header
	title := fmt.Sprintf("%s Issues (%d):", strings.Title(severity), len(detections))
	fmt.Fprintln(f.writer, title)
	fmt.Fprintln(f.writer)

	// Critical and high get detailed output
	showDetailed := severity == "critical" || severity == "high"

	for _, det := range detections {
		if showDetailed {
			f.writeDetailedFinding(det)
		} else {
			f.writeAbbreviatedFinding(det)
		}
	}
	fmt.Fprintln(f.writer)
}

func (f *TextFormatter) writeDetailedFinding(det *dsl.EnrichedDetection) {
	// First line: [severity] [badge] rule-id: rule-name
	fmt.Fprintf(f.writer, "  [%s] %s %s: %s\n",
		det.Rule.Severity,
		det.DetectionBadge(),
		det.Rule.ID,
		det.Rule.Name)

	// Metadata line (only if available)
	var metaParts []string
	if len(det.Rule.CWE) > 0 {
		metaParts = append(metaParts, det.Rule.CWE[0])
	}
	if len(det.Rule.OWASP) > 0 {
		metaParts = append(metaParts, det.Rule.OWASP[0])
	}
	if len(metaParts) > 0 {
		fmt.Fprintf(f.writer, "    %s\n", strings.Join(metaParts, " | "))
	}
	fmt.Fprintln(f.writer)

	// Location
	location := f.formatLocation(det.Location)
	fmt.Fprintf(f.writer, "    %s\n", location)

	// Code snippet
	if len(det.Snippet.Lines) > 0 {
		f.writeCodeSnippet(det.Snippet)
	}
	fmt.Fprintln(f.writer)

	// Taint flow (for taint detections)
	if det.DetectionType == dsl.DetectionTypeTaintLocal || det.DetectionType == dsl.DetectionTypeTaintGlobal {
		f.writeTaintFlow(det)
	}

	// Confidence and detection method
	fmt.Fprintf(f.writer, "    Confidence: %s | Detection: %s\n",
		strings.Title(det.ConfidenceLevel()),
		f.formatDetectionMethod(det.DetectionType))
	fmt.Fprintln(f.writer)
}

func (f *TextFormatter) writeAbbreviatedFinding(det *dsl.EnrichedDetection) {
	// Single line: [severity] [badge] rule-id: location
	location := f.formatLocation(det.Location)
	fmt.Fprintf(f.writer, "  [%s] %s %s: %s\n",
		det.Rule.Severity,
		det.DetectionBadge(),
		det.Rule.ID,
		location)
}

func (f *TextFormatter) formatLocation(loc dsl.LocationInfo) string {
	path := loc.RelPath
	if path == "" {
		path = loc.FilePath
	}
	if path == "" {
		path = loc.Function
	}
	if loc.Line > 0 {
		return fmt.Sprintf("%s:%d", path, loc.Line)
	}
	return path
}

func (f *TextFormatter) writeCodeSnippet(snippet dsl.CodeSnippet) {
	// Find max line number width
	maxLineNum := 0
	for _, line := range snippet.Lines {
		if line.Number > maxLineNum {
			maxLineNum = line.Number
		}
	}
	lineWidth := len(fmt.Sprintf("%d", maxLineNum))

	for _, line := range snippet.Lines {
		marker := " "
		if line.IsHighlight {
			marker = ">"
		}
		fmt.Fprintf(f.writer, "      %s %*d | %s\n",
			marker,
			lineWidth,
			line.Number,
			line.Content)
	}
}

func (f *TextFormatter) writeTaintFlow(det *dsl.EnrichedDetection) {
	if det.Detection.TaintedVar == "" {
		return
	}

	fmt.Fprintf(f.writer, "    Flow: %s (line %d) -> %s (line %d)\n",
		det.Detection.TaintedVar,
		det.Detection.SourceLine,
		det.Detection.SinkCall,
		det.Detection.SinkLine)

	fmt.Fprintf(f.writer, "    Tainted variable '%s' reaches dangerous sink without sanitization\n",
		det.Detection.TaintedVar)
}

func (f *TextFormatter) formatDetectionMethod(dt dsl.DetectionType) string {
	switch dt {
	case dsl.DetectionTypePattern:
		return "Pattern matching"
	case dsl.DetectionTypeTaintLocal:
		return "Intra-procedural taint analysis"
	case dsl.DetectionTypeTaintGlobal:
		return "Inter-procedural taint analysis"
	default:
		return "Unknown"
	}
}

func (f *TextFormatter) writeSummary(summary *Summary) {
	fmt.Fprintln(f.writer, "Summary:")
	fmt.Fprintf(f.writer, "  %d findings across %d rules\n",
		summary.TotalFindings, summary.RulesExecuted)

	// Severity breakdown
	var parts []string
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		if count, ok := summary.BySeverity[sev]; ok && count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, sev))
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(f.writer, "  %s\n", strings.Join(parts, " | "))
	}
	fmt.Fprintln(f.writer)
}

func (f *TextFormatter) writeStatistics(summary *Summary) {
	fmt.Fprintln(f.writer, "Detection Methods:")
	for method, count := range summary.ByDetectionType {
		fmt.Fprintf(f.writer, "  %s: %d findings\n", method, count)
	}
	fmt.Fprintln(f.writer)
}

// Summary holds aggregated statistics.
type Summary struct {
	TotalFindings   int
	RulesExecuted   int
	BySeverity      map[string]int
	ByDetectionType map[string]int
	FilesScanned    int
	Duration        string
}

// BuildSummary creates summary from detections.
func BuildSummary(detections []*dsl.EnrichedDetection, rulesExecuted int) *Summary {
	summary := &Summary{
		TotalFindings:   len(detections),
		RulesExecuted:   rulesExecuted,
		BySeverity:      make(map[string]int),
		ByDetectionType: make(map[string]int),
	}

	for _, det := range detections {
		summary.BySeverity[det.Rule.Severity]++
		summary.ByDetectionType[string(det.DetectionType)]++
	}

	return summary
}
