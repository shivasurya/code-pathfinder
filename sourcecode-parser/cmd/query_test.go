package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryCommand(t *testing.T) {
	cmd := queryCmd

	assert.NotNil(t, cmd)
	assert.Equal(t, "query", cmd.Use)
	assert.Equal(t, "Query code using Python DSL rules", cmd.Short)

	// Test execution returns error when required flags are missing
	err := cmd.RunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}
