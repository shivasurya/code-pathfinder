package extraction

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// ===== Helper Function Tests =====

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected bool
	}{
		// Numeric types
		{"int", "int", true},
		{"int64", "int64", true},
		{"uint", "uint", true},
		{"float64", "float64", true},

		// String types
		{"string", "string", true},
		{"byte", "byte", true},
		{"rune", "rune", true},

		// Boolean
		{"bool", "bool", true},

		// Special
		{"error", "error", true},
		{"any", "any", true},

		// Not builtins
		{"User", "User", false},
		{"Config", "Config", false},
		{"custom", "custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuiltinType(tt.typeStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripPointerPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no pointer", "User", "User"},
		{"single pointer", "*User", "User"},
		{"double pointer", "**User", "User"},
		{"builtin pointer", "*string", "string"},
		{"qualified pointer", "*models.User", "models.User"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripPointerPrefix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFirstReturnType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"multi-return string error", "(string, error)", "string"},
		{"multi-return int bool", "(int, bool)", "int"},
		{"multi-return pointer", "(*User, error)", "*User"},
		{"three returns", "(string, int, error)", "string"},
		{"single type", "string", "string"},
		{"no parens", "User", "User"},
		{"empty parens", "()", ""},
		{"spaces", "(  string  ,  error  )", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFirstReturnType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ===== Main Parser Tests =====

func TestParseGoTypeString_Empty(t *testing.T) {
	result, err := ParseGoTypeString("", nil, "")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseGoTypeString_Builtins(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"int", "builtin.int"},
		{"string", "builtin.string"},
		{"error", "builtin.error"},
		{"bool", "builtin.bool"},
		{"byte", "builtin.byte"},
		{"rune", "builtin.rune"},
		{"float64", "builtin.float64"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseGoTypeString(tt.input, nil, "")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(1.0), result.Confidence)
			assert.Equal(t, "declaration", result.Source)
		})
	}
}

func TestParseGoTypeString_Pointers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single pointer", "*User", "User"},
		{"double pointer", "**Config", "Config"},
		{"pointer to builtin", "*string", "builtin.string"},
		{"pointer to qualified", "*models.User", "models.User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGoTypeString(tt.input, nil, "")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
		})
	}
}

func TestParseGoTypeString_MultiReturn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"string error", "(string, error)", "builtin.string"},
		{"int bool", "(int, bool)", "builtin.int"},
		{"pointer error", "(*User, error)", "User"},
		{"three returns", "(string, int, error)", "builtin.string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGoTypeString(tt.input, nil, "")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
		})
	}
}

func TestParseGoTypeString_Qualified(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"models.User", "models.User", "models.User"},
		{"http.Client", "http.Client", "http.Client"},
		{"handlers.Config", "handlers.Config", "handlers.Config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGoTypeString(tt.input, nil, "")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(0.9), result.Confidence)
		})
	}
}

func TestParseGoTypeString_SamePackage(t *testing.T) {
	// Create mock registry
	registry := &core.GoModuleRegistry{
		ModulePath: "github.com/example/myapp",
		DirToImport: map[string]string{
			"/project/handlers": "github.com/example/myapp/handlers",
			"/project/models":   "github.com/example/myapp/models",
		},
	}

	tests := []struct {
		name     string
		input    string
		filePath string
		expected string
	}{
		{
			"User in handlers",
			"User",
			"/project/handlers/user.go",
			"github.com/example/myapp/handlers.User",
		},
		{
			"Config in models",
			"Config",
			"/project/models/config.go",
			"github.com/example/myapp/models.Config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGoTypeString(tt.input, registry, tt.filePath)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.TypeFQN)
			assert.Equal(t, float32(0.95), result.Confidence)
		})
	}
}

func TestParseGoTypeString_Fallback(t *testing.T) {
	// No registry, unqualified type
	result, err := ParseGoTypeString("UnknownType", nil, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "UnknownType", result.TypeFQN)
	assert.Equal(t, float32(0.5), result.Confidence) // Lower confidence
}

func TestParseGoTypeString_Whitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  int  ", "builtin.int"},
		{" *User ", "User"},
		{" ( string , error ) ", "builtin.string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseGoTypeString(tt.input, nil, "")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.TypeFQN)
		})
	}
}
