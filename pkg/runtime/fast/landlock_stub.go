//go:build !linux
// +build !linux

package fast

import "fmt"

// LandlockConfig holds landlock policy configuration.
type LandlockConfig struct {
	Enabled bool
	Paths   []string
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

// ApplyLandlock applies landlock restrictions.
func ApplyLandlock(cfg *LandlockConfig) error {
	return fmt.Errorf("landlock requires Linux")
}

// IsLandlockAvailable checks if landlock is supported.
func IsLandlockAvailable() bool {
	return false
}
