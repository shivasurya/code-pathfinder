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
