// Package events emits the daemon's service-level lifecycle events
// (ADR-007). It is deliberately small: a fan-out emitter over a Sink
// interface, with a slog sink and an optional webhook sink. Delivery is
// asynchronous and never blocks or fails the operation that produced the
// event.
package events

import (
	"log/slog"
	"sync"
	"time"
)

// Lifecycle event type names (`.pi/block.yaml` § lifecycle_events).
const (
	SandboxCreated    = "pi.sandbox.created"
	RunStarted        = "pi.run.started"
	RunCompleted      = "pi.run.completed"
	SandboxDestroyed  = "pi.sandbox.destroyed"
	ArtifactDelivered = "pi.artifact.delivered"
)

// Event is the envelope defined in ADR-007 §1.
type Event struct {
	Type      string         `json:"type"`
	Time      time.Time      `json:"time"`
	SandboxID string         `json:"sandbox_id,omitempty"`
	RunID     string         `json:"run_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// Sink delivers one event somewhere. Implementations must not panic and
// should not block for long.
type Sink interface {
	Deliver(Event)
}

// Emitter fans an event out to its sinks, each on its own goroutine.
type Emitter struct {
	sinks []Sink
}

// New builds an emitter over the given sinks.
func New(sinks ...Sink) *Emitter {
	return &Emitter{sinks: sinks}
}

// Emit delivers e to every sink asynchronously. Time is stamped here if
// the caller left it zero.
func (em *Emitter) Emit(e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	for _, s := range em.sinks {
		s := s
		go func() {
			defer func() { _ = recover() }()
			s.Deliver(e)
		}()
	}
}

var (
	mu      sync.RWMutex
	current = New(SlogSink{})
)

// SetDefault installs the process emitter. The daemon calls this at
// startup once it knows whether a webhook is configured.
func SetDefault(em *Emitter) {
	mu.Lock()
	defer mu.Unlock()
	current = em
}

// Emit sends an event through the process default emitter.
func Emit(e Event) {
	mu.RLock()
	em := current
	mu.RUnlock()
	em.Emit(e)
}

// SlogSink writes each event as one structured log line.
type SlogSink struct{ Logger *slog.Logger }

// Deliver implements Sink.
func (s SlogSink) Deliver(e Event) {
	l := s.Logger
	if l == nil {
		l = slog.Default()
	}
	l.Info("lifecycle event",
		"event", e.Type,
		"time", e.Time.Format(time.RFC3339),
		"sandbox_id", e.SandboxID,
		"run_id", e.RunID,
		"data", e.Data,
	)
}
