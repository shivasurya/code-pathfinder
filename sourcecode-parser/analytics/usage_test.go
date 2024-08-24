package analytics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name           string
		disableMetrics bool
		wantMetrics    bool
	}{
		{"Metrics enabled", false, true},
		{"Metrics disabled", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.disableMetrics)
			assert.Equal(t, tt.wantMetrics, enableMetrics)
		})
	}
}

func TestCreateEnvFile(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	envFile := filepath.Join(homeDir, ".codepathfinder", ".env")

	// Clean up before test
	os.RemoveAll(filepath.Dir(envFile))

	createEnvFile()

	assert.FileExists(t, envFile)

	env, err := godotenv.Read(envFile)
	assert.NoError(t, err)
	assert.Contains(t, env, "uuid")
	assert.Len(t, env["uuid"], 36) // UUID length

	// Clean up after test
	os.RemoveAll(filepath.Dir(envFile))
}

func TestLoadEnvFile(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	envFile := filepath.Join(homeDir, ".codepathfinder", ".env")

	// Clean up before test
	os.RemoveAll(filepath.Dir(envFile))

	LoadEnvFile()

	// read env file and check if uuid is set
	env, err := godotenv.Read(envFile)
	assert.NoError(t, err)

	assert.Equal(t, env["uuid"], os.Getenv("uuid"))

	// Clean up after test
	os.RemoveAll(filepath.Dir(envFile))
}

func TestReportEvent(t *testing.T) {
	tests := []struct {
		name          string
		enableMetrics bool
		publicKey     string
		event         string
	}{
		{"Metrics disabled", false, "test-key", "test-event"},
		{"Metrics enabled, no public key", true, "", "test-event"},
		{"Metrics enabled, with public key", true, "test-key", "test-event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.enableMetrics)
			PublicKey = tt.publicKey
			ReportEvent(tt.event)
			// Since ReportEvent doesn't return anything, we're just ensuring it doesn't panic
		})
	}
}
