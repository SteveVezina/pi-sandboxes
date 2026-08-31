package api

import (
	"github.com/pi-sandbox/pi/pkg/network"
	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// egressProxyAddr is the daemon egress proxy address (host:port), set once
// by the daemon at startup. Empty disables proxy env injection into
// restricted-mode sandboxes. Daemon-singleton config (ADR-006).
var egressProxyAddr string

// SetEgressProxyAddr records the daemon egress proxy address so
// restricted-mode sandbox containers can be pointed at it.
func SetEgressProxyAddr(addr string) { egressProxyAddr = addr }

// sandboxEgressNetwork builds the driver-facing NetworkSpec for a sandbox
// from its persisted egress mode plus the daemon proxy address. Sandboxes
// predating ADR-006 default to restricted.
func sandboxEgressNetwork(meta *sandbox.Meta) pruntime.NetworkSpec {
	mode := meta.NetworkMode
	if mode == "" {
		mode = string(network.ModeRestricted)
	}
	ns := pruntime.NetworkSpec{Mode: mode}
	if mode == string(network.ModeRestricted) {
		ns.ProxyAddr = egressProxyAddr
	}
	return ns
}

// sandboxNetworkPolicy returns the egress policy for a sandbox, rebuilt from
// its persisted network mode and allowlist (ADR-006 — mode is fixed at
// create). A sandbox created before this field existed defaults to
// restricted.
func sandboxNetworkPolicy(store *sandbox.Store, id string) (*network.Policy, error) {
	meta, err := store.Get(id)
	if err != nil {
		return nil, err
	}
	mode := meta.NetworkMode
	if mode == "" {
		mode = string(network.ModeRestricted)
	}
	return network.PolicyFor(mode, meta.NetworkAllow)
}
