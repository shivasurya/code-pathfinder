package dsl

import "testing"

func TestDetectionType_String(t *testing.T) {
	tests := []struct {
		dt       DetectionType
		expected string
	}{
		{DetectionTypePattern, "pattern"},
		{DetectionTypeTaintLocal, "taint-local"},
		{DetectionTypeTaintGlobal, "taint-global"},
	}

	for _, tt := range tests {
		if string(tt.dt) != tt.expected {
			t.Errorf("got %q, want %q", string(tt.dt), tt.expected)
		}
	}
}

func TestLocationInfo_IsValid(t *testing.T) {
	valid := LocationInfo{FilePath: "/path/file.py", Line: 10}
	if valid.FilePath == "" || valid.Line == 0 {
		t.Error("expected valid location")
	}

	empty := LocationInfo{}
	if empty.FilePath != "" || empty.Line != 0 {
		t.Error("expected empty location")
	}
}
