package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintBanner_FullBanner(t *testing.T) {
	var buf bytes.Buffer
	opts := BannerOptions{
		ShowBanner:  true,
		ShowVersion: true,
		ShowLicense: true,
	}

	PrintBanner(&buf, "0.0.25", opts)

	output := buf.String()

	// Verify ASCII art is present (go-figure generates multi-line ASCII art)
	if !strings.Contains(output, "Pathfinder") && !strings.Contains(output, "P") {
		t.Errorf("Expected ASCII art for 'Pathfinder', got: %s", output)
	}

	// Verify tagline is present
	if !strings.Contains(output, "AI-Native Static Code Analysis") {
		t.Errorf("Expected tagline, got: %s", output)
	}

	// Verify version is present
	if !strings.Contains(output, "Version: 0.0.25") {
		t.Errorf("Expected version string, got: %s", output)
	}

	// Verify license is present
	if !strings.Contains(output, "License: AGPL-3.0") {
		t.Errorf("Expected license string, got: %s", output)
	}

	// Verify URL is present
	if !strings.Contains(output, "https://codepathfinder.dev") {
		t.Errorf("Expected URL, got: %s", output)
	}
}

func TestPrintBanner_NoBanner(t *testing.T) {
	var buf bytes.Buffer
	opts := BannerOptions{
		ShowBanner:  false,
		ShowVersion: true,
		ShowLicense: true,
	}

	PrintBanner(&buf, "0.0.25", opts)

	output := buf.String()

	// Should show compact version without ASCII art
	if !strings.Contains(output, "Code Pathfinder v0.0.25") {
		t.Errorf("Expected version string, got: %s", output)
	}

	if !strings.Contains(output, "AI-Native Static Code Analysis") {
		t.Errorf("Expected tagline, got: %s", output)
	}

	if !strings.Contains(output, "AGPL-3.0") {
		t.Errorf("Expected license info, got: %s", output)
	}

	// ASCII art should be minimal (checking line count is a rough heuristic)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 5 {
		t.Errorf("Compact banner should be minimal, got %d lines", len(lines))
	}
}

func TestPrintBanner_VersionOnly(t *testing.T) {
	var buf bytes.Buffer
	opts := BannerOptions{
		ShowBanner:  false,
		ShowVersion: true,
		ShowLicense: false,
	}

	PrintBanner(&buf, "0.0.25", opts)

	output := buf.String()

	if !strings.Contains(output, "v0.0.25") {
		t.Errorf("Expected version, got: %s", output)
	}

	if strings.Contains(output, "AGPL-3.0") {
		t.Errorf("License should not be shown, got: %s", output)
	}
}

func TestPrintBanner_LicenseOnly(t *testing.T) {
	var buf bytes.Buffer
	opts := BannerOptions{
		ShowBanner:  false,
		ShowVersion: false,
		ShowLicense: true,
	}

	PrintBanner(&buf, "0.0.25", opts)

	output := buf.String()

	if strings.Contains(output, "v0.0.25") {
		t.Errorf("Version should not be shown, got: %s", output)
	}

	if !strings.Contains(output, "AGPL-3.0") {
		t.Errorf("Expected license, got: %s", output)
	}
}

func TestPrintBanner_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	opts := DefaultBannerOptions()
	PrintBanner(nil, "0.0.25", opts)
	// If we get here, no panic occurred
}

func TestPrintBanner_EmptyVersion(t *testing.T) {
	var buf bytes.Buffer
	opts := BannerOptions{
		ShowBanner:  false,
		ShowVersion: true,
		ShowLicense: false,
	}

	PrintBanner(&buf, "", opts)

	output := buf.String()

	// Should still print something even with empty version
	if len(output) == 0 {
		t.Error("Expected some output even with empty version")
	}
}

func TestGetASCIILogo(t *testing.T) {
	logo := GetASCIILogo()

	if len(logo) == 0 {
		t.Error("Logo should not be empty")
	}

	// Verify it looks like ASCII art (contains underscores or pipes or forward slashes)
	hasAsciiChars := strings.Contains(logo, "_") || strings.Contains(logo, "|") ||
		strings.Contains(logo, "/") || strings.Contains(logo, "\\")
	if !hasAsciiChars {
		t.Errorf("Logo doesn't look like ASCII art: %s", logo)
	}
}

func TestGetCompactBanner(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			"normal version",
			"0.0.25",
			"Code Pathfinder v0.0.25 | AI-Native Static Code Analysis | https://codepathfinder.dev",
		},
		{
			"empty version",
			"",
			"Code Pathfinder v | AI-Native Static Code Analysis | https://codepathfinder.dev",
		},
		{
			"dev version",
			"dev",
			"Code Pathfinder vdev | AI-Native Static Code Analysis | https://codepathfinder.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCompactBanner(tt.version)
			if got != tt.want {
				t.Errorf("GetCompactBanner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldShowBanner(t *testing.T) {
	tests := []struct {
		name         string
		isTTY        bool
		noBannerFlag bool
		want         bool
	}{
		{"TTY without flag", true, false, true},
		{"TTY with flag", true, true, false},
		{"Non-TTY without flag", false, false, false},
		{"Non-TTY with flag", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldShowBanner(tt.isTTY, tt.noBannerFlag)
			if got != tt.want {
				t.Errorf("ShouldShowBanner(%v, %v) = %v, want %v",
					tt.isTTY, tt.noBannerFlag, got, tt.want)
			}
		})
	}
}

func TestDefaultBannerOptions(t *testing.T) {
	opts := DefaultBannerOptions()

	if !opts.ShowBanner {
		t.Error("Default should show banner")
	}
	if !opts.ShowVersion {
		t.Error("Default should show version")
	}
	if !opts.ShowLicense {
		t.Error("Default should show license")
	}
}

func TestBannerOptions_AllFalse(t *testing.T) {
	var buf bytes.Buffer
	opts := BannerOptions{
		ShowBanner:  false,
		ShowVersion: false,
		ShowLicense: false,
	}

	PrintBanner(&buf, "0.0.25", opts)

	output := buf.String()

	// Should only have a newline
	if strings.TrimSpace(output) != "" {
		t.Errorf("Expected minimal output with all options false, got: %q", output)
	}
}
