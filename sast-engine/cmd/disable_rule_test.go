package cmd

import (
	"strings"
	"testing"
)

func TestValidateDisableRules(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr string
	}{
		{name: "empty", input: nil, want: nil},
		{name: "single valid", input: []string{"SAST-CMD-001"}, want: []string{"SAST-CMD-001"}},
		{name: "multiple valid", input: []string{"SAST-CMD-001", "GO-SSRF-001"}, want: []string{"SAST-CMD-001", "GO-SSRF-001"}},
		{name: "dedup", input: []string{"SAST-CMD-001", "SAST-CMD-001"}, want: []string{"SAST-CMD-001"}},
		{name: "trim whitespace", input: []string{"  SAST-CMD-001  "}, want: []string{"SAST-CMD-001"}},
		{name: "drop empty entries", input: []string{"SAST-CMD-001", "", "GO-SSRF-001"}, want: []string{"SAST-CMD-001", "GO-SSRF-001"}},
		{name: "reject too long", input: []string{strings.Repeat("X", 65)}, wantErr: "rule id too long"},
		{name: "reject invalid chars", input: []string{"BAD ID"}, wantErr: "invalid rule id"},
		{name: "reject path-like", input: []string{"foo/bar"}, wantErr: "invalid rule id"},
		{name: "reject newline", input: []string{"BAD\nID"}, wantErr: "invalid rule id"},
		{name: "reject null byte", input: []string{"BAD\x00ID"}, wantErr: "invalid rule id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateDisableRules(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want err containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("index %d: want %q, got %q", i, tc.want[i], got[i])
				}
			}
		})
	}
}
