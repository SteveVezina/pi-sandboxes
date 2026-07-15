package runtime

import (
	"context"
	"fmt"
)

// Prober reports availability and capabilities for one runtime mode.
// Full Driver implementations satisfy this; modes not yet migrated to the
// lifecycle contract register probe-only adapters.
type Prober interface {
	Mode() Mode
	Probe(ctx context.Context) CapabilityReport
}

// Registry holds registered runtime probers in priority order.
type Registry struct {
	order   []Mode
	probers map[Mode]Prober
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{probers: make(map[Mode]Prober)}
}

// Register adds a prober. Registration order is priority order.
func (r *Registry) Register(p Prober) error {
	mode := p.Mode()
	if _, exists := r.probers[mode]; exists {
		return fmt.Errorf("runtime mode %s already registered", mode)
	}
	r.probers[mode] = p
	r.order = append(r.order, mode)
	return nil
}

// Reports probes every registered mode in registration order. Probes run
// on every call; availability is never cached silently.
func (r *Registry) Reports(ctx context.Context) []CapabilityReport {
	reports := make([]CapabilityReport, 0, len(r.order))
	for _, mode := range r.order {
		reports = append(reports, r.probers[mode].Probe(ctx))
	}
	return reports
}

// Probe probes a single mode. The second return is false when the mode is
// not registered.
func (r *Registry) Probe(ctx context.Context, mode Mode) (CapabilityReport, bool) {
	p, ok := r.probers[mode]
	if !ok {
		return CapabilityReport{}, false
	}
	return p.Probe(ctx), true
}

// Modes returns the registered modes in priority order.
func (r *Registry) Modes() []Mode {
	return append([]Mode(nil), r.order...)
}
