//go:build linux
// +build linux

package fast

import (
	"fmt"
)

// LandlockConfig holds landlock policy configuration.
type LandlockConfig struct {
	Enabled bool
	Paths   []string // Paths to restrict access to
	Read    bool
	Write   bool
	Execute bool
}

// DefaultLandlockConfig returns the default landlock configuration.
func DefaultLandlockConfig() *LandlockConfig {
	return &LandlockConfig{
		Enabled: true,
		Read:    true,
		Write:   false,
		Execute: false,
	}
}

// ApplyLandlock applies landlock restrictions to the current process.
// Returns an error if landlock is not available (kernel < 5.13).
func ApplyLandlock(cfg *LandlockConfig) error {
	if cfg == nil {
		cfg = DefaultLandlockConfig()
	}

	if !cfg.Enabled {
		return nil
	}

	// Landlock requires kernel >= 5.13 and syscall support.
	// In Go, landlock is available via golang.org/x/exp/syscall on newer versions.
	// For now, we return a placeholder error indicating landlock is not yet implemented.
	// This is a stub that will be filled in when golang.org/x/exp/syscall landlock support is stable.
	return fmt.Errorf("landlock: not yet implemented (requires kernel >= 5.13)")
}

// IsLandlockAvailable checks if the kernel supports landlock.
func IsLandlockAvailable() bool {
	// Check kernel version >= 5.13
	// This is a stub — real implementation would read /proc/version
	return false
}
