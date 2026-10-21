//go:build windows

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/simiooo/squirrel_translate/internal/llm"
)

func TestReasoningIsCoalescedBeforeUIDispatch(t *testing.T) {
	events := make(chan llm.Event, 16)
	updates := &streamUpdates{interval: 30 * time.Millisecond, emit: func(e llm.Event) { events <- e }, cancel: func() {}}
	defer updates.close()
	for range 50 {
		updates.receive(llm.Event{Kind: "reasoning", Text: "词"})
	}
	select {
	case <-events:
		t.Fatal("individual thought tokens scheduled UI work immediately")
	default:
	}
	select {
	case event := <-events:
		if event.Kind != "reasoning" || event.Text != strings.Repeat("词", 50) {
			t.Fatalf("coalescing lost thought deltas: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("trailing thought update was not dispatched")
	}
	for _, text := range []string{"正", "文"} {
		updates.receive(llm.Event{Kind: "text", Text: text})
		select {
		case event := <-events:
			if event.Kind != "text" || event.Text != text {
				t.Fatalf("answer was batched or changed: %+v", event)
			}
		default:
			t.Fatal("answer delta was delayed by reasoning throttle")
		}
	}
}

func TestPendingThoughtCannotArriveAfterAnswerOrClose(t *testing.T) {
	for _, end := range []string{"answer_start", "close"} {
		t.Run(end, func(t *testing.T) {
			events := make(chan llm.Event, 16)
			updates := &streamUpdates{interval: 20 * time.Millisecond, emit: func(e llm.Event) { events <- e }, cancel: func() {}}
			defer updates.close()
			updates.receive(llm.Event{Kind: "reasoning", Text: "pending"})
			if end == "close" {
				updates.close()
			} else {
				updates.receive(llm.Event{Kind: end})
				if event := <-events; event.Kind != end {
					t.Fatalf("phase event was delayed: %+v", event)
				}
			}
			updates.receive(llm.Event{Kind: "reasoning", Text: "late"})
			select {
			case event := <-events:
				t.Fatalf("thought survived %s: %+v", end, event)
			case <-time.After(60 * time.Millisecond):
			}
		})
	}
}
