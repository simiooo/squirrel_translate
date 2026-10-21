//go:build windows

package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/simiooo/squirrel_translate/internal/core"
)

func TestParameterDraftPreservesZeroAndClearsDefaultsWithoutMutatingSnapshot(t *testing.T) {
	c := core.DefaultConfig()
	previous := 0.8
	c.Temperature = &previous
	d := parameterDraft(c)
	d.Temperature, d.Seed, d.MaxOutputTokens = "0", "0", "8192"
	updated, err := d.apply(c)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Temperature == nil || *updated.Temperature != 0 || updated.Seed == nil || *updated.Seed != 0 || *updated.MaxOutputTokens != 8192 || updated.TopP != nil {
		t.Fatal("draft lost explicit values or sent an unset option")
	}
	if *c.Temperature != previous {
		t.Fatal("draft mutated the configuration owned by an existing session")
	}
	cleared, err := (modelParameterDraft{}).apply(updated)
	if err != nil || cleared.Temperature != nil || cleared.MaxOutputTokens != nil || cleared.Seed != nil {
		t.Fatal("blank fields did not restore provider defaults", err)
	}
	for _, bad := range []modelParameterDraft{{Temperature: "NaN"}, {TopP: "Inf"}, {Temperature: "2.1"}, {MaxOutputTokens: "0"}, {MaxOutputTokens: "1.5"}, {Seed: "abc"}} {
		if _, err := bad.apply(c); err == nil {
			t.Fatalf("accepted invalid draft: %+v", bad)
		}
	}
}

func TestReasoningSelectorStoresCanonicalValue(t *testing.T) {
	effort := "default"
	tt := ui.NewTester(func(c *ui.Context) {
		minimalTheme(c)
		reasoningSelect(c, &effort)
	}, 240, 300)
	if err := tt.Click("思考等级"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnd)
	tt.Key(0, ui.KeyEnter)
	if effort != "max" {
		t.Fatalf("translated label was not mapped to the API value: %q", effort)
	}
	_ = tt.Click("思考等级")
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyEnter)
	if effort != "default" {
		t.Fatal("could not restore provider default")
	}
}
