# ADR-002: MicroVM Guest Control Protocol

## Status
Accepted

## Context

F21 requires host-to-guest command execution, file transfer, artifact transfer, lifecycle coordination, and readiness reporting without direct host filesystem mounts.

## Decision

The host and guest communicate over virtio-vsock using newline-delimited JSON control frames.

Each frame has these fields:

- `type`: `request`, `response`, `event`, or `stream`
- `id`: request identifier
- `session_id`: sandbox/session identifier
- `method`: `hello`, `ready`, `exec`, `file.read`, `file.write`, `artifact.list`, `artifact.pull`, or `shutdown`
- `payload`: method-specific JSON object
- `error`: optional object with `code` and `message`

Exec stdout/stderr use `stream` frames. Stream payloads include `stream: stdout|stderr` and `data: base64-bytes`.

The final exec response includes `exit_code`, `duration_ms`, `timed_out`, and `truncated`.

Readiness is explicit: `pi-init` starts `pi-agentd`; `pi-agentd` sends a `ready` event; only then may the host mark the sandbox warm.

## Consequences

- The guest protocol is inspectable and simple to test with line-oriented readers.
- Binary stream data is safely represented as base64 inside JSON frames.
- Readiness is tied to guest agent state instead of VM process startup alone.

## References

- `SPEC.md` §14.7.4 MicroVM mode
- `docs/features/F21-microvm-guest-control-plane.md`
- PROP-003

