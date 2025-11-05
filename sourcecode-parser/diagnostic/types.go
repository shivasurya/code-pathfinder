package diagnostic

// FunctionMetadata contains all information about a function needed for diagnostic analysis.
type FunctionMetadata struct {
	// FilePath is the relative path to the source file
	// Example: "myapp/views.py"
	FilePath string

	// FunctionName is the simple function name
	// Example: "process_input"
	FunctionName string

	// FQN is the fully qualified name (module.Class.function)
	// Example: "myapp.views.process_input" or "myapp.models.User.save"
	FQN string

	// StartLine is the first line of the function definition (1-indexed)
	// Includes decorators if present
	StartLine int

	// EndLine is the last line of the function body (1-indexed)
	EndLine int

	// SourceCode is the complete function source code
	// Includes decorators, signature, and body
	SourceCode string

	// LOC is lines of code (EndLine - StartLine + 1)
	LOC int

	// HasDecorators indicates if function has decorators (@property, @classmethod, etc.)
	HasDecorators bool

	// ClassName is the containing class name (if method), empty if top-level function
	// Example: "User" for myapp.models.User.save
	ClassName string

	// IsMethod indicates if this is a class method (has self/cls parameter)
	IsMethod bool

	// IsAsync indicates if this is an async function
	IsAsync bool
}
