package taint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBuiltinTaintTransparent_KnownFunctions(t *testing.T) {
	tests := []struct {
		name      string
		funcFQN   string
		wantOK    bool
		wantParams []int
	}{
		{"fmt.Sprintf all params", "fmt.Sprintf", true, []int{-1}},
		{"fmt.Errorf all params", "fmt.Errorf", true, []int{-1}},
		{"strings.Replace param 0", "strings.Replace", true, []int{0}},
		{"strings.ToLower param 0", "strings.ToLower", true, []int{0}},
		{"strings.TrimSpace param 0", "strings.TrimSpace", true, []int{0}},
		{"strings.Join param 0", "strings.Join", true, []int{0}},
		{"reflect.ValueOf param 0", "reflect.ValueOf", true, []int{0}},
		{"reflect.Value.String receiver", "reflect.Value.String", true, []int{}},
		{"context.WithValue params 0+2", "context.WithValue", true, []int{0, 2}},
		{"context.Context.Value receiver", "context.Context.Value", true, []int{}},
		{"base64 encode", "encoding/base64.StdEncoding.EncodeToString", true, []int{0}},
		{"url escape", "net/url.QueryEscape", true, []int{0}},
		{"unknown function", "myapp.CustomFunc", false, nil},
		{"empty string", "", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, ok := IsBuiltinTaintTransparent(tt.funcFQN)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantParams, params)
			}
		})
	}
}

func TestIsBuiltinTaintTransparent_AllEntriesValid(t *testing.T) {
	// Every entry in the map should have valid param indices
	for fqn, params := range builtinTaintTransparent {
		t.Run(fqn, func(t *testing.T) {
			require.NotNil(t, params, "params slice should not be nil for %s", fqn)
			for _, idx := range params {
				assert.True(t, idx >= -1, "param index should be >= -1 for %s, got %d", fqn, idx)
			}
		})
	}
}

func TestIsBuiltinTaintTransparent_Coverage(t *testing.T) {
	// Verify key stdlib packages are covered
	packages := map[string]bool{
		"fmt":            false,
		"strings":        false,
		"reflect":        false,
		"context":        false,
		"encoding":       false,
		"net/url":        false,
	}

	for fqn := range builtinTaintTransparent {
		for pkg := range packages {
			if len(fqn) >= len(pkg) && fqn[:len(pkg)] == pkg {
				packages[pkg] = true
			}
		}
	}

	for pkg, covered := range packages {
		assert.True(t, covered, "package %s should have at least one builtin summary", pkg)
	}
}

func TestBuildTaintTransferSummary_BuiltinShortCircuit(t *testing.T) {
	// When a function FQN matches a builtin, BuildTaintTransferSummary
	// should return a pre-built summary without analyzing statements.
	summary := BuildTaintTransferSummary(
		"fmt.Sprintf",
		nil, // no statements — stdlib function body not available
		[]string{"format", "args"},
		[]string{}, []string{}, []string{},
		nil, nil,
	)

	require.NotNil(t, summary)
	assert.Equal(t, "fmt.Sprintf", summary.FunctionFQN)
	// -1 means ALL params propagate → both param 0 and param 1 should be true
	assert.True(t, summary.ParamToReturn[0], "format param should propagate")
	assert.True(t, summary.ParamToReturn[1], "args param should propagate")
}

func TestBuildTaintTransferSummary_BuiltinSpecificParam(t *testing.T) {
	summary := BuildTaintTransferSummary(
		"strings.Replace",
		nil,
		[]string{"s", "old", "new", "n"},
		[]string{}, []string{}, []string{},
		nil, nil,
	)

	require.NotNil(t, summary)
	assert.True(t, summary.ParamToReturn[0], "param 0 (s) should propagate")
	assert.False(t, summary.ParamToReturn[1], "param 1 (old) should NOT propagate")
	assert.False(t, summary.ParamToReturn[2], "param 2 (new) should NOT propagate")
}

func TestBuildTaintTransferSummary_NonBuiltin(t *testing.T) {
	// Regular functions should NOT be short-circuited
	summary := BuildTaintTransferSummary(
		"myapp.ProcessInput",
		nil,
		[]string{"input"},
		[]string{}, []string{}, []string{},
		nil, nil,
	)

	require.NotNil(t, summary)
	// No builtin match — no ParamToReturn set (empty statements = no analysis)
	assert.False(t, summary.ParamToReturn[0])
}
