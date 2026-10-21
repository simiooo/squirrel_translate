//go:build windows

package main

import (
	"fmt"
	"github.com/simiooo/squirrel_translate/internal/core"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestMinimalSelectKeepsKeyboardAndScrollBehavior(t *testing.T) {
	value := "English"
	tt := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		ui.Column(c).Fill().Children(func() {
			ui.Spacer(c)
			plainSelect(c, &value, core.Languages).Label("语言")
		})
	}, 240, 200)
	if err := tt.Click("语言"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnd)
	r, ok := tt.Find("Русский")
	if !ok || r.Y < 0 || r.Y+r.H > 200 {
		t.Fatalf("last option did not scroll into the available area: %+v", r)
	}
	tt.Key(0, ui.KeyEnter)
	if value != "Русский" {
		t.Fatalf("keyboard selection did not update binding: %q", value)
	}
	_ = tt.Click("语言")
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyEscape)
	if value != "Русский" {
		t.Fatal("Escape changed the selected value")
	}
}

func TestMinimalCheckboxKeepsKeyboardBinding(t *testing.T) {
	value := false
	tt := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		check(c, &value, "置顶")
	}, 200, 100)
	_ = tt.Click("置顶")
	if !value {
		t.Fatal("pointer did not toggle checkbox")
	}
	tt.Key(0, ui.KeySpace)
	if value {
		t.Fatal("Space did not toggle focused checkbox")
	}
}

// Optional rendered previews let UI changes be inspected without opening
// desktop windows or sending model requests.
func TestUIPreviews(t *testing.T) {
	dir := os.Getenv("SQUIRREL_UI_PREVIEW_DIR")
	if dir == "" {
		t.Skip("set SQUIRREL_UI_PREVIEW_DIR to render previews")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	original := uiSpacing
	t.Cleanup(func() { uiSpacing = original })
	if err := configureSpacing(os.Getenv("SQUIRREL_SPACING_FILE")); err != nil {
		t.Fatal(err)
	}
	save := func(name string, tt *ui.Tester) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err = png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	value := "搜索词汇 · bank"
	input := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		ui.Column(c).Fill().Padding(12).Background(c.Theme().Background).Children(func() {
			field(c, &value, "搜索词汇").Label("输入状态预览").Width(440)
		})
	}, 464, 54)
	save("ui-input-rest", input)
	r, _ := input.Find("输入状态预览")
	input.Move(r.X+r.W/2, r.Y+r.H/2)
	save("ui-input-hover", input)
	input.ClickAt(r.X+r.W-4, r.Y+r.H/2)
	save("ui-input-focus", input)
	input.SetDark(true)
	save("ui-input-focus-dark", input)
	options := [3]bool{true, true, false}
	checks := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		ui.Row(c).Fill().Padding(12).Gap(uiSpacing.Main.SectionGap).AlignItems(ui.Center).Background(c.Theme().Background).Children(func() {
			for i, label := range []string{"自动划词", "附带上下文", "开机启动"} {
				check(c, &options[i], label)
			}
		})
	}, 340, 54)
	save("ui-checkboxes", checks)
	if r, ok := checks.Find("开机启动"); ok {
		checks.Move(r.X+r.W/2, r.Y+r.H/2)
		save("ui-checkboxes-hover", checks)
	}
	checks.Move(-1, -1)
	checks.Key(0, ui.KeyTab)
	save("ui-checkboxes-focus", checks)
	checks.SetDark(true)
	save("ui-checkboxes-dark", checks)
	dictionary := core.Dictionary{
		Word: "bank", Lemma: "bank", SourceLanguage: "en", Pronunciation: "/bæŋk/",
		Forms: []string{"banks", "banked", "banking"},
		Senses: []core.Sense{
			{PartOfSpeech: "n. 名词", Meaning: "银行；金融机构", InContext: true, Examples: []core.Example{{Text: "She deposited the money in the bank.", Translation: "她把钱存入了银行。"}}},
			{PartOfSpeech: "n. 名词", Meaning: "河岸；堤岸", Examples: []core.Example{{Text: "We sat on the river bank.", Translation: "我们坐在河岸上。"}}},
			{PartOfSpeech: "v. 动词", Meaning: "存款；依靠，指望", Usage: "bank on someone / something"},
		},
	}
	a := &application{cfg: core.DefaultConfig(), draft: core.DefaultConfig(), mode: "自动", target: "自动", timeoutDraft: "90"}
	width, height := toolbarSize()
	toolbar := ui.NewTester(a.toolbarView, width, height)
	save("ui-toolbar", toolbar)
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		toolbar.SetScale(scale)
		save(fmt.Sprintf("ui-toolbar-%g", scale), toolbar)
	}
	toolbar.SetDark(true)
	save("ui-toolbar-dark-2", toolbar)
	home := ui.NewTester(a.view, 1040, 704)
	save("ui-home", home)
	a.input = "The real voyage of discovery consists not in seeking new landscapes, but in having new eyes."
	a.workspace = &translation{app: a, embedded: true, mode: "sentence", modeChoice: "自动", targetChoice: "自动", target: "简体中文", done: true,
		text: "真正的发现之旅，不在于寻找新的风景，而在于拥有新的眼光。"}
	home.Frame()
	save("ui-workspace", home)
	home.SetDark(true)
	save("ui-workspace-dark", home)
	home.SetSize(640, 480)
	save("ui-workspace-narrow-dark", home)
	home.SetDark(false)
	save("ui-workspace-narrow", home)
	a.workspace = nil
	a.tab = 2
	save("ui-settings", ui.NewTester(a.view, 1040, 704))
	a.settingsTab = 1
	save("ui-preferences", ui.NewTester(a.view, 1040, 704))
	a.settingsTab = 3
	save("ui-appearance", ui.NewTester(a.view, 1040, 704))
	a.settingsTab = 0
	a.moreModelParameters = true
	save("ui-model-parameters", ui.NewTester(a.view, 1040, 704))
	a.moreModelParameters = false
	a.advanced = true
	save("ui-settings-advanced", ui.NewTester(a.view, 680, 480))
	a.advanced = false
	a.tab = 1
	a.entries = []core.Entry{{ID: "bank", Word: "bank", SourceLanguage: "en", TargetLanguage: "简体中文", Definition: dictionary, Context: "She deposited the money in the bank."}, {ID: "serendipity", Word: "serendipity", SourceLanguage: "en", TargetLanguage: "简体中文", Definition: core.Dictionary{Senses: []core.Sense{{PartOfSpeech: "n.", Meaning: "意外发现美好事物的机缘；偶然的幸运"}}}}}
	a.selectedEntry = "bank"
	vocab := ui.NewTester(a.view, 1040, 704)
	save("ui-vocabulary", vocab)
	if r, ok := vocab.Find("朗读单词"); ok {
		vocab.Move(r.X+r.W/2, r.Y+r.H/2)
		save("ui-vocabulary-speech-hover", vocab)
		vocab.Move(-1, -1)
	}
	vocab.SetDark(true)
	save("ui-vocabulary-dark", vocab)
	a.vocabDetail = true
	vocab.SetSize(640, 480)
	save("ui-vocabulary-narrow", vocab)
	s := &translation{app: a, mode: "word", modeChoice: "自动", targetChoice: "自动", target: "简体中文", done: true, dictionary: &dictionary, selection: core.Selection{Text: "bank", Context: "She deposited the money in the bank."}}
	dict := ui.NewTester(s.view, 520, 600)
	save("ui-dictionary", dict)
	if r, ok := dict.Find("重新翻译"); ok {
		dict.Move(r.X+1, r.Y+r.H/2)
		save("ui-button-padding", dict)
		dict.Move(-1, -1)
	}
	dict.SetDark(true)
	save("ui-dictionary-dark", dict)
	dict.SetDark(false)
	dict.SetSize(360, 300)
	save("ui-compact", dict)
	loading := &translation{mode: "sentence", targetChoice: "自动", target: "简体中文", reasoning: "先判断原文的含义。\n结合上下文，保留原句的语气与表达。"}
	progress := ui.NewTester(loading.view, 520, 260)
	progress.Advance(thoughtFade)
	save("ui-loading", progress)
	loading.reasoning += "\n这是最新的一行思考内容。"
	progress.Frame()
	progress.Advance(70 * time.Millisecond)
	save("ui-loading-fade", progress)
	progress.Advance(thoughtFade)
	loading.reasoning = "结合句中 deposited the money，可判断 bank 指金融机构。优先列出当前语境中的含义，再补充常见词义与用法。"
	progress.SetSize(360, 260)
	progress.Advance(thoughtFade)
	save("ui-loading-wrapped", progress)
	progress.SetDark(true)
	save("ui-loading-dark", progress)
	s = &translation{mode: "sentence", modeChoice: "自动", targetChoice: "自动", target: "简体中文", done: true, text: "真正的发现之旅，不在于寻找新的风景，而在于拥有新的眼光。", selection: core.Selection{Text: "The real voyage of discovery consists not in seeking new landscapes, but in having new eyes."}}
	save("ui-sentence", ui.NewTester(s.view, 520, 260))
	s = &translation{mode: "phrase", modeChoice: "自动", targetChoice: "自动", target: "简体中文", done: true,
		dictionary: &core.Dictionary{Word: "take into account", Lemma: "take into account", SourceLanguage: "en",
			Senses: []core.Sense{{PartOfSpeech: "动词短语", Meaning: "考虑到；把……纳入考量", InContext: true}}}}
	phrase := ui.NewTester(s.view, 520, 360)
	save("ui-phrase", phrase)
	phrase.SetSize(360, 260)
	save("ui-phrase-compact", phrase)
	failure := &translation{mode: "sentence", targetChoice: "自动", done: true, err: "请求超时，请检查网络或模型配置。"}
	save("ui-error", ui.NewTester(failure.view, 520, 240))
}
