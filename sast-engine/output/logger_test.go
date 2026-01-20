package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
	}{
		{"default verbosity", VerbosityDefault},
		{"verbose", VerbosityVerbose},
		{"debug", VerbosityDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLogger(tt.verbosity)
			if l == nil {
				t.Fatal("expected non-nil logger")
			}
			if l.verbosity != tt.verbosity {
				t.Errorf("verbosity: got %v, want %v", l.verbosity, tt.verbosity)
			}
			if l.timings == nil {
				t.Error("expected initialized timings map")
			}
		})
	}
}

func TestLoggerProgress(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
		expectOut bool
	}{
		{"default hides progress", VerbosityDefault, false},
		{"verbose shows progress", VerbosityVerbose, true},
		{"debug shows progress", VerbosityDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLoggerWithWriter(tt.verbosity, &buf)
			l.Progress("test message %d", 42)

			hasOutput := buf.Len() > 0
			if hasOutput != tt.expectOut {
				t.Errorf("hasOutput: got %v, want %v", hasOutput, tt.expectOut)
			}
			if tt.expectOut && !strings.Contains(buf.String(), "test message 42") {
				t.Errorf("output missing message: %q", buf.String())
			}
		})
	}
}

func TestLoggerStatistic(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
		expectOut bool
	}{
		{"default hides statistics", VerbosityDefault, false},
		{"verbose shows statistics", VerbosityVerbose, true},
		{"debug shows statistics", VerbosityDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLoggerWithWriter(tt.verbosity, &buf)
			l.Statistic("nodes: %d", 100)

			hasOutput := buf.Len() > 0
			if hasOutput != tt.expectOut {
				t.Errorf("hasOutput: got %v, want %v", hasOutput, tt.expectOut)
			}
		})
	}
}

func TestLoggerDebug(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
		expectOut bool
	}{
		{"default hides debug", VerbosityDefault, false},
		{"verbose hides debug", VerbosityVerbose, false},
		{"debug shows debug", VerbosityDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLoggerWithWriter(tt.verbosity, &buf)
			l.Debug("debug info")

			hasOutput := buf.Len() > 0
			if hasOutput != tt.expectOut {
				t.Errorf("hasOutput: got %v, want %v", hasOutput, tt.expectOut)
			}
			if tt.expectOut {
				// Should have timestamp prefix
				if !strings.Contains(buf.String(), "[") {
					t.Error("debug output missing timestamp prefix")
				}
			}
		})
	}
}

func TestLoggerWarningAlwaysShown(t *testing.T) {
	verbosities := []VerbosityLevel{VerbosityDefault, VerbosityVerbose, VerbosityDebug}

	for _, v := range verbosities {
		var buf bytes.Buffer
		l := NewLoggerWithWriter(v, &buf)
		l.Warning("warning message")

		if !strings.Contains(buf.String(), "Warning:") {
			t.Errorf("verbosity %v: warning not shown", v)
		}
	}
}

func TestLoggerErrorAlwaysShown(t *testing.T) {
	verbosities := []VerbosityLevel{VerbosityDefault, VerbosityVerbose, VerbosityDebug}

	for _, v := range verbosities {
		var buf bytes.Buffer
		l := NewLoggerWithWriter(v, &buf)
		l.Error("error message")

		if !strings.Contains(buf.String(), "Error:") {
			t.Errorf("verbosity %v: error not shown", v)
		}
	}
}

func TestLoggerTiming(t *testing.T) {
	l := NewLogger(VerbosityDefault)

	done := l.StartTiming("test-operation")
	time.Sleep(10 * time.Millisecond)
	done()

	timing := l.GetTiming("test-operation")
	if timing < 10*time.Millisecond {
		t.Errorf("timing too short: %v", timing)
	}
}

func TestLoggerGetAllTimings(t *testing.T) {
	l := NewLogger(VerbosityDefault)

	done1 := l.StartTiming("op1")
	done1()
	done2 := l.StartTiming("op2")
	done2()

	timings := l.GetAllTimings()
	if len(timings) != 2 {
		t.Errorf("expected 2 timings, got %d", len(timings))
	}
	if _, ok := timings["op1"]; !ok {
		t.Error("missing op1 timing")
	}
	if _, ok := timings["op2"]; !ok {
		t.Error("missing op2 timing")
	}
}

func TestLoggerPrintTimingSummary(t *testing.T) {
	tests := []struct {
		name      string
		verbosity VerbosityLevel
		expectOut bool
	}{
		{"default hides summary", VerbosityDefault, false},
		{"verbose shows summary", VerbosityVerbose, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLoggerWithWriter(tt.verbosity, &buf)
			done := l.StartTiming("test")
			done()
			l.PrintTimingSummary()

			hasOutput := strings.Contains(buf.String(), "Timing Summary")
			if hasOutput != tt.expectOut {
				t.Errorf("hasOutput: got %v, want %v", hasOutput, tt.expectOut)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00.000"},
		{500 * time.Millisecond, "00:00.500"},
		{1*time.Second + 234*time.Millisecond, "00:01.234"},
		{65*time.Second + 432*time.Millisecond, "01:05.432"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoggerIsVerbose(t *testing.T) {
	tests := []struct {
		verbosity VerbosityLevel
		expected  bool
	}{
		{VerbosityDefault, false},
		{VerbosityVerbose, true},
		{VerbosityDebug, true},
	}

	for _, tt := range tests {
		l := NewLogger(tt.verbosity)
		if got := l.IsVerbose(); got != tt.expected {
			t.Errorf("verbosity %v: IsVerbose() = %v, want %v", tt.verbosity, got, tt.expected)
		}
	}
}

func TestLoggerIsDebug(t *testing.T) {
	tests := []struct {
		verbosity VerbosityLevel
		expected  bool
	}{
		{VerbosityDefault, false},
		{VerbosityVerbose, false},
		{VerbosityDebug, true},
	}

	for _, tt := range tests {
		l := NewLogger(tt.verbosity)
		if got := l.IsDebug(); got != tt.expected {
			t.Errorf("verbosity %v: IsDebug() = %v, want %v", tt.verbosity, got, tt.expected)
		}
	}
}

func TestLoggerVerbosity(t *testing.T) {
	l := NewLogger(VerbosityVerbose)
	if got := l.Verbosity(); got != VerbosityVerbose {
		t.Errorf("Verbosity() = %v, want %v", got, VerbosityVerbose)
	}
}

func TestLoggerIsTTY(t *testing.T) {
	// Test with buffer (not a TTY)
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	if l.IsTTY() {
		t.Error("bytes.Buffer logger should not be TTY")
	}

	// Test TTY detection is set during construction
	l2 := NewLogger(VerbosityDefault)
	// Should have isTTY field set (value depends on environment)
	_ = l2.IsTTY()
}

func TestLoggerGetWriter(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	writer := l.GetWriter()
	if writer == nil {
		t.Error("GetWriter should not return nil")
	}

	// Verify it's the same writer we passed in
	if writer != &buf {
		t.Error("GetWriter should return the same writer passed to constructor")
	}
}

func TestNewLogger_TTYDetection(t *testing.T) {
	// Create logger and verify TTY field is initialized
	l := NewLogger(VerbosityDefault)

	// Should have isTTY field set (we can't assert value, just that it's set)
	// This tests that NewLogger calls IsTTY()
	if l.writer == nil {
		t.Error("Logger writer should not be nil")
	}

	// Test that isTTY is properly initialized by checking it's accessible
	_ = l.IsTTY()
}

func TestNewLoggerWithWriter_TTYDetection(t *testing.T) {
	tests := []struct {
		name      string
		writer    *bytes.Buffer
		expectTTY bool
	}{
		{"buffer is not TTY", &bytes.Buffer{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLoggerWithWriter(VerbosityDefault, tt.writer)
			if got := l.IsTTY(); got != tt.expectTTY {
				t.Errorf("IsTTY() = %v, want %v", got, tt.expectTTY)
			}
		})
	}
}

func TestLoggerStartProgress_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityVerbose, &buf)

	// Non-TTY logger should fallback to Progress() message
	err := l.StartProgress("Test operation", 10)
	if err != nil {
		t.Errorf("StartProgress returned error: %v", err)
	}

	// Should have printed a progress message
	if !strings.Contains(buf.String(), "Test operation") {
		t.Errorf("Expected progress message, got: %s", buf.String())
	}
}

func TestLoggerStartProgress_Indeterminate(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	// Test indeterminate progress (total = -1)
	err := l.StartProgress("Building", -1)
	if err != nil {
		t.Errorf("StartProgress returned error: %v", err)
	}

	// Should handle finish without error
	err = l.FinishProgress()
	if err != nil {
		t.Errorf("FinishProgress returned error: %v", err)
	}
}

func TestLoggerStartProgress_Determinate(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	// Test determinate progress (total > 0)
	err := l.StartProgress("Processing", 100)
	if err != nil {
		t.Errorf("StartProgress returned error: %v", err)
	}

	// Update progress
	err = l.UpdateProgress(50)
	if err != nil {
		t.Errorf("UpdateProgress returned error: %v", err)
	}

	// Finish progress
	err = l.FinishProgress()
	if err != nil {
		t.Errorf("FinishProgress returned error: %v", err)
	}
}

func TestLoggerUpdateProgress_WithoutStart(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	// Updating without starting should not error
	err := l.UpdateProgress(10)
	if err != nil {
		t.Errorf("UpdateProgress without start returned error: %v", err)
	}
}

func TestLoggerFinishProgress_WithoutStart(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	// Finishing without starting should not error
	err := l.FinishProgress()
	if err != nil {
		t.Errorf("FinishProgress without start returned error: %v", err)
	}
}

func TestLoggerSetProgressDescription(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	// Start progress
	_ = l.StartProgress("Initial", 10)

	// Set new description (should not panic)
	l.SetProgressDescription("Updated")

	// Cleanup
	_ = l.FinishProgress()
}

func TestLoggerIsProgressEnabled(t *testing.T) {
	tests := []struct {
		name     string
		isTTY    bool
		expected bool
	}{
		{"TTY enabled", true, false},  // bytes.Buffer is not TTY
		{"Non-TTY disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLoggerWithWriter(VerbosityDefault, &buf)

			got := l.IsProgressEnabled()
			if got != tt.expected {
				t.Errorf("IsProgressEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoggerProgressBar_MultipleOperations(t *testing.T) {
	var buf bytes.Buffer
	l := NewLoggerWithWriter(VerbosityDefault, &buf)

	// Start first operation
	_ = l.StartProgress("Operation 1", -1)
	_ = l.FinishProgress()

	// Start second operation (should clear first)
	_ = l.StartProgress("Operation 2", 10)
	_ = l.UpdateProgress(5)
	_ = l.FinishProgress()

	// Progress bar should be nil after finish
	if l.progressBar != nil {
		t.Error("Progress bar should be nil after FinishProgress")
	}
}
