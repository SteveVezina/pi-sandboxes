//go:build linux
// +build linux

package fast_test

import (
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/fast"
)

func TestDefaultNamespaceConfig_Linux(t *testing.T) {
	cfg := fast.DefaultNamespaceConfig()

	if !cfg.UserNS || !cfg.MountNS || !cfg.PIDNS {
		t.Errorf("Expected all namespaces enabled, got %+v", cfg)
	}
	if cfg.HostUID != 1000 || cfg.HostGID != 1000 {
		t.Errorf("Expected HostUID/HostGID 1000, got %d/%d", cfg.HostUID, cfg.HostGID)
	}
}

// TestValidate_MatchesHostCapability asserts Validate() reflects the host's
// real ability to create user/mount/PID namespaces. A Validate() that returns
// nil on a host where the probe clone fails is the false-positive defect
// PROP-008 T3.1 fixed.
func TestValidate_MatchesHostCapability(t *testing.T) {
	probe := exec.Command("true")
	probe.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}
	hostErr := probe.Run()

	valErr := fast.Validate()

	if (hostErr == nil) != (valErr == nil) {
		t.Fatalf("Validate() = %v, but direct namespace probe = %v; availability must match host capability", valErr, hostErr)
	}
}

func TestSetup_ConfiguresNamespaces(t *testing.T) {
	cfg := fast.DefaultNamespaceConfig()
	cmd, err := fast.Setup(cfg)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	attr := cmd.SysProcAttr
	if attr == nil {
		t.Fatal("Expected SysProcAttr to be set")
	}
	wantFlags := uintptr(syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID)
	if attr.Cloneflags&wantFlags != wantFlags {
		t.Errorf("Expected clone flags %x, got %x", wantFlags, attr.Cloneflags)
	}
	if len(attr.UidMappings) == 0 || len(attr.GidMappings) == 0 {
		t.Error("Expected UID/GID mappings to be set")
	}
}
