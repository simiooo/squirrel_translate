package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
	"github.com/simiooo/squirrel_translate/internal/core"
)

func applyParameters(r *openai.ChatCompletionRequest, c core.Config) {
	if c.ReasoningEffort != "default" {
		r.ReasoningEffort = c.ReasoningEffort
	}
	if c.MaxOutputTokens != nil {
		if c.TokenLimitField == "max_completion_tokens" {
			r.MaxCompletionTokens = *c.MaxOutputTokens
		} else {
			r.MaxTokens = *c.MaxOutputTokens
		}
	}
	r.Seed = c.Seed
	// Retain SDK compatibility validation. The transport below preserves the
	// original optional values, including zero, in the serialized request.
	if c.Temperature != nil {
		r.Temperature = float32(*c.Temperature)
	}
	if c.TopP != nil {
		r.TopP = float32(*c.TopP)
	}
	if c.PresencePenalty != nil {
		r.PresencePenalty = float32(*c.PresencePenalty)
	}
	if c.FrequencyPenalty != nil {
		r.FrequencyPenalty = float32(*c.FrequencyPenalty)
	}
}

// go-openai v1.43 serializes scalar sampling fields with omitempty. This small
// HTTP adapter restores only explicitly configured numeric fields, preserving
// both zero and float64 precision without replacing the SDK's SSE handling.
type parameterTransport struct {
	next   http.RoundTripper
	values map[string]float64
}

func newParameterTransport(next http.RoundTripper, c core.Config) *parameterTransport {
	values := make(map[string]float64)
	for key, value := range map[string]*float64{
		"temperature": c.Temperature, "top_p": c.TopP,
		"presence_penalty": c.PresencePenalty, "frequency_penalty": c.FrequencyPenalty,
	} {
		if value != nil {
			values[key] = *value
		}
	}
	return &parameterTransport{next: next, values: values}
}

func (t *parameterTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if len(t.values) == 0 {
		return t.next.RoundTrip(r)
	}
	data, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return nil, err
	}
	var body map[string]json.RawMessage
	if err = json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	for key, value := range t.values {
		encoded, e := json.Marshal(value)
		if e != nil {
			return nil, e
		}
		body[key] = encoded
	}
	data, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}
	clone := r.Clone(r.Context())
	clone.Body = io.NopCloser(bytes.NewReader(data))
	clone.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil }
	clone.ContentLength = int64(len(data))
	clone.Header.Del("Content-Length")
	return t.next.RoundTrip(clone)
}
