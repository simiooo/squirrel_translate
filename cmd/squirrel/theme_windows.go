//go:build windows

package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/design"
)

var appearance = "跟随系统"

func colorToken(c *ui.Context, name string) ui.Color {
	mode := "Light/"
	dark := c.Theme().Dark
	if appearance == "浅色" {
		dark = false
	} else if appearance == "深色" {
		dark = true
	}
	if dark {
		mode = "Dark/"
	}
	value, ok := design.Tokens.Colors[mode+name]
	if !ok {
		panic("unknown design color: " + name)
	}
	return ui.Hex(value)
}

func metric(name string) float32 {
	v, ok := design.Tokens.Scalars[name]
	if !ok {
		panic("unknown design metric: " + name)
	}
	return v
}

type actionTone uint8

const (
	neutralAction actionTone = iota
	primaryAction
	dangerAction
	successAction
)

func actionAppearance(c *ui.Context, b ui.Element, tone actionTone) ui.Color {
	fg, hover, pressed := "Secondary", "Hover", "Pressed"
	switch tone {
	case primaryAction:
		fg, hover, pressed = "Accent", "AccentHover", "AccentPressed"
	case dangerAction:
		fg, hover, pressed = "Danger", "DangerHover", "DangerPressed"
	case successAction:
		fg, hover, pressed = "Success", "SuccessSubtle", "SuccessSubtle"
	}
	color := colorToken(c, fg)
	b.Background(ui.Transparent).Radius(metric("Radius/Control"))
	if b.IsDisabled() {
		return colorToken(c, "Disabled")
	}
	if b.Pressed() {
		b.Background(colorToken(c, pressed))
	} else if b.Hovered() {
		b.Background(colorToken(c, hover))
	}
	return color
}

func primaryButton(c *ui.Context, label string, icon ...string) ui.Element {
	return toneButton(c, label, primaryAction, icon...)
}
func dangerButton(c *ui.Context, label string, icon ...string) ui.Element {
	return toneButton(c, label, dangerAction, icon...)
}

func sectionTitle(c *ui.Context, title string) {
	ui.Text(c, tr(title)).FontSize(16).Bold().Margin(12, 0, 8)
}

func rule(c *ui.Context) { ui.Box(c).Height(1).Shrink(0).Background(colorToken(c, "Rule")) }

func selectedAppearance(c *ui.Context, b ui.Element, selected bool) ui.Color {
	color := actionAppearance(c, b, neutralAction)
	if selected {
		color = colorToken(c, "SelectedText")
		bg := "Selected"
		if b.Hovered() {
			bg = "SelectedHover"
		}
		b.Background(colorToken(c, bg)).Bold()
		b.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X + 5, Y: r.Y + (r.H-20)/2, W: 3, H: 20}, colorToken(c, "Indicator"), metric("Radius/Indicator"))
		})
	}
	return color
}
