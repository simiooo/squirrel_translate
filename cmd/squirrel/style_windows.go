//go:build windows

package main

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// One palette and one set of control metrics for every application surface.
// The base controls retain their keyboard, focus and accessibility behavior.
func minimalTheme(c *ui.Context) *ui.Theme {
	t := *c.Theme()
	if appearance == "浅色" {
		t.Dark = false
	} else if appearance == "深色" {
		t.Dark = true
	}
	t.Radius = metric("Radius/Control")
	t.FontSize = metric("Type/Body")
	t.Font = "Segoe UI Variable Text, Microsoft YaHei UI, Microsoft YaHei, sans-serif"
	t.Spacing = uiSpacing.Controls.Unit
	t.Background = colorToken(c, "Surface")
	t.Surface = colorToken(c, "Canvas")
	t.SurfaceHover = colorToken(c, "Hover")
	t.SurfacePressed = colorToken(c, "Pressed")
	t.Border = colorToken(c, "Rule")
	t.Text = colorToken(c, "Text")
	t.TextMuted = colorToken(c, "Muted")
	t.Accent = colorToken(c, "Accent")
	t.AccentHover = t.Accent
	t.AccentPressed = t.Accent
	t.AccentText = t.Background
	t.Success = colorToken(c, "Success")
	t.Danger = colorToken(c, "Danger")
	t.Selection = colorToken(c, "Selected")
	t.Focus = colorToken(c, "Focus")
	t.Scrollbar = colorToken(c, "Border").Alpha(.6)
	t.ScrollbarWidth = 4
	c.SetTheme(&t)
	return c.Theme()
}

func textButton(c *ui.Context, label string, icon ...string) ui.Element {
	return toneButton(c, label, neutralAction, icon...)
}
func toneButton(c *ui.Context, label string, tone actionTone, icon ...string) ui.Element {
	s := uiSpacing.Controls
	b := ui.ButtonBase(c).Label(tr(label)).Padding(s.ButtonY, s.ButtonX).MinHeight(32).Gap(s.IconGap).FontSize(13)
	color := actionAppearance(c, b, tone)
	b.TextColor(color)
	b.Children(func() {
		if len(icon) > 0 {
			ui.Box(c).Size(18, 18).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, icon[0], color) })
		}
		ui.Text(c, tr(label)).SingleLine()
	})
	return b
}
func actionColor(c *ui.Context, b ui.Element) ui.Color { return colorToken(c, "Accent") }

// Text alone is the disclosure affordance. Expanded state remains available
// to assistive technology; weight distinguishes an open section visually.
func disclosure(c *ui.Context, label string, open *bool) ui.Element {
	b := textButton(c, label)
	if b.Clicked() {
		*open = !*open
	}
	b.Expanded(*open)
	if *open {
		b.Bold()
	}
	return b
}

func field(c *ui.Context, value *string, placeholder string) ui.Element {
	s := uiSpacing.Controls
	return inputAppearance(c, ui.TextInputBase(c, value)).Placeholder(tr(placeholder)).Padding(s.InputY, s.InputX)
}

func textArea(c *ui.Context, value *string, placeholder string) ui.Element {
	s := uiSpacing.Controls
	return inputAppearance(c, ui.TextAreaBase(c, value)).Placeholder(tr(placeholder)).Padding(s.InputY, s.InputX)
}

// State feedback is painted over the fixed input geometry. Border widths and
// padding never change with hover or focus, keeping glyphs, carets and rows still.
func inputAppearance(c *ui.Context, e ui.Element) ui.Element {
	t := c.Theme()
	e.Background(ui.Transparent).Border(0, t.Border).Radius(t.Radius).FocusRing(false)
	color, height := colorToken(c, "Rule"), float32(1)
	if e.Hovered() {
		color = colorToken(c, "Border")
	}
	if e.Focused() {
		color, height = t.Accent, 2
	}
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - height, W: r.W, H: height}, color, 0)
	})
	return e
}

func check(c *ui.Context, value *bool, label string) ui.Element {
	t := c.Theme()
	s := uiSpacing.Controls
	row := ui.CheckboxBase(c, value).Label(tr(label)).Gap(s.ToggleGap).AlignItems(ui.Center).
		Padding(s.ToggleY, s.ToggleX).Margin(0, -s.ToggleX).MinHeight(s.ToggleMinHeight).
		Radius(t.Radius).FontSize(13).TextColor(t.Text).FocusRing(false)
	accent := actionColor(c, row)
	outline := t.TextMuted.Alpha(.7)
	if row.Pressed() {
		row.Background(t.SurfacePressed)
		outline = accent
	} else if row.Hovered() {
		row.Background(t.SurfaceHover)
		outline = accent
	}
	on := *value
	focus := row.FocusVisible()
	row.Children(func() {
		ui.Box(c).Size(s.ToggleSize, s.ToggleSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			radius := min(float32(4), r.W*.28)
			if on {
				p.Fill(r, accent, radius)
				var tick ui.Path
				tick.MoveTo(r.X+r.W*.23, r.Y+r.H*.51).LineTo(r.X+r.W*.43, r.Y+r.H*.71).LineTo(r.X+r.W*.78, r.Y+r.H*.30)
				p.StrokePath(&tick, 1.5, t.AccentText)
			} else {
				p.Stroke(ui.Rect{X: r.X + .5, Y: r.Y + .5, W: r.W - 1, H: r.H - 1}, outline, radius, 1)
			}
			if focus {
				p.FocusRing(r, [4]float32{radius, radius, radius, radius})
			}
		})
		if label != "" {
			ui.Text(c, tr(label)).SingleLine()
		}
	})
	return row
}

// The trigger has no permanent field outline. Only the floating option list
// needs a surface so it remains readable over the content below it.
func selectField(c *ui.Context, value *string, options []string, caption ...string) ui.Element {
	return selectControl(c, value, options, true, caption...)
}

func plainSelect(c *ui.Context, value *string, options []string, caption ...string) ui.Element {
	return selectControl(c, value, options, false, caption...)
}

func selectControl(c *ui.Context, value *string, options []string, arrow bool, caption ...string) ui.Element {
	t := c.Theme()
	s := uiSpacing.Controls
	// Bind the control to a localized display value; persist the original domain value.
	display := tr(*value)
	sel := ui.SelectBase(c, &display)
	b := sel.Trigger
	b.Padding(s.SelectY, s.SelectX).Radius(t.Radius).AlignItems(ui.Center)
	if arrow {
		b.MinWidth(s.SelectMinWidth).Gap(s.SelectGap).Justify(ui.SpaceBetween)
	}
	color := actionAppearance(c, b, neutralAction)
	b.TextColor(color)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	label := *value
	if len(caption) > 0 && caption[0] != "" {
		label = caption[0]
	}
	b.Children(func() {
		ui.Text(c, tr(label)).SingleLine()
		if arrow {
			ui.Box(c).Size(10, 10).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
				var path ui.Path
				path.MoveTo(r.X+1, r.Y+3).LineTo(r.X+5, r.Y+7).LineTo(r.X+9, r.Y+3)
				p.StrokePath(&path, 1, color)
			})
		}
	})
	sel.Popup(func(panel ui.Element) {
		panel.Margin(s.MenuOffset, 0, 0, 0).Padding(s.MenuInset).Radius(t.Radius+2).Background(t.Background).Shadow(0, 4, 20, 0, ui.RGBA(0, 0, 0, .14))
		_, height := c.Size()
		anchor := b.Bounds()
		room := max(anchor.Y, height-anchor.Y-anchor.H) - s.MenuOffset - 3*s.MenuInset
		ui.Scroll(c).MaxHeight(min(280, max(32, room))).MinHeight(0).Children(func() {
			for _, opt := range options {
				item := sel.Item(tr(opt)).Label(tr(opt)).Padding(s.MenuItemY, s.MenuItemX).Radius(t.Radius).TextColor(t.Text)
				if item.Highlighted() {
					item.Background(t.SurfaceHover)
				}
				if opt == *value {
					item.Bold().Background(colorToken(c, "Selected")).TextColor(colorToken(c, "SelectedText"))
				}
				item.Children(func() { ui.Text(c, tr(opt)).SingleLine() })
			}
		})
	})
	if b.Changed() {
		for _, opt := range options {
			if tr(opt) == display {
				*value = opt
				break
			}
		}
	}
	return b
}

// Preserve the select's hit padding while aligning its text with inputs.
func formSelect(c *ui.Context, value *string, options []string, caption ...string) ui.Element {
	s := uiSpacing.Controls
	return selectField(c, value, options, caption...).Margin(0, s.InputX-s.SelectX)
}

func iconButton(c *ui.Context, icon, label string) ui.Element {
	t := c.Theme()
	size := uiSpacing.Controls.IconButtonSize
	b := ui.ButtonBase(c).Label(tr(label)).Tooltip(tr(label)).Size(size, size).Radius(t.Radius)
	color := actionAppearance(c, b, neutralAction)
	if icon == "close" {
		color = colorToken(c, "Close")
		if b.Hovered() || b.Pressed() {
			color = colorToken(c, "Text")
		}
	}
	if icon == "star-filled" {
		color = colorToken(c, "Success")
	}
	if b.Pressed() {
		b.Background(t.SurfacePressed)
	} else if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.DrawOver(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, icon, color) })
	return b
}

func pinToggle(c *ui.Context, value *bool) ui.Element {
	t := c.Theme()
	size := uiSpacing.Controls.IconButtonSize
	b := ui.CheckboxBase(c, value).Label(tr("置顶")).Size(size, size).Radius(t.Radius)
	color := colorToken(c, "Secondary")
	icon, label := "pin", tr("置顶")
	if *value {
		icon, label = "pinned", tr("取消置顶")
		color = colorToken(c, "SelectedText")
		b.Background(colorToken(c, "Selected"))
	}
	b.Tooltip(tr(label))
	if b.Hovered() {
		if *value {
			b.Background(colorToken(c, "SelectedHover"))
		} else {
			b.Background(t.SurfaceHover)
		}
	}
	b.DrawOver(func(p *ui.Painter, r ui.Rect) { drawIcon(p, r, icon, color) })
	return b
}

func drawIcon(p *ui.Painter, r ui.Rect, name string, color ui.Color) {
	x, y := r.X+(r.W-18)/2, r.Y+(r.H-18)/2
	var path ui.Path
	switch name {
	case "translate":
		path.MoveTo(x+1, y+4).LineTo(x+11, y+4).MoveTo(x+6, y+1).LineTo(x+6, y+4).
			MoveTo(x+3, y+6).CubeTo(x+4, y+10, x+7, y+12, x+10, y+13).
			MoveTo(x+10, y+4).CubeTo(x+9, y+9, x+6, y+13, x+1, y+15).
			MoveTo(x+10, y+16).LineTo(x+14, y+7).LineTo(x+18, y+16).MoveTo(x+12, y+13).LineTo(x+16, y+13)
	case "book":
		path.MoveTo(x+2, y+3).LineTo(x+7, y+3).LineTo(x+9, y+4).LineTo(x+11, y+3).LineTo(x+16, y+3).
			LineTo(x+16, y+15).LineTo(x+11, y+15).LineTo(x+9, y+16).LineTo(x+7, y+15).LineTo(x+2, y+15).Close().
			MoveTo(x+9, y+4).LineTo(x+9, y+16)
	case "settings":
		path.MoveTo(x+7, y+1).LineTo(x+11, y+1).LineTo(x+12, y+4).LineTo(x+15, y+5).LineTo(x+17, y+9).
			LineTo(x+15, y+13).LineTo(x+12, y+14).LineTo(x+11, y+17).LineTo(x+7, y+17).LineTo(x+6, y+14).
			LineTo(x+3, y+13).LineTo(x+1, y+9).LineTo(x+3, y+5).LineTo(x+6, y+4).Close()
		path.MoveTo(x+12, y+9).CubeTo(x+12, y+13, x+6, y+13, x+6, y+9).CubeTo(x+6, y+5, x+12, y+5, x+12, y+9)
	case "minus":
		path.MoveTo(x+4, y+9).LineTo(x+14, y+9)
	case "maximize":
		path.MoveTo(x+4, y+4).LineTo(x+14, y+4).LineTo(x+14, y+14).LineTo(x+4, y+14).Close()
	case "restore":
		path.MoveTo(x+6, y+5).LineTo(x+6, y+2).LineTo(x+16, y+2).LineTo(x+16, y+12).LineTo(x+13, y+12).
			MoveTo(x+3, y+5).LineTo(x+13, y+5).LineTo(x+13, y+15).LineTo(x+3, y+15).Close()
	case "arrow":
		path.MoveTo(x+2, y+9).LineTo(x+16, y+9).MoveTo(x+11, y+4).LineTo(x+16, y+9).LineTo(x+11, y+14)
	case "back":
		path.MoveTo(x+15, y+9).LineTo(x+2, y+9).MoveTo(x+7, y+4).LineTo(x+2, y+9).LineTo(x+7, y+14)
	case "grip":
		for _, dx := range []float32{6, 11} {
			for _, dy := range []float32{4, 9, 14} {
				p.Fill(ui.Rect{X: x + dx, Y: y + dy, W: 2, H: 2}, color, 1)
			}
		}
	case "check":
		path.MoveTo(x+3, y+9).LineTo(x+7, y+13).LineTo(x+15, y+4)
	case "star", "star-filled":
		path.MoveTo(x+9, y+1.5).LineTo(x+11.3, y+6.2).
			LineTo(x+16.5, y+6.9).LineTo(x+12.75, y+10.55).
			LineTo(x+13.65, y+15.7).LineTo(x+9, y+13.25).
			LineTo(x+4.35, y+15.7).LineTo(x+5.25, y+10.55).
			LineTo(x+1.5, y+6.9).LineTo(x+6.7, y+6.2).Close()
		if name == "star-filled" {
			p.FillPath(&path, color)
		}
	case "refresh":
		path.MoveTo(x+15, y+7).CubeTo(x+14, y+3, x+9, y+1, x+5, y+4).CubeTo(x+1, y+7, x+2, y+13, x+6, y+15).CubeTo(x+9, y+17, x+13, y+15, x+15, y+12)
		path.MoveTo(x+15, y+2).LineTo(x+15, y+7).LineTo(x+10, y+7)
	case "close":
		// Keep the 32-DIP hit target, but give the dismiss glyph its own
		// smaller, lighter geometry. Snap directly to device pixels instead
		// of rescaling a raster icon, including at fractional Windows scales.
		scale := max(float32(1), p.Scale())
		stroke := float32(math.Round(float64(metric("Icon/CloseStroke") * scale)))
		span := float32(math.Round(float64(metric("Icon/CloseSize") * scale)))
		left := float32(math.Round(float64((r.X+r.W/2)*scale-span/2-stroke/2))) + stroke/2
		top := float32(math.Round(float64((r.Y+r.H/2)*scale-span/2-stroke/2))) + stroke/2
		x0, y0, x1, y1 := left/scale, top/scale, (left+span)/scale, (top+span)/scale
		path.MoveTo(x0, y0).LineTo(x1, y1).MoveTo(x1, y0).LineTo(x0, y1)
		p.StrokePath(&path, max(1, stroke)/scale, color)
		return
	case "pin", "pinned":
		path.MoveTo(x+9, y+2).LineTo(x+16, y+9).MoveTo(x+10, y+3).LineTo(x+5, y+7).LineTo(x+3, y+7).LineTo(x+3, y+9).LineTo(x+9, y+15).LineTo(x+11, y+15).LineTo(x+11, y+13).LineTo(x+15, y+8).MoveTo(x+6, y+12).LineTo(x+2, y+16)
		if name == "pinned" {
			path.MoveTo(x+1, y+1).LineTo(x+17, y+17)
		}
	case "copy":
		path.MoveTo(x+6, y+5).LineTo(x+6, y+2).LineTo(x+15, y+2).LineTo(x+15, y+12).LineTo(x+12, y+12).MoveTo(x+3, y+5).LineTo(x+12, y+5).LineTo(x+12, y+16).LineTo(x+3, y+16).Close()
	}
	p.StrokePath(&path, 1.65, color)
}
