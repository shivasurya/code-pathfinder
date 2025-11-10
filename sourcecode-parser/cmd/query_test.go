package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryCommandStub(t *testing.T) {
	cmd := queryCmd

	assert.NotNil(t, cmd)
	assert.Equal(t, "query", cmd.Use)

	// Test execution returns error for unimplemented command
	err := cmd.RunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}
