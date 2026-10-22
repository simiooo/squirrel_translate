package core

import "testing"

func TestLocaleInitializesPreferredLanguage(t *testing.T) {
	for _, tt := range []struct{ locale, want string }{{"zh-CN", "简体中文"}, {"zh-Hant-TW", "繁體中文"}, {"en-US", "English"}, {"ja-JP", "日本語"}, {"de_DE", "Deutsch"}} {
		if got := LanguageForLocale(tt.locale); got != tt.want {
			t.Errorf("%s -> %s", tt.locale, got)
		}
	}
}

func TestClassificationAndOverrideHints(t *testing.T) {
	for _, tt := range []struct{ in, want string }{{"bank", "word"}, {"don't", "word"}, {"state-of-the-art", "word"}, {"你好", "word"}, {"これは長い文章です。", "sentence"}, {"This is a sentence.", "sentence"}, {"two words", "phrase"}, {"你好，今天天气不错", "sentence"},
		{"take into account", "phrase"}, {"in spite of", "phrase"}, {"a piece of cake", "phrase"}, {"break the ice", "phrase"}, {"“on the other hand”", "phrase"},
		{"可持续发展", "phrase"}, {"人工智能", "word"}, {"I agree", "sentence"}, {"She left", "sentence"}, {"It's raining", "sentence"}, {"This is fine", "sentence"}, {"我喜欢翻译", "sentence"},
		{"Take care!", "sentence"}, {"first line\nsecond line", "sentence"}, {"", "sentence"}, {"A sentence with far too many words to be a short phrase", "sentence"}} {
		if got := Classify(tt.in); got != tt.want {
			t.Errorf("%q: got %s want %s", tt.in, got, tt.want)
		}
	}
}

func TestExplicitModesAndLegacyVocabulary(t *testing.T) {
	for _, tt := range []struct{ text, choice, want string }{
		{"take into account", "自动", "phrase"}, {"Please go", "句子 / 段落", "sentence"},
		{"I agree", "短语", "phrase"}, {"人工智能", "短语", "phrase"},
		{"你好", "单词", "word"}, {"two words", "单词 / 短语", "phrase"},
	} {
		if got := ResolveMode(tt.text, tt.choice); got != tt.want {
			t.Errorf("%+v: %s", tt, got)
		}
	}
	if (Entry{Word: "bank"}).LexicalKind() != "word" ||
		(Entry{Word: "take into account"}).LexicalKind() != "phrase" ||
		(Entry{Word: "人工智能", Kind: "phrase"}).LexicalKind() != "phrase" {
		t.Fatal("stored category or legacy inference lost")
	}
}
func TestTargetLanguageRules(t *testing.T) {
	c := DefaultConfig()
	if Target("你好", "自动", c) != "English" {
		t.Fatal("Chinese selection should switch to second language")
	}
	if Target("hello", "自动", c) != "简体中文" {
		t.Fatal("default should translate into primary language")
	}
	if Target("hello", "日本語", c) != "日本語" {
		t.Fatal("manual target must win")
	}
}
func TestEndpointValidation(t *testing.T) {
	for _, endpoint := range []string{"ftp://host", "https://key:secret@host", "https://host?key=abc", "https://host#frag", "not-url"} {
		c := DefaultConfig()
		c.Endpoint = endpoint
		if c.Check() == nil {
			t.Errorf("accepted invalid endpoint %q", endpoint)
		}
	}
	c := DefaultConfig()
	c.Endpoint = "http://127.0.0.1:1234/v1"
	if e := c.Check(); e != nil {
		t.Fatal(e)
	}
}
func TestPlacementNeverLeavesNegativeOriginMonitor(t *testing.T) {
	work := Rect{-1920, -200, 1920, 1040}
	for _, anchor := range []Rect{{-1, 830, 1, 1}, {-1920, -200, 60, 20}, {-1000, 780, 300, 50}, {-3000, 300, 100, 50}} {
		for _, size := range [][2]int{{520, 600}, {3000, 2000}} {
			r := Place(anchor, work, size[0], size[1], 8)
			if r.X < work.X || r.Y < work.Y || r.X+r.Width > work.X+work.Width || r.Y+r.Height > work.Y+work.Height {
				t.Fatalf("out of work area: %+v", r)
			}
		}
	}
}
func TestSelectionNeedsVisibleText(t *testing.T) {
	for _, text := range []string{"", " ", "\t\n", "\u00a0", "\u200b", "\ufeff", "\u2060", "\u00ad", "\ufffc", "\x00"} {
		if HasText(text) {
			t.Errorf("%q should count as empty", text)
		}
	}
	for _, text := range []string{"bank", " 你好 ", "take care", "😀", "a\u200bb"} {
		if !HasText(text) {
			t.Errorf("%q should count as text", text)
		}
	}
}
