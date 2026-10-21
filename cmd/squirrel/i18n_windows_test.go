//go:build windows

package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"testing"
)

func TestEnglishSelectKeepsDomainValues(t *testing.T) {
	i18n.Set("en", "")
	defer i18n.Set("zh-CN", "")
	value := "自动"
	tt := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		formSelect(c, &value, core.Modes).Label("Mode")
	}, 400, 300)
	if err := tt.Click("Mode"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Word"); !ok {
		t.Fatal("English option not visible")
	}
	if err := tt.Click("Word"); err != nil {
		t.Fatal(err)
	}
	if value != "单词" || core.ResolveMode("a sentence", value) != "word" {
		t.Fatalf("display label leaked into stored mode: %q", value)
	}
	i18n.Set("zh-CN", "")
	tt.Frame()
	if value != "单词" {
		t.Fatal("language change changed mode")
	}
}
