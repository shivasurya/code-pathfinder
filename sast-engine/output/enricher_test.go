package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

func TestNewEnricher(t *testing.T) {
	tests := []struct {
		name string
		opts *OutputOptions
	}{
		{"nil options uses defaults", nil},
		{"custom options preserved", &OutputOptions{Verbosity: VerbosityDebug}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnricher(nil, tt.opts)
			if e == nil {
				t.Fatal("expected non-nil enricher")
			}
			if e.fileCache == nil {
				t.Error("expected initialized fileCache")
			}
		})
	}
}

func TestDetermineDetectionType(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected dsl.DetectionType
	}{
		{"empty scope is pattern", "", dsl.DetectionTypePattern},
		{"local scope is taint-local", "local", dsl.DetectionTypeTaintLocal},
		{"global scope is taint-global", "global", dsl.DetectionTypeTaintGlobal},
	}

	e := NewEnricher(nil, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			det := dsl.DataflowDetection{Scope: tt.scope}
			got := e.determineDetectionType(det)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractFunctionFromFQN(t *testing.T) {
	tests := []struct {
		fqn      string
		expected string
	}{
		{"myapp.auth.login", "login"},
		{"package.Class.method", "method"},
		{"singlename", "singlename"},
		{"", ""},
		{"a.b.c.d.e.f.g", "g"},
	}

	for _, tt := range tests {
		t.Run(tt.fqn, func(t *testing.T) {
			got := extractFunctionFromFQN(tt.fqn)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalizeSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CRITICAL", "critical"},
		{"High", "high"},
		{"  medium  ", "medium"},
		{"low", "low"},
		{"info", "info"},
		{"unknown", "medium"},
		{"", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeSeverity(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractSnippet(t *testing.T) {
	// Create temp file with test content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")
	content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	e := NewEnricher(nil, &OutputOptions{ContextLines: 2})

	tests := []struct {
		name          string
		line          int
		expectedStart int
		expectedCount int
	}{
		{"middle line", 4, 2, 5},
		{"first line", 1, 1, 3},
		{"last line", 7, 5, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := dsl.LocationInfo{FilePath: testFile, Line: tt.line}
			snippet, err := e.extractSnippet(loc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if snippet.StartLine != tt.expectedStart {
				t.Errorf("StartLine: got %d, want %d", snippet.StartLine, tt.expectedStart)
			}
			if len(snippet.Lines) != tt.expectedCount {
				t.Errorf("line count: got %d, want %d", len(snippet.Lines), tt.expectedCount)
			}
			if snippet.HighlightLine != tt.line {
				t.Errorf("HighlightLine: got %d, want %d", snippet.HighlightLine, tt.line)
			}
		})
	}
}

func TestExtractSnippetMissingFile(t *testing.T) {
	e := NewEnricher(nil, nil)
	loc := dsl.LocationInfo{FilePath: "/nonexistent/file.py", Line: 10}
	_, err := e.extractSnippet(loc)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExtractSnippetEmptyPath(t *testing.T) {
	e := NewEnricher(nil, nil)
	loc := dsl.LocationInfo{FilePath: "", Line: 10}
	snippet, err := e.extractSnippet(loc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(snippet.Lines) != 0 {
		t.Errorf("expected empty snippet for empty path")
	}
}

func TestBuildReferenceURLs(t *testing.T) {
	e := NewEnricher(nil, nil)

	tests := []struct {
		name     string
		cwes     []string
		expected []string
	}{
		{
			"single CWE",
			[]string{"CWE-78"},
			[]string{"https://cwe.mitre.org/data/definitions/78.html"},
		},
		{
			"multiple CWEs",
			[]string{"CWE-78", "CWE-79"},
			[]string{
				"https://cwe.mitre.org/data/definitions/78.html",
				"https://cwe.mitre.org/data/definitions/79.html",
			},
		},
		{
			"empty CWEs",
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.buildReferenceURLs(tt.cwes)
			if len(got) != len(tt.expected) {
				t.Errorf("got %d refs, want %d", len(got), len(tt.expected))
			}
			for i, ref := range got {
				if ref != tt.expected[i] {
					t.Errorf("ref[%d]: got %q, want %q", i, ref, tt.expected[i])
				}
			}
		})
	}
}

func TestEnrichedDetectionConfidenceLevel(t *testing.T) {
	tests := []struct {
		confidence float64
		expected   string
	}{
		{0.9, "high"},
		{0.8, "high"},
		{0.7, "medium"},
		{0.5, "medium"},
		{0.3, "low"},
		{0.0, "low"},
	}

	for _, tt := range tests {
		ed := &dsl.EnrichedDetection{
			Detection: dsl.DataflowDetection{Confidence: tt.confidence},
		}
		got := ed.ConfidenceLevel()
		if got != tt.expected {
			t.Errorf("confidence %v: got %q, want %q", tt.confidence, got, tt.expected)
		}
	}
}

func TestEnrichedDetectionBadge(t *testing.T) {
	tests := []struct {
		detType  dsl.DetectionType
		expected string
	}{
		{dsl.DetectionTypePattern, "[Pattern]"},
		{dsl.DetectionTypeTaintLocal, "[Taint-Local]"},
		{dsl.DetectionTypeTaintGlobal, "[Taint-Global]"},
		{"unknown", "[Unknown]"},
	}

	for _, tt := range tests {
		ed := &dsl.EnrichedDetection{DetectionType: tt.detType}
		got := ed.DetectionBadge()
		if got != tt.expected {
			t.Errorf("type %v: got %q, want %q", tt.detType, got, tt.expected)
		}
	}
}

func TestFileCache(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cached.py")
	if err := os.WriteFile(testFile, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	e := NewEnricher(nil, nil)

	// First read
	lines1, err := e.readFileLines(testFile)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}

	// Second read should use cache
	lines2, err := e.readFileLines(testFile)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}

	// Verify same slice (pointer comparison)
	if &lines1[0] != &lines2[0] {
		t.Error("expected cached result")
	}
}

func TestShouldShowStatistics(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
		expected  bool
	}{
		{"default does not show stats", VerbosityDefault, false},
		{"verbose shows stats", VerbosityVerbose, true},
		{"debug shows stats", VerbosityDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &OutputOptions{Verbosity: tt.verbosity}
			got := opts.ShouldShowStatistics()
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShouldShowDebug(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
		expected  bool
	}{
		{"default does not show debug", VerbosityDefault, false},
		{"verbose does not show debug", VerbosityVerbose, false},
		{"debug shows debug", VerbosityDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &OutputOptions{Verbosity: tt.verbosity}
			got := opts.ShouldShowDebug()
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEnrichDetection(t *testing.T) {
	// Create temp file for snippet extraction
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")
	content := `def dangerous():
    user_input = input()
    exec(user_input)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create enricher with mock options
	opts := &OutputOptions{
		ProjectRoot:  tmpDir,
		ContextLines: 1,
	}
	e := NewEnricher(nil, opts)

	detection := dsl.DataflowDetection{
		FunctionFQN: "test.dangerous",
		SourceLine:  2,
		SinkLine:    3,
		TaintedVar:  "user_input",
		SinkCall:    "exec",
		Confidence:  0.8,
		Scope:       "local",
	}

	rule := dsl.RuleIR{}
	rule.Rule.ID = "code-injection"
	rule.Rule.Name = "Code Injection"
	rule.Rule.Severity = "critical"
	rule.Rule.Description = "Dangerous code execution"
	rule.Rule.CWE = "CWE-94"
	rule.Rule.OWASP = "A1:2017"

	enriched, err := e.EnrichDetection(detection, rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify detection type
	if enriched.DetectionType != dsl.DetectionTypeTaintLocal {
		t.Errorf("detection type: got %v, want %v", enriched.DetectionType, dsl.DetectionTypeTaintLocal)
	}

	// Verify rule metadata
	if enriched.Rule.ID != "code-injection" {
		t.Errorf("rule ID: got %v, want %v", enriched.Rule.ID, "code-injection")
	}
	if enriched.Rule.Severity != "critical" {
		t.Errorf("severity: got %v, want %v", enriched.Rule.Severity, "critical")
	}
	if len(enriched.Rule.CWE) != 1 || enriched.Rule.CWE[0] != "CWE-94" {
		t.Errorf("CWE: got %v, want [CWE-94]", enriched.Rule.CWE)
	}

	// Verify confidence level method
	confidence := enriched.ConfidenceLevel()
	if confidence != "high" {
		t.Errorf("confidence level: got %v, want high", confidence)
	}

	// Verify badge
	badge := enriched.DetectionBadge()
	if badge != "[Taint-Local]" {
		t.Errorf("badge: got %v, want [Taint-Local]", badge)
	}
}

func TestExtractRuleMetadata(t *testing.T) {
	e := NewEnricher(nil, nil)

	rule := dsl.RuleIR{}
	rule.Rule.ID = "sql-injection"
	rule.Rule.Name = "SQL Injection"
	rule.Rule.Severity = "HIGH"
	rule.Rule.Description = "SQL injection vulnerability"
	rule.Rule.CWE = "CWE-89"
	rule.Rule.OWASP = "A1:2017"

	meta := e.extractRuleMetadata(rule)

	if meta.ID != "sql-injection" {
		t.Errorf("ID: got %v, want sql-injection", meta.ID)
	}
	if meta.Severity != "high" {
		t.Errorf("severity: got %v, want high", meta.Severity)
	}
	if len(meta.CWE) != 1 || meta.CWE[0] != "CWE-89" {
		t.Errorf("CWE: got %v, want [CWE-89]", meta.CWE)
	}
	if len(meta.OWASP) != 1 || meta.OWASP[0] != "A1:2017" {
		t.Errorf("OWASP: got %v, want [A1:2017]", meta.OWASP)
	}
	if len(meta.References) == 0 {
		t.Error("expected references to be populated")
	}
}

func TestFallbackLocation(t *testing.T) {
	e := NewEnricher(nil, nil)

	detection := dsl.DataflowDetection{
		FunctionFQN: "myapp.auth.login.authenticate",
		SinkLine:    42,
	}

	loc := e.fallbackLocation(detection)

	if loc.Line != 42 {
		t.Errorf("line: got %d, want 42", loc.Line)
	}
	if loc.Function != "authenticate" {
		t.Errorf("function: got %v, want authenticate", loc.Function)
	}
	if loc.ClassName != "login" {
		t.Errorf("class name: got %v, want login", loc.ClassName)
	}
}

func TestBuildTaintPath(t *testing.T) {
	e := NewEnricher(nil, nil)

	detection := dsl.DataflowDetection{
		FunctionFQN: "myapp.process",
		SourceLine:  10,
		SinkLine:    20,
		TaintedVar:  "user_data",
	}

	path := e.buildTaintPath(detection)

	if len(path) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(path))
	}

	// Verify source node
	if !path[0].IsSource {
		t.Error("first node should be source")
	}
	if path[0].Variable != "user_data" {
		t.Errorf("source variable: got %v, want user_data", path[0].Variable)
	}
	if path[0].Location.Line != 10 {
		t.Errorf("source line: got %d, want 10", path[0].Location.Line)
	}

	// Verify sink node
	if !path[1].IsSink {
		t.Error("second node should be sink")
	}
	if path[1].Location.Line != 20 {
		t.Errorf("sink line: got %d, want 20", path[1].Location.Line)
	}
}

func TestEnrichAll(t *testing.T) {
	e := NewEnricher(nil, nil)

	detections := []dsl.DataflowDetection{
		{FunctionFQN: "test.func1", SinkLine: 10, Scope: "local"},
		{FunctionFQN: "test.func2", SinkLine: 20, Scope: "global"},
	}

	rule := dsl.RuleIR{}
	rule.Rule.ID = "test-rule"
	rule.Rule.Severity = "high"

	enriched, err := e.EnrichAll(detections, rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enriched) != 2 {
		t.Fatalf("expected 2 enriched detections, got %d", len(enriched))
	}

	if enriched[0].DetectionType != dsl.DetectionTypeTaintLocal {
		t.Errorf("first detection type: got %v, want taint-local", enriched[0].DetectionType)
	}
	if enriched[1].DetectionType != dsl.DetectionTypeTaintGlobal {
		t.Errorf("second detection type: got %v, want taint-global", enriched[1].DetectionType)
	}
}

func TestExtractLocationWithCallGraph(t *testing.T) {
	// Create callgraph with a function
	cg := core.NewCallGraph()
	cg.Functions["myapp.process"] = &graph.Node{
		Name: "process",
		SourceLocation: &graph.SourceLocation{
			File: "/project/myapp/handler.py",
		},
	}

	opts := &OutputOptions{ProjectRoot: "/project"}
	e := NewEnricher(cg, opts)

	detection := dsl.DataflowDetection{
		FunctionFQN: "myapp.process",
		SinkLine:    42,
	}

	loc := e.extractLocation(detection)

	if loc.FilePath != "/project/myapp/handler.py" {
		t.Errorf("file path: got %v, want /project/myapp/handler.py", loc.FilePath)
	}
	if loc.Function != "process" {
		t.Errorf("function: got %v, want process", loc.Function)
	}
	if loc.RelPath != "myapp/handler.py" {
		t.Errorf("rel path: got %v, want myapp/handler.py", loc.RelPath)
	}
	if loc.Line != 42 {
		t.Errorf("line: got %d, want 42", loc.Line)
	}
}

func TestExtractLocationWithoutCallGraph(t *testing.T) {
	e := NewEnricher(nil, nil)

	detection := dsl.DataflowDetection{
		FunctionFQN: "myapp.auth.login",
		SinkLine:    10,
	}

	loc := e.extractLocation(detection)

	// Should use fallback location
	if loc.Function != "login" {
		t.Errorf("function: got %v, want login", loc.Function)
	}
	if loc.ClassName != "auth" {
		t.Errorf("class name: got %v, want auth", loc.ClassName)
	}
	if loc.Line != 10 {
		t.Errorf("line: got %d, want 10", loc.Line)
	}
}
