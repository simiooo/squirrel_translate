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
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func TestMain(m *testing.M) {
	// These routes exist only in the test executable; production cannot select
	// a registry key or update endpoint using environment variables.
	if key := os.Getenv("SQUIRREL_UPDATE_TEST_KEY"); strings.HasPrefix(key, `Software\SquirrelTranslateUpdateTests\`) {
		uninstallKey = key
	}
	if len(os.Args) == 3 && os.Args[1] == "--apply-update" {
		if err := RunHelper(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) == 3 && os.Args[1] == "--update-test-parent" {
		p, err := readPlan(os.Args[2])
		if err == nil {
			p.ParentPID = os.Getpid()
			err = writeJSON(filepath.Join(p.Work, "plan.json"), p)
		}
		if err == nil {
			err = p.Start(context.Background())
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) >= 5 && os.Args[1] == "--update-test-restarted" {
		if os.Getenv("SQUIRREL_UPDATE_TEST_FAIL_NEW") == "1" {
			self, _ := os.Executable()
			data, _ := os.ReadFile(self)
			if strings.HasSuffix(string(data), "squirrel-update-fixture-v1.2.0") {
				os.Exit(42)
			}
		}
		work := os.Args[len(os.Args)-1]
		result, err := Finish(work)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		self, _ := os.Executable()
		_ = writeJSON(os.Args[2], map[string]any{"exe": self, "args": os.Args[1:], "result": result})
		// Stay alive until the updater observes readiness and cleanup finishes.
		for i := 0; i < 100; i++ {
			if _, err := os.Stat(work); os.IsNotExist(err) {
				os.Exit(0)
			}
			time.Sleep(100 * time.Millisecond)
		}
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestNativeUpdateReplaceAndRestart(t *testing.T) {
	c, r, newExe := fixture(t)
	for _, mode := range []string{"portable", "powershell", "installer", "startup-failure"} {
		t.Run(mode, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "中文 安装目录")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			name := "squirrel_translate.exe"
			if mode == "installer" {
				name = "Squirrel Translate.exe"
			}
			target := filepath.Join(root, name)
			self, _ := os.Executable()
			if err := copyFile(self, target); err != nil {
				t.Fatal(err)
			}
			oldHash, _ := fileHash(target)
			for _, f := range []string{"LICENSE", "THIRD_PARTY_NOTICES.txt", "PRIVACY.md", "user-notes.txt", "Uninstall.exe"} {
				if err := os.WriteFile(filepath.Join(root, f), []byte("old "+f), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "powershell" {
				if err := writeJSON(filepath.Join(root, ".squirrel-install.json"), map[string]any{"repository": Repository, "directory": root, "version": "1.0.0", "architecture": runtime.GOARCH, "shortcut": true}); err != nil {
					t.Fatal(err)
				}
			}
			keyName := fmt.Sprintf(`Software\SquirrelTranslateUpdateTests\%d-%d`, os.Getpid(), time.Now().UnixNano())
			access := uint32(registry.ALL_ACCESS)
			if mode == "installer" {
				access |= registry.WOW64_32KEY
			} // NSIS is a 32-bit installer.
			key, _, err := registry.CreateKey(registry.CURRENT_USER, keyName, access)
			if err != nil {
				t.Fatal(err)
			}
			defer registry.DeleteKey(registry.CURRENT_USER, keyName)
			defer key.Close()
			location := root
			if mode != "installer" {
				location = filepath.Join(root, "other-installation")
			}
			key.SetStringValue("InstallLocation", location)
			key.SetStringValue("DisplayIcon", filepath.Join(location, "Squirrel Translate.exe"))
			key.SetStringValue("DisplayVersion", "1.0.0")
			output := filepath.Join(t.TempDir(), "restarted.json")
			args := []string{"--update-test-restarted", output, "--data-dir", filepath.Join(root, "独立 用户数据")}
			p, err := c.Prepare(context.Background(), r, target, runtime.GOARCH, args, nil)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, target, "--update-test-parent", p.Work)
			cmd.Env = append(os.Environ(), "SQUIRREL_UPDATE_TEST_KEY="+keyName)
			if mode == "startup-failure" {
				cmd.Env = append(cmd.Env, "SQUIRREL_UPDATE_TEST_FAIL_NEW=1")
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("parent: %v %s", err, out)
			}
			for i := 0; i < 200; i++ {
				if _, err := os.Stat(output); err == nil {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				result, _ := os.ReadFile(filepath.Join(p.Work, "result.json"))
				t.Fatalf("restart missing: %v %s", err, result)
			}
			var got struct {
				Exe    string
				Args   []string
				Result Result
			}
			if err = json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !samePath(got.Exe, target) || (got.Result.Error != "") != (mode == "startup-failure") || got.Result.RecoveryRequired {
				t.Fatal(string(data))
			}
			if len(got.Args) != len(args)+2 || got.Args[3] != args[3] {
				t.Fatal("restart arguments lost", got.Args)
			}
			installed, err := os.ReadFile(target)
			if mode == "startup-failure" {
				if restored, _ := fileHash(target); restored != oldHash {
					t.Fatal("failed new application was not rolled back")
				}
			} else if err != nil || string(installed) != string(newExe) {
				t.Fatal("new executable not installed", err)
			}
			for _, f := range []string{"user-notes.txt", "Uninstall.exe"} {
				data, _ := os.ReadFile(filepath.Join(root, f))
				if string(data) != "old "+f {
					t.Fatal("unrelated file changed", f)
				}
			}
			if mode == "powershell" {
				data, _ := os.ReadFile(filepath.Join(root, ".squirrel-install.json"))
				var marker map[string]any
				json.Unmarshal(data, &marker)
				if marker["version"] != "1.2.0" || marker["shortcut"] != true {
					t.Fatal(marker)
				}
			}
			v, _, _ := key.GetStringValue("DisplayVersion")
			want := "1.0.0"
			if mode == "installer" {
				want = "1.2.0"
			}
			if v != want {
				t.Fatal("registry version", v, "want", want)
			}
			for i := 0; i < 100; i++ {
				if _, err := os.Stat(p.Work); os.IsNotExist(err) {
					t.Log("real helper replacement, restart, metadata and cleanup passed:", mode)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
			t.Fatal("staging cleanup did not finish")
		})
	}
}

func TestReplacementRollsBack(t *testing.T) {
	c, r, _ := fixture(t)
	for _, failure := range []string{"restart", "locked-file", "changed-checksum", "foreign-marker"} {
		t.Run(failure, func(t *testing.T) {
			target := makeTarget(t, "Squirrel Translate.exe")
			before, _ := fileHash(target)
			root := filepath.Dir(target)
			os.WriteFile(filepath.Join(root, "LICENSE"), []byte("old license"), 0600)
			p, err := c.Prepare(context.Background(), r, target, runtime.GOARCH, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "locked-file":
				path, _ := windows.UTF16PtrFromString(filepath.Join(root, "LICENSE"))
				h, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
				if err != nil {
					t.Fatal(err)
				}
				defer windows.CloseHandle(h)
			case "changed-checksum":
				os.WriteFile(filepath.Join(p.Work, "package.zip"), []byte("tampered"), 0600)
			case "foreign-marker":
				writeJSON(filepath.Join(root, ".squirrel-install.json"), map[string]string{"repository": "other/repo", "directory": root})
			}
			result := p.apply(func() error { return errors.New("simulated restart failure") })
			if result.Error == "" || result.RecoveryRequired {
				t.Fatal(result)
			}
			after, _ := fileHash(target)
			if after != before {
				t.Fatal("old executable not restored")
			}
			license, _ := os.ReadFile(filepath.Join(root, "LICENSE"))
			if string(license) != "old license" {
				t.Fatal("old license not restored")
			}
		})
	}
}

func TestRestartArgs(t *testing.T) {
	got := RestartArgs([]string{"--tray", "--update-cleanup", "C:\\old", "--data-dir", "C:\\中文 path", "--update-cleanup=old", "--lang=en"})
	if strings.Join(got, "|") != "--tray|--data-dir|C:\\中文 path|--lang=en" {
		t.Fatal(got)
	}
}
