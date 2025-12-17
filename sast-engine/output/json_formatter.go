package output

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// JSONFormatter formats enriched detections as JSON.
type JSONFormatter struct {
	writer  io.Writer
	options *OutputOptions
}

// NewJSONFormatter creates a JSON formatter.
func NewJSONFormatter(opts *OutputOptions) *JSONFormatter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &JSONFormatter{
		writer:  os.Stdout,
		options: opts,
	}
}

// NewJSONFormatterWithWriter creates a formatter with custom writer (for testing).
func NewJSONFormatterWithWriter(w io.Writer, opts *OutputOptions) *JSONFormatter {
	jf := NewJSONFormatter(opts)
	jf.writer = w
	return jf
}

// JSONOutput represents the complete JSON output structure.
type JSONOutput struct {
	Tool    JSONTool     `json:"tool"`
	Scan    JSONScan     `json:"scan"`
	Results []JSONResult `json:"results"`
	Summary JSONSummary  `json:"summary"`
	Errors  []string     `json:"errors,omitempty"`
}

// JSONTool contains tool metadata.
type JSONTool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

// JSONScan contains scan metadata.
type JSONScan struct {
	Target        string  `json:"target"`
	Timestamp     string  `json:"timestamp"`
	Duration      float64 `json:"duration"`
	RulesExecuted int     `json:"rules_executed"` //nolint:tagliatelle
}

// JSONResult represents a single finding.
type JSONResult struct {
	RuleID     string         `json:"rule_id"`   //nolint:tagliatelle
	RuleName   string         `json:"rule_name"` //nolint:tagliatelle
	Message    string         `json:"message"`
	Severity   string         `json:"severity"`
	Confidence string         `json:"confidence"`
	Location   JSONLocation   `json:"location"`
	Detection  JSONDetection  `json:"detection"`
	Metadata   JSONMetadata   `json:"metadata"`
}

// JSONLocation contains finding location.
type JSONLocation struct {
	File     string       `json:"file"`
	Line     int          `json:"line"`
	Column   int          `json:"column,omitempty"`
	Function string       `json:"function,omitempty"`
	Snippet  *JSONSnippet `json:"snippet,omitempty"`
}

// JSONSnippet contains code context.
type JSONSnippet struct {
	StartLine int      `json:"start_line"` //nolint:tagliatelle
	EndLine   int      `json:"end_line"`   //nolint:tagliatelle
	Lines     []string `json:"lines"`
}

// JSONDetection contains detection method info.
type JSONDetection struct {
	Type            string          `json:"type"`
	Scope           string          `json:"scope,omitempty"`
	ConfidenceScore float64         `json:"confidence_score"` //nolint:tagliatelle
	Source          *JSONTaintNode  `json:"source,omitempty"`
	Sink            *JSONTaintNode  `json:"sink,omitempty"`
}

// JSONTaintNode represents source or sink in taint flow.
type JSONTaintNode struct {
	Line     int    `json:"line"`
	Variable string `json:"variable,omitempty"`
	Call     string `json:"call,omitempty"`
}

// JSONMetadata contains rule metadata.
type JSONMetadata struct {
	CWE        []string `json:"cwe,omitempty"`
	OWASP      []string `json:"owasp,omitempty"`
	References []string `json:"references,omitempty"`
}

// JSONSummary contains aggregated statistics.
type JSONSummary struct {
	Total           int            `json:"total"`
	BySeverity      map[string]int `json:"by_severity"`       //nolint:tagliatelle
	ByDetectionType map[string]int `json:"by_detection_type"` //nolint:tagliatelle
}

// ScanInfo contains metadata about the scan.
type ScanInfo struct {
	Target        string
	Version       string
	Duration      time.Duration
	RulesExecuted int
	Errors        []string
}

// Format outputs all detections as JSON.
func (f *JSONFormatter) Format(detections []*dsl.EnrichedDetection, summary *Summary, scanInfo ScanInfo) error {
	output := f.buildOutput(detections, summary, scanInfo)

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func (f *JSONFormatter) buildOutput(detections []*dsl.EnrichedDetection, summary *Summary, scanInfo ScanInfo) JSONOutput {
	version := scanInfo.Version
	if version == "" {
		version = "unknown"
	}

	output := JSONOutput{
		Tool: JSONTool{
			Name:    "Code Pathfinder",
			Version: version,
			URL:     "https://github.com/shivasurya/code-pathfinder",
		},
		Scan: JSONScan{
			Target:        scanInfo.Target,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Duration:      scanInfo.Duration.Seconds(),
			RulesExecuted: scanInfo.RulesExecuted,
		},
		Results: f.buildResults(detections),
		Summary: JSONSummary{
			Total:           summary.TotalFindings,
			BySeverity:      summary.BySeverity,
			ByDetectionType: summary.ByDetectionType,
		},
		Errors: scanInfo.Errors,
	}

	return output
}

func (f *JSONFormatter) buildResults(detections []*dsl.EnrichedDetection) []JSONResult {
	results := make([]JSONResult, 0, len(detections))

	for _, det := range detections {
		result := JSONResult{
			RuleID:     det.Rule.ID,
			RuleName:   det.Rule.Name,
			Message:    det.Rule.Description,
			Severity:   det.Rule.Severity,
			Confidence: det.ConfidenceLevel(),
			Location:   f.buildLocation(det),
			Detection:  f.buildDetection(det),
			Metadata:   f.buildMetadata(det),
		}
		results = append(results, result)
	}

	return results
}

func (f *JSONFormatter) buildLocation(det *dsl.EnrichedDetection) JSONLocation {
	loc := JSONLocation{
		File:     det.Location.RelPath,
		Line:     det.Location.Line,
		Column:   det.Location.Column,
		Function: det.Location.Function,
	}

	if loc.File == "" {
		loc.File = det.Location.FilePath
	}

	// Add snippet if available
	if len(det.Snippet.Lines) > 0 {
		lines := make([]string, len(det.Snippet.Lines))
		for i, sl := range det.Snippet.Lines {
			lines[i] = sl.Content
		}
		loc.Snippet = &JSONSnippet{
			StartLine: det.Snippet.StartLine,
			EndLine:   det.Snippet.StartLine + len(det.Snippet.Lines) - 1,
			Lines:     lines,
		}
	}

	return loc
}

func (f *JSONFormatter) buildDetection(det *dsl.EnrichedDetection) JSONDetection {
	detection := JSONDetection{
		Type:            string(det.DetectionType),
		ConfidenceScore: det.Detection.Confidence,
	}

	if det.DetectionType == dsl.DetectionTypeTaintLocal || det.DetectionType == dsl.DetectionTypeTaintGlobal {
		detection.Scope = det.Detection.Scope
		detection.Source = &JSONTaintNode{
			Line:     det.Detection.SourceLine,
			Variable: det.Detection.TaintedVar,
		}
		detection.Sink = &JSONTaintNode{
			Line: det.Detection.SinkLine,
			Call: det.Detection.SinkCall,
		}
	}

	return detection
}

func (f *JSONFormatter) buildMetadata(det *dsl.EnrichedDetection) JSONMetadata {
	return JSONMetadata{
		CWE:        det.Rule.CWE,
		OWASP:      det.Rule.OWASP,
		References: det.Rule.References,
	}
}
