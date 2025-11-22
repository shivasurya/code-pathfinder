package output

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Logger provides structured logging with verbosity control.
type Logger struct {
	verbosity VerbosityLevel
	writer    io.Writer
	startTime time.Time
	timings   map[string]time.Duration
}

// NewLogger creates a logger with the specified verbosity.
// Output goes to stderr to keep stdout clean for results.
func NewLogger(verbosity VerbosityLevel) *Logger {
	return &Logger{
		verbosity: verbosity,
		writer:    os.Stderr,
		startTime: time.Now(),
		timings:   make(map[string]time.Duration),
	}
}

// NewLoggerWithWriter creates a logger with custom output writer.
// Primarily used for testing.
func NewLoggerWithWriter(verbosity VerbosityLevel, w io.Writer) *Logger {
	return &Logger{
		verbosity: verbosity,
		writer:    w,
		startTime: time.Now(),
		timings:   make(map[string]time.Duration),
	}
}

// Progress logs progress messages (shown in verbose and debug modes).
// Use for high-level progress like "Building code graph...".
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.verbosity >= VerbosityVerbose {
		fmt.Fprintf(l.writer, format+"\n", args...)
	}
}

// Statistic logs statistics (shown in verbose and debug modes).
// Use for counts and metrics like "Code graph built: 1234 nodes".
func (l *Logger) Statistic(format string, args ...interface{}) {
	if l.verbosity >= VerbosityVerbose {
		fmt.Fprintf(l.writer, format+"\n", args...)
	}
}

// Debug logs debug diagnostics (shown only in debug mode).
// Includes elapsed time prefix for performance analysis.
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbosity >= VerbosityDebug {
		elapsed := time.Since(l.startTime)
		prefix := formatDuration(elapsed)
		fmt.Fprintf(l.writer, "[%s] %s\n", prefix, fmt.Sprintf(format, args...))
	}
}

// Warning logs warnings (always shown).
func (l *Logger) Warning(format string, args ...interface{}) {
	fmt.Fprintf(l.writer, "Warning: %s\n", fmt.Sprintf(format, args...))
}

// Error logs errors (always shown).
func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Fprintf(l.writer, "Error: %s\n", fmt.Sprintf(format, args...))
}

// StartTiming begins timing a named operation.
func (l *Logger) StartTiming(name string) func() {
	start := time.Now()
	return func() {
		l.timings[name] = time.Since(start)
	}
}

// GetTiming returns the duration for a named operation.
func (l *Logger) GetTiming(name string) time.Duration {
	return l.timings[name]
}

// GetAllTimings returns all recorded timings.
func (l *Logger) GetAllTimings() map[string]time.Duration {
	result := make(map[string]time.Duration)
	for k, v := range l.timings {
		result[k] = v
	}
	return result
}

// PrintTimingSummary prints all timings (verbose mode only).
func (l *Logger) PrintTimingSummary() {
	if l.verbosity < VerbosityVerbose {
		return
	}
	fmt.Fprintln(l.writer, "\nTiming Summary:")
	for name, duration := range l.timings {
		fmt.Fprintf(l.writer, "  %s: %s\n", name, duration.Round(time.Millisecond))
	}
}

// formatDuration formats duration as MM:SS.mmm.
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, millis)
}

// Verbosity returns the current verbosity level.
func (l *Logger) Verbosity() VerbosityLevel {
	return l.verbosity
}

// IsVerbose returns true if verbose or debug mode is enabled.
func (l *Logger) IsVerbose() bool {
	return l.verbosity >= VerbosityVerbose
}

// IsDebug returns true if debug mode is enabled.
func (l *Logger) IsDebug() bool {
	return l.verbosity >= VerbosityDebug
}
