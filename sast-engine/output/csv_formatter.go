package output

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// CSVFormatter formats enriched detections as CSV.
type CSVFormatter struct {
	writer  io.Writer
	options *OutputOptions
}

// NewCSVFormatter creates a CSV formatter.
func NewCSVFormatter(opts *OutputOptions) *CSVFormatter {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &CSVFormatter{
		writer:  os.Stdout,
		options: opts,
	}
}

// NewCSVFormatterWithWriter creates a formatter with custom writer (for testing).
func NewCSVFormatterWithWriter(w io.Writer, opts *OutputOptions) *CSVFormatter {
	cf := NewCSVFormatter(opts)
	cf.writer = w
	return cf
}

// CSVHeaders returns the CSV column headers.
func CSVHeaders() []string {
	return []string{
		"severity",
		"confidence",
		"rule_id",
		"rule_name",
		"cwe",
		"owasp",
		"file",
		"line",
		"column",
		"function",
		"message",
		"detection_type",
		"detection_scope",
		"source_line",
		"sink_line",
		"tainted_var",
		"sink_call",
	}
}

// Format outputs all detections as CSV.
func (f *CSVFormatter) Format(detections []*dsl.EnrichedDetection) error {
	w := csv.NewWriter(f.writer)
	defer w.Flush()

	// Write header
	if err := w.Write(CSVHeaders()); err != nil {
		return err
	}

	// Write rows
	for _, det := range detections {
		row := f.buildRow(det)
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return w.Error()
}

func (f *CSVFormatter) buildRow(det *dsl.EnrichedDetection) []string {
	file := det.Location.RelPath
	if file == "" {
		file = det.Location.FilePath
	}

	cwe := ""
	if len(det.Rule.CWE) > 0 {
		cwe = det.Rule.CWE[0]
	}

	owasp := ""
	if len(det.Rule.OWASP) > 0 {
		owasp = det.Rule.OWASP[0]
	}

	return []string{
		det.Rule.Severity,                      // severity
		det.ConfidenceLevel(),                  // confidence
		det.Rule.ID,                            // rule_id
		det.Rule.Name,                          // rule_name
		cwe,                                    // cwe
		owasp,                                  // owasp
		file,                                   // file
		intToString(det.Location.Line),         // line
		intToString(det.Location.Column),       // column
		det.Location.Function,                  // function
		det.Rule.Description,                   // message
		string(det.DetectionType),              // detection_type
		det.Detection.Scope,                    // detection_scope
		intToString(det.Detection.SourceLine),  // source_line
		intToString(det.Detection.SinkLine),    // sink_line
		det.Detection.TaintedVar,               // tainted_var
		det.Detection.SinkCall,                 // sink_call
	}
}

func intToString(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
