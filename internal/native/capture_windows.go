//go:build windows

package native

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/simiooo/squirrel_translate/internal/core"
	"golang.org/x/sys/windows"
)

var ole32 = windows.NewLazySystemDLL("ole32.dll")
var oleaut = windows.NewLazySystemDLL("oleaut32.dll")
var coInit = ole32.NewProc("CoInitializeEx")
var coUninit = ole32.NewProc("CoUninitialize")
var coCreate = ole32.NewProc("CoCreateInstance")
var strLen = oleaut.NewProc("SysStringLen")
var strFree = oleaut.NewProc("SysFreeString")
var arrayAccess = oleaut.NewProc("SafeArrayAccessData")
var arrayUnaccess = oleaut.NewProc("SafeArrayUnaccessData")
var arrayDestroy = oleaut.NewProc("SafeArrayDestroy")
var arrayLower = oleaut.NewProc("SafeArrayGetLBound")
var arrayUpper = oleaut.NewProc("SafeArrayGetUBound")

type comObject struct{ vtable *[96]uintptr }

func call(obj *comObject, index int, args ...uintptr) error {
	if obj == nil {
		return fmt.Errorf("%s", i18n.T("空 COM 对象"))
	}
	fn := obj.vtable[index]
	params := append([]uintptr{uintptr(unsafe.Pointer(obj))}, args...)
	hr, _, _ := syscall.SyscallN(fn, params...)
	runtime.KeepAlive(params)
	runtime.KeepAlive(obj)
	if int32(hr) < 0 {
		return fmt.Errorf("UI Automation 0x%08x", uint32(hr))
	}
	return nil
}
func release(obj *comObject) {
	if obj != nil {
		_ = call(obj, 2)
	}
}
func textOf(r *comObject, maxChars int) string {
	var b *uint16
	if call(r, 12, uintptr(maxChars), uintptr(unsafe.Pointer(&b))) != nil || b == nil {
		return ""
	}
	defer strFree.Call(uintptr(unsafe.Pointer(b)))
	n, _, _ := strLen.Call(uintptr(unsafe.Pointer(b)))
	if n > 100000 {
		return ""
	}
	return syscall.UTF16ToString(unsafe.Slice(b, int(n)))
}

// Capture runs a separate copy of this executable. Killing a stuck provider
// never strands COM calls, locked Go threads or goroutines in the tray process.
func Capture(ctx context.Context) (core.Selection, error) {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	exe, e := os.Executable()
	if e != nil {
		return core.Selection{}, e
	}
	cmd := exec.CommandContext(ctx, exe, "--capture-helper")
	cmd.SysProcAttr = HideConsoleChild()
	b, e := isolatedOutput(cmd)
	if ctx.Err() != nil {
		return core.Selection{}, fmt.Errorf("%s", i18n.T("读取选文超时；请尝试快捷键或剪贴板入口"))
	}
	if e != nil {
		return core.Selection{}, ErrNoSelection()
	}
	var s core.Selection
	if e = json.Unmarshal(b, &s); e != nil {
		return s, e
	}
	if strings.TrimSpace(s.Text) == "" {
		return s, ErrNoSelection()
	}
	return s, nil
}

func isolatedOutput(cmd *exec.Cmd) ([]byte, error) {
	job, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return nil, e
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); e != nil {
		return nil, e
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	if e = cmd.Start(); e != nil {
		return nil, e
	}
	p, e := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if e == nil {
		e = windows.AssignProcessToJobObject(job, p)
		windows.CloseHandle(p)
	}
	if e != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, e
	}
	e = cmd.Wait()
	return output.Bytes(), e
}

func CaptureHelper() int {
	InitDPI()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInit.Call(0, 0)
	if int32(hr) < 0 {
		return 1
	}
	defer coUninit.Call()
	clsid, _ := windows.GUIDFromString("{FF48DBA4-60EF-4201-AA87-54103EEF594E}")
	iid, _ := windows.GUIDFromString("{30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}")
	var automation *comObject
	hr, _, _ = coCreate.Call(uintptr(unsafe.Pointer(&clsid)), 0, 1, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&automation)))
	if int32(hr) < 0 {
		return 1
	}
	defer release(automation)
	owner := Foreground()
	x, y := Cursor()
	var focused, pointed, walker *comObject
	_ = call(automation, 8, uintptr(unsafe.Pointer(&focused)))
	defer release(focused)
	packed := uintptr(uint64(uint32(x)) | uint64(uint32(y))<<32)
	_ = call(automation, 7, packed, uintptr(unsafe.Pointer(&pointed)))
	defer release(pointed)
	_ = call(automation, 14, uintptr(unsafe.Pointer(&walker)))
	defer release(walker)
	for _, start := range []*comObject{focused, pointed} {
		element := start
		for depth := 0; depth < 9 && element != nil; depth++ {
			var password int32
			_ = call(element, 35, uintptr(unsafe.Pointer(&password)))
			if password != 0 {
				if element != start {
					release(element)
				}
				return 1
			}
			var pattern *comObject
			_ = call(element, 16, 10014, uintptr(unsafe.Pointer(&pattern)))
			if pattern != nil {
				s, ok := selectionOf(pattern, x, y)
				release(pattern)
				if ok {
					if element != start {
						release(element)
					}
					if Foreground() != owner {
						return 1
					}
					s.Owner = owner
					s.Source = ProcessName(owner)
					s.Locale = Locale()
					if json.NewEncoder(os.Stdout).Encode(s) != nil {
						return 1
					}
					return 0
				}
			}
			var parent *comObject
			if walker != nil {
				_ = call(walker, 3, uintptr(unsafe.Pointer(element)), uintptr(unsafe.Pointer(&parent)))
			}
			if element != start {
				release(element)
			}
			element = parent
		}
		if element != nil && element != start {
			release(element)
		}
	}
	return 1
}

func selectionOf(pattern *comObject, x, y int) (core.Selection, bool) {
	var ranges *comObject
	if call(pattern, 5, uintptr(unsafe.Pointer(&ranges))) != nil || ranges == nil {
		return core.Selection{}, false
	}
	defer release(ranges)
	var count int32
	_ = call(ranges, 3, uintptr(unsafe.Pointer(&count)))
	if count <= 0 {
		return core.Selection{}, false
	}
	// Disjoint selections are ambiguous. Use the first range consistently,
	// and never expand selection by modifying the source application's UI.
	var r *comObject
	if call(ranges, 4, 0, uintptr(unsafe.Pointer(&r))) != nil || r == nil {
		return core.Selection{}, false
	}
	defer release(r)
	s := core.Selection{Text: strings.TrimSpace(textOf(r, 12000)), Bounds: core.Rect{X: x, Y: y, Width: 1, Height: 1}}
	if s.Text == "" {
		return s, false
	}
	var bounds unsafe.Pointer
	if call(r, 10, uintptr(unsafe.Pointer(&bounds))) == nil && bounds != nil {
		s.Bounds = rangeBounds(bounds, s.Bounds)
		arrayDestroy.Call(uintptr(bounds))
	}
	// Some providers report the entire control for short selected ranges.
	// A pointer anchor is better than putting a tiny word's toolbar at the
	// bottom of a large editor. Accurate provider rectangles stay untouched.
	if len([]rune(s.Text)) < 80 && s.Bounds.Height > 80 {
		s.Bounds = core.Rect{X: x, Y: y, Width: 1, Height: 1}
	}
	var nearby *comObject
	if call(r, 3, uintptr(unsafe.Pointer(&nearby))) == nil && nearby != nil {
		defer release(nearby)
		if call(nearby, 6, 4) == nil {
			s.Context = strings.TrimSpace(textOf(nearby, 2400))
			if s.Context == s.Text {
				s.Context = ""
			}
		}
	}
	return s, true
}
func rangeBounds(array unsafe.Pointer, fallback core.Rect) core.Rect {
	var lo, hi int32
	arrayLower.Call(uintptr(array), 1, uintptr(unsafe.Pointer(&lo)))
	arrayUpper.Call(uintptr(array), 1, uintptr(unsafe.Pointer(&hi)))
	n := int(hi - lo + 1)
	if n < 4 || n > 16384 {
		return fallback
	}
	var p *float64
	hr, _, _ := arrayAccess.Call(uintptr(array), uintptr(unsafe.Pointer(&p)))
	if hr != 0 || p == nil {
		return fallback
	}
	defer arrayUnaccess.Call(uintptr(array))
	v := unsafe.Slice(p, n)
	i := n - 4
	return core.Rect{X: int(v[i]), Y: int(v[i+1]), Width: int(v[i+2]), Height: int(v[i+3])}
}
