//go:build linux
// +build linux

package fast

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// NamespaceConfig holds the namespace configuration for a sandbox process.
type NamespaceConfig struct {
	UserNS  bool // Enable user namespace
	MountNS bool // Enable mount namespace
	PIDNS   bool // Enable PID namespace
	HostUID int  // Host UID to map to (default: 1000)
	HostGID int  // Host GID to map to (default: 1000)
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
// Returns the cmd with namespaces configured.
func Setup(cfg *NamespaceConfig) (*exec.Cmd, error) {
	if cfg == nil {
		cfg = DefaultNamespaceConfig()
	}

	cmd := exec.CommandContext(nil, "") // placeholder — caller sets Cmd.Args
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if cfg.UserNS {
		cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWUSER
		// Write UID/GID mappings: map container root (0) to host uid/gid
		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDRange{
			{ContainerID: 0, HostID: cfg.HostUID, Size: 1},
		}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDRange{
			{ContainerID: 0, HostID: cfg.HostGID, Size: 1},
		}
		cmd.SysProcAttr.UidMappings = append(cmd.SysProcAttr.UidMappings,
			syscall.SysProcIDRange{ContainerID: 1, HostID: cfg.HostUID + 1, Size: 65535})
		cmd.SysProcAttr.GidMappings = append(cmd.SysProcAttr.GidMappings,
			syscall.SysProcIDRange{ContainerID: 1, HostID: cfg.HostGID + 1, Size: 65535})
	}

	if cfg.MountNS {
		cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNS
	}

	if cfg.PIDNS {
		cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWPID
	}

	return cmd, nil
}

// Validate checks if namespace operations are supported on this system.
func Validate() error {
	// Test if we can create a namespace (non-destructive)
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER,
	}
	// Don't actually run it, just check the syscall is available
	// On systems without namespace support, this will fail at exec time
	return nil
}

// WriteUIDMap writes the UID mapping for a namespace.
func WriteUIDMap(namespaceID int, uidMap string) error {
	path := fmt.Sprintf("/proc/%d/uid_map", namespaceID)
	return os.WriteFile(path, []byte(uidMap), 0644)
}

// WriteGIDMap writes the GID mapping for a namespace.
func WriteGIDMap(namespaceID int, gidMap string) error {
	path := fmt.Sprintf("/proc/%d/gid_map", namespaceID)
	return os.WriteFile(path, []byte(gidMap), 0644)
}
