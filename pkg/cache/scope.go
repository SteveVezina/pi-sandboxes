package cache

import (
	"os"
	"path/filepath"
)

// Type represents a cache type.
type Type string

const (
	TypeNPM     Type = "npm"
	TypePNPM    Type = "pnpm"
	TypeYarn    Type = "yarn"
	TypePip     Type = "pip"
	TypeUV      Type = "uv"
	TypeGoMod   Type = "go-mod"
	TypeGoBuild Type = "go-build"
	TypeCargo   Type = "cargo"
	TypeSCCache Type = "sccache"
)

// AllCacheTypes returns all known cache types.
func AllCacheTypes() []Type {
	return []Type{
		TypeNPM, TypePNPM, TypeYarn,
		TypePip, TypeUV,
		TypeGoMod, TypeGoBuild,
		TypeCargo, TypeSCCache,
	}
}

// Scope represents a cache scope (template/runtime/user).
type Scope struct {
	Template string
	Runtime  string
	User     string
}

// Dir returns the cache directory path for this scope and type.
func (s Scope) Dir(t Type) string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".pi-box", "caches", s.String(), string(t))
}

// String returns the scope string.
func (s Scope) String() string {
	if s.Template == "" {
		s.Template = "base"
	}
	if s.Runtime == "" {
		s.Runtime = "auto"
	}
	if s.User == "" {
		s.User = "default"
	}
	return filepath.Join(s.Template, s.Runtime, s.User)
}

// Ensure creates the cache directory.
func (s Scope) Ensure(t Type) error {
	dir := s.Dir(t)
	return os.MkdirAll(dir, 0755)
}
