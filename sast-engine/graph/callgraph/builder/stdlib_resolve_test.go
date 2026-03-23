package builder

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	cgregistry "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
)

// newTestStdlibLoader creates a StdlibRegistryRemote with pre-populated cache.
// The manifest is set up so HasModule returns true, and the module cache is
// populated directly so no HTTP calls are made.
func newTestStdlibLoader(modules map[string]*core.StdlibModule) *cgregistry.StdlibRegistryRemote {
	loader := cgregistry.NewStdlibRegistryRemote("https://test.example.com", "3.14")

	// Build manifest entries so HasModule returns true
	entries := make([]*core.ModuleEntry, 0, len(modules))
	for name := range modules {
		entries = append(entries, &core.ModuleEntry{Name: name})
	}
	loader.Manifest = &core.Manifest{Modules: entries}

	// Populate cache directly
	for name, mod := range modules {
		loader.ModuleCache[name] = mod
	}

	return loader
}

func TestResolveStdlibVariableBindings_PhaseA(t *testing.T) {
	// Setup: sqlite3 module with connect function returning sqlite3.Connection
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"sqlite3": {
			Module: "sqlite3",
			Functions: map[string]*core.StdlibFunction{
				"connect": {
					ReturnType: "sqlite3.Connection",
					Confidence: 0.9,
				},
			},
			Classes: map[string]*core.StdlibClass{},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "conn",
		Type: &core.TypeInfo{
			TypeFQN:    "call:sqlite3.connect",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	binding := scope.GetVariable("conn")
	assert.NotNil(t, binding)
	assert.Equal(t, "sqlite3.Connection", binding.Type.TypeFQN)
	assert.Equal(t, "sqlite3.connect", binding.AssignedFrom)
	assert.InDelta(t, 0.5*0.9*0.95, float64(binding.Type.Confidence), 0.001)
	assert.Equal(t, "stdlib", binding.Type.Source)
}

func TestResolveStdlibVariableBindings_PhaseA_Constructor(t *testing.T) {
	// When GetFunction returns nil but GetClass finds the class, treat as constructor
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"pathlib": {
			Module:    "pathlib",
			Functions: map[string]*core.StdlibFunction{},
			Classes: map[string]*core.StdlibClass{
				"Path": {
					Type:    "class",
					Methods: map[string]*core.StdlibFunction{},
				},
			},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "p",
		Type: &core.TypeInfo{
			TypeFQN:    "call:pathlib.Path",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	binding := scope.GetVariable("p")
	assert.NotNil(t, binding)
	assert.Equal(t, "pathlib.Path", binding.Type.TypeFQN)
	assert.Equal(t, "pathlib.Path", binding.AssignedFrom)
	assert.InDelta(t, 0.5*0.95, float64(binding.Type.Confidence), 0.001)
}

func TestResolveStdlibVariableBindings_PhaseB(t *testing.T) {
	// Phase B: conn.cursor() where conn was resolved in Phase A
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"sqlite3": {
			Module: "sqlite3",
			Functions: map[string]*core.StdlibFunction{
				"connect": {
					ReturnType: "sqlite3.Connection",
					Confidence: 0.9,
				},
			},
			Classes: map[string]*core.StdlibClass{
				"Connection": {
					Type: "class",
					Methods: map[string]*core.StdlibFunction{
						"cursor": {
							ReturnType: "sqlite3.Cursor",
							Confidence: 0.85,
						},
					},
				},
			},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope

	// conn = sqlite3.connect(...)
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "conn",
		Type: &core.TypeInfo{
			TypeFQN:    "call:sqlite3.connect",
			Confidence: 0.5,
		},
	})
	// cur = conn.cursor()
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "cur",
		Type: &core.TypeInfo{
			TypeFQN:    "call:conn.cursor",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	// Phase A should resolve conn
	connBinding := scope.GetVariable("conn")
	assert.Equal(t, "sqlite3.Connection", connBinding.Type.TypeFQN)

	// Phase B should resolve cur
	curBinding := scope.GetVariable("cur")
	assert.NotNil(t, curBinding)
	assert.Equal(t, "sqlite3.Cursor", curBinding.Type.TypeFQN)
	assert.Equal(t, "sqlite3.Connection.cursor", curBinding.AssignedFrom)
	assert.Equal(t, "stdlib", curBinding.Type.Source)
}

func TestResolveStdlibVariableBindings_CDNUnknown_StaysUnresolved(t *testing.T) {
	// CDN has "unknown" return type; no hardcoded fallback — stays unresolved
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"sqlite3": {
			Module: "sqlite3",
			Functions: map[string]*core.StdlibFunction{
				"connect": {
					ReturnType: "unknown",
					Confidence: 0.5,
				},
			},
			Classes: map[string]*core.StdlibClass{},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "conn",
		Type: &core.TypeInfo{
			TypeFQN:    "call:sqlite3.connect",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	binding := scope.GetVariable("conn")
	assert.NotNil(t, binding)
	// CDN returned "unknown", no hardcoded fallback — type stays as call: prefix
	assert.Equal(t, "call:sqlite3.connect", binding.Type.TypeFQN)
}

func TestResolveStdlibVariableBindings_CDNMethodUnknown_StaysUnresolved(t *testing.T) {
	// Phase B: CDN method returns "unknown", no hardcoded fallback — stays unresolved
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"sqlite3": {
			Module: "sqlite3",
			Functions: map[string]*core.StdlibFunction{
				"connect": {
					ReturnType: "sqlite3.Connection",
					Confidence: 0.9,
				},
			},
			Classes: map[string]*core.StdlibClass{
				"Connection": {
					Type: "class",
					Methods: map[string]*core.StdlibFunction{
						"cursor": {
							ReturnType: "unknown",
							Confidence: 0.5,
						},
					},
				},
			},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "conn",
		Type: &core.TypeInfo{
			TypeFQN:    "call:sqlite3.connect",
			Confidence: 0.5,
		},
	})
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "cur",
		Type: &core.TypeInfo{
			TypeFQN:    "call:conn.cursor",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	curBinding := scope.GetVariable("cur")
	assert.NotNil(t, curBinding)
	// CDN method returned "unknown", no hardcoded fallback — stays unresolved
	assert.Equal(t, "call:conn.cursor", curBinding.Type.TypeFQN)
}

func TestResolveStdlibVariableBindings_NilRemote(t *testing.T) {
	// Should not panic when StdlibRemote is nil — type stays unresolved
	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = nil

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "conn",
		Type: &core.TypeInfo{
			TypeFQN:    "call:sqlite3.connect",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	// Should not panic; without CDN, type stays unresolved
	resolveStdlibVariableBindings(typeEngine, logger)

	binding := scope.GetVariable("conn")
	assert.NotNil(t, binding)
	// Nil remote, no CDN data — type stays as call: prefix
	assert.Equal(t, "call:sqlite3.connect", binding.Type.TypeFQN)
}

func TestResolveStdlibVariableBindings_NoKnownModule(t *testing.T) {
	// Module not in CDN and not in hardcoded list — should remain unresolved
	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = nil

	scope := resolution.NewFunctionScope("app.main")
	typeEngine.Scopes["app.main"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "x",
		Type: &core.TypeInfo{
			TypeFQN:    "call:unknownmod.func1",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	binding := scope.GetVariable("x")
	assert.NotNil(t, binding)
	assert.Equal(t, "call:unknownmod.func1", binding.Type.TypeFQN) // unchanged
}

func TestResolveStdlibVariableBindings_PhaseA_ClassMethodAlias(t *testing.T) {
	// tarfile.open is a module-level alias for TarFile.open (classmethod).
	// GetFunction("tarfile", "open") returns nil because it's not in Functions.
	// GetClass("tarfile", "open") returns nil because "open" is not a class.
	// FindClassMethodAlias should find TarFile.open and resolve its return type.
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"tarfile": {
			Module:    "tarfile",
			Functions: map[string]*core.StdlibFunction{},
			Classes: map[string]*core.StdlibClass{
				"TarFile": {
					Type: "class",
					Methods: map[string]*core.StdlibFunction{
						"open": {
							ReturnType: "tarfile.Self",
							Confidence: 0.95,
						},
						"extractall": {
							ReturnType: "builtins.NoneType",
							Confidence: 0.95,
						},
					},
				},
			},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.extract_archive")
	typeEngine.Scopes["app.extract_archive"] = scope
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "tar",
		Type: &core.TypeInfo{
			TypeFQN:    "call:tarfile.open",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	binding := scope.GetVariable("tar")
	assert.NotNil(t, binding)
	// "tarfile.Self" should be resolved to "tarfile.TarFile"
	assert.Equal(t, "tarfile.TarFile", binding.Type.TypeFQN)
	assert.Equal(t, "tarfile.open", binding.AssignedFrom)
	assert.Equal(t, "stdlib", binding.Type.Source)
}

func TestResolveStdlibVariableBindings_PhaseA_ClassMethodAlias_ChainToPhaseB(t *testing.T) {
	// Full chain: tarfile.open() → TarFile, then tar.extractall() via Phase B.
	loader := newTestStdlibLoader(map[string]*core.StdlibModule{
		"tarfile": {
			Module:    "tarfile",
			Functions: map[string]*core.StdlibFunction{},
			Classes: map[string]*core.StdlibClass{
				"TarFile": {
					Type: "class",
					Methods: map[string]*core.StdlibFunction{
						"open": {
							ReturnType: "tarfile.Self",
							Confidence: 0.95,
						},
						"extractall": {
							ReturnType: "builtins.NoneType",
							Confidence: 0.95,
						},
					},
				},
			},
		},
	})

	typeEngine := resolution.NewTypeInferenceEngine(nil)
	typeEngine.StdlibRemote = loader

	scope := resolution.NewFunctionScope("app.extract_archive")
	typeEngine.Scopes["app.extract_archive"] = scope

	// tar = tarfile.open(...)
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "tar",
		Type: &core.TypeInfo{
			TypeFQN:    "call:tarfile.open",
			Confidence: 0.5,
		},
	})
	// result = tar.extractall(...)
	scope.AddVariable(&resolution.VariableBinding{
		VarName: "result",
		Type: &core.TypeInfo{
			TypeFQN:    "call:tar.extractall",
			Confidence: 0.5,
		},
	})

	logger := output.NewLogger(output.VerbosityDefault)
	resolveStdlibVariableBindings(typeEngine, logger)

	// Phase A: tar → tarfile.TarFile
	tarBinding := scope.GetVariable("tar")
	assert.Equal(t, "tarfile.TarFile", tarBinding.Type.TypeFQN)

	// Phase B: result → builtins.NoneType (via TarFile.extractall)
	resultBinding := scope.GetVariable("result")
	assert.NotNil(t, resultBinding)
	assert.Equal(t, "builtins.NoneType", resultBinding.Type.TypeFQN)
	assert.Equal(t, "tarfile.TarFile.extractall", resultBinding.AssignedFrom)
}
