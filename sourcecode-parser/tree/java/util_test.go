package java

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractVisibilityModifier(t *testing.T) {
	tests := []struct {
		name      string
		modifiers string
		expected  string
	}{
		{
			name:      "Public modifier",
			modifiers: "public static void",
			expected:  "public",
		},
		{
			name:      "Private modifier",
			modifiers: "private final int",
			expected:  "private",
		},
		{
			name:      "Protected modifier",
			modifiers: "protected abstract class",
			expected:  "protected",
		},
		{
			name:      "No visibility modifier",
			modifiers: "static final int",
			expected:  "",
		},
		{
			name:      "Empty string",
			modifiers: "",
			expected:  "",
		},
		{
			name:      "Multiple modifiers with public",
			modifiers: "public static final synchronized",
			expected:  "public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVisibilityModifier(tt.modifiers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsJavaSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "Java file",
			filename: "Main.java",
			expected: true,
		},
		{
			name:     "Java file with path",
			filename: "/path/to/Main.java",
			expected: true,
		},
		{
			name:     "Non-Java file",
			filename: "Main.cpp",
			expected: false,
		},
		{
			name:     "File without extension",
			filename: "README",
			expected: false,
		},
		{
			name:     "Java file with uppercase extension",
			filename: "Test.JAVA",
			expected: false, // This will fail because filepath.Ext is case-sensitive
		},
		{
			name:     "File with .java in the middle",
			filename: "Main.java.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJavaSourceFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
