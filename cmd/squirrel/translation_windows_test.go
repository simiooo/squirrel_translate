//go:build windows

package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/llm"
	"github.com/simiooo/squirrel_translate/internal/store"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestTranslationActionsStayInsideSmallWindow(t *testing.T) {
	s := &translation{mode: "word", modeChoice: "单词 / 短语", targetChoice: "自动", target: "简体中文", done: true, demo: true, includeContext: true, selection: core.Selection{Text: "bank"}, dictionary: &core.Dictionary{Word: "bank", Lemma: "bank", SourceLanguage: "en", Senses: []core.Sense{{PartOfSpeech: "noun · 名词", Meaning: "银行；河岸", Examples: []core.Example{{Text: "I went to the bank.", Translation: "我去了银行。"}}}}}}
	tt := ui.NewTester(s.view, 520, 600)
	for _, size := range [][2]int{{520, 600}, {360, 260}} {
		tt.SetSize(size[0], size[1])
		for _, label := range []string{"复制", "加入词汇表", "重新翻译", "目标语言", "置顶", "关闭"} {
			r, ok := tt.Find(label)
			if !ok {
				t.Fatalf("missing %s", label)
			}
			if r.X < 0 || r.Y < 0 || r.X+r.W > float32(size[0]) || r.Y+r.H > float32(size[1]) {
				t.Fatalf("%s overflowed %v: %+v", label, size, r)
			}
		}
	}
	tt.SetSize(520, 600)
	if path := os.Getenv("SQUIRREL_UI_PREVIEW"); path != "" {
		f, e := os.Create(path)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		if e = png.Encode(f, tt.Image()); e != nil {
			t.Fatal(e)
		}
	}
}

func TestThinkingEndsBeforeFirstCompleteDictionarySense(t *testing.T) {
	for _, event := range []llm.Event{{Kind: "answer_start"}, {Kind: "text", Text: "你好"}, {Kind: "dictionary", Dictionary: &core.Dictionary{Word: "bank"}}} {
		s := &translation{mode: "word", targetChoice: "自动", cancel: func() {}}
		s.applyEvent(llm.Event{Kind: "reasoning", Text: "第一行\n正在判断词义"})
		tt := ui.NewTester(s.view, 360, 260)
		if _, ok := tt.Find("最新思考"); !ok {
			t.Fatal("reasoning loading was not visible")
		}
		s.applyEvent(event)
		s.applyEvent(llm.Event{Kind: "reasoning", Text: "late delta"})
		tt.Frame()
		if _, ok := tt.Find("最新思考"); ok || s.reasoning != "" || !s.answerStarted {
			t.Fatalf("%s failed to end thinking permanently", event.Kind)
		}
	}
}

func TestLongTranslationKeepsActionsFixedAndStopsFollowingWhenReading(t *testing.T) {
	s := &translation{mode: "sentence", targetChoice: "自动", text: strings.Repeat("这是一段需要滚动阅读的译文。\n\n", 80), answerStarted: true,
		body: ui.ListState{FollowEnd: true}, cancel: func() {}}
	tt := ui.NewTester(s.view, 360, 260)
	before, ok := tt.Find("复制")
	if !ok || before.Y+before.H > 260 || !s.body.AtEnd() {
		t.Fatal("long result overflowed footer or did not follow stream")
	}
	tt.Scroll(180, 140, 0, -160)
	if s.body.AtEnd() {
		t.Fatal("user could not scroll away from end")
	}
	s.text += "\n\n新生成的一段译文。"
	tt.Frame()
	after, _ := tt.Find("复制")
	if after != before || s.body.AtEnd() {
		t.Fatal("new tokens moved actions or interrupted reading")
	}
}

func TestThinkingLatestRowAndFadeDoNotRestartForEveryToken(t *testing.T) {
	s := &translation{mode: "sentence", targetChoice: "自动", reasoning: "上一行\n最新内容", cancel: func() {}}
	tt := ui.NewTester(s.view, 360, 260)
	appeared := s.loading.appeared
	tt.Advance(90 * time.Millisecond)
	s.reasoning += "继续输出"
	tt.Frame()
	if s.loading.text != "最新内容继续输出" || s.loading.appeared != appeared {
		t.Fatal("same-row token reset the fade or retained old lines")
	}
	s.reasoning += "\n新的一行"
	tt.Frame()
	if s.loading.text != "新的一行" || !s.loading.appeared.After(appeared) {
		t.Fatal("new line did not start its own fade")
	}
	// A custom-drawn wrapped layout must never paint its preceding rows over
	// the header. Capture that region before replacing a short thought.
	var header []color.Color
	before := tt.Image()
	for y := 0; y < 50; y++ {
		for x := 0; x < 360; x++ {
			header = append(header, before.At(x, y))
		}
	}
	s.reasoning = strings.Repeat("Mixed 中文 text with 👩‍💻 and é. ", 100) + "最新内容"
	tt.Frame()
	tt.Advance(thoughtFade)
	after := tt.Image()
	for y := 0; y < 50; y++ {
		for x := 0; x < 360; x++ {
			if header[y*360+x] != after.At(x, y) {
				t.Fatalf("old thought rows painted over header at %d,%d", x, y)
			}
		}
	}
	r, ok := tt.Find("最新思考")
	if !ok || r.H > 30 || r.X < 0 || r.X+r.W > 360 || s.loading.lastRow.Y <= 0 {
		t.Fatalf("wrapped thought should show exactly its last visible row: %+v", r)
	}
	if !strings.HasSuffix(s.loading.text, "最新内容") || !utf8.ValidString(s.loading.text) || len(s.loading.text) > 4096 {
		t.Fatal("bounded tail lost latest text or split a UTF-8 sequence")
	}
	if err := tt.Click("停止"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("最新思考"); ok {
		t.Fatal("stopping left thought visible")
	}
}

func TestStreamingWordCannotBeCollectedEarly(t *testing.T) {
	s := &translation{mode: "word", modeChoice: "单词 / 短语", targetChoice: "自动", target: "简体中文", dictionary: &core.Dictionary{Word: "bank", Senses: []core.Sense{{PartOfSpeech: "noun", Meaning: "银行"}}}}
	tt := ui.NewTester(s.view, 520, 600)
	_ = tt.Click("加入词汇表")
	if s.notice != "" {
		t.Fatal("collection action ran for incomplete stream")
	}
}

func TestCollectionClickReportsInvalidDefinition(t *testing.T) {
	s := &translation{mode: "word", targetChoice: "自动", done: true, dictionary: &core.Dictionary{Word: "bank"}}
	tt := ui.NewTester(s.view, 520, 600)
	if err := tt.Click("加入词汇表"); err != nil {
		t.Fatal(err)
	}
	if s.notice == "" {
		t.Fatal("click was lost: invalid definition must report why saving failed")
	}
}

func TestCollectionClickPersistsWordAndPhraseAndReportsState(t *testing.T) {
	for _, word := range []string{"bank", "take into account"} {
		t.Run(word, func(t *testing.T) {
			db, err := store.New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			a := &application{store: db}
			s := &translation{app: a, mode: core.Classify(word), targetChoice: "自动", target: "简体中文", includeContext: true,
				selection: core.Selection{Text: word, Context: "Context snapshot"}, dictionary: &core.Dictionary{Word: word, Lemma: word, SourceLanguage: "en", Senses: []core.Sense{{PartOfSpeech: "n.", Meaning: "测试释义"}}}}
			tt := ui.NewTester(s.view, 360, 260)
			_ = tt.Click("加入词汇表")
			if entries, _ := db.List(); len(entries) != 0 {
				t.Fatal("saved while generating")
			}
			s.done = true
			tt.Frame()
			if err := tt.Click("加入词汇表"); err != nil {
				t.Fatal(err)
			}
			if s.notice != "已加入词汇表" {
				t.Fatalf("save click did not succeed: %s", s.notice)
			}
			if label, _ := s.collectionLabel(); label != "已收藏" {
				t.Fatalf("missing saved feedback: %s", label)
			}
			reopened, _ := store.New(db.Dir)
			entries, err := reopened.List()
			if err != nil || len(entries) != 1 || entries[0].Kind != s.mode || entries[0].Context != s.selection.Context {
				t.Fatalf("saved entry did not survive reopening: %+v %v", entries, err)
			}
			s.note = "updated note"
			tt.Frame()
			if err := tt.Click("加入词汇表"); err != nil {
				t.Fatal(err)
			}
			entries, err = reopened.List()
			if err != nil || len(entries) != 1 || entries[0].Note != s.note || s.notice != "词汇表中的释义已更新" {
				t.Fatalf("second click was lost or duplicated: %+v %v; %s", entries, err, s.notice)
			}
		})
	}
}

func TestCollectionFailureCanBeRetriedAndSentenceHasNoButton(t *testing.T) {
	db, _ := store.New(t.TempDir())
	path := filepath.Join(db.Dir, "vocabulary.json")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &translation{app: &application{store: db}, mode: "phrase", done: true, targetChoice: "自动", target: "简体中文",
		dictionary: &core.Dictionary{Word: "take care", Lemma: "take care", SourceLanguage: "en", Senses: []core.Sense{{PartOfSpeech: "短语", Meaning: "保重"}}}}
	tt := ui.NewTester(s.view, 520, 360)
	_ = tt.Click("加入词汇表")
	if label, _ := s.collectionLabel(); label != "重试收藏" || !strings.HasPrefix(s.notice, "收藏失败：") {
		t.Fatal("failed save has no actionable feedback")
	}
	if got, _ := os.ReadFile(path); string(got) != "{broken" {
		t.Fatal("failed save destroyed original vocabulary")
	}
	if err := os.Remove(path); err != nil { // Test fixture only.
		t.Fatal(err)
	}
	tt.Frame()
	_ = tt.Click("加入词汇表")
	if s.saveFailed || len(s.app.entries) != 1 {
		t.Fatal("save did not recover on retry")
	}
	s.mode = "sentence"
	tt.Frame()
	if _, found := tt.Find("加入词汇表"); found {
		t.Fatal("sentence incorrectly exposes collection")
	}
	if _, found := tt.Find("翻译类别"); !found {
		t.Fatal("ambiguous sentence has no category correction")
	}
}

func TestTranslationOmitsNoteEditorAndPreservesExistingNote(t *testing.T) {
	s := &translation{mode: "phrase", done: true, targetChoice: "自动", note: "旧版本保存的个人备注",
		dictionary: &core.Dictionary{Word: "take care", Lemma: "take care", SourceLanguage: "en", Senses: []core.Sense{{PartOfSpeech: "短语", Meaning: "保重"}}}}
	tt := ui.NewTester(s.view, 520, 360)
	for _, removed := range []string{"笔记", "词条笔记"} {
		if _, found := tt.Find(removed); found {
			t.Fatalf("removed note control remains: %s", removed)
		}
	}
	if _, found := tt.Find("已有备注"); !found {
		t.Fatal("existing note was lost")
	}
}
