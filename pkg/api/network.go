package api

import (
	"github.com/pi-sandbox/pi/pkg/network"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

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
