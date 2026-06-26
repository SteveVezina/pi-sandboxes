package microvm_test

import (
	"errors"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

// fakeVMM is an in-memory VMM driver for testing lifecycle without Firecracker.
type fakeVMM struct {
	started    bool
	stopped    bool
	rootfsRO   bool
	bootCalled int
	startErr   error
	stopErr    error
}

func (f *fakeVMM) Boot(cfg microvm.VMConfig) error {
	f.bootCalled++
	if f.startErr != nil {
		return f.startErr
	}
	f.started = true
	f.rootfsRO = cfg.Rootfs.ReadOnly
	return nil
}

func (f *fakeVMM) Shutdown() error {
	if f.stopErr != nil {
		return f.stopErr
	}
	f.stopped = true
	return nil
}

func (f *fakeVMM) Running() bool { return f.started && !f.stopped }

func TestSandbox_StartUsesFakeVMM(t *testing.T) {
	vmm := &fakeVMM{}
	sb := microvm.NewSandbox("session-1", vmm)

	if err := sb.Start(microvm.VMConfig{
		Rootfs:    microvm.Disk{Path: "/img/rootfs.ext4", ReadOnly: true},
		Workspace: microvm.Disk{Path: "/img/ws.ext4", ReadOnly: false},
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if !vmm.started {
		t.Fatal("expected VMM to be started")
	}
	if vmm.bootCalled != 1 {
		t.Fatalf("Boot called %d times, want 1", vmm.bootCalled)
	}
	if sb.State() != microvm.SandboxStateBooting {
		t.Fatalf("state = %q, want booting", sb.State())
	}
}

func TestSandbox_StopShutsDownVMM(t *testing.T) {
	vmm := &fakeVMM{}
	sb := microvm.NewSandbox("session-1", vmm)
	_ = sb.Start(microvm.VMConfig{
		Rootfs:    microvm.Disk{Path: "/img/rootfs.ext4", ReadOnly: true},
		Workspace: microvm.Disk{Path: "/img/ws.ext4", ReadOnly: false},
	})

	if err := sb.Stop(); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if !vmm.stopped {
		t.Fatal("expected VMM stopped")
	}
}

func TestSandbox_GuestRootfsIsReadOnly(t *testing.T) {
	vmm := &fakeVMM{}
	sb := microvm.NewSandbox("session-1", vmm)
	_ = sb.Start(microvm.VMConfig{
		Rootfs:    microvm.Disk{Path: "/img/rootfs.ext4", ReadOnly: true},
		Workspace: microvm.Disk{Path: "/img/ws.ext4", ReadOnly: false},
	})

	if !vmm.rootfsRO {
		t.Fatal("guest rootfs must be read-only")
	}
}

func TestSandbox_StartRejectsWritableRootfs(t *testing.T) {
	vmm := &fakeVMM{}
	sb := microvm.NewSandbox("session-1", vmm)

	err := sb.Start(microvm.VMConfig{
		Rootfs:    microvm.Disk{Path: "/img/rootfs.ext4", ReadOnly: false},
		Workspace: microvm.Disk{Path: "/img/ws.ext4", ReadOnly: false},
	})
	if err == nil {
		t.Fatal("expected error when rootfs is writable")
	}
	if vmm.bootCalled != 0 {
		t.Fatalf("VMM Boot called %d times despite invalid config", vmm.bootCalled)
	}
}

func TestSandbox_StartPropagatesVMMError(t *testing.T) {
	want := errors.New("boot failed")
	vmm := &fakeVMM{startErr: want}
	sb := microvm.NewSandbox("session-1", vmm)

	err := sb.Start(microvm.VMConfig{
		Rootfs:    microvm.Disk{Path: "/img/rootfs.ext4", ReadOnly: true},
		Workspace: microvm.Disk{Path: "/img/ws.ext4", ReadOnly: false},
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want wrap of %v", err, want)
	}
}
