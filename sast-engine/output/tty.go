package output

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IsTTY returns true if the writer is connected to a terminal.
func IsTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// GetTerminalWidth returns the terminal width, or 80 as default.
func GetTerminalWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		width, _, err := term.GetSize(int(f.Fd()))
		if err == nil && width > 0 {
			return width
		}
	}
	return 80 // Default terminal width
}
