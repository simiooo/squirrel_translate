//go:build windows

package native

import (
	"fmt"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/simiooo/squirrel_translate/internal/core"
	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var procForeground = user32.NewProc("GetForegroundWindow")
var procPID = user32.NewProc("GetWindowThreadProcessId")
var procCursor = user32.NewProc("GetCursorPos")
var procKey = user32.NewProc("GetAsyncKeyState")
var procAncestor = user32.NewProc("GetAncestor")
var procOwner = user32.NewProc("GetWindow")
var procRect = user32.NewProc("GetWindowRect")
var procMonitor = user32.NewProc("MonitorFromPoint")
var procInfo = user32.NewProc("GetMonitorInfoW")
var procPosition = user32.NewProc("SetWindowPos")
var procStyle = user32.NewProc("SetWindowLongPtrW")
var procGetStyle = user32.NewProc("GetWindowLongPtrW")
var procLocale = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
var procDPI = windows.NewLazySystemDLL("shcore.dll").NewProc("GetDpiForMonitor")

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type monitorInfo struct {
	Size          uint32
	Monitor, Work rect
	Flags         uint32
}

func Foreground() uintptr { h, _, _ := procForeground.Call(); return h }
func ProcessID(h uintptr) uint32 {
	var pid uint32
	procPID.Call(h, uintptr(unsafe.Pointer(&pid)))
	return pid
}
func OwnWindow(h uintptr) bool { return h != 0 && ProcessID(h) == uint32(os.Getpid()) }
func KeyDown(k int) bool       { r, _, _ := procKey.Call(uintptr(k)); return uint16(r)&0x8000 != 0 }
func Cursor() (int, int) {
	var p point
	procCursor.Call(uintptr(unsafe.Pointer(&p)))
	return int(p.X), int(p.Y)
}
func Locale() string {
	var b [85]uint16
	procLocale.Call(uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return windows.UTF16ToString(b[:])
}
func ProcessName(h uintptr) string {
	pid := ProcessID(h)
	p, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if e != nil {
		return ""
	}
	defer windows.CloseHandle(p)
	var b [1024]uint16
	n := uint32(len(b))
	if windows.QueryFullProcessImageName(p, 0, &b[0], &n) != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(b[:n]))
}
func Excluded(h uintptr, list string) bool {
	p := strings.ToLower(ProcessName(h))
	for _, item := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		if p == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
func Contains(h uintptr, x, y int) bool {
	var r rect
	if h == 0 {
		return false
	}
	ok, _, _ := procRect.Call(h, uintptr(unsafe.Pointer(&r)))
	return ok != 0 && x >= int(r.Left) && x < int(r.Right) && y >= int(r.Top) && y < int(r.Bottom)
}
func Related(h, root uintptr) bool {
	// Panels share MyGo's hidden application owner. Equality of root owners
	// would incorrectly make every translation part of the same focus group.
	for i := 0; h != 0 && i < 12; i++ {
		if h == root {
			return true
		}
		parent, _, _ := procAncestor.Call(h, 2)
		if parent == root {
			return true
		}
		h, _, _ = procOwner.Call(h, 4)
	}
	return false
}

func InitDPI() {
	p := user32.NewProc("SetProcessDpiAwarenessContext")
	if p.Find() == nil {
		p.Call(^uintptr(3))
	}
}

func WorkArea(x, y int) (core.Rect, float64) {
	packed := uintptr(uint64(uint32(x)) | uint64(uint32(y))<<32)
	m, _, _ := procMonitor.Call(packed, 2)
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	procInfo.Call(m, uintptr(unsafe.Pointer(&info)))
	scale := 1.0
	var dx, dy uint32
	if procDPI.Find() == nil {
		hr, _, _ := procDPI.Call(m, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy)))
		if hr == 0 && dx > 0 {
			scale = float64(dx) / 96
		}
	}
	r := info.Work
	return core.Rect{X: int(r.Left), Y: int(r.Top), Width: int(r.Right - r.Left), Height: int(r.Bottom - r.Top)}, scale
}
func Place(h uintptr, anchor core.Rect, width, height int) {
	work, scale := WorkArea(anchor.X, anchor.Y)
	r := core.Place(anchor, work, int(float64(width)*scale), int(float64(height)*scale), int(8*scale))
	SetBounds(h, r)
}
func SetBounds(h uintptr, r core.Rect) {
	procPosition.Call(h, 0, uintptr(r.X), uintptr(r.Y), uintptr(r.Width), uintptr(r.Height), 0x0010|0x0004)
}
func Clamp(h uintptr) {
	var r rect
	if h == 0 {
		return
	}
	procRect.Call(h, uintptr(unsafe.Pointer(&r)))
	work, _ := WorkArea(int(r.Left), int(r.Top))
	w := min(int(r.Right-r.Left), work.Width)
	ht := min(int(r.Bottom-r.Top), work.Height)
	x := max(work.X, min(int(r.Left), work.X+work.Width-w))
	y := max(work.Y, min(int(r.Top), work.Y+work.Height-ht))
	if x != int(r.Left) || y != int(r.Top) || w != int(r.Right-r.Left) || ht != int(r.Bottom-r.Top) {
		SetBounds(h, core.Rect{X: x, Y: y, Width: w, Height: ht})
	}
}

// Prevent automatic toolbar activation, including on click. Translation
// windows remain normal keyboard-accessible top-level tool windows.
func NoActivate(h uintptr) {
	idx := ^uintptr(19)
	style, _, _ := procGetStyle.Call(h, idx)
	procStyle.Call(h, idx, style|0x08000000|0x00000080)
}

// Let Windows 11 draw the rounded silhouette and its thin outer border.
// The system owns physical-pixel sizing, active/inactive colors and maximized
// edges; painting another rounded rectangle in content would double the edge.
// Older versions ignore these optional DWM attributes.
func RoundCorners(h uintptr) {
	preference := uint32(2) // DWMWCP_ROUND
	set := windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	set.Call(h, 33, uintptr(unsafe.Pointer(&preference)), unsafe.Sizeof(preference))
	border := uint32(0xffffffff) // DWMWA_COLOR_DEFAULT, not COLOR_NONE.
	set.Call(h, 34, uintptr(unsafe.Pointer(&border)), unsafe.Sizeof(border))
}

func HideConsoleChild() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
func ErrNoSelection() error {
	return fmt.Errorf("%s", i18n.T("当前应用没有提供可读取的选文；可复制后从托盘选择“翻译剪贴板”，或手动输入"))
}
