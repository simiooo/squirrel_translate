//go:build windows

package native

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	frameControls         = windows.NewLazySystemDLL("comctl32.dll")
	setFrameSubclass      = frameControls.NewProc("SetWindowSubclass")
	removeFrameSubclass   = frameControls.NewProc("RemoveWindowSubclass")
	defaultFrameSubclass  = frameControls.NewProc("DefSubclassProc")
	frameCallback         uintptr
	frameChildrenCallback = syscall.NewCallback(func(child, root uintptr) uintptr {
		setFrameSubclass.Call(child, frameCallback, 1, root)
		return 1
	})
)

func init() { frameCallback = syscall.NewCallback(frameWindowProc) }

// Mygo retains the native sizing style and maximized work-area calculation.
// Its frameless raster surface covers the top resize edge, so forward hits
// there to the parent and supply an invisible, DPI-aware eight-DIP border.
func ConfigureMainFrame(h uintptr) {
	RoundCorners(h)
	setFrameSubclass.Call(h, frameCallback, 1, h)
	user32.NewProc("EnumChildWindows").Call(h, frameChildrenCallback, h)
}

func frameWindowProc(h uintptr, msg uint32, wp, lp, id, root uintptr) uintptr {
	switch msg {
	case 0x0084: // WM_NCHITTEST
		zoomed, _, _ := user32.NewProc("IsZoomed").Call(root)
		if zoomed == 0 {
			var r rect
			procRect.Call(root, uintptr(unsafe.Pointer(&r)))
			dpi, _, _ := user32.NewProc("GetDpiForWindow").Call(root)
			if dpi == 0 {
				dpi = 96
			}
			x, y := int(int16(lp&0xffff)), int(int16((lp>>16)&0xffff))
			hit := frameEdge(x-int(r.Left), y-int(r.Top), int(r.Right-r.Left), int(r.Bottom-r.Top), max(1, int(8*dpi/96)))
			if hit != 0 {
				if h != root {
					return ^uintptr(0)
				} // HTTRANSPARENT to parent.
				return uintptr(hit)
			}
		}
	case 0x0005: // WM_SIZE
		if h == root {
			preference := uint32(2)
			if wp == 2 {
				preference = 1
			} // No outer rounding when maximized.
			windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute").Call(h, 33, uintptr(unsafe.Pointer(&preference)), 4)
		}
	case 0x0082: // WM_NCDESTROY
		removeFrameSubclass.Call(h, frameCallback, id)
	}
	result, _, _ := defaultFrameSubclass.Call(h, uintptr(msg), wp, lp)
	return result
}

func frameEdge(x, y, w, h, edge int) int {
	if x < 0 || y < 0 || x >= w || y >= h {
		return 0
	}
	left, right, top, bottom := x < edge, x >= w-edge, y < edge, y >= h-edge
	switch {
	case top && left:
		return 13
	case top && right:
		return 14
	case bottom && left:
		return 16
	case bottom && right:
		return 17
	case left:
		return 10
	case right:
		return 11
	case top:
		return 12
	case bottom:
		return 15
	default:
		return 0
	}
}
