package dsl

import "time"

// QueryTypeConfig centralizes all configurable thresholds and timeouts
// for the QueryType execution pipeline. Zero-value fields use defaults
// via the getOrDefault helper methods.
type QueryTypeConfig struct {
	DefaultMinConfidence  float64               // default 0.5 — minimum type confidence for matching
	FQNBridgeConfidence   float64               // default 0.7 — confidence for FQN-bridge matches
	LocalScopeConfidence  float64               // default 0.7 — confidence for local-scope dataflow
	GlobalScopeConfidence float64               // default 0.8 — confidence for global-scope dataflow
	ExecutionTimeout      time.Duration         // default 30s — Python rule execution timeout
	ConfidenceLevels      ConfidenceLevelConfig // thresholds for human-readable levels
}

// ConfidenceLevelConfig defines thresholds for confidence level labels.
type ConfidenceLevelConfig struct {
	HighThreshold   float64 // default 0.8 — >= this → "high"
	MediumThreshold float64 // default 0.5 — >= this → "medium", below → "low"
}

// DefaultConfig returns a config with production defaults matching
// the previously hardcoded values.
func DefaultConfig() *QueryTypeConfig {
	return &QueryTypeConfig{
		DefaultMinConfidence:  0.5,
		FQNBridgeConfidence:   0.7,
		LocalScopeConfidence:  0.7,
		GlobalScopeConfidence: 0.8,
		ExecutionTimeout:      30 * time.Second,
		ConfidenceLevels: ConfidenceLevelConfig{
			HighThreshold:   0.8,
			MediumThreshold: 0.5,
		},
	}
}

// getDefaultMinConfidence returns the configured or default min confidence.
func (c *QueryTypeConfig) getDefaultMinConfidence() float64 {
	if c == nil || c.DefaultMinConfidence <= 0 {
		return 0.5
	}
	return c.DefaultMinConfidence
}

// getFQNBridgeConfidence returns the configured or default FQN bridge confidence.
func (c *QueryTypeConfig) getFQNBridgeConfidence() float64 {
	if c == nil || c.FQNBridgeConfidence <= 0 {
		return 0.7
	}
	return c.FQNBridgeConfidence
}

// getLocalScopeConfidence returns the configured or default local scope confidence.
func (c *QueryTypeConfig) getLocalScopeConfidence() float64 {
	if c == nil || c.LocalScopeConfidence <= 0 {
		return 0.7
	}
	return c.LocalScopeConfidence
}

// getGlobalScopeConfidence returns the configured or default global scope confidence.
func (c *QueryTypeConfig) getGlobalScopeConfidence() float64 {
	if c == nil || c.GlobalScopeConfidence <= 0 {
		return 0.8
	}
	return c.GlobalScopeConfidence
}

// getExecutionTimeout returns the configured or default execution timeout.
func (c *QueryTypeConfig) getExecutionTimeout() time.Duration {
	if c == nil || c.ExecutionTimeout <= 0 {
		return 30 * time.Second
	}
	return c.ExecutionTimeout
}

// getHighThreshold returns the configured or default high confidence threshold.
func (c *QueryTypeConfig) getHighThreshold() float64 {
	if c == nil || c.ConfidenceLevels.HighThreshold <= 0 {
		return 0.8
	}
	return c.ConfidenceLevels.HighThreshold
}

// getMediumThreshold returns the configured or default medium confidence threshold.
func (c *QueryTypeConfig) getMediumThreshold() float64 {
	if c == nil || c.ConfidenceLevels.MediumThreshold <= 0 {
		return 0.5
	}
	return c.ConfidenceLevels.MediumThreshold
}
