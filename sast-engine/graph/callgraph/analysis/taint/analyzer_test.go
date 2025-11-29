package taint

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

//
// ========== TAINT STATE TESTS ==========
//

func TestTaintState_SetTainted(t *testing.T) {
	ts := NewTaintState()

	ts.SetTainted("x", "request.GET", 1.0, 10)

	assert.True(t, ts.IsTainted("x"))
	info := ts.GetTaintInfo("x")
	assert.NotNil(t, info)
	assert.Equal(t, "request.GET", info.Source)
	assert.Equal(t, 1.0, info.Confidence)
	assert.Equal(t, uint32(10), info.SourceLine)
}

func TestTaintState_SetUntainted(t *testing.T) {
	ts := NewTaintState()

	// Taint variable
	ts.SetTainted("x", "request.GET", 1.0, 10)
	assert.True(t, ts.IsTainted("x"))

	// Sanitize it
	ts.SetUntainted("x")
	assert.False(t, ts.IsTainted("x"))

	info := ts.GetTaintInfo("x")
	assert.Nil(t, info)
}

func TestTaintState_GetTaintInfo_Nonexistent(t *testing.T) {
	ts := NewTaintState()

	info := ts.GetTaintInfo("nonexistent")
	assert.Nil(t, info)
	assert.False(t, ts.IsTainted("nonexistent"))
}

//
// ========== TAINT ANALYSIS TESTS ==========
//

func TestAnalyzeIntraProceduralTaint_SimpleSource(t *testing.T) {
	// x = request.GET['input']
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{"request"},
		CallTarget: "request.GET",
	}

	statements := []*core.Statement{stmt1}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{},
		[]string{},
	)

	assert.True(t, summary.IsTainted("x"))
	assert.False(t, summary.HasDetections())
}

func TestAnalyzeIntraProceduralTaint_SimpleSink(t *testing.T) {
	// eval(x)
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeCall,
		Def:        "",
		Uses:       []string{"x"},
		CallTarget: "eval",
	}

	statements := []*core.Statement{stmt1}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{},
		[]string{"eval"},
		[]string{},
	)

	// Sink is called but x is not tainted
	assert.False(t, summary.HasDetections())
}

func TestAnalyzeIntraProceduralTaint_SourceToSink(t *testing.T) {
	// x = request.GET['input']
	// eval(x)
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{"request"},
		CallTarget: "request.GET",
	}
	stmt2 := &core.Statement{
		LineNumber: 2,
		Type:       core.StatementTypeCall,
		Def:        "",
		Uses:       []string{"x"},
		CallTarget: "eval",
	}

	statements := []*core.Statement{stmt1, stmt2}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{"eval"},
		[]string{},
	)

	assert.True(t, summary.IsTainted("x"))
	assert.True(t, summary.HasDetections())
	assert.Equal(t, 1, summary.GetDetectionCount())

	detection := summary.Detections[0]
	assert.Equal(t, uint32(1), detection.SourceLine)
	assert.Equal(t, uint32(2), detection.SinkLine)
	assert.Equal(t, "eval", detection.SinkCall)
}

func TestAnalyzeIntraProceduralTaint_AssignmentPropagation(t *testing.T) {
	// x = request.GET['input']
	// y = x
	// eval(y)
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{"request"},
		CallTarget: "request.GET",
	}
	stmt2 := &core.Statement{
		LineNumber: 2,
		Type:       core.StatementTypeAssignment,
		Def:        "y",
		Uses:       []string{"x"},
		CallTarget: "x",
	}
	stmt3 := &core.Statement{
		LineNumber: 3,
		Type:       core.StatementTypeCall,
		Def:        "",
		Uses:       []string{"y"},
		CallTarget: "eval",
	}

	statements := []*core.Statement{stmt1, stmt2, stmt3}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{"eval"},
		[]string{},
	)

	assert.True(t, summary.IsTainted("x"))
	assert.True(t, summary.IsTainted("y"))
	assert.True(t, summary.HasDetections())
}

func TestAnalyzeIntraProceduralTaint_CallPropagation(t *testing.T) {
	// x = request.GET['input']
	// y = x.upper()
	// eval(y)
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{"request"},
		CallTarget: "request.GET",
	}
	stmt2 := &core.Statement{
		LineNumber: 2,
		Type:       core.StatementTypeCall,
		Def:        "y",
		Uses:       []string{"x"},
		CallTarget: "upper",
	}
	stmt3 := &core.Statement{
		LineNumber: 3,
		Type:       core.StatementTypeCall,
		Def:        "",
		Uses:       []string{"y"},
		CallTarget: "eval",
	}

	statements := []*core.Statement{stmt1, stmt2, stmt3}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{"eval"},
		[]string{},
	)

	assert.True(t, summary.IsTainted("x"))
	assert.True(t, summary.IsTainted("y"))
	assert.True(t, summary.HasDetections())
}

func TestAnalyzeIntraProceduralTaint_Sanitizer(t *testing.T) {
	// x = request.GET['input']
	// y = html.escape(x)
	// eval(y)
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{"request"},
		CallTarget: "request.GET",
	}
	stmt2 := &core.Statement{
		LineNumber: 2,
		Type:       core.StatementTypeCall,
		Def:        "y",
		Uses:       []string{"x"},
		CallTarget: "html.escape",
	}
	stmt3 := &core.Statement{
		LineNumber: 3,
		Type:       core.StatementTypeCall,
		Def:        "",
		Uses:       []string{"y"},
		CallTarget: "eval",
	}

	statements := []*core.Statement{stmt1, stmt2, stmt3}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{"eval"},
		[]string{"html.escape"},
	)

	assert.True(t, summary.IsTainted("x"))
	assert.False(t, summary.IsTainted("y"), "Sanitizer should remove taint")
	assert.False(t, summary.HasDetections(), "Sanitizer should prevent detection")
}

func TestAnalyzeIntraProceduralTaint_NonPropagator(t *testing.T) {
	// x = request.GET['input']
	// y = len(x)
	// eval(y)
	stmt1 := &core.Statement{
		LineNumber: 1,
		Type:       core.StatementTypeAssignment,
		Def:        "x",
		Uses:       []string{"request"},
		CallTarget: "request.GET",
	}
	stmt2 := &core.Statement{
		LineNumber: 2,
		Type:       core.StatementTypeCall,
		Def:        "y",
		Uses:       []string{"x"},
		CallTarget: "len",
	}
	stmt3 := &core.Statement{
		LineNumber: 3,
		Type:       core.StatementTypeCall,
		Def:        "",
		Uses:       []string{"y"},
		CallTarget: "eval",
	}

	statements := []*core.Statement{stmt1, stmt2, stmt3}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{"eval"},
		[]string{},
	)

	assert.True(t, summary.IsTainted("x"))
	assert.False(t, summary.IsTainted("y"), "len() should not propagate taint")
	assert.False(t, summary.HasDetections())
}

func TestAnalyzeIntraProceduralTaint_EmptyFunction(t *testing.T) {
	statements := []*core.Statement{}
	defUseChain := core.BuildDefUseChains(statements)

	summary := AnalyzeIntraProceduralTaint(
		"test.func",
		statements,
		defUseChain,
		[]string{"request.GET"},
		[]string{"eval"},
		[]string{},
	)

	assert.False(t, summary.HasDetections())
	assert.Equal(t, 0, summary.GetTaintedVarCount())
}

//
// ========== STDLIB INTEGRATION TESTS ==========
//

func TestIsStdlibSource(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{"os.getenv", "os.getenv", true},
		{"os.environ", "os.environ", true},
		{"sys.argv", "sys.argv", true},
		{"socket.recv", "socket.recv", true},
		{"os.path.join", "os.path.join", false},
		{"unknown", "unknown.func", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStdlibSource(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStdlibSanitizer(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{"html.escape", "html.escape", true},
		{"urllib.parse.quote", "urllib.parse.quote", true},
		{"shlex.quote", "shlex.quote", true},
		{"unknown", "unknown.func", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStdlibSanitizer(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNonPropagator(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{"len", "len", true},
		{"type", "type", true},
		{"os.path.exists", "os.path.exists", true},
		{"os.path.isfile", "os.path.isfile", true},
		{"unknown", "unknown.func", false},
		{"upper", "upper", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNonPropagator(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

//
// ========== HELPER FUNCTION TESTS ==========
//

func TestSplitModuleFunction(t *testing.T) {
	tests := []struct {
		name           string
		callTarget     string
		expectedModule string
		expectedFunc   string
	}{
		{"builtin", "len", "", "len"},
		{"single module", "os.getenv", "os", "getenv"},
		{"nested module", "os.path.join", "os.path", "join"},
		{"deep nest", "a.b.c.d.func", "a.b.c.d", "func"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module, funcName := splitModuleFunction(tt.callTarget)
			assert.Equal(t, tt.expectedModule, module)
			assert.Equal(t, tt.expectedFunc, funcName)
		})
	}
}
