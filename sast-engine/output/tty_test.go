package output

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestIsTTY_WithFile(t *testing.T) {
	// Test with stdout (may or may not be TTY depending on environment)
	result := IsTTY(os.Stdout)
	// We can't assert true/false since it depends on test runner
	// Just verify it doesn't panic and returns a bool
	_ = result
}

func TestIsTTY_WithBuffer(t *testing.T) {
	// Test with bytes.Buffer (definitely not a TTY)
	var buf bytes.Buffer
	result := IsTTY(&buf)

	if result {
		t.Error("bytes.Buffer should not be detected as TTY")
	}
}

func TestIsTTY_WithStderr(t *testing.T) {
	// Test with stderr
	result := IsTTY(os.Stderr)
	// Depends on environment
	_ = result
}

type mockWriter struct{}

func (mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestIsTTY_WithNonFileWriter(t *testing.T) {
	// Test with a writer that's not an os.File
	result := IsTTY(mockWriter{})

	if result {
		t.Error("non-file writer should not be detected as TTY")
	}
}

func TestGetTerminalWidth_WithFile(t *testing.T) {
	width := GetTerminalWidth(os.Stdout)

	// Should return either actual width or default 80
	if width <= 0 {
		t.Errorf("Terminal width should be positive, got: %d", width)
	}
}

func TestGetTerminalWidth_WithBuffer(t *testing.T) {
	var buf bytes.Buffer
	width := GetTerminalWidth(&buf)

	// Should return default width of 80 for non-TTY
	if width != 80 {
		t.Errorf("Expected default width 80, got: %d", width)
	}
}

func TestGetTerminalWidth_DefaultFallback(t *testing.T) {
	// Test with a mock writer that's definitely not a file
	width := GetTerminalWidth(mockWriter{})

	if width != 80 {
		t.Errorf("Expected default fallback to 80, got: %d", width)
	}
}

func TestGetTerminalWidth_MultipleWriterTypes(t *testing.T) {
	tests := []struct {
		name   string
		writer io.Writer
		want   int
	}{
		{"buffer should return 80", &bytes.Buffer{}, 80},
		{"mock writer should return 80", mockWriter{}, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTerminalWidth(tt.writer)
			if got != tt.want {
				t.Errorf("GetTerminalWidth() = %v, want %v", got, tt.want)
			}
		})
	}
}
