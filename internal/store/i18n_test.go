package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguagePreferenceAndLegacyConfig(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.Dir, "settings.json"), []byte(`{"primary_language":"简体中文","secondary_language":"English"}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := s.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.UILanguage != "" {
		t.Fatal("legacy config should follow system")
	}
	for _, locale := range []string{"en", "zh-CN", "system"} {
		c.UILanguage = locale
		if err = s.SaveConfig(c); err != nil {
			t.Fatal(err)
		}
		got, err := s.LoadConfig()
		if err != nil {
			t.Fatal(err)
		}
		if got.UILanguage != locale || got.PrimaryLanguage != "简体中文" || got.SecondaryLanguage != "English" {
			t.Fatalf("language persistence changed translation preferences: %+v", got)
		}
	}
	c.UILanguage = "fr"
	if err = s.SaveConfig(c); err == nil {
		t.Fatal("unsupported UI locale accepted")
	}
}
