//go:build windows

package native

import (
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// Exercise the actual COM calls and ownership, including the shell's object
// selection that used to prevent capture from reaching an embedded document.
type captureCOM struct {
	comObject
	methods [96]uintptr
	refs    int
}

type captureFixture struct {
	t       *testing.T
	objects []*captureCOM
	scans   int
}

func (f *captureFixture) object() *captureCOM {
	o := &captureCOM{}
	o.vtable = &o.methods
	o.methods[2] = syscall.NewCallback(func(uintptr) uintptr {
		o.refs--
		return 0
	})
	f.objects = append(f.objects, o)
	return o
}

func (o *captureCOM) output(out unsafe.Pointer) uintptr {
	o.refs++
	*(**comObject)(out) = &o.comObject
	return 0
}

func (f *captureFixture) element(text string, password, offscreen bool) *captureCOM {
	o := f.object()
	o.methods[35] = syscall.NewCallback(func(_ uintptr, out unsafe.Pointer) uintptr {
		if password {
			*(*int32)(out) = 1
		}
		return 0
	})
	o.methods[38] = syscall.NewCallback(func(_ uintptr, out unsafe.Pointer) uintptr {
		if offscreen {
			*(*int32)(out) = 1
		}
		return 0
	})
	var pattern *captureCOM
	if text != "" {
		r := f.object()
		r.methods[12] = syscall.NewCallback(func(_ uintptr, maxChars uintptr, out unsafe.Pointer) uintptr {
			u := syscall.StringToUTF16(text)
			n := min(len(u)-1, int(maxChars))
			b, _, _ := oleaut.NewProc("SysAllocStringLen").Call(uintptr(unsafe.Pointer(&u[0])), uintptr(n))
			*(*uintptr)(out) = b
			return 0
		})
		r.methods[10] = syscall.NewCallback(func(uintptr, uintptr) uintptr { return 0x80004001 })
		r.methods[3] = syscall.NewCallback(func(uintptr, uintptr) uintptr { return 0x80004001 })
		ranges := f.array([]*captureCOM{r})
		pattern = f.object()
		pattern.methods[5] = syscall.NewCallback(func(_ uintptr, out unsafe.Pointer) uintptr { return ranges.output(out) })
	}
	o.methods[16] = syscall.NewCallback(func(_ uintptr, id uintptr, out unsafe.Pointer) uintptr {
		if id != 10014 {
			f.t.Errorf("unexpected pattern %d", id)
		}
		if pattern != nil {
			return pattern.output(out)
		}
		return 0x80004001
	})
	return o
}

func (f *captureFixture) array(elements []*captureCOM) *captureCOM {
	o := f.object()
	o.methods[3] = syscall.NewCallback(func(_ uintptr, out unsafe.Pointer) uintptr {
		*(*int32)(out) = int32(len(elements))
		return 0
	})
	o.methods[4] = syscall.NewCallback(func(_ uintptr, index uintptr, out unsafe.Pointer) uintptr {
		return elements[index].output(out)
	})
	return o
}

func (f *captureFixture) automation(root *captureCOM, docs []*captureCOM) *captureCOM {
	condition := f.object()
	array := f.array(docs)
	root.methods[6] = syscall.NewCallback(func(_ uintptr, scope uintptr, cond, out unsafe.Pointer) uintptr {
		f.scans++
		if scope != 4 || cond != unsafe.Pointer(&condition.comObject) {
			f.t.Error("document lookup must stay within the foreground window's descendants")
		}
		return array.output(out)
	})
	a := f.object()
	a.methods[3] = syscall.NewCallback(func(_ uintptr, left, right, out unsafe.Pointer) uintptr {
		if left == right {
			*(*int32)(out) = 1
		}
		return 0
	})
	a.methods[23] = syscall.NewCallback(func(_ uintptr, property uintptr, value, out unsafe.Pointer) uintptr {
		v := (*automationVariant)(value)
		if property != 30003 || v.Type != 3 || v.Value[0] != 50030 {
			f.t.Error("fallback must query Document controls only")
		}
		return condition.output(out)
	})
	return a
}

func (f *captureFixture) walker(parents map[*comObject]*captureCOM) *captureCOM {
	w := f.object()
	w.methods[3] = syscall.NewCallback(func(_ uintptr, element, out unsafe.Pointer) uintptr {
		if parent := parents[(*comObject)(element)]; parent != nil {
			return parent.output(out)
		}
		return 0
	})
	return w
}

func (f *captureFixture) checkReleased() {
	for i, o := range f.objects {
		if o.refs != 0 {
			f.t.Errorf("COM object %d has %d outstanding references", i, o.refs)
		}
	}
}

func TestCaptureEmbeddedDocumentSelection(t *testing.T) {
	f := &captureFixture{t: t}
	defer f.checkReleased()
	root := f.element("", false, false)
	shell := f.element("\ufffc", false, false)
	document := f.element("Selected PDF text", false, false)
	a := f.automation(root, []*captureCOM{shell, document})
	w := f.walker(map[*comObject]*captureCOM{&shell.comObject: root})
	s, ok := captureSelection(&a.comObject, &root.comObject, &shell.comObject, &shell.comObject, &w.comObject, 120, 240)
	if !ok || s.Text != "Selected PDF text" || s.Bounds.X != 120 || s.Bounds.Y != 240 {
		t.Fatalf("embedded PDF selection lost: %#v, %v", s, ok)
	}
}

func TestCaptureDeepAncestorSelection(t *testing.T) {
	f := &captureFixture{t: t}
	defer f.checkReleased()
	root := f.element("", false, false)
	provider := f.element("Nested selection", false, false)
	parents := map[*comObject]*captureCOM{&provider.comObject: root}
	start := provider
	for i := 0; i < 12; i++ {
		child := f.element("", false, false)
		parents[&child.comObject] = start
		start = child
	}
	a := f.automation(root, nil)
	w := f.walker(parents)
	s, ok := captureSelection(&a.comObject, &root.comObject, &start.comObject, nil, &w.comObject, 0, 0)
	if !ok || s.Text != "Nested selection" || f.scans != 0 {
		t.Fatalf("ancestor selection lost: %#v, %v; scans=%d", s, ok, f.scans)
	}
}

func TestCaptureStopsAtForegroundRoot(t *testing.T) {
	f := &captureFixture{t: t}
	defer f.checkReleased()
	root := f.element("", false, false)
	desktop := f.element("Unrelated selection", false, false)
	a := f.automation(root, nil)
	w := f.walker(map[*comObject]*captureCOM{&root.comObject: desktop})
	if s, ok := captureSelection(&a.comObject, &root.comObject, &root.comObject, nil, &w.comObject, 0, 0); ok {
		t.Fatalf("read selection outside foreground window: %#v", s)
	}
}

func TestCapturePasswordDoesNotSearchDocuments(t *testing.T) {
	f := &captureFixture{t: t}
	defer f.checkReleased()
	root := f.element("", false, false)
	password := f.element("secret", true, false)
	a := f.automation(root, []*captureCOM{f.element("Unrelated selection", false, false)})
	s, ok := captureSelection(&a.comObject, &root.comObject, &password.comObject, nil, nil, 0, 0)
	if ok || f.scans != 0 {
		t.Fatalf("password capture must stop: %#v, %v; scans=%d", s, ok, f.scans)
	}
}

func TestDocumentSelectionSkipsHiddenAndEmptyProviders(t *testing.T) {
	f := &captureFixture{t: t}
	defer f.checkReleased()
	root := f.element("", false, false)
	docs := []*captureCOM{
		f.element("Hidden selection", false, true),
		f.element("secret", true, false),
		f.element(" \u200b\ufffc ", false, false),
		f.element("", false, false),
		f.element(strings.Repeat("x", 13000), false, false),
	}
	a := f.automation(root, docs)
	s, ok := documentSelection(&a.comObject, &root.comObject, 0, 0)
	if !ok || s.Text != strings.Repeat("x", 12000) {
		t.Fatalf("visible text selection not captured with character limit: len=%d, %v", len(s.Text), ok)
	}
}

func TestDocumentSelectionDoesNotReadWithoutTextSelection(t *testing.T) {
	f := &captureFixture{t: t}
	defer f.checkReleased()
	root := f.element("", false, false)
	a := f.automation(root, []*captureCOM{f.element("\ufffc", false, false), f.element("", false, false)})
	if s, ok := documentSelection(&a.comObject, &root.comObject, 0, 0); ok {
		t.Fatalf("empty selection must not open a popup: %#v", s)
	}
}
