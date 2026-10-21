//go:build windows

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/llm"
	"github.com/simiooo/squirrel_translate/internal/native"
	"github.com/simiooo/squirrel_translate/internal/store"
)

type translation struct {
	app                                        *application
	id                                         uint64
	window                                     *mygo.Window
	selection                                  core.Selection
	modeChoice, mode, targetChoice, target     string
	dictionary                                 *core.Dictionary
	text, reasoning, err, notice, note         string
	pinned, closed, done, includeContext, demo bool
	answerStarted                              bool
	saveFailed                                 bool
	embedded                                   bool
	availableWidth                             float32
	copiedUntil                                time.Time
	opened                                     time.Time
	generation                                 uint64
	cancel                                     context.CancelFunc
	body                                       ui.ListState
	md                                         markdown.Options
	loading                                    thoughtLine
}

func (a *application) openTranslation(sel core.Selection, mode, target string) *translation {
	return a.newTranslation(sel, mode, target, true)
}
func (a *application) newTranslation(sel core.Selection, mode, target string, start bool) *translation {
	a.closeToolbar()
	// A new selection replaces only transient sessions. Pinned windows own
	// their selection snapshot and remain independent of later captures.
	for _, s := range a.sessions {
		if !s.pinned {
			s.window.Close()
		}
	}
	a.next++
	s := &translation{app: a, id: a.next, selection: sel, modeChoice: mode, targetChoice: target, includeContext: a.cfg.IncludeContext, demo: a.demo, opened: time.Now(), cancel: func() {}}
	s.body.FollowEnd = true
	s.md.ShareSelection = true
	s.md.Reveal = &markdown.Reveal{Unit: markdown.AtOnce, Fade: -1}
	work, scale := native.WorkArea(sel.Bounds.X, sel.Bounds.Y)
	maxH := min(720, int(float64(work.Height)/scale*.8))
	maxW := min(640, int(float64(work.Width)/scale*.95))
	height := 340
	if core.ResolveMode(sel.Text, mode) == "sentence" {
		height = 260 + min(240, len([]rune(sel.Text))/120*24)
	}
	s.window = mygo.NewWindow(mygo.WindowOptions{Title: tr("翻译") + " · " + productName(), Panel: true, Hidden: true, Frameless: true, Width: min(520, maxW), Height: min(height, maxH), MaxWidth: maxW, MaxHeight: maxH, MinWidth: min(360, maxW), MinHeight: min(260, maxH), Content: ui.View(s.view)})
	_ = s.window.SetIcon(appIconPNG)
	native.RoundCorners(s.window.NativeHandle())
	a.sessions[s.id] = s
	s.window.OnClosed(func() {
		if s.closed {
			return
		}
		s.closed = true
		s.cancel()
		delete(a.sessions, s.id)
		s.text = ""
		s.reasoning = ""
		s.loading = thoughtLine{}
		s.dictionary = nil
		s.selection = core.Selection{}
	})
	native.Place(s.window.NativeHandle(), sel.Bounds, min(520, maxW), min(height, maxH))
	s.window.Show()
	if start {
		s.run()
	}
	return s
}

func (a *application) openStored(entry core.Entry) {
	x, y := native.Cursor()
	s := a.newTranslation(core.Selection{Text: entry.Word, Context: entry.Context, Locale: native.Locale(), Source: entry.Source, Bounds: core.Rect{X: x, Y: y}}, core.ModeLabel(entry.LexicalKind()), entry.TargetLanguage, false)
	s.mode = entry.LexicalKind()
	s.target = entry.TargetLanguage
	s.done = true
	s.dictionary = &entry.Definition
	s.note = entry.Note
	s.body.FollowEnd = false
	s.demo = strings.HasPrefix(entry.Source, "本地演示")
	s.window.Invalidate()
}
func (s *translation) run() {
	if s.closed {
		return
	}
	s.cancel()
	s.generation++
	generation := s.generation
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = false
	s.err = ""
	s.notice = ""
	s.saveFailed = false
	s.text = ""
	s.reasoning = ""
	s.answerStarted = false
	s.loading = thoughtLine{}
	s.dictionary = nil
	s.body = ui.ListState{FollowEnd: true}
	s.mode = core.ResolveMode(s.selection.Text, s.modeChoice)
	s.target = core.Target(s.selection.Text, s.targetChoice, s.app.cfg)
	cfg := s.app.cfg
	cfg.IncludeContext = s.includeContext
	req := llm.Request{Selection: s.selection, Mode: s.mode, Target: s.target, Config: cfg, Key: string(s.app.key), Demo: s.demo}
	go func() {
		updates := &streamUpdates{interval: thoughtRefresh, cancel: cancel, emit: func(e llm.Event) {
			s.window.Update(func() {
				if s.closed || generation != s.generation {
					return
				}
				s.applyEvent(e)
			})
		}}
		stop := context.AfterFunc(ctx, updates.close)
		err := llm.Stream(ctx, req, updates.receive)
		updates.close()
		stop()
		s.window.Update(func() {
			if s.closed || generation != s.generation {
				return
			}
			s.done = true
			s.finishThinking()
			if err != nil {
				if ctx.Err() != nil {
					s.notice = tr("已停止，保留已生成内容")
				} else {
					s.err = err.Error()
				}
			}
		})
	}()
}

// The first answer delta ends thinking, even when a dictionary is still
// waiting for its first complete sense. Late reasoning cannot reopen it.
func (s *translation) applyEvent(e llm.Event) {
	switch e.Kind {
	case "answer_start":
		s.finishThinking()
	case "text":
		s.finishThinking()
		s.text += e.Text
	case "dictionary":
		s.finishThinking()
		s.dictionary = e.Dictionary
	case "reasoning":
		if s.answerStarted || s.done {
			return
		}
		if len(s.reasoning)+len(e.Text) <= 1024*1024 {
			s.reasoning += e.Text
		} else {
			s.cancel()
		}
	}
}

func (s *translation) finishThinking() {
	s.answerStarted = true
	s.reasoning = ""
	s.loading = thoughtLine{}
}

func (s *translation) view(c *ui.Context) {
	t := minimalTheme(c)
	space, controls := uiSpacing.Translation, uiSpacing.Controls
	s.md.Theme = &markdown.Theme{FontSize: 16, LineHeight: space.BodyLineHeight, BlockSpacing: space.ParagraphGap / 16, ItemSpacing: space.ListItemGap / 16}
	ui.Column(c).Fill().MinHeight(0).Background(t.Background).Children(func() {
		if !s.embedded {
			ui.Row(c).Padding(space.HeaderTop, space.InsetX, space.HeaderBottom).Gap(space.HeaderGap).AlignItems(ui.Center).Children(func() {
				ui.Text(c, tr("翻译")).FontSize(16).Bold().DragWindow()
				if iconButton(c, "refresh", tr("重新翻译")).Clicked() {
					s.run()
				}
				ui.Box(c).Grow(1).Height(32).DragWindow()
				if s.demo {
					ui.Text(c, tr("演示")).FontSize(12).TextColor(t.TextMuted)
				}
				language := s.targetChoice
				if language == "自动" && s.target != "" {
					language = s.target
				}
				if plainSelect(c, &s.targetChoice, core.Languages, language).Label(tr("目标语言")).Tooltip(tr("目标语言")).Changed() {
					s.run()
				}
				ui.Box(c).Width(1).Height(16).Margin(0, space.HeaderGap/2).Background(t.Border)
				if pinToggle(c, &s.pinned).Changed() {
					s.window.SetAlwaysOnTop(s.pinned)
				}
				if iconButton(c, "close", tr("关闭")).Clicked() {
					s.window.Close()
				}
			})
		}
		if !s.done && !s.answerStarted {
			s.loading.view(c, s.reasoning, s.availableWidth)
		}
		inset := space.InsetX
		contentY := space.ContentY
		if s.embedded {
			inset = metric("Space/8")
			contentY = metric("Space/8")
		}
		if s.err != "" {
			contentY = metric("Space/8")
		}
		ui.List(c, &s.body, 1, func(i int) {
			s.result(c)
			if s.err != "" {
				ui.Text(c, tr("翻译未完成")).FontSize(18).Bold().Margin(0, 0, 8)
				ui.Text(c, s.err).FontSize(13).TextColor(t.TextMuted).Selectable()
			}
		}).Grow(1).MinHeight(space.BodyMinHeight).Padding(contentY, inset)
		if s.notice != "" {
			ui.Text(c, s.notice).FontSize(12).Padding(controls.CompactY, inset).MaxHeight(50).TextColor(statusColor(c, s.notice))
		}
		footerBottom := space.FooterY
		if s.embedded {
			footerBottom = 0
		}
		ui.Row(c).Padding(space.FooterY, max(0, inset-controls.ButtonX), footerBottom).Gap(space.ActionGap).AlignItems(ui.Center).Children(func() {
			if s.err != "" {
				if textButton(c, tr("模型设置"), "settings").Clicked() && s.app != nil {
					s.app.tab = 2
					s.app.settingsTab = 0
					s.app.window.Show()
				}
				ui.Spacer(c)
				if primaryButton(c, tr("重试"), "refresh").Clicked() {
					s.run()
				}
				return
			}
			if core.IsLexical(s.mode) {
				label, hint := s.collectionLabel()
				icon := "star"
				if label == tr("已收藏") {
					icon = "star-filled"
				}
				if iconButton(c, icon, tr("加入词汇表")).Tooltip(label + " · " + hint).Disabled(!s.done).Clicked() {
					s.saveWord()
				}
			}
			if !s.done {
				if textButton(c, tr("停止")).Clicked() {
					s.cancel()
					s.finishThinking()
				}
			}
			ui.Spacer(c)
			if plainSelect(c, &s.modeChoice, core.Modes, core.ModeLabel(s.mode)).Label(tr("翻译类别")).Tooltip(tr("纠正单词、短语或句子识别")).Changed() {
				s.run()
			}
			copyLabel, copyIcon := tr("复制"), "copy"
			if c.Now().Before(s.copiedUntil) {
				copyLabel, copyIcon = tr("已复制"), "check"
				c.After(s.copiedUntil.Sub(c.Now()))
			}
			if textButton(c, copyLabel, copyIcon).Label(tr("复制")).Disabled(s.text == "" && s.dictionary == nil).Clicked() {
				mygo.Clipboard.WriteText(s.copyText())
				s.copiedUntil = c.Now().Add(2 * time.Second)
			}
		})
	})
}
func (s *translation) result(c *ui.Context) {
	t := c.Theme()
	space := uiSpacing.Dictionary
	if s.mode == "sentence" {
		root := ui.Column(c).Selectable().MinWidth(0)
		root.ContextMenu(func(m *ui.Menu) {
			if m.Item(tr("查询选词并收藏…")).Chosen() {
				parts := root.SelectedTexts()
				var text []string
				for _, p := range parts {
					runes := []rune(p.Text)
					start, end := max(0, p.Start), min(len(runes), p.End)
					if start < end {
						text = append(text, string(runes[start:end]))
					}
				}
				word := strings.TrimSpace(strings.Join(text, " "))
				if word != "" {
					s.lookupWord(word)
				}
			}
			m.Separator()
			m.EditItems()
		})
		root.Children(func() {
			if s.text == "" {
				return
			} else if s.done {
				markdown.View(c, s.text, &s.md)
			} else {
				markdown.Stream(c, s.text, &s.md)
			}
		})
		return
	}
	if s.dictionary == nil {
		if !s.done && s.answerStarted {
			ui.Spinner(c).Label(tr("生成译文")).Size(14, 14)
		}
		return
	}
	d := s.dictionary
	ui.Column(c).Gap(space.SectionGap).Children(func() {
		ui.Column(c).Gap(space.MetadataGap).Children(func() {
			ui.Text(c, d.Word).FontSize(metric("Type/Word")).Bold().Selectable()
			if d.Pronunciation != "" {
				ui.Text(c, d.Pronunciation).TextColor(t.TextMuted).Selectable()
			}
			if d.Lemma != "" && d.Lemma != d.Word {
				ui.Text(c, d.Lemma).FontSize(12).TextColor(t.TextMuted)
			}
			if len(d.Forms) > 0 {
				ui.Text(c, strings.Join(d.Forms, " · ")).FontSize(12).TextColor(t.TextMuted)
			}
		})
		ui.Column(c).Gap(space.SenseGap).Children(func() {
			for _, sense := range d.Senses {
				ui.Column(c).Gap(space.DetailGap).Selectable().Children(func() {
					label := sense.PartOfSpeech
					if sense.InContext {
						label += tr(" · 当前语境")
					}
					ui.Column(c).Gap(space.PairGap).Children(func() {
						ui.Text(c, tr(label)).FontSize(12).TextColor(t.TextMuted)
						ui.Text(c, sense.Meaning).FontSize(16)
					})
					if sense.Usage != "" {
						ui.Text(c, sense.Usage).FontSize(12).TextColor(t.TextMuted)
					}
					for _, example := range sense.Examples {
						ui.Column(c).Gap(space.PairGap).Children(func() {
							ui.Text(c, example.Text).FontSize(13)
							ui.Text(c, example.Translation).FontSize(12).TextColor(t.TextMuted)
						})
					}
				})
			}
		})
		if len(d.Collocations) > 0 {
			ui.Text(c, tr("搭配：")+strings.Join(d.Collocations, " · ")).FontSize(12)
		}
		if s.note != "" {
			ui.Text(c, s.note).Label(tr("已有备注")).FontSize(12).TextColor(t.TextMuted).Selectable()
		}
	})
}
func (s *translation) lookupWord(word string) {
	sel := core.Selection{Text: limitText(word, 160), Context: limitText(s.text, 2400), Locale: s.selection.Locale, Source: tr("译文选词"), Bounds: s.selection.Bounds}
	s.app.openTranslation(sel, core.ModeLabel(core.ResolveMode(word, "单词 / 短语")), s.targetChoice)
}

func (s *translation) collectionLabel() (string, string) {
	if !s.done {
		return tr("生成中"), tr("释义生成完成后可收藏")
	}
	if s.saveFailed {
		return tr("重试收藏"), tr("上次保存失败，点击重试")
	}
	if s.dictionary != nil && s.app != nil {
		for _, entry := range s.app.entries {
			if store.Matches(entry, s.dictionary.Lemma, s.dictionary.SourceLanguage, s.target) {
				return tr("已收藏"), tr("点击更新已收藏的释义")
			}
		}
	}
	return tr("收藏"), tr("加入词汇表")
}

func (s *translation) saveWord() {
	// The button and the action both guard against partial, cancelled or
	// failed generations. Persist first; only then show a successful state.
	if !core.IsLexical(s.mode) || !s.done {
		return
	}
	defer func() {
		if s.window != nil {
			s.window.Invalidate()
		}
	}()
	s.saveFailed = true
	if s.err != "" || s.dictionary == nil || core.Validate.Struct(s.dictionary) != nil {
		s.notice = tr("释义尚不完整，请重新翻译后收藏")
		return
	}
	source := s.selection.Source
	if s.demo {
		source = "本地演示 · " + source
	}
	contextText := ""
	if s.includeContext {
		contextText = s.selection.Context
	}
	added, e := s.app.store.Save(core.Entry{Kind: s.mode, SourceLanguage: s.dictionary.SourceLanguage, TargetLanguage: s.target, Definition: *s.dictionary, Context: contextText, Source: source, Note: s.note})
	if e != nil {
		s.notice = tr("收藏失败：") + e.Error()
		return
	}
	s.saveFailed = false
	entries, listErr := s.app.store.List()
	if listErr != nil {
		s.notice = tr("已保存，但词汇表刷新失败：") + listErr.Error()
		return
	}
	s.app.entries = entries
	if s.app.window != nil {
		s.app.window.Invalidate()
	}
	for _, session := range s.app.sessions {
		if session.window != nil && !session.closed {
			session.window.Invalidate()
		}
	}
	if added {
		s.notice = tr("已加入词汇表")
	} else {
		s.notice = tr("词汇表中的释义已更新")
	}
}
func (s *translation) copyText() string {
	if s.mode == "sentence" {
		return s.text
	}
	if s.dictionary == nil {
		return ""
	}
	d := s.dictionary
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", d.Word, d.Pronunciation)
	for _, sense := range d.Senses {
		fmt.Fprintf(&b, "%s  %s\n", sense.PartOfSpeech, sense.Meaning)
		for _, ex := range sense.Examples {
			fmt.Fprintf(&b, "%s\n%s\n", ex.Text, ex.Translation)
		}
	}
	return b.String()
}
func testModel(cfg core.Config, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return llm.Stream(ctx, llm.Request{Selection: core.Selection{Text: "Hello, world.", Locale: native.Locale()}, Mode: "sentence", Target: cfg.PrimaryLanguage, Config: cfg, Key: key}, func(llm.Event) {})
}
