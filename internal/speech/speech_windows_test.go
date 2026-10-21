//go:build windows

package speech

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestInstalledSystemVoiceSynthesizesPlainText(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Exercise the real Windows engine, silently. Quotes, Unicode and markup
	// must remain literal text, never PowerShell commands or SSML.
	silent := strings.Replace(script, "$synth.SetOutputToDefaultAudioDevice()", "$synth.SetOutputToNull()", 1)
	text := "bank 银行 ' \" ` $(exit 7) <speak>hello</speak>\nnext"
	if out, err := command(ctx, text, "", silent).CombinedOutput(); err != nil {
		t.Fatalf("system synthesis failed: %v %s", err, out)
	}
	var exit *exec.ExitError
	if err := command(ctx, "bank", "zz-ZZ", silent).Run(); !errors.As(err, &exit) || exit.ExitCode() != 3 {
		t.Fatalf("missing voice must be reported: %v", err)
	}
}

func TestCancellationKillsSpeechHelper(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := command(ctx, "bank", "en", "Start-Sleep -Seconds 30").Run(); err == nil || ctx.Err() == nil {
		t.Fatalf("helper did not cancel: %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("helper outlived cancellation")
	}
}
