//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"github.com/simiooo/squirrel_translate/internal/native"
	"github.com/simiooo/squirrel_translate/internal/store"
	"github.com/simiooo/squirrel_translate/internal/update"
	"golang.org/x/sys/windows"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--apply-update" {
		if err := update.RunHelper(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			var recovery *update.RecoveryError
			if errors.As(err, &recovery) {
				message, _ := windows.UTF16PtrFromString("更新失败，请保留以下目录以恢复文件。\nUpdate failed. Keep this folder for recovery.\n\n" + recovery.Work + "\n\n" + recovery.Reason)
				caption, _ := windows.UTF16PtrFromString("松鼠翻译 / Squirrel Translate")
				_, _ = windows.MessageBox(0, message, caption, windows.MB_OK|windows.MB_ICONERROR)
			}
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "--capture-helper" {
		os.Exit(native.CaptureHelper())
	}
	demo := flag.Bool("demo", false, tr("使用不联网的本地演示"))
	hidden := flag.Bool("tray", false, tr("仅启动到托盘"))
	data := flag.String("data-dir", "", tr("独立数据目录，便于测试"))
	spacingFile := flag.String("spacing-file", "", tr("加载 JSON 间距主题，便于界面调试"))
	uiLanguage := flag.String("lang", "", "UI language override: system, zh-CN or en")
	cleanup := flag.String("update-cleanup", "", "Internal update cleanup directory")
	flag.Parse()
	if *uiLanguage != "" && *uiLanguage != "system" && *uiLanguage != "zh-CN" && *uiLanguage != "en" {
		log.Fatal("Invalid --lang; use system, zh-CN or en")
	}
	if err := configureSpacing(*spacingFile); err != nil {
		log.Fatal(err)
	}
	dir := *data
	if dir == "" {
		base, e := os.UserConfigDir()
		if e != nil {
			log.Fatal(e)
		}
		dir = filepath.Join(base, "SquirrelTranslate")
	}
	st, e := store.New(dir)
	if e != nil {
		log.Fatal(e)
	}
	cfg, e := st.LoadConfig()
	if e != nil {
		log.Fatal(e)
	}
	if cfg.Appearance == "" {
		cfg.Appearance = "跟随系统"
	}
	appearance = cfg.Appearance
	if cfg.UILanguage == "" {
		cfg.UILanguage = "system"
	}
	if *uiLanguage != "" {
		cfg.UILanguage = *uiLanguage
	}
	i18n.Set(cfg.UILanguage, native.Locale())
	if _, err := os.Stat(filepath.Join(dir, "settings.json")); os.IsNotExist(err) {
		cfg.PrimaryLanguage = core.LanguageForLocale(native.Locale())
		cfg.SecondaryLanguage = "English"
		if cfg.PrimaryLanguage == "English" {
			cfg.SecondaryLanguage = "简体中文"
		}
	}
	a := &application{store: st, cfg: cfg, draft: cfg, demo: *demo, mode: "自动", target: "自动", sessions: map[uint64]*translation{}, stop: make(chan struct{})}
	a.timeoutDraft = fmt.Sprint(cfg.TimeoutSeconds)
	a.parameters = parameterDraft(cfg)
	mygo.App.SetName("squirrel_translate") // Stable system identity, independent of UI language.
	mygo.App.SetLanguages("en", "zh-CN")
	mygo.App.SetLocale(i18n.Locale())
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	mygo.App.OnSecondInstance(func(_ []string, _ string) {
		if a.window != nil {
			a.window.Show()
		}
	})
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		a.quitting = true
		if a.updates.cancel != nil {
			a.updates.cancel()
		}
		select {
		case <-a.stop:
		default:
			close(a.stop)
		}
		for _, s := range a.sessions {
			s.cancel()
		}
		if a.workspace != nil {
			a.workspace.closed = true
			a.workspace.cancel()
		}
	})
	mygo.App.WhenReady(func() {
		a.window = mygo.NewWindow(mygo.WindowOptions{Title: productName(), Width: int(metric("Layout/MainWidth")), Height: int(metric("Layout/MainHeight")), MinWidth: int(metric("Layout/MinWidth")), MinHeight: int(metric("Layout/MinHeight")), Frameless: true, Hidden: *hidden || mygo.App.WasOpenedAtLogin(), Content: ui.View(a.view)})
		_ = a.window.SetIcon(appIconPNG)
		native.ConfigureMainFrame(a.window.NativeHandle())
		a.openAtLogin = mygo.App.OpenAtLogin()
		a.window.OnClose(func(e *mygo.CloseEvent) {
			if !a.quitting {
				e.PreventDefault()
				if a.tray == nil {
					a.quitting = true
					mygo.App.Quit()
					return
				}
				a.window.Hide()
				if !a.trayHintShown {
					a.trayHintShown = true
					n := mygo.NewNotification(mygo.NotificationOptions{Title: tr("松鼠翻译仍在托盘运行"), Body: tr("点击托盘图标打开窗口，右键菜单可退出。"), Silent: true})
					n.OnClick(func() { a.window.Show() })
					_ = n.Show()
				}
			}
		})
		a.entries, e = st.List()
		if e != nil {
			a.status = e.Error()
		}
		a.key, _ = appSecrets.Get("llm-api-key")
		a.keyDraft = string(a.key)
		a.setupTray()
		a.registerShortcut("")
		a.initUpdates(*cleanup)
		go a.monitor()
	})
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go func() { <-ctx.Done(); mygo.App.Quit() }()
	if e = mygo.App.Run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

// Keep the prototype's credential namespace across the package identity change.
var appSecrets = mygo.Secrets.Service("Squirrel Translate")

type application struct {
	window                                      *mygo.Window
	tray                                        *mygo.Tray
	store                                       *store.Store
	cfg, draft                                  core.Config
	key                                         []byte
	keyDraft                                    string
	timeoutDraft                                string
	parameters                                  modelParameterDraft
	moreModelParameters                         bool
	openAtLogin                                 bool
	advanced                                    bool
	demo, quitting, paused, busy                bool
	tab                                         int
	input, mode, target, status, search, lookup string
	entries                                     []core.Entry
	workspace                                   *translation
	settingsTab                                 int
	selectedEntry                               string
	vocabDetail                                 bool
	trayHintShown                               bool
	sessions                                    map[uint64]*translation
	next                                        uint64
	toolbar                                     *mygo.Window
	toolSelection                               core.Selection
	stop                                        chan struct{}
	updates                                     updateUI
}
