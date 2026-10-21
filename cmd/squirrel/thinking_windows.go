//go:build windows

package main

import (
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const thoughtFade = 180 * time.Millisecond

// Cache only the last paragraph's bounded tail. Shape it with the native text
// engine so the visible last row respects wrapping, graphemes and bidi text.
type thoughtLine struct {
	text     string
	width    float32
	font     ui.Font
	layout   *ui.TextLayout
	lastRow  ui.Rect
	start    int
	appeared time.Time
}

func thoughtTail(text string) (string, int) {
	text = strings.TrimRight(text, " \t\r\n")
	start := strings.LastIndexAny(text, "\r\n") + 1
	if len(text)-start > 4096 {
		start = len(text) - 4096
		for start < len(text) && !utf8.RuneStart(text[start]) {
			start++
		}
	}
	return text[start:], start
}

func (line *thoughtLine) view(c *ui.Context, reasoning string, available ...float32) {
	space, theme := uiSpacing.Translation, c.Theme()
	width, _ := c.Size()
	if len(available) > 0 && available[0] > 0 {
		width = available[0]
	}
	width = max(1, width-2*space.InsetX-14-space.LoadingGap)
	text, start := thoughtTail(reasoning)
	font := ui.Font{Family: theme.Font, Size: 13, Lang: c.Locale()}
	if line.layout == nil || text != line.text || width != line.width || font != line.font {
		layout := ui.ShapeText(text, font, width)
		units := 0
		for _, r := range text {
			units += utf16.RuneLen(r)
		}
		last := layout.CaretAt(ui.TextCaretPosition{Index: units, Affinity: ui.TextUpstream})
		if line.layout == nil || start != line.start || last.Y != line.lastRow.Y || line.text == "" {
			line.appeared = c.Now()
		}
		line.text, line.start, line.width, line.font = text, start, width, font
		line.layout, line.lastRow = layout, last
	}
	alpha := float32(1)
	if text != "" && !c.Preferences().ReduceMotion {
		alpha = min(1, max(0, float32(c.Now().Sub(line.appeared))/float32(thoughtFade)))
		if alpha < 1 {
			c.AnimationFrame()
		}
	}
	ui.Row(c).Padding(space.LoadingY, space.InsetX, 0).Gap(space.LoadingGap).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Spinner(c).Label(tr("翻译中")).Size(14, 14)
		if text != "" {
			layout, last := line.layout, line.lastRow
			ui.Box(c).Grow(1).MinWidth(0).Height(max(last.H, font.Size*space.LoadingLineHeight)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				// Clip applies to descendants; keep the custom drawing inside it.
				ui.Box(c).Width(width).Height(last.H).Clip().Opacity(alpha).Label(tr("最新思考")).Children(func() {
					ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
						p.TextLayout(layout, r.X, r.Y-last.Y, theme.TextMuted)
					})
				})
			})
		}
	})
}
