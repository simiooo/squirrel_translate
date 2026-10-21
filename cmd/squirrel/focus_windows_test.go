//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/native"
)

// Opt in because this regression check activates real desktop windows.
func TestNativeTranslationFocus(t *testing.T) {
	if os.Getenv("SQUIRREL_TEST_FOCUS") != "1" {
		t.Skip("set SQUIRREL_TEST_FOCUS=1 on an interactive Windows desktop")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0])
	cmd.Env = append(os.Environ(), "SQUIRREL_FOCUS_PROBE=1")
	cmd.SysProcAttr = native.HideConsoleChild()
	cmd.SysProcAttr.HideWindow = false // The probe's GUI must be visible.
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native focus probe: %v\n%s", err, out)
	} else {
		t.Log(string(out))
	}
}

func nativeTranslationFocusProbe() error {
	var failure error
	a := &application{cfg: core.DefaultConfig(), sessions: map[uint64]*translation{}, stop: make(chan struct{})}
	a.cfg.AutoCapture = false
	var source, other *mygo.Window
	var current *translation
	canceled := false
	check := func(ok bool, message string) {
		if !ok && failure == nil {
			failure = fmt.Errorf("%s", message)
		}
	}
	open := func() {
		canceled = false
		current = a.newTranslation(core.Selection{Text: "focus verification", Owner: source.NativeHandle(), Bounds: core.Rect{X: 650, Y: 240}}, "自动", "自动", false)
		current.cancel = func() { canceled = true }
		if os.Getenv("SQUIRREL_FOCUS_INTERACTIVE") == "1" {
			fmt.Printf("opened: source=%d foreground=%d translation=%d\n", source.NativeHandle(), native.Foreground(), current.window.NativeHandle())
			current.window.OnFocus(func() { fmt.Println("translation activated") })
			current.window.OnClosed(func() { fmt.Printf("translation closed: canceled=%v sessions=%d\n", canceled, len(a.sessions)) })
		}
	}
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		source = mygo.NewWindow(mygo.WindowOptions{Title: "Squirrel focus test source", Width: 420, Height: 240, Hidden: true, Content: ui.View(func(c *ui.Context) {
			ui.Text(c, "Focus verification source")
			if ui.Button(c, "Open translation").Clicked() {
				open()
			}
		})})
		other = mygo.NewWindow(mygo.WindowOptions{Title: "Squirrel focus test other", Width: 300, Height: 200, Hidden: true})
		a.window = source
		source.Show()
		fmt.Printf("source=%d foreground=%d visible=%v\n", source.NativeHandle(), native.Foreground(), source.IsVisible())
		go a.monitor()
		if os.Getenv("SQUIRREL_FOCUS_INTERACTIVE") == "1" {
			go func() {
				time.Sleep(120 * time.Second)
				mygo.RunOnMain(func() { close(a.stop); mygo.App.Quit() })
			}()
			return
		}
		steps := []func(){
			func() {
				check(native.Foreground() == source.NativeHandle(), "source could not acquire foreground")
				open()
				check(native.Foreground() == source.NativeHandle(), "translation stole focus on show")
			},
			func() {
				check(!current.closed && current.window.IsVisible(), "inactive translation closed after polling")
				check(!current.activated, "inactive translation was marked activated")
				current.window.Focus()
			},
			func() {
				check(native.Foreground() == current.window.NativeHandle() && current.activated, "translation did not activate")
				source.Focus()
			},
			func() {
				check(current.closed && canceled && len(a.sessions) == 0 && current.selection.Text == "", "focus loss did not close/cancel/release translation")
				open()
				other.Show()
			},
			func() {
				check(current.closed, "switching away before activation did not close translation")
				source.Focus()
				open()
				current.pinned = true
				current.window.Focus()
				other.Focus()
			},
			func() {
				check(!current.closed && current.window.IsVisible(), "pinned translation closed on focus loss")
				current.window.Close()
				close(a.stop)
				other.Destroy()
				source.Destroy()
				mygo.App.Quit()
			},
		}
		go func() {
			// Windows may deny foreground activation to a test process launched
			// in the background. Allow the interactive tester to select it first.
			for i := 0; i < 100 && native.Foreground() != source.NativeHandle(); i++ {
				time.Sleep(100 * time.Millisecond)
			}
			for _, step := range steps {
				time.Sleep(700 * time.Millisecond)
				mygo.RunOnMain(step)
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return err
	}
	if failure == nil && os.Getenv("SQUIRREL_FOCUS_INTERACTIVE") != "1" {
		fmt.Println("PASS: inactive show, delayed polling, activation, blur close/cancel/release, pre-activation switch, pinned retention")
	}
	return failure
}
