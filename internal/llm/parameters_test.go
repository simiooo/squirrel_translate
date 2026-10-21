package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simiooo/squirrel_translate/internal/core"
)

func parameterValue[T any](v T) *T { return &v }

func TestConfiguredParametersReachStreamingWireWithoutLosingZero(t *testing.T) {
	keys := []string{"temperature", "top_p", "presence_penalty", "frequency_penalty", "seed", "reasoning_effort", "max_tokens", "max_completion_tokens"}
	for _, tc := range []struct {
		name  string
		setup func(*core.Config)
		want  map[string]any
		word  bool
	}{
		{name: "provider defaults", setup: func(*core.Config) {}, want: map[string]any{}},
		{name: "explicit zero", setup: func(c *core.Config) {
			c.Temperature, c.TopP = parameterValue(0.0), parameterValue(0.0)
			c.PresencePenalty, c.FrequencyPenalty = parameterValue(0.0), parameterValue(0.0)
			c.Seed = parameterValue(0)
			c.ReasoningEffort = "none"
		}, want: map[string]any{"temperature": 0.0, "top_p": 0.0, "presence_penalty": 0.0, "frequency_penalty": 0.0, "seed": 0.0, "reasoning_effort": "none"}},
		{name: "legacy limit and sampling", setup: func(c *core.Config) {
			c.Temperature, c.TopP = parameterValue(0.2), parameterValue(0.95)
			c.PresencePenalty, c.FrequencyPenalty = parameterValue(-0.25), parameterValue(0.5)
			c.MaxOutputTokens, c.Seed = parameterValue(4096), parameterValue(42)
			c.ReasoningEffort = "high"
		}, want: map[string]any{"temperature": 0.2, "top_p": 0.95, "presence_penalty": -0.25, "frequency_penalty": 0.5, "max_tokens": 4096.0, "seed": 42.0, "reasoning_effort": "high"}},
		{name: "completion limit", setup: func(c *core.Config) {
			c.MaxOutputTokens = parameterValue(8192)
			c.TokenLimitField = "max_completion_tokens"
			c.ReasoningEffort = "max"
		}, want: map[string]any{"max_completion_tokens": 8192.0, "reasoning_effort": "max"}},
		{name: "dictionary keeps response format", setup: func(c *core.Config) {
			c.Temperature = parameterValue(0.0)
			c.StructuredMode = "json_object"
		}, want: map[string]any{"temperature": 0.0}, word: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var payload map[string]any
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				data, err := io.ReadAll(r.Body)
				if err != nil || json.Unmarshal(data, &payload) != nil {
					t.Error("invalid request body", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.ContentLength != int64(len(data)) || r.Header.Get("Authorization") != "Bearer test-only" {
					t.Error("parameter adapter changed HTTP framing or authentication")
				}
				text := "你好"
				if tc.word {
					text = `{"senses":[{"part_of_speech":"noun","meaning":"银行","in_context":false}],"source_language":"en","lemma":"bank"}`
				}
				w.Header().Set("Content-Type", "text/event-stream")
				sse(w, map[string]string{"content": text}, "")
				sse(w, map[string]string{}, "stop")
				fmt.Fprint(w, "data: [DONE]\n\n")
			}))
			defer server.Close()
			cfg := core.DefaultConfig()
			cfg.Endpoint = server.URL
			tc.setup(&cfg)
			mode := "sentence"
			if tc.word {
				mode = "word"
			}
			if err := Stream(context.Background(), Request{Selection: core.Selection{Text: "bank"}, Mode: mode, Target: "简体中文", Config: cfg, Key: "test-only"}, func(Event) {}); err != nil {
				t.Fatal(err)
			}
			if requests != 1 || payload["stream"] != true {
				t.Fatalf("expected one streamed request, got %d: %v", requests, payload)
			}
			for _, key := range keys {
				got, present := payload[key]
				want, configured := tc.want[key]
				if present != configured || got != want {
					t.Errorf("%s: got %v (present %v), want %v (configured %v)", key, got, present, want, configured)
				}
			}
			if tc.word && payload["response_format"].(map[string]any)["type"] != "json_object" {
				t.Fatal("sampling adapter lost response format")
			}
		})
	}
}
