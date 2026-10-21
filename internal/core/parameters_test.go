package core

import (
	"math"
	"testing"
)

func parameterValue[T any](v T) *T { return &v }

func TestOptionalModelParametersValidateZeroAndBoundaries(t *testing.T) {
	c := DefaultConfig()
	c.Temperature, c.TopP = parameterValue(0.0), parameterValue(0.0)
	c.PresencePenalty, c.FrequencyPenalty = parameterValue(-2.0), parameterValue(2.0)
	c.MaxOutputTokens, c.Seed = parameterValue(1), parameterValue(0)
	for _, effort := range ReasoningEfforts {
		c.ReasoningEffort = effort
		if err := c.Check(); err != nil {
			t.Fatalf("valid parameters with effort %s: %v", effort, err)
		}
	}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Temperature = parameterValue(-0.1) },
		func(c *Config) { c.Temperature = parameterValue(2.1) },
		func(c *Config) { c.Temperature = parameterValue(math.NaN()) },
		func(c *Config) { c.TopP = parameterValue(math.Inf(1)) },
		func(c *Config) { c.TopP = parameterValue(1.1) },
		func(c *Config) { c.PresencePenalty = parameterValue(-2.1) },
		func(c *Config) { c.FrequencyPenalty = parameterValue(2.1) },
		func(c *Config) { c.MaxOutputTokens = parameterValue(0) },
		func(c *Config) { c.TokenLimitField = "unknown" },
		func(c *Config) { c.ReasoningEffort = "unknown" },
	} {
		bad := DefaultConfig()
		mutate(&bad)
		if err := bad.Check(); err == nil {
			t.Fatalf("accepted invalid model parameters: %+v", bad)
		}
	}
}
