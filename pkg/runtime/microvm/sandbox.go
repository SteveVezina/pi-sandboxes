package microvm

import (
	"fmt"
	"sync"
)

// Disk describes a microVM disk attachment.
type Disk struct {
	// Path is the host path to the backing file or block device.
	Path string
	// ReadOnly indicates whether the guest sees the disk as read-only.
	ReadOnly bool
	// Filesystem is the on-disk filesystem (e.g. "ext4"). Optional for raw images.
	Filesystem string
}

// VMConfig describes the disks and resources for a microVM boot.
type VMConfig struct {
	Rootfs    Disk
	Workspace Disk
}

// VMM is the minimal driver interface a microVM backend must satisfy.
// Firecracker and Cloud Hypervisor implementations both satisfy this.
type VMM interface {
	Boot(VMConfig) error
	Shutdown() error
	Running() bool
}

// Sandbox is the host-side handle for a microVM-backed sandbox session.
type Sandbox struct {
	sessionID string
	driver    VMM
	state     SandboxState
	mu        sync.Mutex
}

// NewSandbox creates a sandbox bound to the given VMM driver and session ID.
func NewSandbox(sessionID string, driver VMM) *Sandbox {
	return &Sandbox{
		sessionID: sessionID,
		driver:    driver,
		state:     SandboxStateBooting,
	}
}

// SessionID returns the sandbox session identifier.
func (s *Sandbox) SessionID() string { return s.sessionID }

// State returns the current sandbox state.
func (s *Sandbox) State() SandboxState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Start boots the microVM with the given config.
// Returns an error if the configuration violates the read-only rootfs invariant.
func (s *Sandbox) Start(cfg VMConfig) error {
	if s.driver == nil {
		return fmt.Errorf("microvm sandbox %s: nil VMM driver", s.sessionID)
	}
	if !cfg.Rootfs.ReadOnly {
		return fmt.Errorf("microvm sandbox %s: guest rootfs must be read-only (ADR-001)", s.sessionID)
	}
	if cfg.Workspace.ReadOnly {
		return fmt.Errorf("microvm sandbox %s: workspace disk must be writable (ADR-001)", s.sessionID)
	}
	if err := s.driver.Boot(cfg); err != nil {
		return fmt.Errorf("microvm sandbox %s: boot: %w", s.sessionID, err)
	}
	s.mu.Lock()
	s.state = SandboxStateBooting
	s.mu.Unlock()
	return nil
}

// Stop shuts the microVM down.
func (s *Sandbox) Stop() error {
	if s.driver == nil {
		return fmt.Errorf("microvm sandbox %s: nil VMM driver", s.sessionID)
	}
	if err := s.driver.Shutdown(); err != nil {
		return fmt.Errorf("microvm sandbox %s: shutdown: %w", s.sessionID, err)
	}
	return nil
}

// MarkWarm transitions the sandbox to warm state. Callers must only invoke
// this after the guest has reported ready (ADR-002).
func (s *Sandbox) MarkWarm() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = SandboxStateWarm
}
