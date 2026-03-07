package core

import (
	"encoding/json"
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

func TestStdlibClass_NewFields_BackwardCompatible(t *testing.T) {
	// Existing JSON without new fields should still deserialize correctly
	jsonData := `{
		"type": "builtins.type",
		"methods": {
			"read": {"return_type": "builtins.str", "confidence": 0.95, "params": [], "source": "stdlib"}
		},
		"docstring": "A file object"
	}`

	var cls StdlibClass
	err := json.Unmarshal([]byte(jsonData), &cls)
	assert.NoError(t, err)
	assert.Equal(t, "builtins.type", cls.Type)
	assert.NotNil(t, cls.Methods)
	assert.Nil(t, cls.Attributes)
	assert.Nil(t, cls.Bases)
	assert.Nil(t, cls.MRO)
	assert.Nil(t, cls.InheritedMethods)
	assert.Nil(t, cls.InheritedAttributes)
}

func TestStdlibClass_NewFields_RoundTrip(t *testing.T) {
	cls := StdlibClass{
		Type: "builtins.type",
		Methods: map[string]*StdlibFunction{
			"get": {ReturnType: "requests.Response", Confidence: 0.95, Params: []*FunctionParam{
				{Name: "url", Type: "builtins.str", Required: true},
			}, Source: "typeshed"},
		},
		Attributes: map[string]*StdlibAttribute{
			"status_code": {Type: "builtins.int", Confidence: 0.95, Kind: "attribute", Source: "typeshed"},
			"content":     {Type: "builtins.bytes", Confidence: 0.95, Kind: "property", Source: "typeshed"},
		},
		Bases: []string{"requests.sessions.SessionRedirectMixin"},
		MRO:   []string{"requests.Session", "requests.sessions.SessionRedirectMixin", "builtins.object"},
		InheritedMethods: map[string]*InheritedMember{
			"close": {
				ReturnType:    "builtins.NoneType",
				Confidence:    0.95,
				Source:        "typeshed",
				InheritedFrom: "io.IOBase",
			},
		},
		InheritedAttributes: map[string]*InheritedMember{
			"encoding": {
				Type:          "builtins.str",
				Confidence:    0.95,
				Source:        "typeshed",
				Kind:          "attribute",
				InheritedFrom: "io.TextIOWrapper",
			},
		},
	}

	data, err := json.Marshal(&cls)
	assert.NoError(t, err)

	var roundTripped StdlibClass
	err = json.Unmarshal(data, &roundTripped)
	assert.NoError(t, err)

	// Verify all fields survived the round trip
	assert.Equal(t, cls.Type, roundTripped.Type)
	assert.Len(t, roundTripped.Methods, 1)
	assert.Len(t, roundTripped.Attributes, 2)
	assert.Equal(t, "attribute", roundTripped.Attributes["status_code"].Kind)
	assert.Equal(t, "property", roundTripped.Attributes["content"].Kind)
	assert.Equal(t, []string{"requests.sessions.SessionRedirectMixin"}, roundTripped.Bases)
	assert.Len(t, roundTripped.MRO, 3)
	assert.Len(t, roundTripped.InheritedMethods, 1)
	assert.Equal(t, "io.IOBase", roundTripped.InheritedMethods["close"].InheritedFrom)
	assert.Len(t, roundTripped.InheritedAttributes, 1)
	assert.Equal(t, "io.TextIOWrapper", roundTripped.InheritedAttributes["encoding"].InheritedFrom)
}

func TestStdlibClass_OmitsEmptyNewFields(t *testing.T) {
	// When new fields are nil/empty, they should not appear in JSON
	cls := StdlibClass{
		Type:    "builtins.type",
		Methods: map[string]*StdlibFunction{},
	}

	data, err := json.Marshal(&cls)
	assert.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "attributes")
	assert.NotContains(t, jsonStr, "bases")
	assert.NotContains(t, jsonStr, "mro")
	assert.NotContains(t, jsonStr, "inherited_methods")
	assert.NotContains(t, jsonStr, "inherited_attributes")
}

func TestInheritedMember_RoundTrip(t *testing.T) {
	member := InheritedMember{
		ReturnType:    "builtins.str",
		Confidence:    0.95,
		Params:        []*FunctionParam{{Name: "self", Type: "builtins.object", Required: true}},
		Source:        "typeshed",
		InheritedFrom: "base.Module.BaseClass",
	}

	data, err := json.Marshal(&member)
	assert.NoError(t, err)

	var roundTripped InheritedMember
	err = json.Unmarshal(data, &roundTripped)
	assert.NoError(t, err)

	assert.Equal(t, member.ReturnType, roundTripped.ReturnType)
	assert.Equal(t, member.Confidence, roundTripped.Confidence)
	assert.Equal(t, member.InheritedFrom, roundTripped.InheritedFrom)
	assert.Len(t, roundTripped.Params, 1)
}

func TestInheritedMember_AttributeStyle(t *testing.T) {
	// InheritedMember used for attributes (Type field, no ReturnType)
	member := InheritedMember{
		Type:          "builtins.int",
		Confidence:    0.95,
		Source:        "typeshed",
		Kind:          "attribute",
		InheritedFrom: "django.views.View",
	}

	data, err := json.Marshal(&member)
	assert.NoError(t, err)

	var roundTripped InheritedMember
	err = json.Unmarshal(data, &roundTripped)
	assert.NoError(t, err)

	assert.Equal(t, "builtins.int", roundTripped.Type)
	assert.Empty(t, roundTripped.ReturnType)
	assert.Equal(t, "attribute", roundTripped.Kind)
	assert.Equal(t, "django.views.View", roundTripped.InheritedFrom)
}

func TestStdlibAttribute_NewFields(t *testing.T) {
	attr := StdlibAttribute{
		Type:       "builtins.int",
		Confidence: 0.95,
		Source:     "typeshed",
		Kind:       "property",
	}

	data, err := json.Marshal(&attr)
	assert.NoError(t, err)

	var roundTripped StdlibAttribute
	err = json.Unmarshal(data, &roundTripped)
	assert.NoError(t, err)

	assert.Equal(t, "builtins.int", roundTripped.Type)
	assert.Equal(t, "typeshed", roundTripped.Source)
	assert.Equal(t, "property", roundTripped.Kind)
}

func TestStdlibAttribute_BackwardCompatible(t *testing.T) {
	// Existing JSON without Source/Kind should still work
	jsonData := `{"type": "os._Environ", "behaves_like": "builtins.dict", "confidence": 0.9}`

	var attr StdlibAttribute
	err := json.Unmarshal([]byte(jsonData), &attr)
	assert.NoError(t, err)
	assert.Equal(t, "os._Environ", attr.Type)
	assert.Equal(t, "builtins.dict", attr.BehavesLike)
	assert.Empty(t, attr.Source)
	assert.Empty(t, attr.Kind)
}

func TestStdlibClass_ThirdPartyJSON_Deserialize(t *testing.T) {
	// Simulate JSON output from the Python converter (PR-01 + PR-02)
	jsonData := `{
		"type": "class",
		"methods": {
			"get": {
				"return_type": "requests.Response",
				"confidence": 0.95,
				"params": [{"name": "url", "type": "builtins.str", "required": true}],
				"source": "typeshed"
			}
		},
		"attributes": {
			"status_code": {"type": "builtins.int", "confidence": 0.95, "source": "typeshed", "kind": "attribute"},
			"content": {"type": "builtins.bytes", "confidence": 0.95, "source": "typeshed", "kind": "property"}
		},
		"bases": ["requests.sessions.SessionRedirectMixin"],
		"mro": ["requests.Response", "requests.sessions.SessionRedirectMixin", "builtins.object"],
		"inherited_methods": {
			"close": {
				"return_type": "builtins.NoneType",
				"confidence": 0.95,
				"source": "typeshed",
				"inherited_from": "io.IOBase"
			}
		},
		"inherited_attributes": {
			"encoding": {
				"type": "builtins.str",
				"confidence": 0.95,
				"source": "typeshed",
				"kind": "attribute",
				"inherited_from": "io.TextIOWrapper"
			}
		}
	}`

	var cls StdlibClass
	err := json.Unmarshal([]byte(jsonData), &cls)
	assert.NoError(t, err)

	assert.Equal(t, "class", cls.Type)
	assert.Len(t, cls.Methods, 1)
	assert.Equal(t, "requests.Response", cls.Methods["get"].ReturnType)
	assert.Len(t, cls.Attributes, 2)
	assert.Equal(t, "builtins.int", cls.Attributes["status_code"].Type)
	assert.Equal(t, "attribute", cls.Attributes["status_code"].Kind)
	assert.Equal(t, "property", cls.Attributes["content"].Kind)
	assert.Equal(t, []string{"requests.sessions.SessionRedirectMixin"}, cls.Bases)
	assert.Len(t, cls.MRO, 3)
	assert.Equal(t, "io.IOBase", cls.InheritedMethods["close"].InheritedFrom)
	assert.Equal(t, "io.TextIOWrapper", cls.InheritedAttributes["encoding"].InheritedFrom)
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
