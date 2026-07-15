// Package detect wires the runtime registry with probers for every
// backend and derives availability answers from capability reports
// (SPEC.md §14.7.5, ADR-005). Probes actually execute their checks;
// a runtime is never summarized by a single security integer.
package detect

import (
	"context"
	"fmt"
	"os/exec"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/runtime/fast"
	"github.com/pi-sandbox/pi/pkg/runtime/gvisor"
	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

// DefaultRegistry returns the registry with all known probers registered
// in priority order: microvm, secure, fast, compat.
func DefaultRegistry(rootDir string) *pruntime.Registry {
	reg := pruntime.NewRegistry()
	// Registration order = priority order; Register only errors on
	// duplicate modes, which cannot happen here.
	_ = reg.Register(microvmProber{})
	_ = reg.Register(secureProber{rootDir: rootDir})
	_ = reg.Register(fastProber{})
	_ = reg.Register(compatProber{})
	return reg
}

// Reports probes every backend and returns capability reports in
// priority order.
func Reports(rootDir string) []pruntime.CapabilityReport {
	return DefaultRegistry(rootDir).Reports(context.Background())
}

// AvailableRuntimes returns the available mode names in priority order.
func AvailableRuntimes(rootDir string) []string {
	var available []string
	for _, rep := range Reports(rootDir) {
		if rep.Available {
			available = append(available, rep.Mode)
		}
	}
	return available
}

// BestMode returns the first available mode in priority order, or
// "unknown" when no backend is available.
func BestMode(rootDir string) string {
	for _, rep := range Reports(rootDir) {
		if rep.Available {
			return rep.Mode
		}
	}
	return "unknown"
}

// ── Probers ────────────────────────────────────────────────────────────

type microvmProber struct{}

func (microvmProber) Mode() pruntime.Mode { return pruntime.ModeMicroVM }

func (microvmProber) Probe(ctx context.Context) pruntime.CapabilityReport {
	avail := microvm.CheckAvailability(microvm.DefaultCapabilityChecker())
	rep := pruntime.CapabilityReport{
		Mode:             string(pruntime.ModeMicroVM),
		Available:        avail.Available,
		Missing:          avail.Reasons,
		Description:      "Firecracker/Cloud Hypervisor microVM — VM kernel boundary, snapshot-first",
		KernelBoundary:   true,
		HardwareVirt:     true,
		NetworkNamespace: true,
		Snapshot:         true,
		WarmExec:         true,
		IsolationTier:    4,
		CompatTier:       2,
	}
	if err := avail.Error(); err != nil {
		rep.Reason = err.Error()
	}
	return rep
}

type secureProber struct {
	rootDir string
}

func (secureProber) Mode() pruntime.Mode { return pruntime.ModeSecure }

func (p secureProber) Probe(ctx context.Context) pruntime.CapabilityReport {
	rep := pruntime.CapabilityReport{
		Mode:             string(pruntime.ModeSecure),
		Description:      "gVisor (runsc) — userspace application kernel, OCI-compatible",
		Seccomp:          true,
		NetworkNamespace: true,
		OCIImages:        true,
		WarmExec:         true,
		IsolationTier:    2,
		CompatTier:       2,
	}
	if _, err := exec.LookPath("runsc"); err != nil {
		rep.Reason = "runsc not found on PATH"
		rep.Missing = []string{"runsc"}
		return rep
	}
	rep.Available = gvisor.Default(p.rootDir).IsAvailable()
	if !rep.Available {
		rep.Reason = "runsc found but not operational"
	}
	return rep
}

type fastProber struct{}

func (fastProber) Mode() pruntime.Mode { return pruntime.ModeFast }

func (fastProber) Probe(ctx context.Context) pruntime.CapabilityReport {
	rep := pruntime.CapabilityReport{
		Mode:          string(pruntime.ModeFast),
		Description:   "Native Linux namespaces/cgroups/seccomp/Landlock — fastest path, Linux-only",
		Rootless:      true,
		UserNamespace: true,
		Seccomp:       true,
		Landlock:      true,
		WarmExec:      true,
		IsolationTier: 1,
		CompatTier:    3,
	}
	if err := fast.Validate(); err != nil {
		rep.Reason = err.Error()
		rep.Missing = []string{"linux user namespaces"}
		return rep
	}
	rep.Available = true
	return rep
}

type compatProber struct{}

func (compatProber) Mode() pruntime.Mode { return pruntime.ModeCompat }

func (compatProber) Probe(ctx context.Context) pruntime.CapabilityReport {
	rep := pruntime.CapabilityReport{
		Mode:             string(pruntime.ModeCompat),
		Description:      "OCI container runtime (Docker/Podman) — best tool compatibility",
		Seccomp:          true,
		NetworkNamespace: true,
		OCIImages:        true,
		WarmExec:         true,
		IsolationTier:    1,
		CompatTier:       4,
	}
	rt := compat.Best()
	if rt == nil {
		rep.Reason = "no OCI runtime found on PATH"
		rep.Missing = []string{"docker or podman"}
		return rep
	}
	if err := validateCompatRuntime(rt); err != nil {
		rep.Reason = err.Error()
		rep.Missing = []string{fmt.Sprintf("%s daemon", rt.Name)}
		return rep
	}
	rep.Available = true
	rep.Rootless = rt.Name == compat.RuntimePodman
	rep.Description = fmt.Sprintf("OCI container runtime (%s) — best tool compatibility", rt.Name)
	return rep
}

func validateCompatRuntime(rt *compat.DetectedRuntime) error {
	switch rt.Name {
	case compat.RuntimeDocker:
		if err := exec.Command(rt.Path, "info").Run(); err != nil {
			return fmt.Errorf("docker runtime unavailable: %w", err)
		}
	case compat.RuntimePodman:
		if err := exec.Command(rt.Path, "info").Run(); err != nil {
			return fmt.Errorf("podman runtime unavailable: %w", err)
		}
	default:
		return fmt.Errorf("unsupported compat runtime: %s", rt.Name)
	}
	return nil
}
