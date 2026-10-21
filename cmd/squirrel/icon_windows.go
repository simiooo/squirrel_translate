//go:build windows

package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/design"
)

var appIconPNG = design.IconPNG
var appIconImage = ui.MustParseSVG(design.IconSVG)
