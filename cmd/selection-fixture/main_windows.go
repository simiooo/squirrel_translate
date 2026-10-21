//go:build windows

// selection-fixture is an independent process for exercising the actual
// cross-process UI Automation capture path without touching user documents.
package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"log"
)

func main() {
	text := "bank\nI went to the bank to deposit a cheque.\nSelect a word here to test the floating toolbar."
	mygo.App.SetName("Selection Test Fixture")
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Selection Test Fixture", Width: 700, Height: 360, Hidden: true, Content: ui.View(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(24).Gap(14).Children(func() {
				ui.Text(c, "Cross-process selection fixture").FontSize(24).Bold()
				ui.Text(c, "Select bank with the mouse, or press Ctrl+Home then Ctrl+Shift+Right.")
				ui.TextArea(c, &text).ReadOnly(true).Grow(1).Label("Selection fixture text")
			})
		})})
		w.ShowInactive()
		w.Show()
	})
	if e := mygo.App.Run(); e != nil {
		log.Fatal(e)
	}
}
