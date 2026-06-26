package microvm_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func TestAvailability_MissingKVMReportsUnavailable(t *testing.T) {
	availability := microvm.CheckAvailability(microvm.CapabilityChecker{
		GOOS:       "linux",
		KVMPath:    "/missing/kvm",
		VMMCommand: "firecracker",
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		LookPath: func(string) (string, error) {
			return "/usr/bin/firecracker", nil
		},
	})

	if availability.Available {
		t.Fatal("expected microvm unavailable when /dev/kvm is missing")
	}
	if !containsReason(availability.Reasons, "/missing/kvm is unavailable") {
		t.Fatalf("reasons = %v, want missing kvm reason", availability.Reasons)
	}
}

func TestAvailability_MissingFirecrackerReportsUnavailable(t *testing.T) {
	availability := microvm.CheckAvailability(microvm.CapabilityChecker{
		GOOS:       "linux",
		KVMPath:    "/dev/kvm",
		VMMCommand: "firecracker",
		Stat: func(string) (os.FileInfo, error) {
			return nil, nil
		},
		LookPath: func(string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	})

	if availability.Available {
		t.Fatal("expected microvm unavailable when firecracker is missing")
	}
	if !containsReason(availability.Reasons, "firecracker not found on PATH") {
		t.Fatalf("reasons = %v, want missing firecracker reason", availability.Reasons)
	}
}

func TestAvailability_AvailableWhenLinuxKVMAndFirecrackerExist(t *testing.T) {
	availability := microvm.CheckAvailability(microvm.CapabilityChecker{
		GOOS:       "linux",
		KVMPath:    "/dev/kvm",
		VMMCommand: "firecracker",
		Stat: func(string) (os.FileInfo, error) {
			return nil, nil
		},
		LookPath: func(string) (string, error) {
			return "/usr/bin/firecracker", nil
		},
	})

	if !availability.Available {
		t.Fatalf("expected available, got reasons %v", availability.Reasons)
	}
	if err := availability.Error(); err != nil {
		t.Fatalf("available runtime returned error: %v", err)
	}
}

func TestRuntime_ExplicitMicroVMModeDoesNotFallback(t *testing.T) {
	rt := microvm.NewRuntime(microvm.CapabilityChecker{
		GOOS:       "linux",
		KVMPath:    "/dev/kvm",
		VMMCommand: "firecracker",
		Stat: func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		LookPath: func(string) (string, error) {
			return "/usr/bin/firecracker", nil
		},
	})

	if rt.GetMode() != "microvm" {
		t.Fatalf("mode = %q, want microvm", rt.GetMode())
	}
	if rt.IsAvailable() {
		t.Fatal("expected explicit microvm runtime to stay unavailable instead of falling back")
	}
	if err := rt.Availability().Error(); err == nil || !strings.Contains(err.Error(), "microvm unavailable") {
		t.Fatalf("availability error = %v, want actionable microvm error", err)
	}
}

func containsReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}
