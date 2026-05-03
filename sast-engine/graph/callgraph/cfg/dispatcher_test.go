package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCFGForLanguage_Go(t *testing.T) {
	source := `package main

func foo() {
	x := 1
	if x > 0 {
		y := x
		_ = y
	}
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildCFGForLanguage("go", "test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, blockStmts)
}

func TestBuildCFGForLanguage_Python(t *testing.T) {
	source := `def foo():
    x = 1
`
	funcNode := parsePythonFunction(t, source)

	cfg, _, err := BuildCFGForLanguage("python", "test.foo", funcNode, []byte(source))
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestBuildCFGForLanguage_Unknown(t *testing.T) {
	_, _, err := BuildCFGForLanguage("ruby", "test.foo", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ruby")
}
