package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdlibKnownReturnTypes(t *testing.T) {
	t.Run("module-level function return types", func(t *testing.T) {
		rt := GetKnownStdlibReturnType("sqlite3", "connect")
		assert.Equal(t, "sqlite3.Connection", rt)

		rt = GetKnownStdlibReturnType("re", "compile")
		assert.Equal(t, "re.Pattern", rt)

		// Unknown function
		rt = GetKnownStdlibReturnType("sqlite3", "unknown_func")
		assert.Equal(t, "", rt)

		// Unknown module
		rt = GetKnownStdlibReturnType("unknown_module", "func")
		assert.Equal(t, "", rt)
	})

	t.Run("class method return types", func(t *testing.T) {
		rt := GetKnownStdlibMethodReturnType("sqlite3", "Connection", "cursor")
		assert.Equal(t, "sqlite3.Cursor", rt)

		rt = GetKnownStdlibMethodReturnType("sqlite3", "Cursor", "fetchone")
		assert.Equal(t, "builtins.tuple", rt)

		rt = GetKnownStdlibMethodReturnType("sqlite3", "Cursor", "fetchall")
		assert.Equal(t, "builtins.list", rt)

		// Unknown method
		rt = GetKnownStdlibMethodReturnType("sqlite3", "Connection", "unknown")
		assert.Equal(t, "", rt)
	})

	t.Run("has known module", func(t *testing.T) {
		assert.True(t, HasKnownStdlibTypes("sqlite3"))
		assert.True(t, HasKnownStdlibTypes("re"))
		assert.True(t, HasKnownStdlibTypes("hashlib"))
		assert.True(t, HasKnownStdlibTypes("io"))
		assert.False(t, HasKnownStdlibTypes("unknown"))
	})
}
