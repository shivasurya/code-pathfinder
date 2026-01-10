package analytics

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/posthog/posthog-go"
)

const (
	VersionCommand       = "executed_version_command"
	QueryCommandJSON     = "executed_query_command_json_mode"
	ErrorProcessingQuery = "error_processing_query"
	QueryCommandStdin    = "executed_query_command_stdin_mode"

	// MCP Server events.
	MCPServerStarted   = "mcp_server_started"
	MCPServerStopped   = "mcp_server_stopped"
	MCPToolCall        = "mcp_tool_call"
	MCPIndexingStarted = "mcp_indexing_started"
	MCPIndexingComplete = "mcp_indexing_complete"
	MCPIndexingFailed  = "mcp_indexing_failed"
	MCPClientConnected = "mcp_client_connected"
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
		client, err := posthog.NewWithConfig(
			PublicKey,
			posthog.Config{
				Endpoint: "https://us.i.posthog.com",
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

		if properties != nil {
			capture.Properties = posthog.NewProperties()
			for k, v := range properties {
				capture.Properties.Set(k, v)
			}
		}

		err = client.Enqueue(capture)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
