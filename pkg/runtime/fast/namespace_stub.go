//go:build !linux
// +build !linux

package fast

import (
	"fmt"
	"os/exec"
)

// NamespaceConfig holds the namespace configuration for a sandbox process.
type NamespaceConfig struct {
	UserNS  bool
	MountNS bool
	PIDNS   bool
	HostUID int
	HostGID int
}

// DefaultNamespaceConfig returns the default namespace configuration.
func DefaultNamespaceConfig() *NamespaceConfig {
	return &NamespaceConfig{
		UserNS:  true,
		MountNS: true,
		PIDNS:   true,
		HostUID: 1000,
		HostGID: 1000,
	}
}

// Setup creates a new process in isolated namespaces.
// On non-Linux systems, returns an error.
func Setup(cfg *NamespaceConfig) (*exec.Cmd, error) {
	return nil, fmt.Errorf("namespace isolation requires Linux")
}

// Validate checks if namespace operations are supported on this system.
func Validate() error {
	return fmt.Errorf("namespace isolation requires Linux")
}

// WriteUIDMap writes the UID mapping for a namespace.
func WriteUIDMap(namespaceID int, uidMap string) error {
	return fmt.Errorf("namespace isolation requires Linux")
}

// WriteGIDMap writes the GID mapping for a namespace.
func WriteGIDMap(namespaceID int, gidMap string) error {
	return fmt.Errorf("namespace isolation requires Linux")
}
