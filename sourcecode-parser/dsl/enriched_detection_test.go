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

func TestEnrichedDetection_ConfidenceLevel(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   string
	}{
		{"high confidence 0.9", 0.9, "high"},
		{"high confidence 0.8", 0.8, "high"},
		{"medium confidence 0.7", 0.7, "medium"},
		{"medium confidence 0.5", 0.5, "medium"},
		{"low confidence 0.4", 0.4, "low"},
		{"low confidence 0.0", 0.0, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := &EnrichedDetection{
				Detection: DataflowDetection{Confidence: tt.confidence},
			}
			got := ed.ConfidenceLevel()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEnrichedDetection_DetectionBadge(t *testing.T) {
	tests := []struct {
		name     string
		detType  DetectionType
		expected string
	}{
		{"pattern badge", DetectionTypePattern, "[Pattern]"},
		{"taint-local badge", DetectionTypeTaintLocal, "[Taint-Local]"},
		{"taint-global badge", DetectionTypeTaintGlobal, "[Taint-Global]"},
		{"unknown badge", DetectionType("unknown"), "[Unknown]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := &EnrichedDetection{DetectionType: tt.detType}
			got := ed.DetectionBadge()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
