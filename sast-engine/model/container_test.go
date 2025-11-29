package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_IsSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		file     File
		expected bool
	}{
		{
			name:     "Java source file",
			file:     File{File: "Test.java"},
			expected: true,
		},
		{
			name:     "Non-source file",
			file:     File{File: "Test.txt"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.file.IsSourceFile()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJarFile_GetManifestEntryAttributes(t *testing.T) {
	jarFile := &JarFile{
		ManifestEntryAttributes: map[string]map[string]string{
			"entry1": {
				"key1": "value1",
				"key2": "value2",
			},
		},
	}

	t.Run("Existing entry and key", func(t *testing.T) {
		value, exists := jarFile.GetManifestEntryAttributes("entry1", "key1")
		assert.True(t, exists)
		assert.Equal(t, "value1", value)
	})

	t.Run("Existing entry, non-existing key", func(t *testing.T) {
		value, exists := jarFile.GetManifestEntryAttributes("entry1", "nonexistent")
		assert.False(t, exists)
		assert.Equal(t, "", value)
	})

	t.Run("Non-existing entry", func(t *testing.T) {
		value, exists := jarFile.GetManifestEntryAttributes("nonexistent", "key1")
		assert.False(t, exists)
		assert.Equal(t, "", value)
	})
}

func TestJarFile_GetManifestMainAttributes(t *testing.T) {
	jarFile := &JarFile{
		ManifestMainAttributes: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	t.Run("Existing key", func(t *testing.T) {
		value, exists := jarFile.GetManifestMainAttributes("key1")
		assert.True(t, exists)
		assert.Equal(t, "value1", value)
	})

	t.Run("Non-existing key", func(t *testing.T) {
		value, exists := jarFile.GetManifestMainAttributes("nonexistent")
		assert.False(t, exists)
		assert.Equal(t, "", value)
	})
}

func TestCompilationUnit_HasName(t *testing.T) {
	cu := &CompilationUnit{Name: "TestClass"}

	t.Run("Matching name", func(t *testing.T) {
		assert.True(t, cu.HasName("TestClass"))
	})

	t.Run("Non-matching name", func(t *testing.T) {
		assert.False(t, cu.HasName("OtherClass"))
	})

	t.Run("Empty name", func(t *testing.T) {
		assert.False(t, cu.HasName(""))
	})
}

func TestFile_IsJavaSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		file     File
		expected bool
	}{
		{
			name:     "Java file",
			file:     File{File: "Test.java"},
			expected: true,
		},
		{
			name:     "Non-Java file",
			file:     File{File: "Test.kt"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.file.IsJavaSourceFile()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFile_IsKotlinSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		file     File
		expected bool
	}{
		{
			name:     "Non-Kotlin file",
			file:     File{File: "Test.java"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.file.IsKotlinSourceFile()
			assert.Equal(t, tt.expected, result)
		})
	}
}
