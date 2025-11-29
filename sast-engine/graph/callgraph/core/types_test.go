package core

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/stretchr/testify/assert"
)

func TestNewCallGraph(t *testing.T) {
	cg := NewCallGraph()

	assert.NotNil(t, cg)
	assert.NotNil(t, cg.Edges)
	assert.NotNil(t, cg.ReverseEdges)
	assert.NotNil(t, cg.CallSites)
	assert.NotNil(t, cg.Functions)
	assert.Equal(t, 0, len(cg.Edges))
	assert.Equal(t, 0, len(cg.ReverseEdges))
}

func TestCallGraph_AddEdge(t *testing.T) {
	tests := []struct {
		name   string
		caller string
		callee string
	}{
		{
			name:   "Add single edge",
			caller: "myapp.views.get_user",
			callee: "myapp.db.query",
		},
		{
			name:   "Add edge with qualified names",
			caller: "myapp.utils.helpers.sanitize_input",
			callee: "myapp.utils.validators.validate_string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := NewCallGraph()
			cg.AddEdge(tt.caller, tt.callee)

			// Check forward edge
			assert.Contains(t, cg.Edges[tt.caller], tt.callee)
			assert.Equal(t, 1, len(cg.Edges[tt.caller]))

			// Check reverse edge
			assert.Contains(t, cg.ReverseEdges[tt.callee], tt.caller)
			assert.Equal(t, 1, len(cg.ReverseEdges[tt.callee]))
		})
	}
}

func TestCallGraph_AddEdge_MultipleCalls(t *testing.T) {
	cg := NewCallGraph()
	caller := "myapp.views.process"
	callees := []string{
		"myapp.db.query",
		"myapp.utils.sanitize",
		"myapp.logging.log",
	}

	for _, callee := range callees {
		cg.AddEdge(caller, callee)
	}

	// Verify all forward edges
	assert.Equal(t, 3, len(cg.Edges[caller]))
	for _, callee := range callees {
		assert.Contains(t, cg.Edges[caller], callee)
	}

	// Verify all reverse edges
	for _, callee := range callees {
		assert.Contains(t, cg.ReverseEdges[callee], caller)
		assert.Equal(t, 1, len(cg.ReverseEdges[callee]))
	}
}

func TestCallGraph_AddEdge_Duplicate(t *testing.T) {
	cg := NewCallGraph()
	caller := "myapp.views.get_user"
	callee := "myapp.db.query"

	// Add same edge twice
	cg.AddEdge(caller, callee)
	cg.AddEdge(caller, callee)

	// Should only appear once
	assert.Equal(t, 1, len(cg.Edges[caller]))
	assert.Contains(t, cg.Edges[caller], callee)
}

func TestCallGraph_AddCallSite(t *testing.T) {
	cg := NewCallGraph()
	caller := "myapp.views.get_user"
	callSite := CallSite{
		Target: "query",
		Location: Location{
			File:   "/path/to/views.py",
			Line:   42,
			Column: 10,
		},
		Arguments: []Argument{
			{Value: "user_id", IsVariable: true, Position: 0},
		},
		Resolved:  true,
		TargetFQN: "myapp.db.query",
	}

	cg.AddCallSite(caller, callSite)

	assert.Equal(t, 1, len(cg.CallSites[caller]))
	assert.Equal(t, callSite.Target, cg.CallSites[caller][0].Target)
	assert.Equal(t, callSite.Location.Line, cg.CallSites[caller][0].Location.Line)
}

func TestCallGraph_AddCallSite_Multiple(t *testing.T) {
	cg := NewCallGraph()
	caller := "myapp.views.process"

	callSites := []CallSite{
		{
			Target:    "query",
			Location:  Location{File: "/path/to/views.py", Line: 10, Column: 5},
			Resolved:  true,
			TargetFQN: "myapp.db.query",
		},
		{
			Target:    "sanitize",
			Location:  Location{File: "/path/to/views.py", Line: 15, Column: 8},
			Resolved:  true,
			TargetFQN: "myapp.utils.sanitize",
		},
	}

	for _, cs := range callSites {
		cg.AddCallSite(caller, cs)
	}

	assert.Equal(t, 2, len(cg.CallSites[caller]))
}

func TestCallGraph_GetCallers(t *testing.T) {
	cg := NewCallGraph()

	// Set up call graph:
	// main → helper
	// main → util
	// process → helper
	cg.AddEdge("myapp.main", "myapp.helper")
	cg.AddEdge("myapp.main", "myapp.util")
	cg.AddEdge("myapp.process", "myapp.helper")

	tests := []struct {
		name           string
		callee         string
		expectedCount  int
		expectedCallers []string
	}{
		{
			name:           "Function with multiple callers",
			callee:         "myapp.helper",
			expectedCount:  2,
			expectedCallers: []string{"myapp.main", "myapp.process"},
		},
		{
			name:           "Function with single caller",
			callee:         "myapp.util",
			expectedCount:  1,
			expectedCallers: []string{"myapp.main"},
		},
		{
			name:           "Function with no callers",
			callee:         "myapp.main",
			expectedCount:  0,
			expectedCallers: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callers := cg.GetCallers(tt.callee)
			assert.Equal(t, tt.expectedCount, len(callers))
			for _, expectedCaller := range tt.expectedCallers {
				assert.Contains(t, callers, expectedCaller)
			}
		})
	}
}

func TestCallGraph_GetCallees(t *testing.T) {
	cg := NewCallGraph()

	// Set up call graph:
	// main → helper, util, logger
	// process → db
	cg.AddEdge("myapp.main", "myapp.helper")
	cg.AddEdge("myapp.main", "myapp.util")
	cg.AddEdge("myapp.main", "myapp.logger")
	cg.AddEdge("myapp.process", "myapp.db")

	tests := []struct {
		name           string
		caller         string
		expectedCount  int
		expectedCallees []string
	}{
		{
			name:           "Function with multiple callees",
			caller:         "myapp.main",
			expectedCount:  3,
			expectedCallees: []string{"myapp.helper", "myapp.util", "myapp.logger"},
		},
		{
			name:           "Function with single callee",
			caller:         "myapp.process",
			expectedCount:  1,
			expectedCallees: []string{"myapp.db"},
		},
		{
			name:           "Function with no callees",
			caller:         "myapp.helper",
			expectedCount:  0,
			expectedCallees: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callees := cg.GetCallees(tt.caller)
			assert.Equal(t, tt.expectedCount, len(callees))
			for _, expectedCallee := range tt.expectedCallees {
				assert.Contains(t, callees, expectedCallee)
			}
		})
	}
}

func TestNewModuleRegistry(t *testing.T) {
	mr := NewModuleRegistry()

	assert.NotNil(t, mr)
	assert.NotNil(t, mr.Modules)
	assert.NotNil(t, mr.ShortNames)
	assert.NotNil(t, mr.ResolvedImports)
	assert.Equal(t, 0, len(mr.Modules))
}

func TestModuleRegistry_AddModule(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		filePath   string
		shortName  string
	}{
		{
			name:       "Simple module",
			modulePath: "myapp.views",
			filePath:   "/path/to/myapp/views.py",
			shortName:  "views",
		},
		{
			name:       "Nested module",
			modulePath: "myapp.utils.helpers",
			filePath:   "/path/to/myapp/utils/helpers.py",
			shortName:  "helpers",
		},
		{
			name:       "Package init",
			modulePath: "myapp.utils",
			filePath:   "/path/to/myapp/utils/__init__.py",
			shortName:  "utils",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := NewModuleRegistry()
			mr.AddModule(tt.modulePath, tt.filePath)

			// Check module is registered
			path, ok := mr.GetModulePath(tt.modulePath)
			assert.True(t, ok)
			assert.Equal(t, tt.filePath, path)

			// Check short name is indexed
			assert.Contains(t, mr.ShortNames[tt.shortName], tt.filePath)
		})
	}
}

func TestModuleRegistry_AddModule_AmbiguousShortNames(t *testing.T) {
	mr := NewModuleRegistry()

	// Add two modules with same short name
	mr.AddModule("myapp.utils.helpers", "/path/to/myapp/utils/helpers.py")
	mr.AddModule("lib.helpers", "/path/to/lib/helpers.py")

	// Both should be indexed under short name "helpers"
	assert.Equal(t, 2, len(mr.ShortNames["helpers"]))
	assert.Contains(t, mr.ShortNames["helpers"], "/path/to/myapp/utils/helpers.py")
	assert.Contains(t, mr.ShortNames["helpers"], "/path/to/lib/helpers.py")

	// But each should be accessible by full module path
	path1, ok1 := mr.GetModulePath("myapp.utils.helpers")
	assert.True(t, ok1)
	assert.Equal(t, "/path/to/myapp/utils/helpers.py", path1)

	path2, ok2 := mr.GetModulePath("lib.helpers")
	assert.True(t, ok2)
	assert.Equal(t, "/path/to/lib/helpers.py", path2)
}

func TestModuleRegistry_GetModulePath_NotFound(t *testing.T) {
	mr := NewModuleRegistry()

	path, ok := mr.GetModulePath("nonexistent.module")
	assert.False(t, ok)
	assert.Equal(t, "", path)
}

func TestNewImportMap(t *testing.T) {
	filePath := "/path/to/file.py"
	im := NewImportMap(filePath)

	assert.NotNil(t, im)
	assert.Equal(t, filePath, im.FilePath)
	assert.NotNil(t, im.Imports)
	assert.Equal(t, 0, len(im.Imports))
}

func TestImportMap_AddImport(t *testing.T) {
	tests := []struct {
		name  string
		alias string
		fqn   string
	}{
		{
			name:  "Simple import",
			alias: "utils",
			fqn:   "myapp.utils",
		},
		{
			name:  "Aliased import",
			alias: "clean",
			fqn:   "myapp.utils.sanitize",
		},
		{
			name:  "Full module import",
			alias: "myapp.db.models",
			fqn:   "myapp.db.models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewImportMap("/path/to/file.py")
			im.AddImport(tt.alias, tt.fqn)

			fqn, ok := im.Resolve(tt.alias)
			assert.True(t, ok)
			assert.Equal(t, tt.fqn, fqn)
		})
	}
}

func TestImportMap_Resolve_NotFound(t *testing.T) {
	im := NewImportMap("/path/to/file.py")

	fqn, ok := im.Resolve("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, "", fqn)
}

func TestImportMap_Multiple(t *testing.T) {
	im := NewImportMap("/path/to/file.py")

	imports := map[string]string{
		"utils":    "myapp.utils",
		"sanitize": "myapp.utils.sanitize",
		"clean":    "myapp.utils.clean",
		"db":       "myapp.db",
	}

	for alias, fqn := range imports {
		im.AddImport(alias, fqn)
	}

	// Verify all imports are resolvable
	for alias, expectedFqn := range imports {
		fqn, ok := im.Resolve(alias)
		assert.True(t, ok)
		assert.Equal(t, expectedFqn, fqn)
	}
}

func TestLocation(t *testing.T) {
	loc := Location{
		File:   "/path/to/file.py",
		Line:   42,
		Column: 10,
	}

	assert.Equal(t, "/path/to/file.py", loc.File)
	assert.Equal(t, 42, loc.Line)
	assert.Equal(t, 10, loc.Column)
}

func TestCallSite(t *testing.T) {
	cs := CallSite{
		Target: "sanitize",
		Location: Location{
			File:   "/path/to/views.py",
			Line:   15,
			Column: 8,
		},
		Arguments: []Argument{
			{Value: "user_input", IsVariable: true, Position: 0},
			{Value: "\"html\"", IsVariable: false, Position: 1},
		},
		Resolved:  true,
		TargetFQN: "myapp.utils.sanitize",
	}

	assert.Equal(t, "sanitize", cs.Target)
	assert.Equal(t, 15, cs.Location.Line)
	assert.Equal(t, 2, len(cs.Arguments))
	assert.True(t, cs.Resolved)
	assert.Equal(t, "myapp.utils.sanitize", cs.TargetFQN)
}

func TestArgument(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		isVariable bool
		position   int
	}{
		{
			name:       "Variable argument",
			value:      "user_input",
			isVariable: true,
			position:   0,
		},
		{
			name:       "String literal argument",
			value:      "\"hello\"",
			isVariable: false,
			position:   1,
		},
		{
			name:       "Number literal argument",
			value:      "42",
			isVariable: false,
			position:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arg := Argument{
				Value:      tt.value,
				IsVariable: tt.isVariable,
				Position:   tt.position,
			}

			assert.Equal(t, tt.value, arg.Value)
			assert.Equal(t, tt.isVariable, arg.IsVariable)
			assert.Equal(t, tt.position, arg.Position)
		})
	}
}

func TestCallGraph_WithFunctions(t *testing.T) {
	cg := NewCallGraph()

	// Create mock function nodes
	funcMain := &graph.Node{
		ID:   "main_id",
		Type: "function_definition",
		Name: "main",
		File: "/path/to/main.py",
	}

	funcHelper := &graph.Node{
		ID:   "helper_id",
		Type: "function_definition",
		Name: "helper",
		File: "/path/to/utils.py",
	}

	// Add functions to call graph
	cg.Functions["myapp.main"] = funcMain
	cg.Functions["myapp.utils.helper"] = funcHelper

	// Add edge
	cg.AddEdge("myapp.main", "myapp.utils.helper")

	// Verify we can access function metadata
	assert.Equal(t, "main", cg.Functions["myapp.main"].Name)
	assert.Equal(t, "helper", cg.Functions["myapp.utils.helper"].Name)
}

func TestExtractShortName(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		expected   string
	}{
		{
			name:       "Simple module",
			modulePath: "views",
			expected:   "views",
		},
		{
			name:       "Two components",
			modulePath: "myapp.views",
			expected:   "views",
		},
		{
			name:       "Three components",
			modulePath: "myapp.utils.helpers",
			expected:   "helpers",
		},
		{
			name:       "Deep nesting",
			modulePath: "myapp.api.v1.endpoints.users",
			expected:   "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractShortName(tt.modulePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "Item exists",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "Item does not exist",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "Empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}
