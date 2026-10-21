//go:build windows

package main

import (
	"context"
	"sync"
	"time"

	"github.com/simiooo/squirrel_translate/internal/llm"
)

const thoughtRefresh = 100 * time.Millisecond

// Coalesce reasoning before scheduling any UI work. Answer events bypass the
// timer and discard pending thought, so a trailing callback cannot reopen it.
type streamUpdates struct {
	mu                    sync.Mutex
	interval              time.Duration
	emit                  llm.Emit
	cancel                context.CancelFunc
	timer                 *time.Timer
	pending               string
	reasoningBytes        int
	answerStarted, closed bool
}

func (updates *streamUpdates) receive(e llm.Event) {
	updates.mu.Lock()
	defer updates.mu.Unlock()
	if updates.closed {
		return
	}
	if e.Kind != "reasoning" {
		if e.Kind == "answer_start" || e.Kind == "text" || e.Kind == "dictionary" {
			updates.answerStarted = true
			updates.discardThought()
		}
		updates.emit(e)
		return
	}
	if updates.answerStarted || e.Text == "" {
		return
	}
	updates.reasoningBytes += len(e.Text)
	if updates.reasoningBytes > 1024*1024 {
		updates.discardThought()
		updates.cancel()
		return
	}
	updates.pending += e.Text
	if updates.timer == nil {
		updates.timer = time.AfterFunc(updates.interval, updates.flush)
	}
}

func (updates *streamUpdates) flush() {
	updates.mu.Lock()
	defer updates.mu.Unlock()
	updates.timer = nil
	if updates.closed || updates.answerStarted || updates.pending == "" {
		return
	}
	updates.emit(llm.Event{Kind: "reasoning", Text: updates.pending})
	updates.pending = ""
}

func (updates *streamUpdates) discardThought() {
	if updates.timer != nil {
		updates.timer.Stop()
		updates.timer = nil
	}
	updates.pending = ""
}

func (updates *streamUpdates) close() {
	updates.mu.Lock()
	defer updates.mu.Unlock()
	updates.closed = true
	updates.discardThought()
}
