---
sidebar_position: 6
---

# Lifecycle events

The daemon emits five service-level lifecycle events. They always go to
the daemon log (one structured line each); with
`pi-sandboxd --events-webhook <url>` they are also `POST`ed as JSON.
Delivery is best-effort and never blocks the operation that produced it.

## Envelope

```json
{
  "type": "pi.sandbox.created",
  "time": "2026-08-31T12:00:00Z",
  "sandbox_id": "…",
  "run_id": "…",
  "data": { "template": "node-python", "mode": "compat" }
}
```

`run_id` is present only for agent-run events. `data` is small and carries
no credential material.

## Event types

| Type | When | `data` |
|------|------|--------|
| `pi.sandbox.created` | after a new sandbox reaches `WARM` | `template`, `mode`, `network` |
| `pi.sandbox.destroyed` | after teardown on explicit delete | `mode`, `reason` |
| `pi.artifact.delivered` | after a successful output `pull` / `pack` | `mode`, `destination`/`path`, `items`/`bytes` |
| `pi.run.started` | when an agent run starts | `agent` |
| `pi.run.completed` | once per agent run, on any terminal state | `status`, `exit_code` |

## Webhook delivery

One retry, 5 s timeout, then the event is dropped with a warning. A
consumer that needs at-least-once or ordering must dedupe on its own.
