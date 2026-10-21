package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/simiooo/squirrel_translate/internal/core"
)

func TestDictionaryStreamsCompletedSenseBeforeWholeDocument(t *testing.T) {
	r, w := io.Pipe()
	first := make(chan core.Dictionary, 2)
	result := make(chan error, 1)
	var final core.Dictionary
	go func() {
		result <- decodeDictionary(r, "banks", func(e Event) {
			if e.Dictionary != nil && len(e.Dictionary.Senses) == 1 {
				final = *e.Dictionary
				select {
				case first <- *e.Dictionary:
				default:
				}
			}
		})
	}()
	_, e := io.WriteString(w, `{"senses":[{"part_of_speech":"n.","meaning":"银行","in_context":true}`)
	if e != nil {
		t.Fatal(e)
	}
	select {
	case partial := <-first:
		if partial.Word != "banks" || partial.SourceLanguage != "" || partial.Lemma != "" || partial.Senses[0].Meaning != "银行" {
			t.Fatalf("first sense waited for or invented metadata: %+v", partial)
		}
	case <-time.After(time.Second):
		t.Fatal("sense was not emitted before document completed")
	}
	_, _ = io.WriteString(w, `],"source_language":"en","lemma":"bank","pronunciation":"/bæŋks/"}`)
	_ = w.Close()
	if e = <-result; e != nil {
		t.Fatal(e)
	}
	if final.Word != "banks" || final.Lemma != "bank" || final.SourceLanguage != "en" || final.Pronunciation != "/bæŋks/" {
		t.Fatalf("compact result lost vocabulary metadata: %+v", final)
	}
}
func TestDictionaryRejectsMalformedOrIncompleteOutput(t *testing.T) {
	for _, s := range []string{"```json\n{}\n```", `{"senses":[]}`, `{"lemma":"x","source_language":"en","senses":[{"part_of_speech":"noun","meaning":"a"}]} trailing`,
		`{"senses":[{"part_of_speech":"n.","meaning":"a","examples":[]}],"source_language":"en","lemma":"x"}`,
		`{"senses":[{"part_of_speech":"n.","meaning":"a"}],"source_language":"en","lemma":"x","lemma":"y"}`,
		`{"senses":[],"senses":[{"part_of_speech":"n.","meaning":"a"}],"source_language":"en","lemma":"x"}`,
		`{"senses":[{"part_of_speech":"n.","meaning":"` + strings.Repeat("字", 81) + `"}],"source_language":"en","lemma":"x"}`,
		`{"senses":[` + strings.Repeat(`{"part_of_speech":"n.","meaning":"a"},`, 3) + `{"part_of_speech":"n.","meaning":"b"}],"source_language":"en","lemma":"x"}`,
	} {
		if e := decodeDictionary(strings.NewReader(s), "x", func(Event) {}); e == nil {
			t.Errorf("accepted %s", s)
		}
	}
}

func TestCompactWordContractReachesEveryResponseFormat(t *testing.T) {
	for _, mode := range []string{"prompt", "json_object", "json_schema"} {
		t.Run(mode, func(t *testing.T) {
			var payload map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				w.Header().Set("Content-Type", "text/event-stream")
				sse(w, map[string]string{"content": `{"senses":[{"part_of_speech":"n.","meaning":"银行","in_context":true}],"source_language":"en","lemma":"bank","pronunciation":"/bæŋks/"}`}, "")
				sse(w, map[string]string{}, "stop")
				fmt.Fprint(w, "data: [DONE]\n\n")
			}))
			defer server.Close()
			cfg := core.DefaultConfig()
			cfg.Endpoint, cfg.StructuredMode = server.URL, mode
			definitions := make(chan core.Dictionary, 4)
			if err := Stream(context.Background(), Request{Selection: core.Selection{Text: "banks", Context: "She deposited money in the banks.", Locale: "zh-CN"}, Mode: "word", Target: "简体中文", Config: cfg, Key: "test-only"}, func(e Event) {
				if e.Kind == "dictionary" {
					definitions <- *e.Dictionary
				}
			}); err != nil {
				t.Fatal(err)
			}
			close(definitions)
			first := <-definitions
			if first.Word != "banks" || first.SourceLanguage != "" || first.Lemma != "" || first.Senses[0].Meaning != "银行" {
				t.Fatalf("first meaning was delayed by metadata: %+v", first)
			}
			var final core.Dictionary
			for d := range definitions {
				final = d
			}
			if err := core.Validate.Struct(final); err != nil || final.Lemma != "bank" || final.SourceLanguage != "en" {
				t.Fatalf("compact result cannot be collected: %+v, %v", final, err)
			}
			if final.Pronunciation != "/bæŋks/" || len(final.Forms) != 0 || len(final.Collocations) != 0 || len(final.Senses[0].Examples) != 0 {
				t.Fatal("pronunciation was lost or unrequested metadata was added")
			}
			if mode != "json_schema" {
				return
			}
			format := payload["response_format"].(map[string]any)
			schema := format["json_schema"].(map[string]any)["schema"].(map[string]any)
			defs := schema["$defs"].(map[string]any)
			root := defs[strings.TrimPrefix(schema["$ref"].(string), "#/$defs/")].(map[string]any)
			props := root["properties"].(map[string]any)
			if len(props) != 4 || props["pronunciation"] == nil || props["senses"] == nil || props["lemma"] == nil || props["source_language"] == nil {
				t.Fatalf("schema still requests a verbose dictionary: %v", props)
			}
			senses := props["senses"].(map[string]any)
			if senses["maxItems"] != float64(3) {
				t.Fatal("schema lost its small sense budget")
			}
			item := senses["items"].(map[string]any)
			sense := defs[strings.TrimPrefix(item["$ref"].(string), "#/$defs/")].(map[string]any)["properties"].(map[string]any)
			if len(sense) != 3 || sense["meaning"].(map[string]any)["maxLength"] != float64(80) {
				t.Fatalf("schema still requests verbose sense fields: %v", sense)
			}
		})
	}
}

func TestPhraseUsesCompactStreamingDictionaryInEveryFormat(t *testing.T) {
	for _, format := range []string{"prompt", "json_object", "json_schema"} {
		t.Run(format, func(t *testing.T) {
			var prompt string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Messages []struct{ Content string }
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if len(body.Messages) > 0 {
					prompt = body.Messages[0].Content
				}
				w.Header().Set("Content-Type", "text/event-stream")
				sse(w, map[string]string{"reasoning_content": "识别短语的整体含义"}, "")
				sse(w, map[string]string{"content": `{"senses":[{"part_of_speech":"动词短语","meaning":"考虑到","in_context":true}`}, "")
				sse(w, map[string]string{"content": `],"source_language":"en","lemma":"take into account"}`}, "")
				sse(w, map[string]string{}, "stop")
				fmt.Fprint(w, "data: [DONE]\n\n")
			}))
			defer server.Close()
			cfg := core.DefaultConfig()
			cfg.Endpoint, cfg.StructuredMode = server.URL, format
			var snapshots []core.Dictionary
			var kinds []string
			err := Stream(context.Background(), Request{
				Mode: "phrase", Target: "简体中文", Key: "test-only", Config: cfg,
				Selection: core.Selection{Text: "take into account", Context: "We take costs into account."},
			}, func(e Event) {
				kinds = append(kinds, e.Kind)
				if e.Dictionary != nil {
					snapshots = append(snapshots, *e.Dictionary)
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(prompt, "never split it into individual words") ||
				strings.Join(kinds, ",") != "reasoning,answer_start,dictionary,dictionary" {
				t.Fatalf("phrase took the sentence path or lost streaming: %v", kinds)
			}
			if snapshots[0].Lemma != "" || snapshots[0].Senses[0].Meaning != "考虑到" {
				t.Fatal("first phrase meaning waited for the complete response")
			}
			final := snapshots[len(snapshots)-1]
			if err := core.Validate.Struct(final); err != nil || final.Lemma != "take into account" {
				t.Fatalf("phrase cannot be collected: %+v %v", final, err)
			}
		})
	}
}

func sse(w io.Writer, delta map[string]string, finish string) {
	b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": delta, "finish_reason": finish, "index": 0}}})
	fmt.Fprintf(w, "data: %s\n\n", b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
func TestSDKStreamsReasoningAndTranslationAndExcludesContext(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.Header().Set("Content-Type", "text/event-stream")
		sse(w, map[string]string{"reasoning_content": "disambiguating", "content": "你好"}, "")
		sse(w, map[string]string{}, "stop")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	cfg := core.DefaultConfig()
	cfg.Endpoint = server.URL
	cfg.IncludeContext = false
	var text, thought string
	var kinds []string
	err := Stream(context.Background(), Request{Selection: core.Selection{Text: "Hello", Context: "PRIVATE CONTEXT", Locale: "zh-CN"}, Mode: "sentence", Target: "简体中文", Config: cfg, Key: "test-only"}, func(e Event) {
		kinds = append(kinds, e.Kind)
		if e.Kind == "text" {
			text += e.Text
		}
		if e.Kind == "reasoning" {
			thought += e.Text
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if text != "你好" || thought != "disambiguating" {
		t.Fatalf("lost delta: %s %s", text, thought)
	}
	if strings.Join(kinds, ",") != "reasoning,answer_start,text" {
		t.Fatalf("wrong phase order: %v", kinds)
	}
	b, _ := json.Marshal(payload)
	if strings.Contains(string(b), "PRIVATE CONTEXT") {
		t.Fatal("disabled context leaked")
	}
}

func TestWordAnswerStartsBeforeJSONCanYieldASense(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		sse(w, map[string]string{"reasoning_content": "判断含义"}, "")
		sse(w, map[string]string{"content": `{"senses":[`}, "")
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Error("answer_start waited for a complete dictionary sense")
		}
		sse(w, map[string]string{"content": `{"part_of_speech":"noun","meaning":"银行","in_context":true}],"source_language":"en","lemma":"bank","pronunciation":"/bæŋks/"}`}, "")
		sse(w, map[string]string{}, "stop")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	cfg := core.DefaultConfig()
	cfg.Endpoint = server.URL
	events := make(chan string, 16)
	err := Stream(context.Background(), Request{Selection: core.Selection{Text: "bank"}, Mode: "word", Target: "中文", Config: cfg, Key: "test"}, func(e Event) {
		events <- e.Kind
		if e.Kind == "answer_start" {
			started <- struct{}{}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	close(events)
	var kinds []string
	for kind := range events {
		kinds = append(kinds, kind)
	}
	if strings.Join(kinds, ",") != "reasoning,answer_start,dictionary,dictionary" {
		t.Fatalf("wrong dictionary phase order: %v", kinds)
	}
}
func TestCancelClosesModelRequest(t *testing.T) {
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		sse(w, map[string]string{"content": "partial"}, "")
		<-r.Context().Done()
		close(closed)
	}))
	defer server.Close()
	cfg := core.DefaultConfig()
	cfg.Endpoint = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	temperature := 0.0
	cfg.Temperature = &temperature
	defer cancel()
	err := Stream(ctx, Request{Selection: core.Selection{Text: "Hello"}, Mode: "sentence", Target: "中文", Config: cfg, Key: "test"}, func(e Event) {
		if e.Kind == "text" {
			cancel()
		}
	})
	if err == nil {
		t.Fatal("cancelled stream reported success")
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("HTTP request remained alive after cancellation")
	}
}

func TestBrokenStreamRetainsTextButReportsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		sse(w, map[string]string{"content": "partial"}, "")
	}))
	defer server.Close()
	cfg := core.DefaultConfig()
	cfg.Endpoint = server.URL
	var text string
	e := Stream(context.Background(), Request{Selection: core.Selection{Text: "hello"}, Mode: "sentence", Target: "中文", Config: cfg, Key: "test"}, func(ev Event) { text += ev.Text })
	if e == nil || text != "partial" {
		t.Fatalf("broken stream: %s %v", text, e)
	}
}
