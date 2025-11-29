package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	t.Run("New location with valid values", func(t *testing.T) {
		loc := Location{
			File: "test.go",
			Line: 42,
		}
		assert.Equal(t, "test.go", loc.File)
		assert.Equal(t, 42, loc.Line)
	})

	t.Run("New location with empty file", func(t *testing.T) {
		loc := Location{
			File: "",
			Line: 1,
		}
		assert.Empty(t, loc.File)
		assert.Equal(t, 1, loc.Line)
	})

	t.Run("New location with zero line", func(t *testing.T) {
		loc := Location{
			File: "main.go",
			Line: 0,
		}
		assert.Equal(t, "main.go", loc.File)
		assert.Zero(t, loc.Line)
	})

	t.Run("New location with negative line", func(t *testing.T) {
		loc := Location{
			File: "src.go",
			Line: -1,
		}
		assert.Equal(t, "src.go", loc.File)
		assert.Equal(t, -1, loc.Line)
	})

	t.Run("New location with file path", func(t *testing.T) {
		loc := Location{
			File: "/path/to/file.go",
			Line: 100,
		}
		assert.Equal(t, "/path/to/file.go", loc.File)
		assert.Equal(t, 100, loc.Line)
	})
}
