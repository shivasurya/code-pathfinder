package builder

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestResolvePromotedMethod_NilRegistry(t *testing.T) {
	fqn, resolved, _ := resolvePromotedMethod("myapp.Handler", "Query", nil)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethod_NilStdlibLoader(t *testing.T) {
	registry := &core.GoModuleRegistry{
		DirToImport: map[string]string{},
		ImportToDir: map[string]string{},
	}
	fqn, resolved, _ := resolvePromotedMethod("myapp.Handler", "Query", registry)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethod_InvalidFQN(t *testing.T) {
	registry := &core.GoModuleRegistry{
		DirToImport: map[string]string{},
		ImportToDir: map[string]string{},
	}
	// No dot in FQN — splitGoTypeFQN returns false
	fqn, resolved, _ := resolvePromotedMethod("error", "Error", registry)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethodFromFields_NamedFieldSkipped(t *testing.T) {
	// Named fields (non-embedded) should be skipped
	fields := []*core.GoStructField{
		{Name: "Host", Type: "string", Exported: true},
		{Name: "Port", Type: "int", Exported: true},
	}
	fqn, resolved, _ := resolvePromotedMethodFromFields(fields, "Query", nil)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethodFromFields_EmbeddedNoStdlib(t *testing.T) {
	// Embedded field but no StdlibLoader — can't resolve
	fields := []*core.GoStructField{
		{Name: "", Type: "database/sql.DB", Exported: true},
	}
	fqn, resolved, _ := resolvePromotedMethodFromFields(fields, "Query", nil)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethodFromFields_EmbeddedInvalidFQN(t *testing.T) {
	// Embedded field with no dot in type — splitGoTypeFQN fails
	fields := []*core.GoStructField{
		{Name: "", Type: "error", Exported: true},
	}
	fqn, resolved, _ := resolvePromotedMethodFromFields(fields, "Error", nil)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethodFromFields_EmptyFields(t *testing.T) {
	fqn, resolved, _ := resolvePromotedMethodFromFields(nil, "Query", nil)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethodFromFields_PointerStripping(t *testing.T) {
	// Embedded *sql.DB — pointer prefix should be stripped
	fields := []*core.GoStructField{
		{Name: "", Type: "*database/sql.DB", Exported: true},
	}
	// Without StdlibLoader, can't validate — but pointer stripping should work
	fqn, resolved, _ := resolvePromotedMethodFromFields(fields, "Query", nil)
	assert.False(t, resolved) // no loader
	assert.Empty(t, fqn)
}

func TestResolvePromotedMethodFromFields_MixedFields(t *testing.T) {
	// Mix of named and embedded fields — only embedded checked
	fields := []*core.GoStructField{
		{Name: "Logger", Type: "log.Logger", Exported: true},   // named — skip
		{Name: "", Type: "database/sql.DB", Exported: true},     // embedded — check
		{Name: "Config", Type: "myapp.Config", Exported: true},  // named — skip
	}
	// Without StdlibLoader, still won't resolve, but exercises the loop
	fqn, resolved, _ := resolvePromotedMethodFromFields(fields, "Query", nil)
	assert.False(t, resolved)
	assert.Empty(t, fqn)
}
