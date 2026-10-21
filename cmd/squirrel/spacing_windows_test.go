//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSpacingOverridePreservesUnspecifiedDefaultsAndExplicitZero(t *testing.T) {
	defaults := mustDefaultSpacing()
	s, err := readSpacing([]byte(`{"translation":{"inset_x":4},"controls":{"button_x":0}}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.Translation.InsetX != 4 || s.Controls.ButtonX != 0 {
		t.Fatal("explicit override, including zero, was not applied")
	}
	s.Translation.InsetX = defaults.Translation.InsetX
	s.Controls.ButtonX = defaults.Controls.ButtonX
	if s != defaults {
		t.Fatal("partial override lost unspecified defaults")
	}
	if s, err = readSpacing([]byte(`{}`)); err != nil || s != defaults {
		t.Fatal("empty override should retain defaults:", err)
	}
}

func TestSpacingRejectsInvalidFilesWithoutChangingActiveTheme(t *testing.T) {
	before := uiSpacing
	path := filepath.Join(t.TempDir(), "spacing.json")
	for _, content := range []string{
		`null`, `[]`,
		`{"translation":{"inset_x":-1}}`,
		`{"controls":{"icon_button_size":8}}`,
		`{"main":{"inset_x":"12"}}`,
		`{"main":{"inset_xx":12}}`,
		`{"main":{"inset_x":12}} {}`,
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if err := configureSpacing(path); err == nil {
			t.Fatalf("accepted invalid theme: %s", content)
		}
		if uiSpacing != before {
			t.Fatal("failed override changed the active theme")
		}
	}
}
