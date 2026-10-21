//go:build windows

package native

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestIsolatedWorkerChild(t *testing.T) {
	if os.Getenv("SQUIRREL_TEST_WORKER") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
	os.Exit(0)
}
func TestIsolatedWorkerIsReapedOnTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestIsolatedWorkerChild$")
	cmd.Env = append(os.Environ(), "SQUIRREL_TEST_WORKER=1")
	cmd.SysProcAttr = HideConsoleChild()
	started := time.Now()
	if _, e := isolatedOutput(cmd); e == nil {
		t.Fatal("timed out worker unexpectedly succeeded")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("worker was not bounded by timeout")
	}
	if cmd.ProcessState == nil {
		t.Fatal("worker process was not reaped")
	}
}
