//go:build !linux
// +build !linux

package fast

import (
	"fmt"
	"os"
)

// SeccompProfile defines the seccomp-bpf filter.
type SeccompProfile struct {
	DefaultAction string
	Architectures []string
	Syscalls      []interface{}
}

// DefaultSeccompProfile returns the default seccomp profile.
func DefaultSeccompProfile() *SeccompProfile {
	return &SeccompProfile{
		DefaultAction: "SCMP_ACT_ERRNO",
	}
}

// Save writes the seccomp profile to a file.
func (p *SeccompProfile) Save(path string) error {
	return fmt.Errorf("seccomp requires Linux")
}

// Load reads a seccomp profile from a file.
func Load(path string) (*SeccompProfile, error) {
	return nil, fmt.Errorf("seccomp requires Linux")
}

// SaveDefault writes the default seccomp profile to a file.
func SaveDefault(path string) error {
	// Create a placeholder file so the path exists
	return os.WriteFile(path, []byte("{}"), 0644)
}
