package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStdlibRegistry(t *testing.T) {
	registry := NewStdlibRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Modules)
	assert.Equal(t, 0, len(registry.Modules))
}

func TestStdlibRegistry_GetModule(t *testing.T) {
	registry := NewStdlibRegistry()

	// Add a module
	module := &StdlibModule{
		Module: "os",
		Functions: make(map[string]*StdlibFunction),
	}
	registry.Modules["os"] = module

	// Test getting existing module
	result := registry.GetModule("os")
	assert.NotNil(t, result)
	assert.Equal(t, "os", result.Module)

	// Test getting non-existent module
	result = registry.GetModule("nonexistent")
	assert.Nil(t, result)
}

func TestStdlibRegistry_HasModule(t *testing.T) {
	registry := NewStdlibRegistry()

	// Test non-existent module
	assert.False(t, registry.HasModule("os"))

	// Add a module
	module := &StdlibModule{Module: "os"}
	registry.Modules["os"] = module

	// Test existing module
	assert.True(t, registry.HasModule("os"))
}

func TestStdlibRegistry_GetFunction(t *testing.T) {
	registry := NewStdlibRegistry()

	// Add a module with a function
	module := &StdlibModule{
		Module: "os",
		Functions: map[string]*StdlibFunction{
			"getcwd": {
				ReturnType: "builtins.str",
				Confidence: 1.0,
			},
		},
	}
	registry.Modules["os"] = module

	// Test getting existing function
	fn := registry.GetFunction("os", "getcwd")
	assert.NotNil(t, fn)
	assert.Equal(t, "builtins.str", fn.ReturnType)
	assert.Equal(t, float32(1.0), fn.Confidence)

	// Test getting non-existent function
	fn = registry.GetFunction("os", "nonexistent")
	assert.Nil(t, fn)

	// Test getting function from non-existent module
	fn = registry.GetFunction("nonexistent", "getcwd")
	assert.Nil(t, fn)
}

func TestStdlibRegistry_GetClass(t *testing.T) {
	registry := NewStdlibRegistry()

	// Add a module with a class
	module := &StdlibModule{
		Module: "pathlib",
		Classes: map[string]*StdlibClass{
			"Path": {
				Type: "builtins.type",
				Methods: make(map[string]*StdlibFunction),
			},
		},
	}
	registry.Modules["pathlib"] = module

	// Test getting existing class
	cls := registry.GetClass("pathlib", "Path")
	assert.NotNil(t, cls)
	assert.Equal(t, "builtins.type", cls.Type)

	// Test getting non-existent class
	cls = registry.GetClass("pathlib", "NonExistent")
	assert.Nil(t, cls)

	// Test getting class from non-existent module
	cls = registry.GetClass("nonexistent", "Path")
	assert.Nil(t, cls)
}

func TestStdlibRegistry_GetConstant(t *testing.T) {
	registry := NewStdlibRegistry()

	// Add a module with a constant
	module := &StdlibModule{
		Module: "os",
		Constants: map[string]*StdlibConstant{
			"O_RDONLY": {
				Type:       "builtins.int",
				Value:      "0",
				Confidence: 1.0,
			},
		},
	}
	registry.Modules["os"] = module

	// Test getting existing constant
	constant := registry.GetConstant("os", "O_RDONLY")
	assert.NotNil(t, constant)
	assert.Equal(t, "builtins.int", constant.Type)
	assert.Equal(t, "0", constant.Value)

	// Test getting non-existent constant
	constant = registry.GetConstant("os", "nonexistent")
	assert.Nil(t, constant)

	// Test getting constant from non-existent module
	constant = registry.GetConstant("nonexistent", "O_RDONLY")
	assert.Nil(t, constant)
}

func TestStdlibRegistry_GetAttribute(t *testing.T) {
	registry := NewStdlibRegistry()

	// Add a module with an attribute
	module := &StdlibModule{
		Module: "os",
		Attributes: map[string]*StdlibAttribute{
			"environ": {
				Type:       "os._Environ",
				BehavesLike: "builtins.dict",
				Confidence: 0.9,
			},
		},
	}
	registry.Modules["os"] = module

	// Test getting existing attribute
	attr := registry.GetAttribute("os", "environ")
	assert.NotNil(t, attr)
	assert.Equal(t, "os._Environ", attr.Type)
	assert.Equal(t, "builtins.dict", attr.BehavesLike)

	// Test getting non-existent attribute
	attr = registry.GetAttribute("os", "nonexistent")
	assert.Nil(t, attr)

	// Test getting attribute from non-existent module
	attr = registry.GetAttribute("nonexistent", "environ")
	assert.Nil(t, attr)
}

func TestStdlibRegistry_ModuleCount(t *testing.T) {
	registry := NewStdlibRegistry()

	// Initially empty
	assert.Equal(t, 0, registry.ModuleCount())

	// Add modules
	registry.Modules["os"] = &StdlibModule{Module: "os"}
	assert.Equal(t, 1, registry.ModuleCount())

	registry.Modules["sys"] = &StdlibModule{Module: "sys"}
	assert.Equal(t, 2, registry.ModuleCount())

	registry.Modules["pathlib"] = &StdlibModule{Module: "pathlib"}
	assert.Equal(t, 3, registry.ModuleCount())
}

func TestStdlibRegistry_Integration(t *testing.T) {
	// Test a complete workflow
	registry := NewStdlibRegistry()

	// Add os module with various components
	osModule := &StdlibModule{
		Module: "os",
		Functions: map[string]*StdlibFunction{
			"getcwd": {ReturnType: "builtins.str", Confidence: 1.0},
			"chdir": {ReturnType: "builtins.NoneType", Confidence: 1.0},
		},
		Constants: map[string]*StdlibConstant{
			"O_RDONLY": {Type: "builtins.int", Value: "0", Confidence: 1.0},
		},
		Attributes: map[string]*StdlibAttribute{
			"environ": {Type: "os._Environ", Confidence: 0.9},
		},
	}
	registry.Modules["os"] = osModule

	// Verify module exists
	assert.True(t, registry.HasModule("os"))
	assert.Equal(t, 1, registry.ModuleCount())

	// Verify function
	getcwd := registry.GetFunction("os", "getcwd")
	assert.NotNil(t, getcwd)
	assert.Equal(t, "builtins.str", getcwd.ReturnType)

	// Verify constant
	oRdonly := registry.GetConstant("os", "O_RDONLY")
	assert.NotNil(t, oRdonly)
	assert.Equal(t, "0", oRdonly.Value)

	// Verify attribute
	environ := registry.GetAttribute("os", "environ")
	assert.NotNil(t, environ)
	assert.Equal(t, "os._Environ", environ.Type)
}
