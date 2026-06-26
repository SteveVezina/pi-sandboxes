package fast_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/fast"
)

func TestDefaultNamespaceConfig(t *testing.T) {
	cfg := fast.DefaultNamespaceConfig()

	if !cfg.UserNS {
		t.Error("Expected UserNS to be true")
	}
	if !cfg.MountNS {
		t.Error("Expected MountNS to be true")
	}
	if !cfg.PIDNS {
		t.Error("Expected PIDNS to be true")
	}
	if cfg.HostUID != 1000 {
		t.Errorf("Expected HostUID 1000, got %d", cfg.HostUID)
	}
	if cfg.HostGID != 1000 {
		t.Errorf("Expected HostGID 1000, got %d", cfg.HostGID)
	}
}

func TestSetupReturnsErrorOnNonLinux(t *testing.T) {
	cfg := fast.DefaultNamespaceConfig()
	_, err := fast.Setup(cfg)
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestValidateReturnsErrorOnNonLinux(t *testing.T) {
	err := fast.Validate()
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestWriteUIDMapReturnsErrorOnNonLinux(t *testing.T) {
	err := fast.WriteUIDMap(1, "0 1000 1\n1 1001 65535\n")
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestWriteGIDMapReturnsErrorOnNonLinux(t *testing.T) {
	err := fast.WriteGIDMap(1, "0 1000 1\n1 1001 65535\n")
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}
