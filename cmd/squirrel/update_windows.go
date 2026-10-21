//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/update"
)

// Only plain development builds use this override (e.g. isolated update tests).
// Official packages obtain their version from mygo's embedded package metadata.
var versionOverride string

type updatePreferences struct {
	Disabled  bool      `json:"disabled"`
	LastCheck time.Time `json:"last_check"`
}
type updateUI struct {
	prefs   updatePreferences
	release *update.Release
	busy    bool
	message string
	cancel  context.CancelFunc
}

func currentVersion() string {
	if versionOverride != "" {
		return versionOverride
	}
	return mygo.App.Version()
}
func (a *application) saveUpdatePreferences() {
	if a.store == nil {
		return
	}
	data, err := json.Marshal(a.updates.prefs)
	if err != nil {
		return
	}
	if err = os.WriteFile(filepath.Join(a.store.Dir, "updates.json"), data, 0600); err != nil {
		a.updates.message = tr("更新偏好保存失败：") + err.Error()
	}
}

func (a *application) initUpdates(cleanup string) {
	if data, err := os.ReadFile(filepath.Join(a.store.Dir, "updates.json")); err == nil {
		_ = json.Unmarshal(data, &a.updates.prefs)
	}
	if cleanup != "" {
		result, err := update.Finish(cleanup)
		if err != nil {
			a.updates.message = tr("更新状态读取失败：") + err.Error()
		} else if result.Error != "" {
			a.updates.message = tr("更新失败，已尝试恢复原版本：") + result.Error
			if result.RecoveryRequired {
				a.updates.message += "\n" + tr("请保留此目录以恢复文件：") + cleanup
			}
		} else {
			a.updates.message = fmt.Sprintf(tr("已更新到 %s"), result.Version)
		}
		a.status = a.updates.message
	}
	// Never download or replace automatically. Background checks only notify.
	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-timer.C:
				mygo.RunOnMain(func() {
					if !a.quitting && !a.updates.prefs.Disabled && time.Since(a.updates.prefs.LastCheck) >= 24*time.Hour && currentVersion() != "" {
						a.checkUpdates(false)
					}
				})
				timer.Reset(24 * time.Hour)
			}
		}
	}()
}

func (a *application) showUpdates() {
	a.tab = 2
	a.settingsTab = 4
	a.window.Show()
	a.window.Invalidate()
}

func (a *application) checkUpdates(show bool) {
	if show {
		a.showUpdates()
	}
	if a.updates.busy {
		return
	}
	if _, err := update.Newer("0.0.0", currentVersion()); err != nil {
		a.updates.message = tr("开发版本无法自动更新，请安装正式发布版。")
		a.window.Invalidate()
		return
	}
	a.updates.busy = true
	a.updates.release = nil
	a.updates.message = tr("正在检查更新……")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	a.updates.cancel = cancel
	go func() {
		defer cancel()
		r, err := update.NewClient().Check(ctx, currentVersion(), runtime.GOARCH)
		a.window.Update(func() {
			if a.quitting {
				return
			}
			a.updates.busy = false
			if err != nil {
				a.updates.message = tr("检查更新失败：") + err.Error()
				return
			}
			a.updates.prefs.LastCheck = time.Now()
			a.saveUpdatePreferences()
			a.updates.release = r
			if r == nil {
				a.updates.message = tr("当前已是最新版本。")
				return
			}
			a.updates.message = fmt.Sprintf(tr("发现新版本 %s"), r.Version())
			if !show {
				n := mygo.NewNotification(mygo.NotificationOptions{Title: productName(), Body: a.updates.message, Silent: true})
				n.OnClick(a.showUpdates)
				_ = n.Show()
			}
		})
	}()
}

func (a *application) installUpdate() {
	if a.updates.busy || a.updates.release == nil {
		return
	}
	release := *a.updates.release
	a.updates.busy = true
	a.updates.message = tr("正在下载并校验更新……")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	a.updates.cancel = cancel
	go func() {
		defer cancel()
		exe, err := os.Executable()
		var plan *update.Plan
		if err == nil {
			plan, err = update.NewClient().Prepare(ctx, release, exe, runtime.GOARCH, update.RestartArgs(os.Args[1:]), func(done, total int64) {
				a.window.Update(func() {
					if !a.quitting {
						a.updates.message = fmt.Sprintf(tr("正在下载更新：%.1f / %.1f MB"), float64(done)/(1<<20), float64(total)/(1<<20))
					}
				})
			})
		}
		if err == nil {
			err = plan.Start(ctx)
		}
		if err != nil && plan != nil {
			_ = os.RemoveAll(plan.Work)
		}
		a.window.Update(func() {
			if a.quitting {
				return
			}
			a.updates.busy = false
			if err != nil {
				a.updates.message = tr("更新失败，当前版本未更改：") + err.Error()
				return
			}
			a.updates.message = tr("正在退出并重启以完成更新……")
			mygo.App.Quit()
		})
	}()
}

func (a *application) updateSettings(c *ui.Context) {
	version := currentVersion()
	if version == "" {
		version = tr("开发版本")
	}
	sectionTitle(c, tr("软件更新"))
	ui.Text(c, fmt.Sprintf(tr("当前版本：%s"), version)).FontSize(14)
	ui.Text(c, tr("从 GitHub Releases 获取正式版；配置、词汇表和密钥将保留。")).FontSize(12).TextColor(colorToken(c, "Muted"))
	automatic := !a.updates.prefs.Disabled
	if check(c, &automatic, tr("自动检查更新（仅提示）")).Changed() {
		a.updates.prefs.Disabled = !automatic
		a.saveUpdatePreferences()
	}
	ui.Row(c).Gap(metric("Space/8")).Children(func() {
		if textButton(c, tr("检查更新")).Disabled(a.updates.busy).Clicked() {
			a.checkUpdates(true)
		}
		if a.updates.release != nil {
			if primaryButton(c, tr("下载更新并重启")).Disabled(a.updates.busy).Clicked() {
				a.installUpdate()
			}
		}
		if a.updates.busy && a.updates.cancel != nil {
			if textButton(c, tr("取消")).Clicked() {
				a.updates.cancel()
			}
		}
	})
	if a.updates.message != "" {
		ui.Text(c, a.updates.message).FontSize(13).Selectable()
	}
	if r := a.updates.release; r != nil {
		ui.Text(c, tr("点击更新后，下载校验成功会自动退出并重新启动。")).FontSize(12).TextColor(colorToken(c, "Muted"))
		if r.Notes != "" {
			ui.Text(c, r.Notes).FontSize(12).Selectable()
		}
	}
}
