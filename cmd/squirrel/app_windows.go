//go:build windows

package main

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"github.com/simiooo/squirrel_translate/internal/native"
)

func (a *application) setupTray() {
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: tr("打开") + " " + productName(), Click: func(*mygo.MenuItem, *mygo.Window) { a.window.Show() }},
		{Label: tr("翻译剪贴板"), Click: func(*mygo.MenuItem, *mygo.Window) { a.fromClipboard() }},
		{Label: tr("词汇表"), Click: func(*mygo.MenuItem, *mygo.Window) { a.tab = 1; a.window.Show() }},
		{Label: tr("模型设置"), Click: func(*mygo.MenuItem, *mygo.Window) { a.tab = 2; a.window.Show() }},
		{Label: tr("暂停 / 恢复自动划词"), Click: func(*mygo.MenuItem, *mygo.Window) { a.paused = !a.paused; a.window.Invalidate(); a.closeToolbar() }},
		mygo.Separator(), {Label: tr("退出"), Click: func(*mygo.MenuItem, *mygo.Window) { a.quitting = true; mygo.App.Quit() }},
	})
	if a.tray != nil {
		a.tray.SetMenu(menu)
		a.tray.SetToolTip(productName() + " · " + a.cfg.Shortcut)
		return
	}
	t, e := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), ToolTip: productName() + " · " + a.cfg.Shortcut, Menu: menu})
	if e != nil {
		a.status = tr("托盘初始化失败：") + e.Error()
	}
	a.tray = t
	if t != nil {
		t.OnClick(func() { a.window.Show() })
	}
}
func trayIcon() []byte { return appIconPNG }
func (a *application) registerShortcut(old string) {
	if old != "" {
		mygo.GlobalShortcut.Unregister(old)
	}
	if e := mygo.GlobalShortcut.Register(a.cfg.Shortcut, func() { a.capture(true) }); e != nil {
		a.status = tr("快捷键注册失败（可能被占用）：") + e.Error()
	}
}
func (a *application) fromClipboard() {
	text := strings.TrimSpace(mygo.Clipboard.ReadText())
	if text == "" {
		a.status = tr("剪贴板中没有文本")
		a.window.Show()
		return
	}
	x, y := native.Cursor()
	a.openTranslation(core.Selection{Text: limitText(text, 12000), Locale: native.Locale(), Source: tr("剪贴板"), Bounds: core.Rect{X: x, Y: y, Width: 1, Height: 1}}, a.mode, a.target)
}
func limitText(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func (a *application) capture(direct bool) {
	if a.busy {
		return
	}
	owner := native.Foreground()
	if native.OwnWindow(owner) {
		if direct {
			a.fromClipboard()
		}
		return
	}
	if native.Excluded(owner, a.cfg.ExcludedApps) {
		return
	}
	a.busy = true
	go func() {
		s, e := native.Capture(context.Background())
		a.window.Update(func() {
			a.busy = false
			if e != nil {
				if direct {
					a.status = e.Error()
					a.window.Show()
				}
				return
			}
			if native.Foreground() != owner || native.OwnWindow(native.Foreground()) {
				return
			}
			if direct {
				a.openTranslation(s, "自动", "自动")
			} else {
				a.showToolbar(s)
			}
		})
	}()
}
func (a *application) closeToolbar() {
	if a.toolbar != nil {
		w := a.toolbar
		a.toolbar = nil
		w.Destroy()
	}
}
func (a *application) showToolbar(sel core.Selection) {
	a.closeToolbar()
	a.toolSelection = sel
	width, height := toolbarSize()
	w := mygo.NewWindow(mygo.WindowOptions{Title: tr("划词翻译工具条"), Width: width, Height: height, UseContentSize: true, Hidden: true, Panel: true, Frameless: true, DisableResize: true, Content: ui.View(a.toolbarView)})
	_ = w.SetIcon(appIconPNG)
	a.toolbar = w
	native.NoActivate(w.NativeHandle())
	native.RoundCorners(w.NativeHandle())
	native.Place(w.NativeHandle(), sel.Bounds, width, height)
	w.ShowInactive()
	w.OnClosed(func() {
		if a.toolbar == w {
			a.toolbar = nil
		}
	})
}

func (a *application) toolbarView(c *ui.Context) {
	t := minimalTheme(c)
	s := uiSpacing.Toolbar
	ui.Row(c).Fill().Padding(s.InsetY, s.InsetX).Gap(s.Gap).AlignItems(ui.Center).Background(t.Background).Children(func() {
		ui.Box(c).Size(18, 24).Padding(0, s.GripX).DragWindow().Draw(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, "grip", t.TextMuted) })
		if primaryButton(c, tr("翻译")).Clicked() {
			snapshot := a.toolSelection
			a.closeToolbar()
			a.openTranslation(snapshot, "自动", "自动")
		}
		if iconButton(c, "close", tr("关闭")).Clicked() {
			a.closeToolbar()
		}
	})
}

// Polling observes key/button state only. Selection text is requested after
// an actual selection gesture; no clipboard mutation or document scraping.
func (a *application) monitor() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-a.stop; cancel() }()
	input, err := native.WatchInput(ctx)
	if err != nil {
		a.window.Update(func() { a.status = err.Error() + tr("；仍可使用全局快捷键与手动输入") })
	}
	ticker := time.NewTicker(90 * time.Millisecond)
	defer ticker.Stop()
	var startX, startY int
	var lastClick time.Time
	var pending time.Time
	for {
		select {
		case <-a.stop:
			return
		case e, ok := <-input:
			if !ok {
				input = nil
				continue
			}
			now := time.Now()
			fg := native.Foreground()
			gesture := false
			switch e.Kind {
			case "down":
				startX, startY = e.X, e.Y
			case "up":
				gesture = math.Abs(float64(e.X-startX))+math.Abs(float64(e.Y-startY)) >= 4 || now.Sub(lastClick) < 400*time.Millisecond
				lastClick = now
			case "selection":
				gesture = true
			}
			if gesture && !native.OwnWindow(fg) {
				pending = now.Add(140 * time.Millisecond)
			}
			if e.Kind == "changed" || e.Kind == "selection" {
				mygo.RunOnMain(a.closeToolbar)
			}
			if e.Kind == "down" || e.Kind == "wheel" {
				mygo.RunOnMain(func() {
					if a.toolbar != nil && !native.Contains(a.toolbar.NativeHandle(), e.X, e.Y) {
						a.closeToolbar()
					}
				})
			}
		case now := <-ticker.C:
			fg := native.Foreground()
			checkCapture := !pending.IsZero() && now.After(pending)
			if checkCapture {
				pending = time.Time{}
			}
			mygo.RunOnMain(func() {
				if a.quitting {
					return
				}
				if a.toolbar != nil {
					h := a.toolbar.NativeHandle()
					if fg != a.toolSelection.Owner && !native.Related(fg, h) {
						a.closeToolbar()
					} else {
						native.Clamp(h)
					}
				}
				for _, s := range a.sessions {
					if s.closed {
						continue
					}
					h := s.window.NativeHandle()
					if !s.pinned && !native.Related(fg, h) && now.Sub(s.opened) > 450*time.Millisecond {
						s.window.Close()
						continue
					}
					native.Clamp(h)
				}
				if checkCapture && a.cfg.AutoCapture && !a.paused && !native.OwnWindow(fg) {
					a.capture(false)
				}
			})
		}
	}
}

func formRow(c *ui.Context, label string, content func()) {
	s := uiSpacing.Main
	ui.Row(c).Gap(s.InlineGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, tr(label)).Width(s.LabelWidth).Shrink(0).FontSize(13).TextColor(c.Theme().TextMuted)
		content()
	})
}
func (a *application) saveSettings() {
	timeout, e := strconv.Atoi(strings.TrimSpace(a.timeoutDraft))
	if e != nil {
		a.status = tr("超时需要填写 10–600 的整数")
		return
	}
	a.draft.TimeoutSeconds = timeout
	draft, e := a.parameters.apply(a.draft)
	if e != nil {
		a.status = e.Error()
		return
	}
	a.draft = draft
	old := a.cfg.Shortcut
	if old != a.draft.Shortcut {
		if e := mygo.GlobalShortcut.Register(a.draft.Shortcut, func() { a.capture(true) }); e != nil {
			a.status = tr("快捷键不可用：") + e.Error()
			return
		}
	}
	keyChanged := a.keyDraft != string(a.key)
	var keyErr error
	if keyChanged {
		keyErr = saveKey([]byte(a.keyDraft))
	}
	if e := keyErr; e != nil {
		if old != a.draft.Shortcut {
			mygo.GlobalShortcut.Unregister(a.draft.Shortcut)
		}
		a.status = tr("密钥保存失败：") + e.Error()
		return
	}
	if e := a.store.SaveConfig(a.draft); e != nil {
		var rollbackErr error
		if keyChanged {
			rollbackErr = saveKey(a.key)
		}
		if old != a.draft.Shortcut {
			mygo.GlobalShortcut.Unregister(a.draft.Shortcut)
		}
		a.status = e.Error()
		if rollbackErr != nil {
			a.status += tr("；凭据恢复失败，请重新保存模型配置")
		}
		return
	}
	if old != a.draft.Shortcut {
		mygo.GlobalShortcut.Unregister(old)
	}
	a.cfg = a.draft
	appearance = a.cfg.Appearance
	i18n.Set(a.cfg.UILanguage, native.Locale())
	mygo.App.SetLocale(i18n.Locale())
	a.window.SetTitle(productName())
	a.setupTray()
	a.closeToolbar()
	for _, s := range a.sessions {
		s.window.SetTitle(tr("翻译") + " · " + productName())
		s.window.Invalidate()
	}
	a.key = []byte(a.keyDraft)
	if a.tray != nil {
		a.tray.SetToolTip(productName() + " · " + a.cfg.Shortcut)
	}
	a.status = tr("设置已保存")
	if !a.cfg.AutoCapture {
		a.closeToolbar()
	}
}

func saveKey(key []byte) error {
	if len(key) == 0 {
		return appSecrets.Delete("llm-api-key")
	}
	return appSecrets.Set("llm-api-key", key)
}
func (a *application) testConnection() {
	if len(a.key) == 0 {
		a.status = tr("请先保存 API Key")
		return
	}
	a.status = tr("正在通过真实 API 测试连接……")
	a.busy = true
	cfg, key := a.cfg, string(a.key)
	go func() {
		e := testModel(cfg, key)
		a.window.Update(func() {
			a.busy = false
			if e != nil {
				a.status = tr("连接测试失败：") + e.Error()
			} else {
				a.status = tr("连接成功：模型已返回测试译文")
			}
		})
	}()
}
