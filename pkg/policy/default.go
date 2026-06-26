package policy

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultPolicy returns the built-in default security policy.
func DefaultPolicy() *Policy {
	return &Policy{
		Filesystem: FilesystemPolicy{
			HostHomeMount: false,
			Workspace:     "read-write",
			Artifacts:     "read-write",
			Caches:        "scoped",
			Root:          "read-only-where-possible",
		},
		Process: ProcessPolicy{
			MaxProcesses:   256,
			DefaultTimeout: 120, // 120 seconds
			MaxOutput:      8 * 1024 * 1024, // 8 MiB
		},
		Network: NetworkPolicy{
			Mode: "restricted",
			Deny: []string{
				"169.254.169.254",  // Cloud metadata
				"host-localhost",
				"private-lans",
				"cluster-local",
			},
			Allow: []string{
				"github.com",
				"registry.npmjs.org",
				"pypi.org",
				"files.pythonhosted.org",
				"proxy.golang.org",
				"crates.io",
				"static.crates.io",
			},
		},
		Secrets: SecretsPolicy{
			Env:            "deny-by-default",
			SSHAgent:       "opt-in",
			GitCredentials: "brokered",
		},
	}
}

// LoadPolicy loads policy from the config file.
func LoadPolicy() (*Policy, error) {
	piHome := filepath.Join(os.Getenv("HOME"), ".pi")
	configPath := filepath.Join(piHome, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// No config file — use defaults
		return DefaultPolicy(), nil
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		// Invalid YAML — use defaults
		return DefaultPolicy(), nil
	}

	// Merge with defaults (defaults fill in missing fields)
	merged := DefaultPolicy()
	// Override with loaded values (empty fields are ignored)
	if p.Filesystem.HostHomeMount {
		merged.Filesystem.HostHomeMount = true
	}
	if p.Process.MaxProcesses > 0 {
		merged.Process.MaxProcesses = p.Process.MaxProcesses
	}
	if p.Process.DefaultTimeout > 0 {
		merged.Process.DefaultTimeout = p.Process.DefaultTimeout
	}
	if p.Process.MaxOutput > 0 {
		merged.Process.MaxOutput = p.Process.MaxOutput
	}
	if p.Network.Mode != "" {
		merged.Network.Mode = p.Network.Mode
	}
	if len(p.Network.Deny) > 0 {
		merged.Network.Deny = p.Network.Deny
	}
	if len(p.Network.Allow) > 0 {
		merged.Network.Allow = p.Network.Allow
	}
	if p.Secrets.Env != "" {
		merged.Secrets.Env = p.Secrets.Env
	}
	if p.Secrets.SSHAgent != "" {
		merged.Secrets.SSHAgent = p.Secrets.SSHAgent
	}
	if p.Secrets.GitCredentials != "" {
		merged.Secrets.GitCredentials = p.Secrets.GitCredentials
	}

	return merged, nil
}

// IsNeverMounted checks if a path is never mounted by default.
func IsNeverMounted(path string) bool {
	if path == "/var/run/docker.sock" {
		return true
	}
	if path == "/" {
		return true
	}
	if path == "/proc" || path == "/sys" {
		return true
	}
	// Check subpaths of /proc and /sys
	if filepath.HasPrefix(path, "/proc/") || filepath.HasPrefix(path, "/sys/") {
		return true
	}
	return false
}
