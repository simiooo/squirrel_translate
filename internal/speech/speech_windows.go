//go:build windows

// Package speech uses installed Windows voices without sending text to a service.
package speech

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

var ErrVoiceUnavailable = errors.New("no installed voice for source language")

// The script is constant. Untrusted text is UTF-8 JSON encoded as base64 on
// stdin, never interpolated into PowerShell or interpreted as SSML.
const script = `
$ErrorActionPreference = 'Stop'
try {
  Add-Type -AssemblyName System.Speech
  $data = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String([Console]::In.ReadToEnd())) | ConvertFrom-Json
  $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
  try {
    $voices = @($synth.GetInstalledVoices() | Where-Object { $_.Enabled })
    if ($data.language) {
      $exact = @($voices | Where-Object { $_.VoiceInfo.Culture.Name -ieq $data.language })
      $base = ($data.language -split '-')[0]
      $matching = @($voices | Where-Object { $_.VoiceInfo.Culture.TwoLetterISOLanguageName -ieq $base })
      if ($exact.Count -gt 0) { $synth.SelectVoice($exact[0].VoiceInfo.Name) }
      elseif ($matching.Count -gt 0) { $synth.SelectVoice($matching[0].VoiceInfo.Name) }
      else { exit 3 }
    }
    $synth.SetOutputToDefaultAudioDevice()
    $synth.Speak([string]$data.text)
  } finally { $synth.Dispose() }
} catch { exit 1 }
`

func command(ctx context.Context, text, language, source string) *exec.Cmd {
	data, _ := json.Marshal(map[string]string{"text": text, "language": strings.ReplaceAll(language, "_", "-")})
	units := utf16.Encode([]rune(source))
	encoded := make([]byte, len(units)*2)
	for i, unit := range units {
		encoded[2*i], encoded[2*i+1] = byte(unit), byte(unit>>8)
	}
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, path, "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Stdin = strings.NewReader(base64.StdEncoding.EncodeToString(data))
	return cmd
}

// Speak blocks until playback finishes or ctx cancels and kills the helper.
func Speak(ctx context.Context, text, language string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	err := command(ctx, text, language, script).Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 3 {
		return ErrVoiceUnavailable
	}
	return err
}
