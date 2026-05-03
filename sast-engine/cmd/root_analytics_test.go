package cmd

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// --- updateCheckSkipReason tests --------------------------------------------

func TestUpdateCheckSkipReason_FlagReturnsFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", true, "")
	reason, skip := updateCheckSkipReason(cmd)
	assert.True(t, skip)
	assert.Equal(t, "flag", reason)
}

func TestUpdateCheckSkipReason_EnvReturnsEnv(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", false, "")
	reason, skip := updateCheckSkipReason(cmd)
	assert.True(t, skip)
	assert.Equal(t, "env", reason)
}

func TestUpdateCheckSkipReason_NoSkipReturnsEmptyReason(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", false, "")
	reason, skip := updateCheckSkipReason(cmd)
	assert.False(t, skip)
	assert.Empty(t, reason)
}

func TestUpdateCheckSkipReason_FlagTakesPriorityOverEnv(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", true, "")
	reason, skip := updateCheckSkipReason(cmd)
	assert.True(t, skip)
	assert.Equal(t, "flag", reason, "flag should be checked before env var")
}

// --- IsDisabled integration -------------------------------------------------

func TestAnalyticsIsDisabled_GatesRoot(t *testing.T) {
	analytics.Init(true)
	assert.True(t, analytics.IsDisabled())
	analytics.Init(false)
	assert.False(t, analytics.IsDisabled())
	t.Cleanup(func() { analytics.Init(false) })
}
