// Package detect provides runtime selection and fallback logic.
// It tries runtimes in order of security: gvisor → fast → compat.
package detect

import (
	"fmt"

	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/runtime/fast"
	"github.com/pi-sandbox/pi/pkg/runtime/gvisor"
)

// Runtime represents any available sandbox runtime.
type Runtime interface {
	Name() string
	IsAvailable() bool
	IsGvisor() bool
	GetMode() string
	GetSecurityLevel() int
}

// Priority defines the order in which runtimes are tried.
var priority = []string{"gvisor", "fast", "compat"}

// Detect tries each runtime in priority order and returns the first available one.
// If no runtime is available, returns an error describing what was tried.
func Detect(rootDir string) (Runtime, error) {
	var tried []string

	for _, name := range priority {
		rt, err := tryRuntime(name, rootDir)
		if err != nil {
			tried = append(tried, name)
			continue
		}
		return rt, nil
	}

	return nil, fmt.Errorf("no sandbox runtime available (tried: %v)", tried)
}

// tryRuntime attempts to create and validate a runtime by name.
func tryRuntime(name, rootDir string) (Runtime, error) {
	switch name {
	case "gvisor":
		rt := gvisor.Default(rootDir)
		if rt.IsAvailable() {
			return rt, nil
		}
		return nil, fmt.Errorf("gVisor not available")
	case "fast":
		if err := fast.Validate(); err != nil {
			return nil, fmt.Errorf("fast backend not available: %w", err)
		}
		return &fastRuntime{rootDir: rootDir}, nil
	case "compat":
		rt := compat.Best()
		if rt == nil {
			return nil, fmt.Errorf("no OCI runtime available")
		}
		return &compatRuntime{detected: rt}, nil
	default:
		return nil, fmt.Errorf("unknown runtime: %s", name)
	}
}

// fastRuntime is a minimal wrapper for the fast backend.
type fastRuntime struct {
	rootDir string
}

func (f *fastRuntime) Name() string        { return "fast" }
func (f *fastRuntime) IsAvailable() bool   { return true }
func (f *fastRuntime) IsGvisor() bool      { return false }
func (f *fastRuntime) GetMode() string     { return "fast" }
func (f *fastRuntime) GetSecurityLevel() int { return 5 }

// compatRuntime is a minimal wrapper for the compat backend.
type compatRuntime struct {
	detected *compat.DetectedRuntime
}

func (c *compatRuntime) Name() string        { return string(c.detected.Name) }
func (c *compatRuntime) IsAvailable() bool   { return true }
func (c *compatRuntime) IsGvisor() bool      { return false }
func (c *compatRuntime) GetMode() string     { return "compat" }
func (c *compatRuntime) GetSecurityLevel() int { return 3 }

// RuntimeInfo holds detailed information about a single runtime backend.
type RuntimeInfo struct {
	Name          string `json:"name"`
	Available     bool   `json:"available"`
	SecurityLevel int    `json:"security_level"`
	Description   string `json:"description"`
}

// AvailableRuntimes returns a list of all available runtime names.
func AvailableRuntimes(rootDir string) []string {
	var available []string
	for _, name := range priority {
		if _, err := tryRuntime(name, rootDir); err == nil {
			available = append(available, name)
		}
	}
	return available
}

// AllRuntimes returns detailed info for every known runtime backend.
func AllRuntimes(rootDir string) []RuntimeInfo {
	var result []RuntimeInfo
	runtimeDescriptions := map[string]string{
		"gvisor": "gVisor sandboxed runtime — strong isolation, may have syscall compatibility issues",
		"fast":   "Native Linux namespaces/cgroups — fastest path, Linux-only",
		"compat": "OCI container runtime (runc/podman) — best compatibility",
		"microvm":"Firecracker/Cloud Hypervisor microVM — highest isolation",
	}
	for _, name := range priority {
		rt, err := tryRuntime(name, rootDir)
		info := RuntimeInfo{
			Name:          name,
			Available:     err == nil,
			SecurityLevel: 0,
			Description:   runtimeDescriptions[name],
		}
		if err == nil {
			info.SecurityLevel = rt.GetSecurityLevel()
		}
		result = append(result, info)
	}
	// Add microvm separately (not in priority but a known backend)
	if _, err := tryRuntime("microvm", rootDir); err == nil {
		result = append(result, RuntimeInfo{
			Name:          "microvm",
			Available:     true,
			SecurityLevel: 10,
			Description:   "Firecracker/Cloud Hypervisor microVM — highest isolation",
		})
	}
	return result
}

// BestMode returns the best available mode string.
func BestMode(rootDir string) string {
	rt, err := Detect(rootDir)
	if err != nil {
		return "unknown"
	}
	return rt.GetMode()
}

// BestSecurityLevel returns the best available security level.
func BestSecurityLevel(rootDir string) int {
	rt, err := Detect(rootDir)
	if err != nil {
		return 0
	}
	return rt.GetSecurityLevel()
}
