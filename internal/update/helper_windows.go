//go:build windows

package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\io.github.simiooo.squirrel_translate`

func (p *Plan) updateRegistration() (func() error, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer key.Close()
	dir, _, err := key.GetStringValue("InstallLocation")
	if err != nil {
		return nil, err
	}
	icon, _, err := key.GetStringValue("DisplayIcon")
	if err != nil {
		return nil, err
	}
	// A portable copy must not change another installation's registration.
	if !samePath(dir, filepath.Dir(p.Target)) || !samePath(icon, p.Target) {
		return nil, nil
	}
	old, _, err := key.GetStringValue("DisplayVersion")
	if err != nil {
		return nil, err
	}
	undo := func() error {
		k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE)
		if err != nil {
			return err
		}
		defer k.Close()
		return k.SetStringValue("DisplayVersion", old)
	}
	writable, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE)
	if err != nil {
		return nil, err
	}
	defer writable.Close()
	return undo, writable.SetStringValue("DisplayVersion", p.Version)
}

// Start waits for a readiness handshake before its caller may quit. The helper
// opens a handle to the exact parent, avoiding PID reuse and exit-time races.
func (p *Plan) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.validate(); err != nil {
		return err
	}
	if _, err := p.updatedMarker(); err != nil {
		return err
	}
	cmd := exec.Command(filepath.Join(p.Work, "helper.exe"), "--apply-update", p.Work)
	cmd.Dir = p.Work
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if err := cmd.Start(); err != nil {
		return err
	}
	exit := make(chan error, 1)
	go func() { exit <- cmd.Wait() }()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-exit:
			return fmt.Errorf("update helper exited before readiness: %v", err)
		case <-ctx.Done():
			cmd.Process.Kill()
			<-exit
			return ctx.Err()
		case <-timer.C:
			cmd.Process.Kill()
			<-exit
			return errors.New("update helper did not become ready")
		case <-ticker.C:
			if data, err := os.ReadFile(filepath.Join(p.Work, "ready")); err == nil && string(data) == "ready" {
				return nil
			}
		}
	}
}

// RunHelper runs before GUI initialization and single-instance registration.
func RunHelper(work string) error {
	p, err := readPlan(work)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if !samePath(self, filepath.Join(p.Work, "helper.exe")) {
		return errors.New("helper must run from its staging directory")
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.ParentPID))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	var name [32768]uint16
	n := uint32(len(name))
	if err = windows.QueryFullProcessImageName(h, 0, &name[0], &n); err != nil {
		return err
	}
	if !samePath(windows.UTF16ToString(name[:n]), p.Target) {
		return errors.New("parent executable does not match update target")
	}
	if _, err = p.updatedMarker(); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(work, "ready"), []byte("ready"), 0600); err != nil {
		return err
	}
	state, err := windows.WaitForSingleObject(h, 30000)
	if err != nil || state != windows.WAIT_OBJECT_0 {
		return errors.New("application did not exit; update was not applied")
	}
	restart := func() error {
		_ = os.Remove(filepath.Join(work, "booted"))
		args := append(append([]string{}, p.Args...), "--update-cleanup", work)
		cmd := exec.Command(p.Target, args...)
		cmd.Dir = filepath.Dir(p.Target)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
		if err := cmd.Start(); err != nil {
			return err
		}
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case err := <-exited:
				return fmt.Errorf("restarted application exited before readiness: %v", err)
			case <-timer.C:
				cmd.Process.Kill()
				<-exited
				return errors.New("restarted application did not become ready")
			case <-ticker.C:
				if _, err := os.Stat(filepath.Join(work, "booted")); err == nil {
					return nil
				}
			}
		}
	}
	result := p.apply(restart)
	if result.Error != "" && !result.RecoveryRequired {
		if err = restart(); err != nil {
			result.Error += "; restarting original version failed: " + err.Error()
			result.RecoveryRequired = true
			_ = writeJSON(filepath.Join(work, "result.json"), result)
		}
	}
	// The relaunched app waits for this marker before removing helper.exe.
	_ = os.WriteFile(filepath.Join(work, "finished"), []byte("finished"), 0600)
	if result.RecoveryRequired {
		return &RecoveryError{Work: work, Reason: result.Error}
	}
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return nil
}

// Finish is called by the relaunched app. Validate against its own executable,
// never recursively remove an arbitrary command-line path.
func Finish(work string) (Result, error) {
	p, err := readPlan(work)
	if err != nil {
		return Result{}, err
	}
	self, err := os.Executable()
	if err != nil || !samePath(self, p.Target) {
		return Result{}, errors.New("cleanup target mismatch")
	}
	data, err := os.ReadFile(filepath.Join(work, "result.json"))
	if err != nil {
		return Result{}, err
	}
	var result Result
	if err = json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	if err = os.WriteFile(filepath.Join(work, "booted"), []byte("booted"), 0600); err != nil {
		return result, err
	}
	if !result.RecoveryRequired {
		go func() {
			for i := 0; i < 60; i++ {
				if _, err := os.Stat(filepath.Join(work, "finished")); err == nil {
					// Remove the helper first; if still running, keep the result and
					// backups intact until Windows releases its executable handle.
					if err = os.Remove(filepath.Join(work, "helper.exe")); err == nil || os.IsNotExist(err) {
						_ = os.RemoveAll(work)
						return
					}
				}
				time.Sleep(500 * time.Millisecond)
			}
		}()
	}
	return result, nil
}

// RestartArgs preserves user-selected data directories, language and tray mode,
// but never carries a previous cleanup request into a later update.
func RestartArgs(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--update-cleanup" || args[i] == "-update-cleanup" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--update-cleanup=") || strings.HasPrefix(args[i], "-update-cleanup=") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}
