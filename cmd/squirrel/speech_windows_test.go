//go:build windows

package main

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
)

func TestSpeechOwnershipAndStopButton(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	owner := &translation{}
	a := &application{speech: speechUI{owner: owner, cancel: cancel}}
	a.stopSpeech(&translation{})
	if ctx.Err() != nil {
		t.Fatal("another session stopped the pinned owner's audio")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		a.speechButton(c, owner, "bank", "en", nil, func(string) {})
	}, 180, 80)
	if err := tt.Click("停止朗读"); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil || a.speech.owner != nil {
		t.Fatal("stop did not release playback")
	}
	if _, ok := tt.Find("朗读单词"); !ok {
		t.Fatal("play button was not restored")
	}
}

func TestVocabularyPronunciationAndSpeechControls(t *testing.T) {
	a := &application{tab: 1, cfg: core.DefaultConfig(), selectedEntry: "bank", vocabDetail: true,
		entries: []core.Entry{{ID: "bank", Word: "bank", SourceLanguage: "en", Definition: core.Dictionary{Pronunciation: "/bæŋk/"}}}}
	tt := ui.NewTester(a.view, 1040, 704)
	for _, size := range [][2]int{{1040, 704}, {640, 480}} {
		tt.SetSize(size[0], size[1])
		for _, label := range []string{"/bæŋk/", "朗读单词"} {
			if r, ok := tt.Find(label); !ok || r.W <= 0 || r.X+r.W > float32(size[0])+1 {
				t.Fatalf("%s unavailable at %v: %+v", label, size, r)
			}
		}
	}
}
