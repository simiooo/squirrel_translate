//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
	"github.com/simiooo/squirrel_translate/internal/i18n"
	"github.com/simiooo/squirrel_translate/internal/store"
	"github.com/simiooo/squirrel_translate/internal/update"
)

func TestUpdateSettingsReachableAndPreferenceSaved(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer i18n.Set("zh-CN", "")
	for _, lang := range []string{"zh-CN", "en"} {
		i18n.Set(lang, "")
		a := &application{store: st, cfg: core.DefaultConfig(), tab: 2, settingsTab: 4, updates: updateUI{release: &update.Release{Tag: "v1.2.0", Notes: strings.Repeat("Release note\n", 100)}}}
		for _, size := range [][2]int{{1040, 704}, {640, 480}} {
			tt := ui.NewTester(a.view, size[0], size[1])
			for _, label := range []string{tr("检查更新"), tr("下载更新并重启"), tr("自动检查更新（仅提示）")} {
				r, ok := tt.Find(label)
				if !ok || r.W <= 0 || r.H <= 0 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(size[0])+1 || r.Y+r.H > float32(size[1])+1 {
					t.Fatalf("%s %v %s unreachable: %+v", lang, size, label, r)
				}
			}
			if err := tt.Click(tr("自动检查更新（仅提示）")); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(st.Dir, "updates.json")); err != nil {
				t.Fatal("update preference not persisted", err)
			}
		}
	}
}
