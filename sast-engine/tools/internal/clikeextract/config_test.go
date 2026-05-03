package clikeextract

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string // empty = expect no error
	}{
		{
			name:    "missing target",
			cfg:     Config{Language: core.LanguageC, OutputDir: "/tmp/out"},
			wantErr: "Target is required",
		},
		{
			name:    "unsupported target",
			cfg:     Config{Target: "freebsd", Language: core.LanguageC, OutputDir: "/tmp/out"},
			wantErr: "unsupported Target",
		},
		{
			name:    "missing language",
			cfg:     Config{Target: core.PlatformLinux, OutputDir: "/tmp/out"},
			wantErr: "Language is required",
		},
		{
			name:    "unsupported language",
			cfg:     Config{Target: core.PlatformLinux, Language: "rust", OutputDir: "/tmp/out"},
			wantErr: "unsupported Language",
		},
		{
			name:    "missing output dir",
			cfg:     Config{Target: core.PlatformLinux, Language: core.LanguageC},
			wantErr: "OutputDir is required",
		},
		{
			name: "valid linux c",
			cfg:  Config{Target: core.PlatformLinux, Language: core.LanguageC, OutputDir: "/tmp/out"},
		},
		{
			name: "valid linux cpp",
			cfg:  Config{Target: core.PlatformLinux, Language: core.LanguageCpp, OutputDir: "/tmp/out"},
		},
		{
			name: "valid windows c",
			cfg:  Config{Target: core.PlatformWindows, Language: core.LanguageC, OutputDir: "/tmp/out"},
		},
		{
			name: "valid darwin cpp",
			cfg:  Config{Target: core.PlatformDarwin, Language: core.LanguageCpp, OutputDir: "/tmp/out"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConfig_EffectiveBaseURL(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, DefaultBaseURL, cfg.effectiveBaseURL())

	cfg.BaseURL = "file:///tmp/registries"
	assert.Equal(t, "file:///tmp/registries", cfg.effectiveBaseURL())
}

func TestVersionConstants(t *testing.T) {
	// Lock the visible version surface — bumping these is a deliberate change,
	// the test is the trip-wire that forces the bump to be intentional.
	assert.Equal(t, "1.0.0", GeneratorVersion)
	assert.Equal(t, "1.0.0", SchemaVersion)
	assert.Equal(t, "v1", RegistryVersion)
	assert.Equal(t, "https://assets.codepathfinder.dev/registries", DefaultBaseURL)
}
