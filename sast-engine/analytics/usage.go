package analytics

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/posthog/posthog-go"
)

const (
	// Scan command events - production command tracking.
	ScanStarted   = "scan_started"
	ScanCompleted = "scan_completed"
	ScanFailed    = "scan_failed"

	// CI command events - production command tracking.
	CIStarted   = "ci_started"
	CICompleted = "ci_completed"
	CIFailed    = "ci_failed"

	// MCP Server events - production command tracking.
	MCPServerStarted    = "mcp_server_started"
	MCPServerStopped    = "mcp_server_stopped"
	MCPToolCall         = "mcp_tool_call"
	MCPIndexingStarted  = "mcp_indexing_started"
	MCPIndexingComplete = "mcp_indexing_complete"
	MCPIndexingFailed   = "mcp_indexing_failed"
	MCPClientConnected  = "mcp_client_connected"
)

var (
	PublicKey     string
	enableMetrics bool
)

func Init(disableMetrics bool) {
	enableMetrics = !disableMetrics
}

func createEnvFile() {
	homeDir, err := os.UserHomeDir()
	envFile := filepath.Join(homeDir, ".codepathfinder", ".env")
	if err != nil {
		fmt.Println("Error getting user home directory:", err)
		return
	}
	// create .env file
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// create directory
		if err := os.MkdirAll(filepath.Dir(envFile), os.ModePerm); err != nil {
			fmt.Println("Error creating directory:", err)
			return
		}
		env := map[string]string{
			"uuid": uuid.New().String(),
		}
		err = godotenv.Write(env, envFile)
		if err != nil {
			fmt.Println("Error writing to .env file:", err)
		}
	}
}

func LoadEnvFile() {
	createEnvFile()
	envFile := filepath.Join(os.Getenv("HOME"), ".codepathfinder", ".env")
	err := godotenv.Load(envFile)
	if err != nil {
		return
	}
}

func ReportEvent(event string) {
	ReportEventWithProperties(event, nil)
}

// ReportEventWithProperties sends an event with additional properties.
// Properties should not contain any PII (no file paths, code, user info).
func ReportEventWithProperties(event string, properties map[string]interface{}) {
	if enableMetrics && PublicKey != "" {
		// Enable GeoIP resolution by setting DisableGeoIP to false (pointer to bool)
		disableGeoIP := false
		client, err := posthog.NewWithConfig(
			PublicKey,
			posthog.Config{
				Endpoint:     "https://us.i.posthog.com",
				DisableGeoIP: &disableGeoIP, // Enable GeoIP resolution for location analytics
			},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		capture := posthog.Capture{
			DistinctId: os.Getenv("uuid"),
			Event:      event,
		}

		// Create properties with automatic platform metadata
		captureProperties := posthog.NewProperties()

		// Add runtime metadata automatically
		captureProperties.Set("os", runtime.GOOS)
		captureProperties.Set("arch", runtime.GOARCH)
		captureProperties.Set("go_version", runtime.Version())

		// Merge user-provided properties
		if properties != nil {
			for k, v := range properties {
				captureProperties.Set(k, v)
			}
		}

		capture.Properties = captureProperties

		err = client.Enqueue(capture)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
