//go:build windows

package main

import (
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestInputStatesKeepTextAndNeighborGeometryStill(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, test := range []struct {
			name, value string
			build       func(*ui.Context, *string) ui.Element
		}{
			{"text", "搜索词汇 bank", func(c *ui.Context, v *string) ui.Element { return field(c, v, "搜索词汇") }},
			{"placeholder", "", func(c *ui.Context, v *string) ui.Element { return field(c, v, "搜索词汇") }},
			{"password", "example-key", func(c *ui.Context, v *string) ui.Element { return field(c, v, "").Password() }},
			{"multiline", "第一行文本\nSecond line", func(c *ui.Context, v *string) ui.Element { return textArea(c, v, "输入文本").Height(96) }},
		} {
			t.Run(test.name+map[bool]string{false: "-light", true: "-dark"}[dark], func(t *testing.T) {
				value := test.value
				tt := ui.NewTester(func(c *ui.Context) {
					minimalTheme(c)
					ui.Column(c).Fill().Padding(12).Gap(8).Background(c.Theme().Background).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
							test.build(c, &value).Width(260).Label("输入框")
							ui.Text(c, "相邻文字").Label("同行")
						})
						textButton(c, "移开焦点")
					})
				}, 400, 180)
				tt.SetDark(dark)
				tt.Move(-1, -1)
				input, _ := tt.Find("输入框")
				neighbor, _ := tt.Find("同行")
				x0, y0 := int(math.Ceil(float64(input.X))), int(math.Ceil(float64(input.Y)))
				x1, y1 := int(input.X+input.W), int(input.Y+input.H)-2
				before := tt.Image()
				pixels := make(map[[2]int]color.Color)
				for y := y0; y < y1; y++ {
					for x := x0; x < x1; x++ {
						pixels[[2]int{x, y}] = before.At(x, y)
					}
				}
				assertStill := func(state string, focused bool) {
					t.Helper()
					if r, _ := tt.Find("输入框"); r != input {
						t.Fatalf("%s changed input geometry: %+v -> %+v", state, input, r)
					}
					if r, _ := tt.Find("同行"); r != neighbor {
						t.Fatalf("%s moved adjacent text: %+v -> %+v", state, neighbor, r)
					}
					after := tt.Image()
					caret, _ := tt.TextCaret()
					for xy, pixel := range pixels {
						// The focused caret is expected to appear; the underlying
						// glyphs and placeholder must stay at exactly the same pixels.
						if focused && float32(xy[0]) >= caret.X-1 && float32(xy[0]) <= caret.X+caret.W+1 {
							continue
						}
						if pixel != after.At(xy[0], xy[1]) {
							t.Fatalf("%s moved input text at %d,%d", state, xy[0], xy[1])
						}
					}
				}
				tt.Move(input.X+input.W/2, input.Y+input.H/2)
				assertStill("hover", false)
				tt.ClickAt(input.X+input.W-4, input.Y+input.H/2)
				if !tt.Focused("输入框") {
					t.Fatal("input lost focus behavior")
				}
				assertStill("focus", true)
				tt.Move(-1, -1)
				assertStill("focused pointer exit", true)
				if err := tt.Click("移开焦点"); err != nil {
					t.Fatal(err)
				}
				assertStill("blur", false)
				tt.Key(0, ui.KeyTab)
				if !tt.Focused("输入框") {
					t.Fatal("input lost keyboard focus behavior")
				}
				if r, _ := tt.Find("输入框"); r != input {
					t.Fatal("keyboard focus changed input geometry")
				}
			})
		}
	}
}
