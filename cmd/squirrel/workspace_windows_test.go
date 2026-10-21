//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/native"
	"golang.org/x/sys/windows"
)

func TestWorkspaceNavigationAndSettingsRemainReachable(t *testing.T) {
	a := &application{cfg: core.DefaultConfig(), draft: core.DefaultConfig(), mode: "自动", target: "自动", timeoutDraft: "90"}
	tt := ui.NewTester(a.view, 1040, 704)
	for _, size := range [][2]int{{1040, 704}, {640, 480}} {
		tt.SetSize(size[0], size[1])
		for _, label := range []string{"导航：设置", "最小化", "最大化或还原", "隐藏到托盘", "开始翻译", "目标语言"} {
			r, ok := tt.Find(label)
			if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(size[0])+1 || r.Y+r.H > float32(size[1])+1 {
				t.Fatalf("%s outside %v: %+v", label, size, r)
			}
		}
		if err := tt.Click("导航：设置"); err != nil {
			t.Fatal(err)
		}
		for _, group := range []struct{ tab, field string }{{"模型", "模型名称"}, {"翻译", "首选语言"}, {"快捷键", "全局快捷键"}, {"外观", "应用主题"}} {
			if err := tt.Click("设置：" + group.tab); err != nil {
				t.Fatal(err)
			}
			if r, ok := tt.Find(group.field); !ok || r.W < 32 || r.Y+r.H > float32(size[1]) {
				t.Fatalf("unreachable %s: %+v", group.field, r)
			}
		}
		_ = tt.Click("导航：翻译")
	}
}

func TestVocabularyListDetailAndNarrowBack(t *testing.T) {
	a := &application{tab: 1, cfg: core.DefaultConfig(), entries: []core.Entry{
		{ID: "bank", Word: "bank", Definition: core.Dictionary{Senses: []core.Sense{{PartOfSpeech: "n.", Meaning: "银行"}}}},
		{ID: "phrase", Word: "take care", Kind: "phrase", Definition: core.Dictionary{Senses: []core.Sense{{PartOfSpeech: "短语", Meaning: "保重"}}}},
	}}
	tt := ui.NewTester(a.view, 1040, 704)
	r, ok := tt.Find("词条：bank")
	if !ok || r.W < 180 {
		t.Fatalf("list was squeezed: %+v", r)
	}
	if err := tt.Click("词条：take care"); err != nil {
		t.Fatal(err)
	}
	if a.selectedEntry != "phrase" {
		t.Fatal("selection did not reach detail")
	}
	tt.SetSize(640, 480)
	if _, ok := tt.Find("移除词条"); !ok {
		t.Fatal("narrow detail missing")
	}
	if err := tt.Click("返回词汇列表"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("词条：bank"); !ok {
		t.Fatal("back did not restore list")
	}
}

// A separate process owns hidden native windows, without a tray, global
// shortcuts, user files, credentials or network requests.
func TestMain(m *testing.M) {
	if os.Getenv("SQUIRREL_FOCUS_PROBE") == "1" {
		if err := nativeTranslationFocusProbe(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if os.Getenv("SQUIRREL_WINDOW_PROBE") == "1" {
		if err := nativeWorkspaceProbe(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func TestNativeWorkspaceStreamAndFrame(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0])
	cmd.Env = append(os.Environ(), "SQUIRREL_WINDOW_PROBE=1")
	cmd.SysProcAttr = native.HideConsoleChild()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native window probe: %v\n%s", err, out)
	}
}
func nativeWorkspaceProbe() error {
	var failure error
	a := &application{cfg: core.DefaultConfig(), draft: core.DefaultConfig(), mode: "自动", target: "自动", demo: true, input: "take into account"}
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		a.window = mygo.NewWindow(mygo.WindowOptions{Title: "Squirrel UI verification", Width: 1040, Height: 704, Frameless: true, Hidden: true, Content: ui.View(a.view)})
		native.ConfigureMainFrame(a.window.NativeHandle())
		// Windows 11 supports DWM's native rounded outline. Older systems
		// reject the attribute, so only assert when the query is supported.
		var border uint32
		hr, _, _ := windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute").Call(
			a.window.NativeHandle(), 34, uintptr(unsafe.Pointer(&border)), 4)
		if hr == 0 && border == 0xfffffffe {
			failure = fmt.Errorf("native window outline is explicitly disabled")
		}
		var rect struct{ Left, Top, Right, Bottom int32 }
		u := windows.NewLazySystemDLL("user32.dll")
		u.NewProc("GetWindowRect").Call(a.window.NativeHandle(), uintptr(unsafe.Pointer(&rect)))
		x, y := int(rect.Left+(rect.Right-rect.Left)/2), int(rect.Top+1)
		lp := uintptr(uint32(uint16(x)) | uint32(uint16(y))<<16)
		hit, _, _ := u.NewProc("SendMessageW").Call(a.window.NativeHandle(), 0x0084, 0, lp)
		if hit != 12 {
			failure = fmt.Errorf("top resize border returned %d", hit)
		}
		a.startWorkspace()
		old := a.workspace
		a.input = "bank"
		a.startWorkspace()
		if !old.closed {
			failure = fmt.Errorf("replacement did not release previous request")
		}
		go func() {
			time.Sleep(3 * time.Second)
			a.window.Update(func() {
				s := a.workspace
				if !s.done || s.dictionary == nil || s.dictionary.Word != "bank" || s.err != "" {
					failure = fmt.Errorf("embedded stream incomplete: done=%v mode=%s err=%s", s.done, s.mode, s.err)
				}
				a.tab = 1
				a.window.Invalidate()
				a.tab = 0
				if a.workspace != s {
					failure = fmt.Errorf("navigation lost the session")
				}
				s.closed = true
				s.cancel()
				a.window.Destroy()
				mygo.App.Quit()
			})
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	return failure
}
