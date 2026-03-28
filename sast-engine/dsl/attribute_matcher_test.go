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
