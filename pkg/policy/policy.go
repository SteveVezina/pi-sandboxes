package policy

import "fmt"

// Policy defines the security constraints for a sandbox session.
type Policy struct {
	Filesystem FilesystemPolicy  `yaml:"filesystem"`
	Process    ProcessPolicy     `yaml:"process"`
	Network    NetworkPolicy     `yaml:"network"`
	Secrets    SecretsPolicy     `yaml:"secrets"`
}

// FilesystemPolicy defines filesystem access rules.
type FilesystemPolicy struct {
	HostHomeMount bool   `yaml:"hostHomeMount"`
	Workspace     string `yaml:"workspace"`     // "read-write", "read-only"
	Artifacts     string `yaml:"artifacts"`     // "read-write", "read-only"
	Caches        string `yaml:"caches"`        // "scoped", "none"
	Root          string `yaml:"root"`          // "read-only-where-possible"
}

// ProcessPolicy defines process execution limits.
type ProcessPolicy struct {
	MaxProcesses int           `yaml:"maxProcesses"`
	DefaultTimeout int64       `yaml:"defaultTimeout"` // seconds
	MaxOutput    int64         `yaml:"maxOutput"`     // bytes
}

// NetworkPolicy defines network access rules.
type NetworkPolicy struct {
	Mode string   `yaml:"mode"` // "restricted", "full"
	Deny []string `yaml:"deny"`
	Allow []string `yaml:"allow"`
}

// SecretsPolicy defines secrets handling rules.
type SecretsPolicy struct {
	Env            string `yaml:"env"`            // "deny-by-default"
	SSHAgent       string `yaml:"sshAgent"`       // "opt-in"
	GitCredentials string `yaml:"gitCredentials"` // "brokered"
}

// Validate checks if the policy is valid.
func (p *Policy) Validate() error {
	if p.Process.MaxProcesses <= 0 {
		return fmt.Errorf("maxProcesses must be > 0")
	}
	if p.Process.DefaultTimeout <= 0 {
		return fmt.Errorf("defaultTimeout must be > 0")
	}
	if p.Process.MaxOutput <= 0 {
		return fmt.Errorf("maxOutput must be > 0")
	}
	return nil
}
