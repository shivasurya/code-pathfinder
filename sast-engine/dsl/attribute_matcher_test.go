package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestAttributeMatcherExecutor_ExactMatch(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"test.upload": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(5),
				Def:             "name",
				Uses:            []string{"uploaded"},
				AttributeAccess: "uploaded.filename",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"uploaded.filename"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1)
	assert.Equal(t, "test.upload", matches[0].FunctionFQN)
	assert.Equal(t, 5, matches[0].Line)
	assert.Equal(t, "uploaded.filename", matches[0].CallSite.Target)
}

func TestAttributeMatcherExecutor_SuffixMatch(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"test.upload": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(5),
				Def:             "name",
				Uses:            []string{"f"},
				AttributeAccess: "f.filename",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"filename"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1, "Pattern 'filename' should match 'f.filename' via suffix")
}

func TestAttributeMatcherExecutor_NoMatch(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"test.config": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(1),
				Def:             "debug",
				Uses:            []string{"Config"},
				AttributeAccess: "Config.DEBUG",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.url"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Empty(t, matches, "Config.DEBUG should not match request.url")
}

func TestAttributeMatcherExecutor_MultiplePatterns(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"test.handler": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(2),
				Def:             "url",
				Uses:            []string{"request"},
				AttributeAccess: "request.url",
			},
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(3),
				Def:             "host",
				Uses:            []string{"request"},
				AttributeAccess: "request.host",
			},
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(4),
				Def:             "debug",
				Uses:            []string{"Config"},
				AttributeAccess: "Config.DEBUG",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.url", "request.host"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 2, "Should match request.url and request.host but not Config.DEBUG")
}

func TestAttributeMatcherExecutor_SkipsEmptyAttributeAccess(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"test.func": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(1),
				Def:             "x",
				Uses:            []string{"foo"},
				CallTarget:      "foo()",
				AttributeAccess: "",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"foo"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Empty(t, matches, "Should not match call statements")
}

func TestAttributeMatcherExecutor_NilCallGraph(t *testing.T) {
	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.url"},
	}

	executor := NewAttributeMatcherExecutor(ir, nil)
	matches := executor.Execute()

	assert.Empty(t, matches)
}

// ========== SUBSCRIPT-SOURCED ATTRIBUTE ACCESS (GAP-012) ==========
//
// These tests verify that AttributeAccess values extracted from subscript
// expressions (e.g., request.GET["key"] → "request.GET") are correctly
// matched by the attribute matcher, completing the end-to-end pipeline.

func TestAttributeMatcherExecutor_SubscriptOnAttribute_DjangoGET(t *testing.T) {
	// Simulates: cmd = request.GET["cmd"]
	// Statement extraction sets AttributeAccess="request.GET" (from subscript value)
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"views.cmd_view": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(3),
				Def:             "cmd",
				Uses:            []string{"request"},
				AttributeAccess: "request.GET",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.GET"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1)
	assert.Equal(t, "views.cmd_view", matches[0].FunctionFQN)
	assert.Equal(t, "request.GET", matches[0].CallSite.Target)
}

func TestAttributeMatcherExecutor_SubscriptOnAttribute_FlaskForm(t *testing.T) {
	// Simulates: folder = flask.request.form["pack_folder"]
	// Statement extraction sets AttributeAccess="flask.request.form"
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"views.edit_package": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(8),
				Def:             "folder",
				Uses:            []string{"flask"},
				AttributeAccess: "flask.request.form",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"request.form", "flask.request.form"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1)
	assert.Equal(t, "flask.request.form", matches[0].CallSite.Target)
}

func TestAttributeMatcherExecutor_SubscriptOnAttribute_OsEnviron(t *testing.T) {
	// Simulates: val = os.environ["SECRET_KEY"]
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"config.load": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(5),
				Def:             "val",
				Uses:            []string{"os"},
				AttributeAccess: "os.environ",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"os.environ"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1)
	assert.Equal(t, "os.environ", matches[0].CallSite.Target)
}

func TestAttributeMatcherExecutor_SubscriptOnAttribute_SuffixMatch(t *testing.T) {
	// Verify suffix matching works for subscript-sourced attribute access:
	// request.GET matches pattern "GET" via suffix
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"views.handler": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(2),
				Def:             "param",
				Uses:            []string{"request"},
				AttributeAccess: "request.GET",
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"GET"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1, "Suffix pattern 'GET' should match 'request.GET'")
}

func TestAttributeMatcherExecutor_SubscriptOnAttribute_NoFalsePositive(t *testing.T) {
	// Subscript on plain identifier (x = d["key"]) should NOT set AttributeAccess
	// This test verifies the matcher doesn't match when extraction correctly leaves it empty
	cg := core.NewCallGraph()
	cg.Statements = map[string][]*core.Statement{
		"util.parse": {
			{
				Type:            core.StatementTypeAssignment,
				LineNumber:      uint32(3),
				Def:             "val",
				Uses:            []string{"d"},
				AttributeAccess: "", // Plain subscript has no attribute chain
			},
		},
	}

	ir := &AttributeMatcherIR{
		Type:     "attribute_matcher",
		Patterns: []string{"d"},
	}

	executor := NewAttributeMatcherExecutor(ir, cg)
	matches := executor.Execute()

	assert.Empty(t, matches, "Plain dict subscript should not produce attribute match")
}
