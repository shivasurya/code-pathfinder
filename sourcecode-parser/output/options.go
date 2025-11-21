package output

// VerbosityLevel controls output detail.
type VerbosityLevel int

const (
	// VerbosityDefault shows clean results only (no progress, no statistics).
	VerbosityDefault VerbosityLevel = iota
	// VerbosityVerbose adds statistics and summary info.
	VerbosityVerbose
	// VerbosityDebug adds timestamps and diagnostic messages.
	VerbosityDebug
)

// OutputOptions configures output behavior.
type OutputOptions struct {
	Verbosity    VerbosityLevel
	Format       OutputFormat
	FailOn       []string // Severities to fail on (empty = never fail)
	ProjectRoot  string   // Project root for relative paths
	ContextLines int      // Lines of context around findings (default 3)
}

// OutputFormat specifies the output format.
type OutputFormat string

const (
	FormatText  OutputFormat = "text"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatSARIF OutputFormat = "sarif"
)

// NewDefaultOptions returns options with sensible defaults.
func NewDefaultOptions() *OutputOptions {
	return &OutputOptions{
		Verbosity:    VerbosityDefault,
		Format:       FormatText,
		FailOn:       nil, // Never fail by default
		ContextLines: 3,
	}
}

// ShouldShowStatistics returns true if statistics should be displayed.
func (o *OutputOptions) ShouldShowStatistics() bool {
	return o.Verbosity >= VerbosityVerbose
}

// ShouldShowDebug returns true if debug output should be displayed.
func (o *OutputOptions) ShouldShowDebug() bool {
	return o.Verbosity >= VerbosityDebug
}
