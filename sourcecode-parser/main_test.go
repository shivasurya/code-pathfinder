package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Run the tests
	os.Exit(m.Run())
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteErr error
		expectedOutput string
		expectedExit   int
	}{
		{
			name:           "Successful execution",
			mockExecuteErr: nil,
			expectedOutput: "Code Pathfinder is designed for identifying vulnerabilities in source code.\n\nUsage:\n  pathfinder [command]\n\nAvailable Commands:\n  ci          Scan a project for vulnerabilities with ruleset in ci mode\n  completion  Generate the autocompletion script for the specified shell\n  help        Help about any command\n  query       Execute queries on the source code\n  scan        Scan a project for vulnerabilities with ruleset\n  version     Print the version and commit information\n\nFlags:\n      --disable-metrics   Disable metrics collection\n  -h, --help              help for pathfinder\n      --verbose           Verbose output\n\nUse \"pathfinder [command] --help\" for more information about a command.\n",
			expectedExit:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Mock os.Exit
			oldOsExit := osExit
			var exitCode int
			osExit = func(code int) {
				exitCode = code
			}
			defer func() { osExit = oldOsExit }()

			// Call main
			main()

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			buf.ReadFrom(r)

			// Assert
			assert.Equal(t, tt.expectedOutput, buf.String())
			if tt.mockExecuteErr != nil {
				assert.Equal(t, tt.expectedExit, exitCode)
			}
		})
	}
}

// Mock for os.Exit
var osExit = os.Exit
