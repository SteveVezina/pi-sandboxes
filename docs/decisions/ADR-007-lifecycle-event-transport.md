# ADR-007: Lifecycle Event Transport

## Status

Proposed (2026-08-31) — awaiting human acceptance.

Unblocks: F9 AC-9.4 (`pi.artifact.delivered`), F4 lifecycle-event ACs,
F29 AC-31.2 (`pi.run.started` / `pi.run.completed`). No block-spec
amendment — the five events are already listed in `.pi/block.yaml`
§ `lifecycle_events` and `SPEC.md` §6; this ADR only chooses how they
leave the daemon.

## Context

The five service-level lifecycle events (`pi.sandbox.created`,
`pi.run.started`, `pi.run.completed`, `pi.sandbox.destroyed`,
`pi.artifact.delivered`) are specified but have **no implementation** —
no emitter, no envelope, no transport. F9's `pi.artifact.delivered` AC has
been open for this reason since PROP-009, and F29 (Agent Run) cannot start
without a way to surface `pi.run.*`.

This is distinct from the F29 *agent event stream* — the raw
per-iteration agent output relayed over SSE/WebSocket. That stream carries
the agent's own frames and is out of scope here. This ADR covers only the
handful of coarse service-level lifecycle events.

## Decision

### 1. Envelope

```json
{
  "type":       "pi.sandbox.created",
  "time":       "2026-08-31T12:00:00Z",
  "sandbox_id": "…",
  "run_id":     "…",          // omitted unless the event belongs to an agent run
  "data":       { … }          // event-specific, optional
}
```

`type` is one of the five names. `time` is RFC3339 UTC. `data` is small
and non-secret (e.g. `{ "template": "node", "mode": "compat" }` for
`created`; `{ "path": "artifacts.tar.gz", "bytes": 12345 }` for
`delivered`). No credential material, no host paths beyond what the API
already returns.

### 2. Transport — two sinks, both best-effort

- **Structured log sink (always on).** Every event is written as one
  `slog` line with a stable `event` attribute plus the envelope fields.
  An external tailer can consume the daemon log. This is the zero-config
  baseline.
- **Webhook sink (opt-in).** If the daemon is started with
  `--events-webhook <url>` (or the equivalent config key), each event is
  `POST`ed as the JSON envelope. One retry on failure, 5s timeout, then
  the event is dropped with a warning. Delivery is asynchronous and
  **never blocks or fails the operation that produced the event**.

No message bus, no per-client SSE topic, no persistence/replay. Those can
be added later behind the same `events.Sink` interface without touching
call sites.

### 3. Emitter

`pkg/events`:

```go
type Event struct { Type, SandboxID, RunID string; Time time.Time; Data map[string]any }
type Sink interface { Deliver(Event) }
type Emitter struct { … }          // fan-out to sinks, async
func Emit(e Event)                  // package-level default emitter
func SetDefault(*Emitter)           // daemon wires sinks at startup
```

The default emitter is a no-op until the daemon calls `SetDefault`
(same daemon-singleton pattern as `api.SetEgressProxyAddr`). Tests can
install a capturing sink.

### 4. Emission points

| Event | Emitted from | When |
|-------|--------------|------|
| `pi.sandbox.created` | `api.CreateSandbox` | after the sandbox reaches `WARM` |
| `pi.sandbox.destroyed` | `api.DeleteSandbox` + TTL reaper | after runtime object removal |
| `pi.artifact.delivered` | `api` output pull / pack handlers | only after the copy/archive succeeds (closes F9 AC-9.4) |
| `pi.run.started` / `pi.run.completed` | F29 agent-run handler | once per run (F29 wires these; emitter is ready now) |

## Consequences

- F9 AC-9.4 becomes implementable now; F29 is unblocked.
- The daemon log gains one line per lifecycle event. Acceptable — these
  are coarse (seconds-to-minutes apart), not per-exec.
- Webhook delivery is fire-and-forget; a consumer that needs
  at-least-once or ordering must add its own dedup. Documented, not
  solved here.
- The `events.Sink` seam lets a future ADR add SSE/replay/a bus without
  revisiting emission points.

## References

- `.pi/block.yaml` § `lifecycle_events`
- `SPEC.md` §6 (F9, F29), AC-9.4, AC-31.2
- `docs/features/F08-output-delivery.md`, `docs/features/F29-agent-run.md`,
  `docs/features/F04-sandbox-lifecycle.md`
- ADR-006 (same daemon-singleton wiring pattern)
