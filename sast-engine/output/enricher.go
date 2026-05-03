package output

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Enricher adds context and metadata to detections.
type Enricher struct {
	callgraph *core.CallGraph
	options   *OutputOptions
	fileCache map[string][]string // Cache file contents
}

// NewEnricher creates an enricher with the given callgraph and options.
func NewEnricher(cg *core.CallGraph, opts *OutputOptions) *Enricher {
	if opts == nil {
		opts = NewDefaultOptions()
	}
	return &Enricher{
		callgraph: cg,
		options:   opts,
		fileCache: make(map[string][]string),
	}
}

// EnrichDetection transforms a raw detection into an enriched detection.
func (e *Enricher) EnrichDetection(detection dsl.DataflowDetection, rule dsl.RuleIR) (*dsl.EnrichedDetection, error) {
	enriched := &dsl.EnrichedDetection{
		Detection:     detection,
		DetectionType: e.determineDetectionType(detection),
	}

	// Resolve location from FQN
	loc := e.extractLocation(detection)
	enriched.Location = loc

	// Resolve source/sink file paths for taint flows
	if enriched.DetectionType == dsl.DetectionTypeTaintLocal || enriched.DetectionType == dsl.DetectionTypeTaintGlobal {
		// Sink file = finding location file (already resolved)
		enriched.Detection.SinkFile = loc.RelPath
		if enriched.Detection.SinkFile == "" {
			enriched.Detection.SinkFile = loc.FilePath
		}

		// Source file: use SourceFunctionFQN if different from FunctionFQN
		if detection.SourceFunctionFQN != "" && detection.SourceFunctionFQN != detection.FunctionFQN {
			sourceLoc := e.extractLocation(dsl.DataflowDetection{
				FunctionFQN: detection.SourceFunctionFQN,
				SinkLine:    detection.SourceLine,
			})
			enriched.Detection.SourceFile = sourceLoc.RelPath
			if enriched.Detection.SourceFile == "" {
				enriched.Detection.SourceFile = sourceLoc.FilePath
			}
			// Store resolved source location for display
			enriched.SourceLocation = sourceLoc
			// Extract source code snippet for inter-procedural display
			if srcSnippet, err := e.extractSnippet(sourceLoc); err == nil {
				enriched.SourceSnippet = srcSnippet
			}
		} else {
			// Same file for source and sink
			enriched.Detection.SourceFile = enriched.Detection.SinkFile
		}
	}

	// Extract code snippet (sink)
	snippet, err := e.extractSnippet(loc)
	if err == nil {
		enriched.Snippet = snippet
	}

	// Extract rule metadata
	enriched.Rule = e.extractRuleMetadata(rule)

	// Build taint path for inter-procedural flows
	if enriched.DetectionType == dsl.DetectionTypeTaintGlobal {
		enriched.TaintPath = e.buildTaintPath(detection)
	}

	return enriched, nil
}

// determineDetectionType classifies the detection based on scope.
func (e *Enricher) determineDetectionType(detection dsl.DataflowDetection) dsl.DetectionType {
	// Check if this is a dataflow rule
	if detection.Scope == "" {
		return dsl.DetectionTypePattern
	}
	if detection.Scope == "local" {
		return dsl.DetectionTypeTaintLocal
	}
	return dsl.DetectionTypeTaintGlobal
}

// extractLocation resolves FQN to file path using callgraph.
func (e *Enricher) extractLocation(detection dsl.DataflowDetection) dsl.LocationInfo {
	loc := dsl.LocationInfo{
		Line: detection.SinkLine,
	}

	// Lookup function in callgraph
	if e.callgraph != nil {
		if fn, ok := e.callgraph.Functions[detection.FunctionFQN]; ok {
			if fn.SourceLocation != nil {
				loc.FilePath = fn.SourceLocation.File
			}
			loc.Function = fn.Name
			// Compute relative path
			if e.options.ProjectRoot != "" && loc.FilePath != "" {
				relPath, err := filepath.Rel(e.options.ProjectRoot, loc.FilePath)
				if err == nil {
					loc.RelPath = relPath
				}
			}
		}
	}

	// Fallback: parse FQN heuristically
	if loc.FilePath == "" {
		loc = e.fallbackLocation(detection)
	}

	return loc
}

// fallbackLocation creates location from FQN parsing when callgraph lookup fails.
func (e *Enricher) fallbackLocation(detection dsl.DataflowDetection) dsl.LocationInfo {
	loc := dsl.LocationInfo{
		Line:     detection.SinkLine,
		Function: extractFunctionFromFQN(detection.FunctionFQN),
	}

	// If FQN is already a file path (e.g. container rules use file path as FQN),
	// use it directly.
	if strings.Contains(detection.FunctionFQN, "/") || strings.Contains(detection.FunctionFQN, string(filepath.Separator)) {
		if _, err := os.Stat(detection.FunctionFQN); err == nil {
			loc.FilePath = detection.FunctionFQN
			if e.options.ProjectRoot != "" {
				if relPath, err := filepath.Rel(e.options.ProjectRoot, detection.FunctionFQN); err == nil {
					loc.RelPath = relPath
				}
			}
			return loc
		}
	}

	// C/C++ FQN format: "<relative/path>::<funcname>" or
	// "<relative/path>::<ns>::<class>::<method>". The first `::`-segment
	// is the project-relative source path; everything after is the
	// scope chain. Resolve the file path against the project root and
	// extract the function name from the trailing component.
	if strings.Contains(detection.FunctionFQN, "::") {
		segments := strings.SplitN(detection.FunctionFQN, "::", 2)
		relFile := segments[0]
		if relFile != "" {
			candidate := relFile
			if e.options.ProjectRoot != "" {
				candidate = filepath.Join(e.options.ProjectRoot, relFile)
			}
			if _, err := os.Stat(candidate); err == nil {
				loc.FilePath = candidate
				loc.RelPath = relFile
				if len(segments) > 1 {
					tail := strings.Split(segments[1], "::")
					loc.Function = tail[len(tail)-1]
					if len(tail) >= 2 {
						loc.ClassName = tail[len(tail)-2]
					}
				}
				return loc
			}
		}
	}

	// Try to extract file path from FQN
	// Format: module.submodule.function or package.Class.method
	parts := strings.Split(detection.FunctionFQN, ".")
	if len(parts) > 0 {
		// Try common patterns
		loc.Function = parts[len(parts)-1]
		if len(parts) > 1 {
			loc.ClassName = parts[len(parts)-2]
		}
	}

	// Try to resolve file path from FQN by converting module path to file path.
	// e.g. "app.views.login" → "app/views.py" or "com.example.Main.run" → "com/example/Main.java"
	if e.options.ProjectRoot != "" && len(parts) > 1 {
		moduleParts := parts[:len(parts)-1] // Drop function name
		modulePath := filepath.Join(e.options.ProjectRoot, filepath.Join(moduleParts...))
		// Try common source file extensions
		for _, ext := range []string{".py", ".java", ".go", ".js", ".ts", ".rb"} {
			candidate := modulePath + ext
			if _, err := os.Stat(candidate); err == nil {
				loc.FilePath = candidate
				if relPath, err := filepath.Rel(e.options.ProjectRoot, candidate); err == nil {
					loc.RelPath = relPath
				}
				break
			}
		}
	}

	return loc
}

// extractFunctionFromFQN extracts the bare function name from a fully
// qualified name. Two FQN shapes are supported:
//
//   - Dot-separated (Python, Go, Java): "pkg.Mod.func"  → "func"
//   - C/C++ scope-resolved:             "src/main.c::main",
//     "src/utils.cpp::ns::Class::method" → "main", "method"
//
// The C/C++ form is detected by the presence of "::" — C/C++ FQNs
// always contain at least one because the prefix is itself a path
// segment joined to the symbol with "::".
func extractFunctionFromFQN(fqn string) string {
	if strings.Contains(fqn, "::") {
		parts := strings.Split(fqn, "::")
		return parts[len(parts)-1]
	}
	parts := strings.Split(fqn, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fqn
}

// extractSnippet reads code context around the finding.
func (e *Enricher) extractSnippet(loc dsl.LocationInfo) (dsl.CodeSnippet, error) {
	snippet := dsl.CodeSnippet{
		HighlightLine: loc.Line,
	}

	if loc.FilePath == "" {
		return snippet, nil
	}

	// Get file contents (cached)
	lines, err := e.readFileLines(loc.FilePath)
	if err != nil {
		return snippet, err
	}

	// Calculate context range
	contextLines := e.options.ContextLines
	if contextLines == 0 {
		contextLines = 3
	}

	startLine := max(loc.Line-contextLines, 1)
	endLine := min(loc.Line+contextLines, len(lines))

	snippet.StartLine = startLine

	// Build snippet lines
	for i := startLine; i <= endLine; i++ {
		if i > 0 && i <= len(lines) {
			snippet.Lines = append(snippet.Lines, dsl.SnippetLine{
				Number:      i,
				Content:     lines[i-1],
				IsHighlight: i == loc.Line,
			})
		}
	}

	return snippet, nil
}

// readFileLines reads and caches file contents.
func (e *Enricher) readFileLines(filePath string) ([]string, error) {
	// Check cache
	if lines, ok := e.fileCache[filePath]; ok {
		return lines, nil
	}

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Cache for reuse
	e.fileCache[filePath] = lines
	return lines, nil
}

// extractRuleMetadata builds rule metadata from RuleIR.
func (e *Enricher) extractRuleMetadata(rule dsl.RuleIR) dsl.RuleMetadata {
	meta := dsl.RuleMetadata{
		ID:          rule.Rule.ID,
		Name:        rule.Rule.Name,
		Severity:    normalizeSeverity(rule.Rule.Severity),
		Description: rule.Rule.Description,
	}

	// Parse CWE
	if rule.Rule.CWE != "" {
		meta.CWE = []string{rule.Rule.CWE}
	}

	// Parse OWASP
	if rule.Rule.OWASP != "" {
		meta.OWASP = []string{rule.Rule.OWASP}
	}

	// Build reference URLs
	meta.References = e.buildReferenceURLs(meta.CWE)

	return meta
}

// normalizeSeverity ensures severity is lowercase and valid.
func normalizeSeverity(sev string) string {
	s := strings.ToLower(strings.TrimSpace(sev))
	switch s {
	case "critical", "high", "medium", "low", "info":
		return s
	default:
		return "medium" // Default to medium if unknown
	}
}

// buildReferenceURLs creates documentation links from CWE.
func (e *Enricher) buildReferenceURLs(cwes []string) []string {
	var refs []string

	for _, cwe := range cwes {
		// Extract number from CWE-XX format
		num := strings.TrimPrefix(strings.ToUpper(cwe), "CWE-")
		if num != "" {
			refs = append(refs, "https://cwe.mitre.org/data/definitions/"+num+".html")
		}
	}

	return refs
}

// buildTaintPath constructs inter-procedural taint path (stub for v1).
func (e *Enricher) buildTaintPath(detection dsl.DataflowDetection) []dsl.TaintPathNode {
	// For v1, return source and sink nodes only
	// Full path reconstruction is a future enhancement
	path := make([]dsl.TaintPathNode, 0, 2)

	// Source node
	sourceLoc := dsl.LocationInfo{
		Line:     detection.SourceLine,
		Function: extractFunctionFromFQN(detection.FunctionFQN),
	}
	path = append(path, dsl.TaintPathNode{
		Location:    sourceLoc,
		Description: "Taint originates here",
		Variable:    detection.TaintedVar,
		IsSource:    true,
	})

	// Sink node
	sinkLoc := dsl.LocationInfo{
		Line:     detection.SinkLine,
		Function: extractFunctionFromFQN(detection.FunctionFQN),
	}
	path = append(path, dsl.TaintPathNode{
		Location:    sinkLoc,
		Description: "Taint reaches dangerous sink",
		Variable:    detection.TaintedVar,
		IsSink:      true,
	})

	return path
}

// EnrichAll enriches multiple detections.
func (e *Enricher) EnrichAll(detections []dsl.DataflowDetection, rule dsl.RuleIR) ([]*dsl.EnrichedDetection, error) {
	enriched := make([]*dsl.EnrichedDetection, 0, len(detections))
	for _, det := range detections {
		e, err := e.EnrichDetection(det, rule)
		if err != nil {
			continue // Skip problematic detections
		}
		enriched = append(enriched, e)
	}
	return enriched, nil
}
