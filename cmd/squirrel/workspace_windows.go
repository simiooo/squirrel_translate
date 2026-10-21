//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/native"
)

func (a *application) view(c *ui.Context) {
	t := minimalTheme(c)
	width, _ := c.Size()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		a.titleBar(c)
		ui.Row(c).Grow(1).MinHeight(0).Gap(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, width < metric("Layout/CompactBreakpoint"))
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				switch a.tab {
				case 0:
					a.home(c)
				case 1:
					a.vocabulary(c)
				case 2:
					a.settings(c)
				}
				if a.status != "" {
					ui.Text(c, a.status).Label(tr("应用状态")).FontSize(12).TextColor(statusColor(c, a.status)).
						Padding(8, uiSpacing.Main.InsetX).MaxHeight(64).Selectable()
				}
			})
		})
	})
}

func statusColor(c *ui.Context, text string) ui.Color {
	if strings.Contains(text, "失败") || strings.Contains(text, "错误") || strings.Contains(strings.ToLower(text), "failed") || strings.Contains(text, "Could not") {
		return colorToken(c, "Danger")
	}
	if strings.Contains(text, "成功") || strings.HasPrefix(text, "已") || strings.Contains(text, "已保存") || strings.Contains(text, "saved") || strings.HasPrefix(text, "Connected") {
		return colorToken(c, "Success")
	}
	return colorToken(c, "Secondary")
}

func (a *application) titleBar(c *ui.Context) {
	ui.Row(c).Height(metric("Layout/TitleBarHeight")).Shrink(0).Padding(6, metric("Space/8"), 6, metric("Space/20")).Gap(metric("Space/4")).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Grow(1).Height(32).AlignItems(ui.Center).Gap(metric("Space/8")).DragWindow().Label(tr("窗口拖动区")).ContextMenu(func(m *ui.Menu) {
			if m.Item(tr("最小化")).Chosen() && a.window != nil {
				a.window.Minimize()
			}
			if m.Item(tr("最大化 / 还原")).Chosen() && a.window != nil {
				a.window.ToggleMaximize()
			}
			m.Separator()
			if m.Item(tr("隐藏到托盘")).Chosen() && a.window != nil {
				a.window.Close()
			}
		}).Children(func() {
			ui.Image(c, appIconImage).Size(20, 20)
			ui.Text(c, productName()).Font("Segoe UI Variable Text").FontSize(14).Bold()
		})
		if iconButton(c, "minus", tr("最小化")).Clicked() && a.window != nil {
			a.window.Minimize()
		}
		icon := "maximize"
		if a.window != nil && a.window.IsMaximized() {
			icon = "restore"
		}
		if iconButton(c, icon, tr("最大化或还原")).Clicked() && a.window != nil {
			a.window.ToggleMaximize()
		}
		if iconButton(c, "close", tr("隐藏到托盘")).Clicked() && a.window != nil {
			a.window.Close()
		}
	})
}

func (a *application) sidebar(c *ui.Context, compact bool) {
	width := metric("Layout/SidebarWidth")
	if compact {
		width = metric("Layout/CompactSidebarWidth")
	}
	ui.Column(c).Width(width).Shrink(0).Padding(metric("Space/16"), metric("Space/12"), metric("Space/24")).Gap(metric("Space/4")).Background(colorToken(c, "Sidebar")).Children(func() {
		for i, label := range []string{tr("翻译"), tr("词汇表"), tr("设置")} {
			icon := []string{"translate", "book", "settings"}[i]
			b := ui.ButtonBase(c).Label(tr("导航：")+label).Tooltip(tr(label)).Height(metric("Control/NavHeight")).Padding(0, metric("Space/8"), 0, metric("Space/16")).Gap(metric("Space/8")).AlignItems(ui.Center).Justify(ui.Start).Radius(metric("Radius/Control"))
			color := selectedAppearance(c, b, a.tab == i)
			b.TextColor(color).Children(func() {
				ui.Box(c).Size(18, 18).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, icon, color) })
				if !compact {
					ui.Text(c, tr(label)).FontSize(14).SingleLine()
				}
			})
			if b.Clicked() {
				a.tab = i
				a.status = ""
			}
		}
		ui.Spacer(c)
		if !compact {
			label := tr("划词翻译已开启")
			if !a.cfg.AutoCapture || a.paused {
				label = tr("划词翻译已暂停")
			}
			ui.Text(c, tr(label)).FontSize(12).TextColor(colorToken(c, "Secondary")).Padding(0, metric("Space/8"))
			ui.Text(c, a.cfg.Shortcut).Font("Segoe UI Variable Text").FontSize(12).TextColor(colorToken(c, "Muted")).Padding(metric("Space/4"), metric("Space/8"), 0)
		}
	})
}

func pageHeading(c *ui.Context, title string, actions func()) {
	ui.Row(c).MinHeight(metric("Layout/TitleBarHeight")).AlignItems(ui.Center).Gap(metric("Space/8")).Children(func() {
		ui.Text(c, tr(title)).FontSize(metric("Type/Title")).Bold()
		ui.Spacer(c)
		if actions != nil {
			actions()
		}
	})
}

func (a *application) home(c *ui.Context) {
	width, _ := c.Size()
	compact := width < metric("Layout/CompactBreakpoint")
	available := width - metric("Layout/SidebarWidth") - 2*uiSpacing.Main.InsetX
	if compact {
		available = width - metric("Layout/CompactSidebarWidth") - 2*uiSpacing.Main.InsetX
	}
	ui.Column(c).Grow(1).MinHeight(0).Padding(metric("Space/12"), uiSpacing.Main.InsetX, metric("Space/16")).Gap(metric("Space/12")).Children(func() {
		pageHeading(c, tr("翻译"), func() { check(c, &a.demo, tr("演示模式")).Tooltip(tr("本地演示，不请求模型")) })
		ui.Row(c).AlignItems(ui.Center).Gap(metric("Space/8")).Children(func() {
			formSelect(c, &a.mode, core.Modes).Label(tr("翻译模式"))
			ui.Spacer(c)
			if primaryButton(c, tr("翻译"), "arrow").Label(tr("开始翻译")).Disabled(strings.TrimSpace(a.input) == "").Clicked() {
				a.startWorkspace()
			}
		})
		rule(c)
		var panes ui.Element
		if available < metric("Layout/PaneBreakpoint") {
			panes = ui.Column(c).Grow(1).MinHeight(0).Gap(metric("Space/16"))
		} else {
			panes = ui.Row(c).Grow(1).MinHeight(0).Gap(metric("Space/24")).AlignItems(ui.Stretch)
		}
		panes.Children(func() {
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Gap(metric("Space/12")).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, tr("原文 · 自动识别")).FontSize(13).TextColor(colorToken(c, "Secondary"))
					ui.Spacer(c)
					if textButton(c, tr("清空")).Clicked() {
						a.input = ""
					}
				})
				textArea(c, &a.input, tr("输入或粘贴文本")).FontSize(17).Padding(0).Label(tr("待翻译文本")).Grow(1).MinHeight(40)
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					if textButton(c, tr("粘贴")).Label(tr("粘贴原文")).Clicked() {
						a.input = limitText(mygo.Clipboard.ReadText(), 12000)
					}
					ui.Spacer(c)
					ui.Text(c, fmt.Sprintf(tr("%d 字符"), len([]rune(a.input)))).FontSize(12).TextColor(colorToken(c, "Muted"))
				})
			})
			if available >= metric("Layout/PaneBreakpoint") {
				ui.Box(c).Width(1).Background(colorToken(c, "Rule"))
			} else {
				rule(c)
			}
			ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					label := core.Target(a.input, a.target, a.cfg)
					if a.workspace != nil && a.workspace.target != "" {
						label = a.workspace.target // The displayed result owns its language snapshot.
					}
					if plainSelect(c, &a.target, core.Languages, label).Label(tr("目标语言")).Changed() && a.workspace != nil {
						a.startWorkspace()
					}
					ui.Spacer(c)
					if a.workspace != nil && iconButton(c, "refresh", tr("重新翻译")).Clicked() {
						a.startWorkspace()
					}
				})
				if a.workspace == nil {
					ui.Text(c, tr("译文")).FontSize(17).TextColor(colorToken(c, "Muted")).Padding(metric("Space/8"))
					ui.Spacer(c)
				} else {
					a.workspace.availableWidth = available
					if available >= metric("Layout/PaneBreakpoint") {
						a.workspace.availableWidth = (available - 49) / 2
					}
					a.workspace.view(c)
				}
			})
		})
	})
}

func (a *application) startWorkspace() {
	if strings.TrimSpace(a.input) == "" {
		return
	}
	if a.workspace != nil {
		a.workspace.closed = true
		a.workspace.cancel()
	}
	s := &translation{app: a, window: a.window, embedded: true, selection: core.Selection{Text: limitText(a.input, 12000), Locale: native.Locale(), Source: "手动输入"},
		modeChoice: a.mode, targetChoice: a.target, includeContext: a.cfg.IncludeContext, demo: a.demo, cancel: func() {}}
	a.workspace = s
	s.run()
}

func (a *application) settings(c *ui.Context) {
	s := uiSpacing.Main
	ui.Column(c).Grow(1).MinHeight(0).Padding(metric("Space/12"), s.InsetX, metric("Space/16")).Gap(metric("Space/12")).Children(func() {
		pageHeading(c, tr("设置"), nil)
		ui.Row(c).Gap(metric("Space/8")).Children(func() {
			for i, label := range []string{tr("模型"), tr("翻译"), tr("快捷键"), tr("外观")} {
				b := textButton(c, label).Label(tr("设置：") + label)
				if i == a.settingsTab {
					b.TextColor(colorToken(c, "SelectedText")).Background(colorToken(c, "Selected")).Bold()
				}
				if b.Clicked() {
					a.settingsTab = i
				}
			}
		})
		rule(c)
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).MaxWidth(metric("Layout/SettingsMaxWidth")).Gap(s.RowGap).Children(func() {
				switch a.settingsTab {
				case 0:
					sectionTitle(c, tr("连接"))
					formRow(c, tr("接口"), func() { field(c, &a.draft.Endpoint, "https://api.example.com/v1").Label(tr("API 地址")).Grow(1) })
					formRow(c, tr("模型"), func() { field(c, &a.draft.Model, tr("模型名称")).Label(tr("模型名称")).Grow(1) })
					formRow(c, "API Key", func() { field(c, &a.keyDraft, "").Password().Label("API Key").Grow(1) })
					sectionTitle(c, tr("生成"))
					a.modelSettings(c)
					disclosure(c, tr("连接选项"), &a.advanced).AlignSelf(ui.Start)
					if a.advanced {
						formRow(c, tr("超时（秒）"), func() { field(c, &a.timeoutDraft, "90").Label(tr("请求超时")).Grow(1) })
						formRow(c, tr("输出格式"), func() {
							formSelect(c, &a.draft.StructuredMode, []string{"prompt", "json_object", "json_schema"}).Label(tr("结构化输出")).Grow(1)
						})
					}
				case 1:
					sectionTitle(c, tr("语言"))
					formRow(c, tr("首选语言"), func() { formSelect(c, &a.draft.PrimaryLanguage, core.Languages[1:]).Label(tr("首选语言")).Grow(1) })
					formRow(c, tr("第二语言"), func() {
						formSelect(c, &a.draft.SecondaryLanguage, core.Languages[1:]).Label(tr("第二语言")).Grow(1)
					})
					sectionTitle(c, tr("划词"))
					check(c, &a.draft.AutoCapture, tr("自动划词"))
					check(c, &a.draft.IncludeContext, tr("附带上下文"))
					formRow(c, tr("排除应用"), func() { field(c, &a.draft.ExcludedApps, "app.exe, other.exe").Label(tr("排除应用")).Grow(1) })
				case 2:
					sectionTitle(c, tr("快捷键与启动"))
					formRow(c, tr("划词翻译"), func() { field(c, &a.draft.Shortcut, "Ctrl+Alt+T").Label(tr("全局快捷键")).Grow(1) })
					if check(c, &a.openAtLogin, tr("开机启动")).Changed() {
						if err := mygo.App.SetOpenAtLogin(a.openAtLogin); err != nil {
							a.openAtLogin = !a.openAtLogin
							a.status = tr("开机启动设置失败：") + err.Error()
						}
					}
				case 3:
					formRow(c, tr("界面语言"), func() {
						formSelect(c, &a.draft.UILanguage, []string{"system", "zh-CN", "en"}).Label(tr("界面语言")).Grow(1)
					})
					ui.Text(c, tr("保存后应用界面语言；不影响翻译目标语言")).FontSize(12).TextColor(colorToken(c, "Muted"))
					sectionTitle(c, tr("主题"))
					formRow(c, tr("外观"), func() {
						formSelect(c, &a.draft.Appearance, []string{"跟随系统", "浅色", "深色"}).Label(tr("应用主题")).Grow(1)
					})
					ui.Text(c, tr("窗口与浮层使用同一主题")).FontSize(12).TextColor(colorToken(c, "Muted"))
				}
			})
		})
		rule(c)
		ui.Row(c).Gap(metric("Space/8")).Children(func() {
			if primaryButton(c, tr("保存")).Label(tr("保存设置")).Clicked() {
				a.saveSettings()
			}
			if a.settingsTab == 0 && textButton(c, tr("测试连接")).Tooltip(tr("使用已保存配置发送测试请求")).Disabled(a.busy).Clicked() {
				a.testConnection()
			}
		})
	})
}

func (a *application) vocabulary(c *ui.Context) {
	w, _ := c.Size()
	compact := w < metric("Layout/CompactBreakpoint")
	ui.Column(c).Grow(1).MinHeight(0).Padding(metric("Space/12"), uiSpacing.Main.InsetX, metric("Space/16")).Gap(metric("Space/12")).Children(func() {
		pageHeading(c, tr("词汇表"), func() {
			ui.Text(c, fmt.Sprintf(tr("%d 条"), len(a.entries))).FontSize(12).TextColor(colorToken(c, "Muted"))
			if textButton(c, tr("导出")).Label(tr("导出 JSON")).Clicked() {
				p := filepath.Join(a.store.Dir, "vocabulary-export.json")
				if err := a.store.Export(p); err != nil {
					a.status = err.Error()
				} else {
					a.status = tr("已导出：") + p
				}
			}
		})
		field(c, &a.search, tr("搜索词汇")).Label(tr("搜索词汇")).MaxWidth(metric("Layout/SearchWidth"))
		rule(c)
		var selected *core.Entry
		for i := range a.entries {
			if a.entries[i].ID == a.selectedEntry {
				selected = &a.entries[i]
				break
			}
		}
		if selected == nil {
			a.vocabDetail = false
		}
		ui.Row(c).Grow(1).MinHeight(0).Gap(metric("Space/24")).AlignItems(ui.Stretch).Children(func() {
			if !compact || !a.vocabDetail {
				list := ui.Scroll(c).Grow(1).MinHeight(0).Gap(metric("Space/4"))
				if !compact {
					list.Width(metric("Layout/VocabularyListWidth")).Basis(metric("Layout/VocabularyListWidth")).MinWidth(metric("Layout/VocabularyListWidth")).Grow(0).Shrink(0)
				}
				list.Children(func() {
					count := 0
					for _, e := range a.entries {
						hay := e.Word + " " + e.Note
						for _, sense := range e.Definition.Senses {
							hay += " " + sense.Meaning
						}
						if !strings.Contains(strings.ToLower(hay), strings.ToLower(a.search)) {
							continue
						}
						count++
						b := ui.ButtonBase(c).Key(e.ID).Label(tr("词条：")+e.Word).Padding(metric("Space/12"), metric("Space/12"), metric("Space/12"), metric("Space/20")).Radius(metric("Radius/Control")).Justify(ui.Start)
						color := selectedAppearance(c, b, selected != nil && e.ID == selected.ID)
						secondary := colorToken(c, "Secondary")
						if selected != nil && e.ID == selected.ID {
							secondary = colorToken(c, "SelectedSecondary")
						}
						b.Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Gap(metric("Space/4")).Children(func() {
								ui.Text(c, e.Word).FontSize(16).TextColor(color).SingleLine()
								if len(e.Definition.Senses) > 0 {
									ui.Text(c, e.Definition.Senses[0].Meaning).FontSize(12).FontWeight(400).TextColor(secondary).SingleLine()
								}
							})
						})
						if b.Clicked() {
							a.selectedEntry = e.ID
							a.vocabDetail = true
						}
					}
					if count == 0 {
						ui.Text(c, tr("暂无词汇")).FontSize(13).TextColor(colorToken(c, "Muted")).Padding(metric("Space/16"), metric("Space/8"))
					}
				})
			}
			if !compact || a.vocabDetail {
				if !compact {
					ui.Box(c).Width(1).Background(colorToken(c, "Rule"))
				}
				ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Gap(metric("Space/12")).Children(func() {
					if compact && iconButton(c, "back", tr("返回词汇列表")).Clicked() {
						a.vocabDetail = false
					}
					if selected == nil {
						ui.Text(c, tr("选择一个词条")).FontSize(14).TextColor(colorToken(c, "Muted")).Padding(metric("Space/16"), 0)
						return
					}
					e := *selected
					ui.Scroll(c).Grow(1).MinHeight(0).Gap(metric("Space/16")).Children(func() {
						ui.Text(c, e.Word).FontSize(28).Bold().Selectable()
						ui.Text(c, tr(core.ModeLabel(e.LexicalKind()))+" · "+e.SourceLanguage+" → "+e.TargetLanguage).FontSize(12).TextColor(colorToken(c, "Secondary"))
						for _, sense := range e.Definition.Senses {
							ui.Column(c).Gap(metric("Space/4")).Children(func() {
								ui.Text(c, sense.PartOfSpeech).FontSize(12).TextColor(colorToken(c, "Muted"))
								ui.Text(c, sense.Meaning).FontSize(18).Selectable()
							})
						}
						if e.Context != "" {
							ui.Text(c, e.Context).FontSize(13).TextColor(colorToken(c, "Secondary")).Selectable()
						}
						if e.Note != "" {
							ui.Text(c, e.Note).FontSize(13).TextColor(colorToken(c, "Secondary")).Selectable()
						}
					})
					ui.Row(c).Gap(metric("Space/8")).Children(func() {
						if textButton(c, tr("复制"), "copy").Clicked() {
							s := translation{dictionary: &e.Definition}
							mygo.Clipboard.WriteText(s.copyText())
							a.status = tr("已复制")
						}
						if textButton(c, tr("浮窗查看")).Clicked() {
							a.openStored(e)
						}
						ui.Spacer(c)
						if dangerButton(c, tr("移除")).Label(tr("移除词条")).Clicked() {
							if err := a.store.Remove(e.ID); err != nil {
								a.status = tr("移除失败：") + err.Error()
							} else {
								a.entries, _ = a.store.List()
								a.selectedEntry = ""
								a.vocabDetail = false
								a.status = tr("已从词汇表移除")
								for _, session := range a.sessions {
									if session.window != nil {
										session.window.Invalidate()
									}
								}
							}
						}
					})
				})
			}
		})
	})
}
