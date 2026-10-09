// Package settings persists user preferences to %APPDATA%\Straggle\settings.json.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"Straggle/internal/i18n"
	"Straggle/internal/model"
)

// Store is the read/write entry point for settings; it is safe for concurrent use.
type Store struct {
	mu  sync.Mutex
	dir string
	cur model.Settings
}

// DefaultDir returns the application configuration directory.
func DefaultDir() string {
	if base, err := os.UserConfigDir(); err == nil && base != "" {
		return filepath.Join(base, "Straggle")
	}
	return "Straggle"
}

// New builds a Store in the default directory.
func New() *Store { return NewAt(DefaultDir()) }

// NewAt builds a Store in dir.
func NewAt(dir string) *Store {
	return &Store{dir: dir, cur: model.DefaultSettings()}
}

// Dir returns the configuration directory.
func (s *Store) Dir() string { return s.dir }

// Path returns the settings file path.
func (s *Store) Path() string { return filepath.Join(s.dir, "settings.json") }

// Normalize replaces illegal values with defaults: the refresh interval only
// accepts 1 / 5 / 10 seconds, the theme light / dark, the language en / zh-CN.
func Normalize(v model.Settings) model.Settings {
	switch v.IntervalSec {
	case 1, 5, 10:
	default:
		v.IntervalSec = model.DefaultSettings().IntervalSec
	}
	switch v.Theme {
	case model.ThemeLight, model.ThemeDark:
	default:
		v.Theme = model.ThemeSystem
	}
	if !i18n.Valid(v.Language) {
		v.Language = ""
	}
	return v
}

// Load reads the settings file, falling back to the defaults and rewriting the
// file when it is missing or corrupt.
func (s *Store) Load() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, needWrite := s.read()
	s.cur = v
	if needWrite {
		_ = s.write(v)
	}
	return v
}

// Current returns the settings held in memory.
func (s *Store) Current() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Save normalizes v and writes it atomically.
func (s *Store) Save(v model.Settings) (model.Settings, error) {
	v = Normalize(v)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.write(v); err != nil {
		return s.cur, err
	}
	s.cur = v
	return v, nil
}

func (s *Store) read() (model.Settings, bool) {
	def := model.DefaultSettings()
	data, err := os.ReadFile(s.Path())
	if err != nil {
		return def, true
	}
	var v model.Settings
	if err := json.Unmarshal(data, &v); err != nil {
		return def, true
	}
	return Normalize(v), false
}

// write goes through a temporary file so a crash cannot leave a half-written
// settings file behind.
func (s *Store) write(v model.Settings) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := s.Path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.Path()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
