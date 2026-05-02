package core_test

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestNewCModuleRegistry_AllocatesMaps(t *testing.T) {
	root := "/projects/myapp"
	reg := core.NewCModuleRegistry(root)

	assert.NotNil(t, reg, "registry must be non-nil")
	assert.Equal(t, root, reg.ProjectRoot, "ProjectRoot must round-trip")
	assert.NotNil(t, reg.FileToPrefix, "FileToPrefix must be allocated")
	assert.NotNil(t, reg.Includes, "Includes must be allocated")
	assert.NotNil(t, reg.FunctionIndex, "FunctionIndex must be allocated")

	// Maps must be writable without nil panics.
	reg.FileToPrefix["/abs/foo.c"] = "foo.c"
	reg.Includes["foo.c"] = []string{"bar.h"}
	reg.FunctionIndex["foo"] = []string{"foo.c::foo"}
	assert.Len(t, reg.FileToPrefix, 1)
	assert.Len(t, reg.Includes, 1)
	assert.Len(t, reg.FunctionIndex, 1)
}

func TestNewCppModuleRegistry_AllocatesMaps(t *testing.T) {
	root := "/projects/cppapp"
	reg := core.NewCppModuleRegistry(root)

	assert.NotNil(t, reg)
	assert.Equal(t, root, reg.ProjectRoot, "embedded ProjectRoot must round-trip")

	// Embedded C registry maps.
	assert.NotNil(t, reg.FileToPrefix)
	assert.NotNil(t, reg.Includes)
	assert.NotNil(t, reg.FunctionIndex)

	// C++-specific maps.
	assert.NotNil(t, reg.NamespaceIndex, "NamespaceIndex must be allocated")
	assert.NotNil(t, reg.ClassIndex, "ClassIndex must be allocated")

	// Embedded fields are addressable through the outer registry.
	reg.NamespaceIndex["mylib::process"] = "src/utils.cpp::mylib::process"
	reg.ClassIndex["Socket"] = []string{"src/net/socket.cpp::mylib::Socket"}
	reg.FunctionIndex["main"] = []string{"src/main.cpp::main"}
	assert.Len(t, reg.NamespaceIndex, 1)
	assert.Len(t, reg.ClassIndex, 1)
	assert.Len(t, reg.FunctionIndex, 1)
}
