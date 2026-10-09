package tests

import (
	"os"
	"path/filepath"
	"testing"

	"Straggle/internal/model"
	"Straggle/internal/settings"
)

func TestLoadWritesDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	store := settings.NewAt(dir)
	got := store.Load()
	if got != model.DefaultSettings() {
		t.Fatalf("a missing file must fall back to the defaults, got %+v", got)
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("a missing file must be written back with the defaults: %v", err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := settings.NewAt(dir)
	want := model.DefaultSettings()
	want.OnlyLocal = true
	want.IntervalSec = 10
	want.Background = false
	want.Language = "zh-CN"

	saved, err := store.Save(want)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved != want {
		t.Fatalf("Save returned %+v, want %+v", saved, want)
	}
	reopened := settings.NewAt(dir)
	if got := reopened.Load(); got != want {
		t.Fatalf("reloading returned %+v, want %+v", got, want)
	}
}

func TestCorruptFileFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{ not JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := settings.NewAt(dir)
	if got := store.Load(); got != model.DefaultSettings() {
		t.Fatalf("a corrupt file must fall back to the defaults, got %+v", got)
	}
	if got := settings.NewAt(dir).Load(); got != model.DefaultSettings() {
		t.Fatalf("the rewritten file must load cleanly, got %+v", got)
	}
}

func TestNormalizeInterval(t *testing.T) {
	v := model.DefaultSettings()
	v.IntervalSec = 7
	if got := settings.Normalize(v); got.IntervalSec != 5 {
		t.Fatalf("an illegal interval must fall back to 5, got %d", got.IntervalSec)
	}
	for _, ok := range []int{1, 5, 10} {
		v.IntervalSec = ok
		if got := settings.Normalize(v); got.IntervalSec != ok {
			t.Fatalf("legal interval %d was rewritten to %d", ok, got.IntervalSec)
		}
	}
}

func TestNormalizeTheme(t *testing.T) {
	v := model.DefaultSettings()
	if v.Theme != model.ThemeSystem {
		t.Fatalf("the factory default must follow the system, got %q", v.Theme)
	}
	v.Theme = "Light" // only lowercase light / dark are accepted
	if got := settings.Normalize(v); got.Theme != model.ThemeSystem {
		t.Fatalf("an illegal theme must fall back to the system, got %q", got.Theme)
	}
	for _, ok := range []string{model.ThemeLight, model.ThemeDark} {
		v.Theme = ok
		if got := settings.Normalize(v); got.Theme != ok {
			t.Fatalf("legal theme %q was rewritten to %q", ok, got.Theme)
		}
	}
}

func TestNormalizeLanguage(t *testing.T) {
	v := model.DefaultSettings()
	if v.Language != "" {
		t.Fatalf("the factory default must follow the system, got %q", v.Language)
	}
	v.Language = "zh"
	if got := settings.Normalize(v); got.Language != "" {
		t.Fatalf("an unsupported language must fall back to the system, got %q", got.Language)
	}
	for _, ok := range []string{"en", "zh-CN"} {
		v.Language = ok
		if got := settings.Normalize(v); got.Language != ok {
			t.Fatalf("supported language %q was rewritten to %q", ok, got.Language)
		}
	}
}
