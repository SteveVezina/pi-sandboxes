package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) defaultMarkerPath() string {
	return filepath.Join(s.templatesDir, ".default")
}

// SetDefault marks a template as the one `pi-box box create` uses when no
// template is given. The template must exist.
func (s *Store) SetDefault(name string) error {
	if _, err := s.Get(name); err != nil {
		return fmt.Errorf("cannot promote %q: %w", name, err)
	}
	if err := os.MkdirAll(s.templatesDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(s.defaultMarkerPath(), []byte(name+"\n"), 0o644); err != nil {
		return fmt.Errorf("write default marker: %w", err)
	}
	audit("promote", name, "set as default")
	return nil
}

// Default returns the promoted default template name, or "" when none is
// set or the marked template no longer exists.
func (s *Store) Default() string {
	data, err := os.ReadFile(s.defaultMarkerPath())
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return ""
	}
	if _, err := s.Get(name); err != nil {
		return ""
	}
	return name
}
