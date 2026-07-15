package microvm

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const (
	RuntimeName       = "microvm"
	DefaultKVMPath    = "/dev/kvm"
	DefaultVMMCommand = "firecracker"
)

// Availability describes whether the host can run the microVM backend.
type Availability struct {
	Available bool
	Reasons   []string
}

// Error returns an actionable unavailable message.
func (a Availability) Error() error {
	if a.Available {
		return nil
	}
	if len(a.Reasons) == 0 {
		return fmt.Errorf("microvm unavailable")
	}
	return fmt.Errorf("microvm unavailable: %s", joinReasons(a.Reasons))
}

// CapabilityChecker checks host support for Firecracker-backed microVM mode.
type CapabilityChecker struct {
	GOOS       string
	KVMPath    string
	VMMCommand string
	Stat       func(string) (os.FileInfo, error)
	LookPath   func(string) (string, error)
}

// DefaultCapabilityChecker returns the real host capability checker.
func DefaultCapabilityChecker() CapabilityChecker {
	return CapabilityChecker{
		GOOS:       runtime.GOOS,
		KVMPath:    DefaultKVMPath,
		VMMCommand: DefaultVMMCommand,
		Stat:       os.Stat,
		LookPath:   exec.LookPath,
	}
}

// CheckAvailability reports whether microVM mode is available.
func CheckAvailability(checker CapabilityChecker) Availability {
	if checker.GOOS == "" {
		checker.GOOS = runtime.GOOS
	}
	if checker.KVMPath == "" {
		checker.KVMPath = DefaultKVMPath
	}
	if checker.VMMCommand == "" {
		checker.VMMCommand = DefaultVMMCommand
	}
	if checker.Stat == nil {
		checker.Stat = os.Stat
	}
	if checker.LookPath == nil {
		checker.LookPath = exec.LookPath
	}

	var reasons []string
	if checker.GOOS != "linux" {
		reasons = append(reasons, "requires Linux")
	}
	if _, err := checker.Stat(checker.KVMPath); err != nil {
		reasons = append(reasons, fmt.Sprintf("%s is unavailable", checker.KVMPath))
	}
	if _, err := checker.LookPath(checker.VMMCommand); err != nil {
		reasons = append(reasons, fmt.Sprintf("%s not found on PATH", checker.VMMCommand))
	}
	return Availability{
		Available: len(reasons) == 0,
		Reasons:   reasons,
	}
}

// Runtime is the Firecracker-backed microVM runtime descriptor.
type Runtime struct {
	availability Availability
}

// NewRuntime creates a microVM runtime descriptor from host availability.
func NewRuntime(checker CapabilityChecker) *Runtime {
	return &Runtime{availability: CheckAvailability(checker)}
}

func (r *Runtime) Name() string               { return RuntimeName }
func (r *Runtime) IsAvailable() bool          { return r.availability.Available }
func (r *Runtime) GetMode() string            { return RuntimeName }
func (r *Runtime) Availability() Availability { return r.availability }

func joinReasons(reasons []string) string {
	if len(reasons) == 1 {
		return reasons[0]
	}
	out := reasons[0]
	for _, reason := range reasons[1:] {
		out += "; " + reason
	}
	return out
}
