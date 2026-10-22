//go:build windows

package main

import (
	"testing"

	"github.com/simiooo/squirrel_translate/internal/core"
)

// A selection gesture can resolve to blank text, invisible formatting
// characters or a selected object. Neither popup may open for those.
func TestEmptySelectionDoesNotOpenPopup(t *testing.T) {
	a := &application{cfg: core.DefaultConfig(), sessions: map[uint64]*translation{}, stop: make(chan struct{})}
	for _, text := range []string{"", " ", "\n\t", "\u00a0", "\u200b", "\ufeff", "\u2060", "\u00ad", "\ufffc"} {
		a.showToolbar(core.Selection{Text: text})
		if a.toolbar != nil {
			t.Fatalf("toolbar opened for empty selection %q", text)
		}
		if s := a.openTranslation(core.Selection{Text: text}, "自动", "自动"); s != nil {
			t.Fatalf("translation opened for empty selection %q", text)
		}
		if len(a.sessions) != 0 {
			t.Fatalf("session created for empty selection %q", text)
		}
	}
}
