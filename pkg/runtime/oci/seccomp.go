package oci

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// SeccompProfileVersion identifies the shipped container seccomp profile.
// Bump when seccomp-profile.json changes so stale copies are replaced.
const SeccompProfileVersion = "1"

//go:embed seccomp-profile.json
var seccompProfile []byte

// SeccompProfilePath materializes the versioned project seccomp profile
// under the Pi Box state dir and returns its path. The profile is passed
// explicitly to every engine — engine default profiles differ between
// Docker and Podman (SPEC.md §14.7.5).
func SeccompProfilePath() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	dir := filepath.Join(home, ".pi-box", "security")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create security dir: %w", err)
	}

	path := filepath.Join(dir, "seccomp-profile-v"+SeccompProfileVersion+".json")
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, seccompProfile) {
		return path, nil
	}
	if err := os.WriteFile(path, seccompProfile, 0o644); err != nil {
		return "", fmt.Errorf("write seccomp profile: %w", err)
	}
	return path, nil
}
