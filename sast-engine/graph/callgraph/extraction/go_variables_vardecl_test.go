package extraction

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: run ExtractGoVariableAssignments on an in-memory source file
// rooted at /test (mapped to import path "test").
func extractVars(t *testing.T, src string, extraImports map[string]string) *resolution.GoTypeInferenceEngine {
	t.Helper()
	registry := &core.GoModuleRegistry{
		ModulePath:  "test",
		DirToImport: map[string]string{"/test": "test"},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)
	imports := map[string]string{}
	for k, v := range extraImports {
		imports[k] = v
	}
	importMap := &core.GoImportMap{Imports: imports}
	err := ExtractGoVariableAssignments("/test/main.go", []byte(src), typeEngine, registry, importMap, nil)
	require.NoError(t, err)
	return typeEngine
}

// getBinding returns the first binding for varName in the given scope, or nil.
func getBinding(engine *resolution.GoTypeInferenceEngine, scopeName, varName string) *resolution.GoVariableBinding {
	scope := engine.GetScope(scopeName)
	if scope == nil {
		return nil
	}
	bindings, ok := scope.Variables[varName]
	if !ok || len(bindings) == 0 {
		return nil
	}
	return bindings[0]
}

// --------------------------------------------------------------------------
// var_declaration: explicit type annotation
// --------------------------------------------------------------------------

// TestVarDecl_StdlibQualifiedType verifies that `var sb strings.Builder` creates
// a binding with TypeFQN == "strings.Builder".
func TestVarDecl_StdlibQualifiedType(t *testing.T) {
	src := `package main
import "strings"
func handler() {
	var sb strings.Builder
	_ = sb
}`
	engine := extractVars(t, src, map[string]string{"strings": "strings"})
	b := getBinding(engine, "test.handler", "sb")
	require.NotNil(t, b, "expected binding for sb")
	assert.Equal(t, "strings.Builder", b.Type.TypeFQN)
	assert.Equal(t, float32(0.9), b.Type.Confidence)
	assert.Equal(t, "var_declaration", b.Type.Source)
}

// TestVarDecl_BytesBuffer verifies `var buf bytes.Buffer`.
func TestVarDecl_BytesBuffer(t *testing.T) {
	src := `package main
import "bytes"
func handler() {
	var buf bytes.Buffer
	_ = buf
}`
	engine := extractVars(t, src, map[string]string{"bytes": "bytes"})
	b := getBinding(engine, "test.handler", "buf")
	require.NotNil(t, b)
	assert.Equal(t, "bytes.Buffer", b.Type.TypeFQN)
}

// TestVarDecl_SyncMutex verifies `var mu sync.Mutex`.
func TestVarDecl_SyncMutex(t *testing.T) {
	src := `package main
import "sync"
func handler() {
	var mu sync.Mutex
	mu.Lock()
}`
	engine := extractVars(t, src, map[string]string{"sync": "sync"})
	b := getBinding(engine, "test.handler", "mu")
	require.NotNil(t, b)
	assert.Equal(t, "sync.Mutex", b.Type.TypeFQN)
}

// TestVarDecl_NetURL verifies alias resolution: `var q url.Values` where
// "url" maps to "net/url".
func TestVarDecl_NetURL(t *testing.T) {
	src := `package main
import "net/url"
func handler() {
	var q url.Values
	_ = q
}`
	engine := extractVars(t, src, map[string]string{"url": "net/url"})
	b := getBinding(engine, "test.handler", "q")
	require.NotNil(t, b)
	assert.Equal(t, "net/url.Values", b.Type.TypeFQN)
}

// TestVarDecl_MultiName verifies `var x, y int` creates bindings for both names.
func TestVarDecl_MultiName(t *testing.T) {
	src := `package main
import "sync"
func handler() {
	var wg1, wg2 sync.WaitGroup
	_ = wg1
	_ = wg2
}`
	engine := extractVars(t, src, map[string]string{"sync": "sync"})
	b1 := getBinding(engine, "test.handler", "wg1")
	b2 := getBinding(engine, "test.handler", "wg2")
	require.NotNil(t, b1)
	require.NotNil(t, b2)
	assert.Equal(t, "sync.WaitGroup", b1.Type.TypeFQN)
	assert.Equal(t, "sync.WaitGroup", b2.Type.TypeFQN)
}

// TestVarDecl_GroupedVar verifies grouped var blocks `var ( a T; b U )`.
func TestVarDecl_GroupedVar(t *testing.T) {
	src := `package main
import (
	"bytes"
	"strings"
)
func handler() {
	var (
		buf bytes.Buffer
		sb  strings.Builder
	)
	_ = buf
	_ = sb
}`
	engine := extractVars(t, src, map[string]string{
		"bytes":   "bytes",
		"strings": "strings",
	})
	buf := getBinding(engine, "test.handler", "buf")
	sb := getBinding(engine, "test.handler", "sb")
	require.NotNil(t, buf)
	require.NotNil(t, sb)
	assert.Equal(t, "bytes.Buffer", buf.Type.TypeFQN)
	assert.Equal(t, "strings.Builder", sb.Type.TypeFQN)
}

// TestVarDecl_UnqualifiedSamePackage verifies that an unqualified type like
// `var svc Service` is qualified to "test.Service".
func TestVarDecl_UnqualifiedSamePackage(t *testing.T) {
	src := `package main
type Service struct{}
func handler() {
	var svc Service
	_ = svc
}`
	engine := extractVars(t, src, nil)
	b := getBinding(engine, "test.handler", "svc")
	require.NotNil(t, b)
	assert.Equal(t, "test.Service", b.Type.TypeFQN)
}

// TestVarDecl_InsideMethod verifies that var declarations inside a method body
// are correctly scoped to the method FQN (package.Type.Method).
func TestVarDecl_InsideMethod(t *testing.T) {
	src := `package main
import "strings"
type Renderer struct{}
func (r *Renderer) Render() string {
	var sb strings.Builder
	return sb.String()
}`
	engine := extractVars(t, src, map[string]string{"strings": "strings"})
	b := getBinding(engine, "test.Renderer.Render", "sb")
	require.NotNil(t, b, "expected binding for sb in method scope")
	assert.Equal(t, "strings.Builder", b.Type.TypeFQN)
}

// TestVarDecl_NoType_WithRHSValue verifies that `var x = someFunc()` falls back
// to RHS inference when there is no explicit type annotation.
func TestVarDecl_NoType_WithRHSValue(t *testing.T) {
	src := `package main
func GetName() string { return "test" }
func handler() {
	var name = GetName()
	_ = name
}`
	registry := &core.GoModuleRegistry{
		ModulePath:  "test",
		DirToImport: map[string]string{"/test": "test"},
	}
	engine := resolution.NewGoTypeInferenceEngine(registry)
	// Pre-populate return type so RHS inference can find it.
	engine.AddReturnType("test.GetName", &core.TypeInfo{
		TypeFQN: "builtin.string", Confidence: 1.0, Source: "return_type",
	})
	importMap := &core.GoImportMap{Imports: map[string]string{}}
	err := ExtractGoVariableAssignments("/test/main.go", []byte(src), engine, registry, importMap, nil)
	require.NoError(t, err)
	b := getBinding(engine, "test.handler", "name")
	require.NotNil(t, b, "expected binding for name via RHS inference")
	assert.Equal(t, "builtin.string", b.Type.TypeFQN)
}

// --------------------------------------------------------------------------
// extractReceiverName
// --------------------------------------------------------------------------

// TestReceiverName_MethodScope verifies that the receiver variable is added as
// a typed binding in the method's scope so that receiver.Field.Method() resolves.
func TestReceiverName_MethodScope(t *testing.T) {
	src := `package main
import "strings"
type Parser struct{}
func (p *Parser) Parse() string {
	var sb strings.Builder
	return sb.String()
}
`
	engine := extractVars(t, src, map[string]string{"strings": "strings"})

	// The receiver `p` should be bound with type "test.Parser".
	b := getBinding(engine, "test.Parser.Parse", "p")
	require.NotNil(t, b, "expected receiver binding for p")
	assert.Equal(t, "test.Parser", b.Type.TypeFQN)
	assert.InDelta(t, 0.95, float64(b.Type.Confidence), 0.01)
	assert.Equal(t, "receiver_declaration", b.Type.Source)
}

// TestReceiverName_ValueReceiver verifies value receivers (not pointer) are also bound.
func TestReceiverName_ValueReceiver(t *testing.T) {
	src := `package main
type Point struct{ X, Y int }
func (pt Point) String() string {
	return ""
}
`
	engine := extractVars(t, src, nil)
	b := getBinding(engine, "test.Point.String", "pt")
	require.NotNil(t, b, "expected receiver binding for value receiver pt")
	assert.Equal(t, "test.Point", b.Type.TypeFQN)
}
