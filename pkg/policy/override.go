package policy

// Override represents per-session policy overrides.
type Override struct {
	MaxProcesses   *int
	MaxOutput      *int64
	DefaultTimeout *int64
	NetworkMode    *string
	HostHomeMount  *bool
}

// ApplyMerge merges overrides into the base policy.
// Overrides cannot relax default deny policies (security invariant).
func (o *Override) ApplyMerge(base *Policy) *Policy {
	result := *base // Copy

	if o.MaxProcesses != nil && *o.MaxProcesses > 0 {
		// Cannot increase max processes beyond default (security)
		if *o.MaxProcesses <= base.Process.MaxProcesses {
			result.Process.MaxProcesses = *o.MaxProcesses
		}
	}

	if o.MaxOutput != nil && *o.MaxOutput > 0 {
		// Cannot increase max output beyond default (security)
		if *o.MaxOutput <= base.Process.MaxOutput {
			result.Process.MaxOutput = *o.MaxOutput
		}
	}

	if o.DefaultTimeout != nil && *o.DefaultTimeout > 0 {
		// Cannot increase timeout beyond default (security)
		if *o.DefaultTimeout <= base.Process.DefaultTimeout {
			result.Process.DefaultTimeout = *o.DefaultTimeout
		}
	}

	if o.NetworkMode != nil {
		// Cannot relax from restricted to full
		if *o.NetworkMode != "full" || base.Network.Mode == "full" {
			result.Network.Mode = *o.NetworkMode
		}
	}

	if o.HostHomeMount != nil {
		// Never allow host home mount if default is false
		if !*o.HostHomeMount || base.Filesystem.HostHomeMount {
			result.Filesystem.HostHomeMount = *o.HostHomeMount
		}
	}

	return &result
}
