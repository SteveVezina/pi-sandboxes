package compat

import (
	"os/exec"
	"sync"
)

// Runtime represents an OCI runtime.
type Runtime string

const (
	RuntimeContainerd Runtime = "containerd"
	RuntimePodman     Runtime = "podman"
	RuntimeRunc       Runtime = "runc"
	RuntimeDocker     Runtime = "docker"
)

// DetectedRuntime holds the detected runtime info.
type DetectedRuntime struct {
	Name    Runtime
	Path    string
	Version string
}

// DetectRuntime finds the first available OCI runtime.
func DetectRuntime() *DetectedRuntime {
	runtimes := []Runtime{
		RuntimeContainerd,
		RuntimePodman,
		RuntimeRunc,
		RuntimeDocker,
	}

	for _, rt := range runtimes {
		if path, err := exec.LookPath(string(rt)); err == nil {
			return &DetectedRuntime{
				Name: rt,
				Path: path,
			}
		}
	}

	return nil
}

// RuntimeName returns the string name of the runtime.
func (r Runtime) Name() string {
	return string(r)
}

// IsAvailable checks if a specific runtime is available.
func IsAvailable(rt Runtime) bool {
	_, err := exec.LookPath(string(rt))
	return err == nil
}

// BestRuntime returns the best available runtime.
var bestRuntime *DetectedRuntime
var bestRuntimeOnce sync.Once

// Best returns the best available OCI runtime.
func Best() *DetectedRuntime {
	bestRuntimeOnce.Do(func() {
		bestRuntime = DetectRuntime()
	})
	return bestRuntime
}
