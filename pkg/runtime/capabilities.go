// Package runtime defines the runtime driver contract shared by all
// isolation backends (SPEC.md §14.7.5, ADR-005).
package runtime

// CapabilityReport describes one runtime mode's availability and guarantees.
// A runtime is never summarized by a single security integer: isolation and
// compatibility are separate axes, and availability carries its own reason.
type CapabilityReport struct {
	Mode        string   `json:"mode"`
	Available   bool     `json:"available"`
	Reason      string   `json:"reason,omitempty"`
	Missing     []string `json:"missing,omitempty"`
	Description string   `json:"description,omitempty"`

	KernelBoundary   bool `json:"kernel_boundary"`
	Rootless         bool `json:"rootless"`
	UserNamespace    bool `json:"user_namespace"`
	Seccomp          bool `json:"seccomp"`
	Landlock         bool `json:"landlock"`
	NetworkNamespace bool `json:"network_namespace"`
	EgressPolicy     bool `json:"egress_policy"`
	Snapshot         bool `json:"snapshot"`
	WarmExec         bool `json:"warm_exec"`
	OCIImages        bool `json:"oci_images"`
	HardwareVirt     bool `json:"hardware_virt"`

	// IsolationTier: 1=contained 2=sandboxed 3=virtualized 4=hardened.
	IsolationTier int `json:"isolation_tier"`
	// CompatTier: 1=narrow ... 4=full Linux tool compatibility.
	CompatTier int `json:"compat_tier"`
}
