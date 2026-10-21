//go:build windows

package native

import (
	"context"
	"fmt"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"golang.org/x/sys/windows"
	"runtime"
	"syscall"
	"unsafe"
)

type InputEvent struct {
	Kind string
	X, Y int
}
type hookMouse struct {
	Position          point
	Data, Flags, Time uint32
	Extra             uintptr
}
type hookKey struct {
	Key, Scan, Flags, Time uint32
	Extra                  uintptr
}
type message struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Position       point
	Private        uint32
}

// Hooks deliver gestures rather than text or a key log. Callbacks never wait
// on application work. Their message loop lives on one dedicated OS thread.
func WatchInput(ctx context.Context) (<-chan InputEvent, error) {
	events := make(chan InputEvent, 64)
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(events)
		setHook := user32.NewProc("SetWindowsHookExW")
		unhook := user32.NewProc("UnhookWindowsHookEx")
		next := user32.NewProc("CallNextHookEx")
		getMessage := user32.NewProc("GetMessageW")
		postThread := user32.NewProc("PostThreadMessageW")
		emit := func(e InputEvent) {
			select {
			case events <- e:
			default:
			}
		}
		mouseCallback := syscall.NewCallback(func(code int32, kind uintptr, data unsafe.Pointer) uintptr {
			if code >= 0 && data != nil {
				m := (*hookMouse)(data)
				e := InputEvent{X: int(m.Position.X), Y: int(m.Position.Y)}
				switch kind {
				case 0x0201:
					e.Kind = "down"
				case 0x0202:
					e.Kind = "up"
				case 0x020A, 0x020E:
					e.Kind = "wheel"
				}
				if e.Kind != "" {
					emit(e)
				}
			}
			r, _, _ := next.Call(0, uintptr(code), kind, uintptr(data))
			return r
		})
		keyCallback := syscall.NewCallback(func(code int32, kind uintptr, data unsafe.Pointer) uintptr {
			if code >= 0 && data != nil && (kind == 0x0101 || kind == 0x0105) {
				k := (*hookKey)(data)
				if k.Key == 16 || k.Key == 160 || k.Key == 161 || (k.Key == 65 && KeyDown(17)) {
					x, y := Cursor()
					emit(InputEvent{Kind: "selection", X: x, Y: y})
				} else if (k.Key >= 0x21 && k.Key <= 0x28) && KeyDown(16) {
					x, y := Cursor()
					emit(InputEvent{Kind: "selection", X: x, Y: y})
				} else if k.Key == 8 || k.Key == 13 || k.Key == 27 || (k.Key >= 0x20 && k.Key <= 0x6F) {
					emit(InputEvent{Kind: "changed"})
				}
			}
			r, _, _ := next.Call(0, uintptr(code), kind, uintptr(data))
			return r
		})
		mouse, _, e := setHook.Call(14, mouseCallback, 0, 0)
		if mouse == 0 {
			ready <- fmt.Errorf(i18n.T("鼠标选择监听不可用：%w"), e)
			return
		}
		defer unhook.Call(mouse)
		key, _, e := setHook.Call(13, keyCallback, 0, 0)
		if key == 0 {
			ready <- fmt.Errorf(i18n.T("键盘选择监听不可用：%w"), e)
			return
		}
		defer unhook.Call(key)
		id := windows.GetCurrentThreadId()
		var msg message
		user32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
		go func() { <-ctx.Done(); postThread.Call(uintptr(id), 0x0012, 0, 0) }()
		ready <- nil
		for {
			r, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
		}
	}()
	if e := <-ready; e != nil {
		return nil, e
	}
	return events, nil
}
