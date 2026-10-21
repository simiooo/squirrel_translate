//go:build windows

package main

import "github.com/simiooo/squirrel_translate/internal/i18n"

func tr(text string) string { return i18n.T(text) }
func productName() string   { return i18n.ProductName() }
