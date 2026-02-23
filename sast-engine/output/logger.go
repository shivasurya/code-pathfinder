package output

import (
	"fmt"
	"io"
	"maps"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Logger provides structured logging with verbosity control.
type Logger struct {
	verbosity    VerbosityLevel
	writer       io.Writer
	startTime    time.Time
	timings      map[string]time.Duration
	isTTY        bool
	progressBar  *progressbar.ProgressBar
	showProgress bool
}

// NewLogger creates a logger with the specified verbosity.
// Output goes to stderr to keep stdout clean for results.
func NewLogger(verbosity VerbosityLevel) *Logger {
	writer := os.Stderr
	isTTY := IsTTY(writer)
	return &Logger{
		verbosity:    verbosity,
		writer:       writer,
		startTime:    time.Now(),
		timings:      make(map[string]time.Duration),
		isTTY:        isTTY,
		showProgress: isTTY,
	}
}

// NewLoggerWithWriter creates a logger with custom output writer.
// Primarily used for testing.
func NewLoggerWithWriter(verbosity VerbosityLevel, w io.Writer) *Logger {
	isTTY := IsTTY(w)
	return &Logger{
		verbosity:    verbosity,
		writer:       w,
		startTime:    time.Now(),
		timings:      make(map[string]time.Duration),
		isTTY:        isTTY,
		showProgress: isTTY,
	}
}

// Progress logs progress messages (shown in verbose and debug modes).
// Use for high-level progress like "Building code graph...".
func (l *Logger) Progress(format string, args ...any) {
	if l.verbosity >= VerbosityVerbose {
		fmt.Fprintf(l.writer, format+"\n", args...)
	}
}

// Statistic logs statistics (shown in verbose and debug modes).
// Use for counts and metrics like "Code graph built: 1234 nodes".
func (l *Logger) Statistic(format string, args ...any) {
	if l.verbosity >= VerbosityVerbose {
		fmt.Fprintf(l.writer, format+"\n", args...)
	}
}

// Debug logs debug diagnostics (shown only in debug mode).
// Includes elapsed time prefix for performance analysis.
func (l *Logger) Debug(format string, args ...any) {
	if l.verbosity >= VerbosityDebug {
		elapsed := time.Since(l.startTime)
		prefix := formatDuration(elapsed)
		fmt.Fprintf(l.writer, "[%s] %s\n", prefix, fmt.Sprintf(format, args...))
	}
}

// Warning logs warnings (always shown).
func (l *Logger) Warning(format string, args ...any) {
	fmt.Fprintf(l.writer, "Warning: %s\n", fmt.Sprintf(format, args...))
}

// Error logs errors (always shown).
func (l *Logger) Error(format string, args ...any) {
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
	maps.Copy(result, l.timings)
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

// IsTTY returns true if the logger's output is connected to a terminal.
func (l *Logger) IsTTY() bool {
	return l.isTTY
}

// GetWriter returns the logger's output writer.
func (l *Logger) GetWriter() io.Writer {
	return l.writer
}

// StartProgress creates and displays a progress bar.
// For indeterminate operations (total = -1), shows a spinner.
// For determinate operations (total > 0), shows percentage progress.
func (l *Logger) StartProgress(description string, total int) error {
	if !l.showProgress || !l.isTTY {
		// In non-TTY mode, just print the description
		l.Progress("%s...", description)
		return nil
	}

	// Clear any existing progress bar
	if l.progressBar != nil {
		_ = l.progressBar.Finish()
	}

	if total < 0 {
		// Indeterminate progress (spinner)
		l.progressBar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription(description),
			progressbar.OptionSetWriter(l.writer),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprintf(l.writer, "\n")
			}),
		)
	} else {
		// Determinate progress (percentage bar)
		l.progressBar = progressbar.NewOptions(total,
			progressbar.OptionSetDescription(description),
			progressbar.OptionSetWriter(l.writer),
			progressbar.OptionSetWidth(40),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprintf(l.writer, "\n")
			}),
			progressbar.OptionSetRenderBlankState(true),
		)
	}

	return nil
}

// UpdateProgress increments the progress bar by delta.
func (l *Logger) UpdateProgress(delta int) error {
	if !l.showProgress || !l.isTTY || l.progressBar == nil {
		return nil
	}

	return l.progressBar.Add(delta)
}

// FinishProgress completes and clears the progress bar.
func (l *Logger) FinishProgress() error {
	if !l.showProgress || !l.isTTY || l.progressBar == nil {
		return nil
	}

	err := l.progressBar.Finish()
	l.progressBar = nil
	return err
}

// SetProgressDescription updates the progress bar description.
func (l *Logger) SetProgressDescription(description string) {
	if !l.showProgress || !l.isTTY || l.progressBar == nil {
		return
	}

	l.progressBar.Describe(description)
}

// IsProgressEnabled returns true if progress bars are enabled.
func (l *Logger) IsProgressEnabled() bool {
	return l.showProgress && l.isTTY
}
