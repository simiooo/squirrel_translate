//go:build windows

package main

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/speech"
)

// Owned and updated on the UI thread. One playback across all windows.
type speechUI struct {
	owner      any
	cancel     context.CancelFunc
	generation uint64
}

func (a *application) stopSpeech(owner any) {
	if owner != nil && a.speech.owner != owner {
		return
	}
	if a.speech.cancel != nil {
		a.speech.cancel()
	}
	a.speech = speechUI{generation: a.speech.generation + 1}
	a.invalidateSpeech()
}

func (a *application) invalidateSpeech() {
	if a.window != nil {
		a.window.Invalidate()
	}
	for _, s := range a.sessions {
		if !s.closed && s.window != nil {
			s.window.Invalidate()
		}
	}
}

func (a *application) speechButton(c *ui.Context, owner any, word, language string, window *mygo.Window, report func(string), wordSize ...float32) {
	label, icon := tr("朗读单词"), "speaker"
	if a.speech.owner == owner {
		label, icon = tr("停止朗读"), "stop"
	}
	offsetY := float32(0)
	if len(wordSize) > 0 {
		// Move the whole button to the word baseline, keeping its icon,
		// hover background and hit area together. Use the same position for stop.
		metrics := (ui.Font{Family: c.Theme().Font, Size: wordSize[0], Weight: 700}).Metrics()
		bottom := float32(6) // speaker path ends 6 DIPs below the icon center.
		offsetY = (metrics.Ascent-metrics.Descent)/2 - bottom
	}
	if !iconButton(c, icon, label).Translate(0, offsetY).Disabled(language == "" || word == "").Clicked() {
		return
	}
	playing := a.speech.owner == owner
	a.stopSpeech(nil)
	if playing {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.speech.owner, a.speech.cancel = owner, cancel
	generation := a.speech.generation
	a.invalidateSpeech()
	go func() {
		defer cancel()
		err := speech.Speak(ctx, word, language)
		window.Update(func() {
			if a.speech.generation != generation {
				return
			}
			a.stopSpeech(nil)
			if errors.Is(err, speech.ErrVoiceUnavailable) {
				report(tr("未安装该语言的系统语音，请在 Windows 设置中添加语音。"))
			} else if err != nil && !errors.Is(err, context.Canceled) {
				report(tr("朗读失败，请检查系统语音和音频输出。"))
			}
		})
	}()
}
