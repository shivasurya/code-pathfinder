package output

import (
	"fmt"
	"io"

	"github.com/common-nighthawk/go-figure"
)

// BannerOptions configures the startup banner display.
type BannerOptions struct {
	ShowBanner  bool // Show ASCII art logo
	ShowVersion bool // Show version information
	ShowLicense bool // Show license information
}

// DefaultBannerOptions returns default banner configuration.
func DefaultBannerOptions() BannerOptions {
	return BannerOptions{
		ShowBanner:  true,
		ShowVersion: true,
		ShowLicense: true,
	}
}

// PrintBanner displays the pathfinder logo and information.
func PrintBanner(w io.Writer, version string, opts BannerOptions) {
	if w == nil {
		return
	}

	if !opts.ShowBanner {
		// Simple text-only banner
		if opts.ShowVersion {
			fmt.Fprintf(w, "Code Pathfinder v%s\n", version)
		}
		if opts.ShowVersion || opts.ShowLicense {
			fmt.Fprintln(w, "AI-Native Static Code Analysis | Graph-First Engine | Privacy-First")
		}
		if opts.ShowLicense {
			fmt.Fprintf(w, "AGPL-3.0 License | https://codepathfinder.dev\n")
		}
		if opts.ShowVersion || opts.ShowLicense {
			fmt.Fprintln(w)
		}
		return
	}

	// Generate ASCII art using go-figure
	asciiArt := GetASCIILogo()
	fmt.Fprintln(w, asciiArt)

	// Tagline
	fmt.Fprintln(w, "AI-Native Static Code Analysis | Graph-First Engine | Privacy-First")
	fmt.Fprintln(w)

	// Version and license info
	if opts.ShowVersion {
		fmt.Fprintf(w, "Version: %s | ", version)
	}

	if opts.ShowLicense {
		fmt.Fprint(w, "License: AGPL-3.0 | ")
	}

	fmt.Fprintln(w, "https://codepathfinder.dev")

	// Empty line separator
	fmt.Fprintln(w)
}

// GetASCIILogo generates the ASCII art logo for "Pathfinder".
func GetASCIILogo() string {
	// Use "standard" font for compact output
	fig := figure.NewFigure("Pathfinder", "standard", true)
	return fig.String()
}

// GetCompactBanner returns a single-line banner for non-TTY output.
func GetCompactBanner(version string) string {
	return fmt.Sprintf("Code Pathfinder v%s | AI-Native Static Code Analysis | https://codepathfinder.dev", version)
}

// ShouldShowBanner determines if banner should be displayed.
func ShouldShowBanner(isTTY bool, noBannerFlag bool) bool {
	// Never show if --no-banner is set
	if noBannerFlag {
		return false
	}
	// Show full banner only in TTY
	return isTTY
}
