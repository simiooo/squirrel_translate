package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModelParametersPersistWithZeroAndLegacyDefaults(t *testing.T) {
	s, _ := New(t.TempDir())
	path := filepath.Join(s.Dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"model":"legacy-model","reasoning_effort":"low"}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := s.LoadConfig()
	if err != nil || c.TokenLimitField != "max_tokens" || c.Temperature != nil || c.Model != "legacy-model" {
		t.Fatal("old settings did not inherit safe defaults", err)
	}
	zero, limit := 0.0, 4096
	seed := 0
	c.Temperature, c.PresencePenalty = &zero, &zero
	c.MaxOutputTokens, c.Seed = &limit, &seed
	c.TokenLimitField, c.ReasoningEffort = "max_completion_tokens", "xhigh"
	if err = s.SaveConfig(c); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var stored map[string]any
	_ = json.Unmarshal(data, &stored)
	if _, ok := stored["top_p"]; ok || stored["temperature"] != 0.0 {
		t.Fatal("persistent format conflated unset and zero", stored)
	}
	reloaded, err := s.LoadConfig()
	if err != nil || reloaded.Temperature == nil || *reloaded.Temperature != 0 || reloaded.TopP != nil || *reloaded.Seed != 0 || *reloaded.MaxOutputTokens != limit || reloaded.ReasoningEffort != "xhigh" || reloaded.TokenLimitField != "max_completion_tokens" {
		t.Fatal("settings lost optional parameters across reload", err)
	}
}
