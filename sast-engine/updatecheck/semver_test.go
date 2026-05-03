package updatecheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input        string
		major, minV  int
		patch        int
		ok           bool
	}{
		{input: "1.2.3", major: 1, minV: 2, patch: 3, ok: true},
		{input: "0.0.0", major: 0, minV: 0, patch: 0, ok: true},
		{input: "10.20.30", major: 10, minV: 20, patch: 30, ok: true},
		{input: "", ok: false},
		{input: "1.2", ok: false},
		{input: "1.2.3.4", ok: false},
		{input: "a.b.c", ok: false},
		{input: "1.x.3", ok: false},
		{input: "1.2.x", ok: false},
		{input: "dev", ok: false},
		{input: "1.2.", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			maj, minor, pat, ok := parseSemver(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.major, maj)
				assert.Equal(t, tt.minV, minor)
				assert.Equal(t, tt.patch, pat)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// Major differs
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		// Minor differs (major equal)
		{"1.2.0", "1.1.0", 1},
		{"1.1.0", "1.2.0", -1},
		// Patch differs (major and minor equal)
		{"1.0.2", "1.0.1", 1},
		{"1.0.1", "1.0.2", -1},
		// Equal
		{"1.0.0", "1.0.0", 0},
		{"2.1.1", "2.1.1", 0},
		// Malformed — returns 0, never panics
		{"dev", "1.0.0", 0},
		{"1.0.0", "dev", 0},
		{"dev", "dev", 0},
		{"", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.want, Compare(tt.a, tt.b))
		})
	}
}

func TestMatch_SingleConstraint(t *testing.T) {
	tests := []struct {
		rangeExpr string
		version   string
		wantMatch bool
		wantErr   bool
	}{
		// Less-than
		{"<2.0.0", "1.9.9", true, false},
		{"<2.0.0", "2.0.0", false, false},
		{"<2.0.0", "2.0.1", false, false},
		// Less-than-or-equal
		{"<=2.0.0", "1.9.9", true, false},
		{"<=2.0.0", "2.0.0", true, false},
		{"<=2.0.0", "2.0.1", false, false},
		// Greater-than
		{">1.0.0", "1.0.1", true, false},
		{">1.0.0", "1.0.0", false, false},
		{">1.0.0", "0.9.9", false, false},
		// Greater-than-or-equal
		{">=1.0.0", "1.0.0", true, false},
		{">=1.0.0", "1.0.1", true, false},
		{">=1.0.0", "0.9.9", false, false},
		// Exact match
		{"=2.0.1", "2.0.1", true, false},
		{"=2.0.1", "2.0.0", false, false},
		{"=2.0.1", "2.0.2", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.rangeExpr+"@"+tt.version, func(t *testing.T) {
			got, err := Match(tt.rangeExpr, tt.version)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMatch, got)
			}
		})
	}
}

func TestMatch_ANDConstraint(t *testing.T) {
	tests := []struct {
		rangeExpr string
		version   string
		wantMatch bool
		wantErr   bool
	}{
		// Both sides true
		{">=1.5.0 <2.0.0", "1.7.0", true, false},
		// First side false
		{">=1.5.0 <2.0.0", "1.0.0", false, false},
		// Second side false
		{">=1.5.0 <2.0.0", "2.1.0", false, false},
		// Both sides false
		{">=2.0.0 <1.0.0", "1.5.0", false, false},
		// Exact range boundary — in
		{">=1.5.0 <=2.0.0", "2.0.0", true, false},
		// Exact range boundary — out
		{">=1.5.0 <2.0.0", "2.0.0", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.rangeExpr+"@"+tt.version, func(t *testing.T) {
			got, err := Match(tt.rangeExpr, tt.version)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMatch, got)
			}
		})
	}
}

func TestMatch_Errors(t *testing.T) {
	tests := []struct {
		name      string
		rangeExpr string
		version   string
	}{
		{
			name:      "empty range expression",
			rangeExpr: "",
			version:   "1.0.0",
		},
		{
			name:      "too many constraints",
			rangeExpr: ">=1.0.0 <2.0.0 =1.5.0",
			version:   "1.5.0",
		},
		{
			name:      "no operator in constraint",
			rangeExpr: "1.0.0",
			version:   "1.0.0",
		},
		{
			name:      "invalid version in constraint",
			rangeExpr: "<1.0",
			version:   "0.9.0",
		},
		{
			name:      "malformed current version",
			rangeExpr: "<2.0.0",
			version:   "dev",
		},
		{
			name:      "first constraint malformed in AND expression",
			rangeExpr: "<1.0 <2.0.0",
			version:   "0.9.0",
		},
		{
			name:      "second constraint malformed in AND expression",
			rangeExpr: ">=1.0.0 <2.0",
			version:   "1.5.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Match(tt.rangeExpr, tt.version)
			require.Error(t, err, "expected error but got nil")
			assert.False(t, got)
		})
	}
}
