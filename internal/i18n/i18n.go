// Package i18n localizes application chrome, never selections or model output.
package i18n

import (
	"embed"
	"encoding/json"
	"strings"
	"sync/atomic"
)

//go:embed en.json
var files embed.FS
var english map[string]string
var language atomic.Bool

func init() {
	data, err := files.ReadFile("en.json")
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(data, &english); err != nil {
		panic(err)
	}
}

// Resolve supports only Simplified Chinese and English; unknown locales use English.
func Resolve(choice, system string) string {
	if choice == "zh-CN" || choice == "en" {
		return choice
	}
	if strings.HasPrefix(strings.ToLower(system), "zh") {
		return "zh-CN"
	}
	return "en"
}

func Set(choice, system string) { language.Store(Resolve(choice, system) == "en") }
func Locale() string {
	if language.Load() {
		return "en"
	}
	return "zh-CN"
}

// Chinese source strings are stable message IDs. Only exact matches are translated.
// Unknown keys are preserved, including paths and provider diagnostics.
func T(source string) string {
	if source == "system" && !language.Load() {
		return "跟随系统"
	}
	if source == "zh-CN" {
		return "简体中文"
	}
	if source == "en" {
		return "English"
	}
	if language.Load() {
		if value, ok := english[source]; ok {
			return value
		}
	}
	return source
}

func ProductName() string {
	if language.Load() {
		return "squirrel_translate"
	}
	return "松鼠翻译"
}
