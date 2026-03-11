package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 0.5, cfg.DefaultMinConfidence)
	assert.Equal(t, 0.7, cfg.FQNBridgeConfidence)
	assert.Equal(t, 0.7, cfg.LocalScopeConfidence)
	assert.Equal(t, 0.8, cfg.GlobalScopeConfidence)
	assert.Equal(t, 30*time.Second, cfg.ExecutionTimeout)
	assert.Equal(t, 0.8, cfg.ConfidenceLevels.HighThreshold)
	assert.Equal(t, 0.5, cfg.ConfidenceLevels.MediumThreshold)
}

func TestConfig_NilSafe(t *testing.T) {
	var cfg *QueryTypeConfig // nil

	assert.Equal(t, 0.5, cfg.getDefaultMinConfidence())
	assert.Equal(t, 0.7, cfg.getFQNBridgeConfidence())
	assert.Equal(t, 0.7, cfg.getLocalScopeConfidence())
	assert.Equal(t, 0.8, cfg.getGlobalScopeConfidence())
	assert.Equal(t, 30*time.Second, cfg.getExecutionTimeout())
	assert.Equal(t, 0.8, cfg.getHighThreshold())
	assert.Equal(t, 0.5, cfg.getMediumThreshold())
}

func TestConfig_ZeroValueSafe(t *testing.T) {
	cfg := &QueryTypeConfig{} // all zero values

	assert.Equal(t, 0.5, cfg.getDefaultMinConfidence())
	assert.Equal(t, 0.7, cfg.getFQNBridgeConfidence())
	assert.Equal(t, 0.7, cfg.getLocalScopeConfidence())
	assert.Equal(t, 0.8, cfg.getGlobalScopeConfidence())
	assert.Equal(t, 30*time.Second, cfg.getExecutionTimeout())
	assert.Equal(t, 0.8, cfg.getHighThreshold())
	assert.Equal(t, 0.5, cfg.getMediumThreshold())
}

func TestConfig_CustomThresholds(t *testing.T) {
	cfg := &QueryTypeConfig{
		DefaultMinConfidence:  0.3,
		FQNBridgeConfidence:   0.6,
		LocalScopeConfidence:  0.5,
		GlobalScopeConfidence: 0.9,
		ExecutionTimeout:      60 * time.Second,
		ConfidenceLevels: ConfidenceLevelConfig{
			HighThreshold:   0.9,
			MediumThreshold: 0.6,
		},
	}

	assert.Equal(t, 0.3, cfg.getDefaultMinConfidence())
	assert.Equal(t, 0.6, cfg.getFQNBridgeConfidence())
	assert.Equal(t, 0.5, cfg.getLocalScopeConfidence())
	assert.Equal(t, 0.9, cfg.getGlobalScopeConfidence())
	assert.Equal(t, 60*time.Second, cfg.getExecutionTimeout())
	assert.Equal(t, 0.9, cfg.getHighThreshold())
	assert.Equal(t, 0.6, cfg.getMediumThreshold())
}
